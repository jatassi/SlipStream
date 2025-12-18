package tv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/websocket"
)

var (
	ErrSeriesNotFound      = errors.New("series not found")
	ErrSeasonNotFound      = errors.New("season not found")
	ErrEpisodeNotFound     = errors.New("episode not found")
	ErrEpisodeFileNotFound = errors.New("episode file not found")
	ErrInvalidSeries       = errors.New("invalid series data")
	ErrDuplicateTvdbID     = errors.New("series with this TVDB ID already exists")
)

// Service provides TV library operations.
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	hub     *websocket.Hub
	logger  zerolog.Logger
}

// NewService creates a new TV service.
func NewService(db *sql.DB, hub *websocket.Hub, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		hub:     hub,
		logger:  logger.With().Str("component", "tv").Logger(),
	}
}

// GetSeries retrieves a series by ID.
func (s *Service) GetSeries(ctx context.Context, id int64) (*Series, error) {
	row, err := s.queries.GetSeries(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSeriesNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	series := s.rowToSeries(row)

	// Get seasons
	seasons, err := s.ListSeasons(ctx, id)
	if err == nil {
		series.Seasons = seasons
	}

	// Get counts
	episodeCount, _ := s.queries.CountEpisodesBySeries(ctx, id)
	fileCount, _ := s.queries.CountEpisodeFilesBySeries(ctx, id)
	series.EpisodeCount = int(episodeCount)
	series.EpisodeFileCount = int(fileCount)

	return series, nil
}

// GetSeriesByTvdbID retrieves a series by TVDB ID.
func (s *Service) GetSeriesByTvdbID(ctx context.Context, tvdbID int) (*Series, error) {
	row, err := s.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: int64(tvdbID), Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSeriesNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}
	return s.rowToSeries(row), nil
}

