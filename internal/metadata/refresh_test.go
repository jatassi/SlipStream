package metadata_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// alone is home holding Metadata by itself, as the tasks runner and the
// Controller's metadata see it.
type alone struct{}

func (alone) Self() role.NodeID { return home }

func (alone) Pick(role.Name, role.NodeID) (role.NodeID, bool) { return home, true }

func (alone) PickOther(role.Name, role.NodeID) (role.NodeID, bool) { return role.NodeID{}, false }

func (alone) Why(role.Name) string { return "" }

func (alone) Standing(role.NodeID) (nodes.Standing, time.Time) { return nodes.InStep, time.Time{} }

func (alone) Route(_ role.Name, home role.NodeID) (role.NodeID, error) { return home, nil }

func (alone) Get(_ context.Context, id role.NodeID) (nodes.Node, error) {
	return nodes.Node{ID: id, Name: "home", Self: true, Link: nodes.Local}, nil
}

func (alone) Names(context.Context) (map[role.NodeID]nodes.Named, error) {
	return map[role.NodeID]nodes.Named{home: {ID: home, Name: "home"}}, nil
}

func (alone) Changed() <-chan struct{} { return nil }

type quiet[T any] struct{}

func (quiet[T]) Publish(context.Context, T) {}

// refreshed is a Controller whose only Content Type is the Fixture, with
// Metadata refresh in its tasks catalog, inside a synctest bubble that
// starts at midnight on 1 January 2000, UTC.
type refreshed struct {
	fx    *contenttypetest.Fixture
	lib   *storage.DB[storage.Library]
	meta  *metadata.Metadata
	works *works.Works
	tasks *tasks.Scheduler
	ids   map[works.Key]works.WorkID
}

