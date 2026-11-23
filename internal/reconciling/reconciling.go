// Package reconciling is what SlipStream does after a restore and after
// every pairing: each Node keeps every torrent it holds and reports its
// Holdings, and the Controller adopts them into the library one page per
// Write. Every gate that waits while Reconciling reads State.
package reconciling

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/reconciling/internal/db"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is reconciling's library.db schema set. Its rows name Nodes,
// roots and Torrents, so it is listed after nodes, roots and
// downloadclients.
func Schema() storage.Schema[file] { return storage.NewSchema[file]("reconciling", schemaFiles) }

// Phase is the one answer every gate reads.
type Phase int

const (
	None Phase = iota
	// Reconciling: a restore's Reconciling runs.
	Reconciling
	// Superseded: a restore on another machine replaced this Controller,
	// or an Admin stopped or retired it.
	Superseded
)

// State is the Phase in r, the snapshot of the Write it guards.
func State(ctx context.Context, r storage.Reader[file]) (Phase, error) {
	f, fenced, err := nodes.FenceIn(ctx, r)
	if err != nil {
		return None, err
	}
	if fenced && f.Superseded() {
		return Superseded, nil
	}
	_, err = db.New(r).GetState(ctx)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return None, nil
	case err != nil:
		return None, fmt.Errorf("read the Reconciling state: %w", err)
	}
	return Reconciling, nil
}

// Blocked is whether State is anything but None, in the shape every
// gate's Config takes.
func Blocked(ctx context.Context, r storage.Reader[file]) (bool, error) {
	p, err := State(ctx, r)
	return p != None, err
}

// Cause is why Reconciling began.
type Cause string

const (
	Restore        Cause = "restore"
	ManualRollback Cause = "manual_rollback"
)

// BeginArgs is the restore Begin starts Reconciling for.
type BeginArgs struct {
	// ID is the restore's, stored so its Write runs once.
	ID      string
	Cause   Cause
	Backup  string
	TakenAt time.Time
	// NewMachine waits for the old Controller's folders to be found.
	NewMachine bool
}

// Begin is reconciling's part of the restore's Write: it drops whatever
// Reconciling the restored library held, then every Node keeps every
// torrent until its Holdings are adopted, and every root waits for its
// scan. In place, every root is saved where it is.
func Begin(ctx context.Context, tx *storage.Tx[file], b BeginArgs) error {
	q := db.New(tx)
	for _, del := range []func(context.Context) error{q.DeleteState, q.DeleteReviews, q.DeleteNodes, q.DeleteRoots} {
		if err := del(ctx); err != nil {
			return fmt.Errorf("begin Reconciling: %w", err)
		}
	}
	now := storage.Now()
	var saved storage.Time
	if !b.NewMachine {
		saved = now
	}
	err := q.InsertState(ctx, db.InsertStateParams{
		RestoreID: db.RestoreID(b.ID), StartedAt: now, Cause: string(b.Cause), BackupName: b.Backup, BackupTakenAt: storage.FromTime(b.TakenAt),
		NewMachine: boolInt(b.NewMachine), RehomeSavedAt: saved,
	})
	if err != nil {
		return fmt.Errorf("begin Reconciling: %w", err)
	}
	named, err := nodes.Names(ctx, tx)
	if err != nil {
		return err
	}
	for _, n := range named {
		if err := q.InsertNode(ctx, db.InsertNodeParams{NodeID: n.ID, Cause: string(Restore), Since: now, Round: freshRound()}); err != nil {
			return fmt.Errorf("keep %s's torrents: %w", n.Name, err)
		}
	}
	all, err := roots.All(ctx, tx)
	if err != nil {
		return err
	}
	for _, root := range all {
		if err := q.InsertRoot(ctx, db.InsertRootParams{RootID: root.ID, SavedAt: saved}); err != nil {
			return fmt.Errorf("wait for a root's scan: %w", err)
		}
	}
	return nil
}

