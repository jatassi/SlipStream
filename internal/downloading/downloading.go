// Package downloading is the Downloading Role's holder half: each Node's
// built-in torrent client, which keeps the torrents its intent lists,
// reports their progress, and reports how each Grab's download ended.
package downloading

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers"
	"google.golang.org/protobuf/proto"
)

// Def is the Downloading Role. A change in which torrents a Node's client
// holds is notable, since Admins see each torrent's title.
var Def = role.Def[*dlv1.Intent, *dlv1.Facts, *dlv1.Result]{Name: role.Downloading, Notable: torrentsDiffer}

func torrentsDiffer(last, next *dlv1.Facts) bool {
	if len(last.GetTorrents()) != len(next.GetTorrents()) {
		return true
	}
	held := map[bencode.InfoHash]bool{}
	for _, t := range last.GetTorrents() {
		held[infoHash(t.GetInfoHash())] = true
	}
	for _, t := range next.GetTorrents() {
		if !held[infoHash(t.GetInfoHash())] {
			return true
		}
	}
	return false
}

type Holder = role.Holder[*dlv1.Intent, *dlv1.Facts, *dlv1.Result]

type Timings struct {
	// Facts is the least time between two sends of the torrents' progress.
	Facts time.Duration `yaml:"facts"`
	// AnnounceFloor is the least time between two regular announces to a
	// tracker, whatever interval the tracker answers.
	AnnounceFloor time.Duration `yaml:"announce_floor"`
}

func DefaultTimings() Timings { return Timings{Facts: time.Second, AnnounceFloor: 5 * time.Minute} }

// ErrNotPresent is a torrent this Node's client hasn't finished.
var ErrNotPresent = transfers.ErrNotPresent

type Config struct {
	Engine engine.Engine
	// Dir holds each torrent's save path, named by its info hash in hex.
	Dir     string
	Timings Timings
	Log     *slog.Logger
}

// Client is a Node's built-in client, one per Node from the moment its
// engine opens, claimed or not. Drain is the engine's only Events reader;
// the Role's run function and Path read what it learned.
type Client struct {
	cfg  Config
	poke chan struct{}

	mu       sync.Mutex
	byHash   map[bencode.InfoHash]engine.Handle
	hashOf   map[engine.Handle]bencode.InfoHash
	finished map[bencode.InfoHash]bool
	failed   map[bencode.InfoHash]string
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:      cfg,
		poke:     make(chan struct{}, 1),
		byHash:   map[bencode.InfoHash]engine.Handle{},
		hashOf:   map[engine.Handle]bencode.InfoHash{},
		finished: map[bencode.InfoHash]bool{},
		failed:   map[bencode.InfoHash]string{},
	}
}

// Drain reads the engine's events until the engine closes or ctx ends.
func (c *Client) Drain(ctx context.Context) error {
	events := c.cfg.Engine.Events()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			c.note(ev)
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *Client) note(ev engine.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch ev := ev.(type) {
	case engine.Finished:
		c.finished[c.hashOf[ev.Torrent]] = true
	case engine.TorrentFailed:
		c.failed[c.hashOf[ev.Torrent]] = ev.Err.Text
	case engine.Dropped:
		for h, handle := range c.byHash {
			if s, err := c.cfg.Engine.Status(handle); err == nil && s.Wanted > 0 && s.Done == s.Wanted {
				c.finished[h] = true
			}
		}
	default:
		return
	}
	select {
	case c.poke <- struct{}{}:
	default:
	}
}

// Seeding is how many finished torrents the client holds.
func (c *Client) Seeding() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for h := range c.byHash {
		if c.finished[h] {
			n++
		}
	}
	return n
}

// Path is where file index of torrent h lies on this Node, ErrNotPresent
// until the client finished h.
func (c *Client) Path(h bencode.InfoHash, index int) (string, error) {
	c.mu.Lock()
	handle, finished := c.byHash[h], c.finished[h]
	c.mu.Unlock()
	if !finished {
		return "", ErrNotPresent
	}
	files, err := c.cfg.Engine.Files(handle)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(files, func(f engine.File) bool { return f.Index == index })
	if i < 0 {
		return "", fmt.Errorf("torrent %s has no file %d", h, index)
	}
	return filepath.Join(c.savePath(h), filepath.FromSlash(files[i].Path)), nil
}

// Dir is the folder each torrent's save path is in.
func (c *Client) Dir() string { return c.cfg.Dir }

func (c *Client) savePath(h bencode.InfoHash) string { return filepath.Join(c.cfg.Dir, h.String()) }

// Run is the Role's run function, reporting through local. It adds each
// torrent its intent lists that the client lacks, and reports each Grab's
// outcome once per run: a restarted Node reports again, which the
// Controller ignores.
func (c *Client) Run(local *storage.DB[storage.Local]) func(context.Context, Holder) error {
	return func(ctx context.Context, h Holder) error {
		snapshots := latest(h.Intent(ctx))
		facts := time.NewTicker(c.cfg.Timings.Facts)
		defer facts.Stop()
		reported := map[storage.UUID]bool{}
		var now role.Snapshot[*dlv1.Intent]
		sent := &dlv1.Facts{}
		for {
			select {
			case s := <-snapshots:
				now = s
			case <-c.poke:
			case <-facts.C:
				if now.Intent == nil {
					continue
				}
				// Each send tells Admins the Node changed, so only a change is sent.
				if f := c.facts(now.Intent); !proto.Equal(f, sent) {
					h.SetFacts(f)
					sent = f
				}
				continue
			case <-ctx.Done():
				return nil
			}
			if now.Intent == nil {
				continue
			}
			c.keep(now)
			if err := c.report(ctx, local, h, now.Intent, reported); err != nil {
				return err
			}
		}
	}
}

