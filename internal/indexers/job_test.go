package indexers_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
)

type alone struct{ self role.NodeID }

func (a alone) Self() role.NodeID                                 { return a.self }
func (a alone) Pick(role.Name, role.NodeID) (role.NodeID, bool)   { return a.self, true }
func (a alone) Route(role.Name, role.NodeID) (role.NodeID, error) { return a.self, nil }
func (alone) Away(role.Name, role.NodeID) (nodes.Absence, bool)   { return nodes.Absence{}, false }

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

// searchOn runs one search against n, and returns the log of the Indexers
// that ran it.
func searchOn(t *testing.T, n *slowNode, sealer *secret.Sealer) ([]indexers.Release, *bytes.Buffer, error) {
	t.Helper()
	var log bytes.Buffer
	x := indexers.New(indexers.Config{
		Sealer: sealer, Roster: alone{role.NodeID{UUID: storage.NewUUID()}},
		Searching: func(role.NodeID) searching.Remote { return n },
		Timings:   indexers.Timings{Job: 2 * time.Minute, Poll: time.Second},
		Log:       slog.New(slog.NewTextHandler(&log, nil)),
	})
	id := indexers.ID{UUID: storage.NewUUID()}
	idx := indexers.Indexer{ID: id, URL: "https://site.test/torznab", Key: sealer.Seal([]byte("k"), secret.NewColumn("indexers_indexer.api_key_sealed").Row(id.String()))}
	releases, err := x.Search(t.Context(), idx, indexers.Terms{Mode: "search", Query: "orchard"})
	return releases, &log, err
}

func TestAJobsDeadlineRunsFromWhenItsNodeStartedIt(t *testing.T) {
	sealer := secrettest.Sealer(t)
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		n := &slowNode{queue: 5 * time.Minute, take: 90 * time.Second, hold: 20 * time.Second}
		releases, _, err := searchOn(t, n, sealer)
		if err != nil || len(releases) != 1 || time.Since(began) != 6*time.Minute+30*time.Second {
			t.Fatalf("a search queued 5m that took 90s = %v, %v after %s; want its Release after 6m30s", releases, err, time.Since(began))
		}
	})
}

func TestALateJobFailsAtItsDeadlineAndIsRecordedWhenItAnswers(t *testing.T) {
	sealer := secrettest.Sealer(t)
	synctest.Test(t, func(t *testing.T) {
		began := time.Now()
		n := &slowNode{take: 3 * time.Minute, hold: 20 * time.Second}
		_, log, err := searchOn(t, n, sealer)
		var unavailable *indexers.UnavailableError
		if !errors.As(err, &unavailable) || !errors.Is(err, indexers.ErrLate) || time.Since(began) != 2*time.Minute {
			t.Fatalf("a search that takes 3m = %v after %s, want ErrLate at 2m", err, time.Since(began))
		}
		time.Sleep(time.Minute)
		synctest.Wait()
		if !strings.Contains(log.String(), "An Indexer's job finished after its deadline") || !strings.Contains(log.String(), "releases=1") {
			t.Fatalf("the late answer at 3m left %q, want it recorded", log.String())
		}
	})
}

func TestAJobIsPolledNoFasterThanThePollGap(t *testing.T) {
	sealer := secrettest.Sealer(t)
	synctest.Test(t, func(t *testing.T) {
		n := &slowNode{take: 10 * time.Second}
		if _, _, err := searchOn(t, n, sealer); err != nil || n.calls != 11 {
			t.Fatalf("a 10s job on a Node that holds no call = %v after %d calls, want its answer after 11", err, n.calls)
		}
	})
}
