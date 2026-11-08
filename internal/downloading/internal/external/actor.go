// Package external is the Node's actor for one external Download Client:
// the only goroutine that calls it, polls it, and writes what the Node
// keeps and reports of its torrents.
package external

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

type Holder = role.Holder[*dlv1.Intent, *dlv1.Facts, *dlv1.Report]

// Control is the Controller as the actor calls it: a Torrent's metainfo
// and the client's secrets.
type Control = downloadingv1connect.DownloadingControlServiceHandler

// Share is one client's part of the intent: its settings, its torrents and
// discards, and the Node's thresholds the built-in client reads too.
type Share struct {
	Client   *dlv1.Client
	Torrents []*dlv1.Torrent
	Discards []*dlv1.Discard
	// StalledAfter is the Node's Stalled after, UnregisteredGap the
	// unregistered release fault's gap.
	StalledAfter, UnregisteredGap time.Duration
	// MayStart is the Role in step and not draining: the actor adds no
	// torrent otherwise.
	MayStart bool
}

type Config struct {
	ID   storage.UUID
	Kind clients.Kind
	Open func(clients.Session) clients.Client
	HTTP *http.Client
	// Subfolder is a Content Type's name as a folder name, as the built-in
	// client names it.
	Subfolder func(contentType string) string
	// Links is how many links a file on this Node has.
	Links   func(path string) (uint64, error)
	Timings Timings
	Local   *storage.DB[storage.Local]
	Holder  Holder
	Control Control
	Rand    *rand.Rand
	Log     *slog.Logger
}

// Actor is one client's actor. Every method but Start asks its goroutine,
// or reads what it published.
type Actor struct {
	cfg  Config
	in   chan func(*state)
	pub  atomic.Pointer[published]
	halt context.CancelFunc
	done chan struct{}
}

// Start runs an actor for s's client until Halt or Stop, or until ctx
// ends.
func Start(ctx context.Context, cfg Config, s Share) *Actor {
	ctx, cancel := context.WithCancel(ctx)
	a := &Actor{cfg: cfg, in: make(chan func(*state)), halt: cancel, done: make(chan struct{})}
	a.pub.Store(&published{byHash: map[bencode.InfoHash]heldFiles{}})
	go a.run(ctx, s)
	return a
}

func (a *Actor) run(ctx context.Context, share Share) {
	defer close(a.done)
	s := newState(ctx, a)
	s.update(ctx, share)
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for {
		now := time.Now()
		if !s.next.IsZero() && !now.Before(s.next) {
			s.pass(ctx, now)
		}
		timer.Stop()
		if !s.next.IsZero() {
			timer.Reset(max(time.Until(s.next), 0))
		}
		select {
		case f := <-a.in:
			f(s)
		case <-timer.C:
		case <-ctx.Done():
			s.close()
			return
		}
	}
}

// send runs f on the actor's goroutine, unless it has returned.
func (a *Actor) send(f func(*state)) bool {
	select {
	case a.in <- f:
		return true
	case <-a.done:
		return false
	}
}

// ask runs f on the actor's goroutine and waits for it, unless the actor
// has returned.
func (a *Actor) ask(f func(*state)) bool {
	done := make(chan struct{})
	if !a.send(func(s *state) {
		defer close(done)
		f(s)
	}) {
		return false
	}
	<-done
	return true
}

// Update hands the actor its share of a new intent. A new settings version
// drops the session and the secrets and clears a refused stop.
func (a *Actor) Update(s Share) { a.send(func(st *state) { st.update(st.ctx, s) }) }

// Halt ends the actor, keeping its local rows for the next run.
func (a *Actor) Halt() {
	a.halt()
	<-a.done
}

// Stop ends the actor and deletes its local rows, keeping every torrent in
// the client: the client left the intent.
func (a *Actor) Stop(ctx context.Context) error {
	a.Halt()
	return a.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).DeleteClientExternals(ctx, a.cfg.ID)
	})
}

// Watching polls at the live cadence until ctx ends.
func (a *Actor) Watching(ctx context.Context) {
	if !a.send(func(s *state) { s.watch(1) }) {
		return
	}
	go func() {
		<-ctx.Done()
		a.send(func(s *state) { s.watch(-1) })
	}()
}

var errStopped = errors.New("the client's actor has stopped")

