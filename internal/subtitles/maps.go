package subtitles

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/backoff"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles/internal/db"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

// defaultMapSlots is subtitles.mapSlots' default: one audio scan at once
// on a Node.
const defaultMapSlots = 1

// mapDrain builds the Speech Maps that are due on each Storage Node: at
// most the Node's map_slots calls in flight, the next due files of its
// roots by rank and newest first. A build counts in flight until the
// Node answers built or failed, however many calls that takes.
type mapDrain struct {
	s *Subtitles

	mu     sync.Mutex
	flying map[role.NodeID]int
	// resting is when each Node that answered node_fault is asked again.
	resting map[role.NodeID]time.Time
	ring    chan struct{}
	builds  sync.WaitGroup
	rand    sync.Mutex
}

func newMapDrain(s *Subtitles) *mapDrain {
	return &mapDrain{s: s, flying: map[role.NodeID]int{}, resting: map[role.NodeID]time.Time{}, ring: make(chan struct{}, 1)}
}

// soon passes again after Timings.WakeDelay, by when the Write that made
// maps due has committed.
func (m *mapDrain) soon() { time.AfterFunc(m.s.cfg.Timings.WakeDelay, m.wake) }

func (m *mapDrain) wake() {
	select {
	case m.ring <- struct{}{}:
	default:
	}
}

// run passes at once, on every wake and every Timings.KeeperPoll, until
// ctx ends and every build it started has returned.
func (m *mapDrain) run(ctx context.Context) {
	defer m.builds.Wait()
	poll := time.NewTicker(m.s.cfg.Timings.KeeperPoll)
	defer poll.Stop()
	for {
		if err := m.pass(ctx); err != nil && ctx.Err() == nil {
			m.s.cfg.Log.WarnContext(ctx, "Scanning audio for subtitle timing failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-m.ring:
		case <-poll.C:
		}
	}
}

// pass starts the due builds each Node has room for.
func (m *mapDrain) pass(ctx context.Context) error {
	now := m.s.cfg.Clock()
	var due []roots.ID
	err := m.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if err := q.RetryMaps(ctx, storage.FromTime(now)); err != nil {
			return err
		}
		var err error
		due, err = q.ListMapRoots(ctx)
		return err
	})
	if err != nil {
		return err
	}
	byNode := map[role.NodeID][]roots.ID{}
	for _, id := range due {
		r, _, err := m.s.cfg.Roots.StateOf(ctx, id)
		if err != nil {
			return err
		}
		byNode[r.Node] = append(byNode[r.Node], id)
	}
	for node, rs := range byNode {
		if err := m.start(ctx, node, rs, now); err != nil {
			return err
		}
	}
	return nil
}

// due is one file whose build may start.
type due struct {
	row  db.SubtitlesFile
	path storing.Rel
}

// start starts node's next builds over its roots rs, as many as it has
// room for.
func (m *mapDrain) start(ctx context.Context, node role.NodeID, rs []roots.ID, now time.Time) error {
	m.mu.Lock()
	flying, resting := m.flying[node], m.resting[node].After(now)
	m.mu.Unlock()
	if resting {
		return nil
	}
	var next []due
	var slots int
	err := m.s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		n, err := q.GetMapSlots(ctx, node)
		if err != nil {
			return err
		}
		slots = cmp.Or(int(n), defaultMapSlots)
		if slots <= flying {
			return nil
		}
		var rows []db.SubtitlesFile
		for _, root := range rs {
			got, err := q.NextMaps(ctx, db.NextMapsParams{RootID: root, N: int64(slots - flying)})
			if err != nil {
				return err
			}
			rows = append(rows, got...)
		}
		slices.SortStableFunc(rows, func(a, b db.SubtitlesFile) int {
			return cmp.Or(cmp.Compare(a.MapRank, b.MapRank), b.MapSince.Compare(a.MapSince))
		})
		next, err = m.placed(ctx, r, rows[:min(len(rows), slots-flying)])
		return err
	})
	if err != nil {
		return err
	}
	for _, d := range next {
		var started int64
		err := m.s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			var err error
			started, err = db.New(tx).StartMap(ctx, d.row.FileID)
			return err
		})
		if err != nil {
			return err
		}
		if started == 0 {
			continue
		}
		m.mu.Lock()
		m.flying[node]++
		m.mu.Unlock()
		m.builds.Go(func() { m.build(ctx, node, d, slots) })
	}
	return nil
}

