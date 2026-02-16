package tv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// ListSeasons returns all seasons for a series.
func (s *Service) ListSeasons(ctx context.Context, seriesID int64) ([]Season, error) {
	rows, err := s.queries.ListSeasonsBySeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to list seasons: %w", err)
	}

	seasons := make([]Season, len(rows))
	for i, row := range rows {
		seasons[i] = s.rowToSeason(row)
		s.enrichSeasonWithCounts(ctx, &seasons[i], seriesID)
	}
	return seasons, nil
}

// UpdateSeasonMonitored updates the monitored status of a season.
func (s *Service) UpdateSeasonMonitored(ctx context.Context, seriesID int64, seasonNumber int, monitored bool) (*Season, error) {
	// Get season by series and number
	row, err := s.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSeasonNotFound
		}
		return nil, fmt.Errorf("failed to get season: %w", err)
	}

	updated, err := s.queries.UpdateSeason(ctx, sqlc.UpdateSeasonParams{
		ID:        row.ID,
		Monitored: boolToInt(monitored),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update season: %w", err)
	}

	// Cascade monitoring to all episodes in this season
	if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
		Monitored:    boolToInt(monitored),
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	}); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", seriesID).Int("seasonNumber", seasonNumber).Msg("Failed to cascade monitoring to episodes")
	}

	season := s.rowToSeason(updated)
	return &season, nil
}

// BulkMonitor applies a monitoring preset to a series.
func (s *Service) BulkMonitor(ctx context.Context, seriesID int64, input BulkMonitorInput) error {
	_, err := s.GetSeries(ctx, seriesID)
	if err != nil {
		return err
	}

	switch input.MonitorType {
	case MonitorTypeAll:
		err = s.applyMonitorAll(ctx, seriesID, input.IncludeSpecials)
	case MonitorTypeNone:
		err = s.applyMonitorNone(ctx, seriesID)
	case MonitorTypeFuture:
		err = s.applyMonitorFuture(ctx, seriesID, input.IncludeSpecials)
	case MonitorTypeFirstSeason:
		err = s.applyMonitorFirstSeason(ctx, seriesID)
	case MonitorTypeLatest:
		err = s.applyMonitorLatest(ctx, seriesID)
	default:
		return fmt.Errorf("unknown monitor type: %s", input.MonitorType)
	}

	if err != nil {
		return err
	}

	s.logger.Info().
		Int64("seriesId", seriesID).
		Str("monitorType", string(input.MonitorType)).
		Bool("includeSpecials", input.IncludeSpecials).
		Msg("Applied bulk monitoring")

	if s.hub != nil {
		s.hub.Broadcast("series:updated", map[string]int64{"id": seriesID})
	}

	return nil
}

func (s *Service) applyMonitorAll(ctx context.Context, seriesID int64, includeSpecials bool) error {
	if includeSpecials {
		if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
			Monitored: 1,
			SeriesID:  seriesID,
		}); err != nil {
			return fmt.Errorf("failed to monitor seasons: %w", err)
		}
		if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
			Monitored: 1,
			SeriesID:  seriesID,
		}); err != nil {
			return fmt.Errorf("failed to monitor episodes: %w", err)
		}
		return nil
	}

	if err := s.queries.UpdateSeasonsMonitoredExcludingSpecials(ctx, sqlc.UpdateSeasonsMonitoredExcludingSpecialsParams{
		Monitored: 1,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to monitor seasons: %w", err)
	}
	if err := s.queries.UpdateEpisodesMonitoredExcludingSpecials(ctx, sqlc.UpdateEpisodesMonitoredExcludingSpecialsParams{
		Monitored: 1,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to monitor episodes: %w", err)
	}
	return s.unmonitorSpecials(ctx, seriesID)
}

func (s *Service) applyMonitorNone(ctx context.Context, seriesID int64) error {
	if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor seasons: %w", err)
	}
	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor episodes: %w", err)
	}
	return nil
}

func (s *Service) applyMonitorFuture(ctx context.Context, seriesID int64, includeSpecials bool) error {
	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor episodes: %w", err)
	}
	if err := s.queries.UpdateFutureEpisodesMonitored(ctx, sqlc.UpdateFutureEpisodesMonitoredParams{
		Monitored: 1,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to monitor future episodes: %w", err)
	}
	if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor seasons: %w", err)
	}
	if err := s.queries.UpdateFutureSeasonsMonitored(ctx, sqlc.UpdateFutureSeasonsMonitoredParams{
		Monitored:  1,
		SeriesID:   seriesID,
		SeriesID_2: seriesID,
	}); err != nil {
		return fmt.Errorf("failed to monitor future seasons: %w", err)
	}
	if !includeSpecials {
		return s.unmonitorSpecials(ctx, seriesID)
	}
	return nil
}

