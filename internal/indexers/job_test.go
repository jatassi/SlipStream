package indexers_test

import (
	"context"
	"fmt"
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
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// slowNode is a Node whose one job waits queue for its first request,
// answers take after it, and holds each Run call up to hold.
type slowNode struct {
	queue, take, hold time.Duration
	mu                sync.Mutex
	begun             time.Time
	calls             int
}

func (n *slowNode) Run(ctx context.Context, _ searching.Job) (searching.Poll, error) {
	n.mu.Lock()
	if n.begun.IsZero() {
		n.begun = time.Now()
	}
	n.calls++
	started := n.begun.Add(n.queue)
	n.mu.Unlock()
	done := started.Add(n.take)
	t := time.NewTimer(min(time.Until(done), n.hold))
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
		return searching.Poll{}, fmt.Errorf("hold the call: %w", ctx.Err())
	}
	switch now := time.Now(); {
	case !now.Before(done):
		return searching.Poll{Started: started, Answer: &searching.Answer{Started: started, Releases: []searching.Release{{Title: "Orchard.1080p"}}}}, nil
	case now.Before(started):
		return searching.Poll{}, nil
	}
	return searching.Poll{Started: started}, nil
}

// searchOn runs one search against n on a saved Indexer, with the
// Indexers running.
func searchOn(t *testing.T, n *slowNode) (indexers.Answer, *storage.DB[storage.Library], indexers.ID) {
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
		Timings: indexers.Timings{Job: 2 * time.Minute, Poll: time.Second, Limited: time.Minute, Ladder: []time.Duration{time.Minute}},
		Log:     slog.New(slog.DiscardHandler),
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
	var lines []indexers.Line
	err = lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		lines, err = x.Plan(ctx, tx, indexerstest.Type)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	answers, err := x.Search(t.Context(), indexers.Search{Type: indexerstest.Type, Terms: indexers.Terms{Mode: "search", Query: "orchard"}, Asked: lines})
	if err != nil || len(answers) != 1 {
		t.Fatalf("search = %+v, %v; want one answer", answers, err)
	}
	return answers[0], lib, id
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func TestAJobsDeadlineRunsFromWhenItsNodeStartedIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		n := &slowNode{queue: 5 * time.Minute, take: 90 * time.Second, hold: 20 * time.Second}
		a, _, _ := searchOn(t, n)
		if a.Outcome.Kind != indexers.Found || len(a.Releases) != 1 || time.Since(began) != 6*time.Minute+30*time.Second {
			t.Fatalf("a search queued 5m that took 90s = %+v after %s; want its Release after 6m30s", a, time.Since(began))
		}
	})
}

func TestALateJobIsLateAtItsDeadlineAndIsRecordedWhenItAnswers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		n := &slowNode{take: 3 * time.Minute, hold: 20 * time.Second}
		a, lib, id := searchOn(t, n)
		if a.Outcome.Kind != indexers.Late || time.Since(began) != 2*time.Minute {
			t.Fatalf("a search that takes 3m = %+v after %s, want late at 2m", a.Outcome, time.Since(began))
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		var outcome string
		var releases, successes int
		err := lib.QueryRowContext(t.Context(), `SELECT outcome, releases,
			(SELECT count(*) FROM indexers_status s WHERE s.indexer_id = q.indexer_id AND success_at IS NOT NULL)
			FROM indexers_query q WHERE indexer_id = ?`, id).Scan(&outcome, &releases, &successes)
		if err != nil {
			t.Fatal(err)
		}
		if outcome != "ok" || releases != 1 || successes != 1 {
			t.Fatalf("the late answer at 3m left its query %s with %d Releases, want ok with 1 and a success in the status", outcome, releases)
		}
	})
}

func TestAJobIsPolledNoFasterThanThePollGap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := &slowNode{take: 10 * time.Second}
		if a, _, _ := searchOn(t, n); a.Outcome.Kind != indexers.Found || n.calls != 11 {
			t.Fatalf("a 10s job on a Node that holds no call = %+v after %d calls, want its answer after 11", a.Outcome, n.calls)
		}
	})
}
