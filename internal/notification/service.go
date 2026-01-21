package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrInvalidSettings      = errors.New("invalid notification settings")
)

// Backoff configuration
const (
	minBackoffDuration  = 5 * time.Minute
	maxEscalationLevel  = 5
)

// Service orchestrates notification sending and management
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	factory *Factory
	logger  zerolog.Logger
	mu      sync.RWMutex
}

// NewService creates a new notification service
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		factory: NewFactory(logger),
		logger:  logger.With().Str("component", "notification").Logger(),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.db = db
	s.queries = sqlc.New(db)
}

// List returns all configured notifications
func (s *Service) List(ctx context.Context) ([]Config, error) {
	rows, err := s.queries.ListNotifications(ctx)
	if err != nil {
		return nil, err
	}
	return s.rowsToConfigs(rows), nil
}

// Get returns a notification by ID
func (s *Service) Get(ctx context.Context, id int64) (*Config, error) {
	row, err := s.queries.GetNotification(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotificationNotFound
		}
		return nil, err
	}
	cfg := s.rowToConfig(row)
	return &cfg, nil
}

// Create creates a new notification
func (s *Service) Create(ctx context.Context, input CreateInput) (*Config, error) {
	if _, ok := GetSchema(input.Type); !ok {
		return nil, errors.New("unsupported notification type")
	}

	settings := input.Settings
	if settings == nil {
		settings = json.RawMessage("{}")
	}

	tags, _ := json.Marshal(input.Tags)

	row, err := s.queries.CreateNotification(ctx, sqlc.CreateNotificationParams{
		Name:                  input.Name,
		Type:                  string(input.Type),
		Enabled:               boolToInt(input.Enabled),
		Settings:              string(settings),
		OnGrab:                boolToInt(input.OnGrab),
		OnDownload:            boolToInt(input.OnDownload),
		OnUpgrade:             boolToInt(input.OnUpgrade),
		OnMovieAdded:          boolToInt(input.OnMovieAdded),
		OnMovieDeleted:        boolToInt(input.OnMovieDeleted),
		OnSeriesAdded:         boolToInt(input.OnSeriesAdded),
		OnSeriesDeleted:       boolToInt(input.OnSeriesDeleted),
		OnHealthIssue:         boolToInt(input.OnHealthIssue),
		OnHealthRestored:      boolToInt(input.OnHealthRestored),
		OnAppUpdate:           boolToInt(input.OnAppUpdate),
		IncludeHealthWarnings: boolToInt(input.IncludeHealthWarnings),
		Tags:                  string(tags),
	})
	if err != nil {
		return nil, err
	}

	cfg := s.rowToConfig(row)
	return &cfg, nil
}

// Update updates an existing notification
func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Config, error) {
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	name := existing.Name
	if input.Name != nil {
		name = *input.Name
	}

	notifType := existing.Type
	if input.Type != nil {
		notifType = *input.Type
	}

	enabled := existing.Enabled
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	settings := existing.Settings
	if input.Settings != nil {
		settings = *input.Settings
	}

	onGrab := existing.OnGrab
	if input.OnGrab != nil {
		onGrab = *input.OnGrab
	}

	onDownload := existing.OnDownload
	if input.OnDownload != nil {
		onDownload = *input.OnDownload
	}

	onUpgrade := existing.OnUpgrade
	if input.OnUpgrade != nil {
		onUpgrade = *input.OnUpgrade
	}

	onMovieAdded := existing.OnMovieAdded
	if input.OnMovieAdded != nil {
		onMovieAdded = *input.OnMovieAdded
	}

	onMovieDeleted := existing.OnMovieDeleted
	if input.OnMovieDeleted != nil {
		onMovieDeleted = *input.OnMovieDeleted
	}

	onSeriesAdded := existing.OnSeriesAdded
	if input.OnSeriesAdded != nil {
		onSeriesAdded = *input.OnSeriesAdded
	}

	onSeriesDeleted := existing.OnSeriesDeleted
	if input.OnSeriesDeleted != nil {
		onSeriesDeleted = *input.OnSeriesDeleted
	}

	onHealthIssue := existing.OnHealthIssue
	if input.OnHealthIssue != nil {
		onHealthIssue = *input.OnHealthIssue
	}

	onHealthRestored := existing.OnHealthRestored
	if input.OnHealthRestored != nil {
		onHealthRestored = *input.OnHealthRestored
	}

	onAppUpdate := existing.OnAppUpdate
	if input.OnAppUpdate != nil {
		onAppUpdate = *input.OnAppUpdate
	}

	includeHealthWarnings := existing.IncludeHealthWarnings
	if input.IncludeHealthWarnings != nil {
		includeHealthWarnings = *input.IncludeHealthWarnings
	}

	tags := existing.Tags
	if input.Tags != nil {
		tags = *input.Tags
	}

	tagsJSON, _ := json.Marshal(tags)

	row, err := s.queries.UpdateNotification(ctx, sqlc.UpdateNotificationParams{
		Name:                  name,
		Type:                  string(notifType),
		Enabled:               boolToInt(enabled),
		Settings:              string(settings),
		OnGrab:                boolToInt(onGrab),
		OnDownload:            boolToInt(onDownload),
		OnUpgrade:             boolToInt(onUpgrade),
		OnMovieAdded:          boolToInt(onMovieAdded),
		OnMovieDeleted:        boolToInt(onMovieDeleted),
		OnSeriesAdded:         boolToInt(onSeriesAdded),
		OnSeriesDeleted:       boolToInt(onSeriesDeleted),
		OnHealthIssue:         boolToInt(onHealthIssue),
		OnHealthRestored:      boolToInt(onHealthRestored),
		OnAppUpdate:           boolToInt(onAppUpdate),
		IncludeHealthWarnings: boolToInt(includeHealthWarnings),
		Tags:                  string(tagsJSON),
		ID:                    id,
	})
	if err != nil {
		return nil, err
	}

	cfg := s.rowToConfig(row)
	return &cfg, nil
}

