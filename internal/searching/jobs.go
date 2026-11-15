package searching

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// jobs are the jobs running on this Node and those that finished within
// their keep, by id. Each runs on life, which stop ends.
type jobs struct {
	mu      sync.Mutex
	byID    map[string]*run
	life    context.Context
	end     context.CancelFunc
	running sync.WaitGroup
}

func newJobs() *jobs {
	life, end := context.WithCancel(context.Background())
	return &jobs{byID: map[string]*run{}, life: life, end: end}
}

// stop cancels every running job and waits for them to answer.
func (js *jobs) stop(ctx context.Context) error {
	js.end()
	done := make(chan struct{})
	go func() {
		js.running.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for the running jobs: %w", ctx.Err())
	}
}

// run is one job's run. Its answer is set once done is closed.
type run struct {
	done    chan struct{}
	answer  Answer
	mu      sync.Mutex
	started time.Time
}

// join is the run of job id: the one running or kept, else a new one that
// do runs on the jobs' life and keeps for keep once it answers.
func (js *jobs) join(id string, keep time.Duration, do func(context.Context, *run) Answer) *run {
	js.mu.Lock()
	defer js.mu.Unlock()
	if r := js.byID[id]; r != nil {
		return r
	}
	r := &run{done: make(chan struct{})}
	js.byID[id] = r
	js.running.Go(func() {
		a := do(js.life, r)
		a.Started = r.startedAt()
		r.answer = a
		close(r.done)
		time.AfterFunc(keep, func() { js.forget(id, r) })
	})
	return r
}

func (js *jobs) forget(id string, r *run) {
	js.mu.Lock()
	defer js.mu.Unlock()
	if js.byID[id] == r {
		delete(js.byID, id)
	}
}

// start marks the run started at its first request.
func (r *run) start() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started.IsZero() {
		r.started = time.Now()
	}
}

func (r *run) startedAt() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.started
}