// Check polls now, refused or not, and returns once the ClientPoll
// carrying token is in the outbox.
func (a *Actor) Check(ctx context.Context, token []byte) error {
	reply := make(chan error, 1)
	if !a.send(func(s *state) { s.check(token, reply) }) {
		return errStopped
	}
	select {
	case err := <-reply:
		return err
	case <-ctx.Done():
		return fmt.Errorf("wait for the client's poll: %w", ctx.Err())
	}
}

// Inspect is torrent id's files with their progress and its tracker
// messages, nil when the actor holds no torrent id.
func (a *Actor) Inspect(ctx context.Context, id storage.UUID) *dlv1.InspectResponse {
	var res *dlv1.InspectResponse
	a.ask(func(s *state) { res = s.inspect(ctx, id) })
	return res
}

// Act announces or rechecks each torrent ids name, answering for each in
// order. A discard and a torrent the actor doesn't hold are not held, and
// every one is refused while the client's polls don't succeed.
func (a *Actor) Act(ctx context.Context, act clients.Act, ids []storage.UUID) []*dlv1.ActResult {
	var res []*dlv1.ActResult
	if !a.ask(func(s *state) { res = s.act(ctx, act, ids) }) {
		res = make([]*dlv1.ActResult, len(ids))
		for i, id := range ids {
			res[i] = &dlv1.ActResult{Id: id.Bytes(), Refused: dlv1.Refusal_REFUSAL_NOT_HELD}
		}
	}
	return res
}

// Frees is what deleting each torrent's files would free: the bytes done of
// each file with no other link. An attached torrent frees nothing, since
// removing it never deletes its files.
func (a *Actor) Frees(ctx context.Context, ids []storage.UUID) []*dlv1.Freed {
	var copies []fileCopy
	a.ask(func(s *state) {
		for _, id := range ids {
			copies = append(copies, s.copyOf(ctx, id))
		}
	})
	out := make([]*dlv1.Freed, len(ids))
	for i, id := range ids {
		out[i] = &dlv1.Freed{Id: id.Bytes()}
		if i < len(copies) {
			out[i] = a.frees(copies[i])
		}
	}
	return out
}

// Path is where file index of torrent h lies on this Node, false until the
// actor has sent its Completion or once the client stopped listing it.
func (a *Actor) Path(h bencode.InfoHash, index int) (string, bool) {
	f, ok := a.pub.Load().byHash[h]
	if !ok || !f.completed || f.vanished || index < 0 || index >= len(f.paths) || f.paths[index] == "" {
		return "", false
	}
	return f.paths[index], true
}

// Open is where the file r names lies on this Node, or why it can't be
// read now: Gone once the client stopped listing the torrent and its
// Vanished is in the outbox, NotReady before its Completion or while the
// file won't open.
func (a *Actor) Open(r source.Torrent) (string, error) {
	p := a.pub.Load()
	f, ok := p.byHash[r.Hash]
	switch {
	case !ok || f.vanished:
		return "", &source.Absent{Reason: source.Gone}
	case !p.polled:
		return "", &source.Absent{Reason: source.NotReady, Detail: "the Node hasn't polled the client yet"}
	case !f.completed:
		return "", &source.Absent{Reason: source.NotReady, Detail: "the torrent is incomplete"}
	}
	path, ok := a.Path(r.Hash, r.File)
	if !ok {
		return "", fmt.Errorf("torrent %s has no file %d", r.Hash, r.File)
	}
	file, err := os.Open(path) //nolint:gosec // a torrent's file as the client's Client folders place it
	if err != nil {
		return "", &source.Absent{Reason: source.NotReady, Detail: err.Error()}
	}
	_ = file.Close()
	return path, nil
}

// Folder is the client's download folder as this Node sees it, false
// before the actor knows it.
func (a *Actor) Folder() (string, bool) {
	f := a.pub.Load().folder
	return f, f != ""
}

// Live is each of the client's torrents as its last poll saw it, and its
// payload rates summed.
func (a *Actor) Live() (rows []*dlv1.Live, down, up int64) {
	p := a.pub.Load()
	return p.live, p.down, p.up
}

// fileCopy is a torrent's files as this Node sees them, with the bytes
// done of each.
type fileCopy struct {
	id       storage.UUID
	attached bool
	paths    []string
	done     []int64
}

func (a *Actor) frees(c fileCopy) *dlv1.Freed {
	f := &dlv1.Freed{Id: c.id.Bytes()}
	if c.attached {
		return f
	}
	for i, path := range c.paths {
		n, err := a.cfg.Links(path)
		switch {
		case err != nil:
		case n > 1:
			f.Linked = true
		default:
			f.Bytes += c.done[i]
		}
	}
	return f
}