// placed is the rows whose bytes sit at their name, with their paths. A
// file mid-move waits.
func (m *mapDrain) placed(ctx context.Context, r storage.Reader[storage.Library], rows []db.SubtitlesFile) ([]due, error) {
	ids := make([]files.ID, len(rows))
	for i, row := range rows {
		ids[i] = row.FileID
	}
	subjects, err := m.s.cfg.Files.SubjectsOf(ctx, r, ids)
	if err != nil {
		return nil, err
	}
	var out []due
	for _, row := range rows {
		if i := slices.IndexFunc(subjects, func(s files.Subject) bool { return s.ID == row.FileID }); i >= 0 && subjects[i].AtName {
			out = append(out, due{row: row, path: subjects[i].Path})
		}
	}
	return out, nil
}

// build asks node for d's map until it answers built or failed, and
// records the answer. A call that fails makes the map due again.
func (m *mapDrain) build(ctx context.Context, node role.NodeID, d due, slots int) {
	defer func() {
		m.mu.Lock()
		m.flying[node]--
		m.mu.Unlock()
		m.wake()
	}()
	calls := m.s.cfg.Storage(node)
	ask := storing.MapBuild{File: d.row.FileID.UUID, Root: d.row.RootID.UUID, Path: d.path, Stream: -1, Slots: slots, Rank: int(d.row.MapRank)}
	for {
		a, err := calls.Build(ctx, ask)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			m.s.cfg.Log.WarnContext(ctx, "An audio scan call failed", "file", d.row.FileID, "node", node, "error", err)
			m.record(ctx, func(q *db.Queries) error { return q.MapDueAgain(ctx, d.row.FileID) })
			return
		}
		switch a.State {
		case storing.MapBuilt:
			m.built(ctx, node, d.row, a)
			return
		case storing.MapFailed:
			m.failed(ctx, node, d.row, a)
			return
		case storing.MapBuilding, storing.MapQueued:
		}
	}
}

func (m *mapDrain) record(ctx context.Context, fn func(q *db.Queries) error) {
	err := m.s.cfg.Library.Write(ctx, func(_ context.Context, tx *storage.Tx[storage.Library]) error { return fn(db.New(tx)) })
	if err != nil && ctx.Err() == nil {
		m.s.cfg.Log.WarnContext(ctx, "An audio scan's result wasn't recorded", "error", err)
	}
}

func (m *mapDrain) built(ctx context.Context, node role.NodeID, row db.SubtitlesFile, a storing.MapAnswer) {
	id, err := json.Marshal(a.Identity)
	if err != nil {
		panic(fmt.Sprintf("subtitles: a map identity doesn't encode: %v", err))
	}
	m.record(ctx, func(q *db.Queries) error {
		err := q.MapBuilt(ctx, db.MapBuiltParams{
			NodeID: node, Stream: new(int64(a.Stream)), Identity: id, Version: new(int64(speech.FormatVersion)), Frames: new(int64(a.Frames)),
			Density: &a.Density, BuiltAt: storage.FromTime(m.s.cfg.Clock()), FileID: row.FileID,
		})
		if err != nil {
			return err
		}
		return q.NodeBuilt(ctx, node)
	})
	m.mu.Lock()
	delete(m.resting, node)
	m.mu.Unlock()
	m.s.keeper.soon(row.FileID)
	m.s.drain.soon()
}

var faultNames = map[storing.MapFault]string{storing.NoAudio: "no_audio", storing.DecodeFailed: "decode_failed", storing.NodeFault: "node_fault"}