func openRefreshed(t *testing.T, pages int) *refreshed {
	t.Helper()
	l := &refreshed{fx: contenttypetest.New(), ids: map[works.Key]works.WorkID{}}
	l.lib = storagetest.Library(t, identity.Schema(), works.Schema(), metadata.Schema(), targets.Schema(), tasks.Schema())
	hub := live.NewHub([]live.Event{nodes.ChangedEvent, tasks.ChangedEvent, tasks.ProgressEvent, works.ChangedEvent, works.TypeChangedEvent})
	t.Cleanup(hub.Close)
	timings := metadata.DefaultTimings()
	timings.ChangesPages = pages
	d, cluster := rpc.Alone(home, func(role.Name) bool { return true })
	metadatav1connect.MetadataService.Mount(d, metadata.NewHolder(metadata.HolderConfig{
		Types: map[works.TypeID]metadata.Source{contenttypetest.ID: l.fx.Def().OpenProvider(contenttype.ProviderDeps{}, nil)}, Timings: timings,
	}))
	var err error
	l.meta, err = metadata.Open(t.Context(), metadata.Config{
		Library: l.lib, Sealer: secrettest.Sealer(t), Roster: alone{},
		Service: func(node role.NodeID) metadatav1connect.MetadataServiceHandler {
			return metadatav1connect.MetadataService.On(cluster, node)
		},
		Changed: &changes{}, Timings: timings, Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw the test replays
		Types: []works.TypeID{contenttypetest.ID}, TypeChanged: works.TypeChangedEvent.Bind(hub),
		Local: storagetest.Local(t, artwork.Schema()), ArtworkDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	l.works, err = works.Open(t.Context(), works.Config{
		Library: l.lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: l.fx}, Source: l.meta, Timings: works.DefaultTimings(),
		Changed: works.ChangedEvent.Bind(hub), TypesChanged: quiet[works.ContentTypesChanged]{}, TypeChanged: works.TypeChangedEvent.Bind(hub),
		Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	goals, err := targets.Open(t.Context(), targets.Config{
		Library: l.lib, Works: l.works, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: l.fx}, Changed: quiet[targets.Changed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	l.tasks, err = tasks.Open(t.Context(), tasks.Config{
		Library: l.lib,
		Catalog: []tasks.Task{metadata.RefreshTask(metadata.RefreshConfig{
			Library: l.lib, Metadata: l.meta, Works: l.works, Targets: goals,
			Types: map[works.TypeID]metadata.Refreshing{contenttypetest.ID: l.fx},
			Today: func() time.Time { return time.Now().UTC() }, Log: slog.New(slog.DiscardHandler),
		})},
		Roster: alone{}, Zone: time.UTC, Rand: rand.New(rand.NewPCG(3, 4)), //nolint:gosec // a fixed draw the test replays
		Hub: hub, Changed: tasks.ChangedEvent.Bind(hub), Progressed: tasks.ProgressEvent.Bind(hub), Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = l.tasks.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return l
}

// add adds a Work whose one Item is released, unless unreleased.
func (l *refreshed) add(t *testing.T, key works.Key, unreleased bool) works.WorkID {
	t.Helper()
	l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: strings.ToUpper(string(key)), Children: []works.Entry{{Key: "box", Label: "Box", Children: []works.Entry{{Key: "item", Label: "Item"}}}}, RatingsRead: true}})
	tree, _, err := l.works.Add(t.Context(), identity.Admin{}, contenttypetest.ID, key, works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	if unreleased {
		l.fx.SetTiming(tree.Items[0].ID, targets.Timing{})
	}
	l.ids[key] = tree.ID
	return tree.ID
}

// nightly runs the window's scheduled run of each of n nights, from the
// morning after, and answers which Works each fetched, by night.
func (l *refreshed) nightly(t *testing.T, n int) [][]works.Key {
	t.Helper()
	var out [][]works.Key
	for range n {
		before := l.fetches()
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 7, 0, 0, 0, time.UTC)
		if now.Hour() < 4 {
			next = next.AddDate(0, 0, -1)
		}
		time.Sleep(time.Until(next))
		synctest.Wait()
		out = append(out, l.fetchedSince(before))
	}
	return out
}

func (l *refreshed) fetches() map[works.Key]int {
	out := map[works.Key]int{}
	for k := range l.ids {
		out[k] = l.fx.Fetches(k)
	}
	return out
}

func (l *refreshed) fetchedSince(before map[works.Key]int) []works.Key {
	var out []works.Key
	for k, n := range l.fetches() {
		if n > before[k] {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// runNow runs Metadata refresh now and answers the run once it closes.
func (l *refreshed) runNow(t *testing.T) tasks.Run {
	t.Helper()
	if _, err := l.tasks.RunNow(t.Context(), metadata.RefreshTaskID, storage.UUID{}); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return l.latest(t)
}

func (l *refreshed) latest(t *testing.T) tasks.Run {
	t.Helper()
	d, err := l.tasks.Task(t.Context(), metadata.RefreshTaskID)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Runs) == 0 {
		t.Fatal("Metadata refresh has run no run")
	}
	return d.Runs[0]
}

func (l *refreshed) tune(t *testing.T, values map[string]any) {
	t.Helper()
	p := setting.Patch{}
	for k, v := range values {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		p[k] = raw
	}
	if err := l.tasks.SetTuning(t.Context(), metadata.RefreshTaskID, p); err != nil {
		t.Fatalf("tune %v: %v", values, err)
	}
}

// labels is each unit of run as "label outcome", with its cause when it
// has one.
func labels(run tasks.Run) []string {
	var out []string
	for _, line := range run.Lines {
		s := line.Label + " " + string(line.Outcome)
		if line.Cause != "" {
			s += ": " + line.Cause
		}
		out = append(out, s)
	}
	return out
}

// changesAsked is the since of each changes list read, in order.
func (l *refreshed) changesAsked() []time.Time {
	var out []time.Time
	for _, c := range l.fx.Calls() {
		if c.Page == 1 {
			out = append(out, c.Since)
		}
	}
	return out
}

func keys(ks ...works.Key) []works.Key { return ks }

func TestEachTierMakesItsWorksDue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "out", false)
		l.add(t, "coming", true)
		growing := l.add(t, "growing", false)
		l.fx.SetGrowing(growing, true)

		nights := l.nightly(t, 3)
		want := [][]works.Key{nil, keys("coming", "growing"), keys("coming", "growing")}
		if fmt.Sprint(nights) != fmt.Sprint(want) {
			t.Fatalf("the first three nights fetched %v, want %v: unreleased and growing Works daily", nights, want)
		}
		if run := l.latest(t); run.Trigger != tasks.BySchedule || run.Result != tasks.RunSucceeded || run.Summary != "2 Works" {
			t.Fatalf("the third night's run %s %s %q, want a scheduled run that succeeded with 2 Works", run.Trigger, run.Result, run.Summary)
		}

		l.tune(t, map[string]any{"metadata.refresh.unreleased": 2 * 86400, "metadata.refresh.growing": 3 * 86400})
		nights = l.nightly(t, 6)
		want = [][]works.Key{nil, keys("coming"), keys("growing"), keys("coming"), nil, keys("coming", "growing")}
		if fmt.Sprint(nights) != fmt.Sprint(want) {
			t.Fatalf("with unreleased every 2 days and growing every 3, six nights fetched %v, want %v", nights, want)
		}

		var outOn []int
		for i, night := range l.nightly(t, 25) {
			if slices.Contains(night, "out") {
				outOn = append(outOn, 10+i)
			}
		}
		if !slices.Equal(outOn, []int{31}) {
			t.Fatalf("a released Work added on night 1 was fetched on nights %v, want night 31 alone: every other Work every 30 days", outOn)
		}
	})
}

func TestTheTaskIsListedOnceAWorkExistsAndRunsWithNothingDue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		listed := func() bool {
			o, err := l.tasks.Overview(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			return slices.ContainsFunc(o.Tasks, func(v tasks.View) bool { return v.Task.ID == metadata.RefreshTaskID })
		}
		if listed() {
			t.Fatal("Metadata refresh is listed in an empty library")
		}
		l.add(t, "out", false)
		if !listed() {
			t.Fatal("Metadata refresh isn't listed once the library holds a Work")
		}
		if run := l.runNow(t); run.Result != tasks.RunSucceeded || run.Summary != "0 Works" {
			t.Fatalf("a run with nothing due %s %q, want succeeded with 0 Works", run.Result, run.Summary)
		}
	})
}

