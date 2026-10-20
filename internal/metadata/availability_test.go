package metadata_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// zone is the Controller's zone: three hours east of the bubble's UTC
// start, so a day begins at 21:00 UTC the day before.
var zone = time.FixedZone("UTC+3", 3*60*60)

// library is a Controller's library under the availability timer, whose
// clock is a synctest bubble's, starting on 1 January 2000 at 03:00 local.
type library struct {
	lib     *storage.DB[storage.Library]
	fx      *contenttypetest.Fixture
	works   *works.Works
	targets *targets.Targets
	hub     *live.Hub
	hook    *hook
	changed *changed
}

func openLibrary(t *testing.T) *library {
	t.Helper()
	hub := live.NewHub([]live.Event{works.ChangedEvent, works.ContentTypesChangedEvent, works.TypeChangedEvent})
	lib := storagetest.Library(t, works.Schema(), targets.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library:      lib,
		Catalogs:     map[works.TypeID]works.Catalog{contenttypetest.ID: fx},
		Source:       fx.Source(),
		Changed:      works.ChangedEvent.Bind(hub),
		TypesChanged: works.ContentTypesChangedEvent.Bind(hub),
		TypeChanged:  works.TypeChangedEvent.Bind(hub),
	})
	if err != nil {
		t.Fatal(err)
	}
	goals, err := targets.Open(t.Context(), targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &library{lib: lib, fx: fx, works: w, targets: goals, hub: hub, hook: &hook{}, changed: &changed{hub: works.ChangedEvent.Bind(hub)}}
}

// run starts the timer over local and answers how to stop it.
func (l *library) run(t *testing.T, local *storage.DB[storage.Local]) (stop func()) {
	t.Helper()
	timer := metadata.NewAvailability(metadata.AvailabilityConfig{
		Library: l.lib, Local: local, Targets: l.targets, Hub: l.hub, Changed: l.changed,
		Now:      func() time.Time { return time.Now().In(zone) },
		Released: l.hook.released,
		Log:      slog.New(slog.NewTextHandler(t.Output(), nil)),
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error)
	go func() { done <- timer.Run(ctx) }()
	stop = sync.OnceFunc(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run = %v", err)
		}
	})
	t.Cleanup(stop)
	synctest.Wait()
	return stop
}

// shelf adds a Work whose Items, keyed as in out, come out on those days
// of January 2000, monitored as monitor says.
func (l *library) shelf(t *testing.T, key works.Key, monitor bool, out map[works.Key]works.Date) works.Tree {
	t.Helper()
	var things []works.Entry
	for _, k := range []works.Key{"c1", "c2", "c3"} {
		if _, ok := out[k]; ok {
			things = append(things, works.Entry{Key: k, Label: "Thing " + string(k)})
		}
	}
	l.fx.Put(key, contenttypetest.Record{Tree: works.Fetched{Label: "Shelf " + string(key), Children: []works.Entry{
		{Key: "b1", Label: "Box", Children: things},
	}}})
	tree, _, err := l.works.Add(t.Context(), identity.Admin{}, contenttypetest.ID, key, works.Adding{
		Monitor: monitor,
		Also:    func(context.Context, *storage.Tx[storage.Library], works.Summary) error { return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range tree.Items {
		l.fx.SetTiming(it.ID, targets.Timing{Released: release.Known(out[it.Key])})
	}
	return tree
}

func item(t *testing.T, tree works.Tree, key works.Key) works.ItemID {
	t.Helper()
	for _, it := range tree.Items {
		if it.Key == key {
			return it.ID
		}
	}
	t.Fatalf("shelf %s has no Item %s", tree.Key, key)
	return works.ItemID{}
}

func jan(day int) works.Date { return works.Date{Year: 2000, Month: 1, Day: day} }

// until sleeps the bubble to the given local time in January 2000 and lets
// the timer settle.
func until(day, hour, minute int) {
	time.Sleep(time.Until(time.Date(2000, time.January, day, hour, minute, 0, 0, zone)))
	synctest.Wait()
}

// hook is a stand-in for S34 (Automatic search, Backlog search and RSS
// sync)'s hook: it records each call and fails while fail is set.
type hook struct {
	mu    sync.Mutex
	calls [][]works.ItemID
	fail  error
}

func (h *hook) released(_ context.Context, items []works.ItemID) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = append(h.calls, items)
	return h.fail
}

func (h *hook) failWith(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fail = err
}

func (h *hook) want(t *testing.T, calls ...[]works.ItemID) {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	if !reflect.DeepEqual(h.calls, calls) {
		t.Fatalf("the hook got %v, want %v", h.calls, calls)
	}
}

// changed records the timer's works.changed and passes each on to the
// hub, which wakes the timer as it does on a Controller.
type changed struct {
	hub   works.Publisher[works.WorkChanged]
	mu    sync.Mutex
	works []string
}

func (c *changed) Publish(ctx context.Context, v works.WorkChanged) {
	c.mu.Lock()
	c.works = append(c.works, v.WorkID)
	c.mu.Unlock()
	c.hub.Publish(ctx, v)
}

func (c *changed) want(t *testing.T, ws ...works.Tree) {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()
	var want []string
	for _, w := range ws {
		want = append(want, w.ID.String())
	}
	if !reflect.DeepEqual(c.works, want) {
		t.Fatalf("works.changed named %v, want %v", c.works, want)
	}
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func ids(items ...works.ItemID) []works.ItemID { return items }

func TestTheTimerAnnouncesOnTheDay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openLibrary(t)
		shelf := l.shelf(t, "s1", true, map[works.Key]works.Date{"c1": jan(3), "c2": jan(5)})
		l.run(t, storagetest.Local(t, metadata.LocalSchema()))
		l.hook.want(t)

		until(2, 23, 59)
		l.hook.want(t)
		l.changed.want(t)

		until(3, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c1")))
		l.changed.want(t, shelf)

		until(5, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c1")), ids(item(t, shelf, "c2")))
		l.changed.want(t, shelf, shelf)
	})
}

func TestTheTimerCatchesUpAfterARestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openLibrary(t)
		shelf := l.shelf(t, "s1", true, map[works.Key]works.Date{"c1": jan(2), "c2": jan(3), "c3": jan(5)})
		local := storagetest.Local(t, metadata.LocalSchema())
		stop := l.run(t, local)
		until(1, 12, 0)
		stop()

		until(4, 12, 0)
		l.hook.want(t)
		l.run(t, local)
		l.hook.want(t, ids(item(t, shelf, "c1"), item(t, shelf, "c2")))
		l.changed.want(t, shelf)

		until(5, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c1"), item(t, shelf, "c2")), ids(item(t, shelf, "c3")))
	})
}