func (s *Service) applyMonitorFirstSeason(ctx context.Context, seriesID int64) error {
	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor episodes: %w", err)
	}
	if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor seasons: %w", err)
	}
	return s.monitorSeasonBySeason(ctx, seriesID, 1)
}

func (s *Service) applyMonitorLatest(ctx context.Context, seriesID int64) error {
	latestSeason, err := s.getLatestSeasonNumber(ctx, seriesID)
	if err != nil {
		return err
	}
	if latestSeason == 0 {
		return nil
	}

	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor episodes: %w", err)
	}
	if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: 0,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor seasons: %w", err)
	}
	return s.monitorSeasonBySeason(ctx, seriesID, latestSeason)
}

func (s *Service) unmonitorSpecials(ctx context.Context, seriesID int64) error {
	if err := s.queries.UpdateSeasonMonitoredByNumber(ctx, sqlc.UpdateSeasonMonitoredByNumberParams{
		Monitored:    0,
		SeriesID:     seriesID,
		SeasonNumber: 0,
	}); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to unmonitor specials season: %w", err)
	}
	if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
		Monitored:    0,
		SeriesID:     seriesID,
		SeasonNumber: 0,
	}); err != nil {
		return fmt.Errorf("failed to unmonitor specials: %w", err)
	}
	return nil
}

func (s *Service) monitorSeasonBySeason(ctx context.Context, seriesID, seasonNumber int64) error {
	if err := s.queries.UpdateSeasonMonitoredByNumber(ctx, sqlc.UpdateSeasonMonitoredByNumberParams{
		Monitored:    1,
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	}); err != nil {
		return fmt.Errorf("failed to monitor season: %w", err)
	}
	if err := s.queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
		Monitored:    1,
		SeriesID:     seriesID,
		SeasonNumber: seasonNumber,
	}); err != nil {
		return fmt.Errorf("failed to monitor season episodes: %w", err)
	}
	return nil
}

func (s *Service) getLatestSeasonNumber(ctx context.Context, seriesID int64) (int64, error) {
	latestResult, err := s.queries.GetLatestSeasonNumber(ctx, seriesID)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest season: %w", err)
	}
	switch v := latestResult.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case nil:
		return 0, nil
	default:
		return 0, nil
	}
}

// GetMonitoringStats returns monitoring statistics for a series.
func (s *Service) GetMonitoringStats(ctx context.Context, seriesID int64) (*MonitoringStats, error) {
	row, err := s.queries.GetSeriesMonitoringStats(ctx, sqlc.GetSeriesMonitoringStatsParams{
		SeriesID:   seriesID,
		SeriesID_2: seriesID,
		SeriesID_3: seriesID,
		SeriesID_4: seriesID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring stats: %w", err)
	}

	return &MonitoringStats{
		TotalSeasons:      row.TotalSeasons,
		MonitoredSeasons:  row.MonitoredSeasons,
		TotalEpisodes:     row.TotalEpisodes,
		MonitoredEpisodes: row.MonitoredEpisodes,
	}, nil
}

// AreSeasonsComplete checks if all monitored, released episodes in the specified seasons have files.
func (s *Service) AreSeasonsComplete(ctx context.Context, seriesID int64, seasonNumbers []int64) (bool, error) {
	if len(seasonNumbers) == 0 {
		return true, nil
	}

	count, err := s.queries.CountMissingEpisodesBySeasons(ctx, sqlc.CountMissingEpisodesBySeasonsParams{
		SeriesID:      seriesID,
		SeasonNumbers: seasonNumbers,
	})
	if err != nil {
		return false, fmt.Errorf("failed to count missing episodes: %w", err)
	}

	return count == 0, nil
}

// rowToSeason converts a database row to a Season.
func (s *Service) rowToSeason(row *sqlc.Season) Season {
	season := Season{
		ID:           row.ID,
		SeriesID:     row.SeriesID,
		SeasonNumber: int(row.SeasonNumber),
		Monitored:    row.Monitored == 1,
	}
	if row.Overview.Valid {
		season.Overview = row.Overview.String
	}
	if row.PosterUrl.Valid {
		season.PosterURL = row.PosterUrl.String
	}
	return season
}

// enrichSeasonWithCounts populates the StatusCounts field on a season by querying episode statuses.
func (s *Service) enrichSeasonWithCounts(ctx context.Context, season *Season, seriesID int64) {
	counts, err := s.queries.GetEpisodeStatusCountsBySeason(ctx, sqlc.GetEpisodeStatusCountsBySeasonParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(season.SeasonNumber),
	})
	if err != nil {
		return
	}
	season.StatusCounts = StatusCounts{
		Unreleased:  toInt(counts.Unreleased),
		Missing:     toInt(counts.Missing),
		Downloading: toInt(counts.Downloading),
		Failed:      toInt(counts.Failed),
		Upgradable:  toInt(counts.Upgradable),
		Available:   toInt(counts.Available),
		Total:       int(counts.Total),
	}
}