// Delete deletes a notification
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.queries.DeleteNotification(ctx, id)
}

// Test tests a notification configuration
func (s *Service) Test(ctx context.Context, id int64) (*TestResult, error) {
	cfg, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	notifier, err := s.factory.Create(*cfg)
	if err != nil {
		return &TestResult{Success: false, Message: err.Error()}, nil
	}

	if err := notifier.Test(ctx); err != nil {
		return &TestResult{Success: false, Message: err.Error()}, nil
	}

	return &TestResult{Success: true, Message: "Notification test successful"}, nil
}

// TestConfig tests a notification configuration without saving
func (s *Service) TestConfig(ctx context.Context, input CreateInput) (*TestResult, error) {
	cfg := Config{
		Name:     input.Name,
		Type:     input.Type,
		Settings: input.Settings,
	}

	notifier, err := s.factory.Create(cfg)
	if err != nil {
		return &TestResult{Success: false, Message: err.Error()}, nil
	}

	if err := notifier.Test(ctx); err != nil {
		return &TestResult{Success: false, Message: err.Error()}, nil
	}

	return &TestResult{Success: true, Message: "Notification test successful"}, nil
}

// Dispatch sends an event to all enabled notifications that subscribe to it
func (s *Service) Dispatch(ctx context.Context, eventType EventType, event any) {
	configs, err := s.getEnabledConfigs(ctx, eventType)
	if err != nil {
		s.logger.Error().Err(err).Str("event", string(eventType)).Msg("Failed to get enabled notifications")
		return
	}

	if len(configs) == 0 {
		return
	}

	s.logger.Debug().
		Str("event", string(eventType)).
		Int("count", len(configs)).
		Msg("Dispatching notification event")

	for _, cfg := range configs {
		go s.sendNotification(ctx, cfg, eventType, event)
	}
}

func (s *Service) sendNotification(ctx context.Context, cfg Config, eventType EventType, event any) {
	notifier, err := s.factory.Create(cfg)
	if err != nil {
		s.logger.Error().Err(err).Str("name", cfg.Name).Msg("Failed to create notifier")
		return
	}

	var sendErr error
	switch eventType {
	case EventGrab:
		if e, ok := event.(GrabEvent); ok {
			sendErr = notifier.OnGrab(ctx, e)
		}
	case EventDownload:
		if e, ok := event.(DownloadEvent); ok {
			sendErr = notifier.OnDownload(ctx, e)
		}
	case EventUpgrade:
		if e, ok := event.(UpgradeEvent); ok {
			sendErr = notifier.OnUpgrade(ctx, e)
		}
	case EventMovieAdded:
		if e, ok := event.(MovieAddedEvent); ok {
			sendErr = notifier.OnMovieAdded(ctx, e)
		}
	case EventMovieDeleted:
		if e, ok := event.(MovieDeletedEvent); ok {
			sendErr = notifier.OnMovieDeleted(ctx, e)
		}
	case EventSeriesAdded:
		if e, ok := event.(SeriesAddedEvent); ok {
			sendErr = notifier.OnSeriesAdded(ctx, e)
		}
	case EventSeriesDeleted:
		if e, ok := event.(SeriesDeletedEvent); ok {
			sendErr = notifier.OnSeriesDeleted(ctx, e)
		}
	case EventHealthIssue:
		if e, ok := event.(HealthEvent); ok {
			sendErr = notifier.OnHealthIssue(ctx, e)
		}
	case EventHealthRestored:
		if e, ok := event.(HealthEvent); ok {
			sendErr = notifier.OnHealthRestored(ctx, e)
		}
	case EventAppUpdate:
		if e, ok := event.(AppUpdateEvent); ok {
			sendErr = notifier.OnApplicationUpdate(ctx, e)
		}
	}

	if sendErr != nil {
		s.logger.Error().
			Err(sendErr).
			Str("name", cfg.Name).
			Str("type", string(cfg.Type)).
			Str("event", string(eventType)).
			Msg("Notification failed")
		s.recordFailure(ctx, cfg.ID)
	} else {
		s.logger.Debug().
			Str("name", cfg.Name).
			Str("event", string(eventType)).
			Msg("Notification sent successfully")
		s.clearFailure(ctx, cfg.ID)
	}
}