func TestALostLocalDBStartsFromToday(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openLibrary(t)
		shelf := l.shelf(t, "s1", true, map[works.Key]works.Date{
			"c1": {Year: 1999, Month: 12, Day: 31}, "c2": jan(1), "c3": jan(2),
		})
		l.run(t, storagetest.Local(t, metadata.LocalSchema()))
		l.hook.want(t, ids(item(t, shelf, "c2")))

		until(2, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c2")), ids(item(t, shelf, "c3")))
	})
}

func TestMonitoringReArmsTheTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openLibrary(t)
		shelf := l.shelf(t, "s1", false, map[works.Key]works.Date{"c1": jan(3)})
		l.run(t, storagetest.Local(t, metadata.LocalSchema()))

		until(2, 12, 0)
		if err := l.works.SetMonitored(t.Context(), identity.Admin{}, ids(item(t, shelf, "c1")), true); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		l.hook.want(t)

		until(3, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c1")))
	})
}

func TestASettingChangeReArmsTheTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openLibrary(t)
		shelf := l.shelf(t, "s1", true, map[works.Key]works.Date{"c1": jan(10)})
		l.run(t, storagetest.Local(t, metadata.LocalSchema()))

		until(2, 12, 0)
		l.fx.SetTiming(item(t, shelf, "c1"), targets.Timing{Released: release.Known(jan(3))})
		patch := setting.Patch{contenttypetest.Depth.Key: json.RawMessage("4")}
		if _, err := l.works.SetSettings(t.Context(), identity.Admin{}, contenttypetest.ID, patch); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		l.hook.want(t)

		until(3, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c1")))
	})
}

func TestTheHookGetsOnlyMonitoredItems(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openLibrary(t)
		watched := l.shelf(t, "s1", true, map[works.Key]works.Date{"c1": jan(2), "c2": jan(2)})
		l.shelf(t, "s2", false, map[works.Key]works.Date{"c1": jan(2)})
		if err := l.works.SetMonitored(t.Context(), identity.Admin{}, ids(item(t, watched, "c2")), false); err != nil {
			t.Fatal(err)
		}
		l.run(t, storagetest.Local(t, metadata.LocalSchema()))

		until(2, 0, 0)
		l.hook.want(t, ids(item(t, watched, "c1")))
		l.changed.want(t, watched)
	})
}

func TestAFailedHookGetsTheItemsAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		l := openLibrary(t)
		shelf := l.shelf(t, "s1", true, map[works.Key]works.Date{"c1": jan(2)})
		l.hook.failWith(errors.New("the queue is full"))
		l.run(t, storagetest.Local(t, metadata.LocalSchema()))

		until(2, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c1")))
		l.hook.failWith(nil)

		until(2, 0, 5)
		l.hook.want(t, ids(item(t, shelf, "c1")), ids(item(t, shelf, "c1")))

		until(3, 0, 0)
		l.hook.want(t, ids(item(t, shelf, "c1")), ids(item(t, shelf, "c1")))
	})
}
