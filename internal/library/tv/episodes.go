package tv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/status"
	"github.com/slipstream/slipstream/internal/mediainfo"
)

// ListEpisodes returns episodes for a series, optionally filtered by season.
func (s *Service) ListEpisodes(ctx context.Context, seriesID int64, seasonNumber *int) ([]Episode, error) {
	var rows []*sqlc.Episode
	var err error

	if seasonNumber != nil {
		rows, err = s.queries.ListEpisodesBySeason(ctx, sqlc.ListEpisodesBySeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: int64(*seasonNumber),
		})
	} else {
		rows, err = s.queries.ListEpisodesBySeries(ctx, seriesID)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list episodes: %w", err)
	}

	episodes := make([]Episode, len(rows))
	for i, row := range rows {
		episodes[i] = s.rowToEpisode(row)
		files, _ := s.queries.ListEpisodeFilesByEpisode(ctx, row.ID)
		if len(files) > 0 {
			ef := s.rowToEpisodeFile(files[0])
			episodes[i].EpisodeFile = &ef
		}
	}
	return episodes, nil
}

// GetEpisode retrieves an episode by ID.
func (s *Service) GetEpisode(ctx context.Context, id int64) (*Episode, error) {
	row, err := s.queries.GetEpisode(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEpisodeNotFound
		}
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	episode := s.rowToEpisode(row)

	files, _ := s.queries.ListEpisodeFilesByEpisode(ctx, id)
	if len(files) > 0 {
		ef := s.rowToEpisodeFile(files[0])
		episode.EpisodeFile = &ef
	}

	return &episode, nil
}

// GetEpisodeByNumber retrieves an episode by series ID, season number, and episode number.
func (s *Service) GetEpisodeByNumber(ctx context.Context, seriesID int64, seasonNumber, episodeNumber int) (*Episode, error) {
	row, err := s.queries.GetEpisodeByNumber(ctx, sqlc.GetEpisodeByNumberParams{
		SeriesID:      seriesID,
		SeasonNumber:  int64(seasonNumber),
		EpisodeNumber: int64(episodeNumber),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEpisodeNotFound
		}
		return nil, fmt.Errorf("failed to get episode by number: %w", err)
	}

	episode := s.rowToEpisode(row)

	files, _ := s.queries.ListEpisodeFilesByEpisode(ctx, episode.ID)
	if len(files) > 0 {
		ef := s.rowToEpisodeFile(files[0])
		episode.EpisodeFile = &ef
	}

	return &episode, nil
}

// UpdateEpisode updates an episode.
func (s *Service) UpdateEpisode(ctx context.Context, id int64, input UpdateEpisodeInput) (*Episode, error) {
	current, err := s.GetEpisode(ctx, id)
	if err != nil {
		return nil, err
	}

	title := current.Title
	if input.Title != nil {
		title = *input.Title
	}

	overview := current.Overview
	if input.Overview != nil {
		overview = *input.Overview
	}

	airDate := current.AirDate
	if input.AirDate != nil {
		airDate = input.AirDate
	}

	monitored := current.Monitored
	if input.Monitored != nil {
		monitored = *input.Monitored
	}

	var airDateSQL sql.NullTime
	if airDate != nil {
		airDateSQL = sql.NullTime{Time: *airDate, Valid: true}
	}

	row, err := s.queries.UpdateEpisode(ctx, sqlc.UpdateEpisodeParams{
		ID:        id,
		Title:     sql.NullString{String: title, Valid: title != ""},
		Overview:  sql.NullString{String: overview, Valid: overview != ""},
		AirDate:   airDateSQL,
		Monitored: boolToInt(monitored),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update episode: %w", err)
	}

	episode := s.rowToEpisode(row)
	return &episode, nil
}

// CreateEpisode creates a new episode in the database.
// This is used during season pack imports when episodes don't exist in metadata.
func (s *Service) CreateEpisode(ctx context.Context, seriesID int64, seasonNumber, episodeNumber int, title string) (*Episode, error) {
	// Check if season exists, create if not
	_, err := s.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     seriesID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create the season first
			_, err = s.queries.UpsertSeason(ctx, sqlc.UpsertSeasonParams{
				SeriesID:     seriesID,
				SeasonNumber: int64(seasonNumber),
				Monitored:    1,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create season: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to check season: %w", err)
		}
	}

	// Create the episode with status "missing" since we have a file for it (it will become "available" once the file is linked)
	row, err := s.queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
		SeriesID:      seriesID,
		SeasonNumber:  int64(seasonNumber),
		EpisodeNumber: int64(episodeNumber),
		Title:         sql.NullString{String: title, Valid: title != ""},
		Monitored:     1,
		Status:        status.Missing,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create episode: %w", err)
	}

	s.logger.Info().
		Int64("seriesId", seriesID).
		Int("season", seasonNumber).
		Int("episode", episodeNumber).
		Str("title", title).
		Msg("Created episode from season pack import")

	episode := s.rowToEpisode(row)
	return &episode, nil
}

