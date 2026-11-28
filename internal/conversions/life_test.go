package conversions

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

// life is a library with one Work, two Converting Nodes, Attic and Den, a
// root on Attic, and the Target "720p mobile" whose Conversions the tests
// move along the stage table. Its source files are on the root. Targets
// opens before the Work is added, since no Content Type is registered.
type life struct {
	lib       *storage.DB[storage.Library]
	c         *Conversions
	attic     role.NodeID
	den       role.NodeID
	root      roots.ID
	work      works.WorkID
	target    targets.TargetID
	settings  encode.Settings
	version   int
	media     media.Record
	files     *located
	transfers *wanted
	importer  *imported
	history   *entrytest.Recorder
	now       time.Time
	// key is the proof every seeded Conversion runs under.
	key encode.Key
}

// located is Files: each source file's Located.
type located struct {
	mu sync.Mutex
	at map[files.ID]files.Located
}

func (l *located) Sourced(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.WorkID][]files.Sourced, error) {
	return nil, nil
}

func (l *located) Located(_ context.Context, _ storage.Reader[storage.Library], id files.ID) (files.Located, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, ok := l.at[id]
	if !ok {
		return files.Located{}, storage.ErrNoRows
	}
	return f, nil
}

// wanted is Transfers: every Request, by job, and whether its last one
// ended it.
type wanted struct {
	mu   sync.Mutex
	last map[string]transfers.Request
	ids  map[string]transfers.ID
}

func (w *wanted) Want(_ context.Context, _ *storage.Tx[storage.Library], r transfers.Request) (transfers.ID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.last[r.Job] = r
	if len(r.Files) == 0 {
		return transfers.ID{}, nil
	}
	if _, ok := w.ids[r.Job]; !ok {
		w.ids[r.Job] = transfers.ID{UUID: storage.NewUUID()}
	}
	return w.ids[r.Job], nil
}

func (w *wanted) Touch() {}

// holding is whether job's last Request holds files.
func (w *wanted) holding(job string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	r, ok := w.last[job]
	return ok && len(r.Files) > 0
}

func (w *wanted) asked(job string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, ok := w.last[job]
	return ok
}

// imported is the Importer: the hand-offs it was given and the Imports it
// ended. placing refuses End as a placing file would.
type imported struct {
	mu      sync.Mutex
	started []Handoff
	ended   []ID
	placing bool
}

func (i *imported) Start(_ context.Context, _ *storage.Tx[storage.Library], h Handoff) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.started = append(i.started, h)
	return nil
}

func (i *imported) End(_ context.Context, _ *storage.Tx[storage.Library], id ID) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.placing {
		return ErrPlacing
	}
	i.ended = append(i.ended, id)
	return nil
}