// ListSeries returns series with optional filtering.
func (s *Service) ListSeries(ctx context.Context, opts ListSeriesOptions) ([]*Series, error) {
	var rows []*sqlc.Series
	var err error

	// Default pagination
	if opts.PageSize <= 0 {
		opts.PageSize = 100
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	offset := int64((opts.Page - 1) * opts.PageSize)

	if opts.Search != "" {
		searchTerm := "%" + opts.Search + "%"
		rows, err = s.queries.SearchSeries(ctx, sqlc.SearchSeriesParams{
			Title:     searchTerm,
			SortTitle: searchTerm,
			Limit:     int64(opts.PageSize),
			Offset:    offset,
		})
	} else if opts.RootFolderID != nil {
		rows, err = s.queries.ListSeriesByRootFolder(ctx, sql.NullInt64{Int64: *opts.RootFolderID, Valid: true})
	} else if opts.Monitored != nil && *opts.Monitored {
		rows, err = s.queries.ListMonitoredSeries(ctx)
	} else {
		rows, err = s.queries.ListSeriesPaginated(ctx, sqlc.ListSeriesPaginatedParams{
			Limit:  int64(opts.PageSize),
			Offset: offset,
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}

	seriesList := make([]*Series, len(rows))
	for i, row := range rows {
		seriesList[i] = s.rowToSeries(row)
		// Get counts
		episodeCount, _ := s.queries.CountEpisodesBySeries(ctx, row.ID)
		fileCount, _ := s.queries.CountEpisodeFilesBySeries(ctx, row.ID)
		seriesList[i].EpisodeCount = int(episodeCount)
		seriesList[i].EpisodeFileCount = int(fileCount)
	}
	return seriesList, nil
}

// CreateSeries creates a new series.
func (s *Service) CreateSeries(ctx context.Context, input CreateSeriesInput) (*Series, error) {
	if input.Title == "" {
		return nil, ErrInvalidSeries
	}

	// Check for duplicate TVDB ID
	if input.TvdbID > 0 {
		_, err := s.GetSeriesByTvdbID(ctx, input.TvdbID)
		if err == nil {
			return nil, ErrDuplicateTvdbID
		}
		if !errors.Is(err, ErrSeriesNotFound) {
			return nil, err
		}
	}

	sortTitle := generateSortTitle(input.Title)

	row, err := s.queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
		Title:            input.Title,
		SortTitle:        sortTitle,
		Year:             sql.NullInt64{Int64: int64(input.Year), Valid: input.Year > 0},
		TvdbID:           sql.NullInt64{Int64: int64(input.TvdbID), Valid: input.TvdbID > 0},
		TmdbID:           sql.NullInt64{Int64: int64(input.TmdbID), Valid: input.TmdbID > 0},
		ImdbID:           sql.NullString{String: input.ImdbID, Valid: input.ImdbID != ""},
		Overview:         sql.NullString{String: input.Overview, Valid: input.Overview != ""},
		Runtime:          sql.NullInt64{Int64: int64(input.Runtime), Valid: input.Runtime > 0},
		Path:             sql.NullString{String: input.Path, Valid: input.Path != ""},
		RootFolderID:     sql.NullInt64{Int64: input.RootFolderID, Valid: input.RootFolderID > 0},
		QualityProfileID: sql.NullInt64{Int64: input.QualityProfileID, Valid: input.QualityProfileID > 0},
		Monitored:        boolToInt(input.Monitored),
		SeasonFolder:     boolToInt(input.SeasonFolder),
		Status:           "continuing",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create series: %w", err)
	}

	// Create seasons and episodes if provided
	for _, seasonInput := range input.Seasons {
		_, err := s.queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
			SeriesID:     row.ID,
			SeasonNumber: int64(seasonInput.SeasonNumber),
			Monitored:    boolToInt(seasonInput.Monitored),
		})
		if err != nil {
			s.logger.Warn().Err(err).Int("season", seasonInput.SeasonNumber).Msg("Failed to create season")
			continue
		}

		for _, episodeInput := range seasonInput.Episodes {
			var airDate sql.NullTime
			if episodeInput.AirDate != nil {
				airDate = sql.NullTime{Time: *episodeInput.AirDate, Valid: true}
			}
			_, err := s.queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
				SeriesID:      row.ID,
				SeasonNumber:  int64(seasonInput.SeasonNumber),
				EpisodeNumber: int64(episodeInput.EpisodeNumber),
				Title:         sql.NullString{String: episodeInput.Title, Valid: episodeInput.Title != ""},
				Overview:      sql.NullString{String: episodeInput.Overview, Valid: episodeInput.Overview != ""},
				AirDate:       airDate,
				Monitored:     boolToInt(episodeInput.Monitored),
			})
			if err != nil {
				s.logger.Warn().Err(err).Int("episode", episodeInput.EpisodeNumber).Msg("Failed to create episode")
			}
		}
	}

	series := s.rowToSeries(row)
	s.logger.Info().Int64("id", series.ID).Str("title", series.Title).Msg("Created series")

	if s.hub != nil {
		s.hub.Broadcast("series:added", series)
	}

	return series, nil
}