// AddEpisodeFile adds a file to an episode.
func (s *Service) AddEpisodeFile(ctx context.Context, episodeID int64, input *CreateEpisodeFileInput) (*EpisodeFile, error) {
	episode, err := s.GetEpisode(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	qualityID := sql.NullInt64{}
	if input.QualityID != nil {
		qualityID = sql.NullInt64{Int64: *input.QualityID, Valid: true}
	}

	var row *sqlc.EpisodeFile

	// Use CreateEpisodeFileWithImportInfo when original path is provided (for import tracking)
	if input.OriginalPath != "" {
		row, err = s.queries.CreateEpisodeFileWithImportInfo(ctx, sqlc.CreateEpisodeFileWithImportInfoParams{
			EpisodeID:        episodeID,
			Path:             input.Path,
			Size:             input.Size,
			Quality:          sql.NullString{String: input.Quality, Valid: input.Quality != ""},
			QualityID:        qualityID,
			VideoCodec:       sql.NullString{String: input.VideoCodec, Valid: input.VideoCodec != ""},
			AudioCodec:       sql.NullString{String: input.AudioCodec, Valid: input.AudioCodec != ""},
			AudioChannels:    sql.NullString{String: input.AudioChannels, Valid: input.AudioChannels != ""},
			DynamicRange:     sql.NullString{String: input.DynamicRange, Valid: input.DynamicRange != ""},
			Resolution:       sql.NullString{String: input.Resolution, Valid: input.Resolution != ""},
			OriginalPath:     sql.NullString{String: input.OriginalPath, Valid: true},
			OriginalFilename: sql.NullString{String: input.OriginalFilename, Valid: input.OriginalFilename != ""},
			ImportedAt:       sql.NullTime{Time: time.Now(), Valid: true},
		})
	} else {
		row, err = s.queries.CreateEpisodeFile(ctx, sqlc.CreateEpisodeFileParams{
			EpisodeID:     episodeID,
			Path:          input.Path,
			Size:          input.Size,
			Quality:       sql.NullString{String: input.Quality, Valid: input.Quality != ""},
			QualityID:     qualityID,
			VideoCodec:    sql.NullString{String: input.VideoCodec, Valid: input.VideoCodec != ""},
			AudioCodec:    sql.NullString{String: input.AudioCodec, Valid: input.AudioCodec != ""},
			AudioChannels: sql.NullString{String: input.AudioChannels, Valid: input.AudioChannels != ""},
			DynamicRange:  sql.NullString{String: input.DynamicRange, Valid: input.DynamicRange != ""},
			Resolution:    sql.NullString{String: input.Resolution, Valid: input.Resolution != ""},
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create episode file: %w", err)
	}

	epStatus := status.Available
	if qualityID.Valid && s.qualityProfiles != nil {
		if series, seriesErr := s.GetSeries(ctx, episode.SeriesID); seriesErr == nil {
			if profile, profileErr := s.qualityProfiles.Get(ctx, series.QualityProfileID); profileErr == nil {
				epStatus = profile.StatusForQuality(int(qualityID.Int64))
			}
		}
	}
	_ = s.queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:     episodeID,
		Status: epStatus,
	})

	file := s.rowToEpisodeFile(row)
	s.logger.Info().Int64("episodeId", episodeID).Str("path", input.Path).Msg("Added episode file")

	return &file, nil
}

// GetEpisodeFileByPath retrieves an episode file by its path.
// Returns sql.ErrNoRows if the file doesn't exist.
func (s *Service) GetEpisodeFileByPath(ctx context.Context, path string) (*EpisodeFile, error) {
	row, err := s.queries.GetEpisodeFileByPath(ctx, path)
	if err != nil {
		return nil, err
	}
	file := s.rowToEpisodeFile(row)
	return &file, nil
}

// GetEpisodeFile returns the primary file for an episode.
// Returns sql.ErrNoRows if no file exists.
func (s *Service) GetEpisodeFile(ctx context.Context, episodeID int64) (*EpisodeFile, error) {
	rows, err := s.queries.ListEpisodeFilesByEpisode(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list episode files: %w", err)
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	file := s.rowToEpisodeFile(rows[0])
	return &file, nil
}

// RemoveEpisodeFile removes a file from an episode.
// Req 12.1.1: Deleting file from slot does NOT trigger automatic search
// Req 12.1.2: Slot becomes empty; waits for next scheduled search
func (s *Service) RemoveEpisodeFile(ctx context.Context, fileID int64) error {
	row, err := s.queries.GetEpisodeFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEpisodeFileNotFound
		}
		return fmt.Errorf("failed to get episode file: %w", err)
	}

	if s.fileDeleteHandler != nil {
		if err := s.fileDeleteHandler.OnFileDeleted(ctx, "episode", fileID); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear slot assignment")
		}
	}

	if err := s.queries.DeleteEpisodeFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete episode file: %w", err)
	}

	if err := s.queries.DeleteImportDecisionsByExistingFile(ctx, sql.NullInt64{Int64: fileID, Valid: true}); err != nil {
		s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear import decisions for removed file")
	}

	count, _ := s.queries.CountEpisodeFiles(ctx, row.EpisodeID)
	if count == 0 {
		s.transitionEpisodeToMissingAfterFileRemoval(ctx, row.EpisodeID)
	}

	s.logger.Info().Int64("fileId", fileID).Int64("episodeId", row.EpisodeID).Msg("Removed episode file")
	return nil
}

