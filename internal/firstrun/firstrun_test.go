package firstrun_test

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type poked struct{}

var pokedEvent = live.AdminEvent[poked]("test.poked")

type counter struct{ n atomic.Int64 }

func (c *counter) Publish(context.Context, firstrun.Changed) { c.n.Add(1) }

// steps is every step, the roots step done while roots holds and the media
// step shown while media holds.
func steps(roots, media *atomic.Bool) []firstrun.Step {
	fact := func(b *atomic.Bool) firstrun.Fact {
		return func(context.Context, storage.Reader[storage.Library]) (bool, error) { return b.Load(), nil }
	}
	var out []firstrun.Step
	for _, id := range firstrun.IDs() {
		s := firstrun.Step{ID: id}
		if id == firstrun.RootsID {
			s.Done, s.Events = fact(roots), []live.Event{pokedEvent}
		}
		if id == firstrun.MediaID {
			s.Shows = fact(media)
		}
		out = append(out, s)
	}
	return out
}

type world struct {
	fr           *firstrun.FirstRun
	hub          *live.Hub
	published    *counter
	roots, media atomic.Bool
}

func newWorld(t *testing.T, lib *storage.DB[storage.Library]) *world {
	t.Helper()
	w := &world{hub: live.NewHub([]live.Event{pokedEvent, firstrun.ChangedEvent}), published: &counter{}}
	t.Cleanup(w.hub.Close)
	w.fr = firstrun.New(firstrun.Config{Library: lib, Steps: steps(&w.roots, &w.media), Hub: w.hub, Changed: w.published, Log: slog.New(slog.DiscardHandler)})
	return w
}

// view shows ids, each in the state at its index in states.
func view(open bool, ids []firstrun.StepID, states ...firstrun.State) firstrun.View {
	v := firstrun.View{Open: open, Steps: []firstrun.StepView{}}
	for i, id := range ids {
		v.Steps = append(v.Steps, firstrun.StepView{ID: id, State: states[i]})
	}
	return v
}

func TestNewPanicsUnlessTheStepsAreIDs(t *testing.T) {
	all := steps(&atomic.Bool{}, &atomic.Bool{})
	cases := map[string][]firstrun.Step{
		"one missing":  all[1:],
		"out of order": append([]firstrun.Step{all[1], all[0]}, all[2:]...),
		"one unknown":  append(append([]firstrun.Step{}, all...), firstrun.Step{ID: "trash"}),
	}
	for name, list := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("New didn't panic")
				}
			}()
			firstrun.New(firstrun.Config{Steps: list})
		})
	}
}

