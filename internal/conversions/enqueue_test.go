package conversions

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/files"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// sourced is Files with the made-from cells the test gives each Work.
type sourced struct {
	*located
	mu    sync.Mutex
	cells map[works.WorkID][]files.Sourced
}

func (s *sourced) Sourced(_ context.Context, _ storage.Reader[storage.Library], ws works.Among[works.WorkID]) (map[works.WorkID][]files.Sourced, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[works.WorkID][]files.Sourced{}
	for _, w := range ws.IDs() {
		out[w] = s.cells[w]
	}
	return out, nil
}

func (l *life) seam() Seam {
	return Seam{Poke: l.c.cfg.Poke, Conversions: func() *Conversions { return l.c }}
}

// cells gives the Work cells from now on.
func (l *life) cells(cells ...files.Sourced) {
	s, ok := l.c.cfg.Files.(*sourced)
	if !ok {
		s = &sourced{located: l.files, cells: map[works.WorkID][]files.Sourced{}}
		l.c.cfg.Files, l.c.cfg.Timings = s, DefaultTimings()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cells[l.work] = cells
}

func (l *life) queue(t *testing.T, a files.Ask) []ID {
	t.Helper()
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return l.seam().Queue(ctx, tx, a) })
	changed, _, err := l.c.drain(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return changed
}

// open is the Work's unfinished Conversions.
func (l *life) open(t *testing.T) []Row {
	t.Helper()
	var out []Row
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		raws, err := db.New(r).ListOpenOfWorks(ctx, []works.WorkID{l.work})
		for _, raw := range raws {
			row, err := rowOf(raw)
			if err != nil {
				return err
			}
			out = append(out, row)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func (l *life) decline(t *testing.T, src files.ID) {
	t.Helper()
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).Decline(ctx, db.DeclineParams{SourceFileID: src, TargetID: l.target})
	})
}

type before int

const (
	nothing before = iota
	queuedRow
	failedRow
	adminCanceled
	declinedPair
)

type filling int

const (
	none filling = iota
	downloaded
	madeNow
	madeOlder
	madeUnranked
)

const grace = 72 * time.Hour

type drainCase struct {
	name     string
	sourcing targets.Sourcing
	why      files.Arrival
	backfill bool
	released bool
	noSource bool
	file     filling
	before   before
	// lane is the queued row's, "" when nothing is queued.
	lane     Lane
	waits    bool
	canceled CancelWhy
}

func TestTheDrainQueuesWhatCellsWantOnce(t *testing.T) {
	cases := []drainCase{
		{name: "a new source queues in new", sourcing: targets.Derive, why: files.ArrivalNew, released: true, lane: LaneNew},
		{name: "a download-or-made cell waits its grace", sourcing: targets.DownloadThenDerive, why: files.ArrivalNew, released: true, lane: LaneNew, waits: true},
		{name: "an adoption queues in backfill", sourcing: targets.Derive, why: files.ArrivalAdopted, released: true, lane: LaneBackfill},
		{name: "a Profile change queues in backfill", sourcing: targets.Derive, why: files.ArrivalProfile, released: true, lane: LaneBackfill},
		{name: "an unreleased cell queues nothing", sourcing: targets.Derive, why: files.ArrivalNew},
		{name: "no source file queues nothing", sourcing: targets.Derive, why: files.ArrivalNew, released: true, noSource: true},
		{name: "a downloaded file queues nothing", sourcing: targets.DownloadThenDerive, why: files.ArrivalNew, released: true, file: downloaded},
		{name: "a file made from it with these settings queues nothing", sourcing: targets.Derive, why: files.ArrivalNew, released: true, file: madeNow},
		{name: "older settings queue nothing on their own", sourcing: targets.Derive, why: files.ArrivalChanged, released: true, file: madeOlder},
		{name: "a remake of older settings queues in backfill", sourcing: targets.Derive, why: files.ArrivalRemake, released: true, file: madeOlder, lane: LaneBackfill},
		{name: "a made file it upgrades queues in upgrade", sourcing: targets.Derive, why: files.ArrivalNew, released: true, file: madeUnranked, lane: LaneUpgrade},
		{name: "a declined pair queues nothing", sourcing: targets.Derive, why: files.ArrivalNew, released: true, before: declinedPair},
		{name: "a backfill asks for a declined pair again", sourcing: targets.Derive, why: files.ArrivalProfile, backfill: true, released: true, before: declinedPair, lane: LaneBackfill},
		{name: "an Admin's cancel isn't undone", sourcing: targets.Derive, why: files.ArrivalChanged, released: true, before: adminCanceled},
		{name: "a failed row holds its pair", sourcing: targets.Derive, why: files.ArrivalChanged, released: true, before: failedRow},
		{name: "a queued row holds its pair", sourcing: targets.Derive, why: files.ArrivalChanged, released: true, before: queuedRow, lane: LaneNew},
		{name: "a download cancels the queued row", sourcing: targets.DownloadThenDerive, why: files.ArrivalChanged, released: true, file: downloaded, before: queuedRow, canceled: NotWanted},
		{name: "a lost source cancels the failed row", sourcing: targets.Derive, why: files.ArrivalChanged, released: true, noSource: true, before: failedRow, canceled: NotWanted},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := newLife(t)
			since := l.now.Add(-time.Minute)
			src, prior := c.arrange(t, l)
			item := works.ItemID{UUID: storage.NewUUID()}
			l.cells(c.cell(l, src, item))

			l.queue(t, files.Ask{Work: l.work, Why: c.why, Since: since, Backfill: c.backfill})

			c.check(t, l, src, prior, item, since)
			if again := l.queue(t, files.Ask{Work: l.work, Why: c.why, Since: since}); len(again) != 0 {
				t.Errorf("the second drain changed %v", again)
			}
		})
	}
}

