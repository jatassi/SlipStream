package indexers_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// slowNode is a Node whose one job waits queue for its first request,
// answers take after it, and holds each Run call up to hold. A held call
// ends early before the caller's deadline, as a call over the link does
// when its timeout reaches the Node in whole milliseconds. A fetch's
// answer is a .torrent, a search's one Release.
type slowNode struct {
	queue, take, hold, early time.Duration
	mu                       sync.Mutex
	begun                    time.Time
	calls                    int
}

func (*slowNode) ReachSolver(context.Context, searching.Solver) (int, error) {
	return 0, errors.New("the slow Node has no Challenge solver")
}

func (n *slowNode) Run(ctx context.Context, j searching.Job) (searching.Poll, error) {
	n.mu.Lock()
	if n.begun.IsZero() {
		n.begun = time.Now()
	}
	n.calls++
	started := n.begun.Add(n.queue)
	n.mu.Unlock()
	if d, ok := ctx.Deadline(); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, d.Add(-n.early))
		defer cancel()
	}
	done := started.Add(n.take)
	t := time.NewTimer(min(time.Until(done), n.hold))
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
		return searching.Poll{}, rpc.Errorf(rpc.CodeDeadlineExceeded, "hold the call: %v", ctx.Err())
	}
	switch now := time.Now(); {
	case !now.Before(done):
		a := &searching.Answer{Started: started, Releases: []searching.Release{{Title: "Orchard.1080p"}}}
		if _, ok := j.Call.(searching.FetchCall); ok {
			a = &searching.Answer{Started: started, Got: searching.Torrent{Raw: []byte("d4:infoe")}}
		}
		return searching.Poll{Started: started, Answer: a}, nil
	case now.Before(started):
		return searching.Poll{}, nil
	}
	return searching.Poll{Started: started}, nil
}

// slowWorld is one saved Indexer whose jobs run on a slowNode, with the
// Indexers running.
type slowWorld struct {
	x   *indexers.Indexers
	lib *storage.DB[storage.Library]
	id  indexers.ID
}

func newSlowWorld(t *testing.T, n *slowNode) slowWorld {
	t.Helper()
	sealer := secrettest.Sealer(t)
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), indexers.Schema())
	home := role.NodeID{UUID: storage.NewUUID()}
	id := indexers.ID{UUID: storage.NewUUID()}
	err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := nodestest.Insert(ctx, tx, home, "home"); err != nil {
			return err
		}
		return indexerstest.Torznab(ctx, tx, sealer, id, "Fixture Site", home, "https://site.test/torznab", "k")
	})
	if err != nil {
		t.Fatal(err)
	}
	x := indexers.New(indexers.Config{
		Library: lib, Sealer: sealer, Roster: indexerstest.Alone(home),
		Searching: func(role.NodeID) searching.Remote { return n },
		Types:     map[works.TypeID]indexers.Facet{indexerstest.Type: indexerstest.Facet{}},
		Changed:   discard[indexers.Changed]{}, Status: discard[indexers.StatusChanged]{}, Trackers: discard[indexers.TrackerChanged]{},
		Settings: discard[indexers.SettingsChanged]{},
		Timings:  indexers.Timings{Job: 2 * time.Minute, Poll: time.Second, Limited: time.Minute, Ladder: []time.Duration{time.Minute}},
		Log:      slog.New(slog.DiscardHandler),
	})
	ctx, stop := context.WithCancel(t.Context())
	ran := make(chan struct{})
	go func() {
		_ = x.Run(ctx)
		close(ran)
	}()
	t.Cleanup(func() {
		stop()
		<-ran
	})
	return slowWorld{x: x, lib: lib, id: id}
}

// search runs one automatic search of the world's Indexer.
func (w slowWorld) search(t *testing.T) indexers.Answer {
	t.Helper()
	var lines []indexers.Line
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		lines, err = w.x.Plan(ctx, tx, indexerstest.Type, indexers.AutoSearch)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	answers, err := w.x.Search(t.Context(), indexers.Search{Type: indexerstest.Type, Terms: indexers.Terms{Mode: "search", Query: "orchard"}, Asked: lines}, nil)
	if err != nil || len(answers) != 1 {
		t.Fatalf("search = %+v, %v; want one answer", answers, err)
	}
	return answers[0]
}

// fetch takes and fetches a Release's .torrent link from the world's
// Indexer.
func (w slowWorld) fetch(t *testing.T) (indexers.Got, error) {
	t.Helper()
	r := indexers.Release{Indexer: w.id, Release: searching.Release{Title: "Orchard.1080p", Link: searching.LinkOf("https://site.test/get/1")}}
	var ticket indexers.Ticket
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		ticket, err = w.x.Take(ctx, tx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return w.x.Fetch(t.Context(), ticket, r.Link)
}

// setWait stores raw as the search wait.
func (w slowWorld) setWait(t *testing.T, raw string) error {
	t.Helper()
	return w.x.Settings().Apply(t.Context(), setting.Patch{indexers.SearchWait.Key: []byte(raw)})
}

// query is the world's Indexer's one query row: its outcome and Releases.
func (w slowWorld) query(t *testing.T) (outcome string, releases, successes int) {
	t.Helper()
	err := w.lib.QueryRowContext(t.Context(), `SELECT outcome, releases,
		(SELECT count(*) FROM indexers_status s WHERE s.indexer_id = q.indexer_id AND success_at IS NOT NULL)
		FROM indexers_query q WHERE indexer_id = ?`, w.id).Scan(&outcome, &releases, &successes)
	if err != nil {
		t.Fatal(err)
	}
	return outcome, releases, successes
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func TestASearchWaitsFromItsFirstAskWhileItsNodeQueuesTheJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		w := newSlowWorld(t, &slowNode{queue: 5 * time.Minute, take: 90 * time.Second, hold: 20 * time.Second})
		if a := w.search(t); a.Outcome.Kind != indexers.Late || time.Since(began) != 2*time.Minute {
			t.Fatalf("a search queued 5m on its Node = %+v after %s, want late at the 2m default wait", a.Outcome, time.Since(began))
		}
	})
}

