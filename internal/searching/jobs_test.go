package searching

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	searchv1 "github.com/jatassi/slipstream/internal/pb/slipstream/search/v1"
)

// stall is a driver whose searches send their first request at once when
// sends is set, and answer once release is closed or their job ends.
type stall struct {
	sends   bool
	release chan struct{}
	runs    atomic.Int32
}

func (d *stall) Search(ctx context.Context, s *Site, _ Query) ([]Release, error) {
	d.runs.Add(1)
	if d.sends {
		s.run.start()
	}
	select {
	case <-d.release:
		return []Release{{Title: "Orchard.1080p"}}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("stall: %w", ctx.Err())
	}
}

func (*stall) Recent(context.Context, *Site) ([]Release, error) { panic("not asked") }

func (*stall) Fetch(context.Context, *Site, Link) (Got, error) { panic("not asked") }

func (*stall) Test(context.Context, *Site, *CaptchaAnswer) (Report, error) { panic("not asked") }

func (*stall) Captcha(context.Context, *Site) (Captcha, error) { panic("not asked") }

var testTimings = Timings{Request: time.Minute, Hold: 20 * time.Second, JobCap: 10 * time.Minute, Keep: 2 * time.Minute, Results: 1000}

func stalled(d *stall) *Service {
	s := New(Config{HTTP: &http.Client{}, Timings: testTimings, Log: slog.New(slog.DiscardHandler)})
	s.drivers[Torznab] = d
	return s
}

func request(id string) *searchv1.RunRequest {
	return &searchv1.RunRequest{Job: encodeJob(Job{ID: id, Indexer: "i", Kind: Torznab, Address: "https://site.test/torznab", Call: SearchCall{}})}
}

func TestAJobIsHeldJoinedAndKeptForALostAnswer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &stall{sends: true, release: make(chan struct{})}
		s := stalled(d)
		began := time.Now()
		res, err := s.Run(t.Context(), request("j1"))
		if err != nil || res.GetRunning().GetStartedUnixMs() != began.UnixMilli() || time.Since(began) != testTimings.Hold {
			t.Fatalf("Run = %v, %v after %s; want Running since %d after the hold", res, err, time.Since(began), began.UnixMilli())
		}
		joined := make(chan *searchv1.RunResponse)
		go func() {
			res, _ := s.Run(context.Background(), request("j1"))
			joined <- res
		}()
		time.Sleep(time.Second)
		close(d.release)
		answer := <-joined
		if got := answer.GetDone(); got.GetStartedUnixMs() != began.UnixMilli() || len(got.GetReleases().GetReleases()) != 1 || got.GetOutcome().GetOk() == nil {
			t.Fatalf("the joined call = %v, want the one Release, ok, started %d", answer, began.UnixMilli())
		}
		time.Sleep(testTimings.Keep - time.Second)
		if res, _ := s.Run(t.Context(), request("j1")); res.GetDone() == nil || d.runs.Load() != 1 {
			t.Fatalf("a call within keep = %v after %d runs, want the kept answer of one run", res, d.runs.Load())
		}
		time.Sleep(2 * time.Second)
		if res, _ := s.Run(t.Context(), request("j1")); res.GetDone() == nil || d.runs.Load() != 2 {
			t.Fatalf("a call past keep = %v after %d runs, want a second run", res, d.runs.Load())
		}
	})
}

func TestAJobWaitingForItsFirstRequestHasNotStarted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &stall{release: make(chan struct{})}
		s := stalled(d)
		res, err := s.Run(t.Context(), request("j2"))
		if err != nil || res.GetRunning() == nil || res.GetRunning().GetStartedUnixMs() != 0 {
			t.Fatalf("Run = %v, %v; want Running not yet started", res, err)
		}
		close(d.release)
		synctest.Wait()
	})
}

func TestAJobEndsAtItsCap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := stalled(&stall{sends: true, release: make(chan struct{})})
		began := time.Now()
		for {
			res, err := s.Run(t.Context(), request("j3"))
			if err != nil {
				t.Fatal(err)
			}
			if done := res.GetDone(); done != nil {
				if got := done.GetOutcome().GetFailing().GetText(); got != "the job ran past 10m0s" || time.Since(began) != testTimings.JobCap {
					t.Fatalf("the capped job = %q after %s, want failing past 10m0s", got, time.Since(began))
				}
				return
			}
		}
	})
}

func TestACallCutShortLeavesItsJobRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &stall{sends: true, release: make(chan struct{})}
		s := stalled(d)
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		if _, err := s.Run(ctx, request("j4")); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("a call cut short = %v, want its deadline", err)
		}
		close(d.release)
		if res, _ := s.Run(t.Context(), request("j4")); res.GetDone() == nil || d.runs.Load() != 1 {
			t.Fatalf("the next call = %v after %d runs, want the first run's answer", res, d.runs.Load())
		}
	})
}

func TestTheSpacerSpacesEachHostByTheFloorOrALongerDelay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sp := &spacer{floor: 2 * time.Second, next: map[string]time.Time{}}
		began := time.Now()
		for _, host := range []string{"a.test", "b.test", "a.test", "a.test"} {
			if err := sp.wait(t.Context(), host, time.Second); err != nil {
				t.Fatal(err)
			}
		}
		if waited := time.Since(began); waited != 4*time.Second {
			t.Fatalf("three requests to a.test and one to b.test took %s, want 4s", waited)
		}
		for range 2 {
			if err := sp.wait(t.Context(), "c.test", 3*time.Second); err != nil {
				t.Fatal(err)
			}
		}
		if waited := time.Since(began); waited != 7*time.Second {
			t.Fatalf("two requests to c.test with a 3s delay took %s in all, want 7s", waited)
		}
	})
}