func TestFirstSetupIsOpenUntilFinishedAndThenSkipsWhatIsLeft(t *testing.T) {
	ctx := t.Context()
	w := newWorld(t, storagetest.Library(t, identity.Schema(), firstrun.Schema()))
	a, r, l := firstrun.AddressID, firstrun.RootsID, firstrun.LanguageID
	six := []firstrun.StepID{a, firstrun.PasskeyID, l, r, firstrun.DownloadsID, firstrun.TargetsID}
	seven := firstrun.IDs()
	todo, done, skipped := firstrun.StateTodo, firstrun.StateDone, firstrun.StateSkipped

	got, err := w.fr.View(ctx)
	if want := view(true, six, todo, todo, todo, todo, todo, todo); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("a fresh View = %+v, %v; want %+v", got, err, want)
	}
	if _, err = w.fr.Set(ctx, a, firstrun.OutcomeDone); err != nil {
		t.Fatal(err)
	}
	if got, err = w.fr.Set(ctx, r, firstrun.OutcomeSkipped); err != nil {
		t.Fatal(err)
	}
	if want := view(true, six, done, todo, todo, skipped, todo, todo); !reflect.DeepEqual(got, want) {
		t.Fatalf("Set = %+v, want %+v", got, want)
	}
	if w.published.n.Load() != 2 {
		t.Errorf("two step writes published %d times, want 2", w.published.n.Load())
	}
	if _, err := w.fr.Set(ctx, r, firstrun.OutcomeSkipped); err != nil || w.published.n.Load() != 2 {
		t.Errorf("storing the same skip again = %v and published %d times, want nil and 2", err, w.published.n.Load())
	}

	w.roots.Store(true)
	w.media.Store(true)
	got, _ = w.fr.View(ctx)
	if want := view(true, seven, done, todo, todo, todo, done, todo, todo); !reflect.DeepEqual(got, want) {
		t.Fatalf("after roots and media came to hold, View = %+v, want %+v", got, want)
	}
	if _, err := w.fr.Set(ctx, "trash", firstrun.OutcomeDone); !errors.Is(err, firstrun.ErrStepUnknown) {
		t.Errorf("Set on an unknown id = %v, want ErrStepUnknown", err)
	}

	if err := w.fr.Finish(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.fr.Finish(ctx); err != nil || w.published.n.Load() != 3 {
		t.Errorf("a second Finish = %v and published %d times in all, want nil and 3", err, w.published.n.Load())
	}
	if open, err := w.fr.IsOpen(ctx); err != nil || open {
		t.Errorf("IsOpen after Finish = %v, %v; want false", open, err)
	}
	got, _ = w.fr.View(ctx)
	if want := view(false, seven, done, skipped, skipped, skipped, done, skipped, skipped); !reflect.DeepEqual(got, want) {
		t.Fatalf("a finished View = %+v, want %+v", got, want)
	}
	if _, err := w.fr.Set(ctx, l, firstrun.OutcomeDone); !errors.Is(err, firstrun.ErrFinished) {
		t.Errorf("Set after Finish = %v, want ErrFinished", err)
	}
}

func TestRunRepublishesTheStepsEventsUntilFinished(t *testing.T) {
	ctx := t.Context()
	w := newWorld(t, storagetest.Library(t, identity.Schema(), firstrun.Schema()))
	ran := make(chan error, 1)
	go func() { ran <- w.fr.Run(ctx) }()
	poke := pokedEvent.Bind(w.hub)
	deadline := time.Now().Add(5 * time.Second)
	for w.published.n.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("Run didn't publish after a step's event")
		}
		poke.Publish(ctx, poked{})
		time.Sleep(10 * time.Millisecond)
	}
	if err := w.fr.Finish(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-ran:
		if err != nil {
			t.Errorf("Run = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run didn't return after Finish")
	}
	if err := w.fr.Run(ctx); err != nil {
		t.Errorf("Run on a finished first setup = %v, want it to return nil at once", err)
	}
}

func TestALibraryClaimedBeforeFirstSetupExistedStartsFinished(t *testing.T) {
	ctx := t.Context()
	log := slog.New(slog.DiscardHandler)
	open := func(path string, schemas ...storage.Schema[storage.Library]) *storage.DB[storage.Library] {
		lib, err := storage.OpenLibrary(ctx, path, schemas, nil, storage.DefaultTimings(), log)
		if err != nil {
			t.Fatal(err)
		}
		return lib
	}
	claimed := filepath.Join(t.TempDir(), storage.LibraryFile)
	old := open(claimed, identity.Schema())
	identitytest.Admin(t, old, "ada@example.com")
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	lib := open(claimed, identity.Schema(), firstrun.Schema())
	t.Cleanup(func() { _ = lib.Close() })
	if open, err := newWorld(t, lib).fr.IsOpen(ctx); err != nil || open {
		t.Errorf("IsOpen on a library claimed before first setup = %v, %v; want false", open, err)
	}
	if open, err := newWorld(t, storagetest.Library(t, identity.Schema(), firstrun.Schema())).fr.IsOpen(ctx); err != nil || !open {
		t.Errorf("IsOpen on a fresh library = %v, %v; want true", open, err)
	}
}