// GetEpisodeFileByID retrieves an episode file by its ID.
func (s *Service) GetEpisodeFileByID(ctx context.Context, fileID int64) (*EpisodeFile, error) {
	row, err := s.queries.GetEpisodeFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEpisodeFileNotFound
		}
		return nil, fmt.Errorf("failed to get episode file: %w", err)
	}
	file := s.rowToEpisodeFile(row)
	return &file, nil
}

// UpdateEpisodeFilePath updates the path of an episode file.
func (s *Service) UpdateEpisodeFilePath(ctx context.Context, fileID int64, newPath string) error {
	return s.queries.UpdateEpisodeFilePath(ctx, sqlc.UpdateEpisodeFilePathParams{
		Path: newPath,
		ID:   fileID,
	})
}

// UpdateEpisodeFileMediaInfo updates the MediaInfo fields of an episode file.
func (s *Service) UpdateEpisodeFileMediaInfo(ctx context.Context, episodeID int64, info *mediainfo.MediaInfo) error {
	return s.queries.UpdateEpisodeFileMediaInfo(ctx, sqlc.UpdateEpisodeFileMediaInfoParams{
		VideoCodec: sql.NullString{String: info.VideoCodec, Valid: info.VideoCodec != ""},
		AudioCodec: sql.NullString{String: info.AudioCodec, Valid: info.AudioCodec != ""},
		Resolution: sql.NullString{String: info.VideoResolution, Valid: info.VideoResolution != ""},
		EpisodeID:  episodeID,
	})
}