// failed records a failed build. No audio holds until the file changes,
// a decode failure backs off, and a Node that can't scan at all keeps the
// map due, raises its fault row and rests for Timings.NodeRetry.
func (m *mapDrain) failed(ctx context.Context, node role.NodeID, row db.SubtitlesFile, a storing.MapAnswer) {
	now := m.s.cfg.Clock()
	if a.Fault == storing.NodeFault {
		m.mu.Lock()
		m.resting[node] = now.Add(m.s.cfg.Timings.NodeRetry)
		m.mu.Unlock()
		time.AfterFunc(m.s.cfg.Timings.NodeRetry, m.wake)
		m.record(ctx, func(q *db.Queries) error {
			if err := q.MapDueAgain(ctx, row.FileID); err != nil {
				return err
			}
			return q.NodeFault(ctx, db.NodeFaultParams{NodeID: node, FaultAt: storage.FromTime(now), FaultWords: &a.Words})
		})
		return
	}
	var next storage.Time
	if a.Fault == storing.DecodeFailed {
		next = storage.FromTime(now.Add(m.backoff(int(row.MapTries))))
	}
	m.record(ctx, func(q *db.Queries) error {
		return q.MapFailed(ctx, db.MapFailedParams{Fault: new(faultNames[a.Fault]), Words: some(a.Words), NodeID: node, NextAt: next, FileID: row.FileID})
	})
}

func (m *mapDrain) backoff(tries int) time.Duration {
	m.rand.Lock()
	defer m.rand.Unlock()
	return backoff.Delay(m.s.cfg.Timings.MapRetryMin, m.s.cfg.Timings.MapRetryMax, tries, m.s.cfg.Rand)
}

// markMapsDue makes the maps of ids due where a want is unmet or a
// Sidecar waits to be timed, and wakes the drain.
func (s *Subtitles) markMapsDue(ctx context.Context, q *db.Queries, ids []files.ID) error {
	if len(ids) == 0 {
		return nil
	}
	if err := q.MarkMapsDue(ctx, ids); err != nil {
		return err
	}
	s.maps.soon()
	return nil
}

// replaced is a file whose bytes changed in place: its map is built
// again and its Sidecars are timed against it.
func (s *Subtitles) replaced(ctx context.Context, q *db.Queries, f files.ID) error {
	if err := q.MarkTimingDue(ctx, f); err != nil {
		return err
	}
	if err := q.ForgetMap(ctx, f); err != nil {
		return err
	}
	return s.markMapsDue(ctx, q, []files.ID{f})
}

// errMapDue is a file whose map isn't current: it is due again, and what
// needed it waits for the build.
var errMapDue = errors.New("the file's audio scan isn't current")

// speechMap is sub's current map, read from the Node that built it, and
// its encoding, which a holder times against. A map the Node holds no
// longer, or that another Node built, is due again.
func (s *Subtitles) speechMap(ctx context.Context, row db.SubtitlesFile, sub files.Subject) (timing.Reference, []byte, error) {
	if row.MapState != "built" {
		return timing.Reference{}, nil, errMapDue
	}
	r, _, err := s.cfg.Roots.StateOf(ctx, sub.Root)
	if err != nil {
		return timing.Reference{}, nil, err
	}
	var id storing.Identity
	if err := json.Unmarshal(row.MapIdentity, &id); err != nil {
		return timing.Reference{}, nil, fmt.Errorf("file %s's map identity: %w", sub.ID, err)
	}
	var b []byte
	if r.Node == row.MapNodeID {
		b, err = s.cfg.Storage(r.Node).ReadMap(ctx, sub.ID.UUID, sub.Root.UUID, sub.Path, id)
	}
	if r.Node != row.MapNodeID || errors.Is(err, storing.ErrStale) {
		err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return db.New(tx).MapStale(ctx, sub.ID)
		})
		s.maps.soon()
		return timing.Reference{}, nil, errors.Join(errMapDue, err)
	}
	if err != nil {
		return timing.Reference{}, nil, callError(err)
	}
	sm, err := speech.Decode(b)
	if err != nil {
		return timing.Reference{}, nil, fmt.Errorf("file %s's map: %w", sub.ID, err)
	}
	return timing.NewReference(sm.Bits(), sm.Header.Frames), b, nil
}

// identityOf is what a write beside sub names its video as: the identity
// its map was built from, else what a scan or an import last read.
func identityOf(row db.SubtitlesFile, sub files.Subject) storing.Identity {
	var id storing.Identity
	if row.MapState == "built" && json.Unmarshal(row.MapIdentity, &id) == nil {
		return id
	}
	if sub.Seen != nil {
		id.Seen = *sub.Seen
	}
	id.Size = sub.Size
	return id
}