func (s *Service) getEnabledConfigs(ctx context.Context, eventType EventType) ([]Config, error) {
	rows, err := s.queries.ListEnabledNotifications(ctx)
	if err != nil {
		return nil, err
	}

	var configs []Config
	for _, row := range rows {
		cfg := s.rowToConfig(row)

		if !s.configSubscribesToEvent(cfg, eventType) {
			continue
		}

		if s.isDisabled(ctx, cfg.ID) {
			continue
		}

		configs = append(configs, cfg)
	}

	return configs, nil
}

func (s *Service) configSubscribesToEvent(cfg Config, eventType EventType) bool {
	switch eventType {
	case EventGrab:
		return cfg.OnGrab
	case EventDownload:
		return cfg.OnDownload
	case EventUpgrade:
		return cfg.OnUpgrade
	case EventMovieAdded:
		return cfg.OnMovieAdded
	case EventMovieDeleted:
		return cfg.OnMovieDeleted
	case EventSeriesAdded:
		return cfg.OnSeriesAdded
	case EventSeriesDeleted:
		return cfg.OnSeriesDeleted
	case EventHealthIssue:
		return cfg.OnHealthIssue
	case EventHealthRestored:
		return cfg.OnHealthRestored
	case EventAppUpdate:
		return cfg.OnAppUpdate
	default:
		return false
	}
}

func (s *Service) isDisabled(ctx context.Context, id int64) bool {
	status, err := s.queries.GetNotificationStatus(ctx, id)
	if err != nil {
		return false
	}

	if status.DisabledTill.Valid && status.DisabledTill.Time.After(time.Now()) {
		return true
	}

	return false
}

func (s *Service) recordFailure(ctx context.Context, id int64) {
	now := time.Now()

	status, err := s.queries.GetNotificationStatus(ctx, id)
	if err != nil {
		// First failure
		s.queries.UpsertNotificationStatus(ctx, sqlc.UpsertNotificationStatusParams{
			NotificationID:    id,
			InitialFailure:    sql.NullTime{Time: now, Valid: true},
			MostRecentFailure: sql.NullTime{Time: now, Valid: true},
			EscalationLevel:   1,
			DisabledTill:      sql.NullTime{Time: now.Add(minBackoffDuration), Valid: true},
		})
		return
	}

	escalation := status.EscalationLevel + 1
	if escalation > maxEscalationLevel {
		escalation = maxEscalationLevel
	}

	backoff := minBackoffDuration * time.Duration(1<<(escalation-1))
	disabledTill := now.Add(backoff)

	s.queries.UpsertNotificationStatus(ctx, sqlc.UpsertNotificationStatusParams{
		NotificationID:    id,
		InitialFailure:    status.InitialFailure,
		MostRecentFailure: sql.NullTime{Time: now, Valid: true},
		EscalationLevel:   escalation,
		DisabledTill:      sql.NullTime{Time: disabledTill, Valid: true},
	})
}

func (s *Service) clearFailure(ctx context.Context, id int64) {
	s.queries.ClearNotificationStatus(ctx, id)
}

