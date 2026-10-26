// Package downloading is the Downloading Role's holder half: each Node's
// built-in torrent client. One actor, Loop, holds every torrent's state
// and runs what the pure plan decides, gated by the announce ledger, and
// reports each torrent's state to the Controller.
package downloading

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

// Def is the Downloading Role. Its facts are the client itself, so any
// change but a new volume sample is notable.
var Def = role.Def[*dlv1.Intent, *dlv1.Facts, *dlv1.Report]{Name: role.Downloading, Notable: clientDiffers}

func clientDiffers(last, next *dlv1.Facts) bool {
	l, n := proto.CloneOf(last), proto.CloneOf(next)
	l.Downloads, n.Downloads = nil, nil
	return !proto.Equal(l, n)
}

type Holder = role.Holder[*dlv1.Intent, *dlv1.Facts, *dlv1.Report]

// Control is the Controller as the client calls it.
type Control = downloadingv1connect.DownloadingControlServiceHandler

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is the downloading local set: what the Node keeps of each
// torrent it holds.
func Schema() storage.Schema[storage.Local] {
	return storage.NewSchema[storage.Local]("downloading", schemaFiles)
}

type Timings struct {
	// Facts is how often the client reads its torrents' status and sends
	// its facts when they changed.
	Facts time.Duration `yaml:"facts"`
	// AnnounceFloor is the least time between two regular announces to a
	// tracker, whatever interval the tracker answers.
	AnnounceFloor time.Duration `yaml:"announce_floor"`
	// ResumeSave is how often a changed torrent's resume data is saved.
	ResumeSave time.Duration `yaml:"resume_save"`
	// ShutdownSave is how long a stopping Node waits for its resume data.
	ShutdownSave time.Duration `yaml:"shutdown_save"`
	// StartSpread spreads the starts of the torrents the client adds back
	// from resume data over this long, so their started announces don't
	// all go at once.
	StartSpread time.Duration `yaml:"start_spread"`
	// AnnounceJitter is the most a regular announce waits past the
	// tracker's interval.
	AnnounceJitter time.Duration `yaml:"announce_jitter"`
	// TotalsReport is how often the torrents' totals go to the Controller.
	TotalsReport time.Duration `yaml:"totals_report"`
	// LiveSample is how often the torrents' rates and progress are sampled
	// while an Admin watches.
	LiveSample time.Duration `yaml:"live_sample"`
	// RouteCheck is how often a client listening on an unspecified address
	// checks which address the default route leaves from, to follow it.
	RouteCheck time.Duration `yaml:"route_check"`
}

func DefaultTimings() Timings {
	return Timings{
		Facts: time.Second, AnnounceFloor: 5 * time.Minute, ResumeSave: 5 * time.Minute, ShutdownSave: 15 * time.Second,
		StartSpread: 2 * time.Minute, AnnounceJitter: 3 * time.Minute, TotalsReport: 5 * time.Minute, LiveSample: time.Second, RouteCheck: 30 * time.Second,
	}
}

// ErrNotPresent is a torrent this Node's client hasn't finished.
var ErrNotPresent = errors.New("the torrent client hasn't finished the torrent")

type Config struct {
	Engine engine.Engine
	// Dir is the downloads folder: a new torrent's save folder is in it.
	Dir string
	// Listen is the engine's configured listen address.
	Listen  netip.AddrPort
	Timings Timings
	// RetryMin and RetryMax bound the backoff of a failed Torrent fetch.
	RetryMin, RetryMax time.Duration
	// Rand is the torrent stream, after the engine's Open used it.
	Rand *rand.Rand
	// ProvesPort says whether an incoming peer at an address proves the
	// listen port reachable from the internet: PublicPeer.
	ProvesPort func(netip.Addr) bool
	// Volume reads the volume that holds a folder. It may block on a hung
	// disk, so Loop never calls it.
	Volume func(ctx context.Context, path string) (storing.Volume, error)
	Log    *slog.Logger
}

// Client is a Node's built-in client, one per Node from the moment its
// engine opens, claimed or not. Loop is the only goroutine that touches
// its torrents; every other method asks Loop or reads what Loop
// published.
type Client struct {
	cfg     Config
	in      chan func(*loop)
	stopped chan struct{}
	held    atomic.Pointer[published]
}

// published is what Loop last made of its torrents for other goroutines.
type published struct {
	// intended is false until the first intent comes, while the client
	// can't tell a torrent it lost from one it hasn't listed yet.
	intended bool
	torrents map[bencode.InfoHash]held
	seeding  int
	folders  []string
}

// held is one torrent as Loop last published it.
type held struct {
	// live is whether the intent lists it among its Torrents.
	live bool
	// handle is 0 while the engine doesn't hold it.
	handle   engine.Handle
	folder   string
	complete bool
	checking bool
	// failed is the engine's error text, "" while it holds the torrent.
	failed string
}

func (h held) finished() bool { return h.handle != 0 && h.complete }

func NewClient(cfg Config) *Client {
	c := &Client{cfg: cfg, in: make(chan func(*loop)), stopped: make(chan struct{})}
	c.held.Store(&published{folders: []string{cfg.Dir}})
	return c
}