// arrange writes what is there before the drain, answering the source
// file and the Conversion seeded for it.
func (c drainCase) arrange(t *testing.T, l *life) (files.ID, *Row) {
	t.Helper()
	switch c.before {
	case queuedRow, failedRow, adminCanceled:
		s := seed{Stage: StageQueued}
		if c.before == failedRow {
			s = seed{Stage: StageFailed, Attempt: 1}
		}
		r := l.seed(t, s)
		if c.before == adminCanceled {
			if _, err := l.c.Cancel(t.Context(), r.ID); err != nil {
				t.Fatal(err)
			}
		}
		return *r.SourceFile, &r
	case declinedPair:
		src, _ := l.file(t)
		l.decline(t, src)
		return src, nil
	case nothing:
	}
	src, _ := l.file(t)
	return src, nil
}

func (c drainCase) cell(l *life, src files.ID, item works.ItemID) files.Sourced {
	cell := files.Sourced{
		Work: l.work, Items: []works.ItemID{item}, Edition: targets.AnyEdition, Target: l.target, Sourcing: c.sourcing,
		From: targets.TargetID{UUID: storage.NewUUID()}, Counted: true, Released: c.released,
	}
	if c.sourcing == targets.DownloadThenDerive {
		cell.Grace = grace
	}
	if !c.noSource {
		cell.SourceFile = &files.File{ID: src, WorkID: l.work, Path: "Heat (1995)/Heat.mkv", Media: l.media,
			Title: release.Read{Facts: release.Facts{Quality: release.Known(release.Bluray2160p), Source: release.Known(release.OriginBluray)}}}
	}
	other := files.ID{UUID: storage.NewUUID()}
	switch c.file {
	case downloaded:
		cell.File = &files.File{ID: other, Quality: release.Bluray720p}
	case madeNow:
		cell.File = &files.File{ID: other, Quality: release.Bluray720p, Derived: &files.Derived{From: src, Target: l.target, Version: l.version}}
	case madeOlder:
		cell.File = &files.File{ID: other, Quality: release.Bluray720p, Derived: &files.Derived{From: src, Target: l.target, Version: l.version - 1}}
	case madeUnranked:
		cell.File = &files.File{ID: other, Quality: release.SDTV, Derived: &files.Derived{From: files.ID{UUID: storage.NewUUID()}, Target: l.target, Version: l.version}}
	case none:
	}
	return cell
}