func (s *Service) rowToConfig(row *sqlc.Notification) Config {
	var tags []int64
	json.Unmarshal([]byte(row.Tags), &tags)

	return Config{
		ID:                    row.ID,
		Name:                  row.Name,
		Type:                  NotifierType(row.Type),
		Enabled:               row.Enabled == 1,
		Settings:              json.RawMessage(row.Settings),
		OnGrab:                row.OnGrab == 1,
		OnDownload:            row.OnDownload == 1,
		OnUpgrade:             row.OnUpgrade == 1,
		OnMovieAdded:          row.OnMovieAdded == 1,
		OnMovieDeleted:        row.OnMovieDeleted == 1,
		OnSeriesAdded:         row.OnSeriesAdded == 1,
		OnSeriesDeleted:       row.OnSeriesDeleted == 1,
		OnHealthIssue:         row.OnHealthIssue == 1,
		OnHealthRestored:      row.OnHealthRestored == 1,
		OnAppUpdate:           row.OnAppUpdate == 1,
		IncludeHealthWarnings: row.IncludeHealthWarnings == 1,
		Tags:                  tags,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

func (s *Service) rowsToConfigs(rows []*sqlc.Notification) []Config {
	configs := make([]Config, len(rows))
	for i, row := range rows {
		configs[i] = s.rowToConfig(row)
	}
	return configs
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// DispatchHealthIssue dispatches a health issue notification.
// Implements health.NotificationDispatcher interface.
func (s *Service) DispatchHealthIssue(ctx context.Context, source, healthType, message string) {
	event := HealthEvent{
		Source:    source,
		Type:      healthType,
		Message:   message,
		OccuredAt: time.Now(),
	}
	s.Dispatch(ctx, EventHealthIssue, event)
}

// DispatchHealthRestored dispatches a health restored notification.
// Implements health.NotificationDispatcher interface.
func (s *Service) DispatchHealthRestored(ctx context.Context, source, healthType, message string) {
	event := HealthEvent{
		Source:    source,
		Type:      healthType,
		Message:   message,
		OccuredAt: time.Now(),
	}
	s.Dispatch(ctx, EventHealthRestored, event)
}

// DispatchDownload dispatches a download completed notification.
func (s *Service) DispatchDownload(ctx context.Context, event DownloadEvent) {
	s.Dispatch(ctx, EventDownload, event)
}

// DispatchUpgrade dispatches an upgrade notification.
func (s *Service) DispatchUpgrade(ctx context.Context, event UpgradeEvent) {
	s.Dispatch(ctx, EventUpgrade, event)
}

// DispatchMovieAdded dispatches a movie added notification.
func (s *Service) DispatchMovieAdded(ctx context.Context, event MovieAddedEvent) {
	s.Dispatch(ctx, EventMovieAdded, event)
}

// DispatchMovieDeleted dispatches a movie deleted notification.
func (s *Service) DispatchMovieDeleted(ctx context.Context, event MovieDeletedEvent) {
	s.Dispatch(ctx, EventMovieDeleted, event)
}

// DispatchSeriesAdded dispatches a series added notification.
func (s *Service) DispatchSeriesAdded(ctx context.Context, event SeriesAddedEvent) {
	s.Dispatch(ctx, EventSeriesAdded, event)
}

// DispatchSeriesDeleted dispatches a series deleted notification.
func (s *Service) DispatchSeriesDeleted(ctx context.Context, event SeriesDeletedEvent) {
	s.Dispatch(ctx, EventSeriesDeleted, event)
}

// CreateNotifierFromConfig creates a notifier from type, name, and settings.
// This is used by portal notifications to create notifiers for user-configured channels.
func (s *Service) CreateNotifierFromConfig(notifType, name string, settings string) (Notifier, error) {
	cfg := Config{
		Type:     NotifierType(notifType),
		Name:     name,
		Settings: json.RawMessage(settings),
	}
	return s.factory.Create(cfg)
}

// DispatchGenericMessage sends a generic text message to all enabled notifications.
// This is used for admin notifications that don't fit a specific event type.
func (s *Service) DispatchGenericMessage(ctx context.Context, message string) {
	configs, err := s.List(ctx)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to list notifications for generic message")
		return
	}

	for _, cfg := range configs {
		if !cfg.Enabled {
			continue
		}

		go func(cfg Config) {
			notifier, err := s.factory.Create(cfg)
			if err != nil {
				s.logger.Warn().Err(err).Str("name", cfg.Name).Msg("failed to create notifier for generic message")
				return
			}

			// Use OnHealthIssue as a generic message channel since it's commonly enabled
			event := HealthEvent{
				Source:    "SlipStream",
				Type:      "info",
				Message:   message,
				OccuredAt: time.Now(),
			}
			if err := notifier.OnHealthIssue(ctx, event); err != nil {
				s.logger.Warn().Err(err).Str("name", cfg.Name).Msg("failed to send generic message")
			}
		}(cfg)
	}
}
