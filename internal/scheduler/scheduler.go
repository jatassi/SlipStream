package scheduler

import (
	"sync"
	"time"
)

// Scheduler manages background jobs.
type Scheduler struct {
	jobs    map[string]*Job
	mu      sync.RWMutex
	running bool
}

// Job represents a scheduled job.
type Job struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Interval time.Duration `json:"interval"`
	LastRun  time.Time     `json:"lastRun"`
	NextRun  time.Time     `json:"nextRun"`
	Running  bool          `json:"running"`
	Func     func() error  `json:"-"`
}

// New creates a new scheduler.
func New() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
	}
}

// Add adds a job to the scheduler.
func (s *Scheduler) Add(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Start starts the scheduler.
func (s *Scheduler) Start() {
	s.running = true
	// TODO: Implement job execution loop
}

// Stop stops the scheduler.
func (s *Scheduler) Stop() {
	s.running = false
}

// List returns all scheduled jobs.
func (s *Scheduler) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*Job, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}