func (c drainCase) check(t *testing.T, l *life, src files.ID, prior *Row, item works.ItemID, since time.Time) {
	t.Helper()
	open := l.open(t)
	switch {
	case c.lane != "":
		c.checkQueued(t, open, src, prior, item, since)
	case c.before == failedRow && c.canceled == "":
		if len(open) != 1 || open[0].Stage != StageFailed {
			t.Errorf("open = %+v, want the failed row alone", open)
		}
	case len(open) != 0:
		t.Errorf("queued %+v, want nothing", open)
	}
	if c.canceled != "" {
		if r := l.read(t, prior.ID); r.Stage != StageCanceled || r.CancelWhy != c.canceled {
			t.Errorf("prior row %s %s, want canceled %s", r.Stage, r.CancelWhy, c.canceled)
		}
	}
	if c.before == adminCanceled && !l.declined(t, *prior) {
		t.Error("the Admin's cancel didn't decline the pair")
	}
	if c.backfill && l.declined(t, Row{SourceFile: &src, Target: &l.target}) {
		t.Error("the backfill left the pair declined")
	}
}

func (c drainCase) checkQueued(t *testing.T, open []Row, src files.ID, prior *Row, item works.ItemID, since time.Time) {
	t.Helper()
	if len(open) != 1 || open[0].Lane != c.lane || open[0].Stage != StageQueued || *open[0].SourceFile != src {
		t.Fatalf("open = %+v, want one queued row of %s in %s", open, src, c.lane)
	}
	if prior != nil {
		if open[0].ID != prior.ID {
			t.Errorf("queued %s beside %s", open[0].ID, prior.ID)
		}
		return
	}
	want := since
	if c.waits {
		want = since.Add(grace)
	}
	if !open[0].NotBefore.Equal(want) || !slices.Equal(open[0].Items, []works.ItemID{item}) {
		t.Errorf("queued not before %v for %v, want %v for %v", open[0].NotBefore, open[0].Items, want, item)
	}
}

func TestLeavingCancelsBeforeTheSourceGoes(t *testing.T) {
	l := newLife(t)
	queued := l.seed(t, seed{Stage: StageQueued})
	running := l.seed(t, seed{Stage: StageRunning, Node: &l.attic, Attempt: 1})
	for _, r := range []Row{queued, running} {
		l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			if err := l.seam().Leaving(ctx, tx, files.File{ID: *r.SourceFile, WorkID: l.work}, files.ID{}); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `DELETE FROM files_file WHERE id = ?`, *r.SourceFile)
			return err
		})
		if got := l.read(t, r.ID); got.Stage != StageCanceled || got.CancelWhy != SourceGone {
			t.Errorf("%s row is %s %s, want canceled %s", r.Stage, got.Stage, got.CancelWhy, SourceGone)
		}
	}
	if l.transfers.holding(sourceJob(l.attic, *running.SourceFile)) {
		t.Error("the running row's source copy wasn't ended")
	}
}

func TestHoldingClaimsBothNamesOfAWritingConversion(t *testing.T) {
	l := newLife(t)
	cases := []struct {
		stage  Stage
		claims bool
	}{
		{StageQueued, false},
		{StageRunning, true},
		{StageChecking, true},
		{StageLanding, true},
		{StageFailed, false},
		{StageDone, false},
	}
	rows := make([]Row, len(cases))
	for i, c := range cases {
		s := seed{Stage: c.stage, Attempt: 1, Finished: l.now}
		if c.stage == StageRunning {
			s.Node = &l.attic
		}
		if c.stage == StageQueued {
			s.Attempt = 0
		}
		rows[i] = l.seed(t, s)
		if c.stage == StageChecking || c.stage == StageLanding || c.stage == StageDone {
			l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, err := tx.ExecContext(ctx, `UPDATE conversions_conversion SET root_id = ? WHERE id = ?`, l.root, rows[i].ID)
				return err
			})
		}
	}
	var held map[storage.UUID]map[string]bool
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		got, err := l.seam().Holding(ctx, r)
		held = map[storage.UUID]map[string]bool{}
		for root, names := range got {
			held[root.UUID] = names
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		name := storing.HoldingDir + "/c-" + rows[i].ID.String() + "-1." + string(l.settings.Container)
		for _, n := range []string{name, name + ".part"} {
			if held[l.root.UUID][n] != c.claims {
				t.Errorf("%s: claims %s = %v, want %v", c.stage, n, held[l.root.UUID][n], c.claims)
			}
		}
	}
}

