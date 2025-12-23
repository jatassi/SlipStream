package watcher

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/scanner"
)

// FileEvent represents a file system event.
type FileEvent struct {
	Path      string    `json:"path"`
	Op        string    `json:"op"` // "create", "write", "remove", "rename"
	Timestamp time.Time `json:"timestamp"`
}

// FileEventHandler is called when file events are ready to be processed.
type FileEventHandler func(events []FileEvent)

// Config holds watcher configuration.
type Config struct {
	// DebounceDelay is how long to wait after the last event before processing.
	DebounceDelay time.Duration

	// MaxBatchSize is the maximum number of events to batch before forcing processing.
	MaxBatchSize int

	// RecursiveWatch enables watching subdirectories.
	RecursiveWatch bool
}

// DefaultConfig returns default watcher configuration.
func DefaultConfig() Config {
	return Config{
		DebounceDelay:  500 * time.Millisecond,
		MaxBatchSize:   100,
		RecursiveWatch: true,
	}
}

// Watcher monitors directories for file changes.
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	config    Config
	logger    zerolog.Logger
	handler   FileEventHandler

	// Tracked paths
	watchedPaths map[string]bool
	pathsMu      sync.RWMutex

	// Event debouncing
	pendingEvents map[string]FileEvent
	eventsMu      sync.Mutex
	debounceTimer *time.Timer

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a new file watcher.
func New(config Config, logger zerolog.Logger) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	w := &Watcher{
		fsWatcher:     fsWatcher,
		config:        config,
		logger:        logger.With().Str("component", "watcher").Logger(),
		watchedPaths:  make(map[string]bool),
		pendingEvents: make(map[string]FileEvent),
		ctx:           ctx,
		cancel:        cancel,
	}

	return w, nil
}

// SetHandler sets the event handler function.
func (w *Watcher) SetHandler(handler FileEventHandler) {
	w.handler = handler
}

// Start begins watching for file events.
func (w *Watcher) Start() {
	w.wg.Add(1)
	go w.eventLoop()
}

// Stop stops the watcher and waits for cleanup.
func (w *Watcher) Stop() error {
	w.cancel()
	w.wg.Wait()
	return w.fsWatcher.Close()
}

// AddPath adds a path to watch.
func (w *Watcher) AddPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	w.pathsMu.Lock()
	defer w.pathsMu.Unlock()

	if w.watchedPaths[absPath] {
		return nil // Already watching
	}

	// Add the root path
	if err := w.fsWatcher.Add(absPath); err != nil {
		return err
	}
	w.watchedPaths[absPath] = true

	w.logger.Info().Str("path", absPath).Msg("Added watch path")

	// If recursive, walk and add subdirectories
	if w.config.RecursiveWatch {
		err = filepath.WalkDir(absPath, func(subPath string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if d.IsDir() && subPath != absPath {
				if err := w.fsWatcher.Add(subPath); err != nil {
					w.logger.Warn().Err(err).Str("path", subPath).Msg("Failed to add subdirectory watch")
					return nil
				}
				w.watchedPaths[subPath] = true
			}
			return nil
		})
		if err != nil {
			w.logger.Warn().Err(err).Str("path", absPath).Msg("Error walking subdirectories")
		}
	}

	return nil
}

// RemovePath removes a path from watching.
func (w *Watcher) RemovePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	w.pathsMu.Lock()
	defer w.pathsMu.Unlock()

	// Remove all paths that start with this path
	for watchedPath := range w.watchedPaths {
		if watchedPath == absPath || (w.config.RecursiveWatch && isSubPath(watchedPath, absPath)) {
			w.fsWatcher.Remove(watchedPath)
			delete(w.watchedPaths, watchedPath)
		}
	}

	w.logger.Info().Str("path", absPath).Msg("Removed watch path")
	return nil
}

// WatchedPaths returns the list of currently watched paths.
func (w *Watcher) WatchedPaths() []string {
	w.pathsMu.RLock()
	defer w.pathsMu.RUnlock()

	paths := make([]string, 0, len(w.watchedPaths))
	for path := range w.watchedPaths {
		paths = append(paths, path)
	}
	return paths
}

// eventLoop processes fsnotify events.
func (w *Watcher) eventLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			w.flushPendingEvents()
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleFsEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			w.logger.Error().Err(err).Msg("Watcher error")
		}
	}
}

// handleFsEvent processes a single fsnotify event.
func (w *Watcher) handleFsEvent(event fsnotify.Event) {
	// Skip non-video files early
	if !scanner.IsVideoFile(filepath.Base(event.Name)) {
		// But still track directory changes for recursive watching
		if w.config.RecursiveWatch && event.Has(fsnotify.Create) {
			info, err := os.Stat(event.Name)
			if err == nil && info.IsDir() {
				w.fsWatcher.Add(event.Name)
				w.pathsMu.Lock()
				w.watchedPaths[event.Name] = true
				w.pathsMu.Unlock()
				w.logger.Debug().Str("path", event.Name).Msg("Added new subdirectory to watch")
			}
		}
		return
	}

	// Skip sample files
	if scanner.IsSampleFile(filepath.Base(event.Name)) {
		return
	}

	// Determine operation type
	var op string
	switch {
	case event.Has(fsnotify.Create):
		op = "create"
	case event.Has(fsnotify.Write):
		op = "write"
	case event.Has(fsnotify.Remove):
		op = "remove"
	case event.Has(fsnotify.Rename):
		op = "rename"
	default:
		return
	}

	fileEvent := FileEvent{
		Path:      event.Name,
		Op:        op,
		Timestamp: time.Now(),
	}

	w.addPendingEvent(fileEvent)
}

// addPendingEvent adds an event to the pending batch and resets debounce timer.
func (w *Watcher) addPendingEvent(event FileEvent) {
	w.eventsMu.Lock()
	defer w.eventsMu.Unlock()

	// Use path as key to deduplicate rapid events on same file
	w.pendingEvents[event.Path] = event

	// Check if we should force flush due to batch size
	if len(w.pendingEvents) >= w.config.MaxBatchSize {
		w.flushPendingEventsLocked()
		return
	}

	// Reset or start debounce timer
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
	}
	w.debounceTimer = time.AfterFunc(w.config.DebounceDelay, func() {
		w.eventsMu.Lock()
		defer w.eventsMu.Unlock()
		w.flushPendingEventsLocked()
	})
}

// flushPendingEvents flushes pending events (with lock).
func (w *Watcher) flushPendingEvents() {
	w.eventsMu.Lock()
	defer w.eventsMu.Unlock()
	w.flushPendingEventsLocked()
}

// flushPendingEventsLocked flushes pending events (caller must hold lock).
func (w *Watcher) flushPendingEventsLocked() {
	if len(w.pendingEvents) == 0 {
		return
	}

	// Stop timer if running
	if w.debounceTimer != nil {
		w.debounceTimer.Stop()
		w.debounceTimer = nil
	}

	// Collect events
	events := make([]FileEvent, 0, len(w.pendingEvents))
	for _, event := range w.pendingEvents {
		events = append(events, event)
	}

	// Clear pending
	w.pendingEvents = make(map[string]FileEvent)

	// Call handler in separate goroutine to avoid blocking
	if w.handler != nil && len(events) > 0 {
		go w.handler(events)
	}

	w.logger.Debug().Int("count", len(events)).Msg("Flushed file events")
}

// isSubPath checks if child is a subdirectory of parent.
func isSubPath(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return rel != ".." && !filepath.IsAbs(rel) && rel != "."
}