// UpdateSeries updates an existing series.
func (s *Service) UpdateSeries(ctx context.Context, id int64, input UpdateSeriesInput) (*Series, error) {
	current, err := s.GetSeries(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply updates
	title := current.Title
	if input.Title != nil {
		title = *input.Title
	}
	sortTitle := generateSortTitle(title)

	year := current.Year
	if input.Year != nil {
		year = *input.Year
	}

	tvdbID := current.TvdbID
	if input.TvdbID != nil {
		tvdbID = *input.TvdbID
	}

	tmdbID := current.TmdbID
	if input.TmdbID != nil {
		tmdbID = *input.TmdbID
	}

	imdbID := current.ImdbID
	if input.ImdbID != nil {
		imdbID = *input.ImdbID
	}

	overview := current.Overview
	if input.Overview != nil {
		overview = *input.Overview
	}

	runtime := current.Runtime
	if input.Runtime != nil {
		runtime = *input.Runtime
	}

	path := current.Path
	if input.Path != nil {
		path = *input.Path
	}

	rootFolderID := current.RootFolderID
	if input.RootFolderID != nil {
		rootFolderID = *input.RootFolderID
	}

	qualityProfileID := current.QualityProfileID
	if input.QualityProfileID != nil {
		qualityProfileID = *input.QualityProfileID
	}

	monitored := current.Monitored
	if input.Monitored != nil {
		monitored = *input.Monitored
	}

	seasonFolder := current.SeasonFolder
	if input.SeasonFolder != nil {
		seasonFolder = *input.SeasonFolder
	}

	status := current.Status
	if input.Status != nil {
		status = *input.Status
	}

	row, err := s.queries.UpdateSeries(ctx, sqlc.UpdateSeriesParams{
		ID:               id,
		Title:            title,
		SortTitle:        sortTitle,
		Year:             sql.NullInt64{Int64: int64(year), Valid: year > 0},
		TvdbID:           sql.NullInt64{Int64: int64(tvdbID), Valid: tvdbID > 0},
		TmdbID:           sql.NullInt64{Int64: int64(tmdbID), Valid: tmdbID > 0},
		ImdbID:           sql.NullString{String: imdbID, Valid: imdbID != ""},
		Overview:         sql.NullString{String: overview, Valid: overview != ""},
		Runtime:          sql.NullInt64{Int64: int64(runtime), Valid: runtime > 0},
		Path:             sql.NullString{String: path, Valid: path != ""},
		RootFolderID:     sql.NullInt64{Int64: rootFolderID, Valid: rootFolderID > 0},
		QualityProfileID: sql.NullInt64{Int64: qualityProfileID, Valid: qualityProfileID > 0},
		Monitored:        boolToInt(monitored),
		SeasonFolder:     boolToInt(seasonFolder),
		Status:           status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update series: %w", err)
	}

	series := s.rowToSeries(row)
	s.logger.Info().Int64("id", id).Str("title", series.Title).Msg("Updated series")

	if s.hub != nil {
		s.hub.Broadcast("series:updated", series)
	}

	return series, nil
}

// DeleteSeries deletes a series and all its seasons/episodes.
func (s *Service) DeleteSeries(ctx context.Context, id int64, deleteFiles bool) error {
	series, err := s.GetSeries(ctx, id)
	if err != nil {
		return err
	}

	// Delete all episode files, episodes, seasons
	// TODO: If deleteFiles is true, delete actual files from disk

	if err := s.queries.DeleteEpisodesBySeries(ctx, id); err != nil {
		return fmt.Errorf("failed to delete episodes: %w", err)
	}

	if err := s.queries.DeleteSeasonsBySeries(ctx, id); err != nil {
		return fmt.Errorf("failed to delete seasons: %w", err)
	}

	if err := s.queries.DeleteSeries(ctx, id); err != nil {
		return fmt.Errorf("failed to delete series: %w", err)
	}

	s.logger.Info().Int64("id", id).Str("title", series.Title).Msg("Deleted series")

	if s.hub != nil {
		s.hub.Broadcast("series:deleted", map[string]int64{"id": id})
	}

	return nil
}

// ListSeasons returns all seasons for a series.
func (s *Service) ListSeasons(ctx context.Context, seriesID int64) ([]Season, error) {
	rows, err := s.queries.ListSeasonsBySeries(ctx, seriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to list seasons: %w", err)
	}

	seasons := make([]Season, len(rows))
	for i, row := range rows {
		seasons[i] = s.rowToSeason(row)
		// Get counts
		episodeCount, _ := s.queries.CountEpisodesBySeason(ctx, sqlc.CountEpisodesBySeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: row.SeasonNumber,
		})
		fileCount, _ := s.queries.CountEpisodeFilesBySeason(ctx, sqlc.CountEpisodeFilesBySeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: row.SeasonNumber,
		})
		seasons[i].EpisodeCount = int(episodeCount)
		seasons[i].EpisodeFileCount = int(fileCount)
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

	season := s.rowToSeason(updated)
	return &season, nil
}

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
		// Check for files
		files, _ := s.queries.ListEpisodeFilesByEpisode(ctx, row.ID)
		episodes[i].HasFile = len(files) > 0
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

	// Get files
	files, _ := s.queries.ListEpisodeFilesByEpisode(ctx, id)
	episode.HasFile = len(files) > 0
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

// AddEpisodeFile adds a file to an episode.
func (s *Service) AddEpisodeFile(ctx context.Context, episodeID int64, input CreateEpisodeFileInput) (*EpisodeFile, error) {
	// Verify episode exists
	_, err := s.GetEpisode(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	row, err := s.queries.CreateEpisodeFile(ctx, sqlc.CreateEpisodeFileParams{
		EpisodeID:  episodeID,
		Path:       input.Path,
		Size:       input.Size,
		Quality:    sql.NullString{String: input.Quality, Valid: input.Quality != ""},
		VideoCodec: sql.NullString{String: input.VideoCodec, Valid: input.VideoCodec != ""},
		AudioCodec: sql.NullString{String: input.AudioCodec, Valid: input.AudioCodec != ""},
		Resolution: sql.NullString{String: input.Resolution, Valid: input.Resolution != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create episode file: %w", err)
	}

	file := s.rowToEpisodeFile(row)
	s.logger.Info().Int64("episodeId", episodeID).Str("path", input.Path).Msg("Added episode file")

	return &file, nil
}

// RemoveEpisodeFile removes a file from an episode.
func (s *Service) RemoveEpisodeFile(ctx context.Context, fileID int64) error {
	_, err := s.queries.GetEpisodeFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEpisodeFileNotFound
		}
		return fmt.Errorf("failed to get episode file: %w", err)
	}

	if err := s.queries.DeleteEpisodeFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete episode file: %w", err)
	}

	s.logger.Info().Int64("fileId", fileID).Msg("Removed episode file")
	return nil
}

// Count returns the total number of series.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.queries.CountSeries(ctx)
}