// latest pumps seq into a channel that holds only the newest snapshot, so
// the run loop never falls behind its intent.
func latest(seq iter.Seq[role.Snapshot[*dlv1.Intent]]) <-chan role.Snapshot[*dlv1.Intent] {
	ch := make(chan role.Snapshot[*dlv1.Intent], 1)
	go func() {
		for s := range seq {
			select {
			case <-ch:
			default:
			}
			ch <- s
		}
	}()
	return ch
}

// keep makes the client hold the torrents the intent lists: it removes
// each one the intent no longer lists, keeping its files, and adds each
// one the client lacks. When it may not start, out of step or draining,
// it adds only a torrent whose save path exists, which it held before:
// that resumes seeding and starts nothing new.
func (c *Client) keep(s role.Snapshot[*dlv1.Intent]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	listed := map[bencode.InfoHash]bool{}
	for _, t := range s.Intent.GetTorrents() {
		listed[infoHash(t.GetInfoHash())] = true
	}
	for h := range c.byHash {
		if !listed[h] {
			c.remove(h)
		}
	}
	for h := range c.failed {
		if !listed[h] {
			delete(c.failed, h)
		}
	}
	for _, t := range s.Intent.GetTorrents() {
		h := infoHash(t.GetInfoHash())
		if _, ok := c.byHash[h]; ok || c.failed[h] != "" {
			continue
		}
		if _, err := os.Stat(c.savePath(h)); !s.MayStart() && err != nil {
			continue
		}
		handle, err := c.start(t.GetMetainfo(), c.savePath(h))
		if err != nil {
			c.failed[h] = err.Error()
			c.cfg.Log.Error("add a torrent", "info_hash", h.String(), "err", err.Error())
			continue
		}
		c.byHash[h], c.hashOf[handle] = handle, h
	}
}

// start adds a torrent and resumes it, since the engine adds paused.
func (c *Client) start(metainfo []byte, savePath string) (engine.Handle, error) {
	handle, err := c.cfg.Engine.Add(engine.AddParams{Metainfo: metainfo, SavePath: savePath})
	if err != nil {
		return 0, err
	}
	return handle, c.cfg.Engine.Resume(handle)
}

// Release removes every torrent the client holds, keeping their files: the
// Node let Downloading go.
func (c *Client) Release(context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for h := range c.byHash {
		c.remove(h)
	}
	clear(c.failed)
	return nil
}

// remove takes torrent h out of the engine, leaving its files where they
// lie. The caller holds c.mu.
func (c *Client) remove(h bencode.InfoHash) {
	handle := c.byHash[h]
	if err := c.cfg.Engine.Remove(handle, false); err != nil {
		c.cfg.Log.Error("remove a torrent", "info_hash", h.String(), "err", err.Error())
	}
	delete(c.byHash, h)
	delete(c.hashOf, handle)
	delete(c.finished, h)
}

func (c *Client) report(ctx context.Context, local *storage.DB[storage.Local], h Holder, intent *dlv1.Intent, reported map[storage.UUID]bool) error {
	for _, t := range intent.GetTorrents() {
		grab, err := storage.UUIDFromBytes(t.GetGrabId())
		if err != nil {
			return fmt.Errorf("intent: grab id: %w", err)
		}
		if reported[grab] {
			continue
		}
		res, err := c.result(infoHash(t.GetInfoHash()))
		if err != nil {
			return err
		}
		if res == nil {
			continue
		}
		res.GrabId = t.GetGrabId()
		if err := local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
			return h.Report(ctx, tx, res)
		}); err != nil {
			return err
		}
		reported[grab] = true
	}
	return nil
}

// result is how torrent h ended, nil while it downloads.
func (c *Client) result(h bencode.InfoHash) (*dlv1.Result, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if msg := c.failed[h]; msg != "" {
		return &dlv1.Result{Outcome: &dlv1.Result_Failed{Failed: &dlv1.Failed{InfoHash: h[:], Error: msg}}}, nil
	}
	if !c.finished[h] {
		return nil, nil
	}
	files, err := c.cfg.Engine.Files(c.byHash[h])
	if err != nil {
		return nil, fmt.Errorf("list torrent %s's files: %w", h, err)
	}
	completed := &dlv1.Completed{InfoHash: h[:]}
	for _, f := range files {
		completed.Files = append(completed.Files, &dlv1.File{Index: int32(f.Index), Path: f.Path, Size: f.Size}) //nolint:gosec // a torrent's file index fits
	}
	return &dlv1.Result{Outcome: &dlv1.Result_Completed{Completed: completed}}, nil
}

// facts is the progress of each torrent the intent lists and the client
// holds, in the intent's info hash order.
func (c *Client) facts(intent *dlv1.Intent) *dlv1.Facts {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := &dlv1.Facts{}
	for _, t := range intent.GetTorrents() {
		h := infoHash(t.GetInfoHash())
		handle, ok := c.byHash[h]
		if n := len(out.Torrents); !ok || n > 0 && infoHash(out.Torrents[n-1].GetInfoHash()) == h {
			continue
		}
		s, err := c.cfg.Engine.Status(handle)
		if err != nil {
			continue
		}
		out.Torrents = append(out.Torrents, &dlv1.TorrentFacts{InfoHash: h[:], State: s.State.String(), Done: s.Done, Wanted: s.Wanted, Peers: int32(s.Peers), Uploaded: s.Totals.Uploaded}) //nolint:gosec // a peer count fits
	}
	return out
}

func infoHash(b []byte) bencode.InfoHash {
	var h bencode.InfoHash
	copy(h[:], b)
	return h
}
