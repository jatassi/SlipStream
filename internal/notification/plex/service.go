package plex

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

const (
	maxPathsBeforeFullRefresh = 10
	refreshDelayBetweenCalls  = 1 * time.Second
)

// RefreshService manages the Plex library refresh queue
type RefreshService struct {
	queries *sqlc.Queries
	client  *Client
	logger  *zerolog.Logger
}

// NewRefreshService creates a new RefreshService
func NewRefreshService(queries *sqlc.Queries, client *Client, logger *zerolog.Logger) *RefreshService {
	subLogger := logger.With().Str("component", "plex-refresh-service").Logger()
	return &RefreshService{
		queries: queries,
		client:  client,
		logger:  &subLogger,
	}
}

// EnqueueRefresh adds a path to the refresh queue
func (s *RefreshService) EnqueueRefresh(ctx context.Context, notificationID int64, serverID string, sectionKey int, path string) error {
	var pathVal sql.NullString
	if path != "" {
		pathVal = sql.NullString{String: path, Valid: true}
	}

	_, err := s.queries.EnqueuePlexRefresh(ctx, sqlc.EnqueuePlexRefreshParams{
		NotificationID: notificationID,
		ServerID:       serverID,
		SectionKey:     int64(sectionKey),
		Path:           pathVal,
	})
	if err != nil {
		return err
	}

	s.logger.Debug().
		Int64("notificationId", notificationID).
		Str("serverId", serverID).
		Int("sectionKey", sectionKey).
		Str("path", path).
		Msg("Enqueued Plex refresh")

	return nil
}

// ProcessQueue processes all pending refresh items
func (s *RefreshService) ProcessQueue(ctx context.Context, getSettings func(notificationID int64) (*Settings, error), getServerURL func(ctx context.Context, settings *Settings) (string, error)) error {
	counts, err := s.queries.CountPendingPlexRefreshesPerSection(ctx)
	if err != nil {
		return err
	}

	if len(counts) == 0 {
		return nil
	}

	s.logger.Info().Int("sections", len(counts)).Msg("Processing Plex refresh queue")

	for _, count := range counts {
		s.processSection(ctx, *count, getSettings, getServerURL)
		time.Sleep(refreshDelayBetweenCalls)
	}

	return nil
}

func (s *RefreshService) processSection(ctx context.Context, count sqlc.CountPendingPlexRefreshesPerSectionRow, getSettings func(notificationID int64) (*Settings, error), getServerURL func(ctx context.Context, settings *Settings) (string, error)) {
	settings, err := getSettings(count.NotificationID)
	if err != nil {
		s.logger.Error().Err(err).Int64("notificationId", count.NotificationID).Msg("Failed to get notification settings")
		return
	}

	serverURL, err := getServerURL(ctx, settings)
	if err != nil {
		s.logger.Error().Err(err).Int64("notificationId", count.NotificationID).Msg("Failed to get server URL")
		return
	}

	s.refreshSection(ctx, count, serverURL, settings)
	s.clearSection(ctx, count)
}

func (s *RefreshService) refreshSection(ctx context.Context, count sqlc.CountPendingPlexRefreshesPerSectionRow, serverURL string, settings *Settings) {
	if count.Count > maxPathsBeforeFullRefresh || !settings.UsePartialRefresh {
		if err := s.doFullRefresh(ctx, serverURL, int(count.SectionKey), settings.AuthToken, settings.ClientID); err != nil {
			s.logger.Error().Err(err).Int64("sectionKey", count.SectionKey).Msg("Full section refresh failed")
		}
		return
	}

	if err := s.doPartialRefreshes(ctx, count.NotificationID, count.ServerID, int(count.SectionKey), serverURL, settings.AuthToken, settings.ClientID); err != nil {
		s.logger.Error().Err(err).Int64("sectionKey", count.SectionKey).Msg("Partial refresh failed, attempting full refresh")
		if err := s.doFullRefresh(ctx, serverURL, int(count.SectionKey), settings.AuthToken, settings.ClientID); err != nil {
			s.logger.Error().Err(err).Int64("sectionKey", count.SectionKey).Msg("Full section refresh also failed")
		}
	}
}

func (s *RefreshService) clearSection(ctx context.Context, count sqlc.CountPendingPlexRefreshesPerSectionRow) {
	if err := s.queries.ClearPlexRefreshesBySection(ctx, sqlc.ClearPlexRefreshesBySectionParams{
		NotificationID: count.NotificationID,
		ServerID:       count.ServerID,
		SectionKey:     count.SectionKey,
	}); err != nil {
		s.logger.Error().Err(err).Msg("Failed to clear processed refresh items")
	}
}

func (s *RefreshService) doFullRefresh(ctx context.Context, serverURL string, sectionKey int, token, clientID string) error {
	s.logger.Info().Int("sectionKey", sectionKey).Msg("Performing full section refresh")
	return s.client.RefreshSectionWithClientID(ctx, serverURL, sectionKey, token, clientID)
}

func (s *RefreshService) doPartialRefreshes(ctx context.Context, notificationID int64, serverID string, sectionKey int, serverURL, token, clientID string) error {
	items, err := s.queries.GetPendingPlexRefreshesBySection(ctx, sqlc.GetPendingPlexRefreshesBySectionParams{
		NotificationID: notificationID,
		ServerID:       serverID,
		SectionKey:     int64(sectionKey),
	})
	if err != nil {
		return err
	}

	for _, item := range items {
		if !item.Path.Valid || item.Path.String == "" {
			continue
		}

		s.logger.Debug().
			Int("sectionKey", sectionKey).
			Str("path", item.Path.String).
			Msg("Performing partial path refresh")

		if err := s.client.RefreshPathWithClientID(ctx, serverURL, sectionKey, item.Path.String, token, clientID); err != nil {
			return err
		}

		time.Sleep(refreshDelayBetweenCalls)
	}

	return nil
}

// MapPath applies path mappings to transform a local path to a Plex path
func MapPath(path string, mappings []PathMapping) string {
	for _, m := range mappings {
		if strings.HasPrefix(path, m.From) {
			return m.To + strings.TrimPrefix(path, m.From)
		}
	}
	return path
}

// FindMatchingSection finds the library section that contains the given path
func FindMatchingSection(path string, sections []LibrarySection, targetSectionIDs []int) *LibrarySection {
	for _, section := range sections {
		if !containsInt(targetSectionIDs, section.Key) {
			continue
		}

		for _, loc := range section.Locations {
			if strings.HasPrefix(path, loc.Path) {
				return &section
			}
		}
	}
	return nil
}

func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}