// freshRound is a Holdings round no Node was asked for before. A Worker
// sends each round once while it runs, so a restore that asked its
// Workers for round 1 again would wait on each one until it restarted.
func freshRound() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.BigEndian.Uint64(b[:])>>11) + 1
}

// Paired is nodes.Config.Paired: a newly paired Node keeps every torrent
// until its first Holdings are adopted.
func Paired(ctx context.Context, tx *storage.Tx[file], node nodes.ID) error {
	if err := db.New(tx).InsertNode(ctx, db.InsertNodeParams{NodeID: node, Cause: "pairing", Since: storage.Now(), Round: freshRound()}); err != nil {
		return fmt.Errorf("keep a new Node's torrents: %w", err)
	}
	return nil
}

// Pending is nodes.Config.Pending: every Node whose Holdings aren't
// adopted yet.
func Pending(ctx context.Context, r storage.Reader[file]) (map[nodes.ID]bool, error) {
	ids, err := db.New(r).ListPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the Nodes keeping their torrents: %w", err)
	}
	out := make(map[nodes.ID]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// ContinueWithout stops Reconciling waiting for node. It still keeps
// every torrent until its Holdings are adopted.
func ContinueWithout(ctx context.Context, tx *storage.Tx[file], node nodes.ID) error {
	return db.New(tx).ContinueWithout(ctx, db.ContinueWithoutParams{At: storage.Now(), NodeID: node})
}

// Config is what the Service adopts Holdings with.
type Config struct {
	Library *storage.DB[file]
	// Self is the Controller's own Node, whose client a Superseded
	// Controller pauses.
	Self func() role.NodeID
	// Touch derives the Nodes' Downloading intents again.
	Touch func(nodes ...role.NodeID)
	// Metainfo asks node for the metainfo of a torrent it holds in client,
	// nil for the built-in one.
	Metainfo func(ctx context.Context, node role.NodeID, hash bencode.InfoHash, client *downloadclients.ClientID) ([]byte, error)
	// Page is how many held torrents one adoption Write decides, and
	// Max how many one Node's round may list.
	Page, Max int
	// Rehome is the root edit's re-home.
	Rehome func(ctx context.Context, by identity.Admin, id roots.ID, h roots.Rehome) (roots.View, error)
	// Scan starts the Library scan of roots.
	Scan func(ctx context.Context, roots []roots.ID) error
	// Ready is each root its Node reports Ready now.
	Ready func(ctx context.Context) ([]roots.Root, error)
	// Hub wakes Run on roots.changed, when a root may have turned Ready.
	Hub *live.Hub
	// Ending runs in the Write that ends Reconciling: each Grab at
	// fetching takes its adopted torrent or fails.
	Ending func(ctx context.Context, tx *storage.Tx[file]) error
	// Ended runs once that Write commits: queues dispatch, Holds end,
	// Imports run, the disk queue drains and paused tasks catch up.
	Ended func(ctx context.Context)
	// RootWords is each root as a person reads it.
	RootWords func(ctx context.Context, r storage.Reader[file]) (map[roots.ID]string, error)
	// Roster is each Node's link, and the Nodes not in the library that
	// said Hello.
	Roster Roster
	// Look is whether each of files is in folder on node at its size.
	Look func(ctx context.Context, node role.NodeID, folder string, files []downloadclients.LookFile) ([]bool, error)
	// Forgot runs once a Forget commits: its Grabs fail.
	Forgot  func()
	Changed ChangedPublisher
	// History is bound once the History catalog opens, before Run.
	History *entry.Recorder
	Log     *slog.Logger
}

// Roster is what the Reconciling page reads of the roster.
type Roster interface {
	Standing(node nodes.ID) (nodes.Standing, time.Time)
	Unknowns() []nodes.Unknown
}

// Service takes in each Node's Holdings and adopts them. It is
// downloadclients' Keeps.
type Service struct {
	cfg  Config
	wake chan struct{}
	// reported is set in a Write that completes a Node's round, so the
	// commit after it wakes Run.
	reported atomic.Bool
	// scanned is set in a Write that finishes a root's scan.
	scanned atomic.Bool
	mu      sync.Mutex
	// asked is each root whose scan Run started since it last turned
	// Ready, and looked each review list torrent whose Node Run asked
	// about its files, under mu.
	asked  map[roots.ID]bool
	looked map[downloadclients.ID]bool
}

func New(cfg Config) *Service {
	return &Service{cfg: cfg, wake: make(chan struct{}, 1), asked: map[roots.ID]bool{}, looked: map[downloadclients.ID]bool{}}
}

var _ downloadclients.Keeps = (*Service)(nil)

// Wake has Run look again, once what it waits on may have changed.
func (s *Service) Wake() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Applied is nodes.Config.Applied: a Downloading batch that completed a
// Node's round wakes Run.
func (s *Service) Applied(_ role.NodeID, scope string) {
	if scope == string(role.Downloading) && s.reported.Swap(false) {
		s.Wake()
	}
}

// ScanEnded wakes Run once a Write that finished a root's scan
// committed.
func (s *Service) ScanEnded() {
	if s.scanned.Swap(false) {
		s.Wake()
	}
}

// Blocked is downloadclients' vanish gate.
func (s *Service) Blocked(ctx context.Context, r storage.Reader[file]) (bool, error) {
	return Blocked(ctx, r)
}

// KeepOf is what node's Downloading intent keeps: everything until its
// Holdings are adopted, with the round they are asked for, and afterwards
// each torrent kept with no record. A Superseded Controller keeps
// everything on every Node, and pauses its own.
func (s *Service) KeepOf(ctx context.Context, r storage.Reader[file], node role.NodeID) (downloadclients.Keep, error) {
	if f, fenced, err := nodes.FenceIn(ctx, r); err != nil || (fenced && f.Superseded()) {
		return downloadclients.Keep{All: true, PauseAll: node == s.cfg.Self()}, err
	}
	q := db.New(r)
	row, err := q.GetNode(ctx, node)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return downloadclients.Keep{}, nil
	case err != nil:
		return downloadclients.Keep{}, fmt.Errorf("read what a Node keeps: %w", err)
	}
	var k downloadclients.Keep
	if row.AdoptedAt.IsZero() {
		k.All, k.Round = true, uint64(row.Round) //nolint:gosec // round is at least 1
	}
	hashes, err := q.ListKeepHashes(ctx, node)
	if err != nil {
		return downloadclients.Keep{}, fmt.Errorf("read what a Node keeps: %w", err)
	}
	for _, h := range hashes {
		k.Hashes = append(k.Hashes, bencode.InfoHash(h))
	}
	return k, nil
}