func TestTheChangesListCoversSinceTheLastGoodRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "out", false)
		l.add(t, "still", false)
		l.fx.Change("out", "elsewhere")

		if nights := l.nightly(t, 1); len(nights[0]) != 0 || len(l.changesAsked()) != 0 {
			t.Fatalf("the first night fetched %v and read changes lists from %v, want neither: it starts the list there", nights[0], l.changesAsked())
		}
		first := l.latest(t).Started
		if nights := l.nightly(t, 1); !slices.Equal(nights[0], keys("out")) {
			t.Fatalf("the second night fetched %v, want the Work the changes list named", nights[0])
		}
		second := l.latest(t).Started
		l.fx.FailChanges(&works.SourceError{Source: "Fixture", Words: "didn't answer", Err: works.ErrUnavailable})
		if nights := l.nightly(t, 1); len(nights[0]) != 0 {
			t.Fatalf("a night whose changes list failed fetched %v, want none", nights[0])
		}
		l.fx.FailChanges(nil)
		if nights := l.nightly(t, 1); !slices.Equal(nights[0], keys("out")) {
			t.Fatalf("the night after a failed list fetched %v, want the Work the list named", nights[0])
		}
		if asked, want := l.changesAsked(), []time.Time{first, second, second}; !slices.EqualFunc(asked, want, time.Time.Equal) {
			t.Fatalf("changes lists were read from %v, want from %v: each from the last good run's start", asked, want)
		}
	})
}

