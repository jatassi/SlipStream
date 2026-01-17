package status

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

var (
	ErrStatusNotFound = errors.New("indexer status not found")
)

// HealthService is the interface for central health tracking.
type HealthService interface {
	RegisterItemStr(category, id, name string)
	UnregisterItemStr(category, id string)
	SetErrorStr(category, id, message string)
	SetWarningStr(category, id, message string)
	ClearStatusStr(category, id string)
}

// Service tracks indexer health and status.
type Service struct {
	queries       *sqlc.Queries
	config        BackoffConfig
	logger        zerolog.Logger
	healthService HealthService
}

// NewService creates a new status service with default configuration.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return NewServiceWithConfig(db, DefaultBackoffConfig(), logger)
}

// NewServiceWithConfig creates a new status service with custom configuration.
func NewServiceWithConfig(db *sql.DB, config BackoffConfig, logger zerolog.Logger) *Service {
	return &Service{
		queries: sqlc.New(db),
		config:  config,
		logger:  logger.With().Str("component", "indexer-status").Logger(),
	}
}

// SetHealthService sets the central health service for forwarding status updates.
func (s *Service) SetHealthService(hs HealthService) {
	s.healthService = hs
}

// SetDB updates the database connection used by this service.
func (s *Service) SetDB(db *sql.DB) {
	s.queries = sqlc.New(db)
}

// GetConfig returns the current backoff configuration.
func (s *Service) GetConfig() BackoffConfig {
	return s.config
}

// updateHealthStatus forwards status changes to the central health service.
func (s *Service) updateHealthStatus(indexerID int64, indexerName string, escalationLevel int, isDisabled bool, message string) {
	if s.healthService == nil {
		return
	}

	idStr := fmt.Sprintf("%d", indexerID)
	const category = "indexers"

	if isDisabled {
		s.healthService.SetErrorStr(category, idStr, message)
	} else if escalationLevel > 0 {
		s.healthService.SetWarningStr(category, idStr, message)
	} else {
		s.healthService.ClearStatusStr(category, idStr)
	}
}