// retryAfter is how long Run waits before it asks a Node that couldn't
// answer for metainfo again: the link redials within it.
const retryAfter = time.Minute

// Run adopts each Node's Holdings once its round is in, and ends
// Reconciling once nothing is left to wait for.
func (s *Service) Run(ctx context.Context) error {
	if err := s.announce(ctx); err != nil && ctx.Err() == nil {
		s.cfg.Log.Error("SlipStream couldn't record the restore in History", "err", err)
	}
	woken := s.cfg.Hub.Watch(ctx, roots.ChangedEvent)
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		retry, err := s.settle(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			s.cfg.Log.Warn("SlipStream couldn't adopt a Node's torrents", "err", err)
			retry = true
		}
		timer.Stop()
		if retry {
			timer.Reset(retryAfter)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-s.wake:
		case <-woken:
		case <-timer.C:
		}
	}
}

// settle runs passes until one changes nothing, then tries to end
// Reconciling. retry is a Node that couldn't answer.
func (s *Service) settle(ctx context.Context) (retry bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		changed, again, err := s.pass(ctx)
		retry = retry || again
		if err != nil {
			return retry, err
		}
		if !changed {
			if err := s.scan(ctx); err != nil {
				return retry, err
			}
			if err := s.look(ctx); err != nil {
				return retry, err
			}
			return retry, s.tryEnd(ctx)
		}
	}
}

func boolInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
