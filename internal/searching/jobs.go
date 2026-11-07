package searching

import (
	"sync"
	"time"
)

// jobs are the jobs running on this Node and those that finished within
// keep, by id.
type jobs struct {
	keep time.Duration
	mu   sync.Mutex
	byID map[string]*run
}

// run is one job's run. Its answer is set once done is closed.
type run struct {
	done    chan struct{}
	answer  Answer
	mu      sync.Mutex
	started time.Time
}

// join is the run of job id: the one running or kept, else a new one that
// do runs.
func (js *jobs) join(id string, do func(*run) Answer) *run {
	js.mu.Lock()
	defer js.mu.Unlock()
	if r := js.byID[id]; r != nil {
		return r
	}
	r := &run{done: make(chan struct{})}
	js.byID[id] = r
	go func() {
		a := do(r)
		a.Started = r.startedAt()
		r.answer = a
		close(r.done)
		time.AfterFunc(js.keep, func() { js.forget(id, r) })
	}()
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
