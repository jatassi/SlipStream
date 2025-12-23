// Package progress provides a standardized way to broadcast progress events
// to connected WebSocket clients. This package is designed for reuse across
// all features that need to report progress (scanning, downloading, importing, etc.).
package progress

import (
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/websocket"
)

// ActivityType identifies the type of activity being tracked.
type ActivityType string

const (
	ActivityTypeScan           ActivityType = "scan"
	ActivityTypeDownload       ActivityType = "download"
	ActivityTypeImport         ActivityType = "import"
	ActivityTypeMetadataRefresh ActivityType = "metadata-refresh"
	ActivityTypeFileOperation  ActivityType = "file-operation"
)

// Status represents the current state of an activity.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
)

// Activity represents a trackable activity with progress.
type Activity struct {
	ID           string                 `json:"id"`           // Unique identifier
	Type         ActivityType           `json:"type"`         // Activity type for categorization
	Title        string                 `json:"title"`        // Human-readable title
	Subtitle     string                 `json:"subtitle"`     // Current phase/status description
	Progress     int                    `json:"progress"`     // 0-100, -1 for indeterminate
	Status       Status                 `json:"status"`       // Current status
	StartedAt    time.Time              `json:"startedAt"`    // When activity started
	CompletedAt  *time.Time             `json:"completedAt"`  // When activity completed (nil if ongoing)
	Metadata     map[string]interface{} `json:"metadata"`     // Activity-specific data
}

// EventType identifies the type of progress event.
type EventType string

const (
	EventTypeStarted   EventType = "progress:started"
	EventTypeUpdate    EventType = "progress:update"
	EventTypeCompleted EventType = "progress:completed"
	EventTypeError     EventType = "progress:error"
	EventTypeCancelled EventType = "progress:cancelled"
)

// Manager tracks and broadcasts progress for all activities.
type Manager struct {
	hub        *websocket.Hub
	activities map[string]*Activity
	mu         sync.RWMutex
	logger     zerolog.Logger
}

// NewManager creates a new progress manager.
func NewManager(hub *websocket.Hub, logger zerolog.Logger) *Manager {
	return &Manager{
		hub:        hub,
		activities: make(map[string]*Activity),
		logger:     logger.With().Str("component", "progress").Logger(),
	}
}

// StartActivity creates and starts tracking a new activity.
func (m *Manager) StartActivity(id string, activityType ActivityType, title string) *Activity {
	m.mu.Lock()
	defer m.mu.Unlock()

	activity := &Activity{
		ID:        id,
		Type:      activityType,
		Title:     title,
		Subtitle:  "Starting...",
		Progress:  0,
		Status:    StatusInProgress,
		StartedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	m.activities[id] = activity
	m.broadcast(EventTypeStarted, activity)

	m.logger.Debug().
		Str("id", id).
		Str("type", string(activityType)).
		Str("title", title).
		Msg("Activity started")

	return activity
}

// UpdateActivity updates an existing activity's progress.
func (m *Manager) UpdateActivity(id string, subtitle string, progress int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	activity, exists := m.activities[id]
	if !exists {
		return
	}

	activity.Subtitle = subtitle
	activity.Progress = progress

	m.broadcast(EventTypeUpdate, activity)
}

// UpdateActivityMetadata updates an activity's metadata.
func (m *Manager) UpdateActivityMetadata(id string, key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	activity, exists := m.activities[id]
	if !exists {
		return
	}

	activity.Metadata[key] = value
}

// CompleteActivity marks an activity as completed.
func (m *Manager) CompleteActivity(id string, subtitle string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	activity, exists := m.activities[id]
	if !exists {
		return
	}

	now := time.Now()
	activity.Status = StatusCompleted
	activity.Progress = 100
	activity.Subtitle = subtitle
	activity.CompletedAt = &now

	m.broadcast(EventTypeCompleted, activity)

	// Remove from active tracking after a short delay
	// (frontend will handle display timeout)
	go func() {
		time.Sleep(5 * time.Second)
		m.mu.Lock()
		delete(m.activities, id)
		m.mu.Unlock()
	}()

	m.logger.Debug().
		Str("id", id).
		Str("title", activity.Title).
		Msg("Activity completed")
}

// FailActivity marks an activity as failed.
func (m *Manager) FailActivity(id string, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	activity, exists := m.activities[id]
	if !exists {
		return
	}

	now := time.Now()
	activity.Status = StatusFailed
	activity.Subtitle = errorMsg
	activity.CompletedAt = &now
	activity.Metadata["error"] = errorMsg

	m.broadcast(EventTypeError, activity)

	// Remove from active tracking after a delay
	go func() {
		time.Sleep(10 * time.Second)
		m.mu.Lock()
		delete(m.activities, id)
		m.mu.Unlock()
	}()

	m.logger.Debug().
		Str("id", id).
		Str("title", activity.Title).
		Str("error", errorMsg).
		Msg("Activity failed")
}

// CancelActivity marks an activity as cancelled.
func (m *Manager) CancelActivity(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	activity, exists := m.activities[id]
	if !exists {
		return
	}

	now := time.Now()
	activity.Status = StatusCancelled
	activity.Subtitle = "Cancelled"
	activity.CompletedAt = &now

	m.broadcast(EventTypeCancelled, activity)

	delete(m.activities, id)

	m.logger.Debug().
		Str("id", id).
		Str("title", activity.Title).
		Msg("Activity cancelled")
}

// GetActivity returns an activity by ID.
func (m *Manager) GetActivity(id string) *Activity {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activities[id]
}

// GetAllActivities returns all active activities.
func (m *Manager) GetAllActivities() []*Activity {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Activity, 0, len(m.activities))
	for _, activity := range m.activities {
		result = append(result, activity)
	}
	return result
}