// GetStatus retrieves the current status for an indexer.
func (s *Service) GetStatus(ctx context.Context, indexerID int64) (*IndexerStatus, error) {
	row, err := s.queries.GetIndexerStatus(ctx, indexerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return a default status for indexers without status records
			return &IndexerStatus{
				IndexerID:       indexerID,
				EscalationLevel: 0,
				IsDisabled:      false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get indexer status: %w", err)
	}

	return s.rowToStatus(row), nil
}

// RecordSuccess records a successful operation and clears any failure state.
func (s *Service) RecordSuccess(ctx context.Context, indexerID int64) error {
	// Clear any existing failure state
	if err := s.queries.ClearIndexerFailure(ctx, indexerID); err != nil {
		return fmt.Errorf("failed to clear failure state: %w", err)
	}

	s.logger.Debug().
		Int64("indexerId", indexerID).
		Msg("Recorded successful indexer operation")

	// Forward to central health service
	s.updateHealthStatus(indexerID, "", 0, false, "")

	return nil
}

// RecordFailure records a failed operation with escalating backoff.
func (s *Service) RecordFailure(ctx context.Context, indexerID int64, opError error) error {
	now := time.Now()

	// Get current status
	status, err := s.GetStatus(ctx, indexerID)
	if err != nil {
		return err
	}

	// Calculate new escalation level (capped at max)
	newLevel := status.EscalationLevel + 1
	if newLevel > s.config.MaxEscalation {
		newLevel = s.config.MaxEscalation
	}

	// Calculate disabled until time using configurable backoff
	backoff := s.calculateBackoffDuration(newLevel)
	disabledTill := now.Add(backoff)

	// Set initial failure time if not set
	var initialFailure *time.Time
	if status.InitialFailure == nil {
		initialFailure = &now
	} else {
		initialFailure = status.InitialFailure
	}

	// Update status
	_, err = s.queries.UpsertIndexerStatus(ctx, sqlc.UpsertIndexerStatusParams{
		IndexerID:         indexerID,
		InitialFailure:    toNullTime(initialFailure),
		MostRecentFailure: toNullTime(&now),
		EscalationLevel:   sql.NullInt64{Int64: int64(newLevel), Valid: true},
		DisabledTill:      toNullTime(&disabledTill),
		LastRssSync:       sql.NullTime{}, // Don't update RSS sync time
	})
	if err != nil {
		return fmt.Errorf("failed to record failure: %w", err)
	}

	s.logger.Warn().
		Int64("indexerId", indexerID).
		Int("escalationLevel", newLevel).
		Dur("backoff", backoff).
		Time("disabledTill", disabledTill).
		Err(opError).
		Msg("Recorded indexer failure, applying backoff")

	// Forward to central health service
	// Determine if this is now disabled (backoff > 0 means it's temporarily disabled)
	isDisabled := backoff > 0 && newLevel >= s.config.MaxEscalation
	message := opError.Error()
	if isDisabled {
		message = fmt.Sprintf("Disabled due to repeated failures: %s", opError.Error())
	} else if newLevel > 0 {
		message = fmt.Sprintf("Experienced %d failure(s): %s", newLevel, opError.Error())
	}
	s.updateHealthStatus(indexerID, "", newLevel, isDisabled, message)

	return nil
}

// IsDisabled checks if an indexer is temporarily disabled.
func (s *Service) IsDisabled(ctx context.Context, indexerID int64) (bool, *time.Time, error) {
	status, err := s.GetStatus(ctx, indexerID)
	if err != nil {
		return false, nil, err
	}

	if status.DisabledTill == nil {
		return false, nil, nil
	}

	if time.Now().After(*status.DisabledTill) {
		// Backoff period has passed
		return false, nil, nil
	}

	return true, status.DisabledTill, nil
}

// RecordRSSSync records an RSS sync operation.
func (s *Service) RecordRSSSync(ctx context.Context, indexerID int64) error {
	if err := s.queries.UpdateIndexerLastRssSync(ctx, indexerID); err != nil {
		return fmt.Errorf("failed to record RSS sync: %w", err)
	}
	return nil
}

// GetHealth returns the health summary for an indexer.
func (s *Service) GetHealth(ctx context.Context, indexerID int64, indexerName string) (*IndexerHealth, error) {
	status, err := s.GetStatus(ctx, indexerID)
	if err != nil {
		return nil, err
	}

	health := &IndexerHealth{
		IndexerID:   indexerID,
		IndexerName: indexerName,
		LastFailure: status.MostRecentFailure,
	}

	// Determine health status
	if status.DisabledTill != nil && time.Now().Before(*status.DisabledTill) {
		health.Status = HealthStatusDisabled
		remaining := time.Until(*status.DisabledTill)
		health.DisabledFor = &Duration{remaining}
		health.Message = fmt.Sprintf("Disabled for %s due to repeated failures", remaining.Round(time.Minute))
	} else if status.EscalationLevel > 0 {
		health.Status = HealthStatusWarning
		health.Message = fmt.Sprintf("Experienced %d recent failure(s)", status.EscalationLevel)
	} else {
		health.Status = HealthStatusHealthy
		health.Message = "Operating normally"
	}

	return health, nil
}

// ClearStatus clears all status information for an indexer.
func (s *Service) ClearStatus(ctx context.Context, indexerID int64) error {
	if err := s.queries.ClearIndexerFailure(ctx, indexerID); err != nil {
		return fmt.Errorf("failed to clear status: %w", err)
	}

	s.logger.Info().
		Int64("indexerId", indexerID).
		Msg("Cleared indexer status")

	return nil
}

// GetAllStatuses returns status for all indexers with status records.
func (s *Service) GetAllStatuses(ctx context.Context) ([]*IndexerStatus, error) {
	// Get disabled indexers
	rows, err := s.queries.ListDisabledIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list disabled indexers: %w", err)
	}

	statuses := make([]*IndexerStatus, 0, len(rows))
	for _, row := range rows {
		status := &IndexerStatus{
			IndexerID:  row.ID,
			IsDisabled: row.DisabledTill.Valid && row.DisabledTill.Time.After(time.Now()),
		}
		if row.DisabledTill.Valid {
			status.DisabledTill = &row.DisabledTill.Time
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetStats returns statistics about indexer statuses.
func (s *Service) GetStats(ctx context.Context, totalIndexers int) (*StatusStats, error) {
	disabled, err := s.queries.ListDisabledIndexers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get disabled indexers: %w", err)
	}

	disabledCount := 0
	for _, d := range disabled {
		if d.DisabledTill.Valid && d.DisabledTill.Time.After(time.Now()) {
			disabledCount++
		}
	}

	return &StatusStats{
		TotalIndexers:    totalIndexers,
		HealthyIndexers:  totalIndexers - disabledCount,
		WarningIndexers:  0, // Would need more complex query
		DisabledIndexers: disabledCount,
	}, nil
}

// rowToStatus converts a database row to IndexerStatus.
func (s *Service) rowToStatus(row *sqlc.IndexerStatus) *IndexerStatus {
	status := &IndexerStatus{
		IndexerID:       row.IndexerID,
		EscalationLevel: int(row.EscalationLevel.Int64),
	}

	if row.InitialFailure.Valid {
		status.InitialFailure = &row.InitialFailure.Time
	}
	if row.MostRecentFailure.Valid {
		status.MostRecentFailure = &row.MostRecentFailure.Time
	}
	if row.DisabledTill.Valid {
		status.DisabledTill = &row.DisabledTill.Time
		status.IsDisabled = row.DisabledTill.Time.After(time.Now())
	}
	if row.LastRssSync.Valid {
		status.LastRSSSync = &row.LastRssSync.Time
	}

	return status
}

// calculateBackoffDuration calculates the backoff duration for a given escalation level.
func (s *Service) calculateBackoffDuration(level int) time.Duration {
	if level <= 0 {
		return 0
	}

	// Use exponential backoff with configured multiplier
	backoff := s.config.InitialBackoff
	for i := 1; i < level; i++ {
		backoff = time.Duration(float64(backoff) * s.config.Multiplier)
		if backoff > s.config.MaxBackoff {
			return s.config.MaxBackoff
		}
	}

	return backoff
}

// GetCookies retrieves cached session cookies for an indexer.
func (s *Service) GetCookies(ctx context.Context, indexerID int64) (*CookieData, error) {
	row, err := s.queries.GetIndexerCookies(ctx, indexerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // No cookies cached
		}
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}

	// Check if cookies exist and are valid
	if !row.Cookies.Valid || row.Cookies.String == "" {
		return nil, nil
	}

	// Check expiration
	if row.CookiesExpiration.Valid && time.Now().After(row.CookiesExpiration.Time) {
		s.logger.Debug().Int64("indexerId", indexerID).Msg("Cached cookies have expired")
		return nil, nil
	}

	data := &CookieData{
		Cookies: row.Cookies.String,
	}
	if row.CookiesExpiration.Valid {
		data.ExpiresAt = &row.CookiesExpiration.Time
	}

	s.logger.Debug().
		Int64("indexerId", indexerID).
		Msg("Retrieved cached cookies")

	return data, nil
}

// SaveCookies stores session cookies for an indexer.
func (s *Service) SaveCookies(ctx context.Context, indexerID int64, cookies string, expiresAt *time.Time) error {
	err := s.queries.UpdateIndexerCookies(ctx, sqlc.UpdateIndexerCookiesParams{
		IndexerID:         indexerID,
		Cookies:           toNullString(cookies),
		CookiesExpiration: toNullTime(expiresAt),
	})
	if err != nil {
		return fmt.Errorf("failed to save cookies: %w", err)
	}

	s.logger.Debug().
		Int64("indexerId", indexerID).
		Msg("Saved session cookies")

	return nil
}

// ClearCookies removes cached cookies for an indexer.
func (s *Service) ClearCookies(ctx context.Context, indexerID int64) error {
	if err := s.queries.ClearIndexerCookies(ctx, indexerID); err != nil {
		return fmt.Errorf("failed to clear cookies: %w", err)
	}

	s.logger.Debug().
		Int64("indexerId", indexerID).
		Msg("Cleared session cookies")

	return nil
}

// Helper functions

func toNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
