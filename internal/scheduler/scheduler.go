package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/rs/zerolog"
)

// TaskFunc is the function signature for scheduled tasks.
type TaskFunc func(ctx context.Context) error

// TaskConfig contains configuration for a scheduled task.
type TaskConfig struct {
	ID          string
	Name        string
	Description string
	Cron        string   // Cron expression: "0 0 * * *" for midnight daily
	Func        TaskFunc
	RunOnStart  bool // Execute immediately on startup
}

// TaskInfo contains information about a scheduled task for API responses.
type TaskInfo struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Cron        string     `json:"cron"`
	LastRun     *time.Time `json:"lastRun,omitempty"`
	NextRun     *time.Time `json:"nextRun,omitempty"`
	Running     bool       `json:"running"`
}

// taskEntry holds internal task state.
type taskEntry struct {
	config  TaskConfig
	job     gocron.Job
	lastRun *time.Time
	running bool
}

// Scheduler manages background scheduled tasks.
type Scheduler struct {
	gocron gocron.Scheduler
	logger zerolog.Logger
	tasks  map[string]*taskEntry
	mu     sync.RWMutex
}

// New creates a new scheduler.
func New(logger zerolog.Logger) (*Scheduler, error) {
	gs, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("failed to create gocron scheduler: %w", err)
	}

	return &Scheduler{
		gocron: gs,
		logger: logger.With().Str("component", "scheduler").Logger(),
		tasks:  make(map[string]*taskEntry),
	}, nil
}

// RegisterTask registers a new scheduled task.
func (s *Scheduler) RegisterTask(config TaskConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[config.ID]; exists {
		return fmt.Errorf("task with ID %q already registered", config.ID)
	}

	// Create the job function wrapper
	taskFunc := func() {
		s.executeTask(config.ID)
	}

	// Parse cron expression and create job
	job, err := s.gocron.NewJob(
		gocron.CronJob(config.Cron, false),
		gocron.NewTask(taskFunc),
		gocron.WithName(config.Name),
		gocron.WithTags(config.ID),
	)
	if err != nil {
		return fmt.Errorf("failed to create job for task %q: %w", config.ID, err)
	}

	s.tasks[config.ID] = &taskEntry{
		config: config,
		job:    job,
	}

	s.logger.Info().
		Str("id", config.ID).
		Str("name", config.Name).
		Str("cron", config.Cron).
		Bool("runOnStart", config.RunOnStart).
		Msg("Registered task")

	return nil
}

// executeTask runs a task and updates its state.
func (s *Scheduler) executeTask(taskID string) {
	s.mu.Lock()
	entry, exists := s.tasks[taskID]
	if !exists {
		s.mu.Unlock()
		return
	}
	entry.running = true
	s.mu.Unlock()

	startTime := time.Now()
	s.logger.Info().
		Str("id", taskID).
		Str("name", entry.config.Name).
		Msg("Starting task")

	ctx := context.Background()
	err := entry.config.Func(ctx)

	s.mu.Lock()
	entry.running = false
	entry.lastRun = &startTime
	s.mu.Unlock()

	duration := time.Since(startTime)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("id", taskID).
			Str("name", entry.config.Name).
			Dur("duration", duration).
			Msg("Task failed")
	} else {
		s.logger.Info().
			Str("id", taskID).
			Str("name", entry.config.Name).
			Dur("duration", duration).
			Msg("Task completed")
	}
}

// Start starts the scheduler and runs any tasks configured with RunOnStart.
func (s *Scheduler) Start() error {
	s.logger.Info().Msg("Starting scheduler")

	// Start the gocron scheduler
	s.gocron.Start()

	// Run tasks configured to run on start
	s.mu.RLock()
	tasksToRun := make([]string, 0)
	for id, entry := range s.tasks {
		if entry.config.RunOnStart {
			tasksToRun = append(tasksToRun, id)
		}
	}
	s.mu.RUnlock()

	// Execute startup tasks in goroutines
	for _, taskID := range tasksToRun {
		go s.executeTask(taskID)
	}

	return nil
}

// Stop stops the scheduler gracefully.
func (s *Scheduler) Stop() error {
	s.logger.Info().Msg("Stopping scheduler")
	return s.gocron.Shutdown()
}

// RunNow manually triggers a task to run immediately.
func (s *Scheduler) RunNow(taskID string) error {
	s.mu.RLock()
	entry, exists := s.tasks[taskID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %q not found", taskID)
	}

	if entry.running {
		return fmt.Errorf("task %q is already running", taskID)
	}

	go s.executeTask(taskID)
	return nil
}

// ListTasks returns information about all registered tasks.
func (s *Scheduler) ListTasks() []TaskInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]TaskInfo, 0, len(s.tasks))
	for _, entry := range s.tasks {
		info := TaskInfo{
			ID:          entry.config.ID,
			Name:        entry.config.Name,
			Description: entry.config.Description,
			Cron:        entry.config.Cron,
			LastRun:     entry.lastRun,
			Running:     entry.running,
		}

		// Get next run time from gocron
		nextRun, err := entry.job.NextRun()
		if err == nil {
			info.NextRun = &nextRun
		}

		tasks = append(tasks, info)
	}

	return tasks
}

// GetTask returns information about a specific task.
func (s *Scheduler) GetTask(taskID string) (*TaskInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %q not found", taskID)
	}

	info := &TaskInfo{
		ID:          entry.config.ID,
		Name:        entry.config.Name,
		Description: entry.config.Description,
		Cron:        entry.config.Cron,
		LastRun:     entry.lastRun,
		Running:     entry.running,
	}

	nextRun, err := entry.job.NextRun()
	if err == nil {
		info.NextRun = &nextRun
	}

	return info, nil
}
