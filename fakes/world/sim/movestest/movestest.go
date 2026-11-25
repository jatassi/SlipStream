// Package movestest is the levers of S48 (Moves)'s sim cases: the moves
// Scenario with one movie and its subtitle in den's movies-e, a crash of
// the Controller before a Move's step, the Nodes that go offline while it
// is down, and the invariants every case asserts.
package movestest

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Film is The Quiet Orchard's file, relative to its root.
const Film = "The Quiet Orchard (2021) {tmdb-9000001}/The Quiet Orchard (2021).mkv"

// Subtitle is the subtitle Launch writes beside Film.
var Subtitle = strings.TrimSuffix(Film, ".mkv") + ".en.srt"

// Steps is every IO step of a Work Move with no Media Server wait, in the
// order a Move runs them.
var Steps = []string{"take_lane", "place_file", "final_pass", "switch", "remove_source", "finish_cleanup"}

// World is the moves Scenario once movies-e's first scan recorded The
// Quiet Orchard.
type World struct {
	*sim.World
	Work works.WorkID
	// Seeded is each movie Seed adds, by id.
	Seeded []works.WorkID
	// Roots is each root by its folder, such as movies-d.
	Roots map[string]simtest.RootRow
	// Was is each source file's bytes by its path in movies-e.
	Was map[string][]byte
}

// Scenario is the moves Scenario with movies-f a volume of its own on
// den, which only the simulation tier plays.
func Scenario(t testing.TB) scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("moves")
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range s.Library.Roots {
		if r.Folder == "movies-f" {
			s.Library.Roots[i].Volume = &scenario.LibraryVolume{TotalGiB: 500, FreeGiB: 400}
		}
	}
	return s
}

// Launch starts the moves Scenario with The Quiet Orchard and its
// subtitle in den's movies-e, walking, with opts.
func Launch(t *testing.T, opts ...sim.Option) World {
	t.Helper()
	return launch(t, Scenario(t), hd, simtest.Settled, opts)
}

// LaunchUpgradable launches as Launch does with the film at 720p, so the
// Scenario's 1080p release of The Quiet Orchard is its upgrade.
func LaunchUpgradable(t *testing.T, opts ...sim.Option) World {
	t.Helper()
	return launch(t, Scenario(t), scenario.MediaVideo{Width: 1280, Height: 720, Seconds: 6, HDR: "sdr"}, simtest.Settled, opts)
}

// hd is the film's video unless a launch says otherwise.
var hd = scenario.MediaVideo{Width: 1920, Height: 1080, Seconds: 6, HDR: "sdr"}

// Seed launches as Launch does, with n generated movies more, each a film
// in movies-d or movies-e by turns, and waits until the scans record every
// one of them.
func Seed(t *testing.T, n int, opts ...sim.Option) World {
	t.Helper()
	s := Scenario(t)
	gen := scenario.TMDBGenerated{Count: n, Seed: "moves-seed"}
	s.TMDB.Generated = &gen
	s.Timings.TMDB = tmdb.Timings{Interval: time.Millisecond, Burst: 100}
	for i := range n {
		m := gen.Movie(i)
		name := fmt.Sprintf("%s (%s)", m.Title, m.Dates[0].Date[:4])
		add(&s, []string{"movies-d", "movies-e"}[i%2], scenario.MediaFile{
			Path:  fmt.Sprintf("%s {tmdb-%d}/%s.mkv", name, m.ID, name),
			Video: scenario.MediaVideo{Width: 1920, Height: 1080, Seconds: 6, HDR: "sdr"},
		})
	}
	settled := simtest.Settled + time.Duration(n)*time.Second
	w := launch(t, s, hd, settled, opts)
	await(t, settled, fmt.Sprintf("the scans record %d seeded movies", n), func() bool {
		w.Seeded = w.Seeded[:0]
		err := w.Library(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			rows, err := r.QueryContext(ctx, `SELECT DISTINCT work_id FROM files_file WHERE work_id <> ? ORDER BY work_id`, w.Work)
			if err != nil {
				return err
			}
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var id works.WorkID
				if err := rows.Scan(&id); err != nil {
					return fmt.Errorf("scan a seeded movie: %w", err)
				}
				w.Seeded = append(w.Seeded, id)
			}
			return rows.Err()
		})
		return err == nil && len(w.Seeded) == n
	})
	return w
}

// add puts f in s's library folder named folder.
func add(s *scenario.Scenario, folder string, f scenario.MediaFile) {
	i := slices.IndexFunc(s.Library.Folders, func(l scenario.LibraryFolder) bool { return l.Folder == folder })
	if i < 0 {
		s.Library.Folders, i = append(s.Library.Folders, scenario.LibraryFolder{Folder: folder}), len(s.Library.Folders)
	}
	s.Library.Folders[i].Files = append(s.Library.Folders[i].Files, f)
}