// spot is a row's place in the queue.
type spot struct {
	Lane     Lane
	Position string
}

// lined is four queued rows: the first in Moved up, the rest in New.
func (l *life) lined(t *testing.T) ([]Row, []spot) {
	t.Helper()
	rows, at := make([]Row, 4), make([]spot, 4)
	for i := range rows {
		rows[i] = l.seed(t, seed{Stage: StageQueued})
		at[i] = spot{LaneNew, string(rune('b'+i)) + "0"}
		if i == 0 {
			at[i].Lane = LaneMovedUp
		}
		l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE conversions_conversion SET lane = ?, position = ? WHERE id = ?`, at[i].Lane, at[i].Position, rows[i].ID)
			return err
		})
	}
	return rows, at
}

// runOrder is the indexes of at in the order the queue runs them.
func runOrder(at []spot) []int {
	lanes := []Lane{LaneMovedUp, LaneNew, LaneUpgrade, LaneBackfill}
	out := make([]int, len(at))
	for i := range out {
		out[i] = i
	}
	slices.SortFunc(out, func(a, b int) int {
		if d := slices.Index(lanes, at[a].Lane) - slices.Index(lanes, at[b].Lane); d != 0 {
			return d
		}
		return strings.Compare(at[a].Position, at[b].Position)
	})
	return out
}

func TestMovesRewriteOnlyTheMovedRow(t *testing.T) {
	cases := []struct {
		name string
		// after is the row rows[1] moves after, -1 for Move up.
		after int
		want  []int
	}{
		{name: "move up", after: -1, want: []int{1, 0, 2, 3}},
		{name: "move after the last", after: 3, want: []int{0, 2, 3, 1}},
		{name: "move after the first of the lane", after: 2, want: []int{0, 2, 1, 3}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l := newLife(t)
			rows, before := l.lined(t)
			var err error
			if c.after < 0 {
				_, err = l.c.MoveUp(t.Context(), rows[1].ID)
			} else {
				_, err = l.c.MoveAfter(t.Context(), rows[1].ID, &rows[c.after].ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			got := make([]spot, len(rows))
			for i, r := range rows {
				read := l.read(t, r.ID)
				got[i] = spot{read.Lane, string(read.Position)}
				if i != 1 && got[i] != before[i] {
					t.Errorf("row %d moved from %v to %v", i, before[i], got[i])
				}
			}
			if order := runOrder(got); !slices.Equal(order, c.want) {
				t.Errorf("order = %v, want %v", order, c.want)
			}
		})
	}
	l := newLife(t)
	running := l.seed(t, seed{Stage: StageRunning, Node: &l.attic, Attempt: 1})
	if _, err := l.c.MoveUp(t.Context(), running.ID); !errors.Is(err, ErrNotWaiting) {
		t.Errorf("moving a running row = %v, want %v", err, ErrNotWaiting)
	}
}

func TestNodeSettingsReadBackWhatIsSet(t *testing.T) {
	l := newLife(t)
	ctl := converting.Def.Serve(l.c.Derive, l.c.Apply)
	role.NewControls(role.ControlsConfig{Library: l.lib}, ctl)
	l.c.cfg.Control = func() *role.Control[*cv1.Intent, *cv1.Facts, *cv1.Result] { return ctl }
	three := 3
	cases := []struct {
		name string
		set  *NodeSettings
		want NodeSettings
	}{
		{name: "no row is the defaults", want: NodeSettings{Hours: setting.Span{Start: 60, End: 420}, Processor: true}},
		{name: "a set count", set: &NodeSettings{AtOnce: &three, HoursOn: true, Hours: setting.Span{Start: 120, End: 360}}},
		{name: "no count is the default", set: &NodeSettings{Processor: true, Hours: setting.Span{Start: 0, End: 1439}}},
	}
	for _, c := range cases {
		want := c.want
		if c.set != nil {
			want = *c.set
			if _, err := l.c.SetNodeSettings(t.Context(), l.den, *c.set); err != nil {
				t.Fatal(err)
			}
		}
		got, err := l.c.NodeSettings(t.Context(), l.den)
		if err != nil {
			t.Fatal(err)
		}
		if (got.Settings.AtOnce == nil) != (want.AtOnce == nil) || want.AtOnce != nil && *got.Settings.AtOnce != *want.AtOnce ||
			got.Settings.HoursOn != want.HoursOn || got.Settings.Hours != want.Hours || got.Settings.Processor != want.Processor {
			t.Errorf("%s: read %+v, want %+v", c.name, got.Settings, want)
		}
	}
}

func TestCellsCarryTheConversionsFacts(t *testing.T) {
	l := newLife(t)
	cases := []struct {
		seed       seed
		due        bool
		code       string
		class      Class
		converting bool
	}{
		{seed: seed{Stage: StageQueued}, due: true, converting: true},
		{seed: seed{Stage: StageQueued, NotBefore: l.now.Add(time.Hour)}},
		{seed: seed{Stage: StageRunning, Node: &l.attic, Attempt: 1}, due: true, converting: true},
		{seed: seed{Stage: StageFailed, Attempt: 1}, due: true, code: string(CodeEncoder), class: RunClass},
	}
	item := works.ItemID{UUID: storage.NewUUID()}
	rows := make([]Row, len(cases))
	for i, c := range cases {
		rows[i] = l.seed(t, c.seed)
		l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE conversions_conversion SET items = ?, edition = ? WHERE id = ?`, jsonOf([]works.ItemID{item}), "e"+rows[i].ID.String(), rows[i].ID)
			return err
		})
	}
	var got map[files.CellKey]files.ConversionCell
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		got, err = l.seam().Cells(ctx, r, works.Only([]works.WorkID{l.work}))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		v := got[files.CellKey{Item: item, Edition: targets.Edition("e" + rows[i].ID.String()), Target: l.target}]
		if v.ID != rows[i].ID.UUID || v.Due != c.due || v.Code != c.code || v.Class != string(c.class) || v.Converting() != c.converting {
			t.Errorf("%s: cell %+v, want due %v, code %q, class %q, converting %v", c.seed.Stage, v, c.due, c.code, c.class, c.converting)
		}
	}
}

