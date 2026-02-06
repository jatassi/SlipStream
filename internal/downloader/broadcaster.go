package downloader

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	// Fast polling when downloads are active
	activeInterval = 2 * time.Second
	// Slow polling when queue is idle
	idleInterval = 30 * time.Second
)

// Broadcaster defines the interface for broadcasting messages.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{}) error
}

// QueueTrigger defines the interface for triggering immediate queue broadcasts.
type QueueTrigger interface {
	Trigger()
}

// CompletionHandler is called when downloads transition to completed state.
type CompletionHandler interface {
	CheckAndProcessCompletedDownloads(ctx context.Context) error
}

// QueueBroadcaster periodically polls the download queue and broadcasts updates via WebSocket.
// Uses adaptive polling: fast when downloads are active, slow when idle.
// When downloads complete, it triggers the completion handler for immediate import processing.
type QueueBroadcaster struct {
	service           *Service
	hub               Broadcaster
	completionHandler CompletionHandler
	logger            zerolog.Logger
	stopCh            chan struct{}
	stoppedCh         chan struct{}
	triggerCh         chan struct{}
	mu                sync.Mutex
	running           bool
	activeMode        bool // true when polling at activeInterval
	processingImports bool // true when import processing is in progress
}

// NewQueueBroadcaster creates a new queue broadcaster.
func NewQueueBroadcaster(service *Service, hub Broadcaster, logger zerolog.Logger) *QueueBroadcaster {
	return &QueueBroadcaster{
		service: service,
		hub:     hub,
		logger:  logger.With().Str("component", "queue-broadcaster").Logger(),
	}
}

// SetCompletionHandler sets the handler to be called when downloads complete.
// This enables immediate import triggering when downloads transition to completed state.
func (b *QueueBroadcaster) SetCompletionHandler(handler CompletionHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.completionHandler = handler
}

// Start begins the periodic queue broadcasting.
func (b *QueueBroadcaster) Start() {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return
	}
	b.running = true
	b.stopCh = make(chan struct{})
	b.stoppedCh = make(chan struct{})
	b.triggerCh = make(chan struct{}, 1) // Buffered to avoid blocking
	b.mu.Unlock()

	go b.run()
	b.logger.Info().
		Dur("activeInterval", activeInterval).
		Dur("idleInterval", idleInterval).
		Msg("Queue broadcaster started with adaptive polling")
}

// Stop stops the periodic queue broadcasting.
func (b *QueueBroadcaster) Stop() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	close(b.stopCh)
	b.mu.Unlock()

	<-b.stoppedCh
	b.logger.Info().Msg("Queue broadcaster stopped")
}

// Trigger causes an immediate broadcast and switches to fast polling.
// Call this when a download is added or resumed to ensure immediate UI update.
func (b *QueueBroadcaster) Trigger() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	// Non-blocking send - if channel is full, a trigger is already pending
	select {
	case b.triggerCh <- struct{}{}:
	default:
	}
}

func (b *QueueBroadcaster) run() {
	defer close(b.stoppedCh)

	// Broadcast immediately on start
	hasActive := b.broadcast()

	// Start with appropriate interval based on initial state
	interval := idleInterval
	if hasActive {
		interval = activeInterval
	}
	b.setActiveMode(interval == activeInterval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			return
		case <-b.triggerCh:
			// Triggered externally (e.g., download added) - broadcast immediately
			b.broadcast()
			// Switch to fast polling since we likely have active downloads
			if interval != activeInterval {
				interval = activeInterval
				ticker.Reset(interval)
				b.setActiveMode(true)
			}
		case <-ticker.C:
			hasActive := b.broadcast()

			// Adjust polling rate based on queue activity
			newInterval := idleInterval
			if hasActive {
				newInterval = activeInterval
			}
			if newInterval != interval {
				interval = newInterval
				ticker.Reset(interval)
				b.setActiveMode(interval == activeInterval)
			}
		}
	}
}

func (b *QueueBroadcaster) setActiveMode(active bool) {
	b.mu.Lock()
	b.activeMode = active
	b.mu.Unlock()
}

// broadcast fetches the queue and broadcasts it. Returns true if there are active downloads.
func (b *QueueBroadcaster) broadcast() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queue, err := b.service.GetQueue(ctx)
	if err != nil {
		b.logger.Warn().Err(err).Msg("Failed to get queue for broadcast")
		return false
	}

	if err := b.hub.Broadcast("queue:state", queue); err != nil {
		b.logger.Warn().Err(err).Msg("Failed to broadcast queue state")
	}

	// Check for completed downloads and trigger import processing
	b.checkForCompletions(ctx)

	// Check for downloads that disappeared from clients
	if err := b.service.CheckForDisappearedDownloads(ctx); err != nil {
		b.logger.Debug().Err(err).Msg("Failed to check for disappeared downloads")
	}

	// Check if any downloads are actively progressing
	for _, item := range queue {
		if item.Status == "downloading" || item.Status == "queued" {
			return true
		}
	}
	return false
}

// checkForCompletions checks for completed downloads and triggers import processing.
// This enables immediate import when downloads complete, rather than waiting for the
// scheduled import scan (which runs every 5 minutes).
func (b *QueueBroadcaster) checkForCompletions(ctx context.Context) {
	b.mu.Lock()
	handler := b.completionHandler
	alreadyProcessing := b.processingImports
	b.mu.Unlock()

	if handler == nil {
		return
	}

	// Skip if we're already processing imports from a previous poll cycle
	// This prevents spawning multiple goroutines for the same completed downloads
	if alreadyProcessing {
		return
	}

	// Check for completed downloads using the downloader service
	completed, err := b.service.CheckForCompletedDownloads(ctx)
	if err != nil {
		b.logger.Debug().Err(err).Msg("Failed to check for completed downloads")
		return
	}

	if len(completed) == 0 {
		return
	}

	// Mark that we're processing imports
	b.mu.Lock()
	b.processingImports = true
	b.mu.Unlock()

	b.logger.Info().Int("count", len(completed)).Msg("Detected completed downloads, triggering import")

	// Trigger import processing asynchronously to not block the broadcast loop
	go func() {
		defer func() {
			b.mu.Lock()
			b.processingImports = false
			b.mu.Unlock()
		}()

		importCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := handler.CheckAndProcessCompletedDownloads(importCtx); err != nil {
			b.logger.Warn().Err(err).Msg("Failed to process completed downloads")
		}
	}()
}