// rowToSeries converts a database row to a Series.
func (s *Service) rowToSeries(row *sqlc.Series) *Series {
	series := &Series{
		ID:           row.ID,
		Title:        row.Title,
		SortTitle:    row.SortTitle,
		Monitored:    row.Monitored == 1,
		SeasonFolder: row.SeasonFolder == 1,
		Status:       row.Status,
	}

	if row.Year.Valid {
		series.Year = int(row.Year.Int64)
	}
	if row.TvdbID.Valid {
		series.TvdbID = int(row.TvdbID.Int64)
	}
	if row.TmdbID.Valid {
		series.TmdbID = int(row.TmdbID.Int64)
	}
	if row.ImdbID.Valid {
		series.ImdbID = row.ImdbID.String
	}
	if row.Overview.Valid {
		series.Overview = row.Overview.String
	}
	if row.Runtime.Valid {
		series.Runtime = int(row.Runtime.Int64)
	}
	if row.Path.Valid {
		series.Path = row.Path.String
	}
	if row.RootFolderID.Valid {
		series.RootFolderID = row.RootFolderID.Int64
	}
	if row.QualityProfileID.Valid {
		series.QualityProfileID = row.QualityProfileID.Int64
	}
	if row.AddedAt.Valid {
		series.AddedAt = row.AddedAt.Time
	}
	if row.UpdatedAt.Valid {
		series.UpdatedAt = row.UpdatedAt.Time
	}

	return series
}

// rowToSeason converts a database row to a Season.
func (s *Service) rowToSeason(row *sqlc.Season) Season {
	return Season{
		ID:           row.ID,
		SeriesID:     row.SeriesID,
		SeasonNumber: int(row.SeasonNumber),
		Monitored:    row.Monitored == 1,
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
	if row.Resolution.Valid {
		f.Resolution = row.Resolution.String
	}
	if row.CreatedAt.Valid {
		f.CreatedAt = row.CreatedAt.Time
	}

	return f
}

// GenerateSeriesPath generates a path for a series.
func GenerateSeriesPath(rootPath, title string) string {
	return filepath.Join(rootPath, title)
}

// GenerateSeasonPath generates a path for a season folder.
func GenerateSeasonPath(seriesPath string, seasonNumber int) string {
	return filepath.Join(seriesPath, fmt.Sprintf("Season %02d", seasonNumber))
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
