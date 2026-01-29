package mock

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/notification/types"
)

// NotificationRecord stores a sent notification for debugging/preview
type NotificationRecord struct {
	ID        int64     `json:"id"`
	EventType string    `json:"eventType"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Data      any       `json:"data,omitempty"`
	SentAt    time.Time `json:"sentAt"`
}

// Notifier is a mock notification provider for dev mode testing.
// It logs all notifications and stores them in memory for UI preview.
type Notifier struct {
	name   string
	logger zerolog.Logger

	mu           sync.RWMutex
	records      []NotificationRecord
	nextID       int64
	maxRecords   int
	broadcaster  Broadcaster
}

// Broadcaster interface for sending WebSocket events
type Broadcaster interface {
	Broadcast(eventType string, data any)
}

// New creates a new mock notifier
func New(name string, logger zerolog.Logger) *Notifier {
	return &Notifier{
		name:       name,
		logger:     logger.With().Str("notifier", "mock").Str("name", name).Logger(),
		records:    make([]NotificationRecord, 0),
		nextID:     1,
		maxRecords: 100,
	}
}

// SetBroadcaster sets the WebSocket broadcaster for real-time updates
func (n *Notifier) SetBroadcaster(b Broadcaster) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.broadcaster = b
}

func (n *Notifier) Type() types.NotifierType {
	return types.NotifierMock
}

func (n *Notifier) Name() string {
	return n.name
}

func (n *Notifier) Test(ctx context.Context) error {
	n.recordNotification("test", "Test Notification", "This is a test notification from the mock notifier", nil)
	return nil
}

func (n *Notifier) OnGrab(ctx context.Context, event types.GrabEvent) error {
	title := "Release Grabbed"
	var message string
	if event.Movie != nil {
		message = event.Movie.Title
		if event.Movie.Year > 0 {
			message += " (" + string(rune(event.Movie.Year)) + ")"
		}
	} else if event.Episode != nil {
		message = event.Episode.SeriesTitle + " S" + padZero(event.Episode.SeasonNumber) + "E" + padZero(event.Episode.EpisodeNumber)
	}
	message += " - " + event.Release.Quality

	n.recordNotification("grab", title, message, event)
	return nil
}

func (n *Notifier) OnImport(ctx context.Context, event types.ImportEvent) error {
	title := "Download Imported"
	var message string
	if event.Movie != nil {
		message = event.Movie.Title
	} else if event.Episode != nil {
		message = event.Episode.SeriesTitle + " S" + padZero(event.Episode.SeasonNumber) + "E" + padZero(event.Episode.EpisodeNumber)
	}
	message += " - " + event.Quality

	n.recordNotification("download", title, message, event)
	return nil
}

func (n *Notifier) OnUpgrade(ctx context.Context, event types.UpgradeEvent) error {
	title := "Quality Upgraded"
	var message string
	if event.Movie != nil {
		message = event.Movie.Title
	} else if event.Episode != nil {
		message = event.Episode.SeriesTitle + " S" + padZero(event.Episode.SeasonNumber) + "E" + padZero(event.Episode.EpisodeNumber)
	}
	message += " - " + event.OldQuality + " -> " + event.NewQuality

	n.recordNotification("upgrade", title, message, event)
	return nil
}

func (n *Notifier) OnMovieAdded(ctx context.Context, event types.MovieAddedEvent) error {
	title := "Movie Added"
	message := event.Movie.Title
	if event.Movie.Year > 0 {
		message += " (" + itoa(event.Movie.Year) + ")"
	}

	n.recordNotification("movie_added", title, message, event)
	return nil
}

func (n *Notifier) OnMovieDeleted(ctx context.Context, event types.MovieDeletedEvent) error {
	title := "Movie Deleted"
	message := event.Movie.Title
	if event.DeletedFiles {
		message += " (files deleted)"
	}

	n.recordNotification("movie_deleted", title, message, event)
	return nil
}

func (n *Notifier) OnSeriesAdded(ctx context.Context, event types.SeriesAddedEvent) error {
	title := "Series Added"
	message := event.Series.Title
	if event.Series.Year > 0 {
		message += " (" + itoa(event.Series.Year) + ")"
	}

	n.recordNotification("series_added", title, message, event)
	return nil
}

func (n *Notifier) OnSeriesDeleted(ctx context.Context, event types.SeriesDeletedEvent) error {
	title := "Series Deleted"
	message := event.Series.Title
	if event.DeletedFiles {
		message += " (files deleted)"
	}

	n.recordNotification("series_deleted", title, message, event)
	return nil
}

func (n *Notifier) OnHealthIssue(ctx context.Context, event types.HealthEvent) error {
	title := "Health Issue: " + event.Source
	message := event.Message

	n.recordNotification("health_issue", title, message, event)
	return nil
}

func (n *Notifier) OnHealthRestored(ctx context.Context, event types.HealthEvent) error {
	title := "Health Restored: " + event.Source
	message := event.Message

	n.recordNotification("health_restored", title, message, event)
	return nil
}

func (n *Notifier) OnApplicationUpdate(ctx context.Context, event types.AppUpdateEvent) error {
	title := "Application Updated"
	message := event.PreviousVersion + " -> " + event.NewVersion

	n.recordNotification("app_update", title, message, event)
	return nil
}

func (n *Notifier) SendMessage(ctx context.Context, event types.MessageEvent) error {
	n.recordNotification("message", event.Title, event.Message, event)
	return nil
}

// GetRecords returns all stored notification records
func (n *Notifier) GetRecords() []NotificationRecord {
	n.mu.RLock()
	defer n.mu.RUnlock()

	records := make([]NotificationRecord, len(n.records))
	copy(records, n.records)
	return records
}

// Clear removes all stored notification records
func (n *Notifier) Clear() {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.records = make([]NotificationRecord, 0)
	n.nextID = 1
}

func (n *Notifier) recordNotification(eventType, title, message string, data any) {
	n.mu.Lock()
	record := NotificationRecord{
		ID:        n.nextID,
		EventType: eventType,
		Title:     title,
		Message:   message,
		Data:      data,
		SentAt:    time.Now(),
	}
	n.nextID++

	// Trim old records if we exceed max
	if len(n.records) >= n.maxRecords {
		n.records = n.records[1:]
	}
	n.records = append(n.records, record)

	broadcaster := n.broadcaster
	n.mu.Unlock()

	// Log the notification
	n.logger.Info().
		Str("eventType", eventType).
		Str("title", title).
		Str("message", message).
		Msg("Mock notification sent")

	// Broadcast to connected clients
	if broadcaster != nil {
		broadcaster.Broadcast("notification:mock", record)
	}
}

func padZero(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	negative := n < 0
	if negative {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