func (i *imported) Wake() {}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func newLife(t *testing.T) *life {
	t.Helper()
	ctx := t.Context()
	l := &life{
		lib:   storagetest.Library(t, identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), transfers.LibrarySchema(), Schema()),
		attic: role.NodeID{UUID: storage.NewUUID()}, den: role.NodeID{UUID: storage.NewUUID()},
		root: roots.ID{UUID: storage.NewUUID()}, work: works.WorkID{UUID: storage.NewUUID()},
		media: media.Record{
			Version: media.Version, Container: "matroska,webm", Duration: 2 * time.Hour,
			Video: media.Video{Codec: "hevc", Width: 3840, Height: 2160, BitDepth: 10, HDR: []release.HDR{"HDR10"}},
		},
		files:     &located{at: map[files.ID]files.Located{}},
		transfers: &wanted{last: map[string]transfers.Request{}, ids: map[string]transfers.ID{}},
		importer:  &imported{},
		now:       time.Date(2026, 11, 28, 22, 0, 0, 0, time.UTC),
	}
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, l.attic, "Attic"); err != nil {
			return err
		}
		if err := nodestest.Insert(ctx, tx, l.den, "Den"); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO roots_root (id, content_type, node_id, path, is_default) VALUES (?, 'movie', ?, '/movies', 1)`, l.root, l.attic)
		return err
	})
	goals, err := targets.Open(ctx, targets.Config{Library: l.lib, Changed: discard[targets.Changed]{}})
	if err != nil {
		t.Fatal(err)
	}
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO works_work (id, content_type, key, label, sort_label, added_at, refreshed_at) VALUES (?, 'movie', 'heat', 'Heat', 'Heat', 0, 0)`, l.work)
		return err
	})
	l.settings = encode.Defaults()
	l.settings.MaxHeight, l.settings.Kbps = encode.P720, 4000
	made, err := goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{
		Name: "720p mobile", Tiers: [][]release.Quality{{release.Bluray720p}}, Cutoff: release.Bluray720p,
		Scores: targets.Scores{Increment: 1}, Conversion: &targets.Conversion{Settings: l.settings},
	})
	if err != nil {
		t.Fatal(err)
	}
	l.target = made.ID
	if err := l.lib.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		all, err := targets.All(ctx, r)
		for _, t := range all {
			if t.ID == made.ID {
				l.version = t.Conversion.Version
			}
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	rec, kept := entrytest.New("conversions", Entries())
	l.history = kept
	l.c = New(Config{
		Library: l.lib, Files: l.files, Transfers: l.transfers, History: rec, Changed: discard[Changed]{}, Poke: NewPoke(),
		Clock: func() time.Time { return l.now }, Log: slog.New(slog.DiscardHandler),
		RootRoom: func(context.Context, storage.Reader[storage.Library], roots.ID) (int64, bool, error) {
			return 1 << 50, true, nil
		},
		TranscodingOn: func(context.Context, storage.Reader[storage.Library], role.NodeID) ([]mediaservers.Transcoding, error) {
			return nil, nil
		},
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return false, nil },
		Zone:        func() *time.Location { return time.UTC }, Timings: DefaultTimings(), Run: converting.DefaultTimings(),
	})
	l.c.Register(l.importer)
	l.c.roster = func() roster {
		return roster{holders: nobody{}, facts: func(role.NodeID) (*cv1.Facts, bool) { return nil, false }}
	}
	return l
}

func (l *life) write(t *testing.T, fn func(ctx context.Context, tx *storage.Tx[storage.Library]) error) {
	t.Helper()
	if err := l.lib.Write(t.Context(), fn); err != nil {
		t.Fatal(err)
	}
}

// seed is a Conversion of a new source file on the root, as a row
// written at stage would be.
type seed struct {
	Stage   Stage
	Node    *role.NodeID
	Attempt int
	Pin     *role.NodeID
	// Failed are the attempts before Attempt, each a run fault on its
	// Node.
	Failed     []role.NodeID
	BudgetFrom int
	NotBefore  time.Time
	Finished   time.Time
}

func (l *life) seed(t *testing.T, s seed) Row {
	t.Helper()
	src, rel := l.file(t)
	l.files.mu.Lock()
	l.files.at[src] = files.Located{ID: src, Root: l.root, Rel: rel, Node: l.attic, Name: "Heat.1995.2160p.UHD.BluRay.mkv", Size: 40 << 30, Media: l.media}
	l.files.mu.Unlock()
	id := ID{UUID: storage.NewUUID()}
	if s.BudgetFrom == 0 {
		s.BudgetFrom = 1
	}
	if s.NotBefore.IsZero() {
		s.NotBefore = l.now.Add(-time.Hour)
	}
	var code, class, detail, why, finished any
	if s.Stage == StageFailed {
		code, class, detail = "encoder", "run", "The 720p mobile Conversion of Heat failed on Attic."
	}
	if s.Stage == StageDone || s.Stage == StageCanceled {
		finished = storage.FromTime(s.Finished)
	}
	if s.Stage == StageCanceled {
		why = "admin"
	}
	var root any
	if s.Node != nil {
		root = l.root
	}
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO conversions_conversion (id, work_id, type, items, edition, source_file_id, target_id, version,
			lane, position, stage, node_id, root_id, pin_node_id, attempt, budget_from, not_before, failure_code, failure_class, failure_detail,
			cancel_why, snapshot, queued_at, finished_at)
			VALUES (?, ?, 'movie', '[]', 'any', ?, ?, ?, 'new', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			'{"work":"Heat","target":"720p mobile","sourceName":"Heat.1995.2160p.UHD.BluRay.mkv"}', 0, ?)`,
			id, l.work, src, l.target, l.version, "m"+id.String(), s.Stage, value(s.Node), root, value(s.Pin), s.Attempt, s.BudgetFrom,
			storage.FromTime(s.NotBefore), code, class, detail, why, finished); err != nil {
			return err
		}
		q := db.New(tx)
		for i, n := range s.Failed {
			if err := l.attempt(ctx, q, id, i+1, n, true); err != nil {
				return err
			}
		}
		if s.Attempt > len(s.Failed) {
			on := l.attic
			if s.Node != nil {
				on = *s.Node
			}
			return l.attempt(ctx, q, id, s.Attempt, on, s.Stage == StageFailed)
		}
		return nil
	})
	return l.read(t, id)
}

// file is a new file of the Work on the root.
func (l *life) file(t *testing.T) (files.ID, storing.Rel) {
	t.Helper()
	id := files.ID{UUID: storage.NewUUID()}
	rel, err := storing.ParseRel("Heat (1995)/" + id.String() + ".mkv")
	if err != nil {
		t.Fatal(err)
	}
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO files_file (id, work_id, cut, root_id, path, size, probe) VALUES (?, ?, 'standard', ?, ?, 1, '{}')`, id, l.work, l.root, rel.String())
		return err
	})
	return id, rel
}