// send runs f on Loop's goroutine, unless Loop has returned.
func (c *Client) send(f func(*loop)) bool {
	select {
	case c.in <- f:
		return true
	case <-c.stopped:
		return false
	}
}

// ask runs f on Loop's goroutine and waits for it, unless Loop has
// returned.
func (c *Client) ask(f func(*loop)) bool {
	done := make(chan struct{})
	if !c.send(func(l *loop) {
		defer close(done)
		f(l)
	}) {
		return false
	}
	<-done
	return true
}

// Run is the Role's run function: Loop reports through h and local, and
// fetches a torrent it holds no data for from ctl, while it runs.
func (c *Client) Run(local *storage.DB[storage.Local], ctl Control) func(context.Context, Holder) error {
	return func(ctx context.Context, h Holder) error {
		run := &attached{ctx: ctx, h: h, ctl: ctl}
		c.send(func(l *loop) { l.attach(run, local) })
		for s := range h.Intent(ctx) {
			c.send(func(l *loop) { l.snapshot(s) })
		}
		c.ask(func(l *loop) { l.detach(run) })
		return nil
	}
}

// Release removes every torrent the client holds, keeping their files, and
// forgets them: the Node let Downloading go.
func (c *Client) Release(context.Context) error {
	var err error
	c.ask(func(l *loop) { err = l.release() })
	return err
}

// Shutdown saves the resume data of every torrent that changed since its
// last save into local, waiting at most Timings.ShutdownSave. It runs as a
// stop hook before local closes.
func (c *Client) Shutdown(local *storage.DB[storage.Local]) func(context.Context) error {
	return func(ctx context.Context) error {
		saved := make(chan struct{})
		if !c.send(func(l *loop) { l.shutdown(local, saved) }) {
			return nil
		}
		timer := time.NewTimer(c.cfg.Timings.ShutdownSave)
		defer timer.Stop()
		select {
		case <-saved:
		case <-timer.C:
			c.cfg.Log.Warn("Some torrents' resume data wasn't saved before the Node stopped. Their files are rechecked when the Node starts")
		case <-ctx.Done():
		}
		return nil
	}
}

// Seeding is how many finished torrents the client holds.
func (c *Client) Seeding() int { return c.held.Load().seeding }

// Path is where file index of torrent h lies on this Node, ErrNotPresent
// until the client finished h.
func (c *Client) Path(h bencode.InfoHash, index int) (string, error) {
	t, ok := c.held.Load().torrents[h]
	if !ok || !t.finished() {
		return "", ErrNotPresent
	}
	return c.pathOf(h, t, index)
}

// Open is where the file ref names lies on this Node, once the torrent is
// complete and the file opens, or why it can't be read now:
//   - *source.Absent{Gone}: the torrent is in neither the intent's live
//     Torrents nor the engine.
//   - *source.Absent{Failed}: the engine failed it, with the engine's
//     text. A torrent that completed and then failed is Failed.
//   - *source.Absent{NotReady}: present but checking, incomplete, or the
//     file won't open, with the open error.
func (c *Client) Open(ref source.Ref) (string, error) {
	r, ok := ref.(source.Torrent)
	if !ok {
		return "", fmt.Errorf("the torrent client can't open a %s source", ref.Kind())
	}
	p := c.held.Load()
	t, ok := p.torrents[r.Hash]
	switch {
	case !p.intended:
		return "", &source.Absent{Reason: source.NotReady, Detail: "the torrent client is starting"}
	case !ok || !t.live && t.handle == 0:
		return "", &source.Absent{Reason: source.Gone}
	case t.failed != "":
		return "", &source.Absent{Reason: source.Failed, Detail: t.failed}
	case t.checking:
		return "", &source.Absent{Reason: source.NotReady, Detail: "the torrent client is checking the torrent's files"}
	case !t.finished():
		return "", &source.Absent{Reason: source.NotReady, Detail: "the torrent is incomplete"}
	}
	path, err := c.pathOf(r.Hash, t, r.File)
	if err != nil {
		return "", err
	}
	f, err := os.Open(path) //nolint:gosec // the client's own path for a torrent's file
	if err != nil {
		return "", &source.Absent{Reason: source.NotReady, Detail: err.Error()}
	}
	_ = f.Close()
	return path, nil
}

func (c *Client) pathOf(h bencode.InfoHash, t held, index int) (string, error) {
	files, err := c.cfg.Engine.Files(t.handle)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(files, func(file engine.File) bool { return file.Index == index })
	if i < 0 {
		return "", fmt.Errorf("torrent %s has no file %d", h, index)
	}
	return filepath.Join(t.folder, filepath.FromSlash(files[i].Path)), nil
}

// Dir is the downloads folder.
func (c *Client) Dir() string { return c.cfg.Dir }

// volume reads the downloads folder's volume, making the folder first so
// the guard knows the volume before the first torrent makes it.
func (c *Client) volume(ctx context.Context) (storing.Volume, error) {
	if err := os.MkdirAll(c.cfg.Dir, 0o750); err != nil {
		return storing.Volume{}, fmt.Errorf("make the downloads folder: %w", err)
	}
	return c.cfg.Volume(ctx, c.cfg.Dir)
}

// Folders is the downloads folder and every save folder outside it, which
// no root may overlap.
func (c *Client) Folders() []string { return c.held.Load().folders }