func TestRemakesListOlderFilesAndQueueTheChosen(t *testing.T) {
	l := newLife(t)
	l.cells()
	made := func(version int, pruned bool) {
		r := l.seed(t, seed{Stage: StageDone, Finished: l.now})
		out, _ := l.file(t)
		l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			if _, err := tx.ExecContext(ctx, `UPDATE files_file SET derived = 1, derived_from_file_id = ?, derived_target_id = ?, conversion_version = ? WHERE id = ?`,
				r.SourceFile, l.target, version, out); err != nil {
				return err
			}
			query, args := `UPDATE conversions_conversion SET version = ?, output_file_id = ? WHERE id = ?`, []any{version, out, r.ID}
			if pruned {
				query, args = `DELETE FROM conversions_conversion WHERE id = ?`, []any{r.ID}
			}
			_, err := tx.ExecContext(ctx, query, args...)
			return err
		})
	}
	made(l.version-1, false)
	made(l.version-2, true)
	made(l.version, false)
	var preview, saved files.RemakeImpact
	l.write(t, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		if preview, err = l.seam().Remakes(ctx, tx, l.target, l.version, nil, true); err != nil {
			return err
		}
		saved, err = l.seam().Remakes(ctx, tx, l.target, l.version, []works.WorkID{l.work}, false)
		return err
	})
	for _, got := range []files.RemakeImpact{preview, saved} {
		if got.Older != 2 || len(got.Rows) != 1 || got.Rows[0].Work != l.work || got.Rows[0].Files != 2 || got.Rows[0].Bytes <= 0 {
			t.Errorf("impact = %+v, want two older files of the Work, the one whose done row Retention pruned included", got)
		}
	}
	var nudges []db.ConversionsNudge
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		nudges, err = db.New(r).TakeNudges(ctx, 10)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(nudges) != 1 || nudges[0].WorkID != l.work || nudges[0].Why != string(files.ArrivalRemake) || nudges[0].BackfillID.IsZero() {
		t.Errorf("nudges = %+v, want one remake of the Work into a backfill", nudges)
	}
}