// GetActivitiesByType returns all activities of a specific type.
func (m *Manager) GetActivitiesByType(activityType ActivityType) []*Activity {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Activity, 0)
	for _, activity := range m.activities {
		if activity.Type == activityType {
			result = append(result, activity)
		}
	}
	return result
}

// broadcast sends an activity update to all connected clients.
func (m *Manager) broadcast(eventType EventType, activity *Activity) {
	if m.hub == nil {
		return
	}

	m.hub.Broadcast(string(eventType), activity)
}

// ActivityBuilder provides a fluent interface for creating and managing activities.
type ActivityBuilder struct {
	manager  *Manager
	activity *Activity
}

// NewActivityBuilder creates a new activity builder.
func (m *Manager) NewActivityBuilder(id string, activityType ActivityType, title string) *ActivityBuilder {
	activity := m.StartActivity(id, activityType, title)
	return &ActivityBuilder{
		manager:  m,
		activity: activity,
	}
}

// SetSubtitle updates the activity's subtitle.
func (b *ActivityBuilder) SetSubtitle(subtitle string) *ActivityBuilder {
	b.manager.UpdateActivity(b.activity.ID, subtitle, b.activity.Progress)
	return b
}

// SetProgress updates the activity's progress.
func (b *ActivityBuilder) SetProgress(progress int) *ActivityBuilder {
	b.manager.UpdateActivity(b.activity.ID, b.activity.Subtitle, progress)
	return b
}

// Update updates both subtitle and progress.
func (b *ActivityBuilder) Update(subtitle string, progress int) *ActivityBuilder {
	b.manager.UpdateActivity(b.activity.ID, subtitle, progress)
	return b
}

// SetMetadata adds metadata to the activity.
func (b *ActivityBuilder) SetMetadata(key string, value interface{}) *ActivityBuilder {
	b.manager.UpdateActivityMetadata(b.activity.ID, key, value)
	return b
}

// Complete marks the activity as completed.
func (b *ActivityBuilder) Complete(subtitle string) {
	b.manager.CompleteActivity(b.activity.ID, subtitle)
}

// Fail marks the activity as failed.
func (b *ActivityBuilder) Fail(errorMsg string) {
	b.manager.FailActivity(b.activity.ID, errorMsg)
}

// Cancel marks the activity as cancelled.
func (b *ActivityBuilder) Cancel() {
	b.manager.CancelActivity(b.activity.ID)
}

// ID returns the activity's ID.
func (b *ActivityBuilder) ID() string {
	return b.activity.ID
}