func TestAChangesListFromPastTheSourcesWindowStartsAgainFromNow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "out", false)
		l.nightly(t, 1)
		first := l.latest(t).Started
		l.tune(t, map[string]any{"metadata.refresh.changes": false})
		l.nightly(t, 15)
		if asked := l.changesAsked(); len(asked) != 0 {
			t.Fatalf("with the changes switch off, changes lists were read from %v, want none", asked)
		}
		l.tune(t, map[string]any{"metadata.refresh.changes": true})
		l.fx.Change("out")
		nights := l.nightly(t, 1)
		late := l.latest(t).Started
		nights = append(nights, l.nightly(t, 1)...)
		if want := [][]works.Key{nil, keys("out")}; fmt.Sprint(nights) != fmt.Sprint(want) {
			t.Fatalf("the two nights after the switch came back on fetched %v, want %v", nights, want)
		}
		if asked, want := l.changesAsked(), []time.Time{first, late}; !slices.EqualFunc(asked, want, time.Time.Equal) {
			t.Fatalf("changes lists were read from %v, want from %v: a list from past the source's window starts again from that run", asked, want)
		}
	})
}

func TestAChangesListPastThePageBoundCountsAsNone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 2)
		for _, k := range []works.Key{"a", "b", "c", "d", "e"} {
			l.add(t, k, false)
		}
		l.nightly(t, 1)
		l.fx.Change("a", "b", "c", "d")
		if nights := l.nightly(t, 1); !slices.Equal(nights[0], keys("a", "b", "c", "d")) {
			t.Fatalf("a list of two pages, the bound, fetched %v, want every Work it named", nights[0])
		}
		l.fx.Change("a", "b", "c", "d", "e")
		third := l.nightly(t, 1)[0]
		started := l.latest(t).Started
		l.fx.Change()
		l.nightly(t, 1)
		if asked := l.changesAsked(); len(third) != 0 || len(asked) != 3 || !asked[2].Equal(started) {
			t.Fatalf("a list of three pages fetched %v and the lists were read from %v, want none fetched and the next list read from %v", third, asked, started)
		}
	})
}

func TestRefreshAllMakesEveryWorkDueOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "a", false)
		l.add(t, "b", false)
		time.Sleep(time.Minute)
		if err := l.meta.RefreshAll(t.Context(), identity.Admin{}, contenttypetest.ID); err != nil {
			t.Fatal(err)
		}
		if run := l.runNow(t); run.Summary != "2 Works" || !slices.Equal(labels(run), []string{"A succeeded", "B succeeded"}) {
			t.Fatalf("the run after refresh all %q %v, want both Works", run.Summary, labels(run))
		}
		if run := l.runNow(t); run.Summary != "0 Works" {
			t.Fatalf("the next run %q, want 0 Works: refresh all asks once", run.Summary)
		}
		if err := l.meta.RefreshAll(t.Context(), identity.Admin{}, "nothing"); !errors.Is(err, works.ErrNoType) {
			t.Fatalf("refresh all of an unknown Content Type = %v, want ErrNoType", err)
		}
	})
}

func TestANewLanguageMakesEveryWorkDueAtTheNextRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "a", false)
		l.add(t, "b", false)
		time.Sleep(time.Minute)
		raw, _ := json.Marshal("fr-FR")
		if err := l.meta.Settings().Apply(t.Context(), setting.Patch{"metadata.language": raw}); err != nil {
			t.Fatal(err)
		}
		if run := l.runNow(t); run.Summary != "2 Works" {
			t.Fatalf("the run after a new language %q, want 2 Works", run.Summary)
		}
		calls := l.fx.Calls()
		for _, c := range calls[len(calls)-2:] {
			if c.Ask.Language != "fr-FR" {
				t.Fatalf("%s was refreshed in %q, want fr-FR", c.Key, c.Ask.Language)
			}
		}
	})
}