func (l *life) attempt(ctx context.Context, q *db.Queries, id ID, n int, node role.NodeID, failed bool) error {
	if err := q.OpenAttempt(ctx, db.OpenAttemptParams{ConversionID: id, N: int64(n), NodeID: node, NodeName: "Node", Path: "hardware", Api: "qsv", StartedAt: storage.FromTime(l.now.Add(-time.Hour))}); err != nil || !failed {
		return err
	}
	outcome, code, class := "failed", "encoder", "run"
	return q.CloseAttempt(ctx, db.CloseAttemptParams{ConversionID: id, N: int64(n), EndedAt: storage.FromTime(l.now.Add(-time.Minute)), Outcome: &outcome, Code: &code, Class: &class})
}

func (l *life) read(t *testing.T, id ID) Row {
	t.Helper()
	var r Row
	err := l.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		raw, err := db.New(rd).GetConversion(ctx, id)
		if err != nil {
			return err
		}
		r, err = rowOf(raw)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (l *life) attempts(t *testing.T, id ID) []Attempt {
	t.Helper()
	var out []Attempt
	err := l.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		out, err = attemptsOf(ctx, db.New(rd), id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (l *life) advance(t *testing.T, id ID, ev Event) error {
	t.Helper()
	return l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return l.c.advance(ctx, tx, id, ev)
	})
}

// prove gives node a proof of the world's key.
func (l *life) prove(t *testing.T, node role.NodeID) encode.Key {
	t.Helper()
	k := encode.Key{Shape: l.settings.Shape(), Class: encode.ClassOf(l.media)}
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		speed := 4.0
		return db.New(tx).UpsertProof(ctx, db.UpsertProofParams{NodeID: node, Signature: "s", Shape: k.Shape.String(), Class: string(k.Class),
			Path: "hardware", Api: "qsv", Speed: &speed, ProvenAt: storage.FromTime(l.now.Add(-24 * time.Hour))})
	})
	return k
}

func (l *life) proofLost(t *testing.T, node role.NodeID, k encode.Key) bool {
	t.Helper()
	var lost bool
	err := l.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		p, err := db.New(rd).GetProof(ctx, db.GetProofParams{NodeID: node, Shape: k.Shape.String(), Class: string(k.Class)})
		lost = !p.LostAt.IsZero()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return lost
}

func (l *life) declined(t *testing.T, r Row) bool {
	t.Helper()
	var n int
	err := l.lib.QueryRowContext(t.Context(), `SELECT count(*) FROM conversions_declined WHERE source_file_id = ? AND target_id = ?`, *r.SourceFile, *r.Target).Scan(&n)
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		t.Fatal(err)
	}
	return n == 1
}

// nobody is a Board on which no Node holds Converting.
type nobody struct{}

func (nobody) Holds(nodes.ID) bool                 { return false }
func (nobody) May(nodes.ID) bool                   { return false }
func (nobody) Lapsed(nodes.ID) bool                { return false }
func (nobody) Away(nodes.ID) (nodes.Absence, bool) { return nodes.Absence{}, false }
func (nobody) WrittenOff(nodes.ID) bool            { return false }
func (nobody) Serving() []nodes.ID                 { return nil }