// BulkMonitorEpisodes updates the monitored status of multiple episodes.
func (s *Service) BulkMonitorEpisodes(ctx context.Context, seriesID int64, input BulkEpisodeMonitorInput) error {
	// Verify series exists
	_, err := s.GetSeries(ctx, seriesID)
	if err != nil {
		return err
	}

	if len(input.EpisodeIDs) == 0 {
		return nil
	}

	if err := s.queries.UpdateEpisodesMonitoredByIDs(ctx, sqlc.UpdateEpisodesMonitoredByIDsParams{
		Monitored: boolToInt(input.Monitored),
		Ids:       input.EpisodeIDs,
	}); err != nil {
		return fmt.Errorf("failed to update episodes: %w", err)
	}

	s.logger.Info().
		Int64("seriesId", seriesID).
		Int("episodeCount", len(input.EpisodeIDs)).
		Bool("monitored", input.Monitored).
		Msg("Applied bulk episode monitoring")

	if s.hub != nil {
		s.hub.Broadcast("series:updated", map[string]int64{"id": seriesID})
	}

	return nil
}

func (s *Service) transitionEpisodeToMissingAfterFileRemoval(ctx context.Context, episodeID int64) {
	episode, _ := s.queries.GetEpisode(ctx, episodeID)
	oldStatus := ""
	if episode != nil {
		oldStatus = episode.Status
	}
	_ = s.queries.UpdateEpisodeStatusWithDetails(ctx, sqlc.UpdateEpisodeStatusWithDetailsParams{
		ID:     episodeID,
		Status: status.Missing,
	})
	_ = s.queries.UpdateEpisodeMonitored(ctx, sqlc.UpdateEpisodeMonitoredParams{
		ID:        episodeID,
		Monitored: 0,
	})
	if s.statusChangeLogger != nil && oldStatus != "" && oldStatus != status.Missing {
		_ = s.statusChangeLogger.LogStatusChanged(ctx, "episode", episodeID, oldStatus, status.Missing, "File removed")
	}
	if s.hub != nil && episode != nil {
		s.hub.Broadcast("series:updated", map[string]any{"id": episode.SeriesID})
	}
}

// rowToEpisode converts a database row to an Episode.
func (s *Service) rowToEpisode(row *sqlc.Episode) Episode {
	ep := Episode{
		ID:            row.ID,
		SeriesID:      row.SeriesID,
		SeasonNumber:  int(row.SeasonNumber),
		EpisodeNumber: int(row.EpisodeNumber),
		Monitored:     row.Monitored == 1,
		Status:        row.Status,
	}

	if row.Title.Valid {
		ep.Title = row.Title.String
	}
	if row.Overview.Valid {
		ep.Overview = row.Overview.String
	}
	if row.AirDate.Valid {
		ep.AirDate = &row.AirDate.Time
	}
	if row.StatusMessage.Valid {
		msg := row.StatusMessage.String
		ep.StatusMessage = &msg
	}
	if row.ActiveDownloadID.Valid {
		dlID := row.ActiveDownloadID.String
		ep.ActiveDownloadID = &dlID
	}

	return ep
}

// rowToEpisodeFile converts a database row to an EpisodeFile.
func (s *Service) rowToEpisodeFile(row *sqlc.EpisodeFile) EpisodeFile {
	f := EpisodeFile{
		ID:        row.ID,
		EpisodeID: row.EpisodeID,
		Path:      row.Path,
		Size:      row.Size,
	}

	if row.Quality.Valid {
		f.Quality = row.Quality.String
	}
	if row.VideoCodec.Valid {
		f.VideoCodec = row.VideoCodec.String
	}
	if row.AudioCodec.Valid {
		f.AudioCodec = row.AudioCodec.String
	}
	if row.AudioChannels.Valid {
		f.AudioChannels = row.AudioChannels.String
	}
	if row.DynamicRange.Valid {
		f.DynamicRange = row.DynamicRange.String
	}
	if row.Resolution.Valid {
		f.Resolution = row.Resolution.String
	}
	if row.CreatedAt.Valid {
		f.CreatedAt = row.CreatedAt.Time
	}
	if row.SlotID.Valid {
		slotID := row.SlotID.Int64
		f.SlotID = &slotID
	}

	return f
}
