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

// QueueBroadcaster periodically polls the download queue and broadcasts updates via WebSocket.
// Uses adaptive polling: fast when downloads are active, slow when idle.
type QueueBroadcaster struct {
	service    *Service
	hub        Broadcaster
	logger     zerolog.Logger
	stopCh     chan struct{}
	stoppedCh  chan struct{}
	triggerCh  chan struct{}
	mu         sync.Mutex
	running    bool
	activeMode bool // true when polling at activeInterval
}

// NewQueueBroadcaster creates a new queue broadcaster.
func NewQueueBroadcaster(service *Service, hub Broadcaster, logger zerolog.Logger) *QueueBroadcaster {
	return &QueueBroadcaster{
		service: service,
		hub:     hub,
		logger:  logger.With().Str("component", "queue-broadcaster").Logger(),
	}
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

	// Check if any downloads are actively progressing
	for _, item := range queue {
		if item.Status == "downloading" || item.Status == "queued" {
			return true
		}
	}
	return false
}