func TestFailedWorksStayDueAndRunFirst(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "a", false)
		l.add(t, "b", true)
		time.Sleep(time.Minute)
		if err := l.meta.RefreshAll(t.Context(), identity.Admin{}, contenttypetest.ID); err != nil {
			t.Fatal(err)
		}
		l.fx.Fail("a", &works.SourceError{Source: "Fixture", Words: "didn't answer", Err: works.ErrUnavailable})
		run := l.runNow(t)
		if want := []string{"A failed: Fixture: didn't answer", "B succeeded"}; run.Result != tasks.RunPartial || run.Summary != "1 of 2 Works, 1 failed" || !slices.Equal(labels(run), want) {
			t.Fatalf("a run where A failed %s %q %v, want partial, 1 of 2 Works, 1 failed, %v", run.Result, run.Summary, labels(run), want)
		}
		l.nightly(t, 1)
		l.fx.Fail("a", nil)
		l.nightly(t, 1)
		if got := labels(l.latest(t)); !slices.Equal(got, []string{"A succeeded", "B succeeded"}) {
			t.Fatalf("the night after A failed twice ran %v, want A, due longest, before B", got)
		}
	})
}

func TestAGoneWorkFailsWithItsCauseAndRidesTheEveryOtherTier(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "lost", true)
		l.fx.Fail("lost", works.ErrNotFound)
		l.nightly(t, 2)
		if run := l.latest(t); run.Result != tasks.RunFailed || !slices.Equal(labels(run), []string{"LOST failed: " + works.ErrGoneUpstream.Error()}) {
			t.Fatalf("the night a Work went upstream %s %v, want it failed as gone upstream", run.Result, labels(run))
		}
		var fetched []int
		for i, night := range l.nightly(t, 30) {
			if len(night) > 0 {
				fetched = append(fetched, 3+i)
			}
		}
		if !slices.Equal(fetched, []int{32}) {
			t.Fatalf("an unreleased Work gone upstream on night 2 was fetched again on nights %v, want night 32 alone", fetched)
		}
	})
}

func TestTheRunReads40Of42Works2Failed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		for i := range 42 {
			l.add(t, works.Key(fmt.Sprintf("w%02d", i)), false)
		}
		l.fx.Fail("w07", works.ErrUnavailable)
		l.fx.Fail("w31", works.ErrUnavailable)
		time.Sleep(time.Minute)
		if err := l.meta.RefreshAll(t.Context(), identity.Admin{}, contenttypetest.ID); err != nil {
			t.Fatal(err)
		}
		if run := l.runNow(t); run.Result != tasks.RunPartial || run.Summary != "40 of 42 Works, 2 failed" {
			t.Fatalf("the run %s %q, want partial, 40 of 42 Works, 2 failed", run.Result, run.Summary)
		}
	})
}

func TestAFailedRunWarnsAndTheNextGoodRunClears(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openRefreshed(t, 500)
		l.add(t, "a", false)
		time.Sleep(time.Minute)
		if err := l.meta.RefreshAll(t.Context(), identity.Admin{}, contenttypetest.ID); err != nil {
			t.Fatal(err)
		}
		l.fx.Fail("a", &works.SourceError{Source: "Fixture", Words: "didn't answer", Err: works.ErrUnavailable})
		if run := l.runNow(t); run.Result != tasks.RunFailed {
			t.Fatalf("a run whose every Work failed %s, want failed", run.Result)
		}
		warnings := func() []string {
			ws, err := l.tasks.Warnings(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			var out []string
			for _, w := range ws {
				out = append(out, w.Text)
			}
			return out
		}
		if got := warnings(); len(got) != 1 || !strings.HasPrefix(got[0], "Metadata refresh failed at ") || !strings.HasSuffix(got[0], ": Fixture: didn't answer") {
			t.Fatalf("Health warnings %q, want one saying Metadata refresh failed and why", got)
		}
		l.fx.Fail("a", nil)
		if run := l.runNow(t); run.Result != tasks.RunSucceeded {
			t.Fatalf("the next run %s, want succeeded", run.Result)
		}
		if got := warnings(); len(got) != 0 {
			t.Fatalf("Health warnings %q after a good run, want none", got)
		}
	})
}