// await waits in fake time, up to limit, until ok, failing with what.
func await(t *testing.T, limit time.Duration, what string, ok func() bool) {
	t.Helper()
	for range limit / time.Second {
		if ok() {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("not within %s: %s", limit, what)
}

func launch(t *testing.T, s scenario.Scenario, film scenario.MediaVideo, settled time.Duration, opts []sim.Option) World {
	t.Helper()
	add(&s, "movies-e", scenario.MediaFile{Path: Film, Video: film})
	w := World{World: sim.Launch(t, s, append([]sim.Option{sim.WithWalk()}, opts...)...), Roots: map[string]simtest.RootRow{}}
	for _, r := range simtest.Decode[struct {
		Roots []simtest.RootRow `json:"roots"`
	}](t, "listRoots", w.API().Admin("home", http.MethodGet, "/api/v1/roots", "")).Roots {
		w.Roots[filepath.Base(r.Path)] = r
	}
	await(t, settled, "movies-e's first scan records The Quiet Orchard", func() bool {
		err := w.Library(t, func(ctx context.Context, r storage.Reader[storage.Library]) error {
			return r.QueryRowContext(ctx, `SELECT work_id FROM files_file WHERE path = ?`, Film).Scan(&w.Work)
		})
		return err == nil
	})
	source := w.Roots["movies-e"].Path
	if err := os.WriteFile(filepath.Join(source, filepath.FromSlash(Subtitle)), []byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	w.Was = map[string][]byte{}
	for _, rel := range []string{Film, Subtitle} {
		b, err := os.ReadFile(filepath.Join(source, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		w.Was[rel] = b
	}
	return w
}

// Library reads home's library.db.
func (w World) Library(t *testing.T, read func(ctx context.Context, r storage.Reader[storage.Library]) error) error {
	t.Helper()
	return w.Node("home").Half().Library.Read(t.Context(), read)
}

// Moves is home's Moves, read again after each restart.
func (w World) Moves() *moves.Moves { return w.Node("home").Half().Moves }

// Root is the id of the root in folder.
func (w World) Root(t *testing.T, folder string) roots.ID {
	t.Helper()
	id, err := storage.ParseUUID(w.Roots[folder].ID)
	if err != nil {
		t.Fatal(err)
	}
	return roots.ID{UUID: id}
}

// Start previews and starts The Quiet Orchard's Move to the root in
// folder, failing unless the preview moves it.
func (w World) Start(t *testing.T, folder string) moves.BatchID {
	t.Helper()
	m := w.Moves()
	plan, err := m.Preview(t.Context(), moves.Ask{Origin: moves.OriginWork, Works: []moves.WorkTo{{Work: w.Work, To: w.Root(t, folder)}}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Movable() != 1 || len(plan.Units[0].Files) != len(w.Was) {
		t.Fatalf("the preview is %+v, want The Quiet Orchard and its subtitle moving", plan.Units)
	}
	started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
	if err != nil {
		t.Fatal(err)
	}
	return started.Batch
}

// Unit is batch's one unit.
func (w World) Unit(t *testing.T, batch moves.BatchID) moves.View {
	t.Helper()
	_, units, err := w.Moves().Get(t.Context(), batch, -1, 10)
	if err != nil || len(units) != 1 {
		t.Fatalf("batch %s has %d units (%v), want 1", batch, len(units), err)
	}
	return units[0]
}

// Ended is batch's one unit once it is done or failed.
func (w World) Ended(t *testing.T, batch moves.BatchID) moves.View {
	t.Helper()
	var unit moves.View
	simtest.Eventually(t, "the unit finishes", func() bool {
		_, units, err := w.Moves().Get(t.Context(), batch, -1, 10)
		if err != nil || len(units) != 1 {
			return false
		}
		unit = units[0]
		return unit.Phase == moves.PhaseDone || unit.Phase == moves.PhaseFailed
	})
	return unit
}

// Invariants asserts I1, every file row of the Work names a file on its
// root's disk, and I2, the source keeps every file until batch's unit
// switches.
func (w World) Invariants(t *testing.T, batch moves.BatchID) {
	t.Helper()
	byID := map[string]string{}
	for _, r := range w.Roots {
		byID[r.ID] = r.Path
	}
	rows, err := w.Node("home").Half().Library.QueryContext(t.Context(), `SELECT root_id, path FROM files_file WHERE work_id = ?`, w.Work)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var root roots.ID
		var rel string
		if err := rows.Scan(&root, &rel); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(byID[root.String()], filepath.FromSlash(rel))); err != nil {
			t.Errorf("I1: the row of %s names root %s, where it isn't: %v", rel, root, err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if u := w.Unit(t, batch); u.Phase == moves.PhaseCleanup || u.Phase == moves.PhaseDone {
		return
	}
	for rel, want := range w.Was {
		if got, err := os.ReadFile(filepath.Join(w.Roots["movies-e"].Path, filepath.FromSlash(rel))); err != nil || !bytes.Equal(got, want) {
			t.Errorf("I2: the source lost %s before its switch (%d bytes, %v)", rel, len(got), err)
		}
	}
}

// Moved asserts every source file is in the root in folder with its bytes
// and gone from movies-e.
func (w World) Moved(t *testing.T, folder string) {
	t.Helper()
	for rel, want := range w.Was {
		got, err := os.ReadFile(filepath.Join(w.Roots[folder].Path, filepath.FromSlash(rel)))
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s holds %s as %d bytes (%v), want the source's %d", folder, rel, len(got), err, len(want))
		}
		if _, err := os.Stat(filepath.Join(w.Roots["movies-e"].Path, filepath.FromSlash(rel))); !os.IsNotExist(err) {
			t.Errorf("movies-e still holds %s: %v", rel, err)
		}
	}
}

// Crash stops the Controller once, just before a Move's step: the step
// waits until the Controller stops, so it never runs in that process.
type Crash struct {
	step    string
	reached chan struct{}
	held    atomic.Bool
}

// CrashAt is a Crash before step, one of Steps.
func CrashAt(step string) *Crash { return &Crash{step: step, reached: make(chan struct{})} }

// Option sets the Crash on home's BeforeMoveStep.
func (c *Crash) Option() sim.Option {
	return sim.Edit(func(name string, e *node.Env) {
		if name != "home" {
			return
		}
		e.Test.BeforeMoveStep = func(ctx context.Context, s moves.Step) {
			if s.Action == c.step && c.held.CompareAndSwap(false, true) {
				close(c.reached)
				<-ctx.Done()
			}
		}
	})
}

// Reached waits until the Move reached the Crash's step.
func (c *Crash) Reached(t *testing.T) {
	t.Helper()
	select {
	case <-c.reached:
	case <-time.After(time.Hour):
		t.Fatalf("the Move never reached %s within an hour", c.step)
	}
}

// Pause holds a Move once, just before its step, until Resume: the test
// changes the world while the step waits.
type Pause struct {
	step            string
	reached, resume chan struct{}
	held            atomic.Bool
}

// PauseAt is a Pause before step, one of Steps.
func PauseAt(step string) *Pause {
	return &Pause{step: step, reached: make(chan struct{}), resume: make(chan struct{})}
}

// Option sets the Pause on home's BeforeMoveStep.
func (p *Pause) Option() sim.Option {
	return sim.Edit(func(name string, e *node.Env) {
		if name != "home" {
			return
		}
		e.Test.BeforeMoveStep = func(ctx context.Context, s moves.Step) {
			if s.Action == p.step && p.held.CompareAndSwap(false, true) {
				close(p.reached)
				select {
				case <-p.resume:
				case <-ctx.Done():
				}
			}
		}
	})
}

// Reached waits until the Move reached the Pause's step.
func (p *Pause) Reached(t *testing.T) {
	t.Helper()
	select {
	case <-p.reached:
	case <-time.After(time.Hour):
		t.Fatalf("the Move never reached %s within an hour", p.step)
	}
}

// Resume lets the step run.
func (p *Pause) Resume() { close(p.resume) }

// StrayAt writes rel's source bytes at rel in the root in folder, as a
// Place leaves its file when a crash comes before its Write records it,
// and answers the stray's path.
func (w World) StrayAt(t *testing.T, folder, rel string) string {
	t.Helper()
	path := filepath.Join(w.Roots[folder].Path, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, w.Was[rel], 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// Fault is what goes wrong with the Controller down: the restart alone,
// or the restart with one Node offline until after it.
type Fault struct {
	Name string
	// Offline is the Node that stops before the restart and starts a
	// minute after it; empty for none.
	Offline string
}

// Faults is the fault matrix of a Move from movies-e to the root in
// folder: the restart, then the restart with each Node the Move touches
// offline.
func Faults(t *testing.T, folder string) []Fault {
	t.Helper()
	s := Scenario(t)
	out := []Fault{{Name: "restart"}, {Name: "restart, den offline", Offline: "den"}}
	for _, r := range s.Library.Roots {
		if r.Folder == folder && r.Node != "den" {
			out = append(out, Fault{Name: "restart, " + r.Node + " offline", Offline: r.Node})
		}
	}
	return out
}

// Restart stops home and starts it again on its link port with f.
func (w World) Restart(t *testing.T, f Fault) {
	t.Helper()
	h := w.Node("home")
	link := h.Ready().Listeners.Link
	if f.Offline != "" {
		w.Node(f.Offline).Stop()
	}
	h.Stop()
	h.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_LINK_LISTEN="+link.String()) })
	if f.Offline != "" {
		time.Sleep(time.Minute)
		w.Node(f.Offline).Start()
	}
}