func TestAFetchsDeadlineRunsFromWhenItsNodeStartedIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		w := newSlowWorld(t, &slowNode{queue: 5 * time.Minute, take: 90 * time.Second, hold: 20 * time.Second})
		if err := w.setWait(t, "10"); err != nil {
			t.Fatal(err)
		}
		got, err := w.fetch(t)
		if _, ok := got.(indexers.Torrent); err != nil || !ok || time.Since(began) != 6*time.Minute+30*time.Second {
			t.Fatalf("a fetch queued 5m that took 90s = %+v, %v after %s; want its .torrent after 6m30s, whatever the search wait", got, err, time.Since(began))
		}
	})
}

func TestALateJobIsLateAtItsDeadlineAndIsRecordedWhenItAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		w := newSlowWorld(t, &slowNode{take: 3 * time.Minute, hold: 20 * time.Second})
		if a := w.search(t); a.Outcome.Kind != indexers.Late || time.Since(began) != 2*time.Minute {
			t.Fatalf("a search that takes 3m = %+v after %s, want late at 2m", a.Outcome, time.Since(began))
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		if outcome, releases, successes := w.query(t); outcome != "ok" || releases != 1 || successes != 1 {
			t.Fatalf("the late answer at 3m left its query %s with %d Releases, want ok with 1 and a success in the status", outcome, releases)
		}
	})
}

func TestASearchPastTheStoredWaitIsLateBeforeItsNodeSaysItStarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newSlowWorld(t, &slowNode{take: 15 * time.Second, hold: 20 * time.Second})
		if err := w.setWait(t, "10"); err != nil {
			t.Fatal(err)
		}
		began := time.Now()
		if a := w.search(t); a.Outcome.Kind != indexers.Late || time.Since(began) != 10*time.Second {
			t.Fatalf("a 15s search with a 10s wait, its Node holding each call 20s = %+v after %s, want late at 10s", a.Outcome, time.Since(began))
		}
		time.Sleep(10 * time.Second)
		synctest.Wait()
		if outcome, releases, _ := w.query(t); outcome != "ok" || releases != 1 {
			t.Fatalf("the late answer at 15s left its query %s with %d Releases, want ok with 1", outcome, releases)
		}
	})
}

func TestTheSearchWaitIsStoredFrom10SecondsTo10MinutesInWholeSeconds(t *testing.T) {
	w := newSlowWorld(t, &slowNode{})
	wait := func() setting.DurationField {
		t.Helper()
		fields, err := w.x.Settings().Fields(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range fields {
			if d, ok := f.(setting.DurationField); ok && d.Key == indexers.SearchWait.Key {
				return d
			}
		}
		t.Fatalf("the Indexer settings %+v hold no search wait", fields)
		return setting.DurationField{}
	}
	if d := wait(); d.Value != 120 || !d.IsDefault || d.Min != 10 || d.Max != 600 || d.Step != 1 {
		t.Fatalf("the search wait at first = %+v, want the 2m default from 10s to 10m by the second", d)
	}
	if err := w.setWait(t, "45"); err != nil {
		t.Fatal(err)
	}
	if d := wait(); d.Value != 45 || d.IsDefault {
		t.Fatalf("the search wait after setting 45s = %+v, want 45s", d)
	}
	for _, raw := range []string{"9", "601", "10.5"} {
		if err := w.setWait(t, raw); !errors.As(err, new(*setting.RangeError)) {
			t.Errorf("setting the search wait to %s = %v, want *setting.RangeError", raw, err)
		}
	}
	if err := w.setWait(t, "null"); err != nil {
		t.Fatal(err)
	}
	if d := wait(); d.Value != 120 || !d.IsDefault {
		t.Fatalf("the search wait after a reset = %+v, want the 2m default", d)
	}
}

func TestAJobWhoseLastCallEndsJustBeforeItsDeadlineIsLate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newSlowWorld(t, &slowNode{take: 3 * time.Minute, hold: 20 * time.Second, early: time.Millisecond / 2})
		if a := w.search(t); a.Outcome.Kind != indexers.Late {
			t.Fatalf("a search that takes 3m, its last call ended 0.5ms before the 2m deadline = %+v, want late", a.Outcome)
		}
	})
}

func TestAJobIsPolledNoFasterThanThePollGap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := &slowNode{take: 10 * time.Second}
		if a := newSlowWorld(t, n).search(t); a.Outcome.Kind != indexers.Found || n.calls != 11 {
			t.Fatalf("a 10s job on a Node that holds no call = %+v after %d calls, want its answer after 11", a.Outcome, n.calls)
		}
	})
}
