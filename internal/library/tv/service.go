package tv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/pathutil"
	"github.com/slipstream/slipstream/internal/websocket"
)

// NotificationDispatcher defines the interface for series notifications.
type NotificationDispatcher interface {
	DispatchSeriesAdded(ctx context.Context, series *SeriesNotificationInfo, addedAt time.Time)
	DispatchSeriesDeleted(ctx context.Context, series *SeriesNotificationInfo, deletedFiles bool, deletedAt time.Time)
}

// SeriesNotificationInfo contains series info for notifications.
type SeriesNotificationInfo struct {
	ID        int64
	Title     string
	Year      int
	TvdbID    int
	TmdbID    int
	ImdbID    string
	Overview  string
	PosterURL string
}

var (
	ErrSeriesNotFound      = errors.New("series not found")
	ErrSeasonNotFound      = errors.New("season not found")
	ErrEpisodeNotFound     = errors.New("episode not found")
	ErrEpisodeFileNotFound = errors.New("episode file not found")
	ErrInvalidSeries       = errors.New("invalid series data")
	ErrDuplicateTvdbID     = errors.New("series with this TVDB ID already exists")
)

// FileDeleteHandler is called when a file is deleted to update slot assignments.
type FileDeleteHandler interface {
	OnFileDeleted(ctx context.Context, mediaType string, fileID int64) error
}

// StatusChangeLogger logs status transition history events.
type StatusChangeLogger interface {
	LogStatusChanged(ctx context.Context, mediaType string, mediaID int64, from, to, reason string) error
}

// Service provides TV library operations.
type Service struct {
	db                 *sql.DB
	queries            *sqlc.Queries
	hub                *websocket.Hub
	logger             *zerolog.Logger
	fileDeleteHandler  FileDeleteHandler
	statusChangeLogger StatusChangeLogger
	notifier           NotificationDispatcher
	qualityProfiles    *quality.Service
}

// SetNotificationDispatcher sets the notification dispatcher for series events.
func (s *Service) SetNotificationDispatcher(n NotificationDispatcher) {
	s.notifier = n
}

// SetFileDeleteHandler sets the handler for file deletion events.
// Req 12.1.1: Deleting file from slot does NOT trigger automatic search
func (s *Service) SetFileDeleteHandler(handler FileDeleteHandler) {
	s.fileDeleteHandler = handler
}

// SetStatusChangeLogger sets the logger for status transition history events.
func (s *Service) SetStatusChangeLogger(logger StatusChangeLogger) {
	s.statusChangeLogger = logger
}

// SetQualityService sets the quality profile service for quality evaluation.
func (s *Service) SetQualityService(qs *quality.Service) {
	s.qualityProfiles = qs
}

// NewService creates a new TV service.
func NewService(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "tv").Logger()
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		hub:     hub,
		logger:  &subLogger,
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// GetSeries retrieves a series by ID.
func (s *Service) GetSeries(ctx context.Context, id int64) (*Series, error) {
	row, err := s.queries.GetSeriesWithAddedBy(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSeriesNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	series := s.getSeriesRowToSeries(row)

	// Get seasons
	seasons, err := s.ListSeasons(ctx, id)
	if err == nil {
		series.Seasons = seasons
	}

	s.enrichSeriesWithCounts(ctx, series)

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

	switch {
	case opts.Search != "":
		searchTerm := "%" + opts.Search + "%"
		rows, err = s.queries.SearchSeries(ctx, sqlc.SearchSeriesParams{
			SearchTerm: searchTerm,
			Lim:        1000,
			Off:        0,
		})
	case opts.RootFolderID != nil:
		rows, err = s.queries.ListSeriesByRootFolder(ctx, sql.NullInt64{Int64: *opts.RootFolderID, Valid: true})
	case opts.Monitored != nil && *opts.Monitored:
		rows, err = s.queries.ListMonitoredSeries(ctx)
	default:
		rows, err = s.queries.ListSeries(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list series: %w", err)
	}

	seriesList := make([]*Series, len(rows))
	for i, row := range rows {
		seriesList[i] = s.rowToSeries(row)
		s.enrichSeriesWithCounts(ctx, seriesList[i])
	}
	return seriesList, nil
}

// ListUnmatchedByRootFolder returns series without metadata (no TVDB/TMDB ID) in a root folder.
func (s *Service) ListUnmatchedByRootFolder(ctx context.Context, rootFolderID int64) ([]*Series, error) {
	rows, err := s.queries.ListUnmatchedSeriesByRootFolder(ctx, sql.NullInt64{Int64: rootFolderID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list unmatched series: %w", err)
	}

	seriesList := make([]*Series, len(rows))
	for i, row := range rows {
		seriesList[i] = s.rowToSeries(row)
	}
	return seriesList, nil
}

// CreateSeries creates a new series.
func (s *Service) CreateSeries(ctx context.Context, input *CreateSeriesInput) (*Series, error) {
	if input.Title == "" {
		return nil, ErrInvalidSeries
	}

	input.Path = pathutil.NormalizePath(input.Path)

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

	productionStatus := input.ProductionStatus
	if productionStatus == "" {
		productionStatus = "continuing"
	}

	var addedBy sql.NullInt64
	if input.AddedBy != nil {
		addedBy = sql.NullInt64{Int64: *input.AddedBy, Valid: true}
	}

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
		ProductionStatus: productionStatus,
		Network:          sql.NullString{String: input.Network, Valid: input.Network != ""},
		FormatType:       sql.NullString{String: input.FormatType, Valid: input.FormatType != ""},
		NetworkLogoUrl:   sql.NullString{String: input.NetworkLogoURL, Valid: input.NetworkLogoURL != ""},
		AddedBy:          addedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create series: %w", err)
	}

	s.createSeasonsAndEpisodes(ctx, row.ID, input.Seasons)

	series := s.rowToSeries(row)
	s.enrichSeriesWithCounts(ctx, series)

	s.logger.Info().Int64("id", series.ID).Str("title", series.Title).Msg("Created series")

	if s.hub != nil {
		s.hub.Broadcast("series:added", series)
	}

	// Dispatch notification
	if s.notifier != nil {
		s.notifier.DispatchSeriesAdded(ctx, &SeriesNotificationInfo{
			ID:       series.ID,
			Title:    series.Title,
			Year:     series.Year,
			TvdbID:   series.TvdbID,
			TmdbID:   series.TmdbID,
			ImdbID:   series.ImdbID,
			Overview: series.Overview,
		}, time.Now())
	}

	return series, nil
}

// UpdateSeries updates an existing series.
func (s *Service) UpdateSeries(ctx context.Context, id int64, input *UpdateSeriesInput) (*Series, error) {
	current, err := s.GetSeries(ctx, id)
	if err != nil {
		return nil, err
	}

	params := s.buildSeriesUpdateParams(id, current, input)

	row, err := s.queries.UpdateSeries(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update series: %w", err)
	}

	s.cascadeMonitoringChanges(ctx, id, current, input)

	series := s.rowToSeries(row)
	s.logger.Info().Int64("id", id).Str("title", series.Title).Msg("Updated series")

	if s.hub != nil {
		s.hub.Broadcast("series:updated", series)
	}

	return series, nil
}

// BulkUpdateSeriesMonitored updates the monitored flag for multiple series,
// cascading the change to all their seasons and episodes.
func (s *Service) BulkUpdateSeriesMonitored(ctx context.Context, input BulkSeriesMonitorInput) error {
	if len(input.IDs) == 0 {
		return nil
	}

	monitored := boolToInt(input.Monitored)

	if err := s.queries.UpdateSeriesMonitoredByIDs(ctx, sqlc.UpdateSeriesMonitoredByIDsParams{
		Monitored: monitored,
		Ids:       input.IDs,
	}); err != nil {
		return fmt.Errorf("failed to bulk update series monitored: %w", err)
	}

	if err := s.queries.UpdateSeasonMonitoredBySeriesIDs(ctx, sqlc.UpdateSeasonMonitoredBySeriesIDsParams{
		Monitored: monitored,
		Ids:       input.IDs,
	}); err != nil {
		return fmt.Errorf("failed to bulk update season monitored: %w", err)
	}

	if err := s.queries.UpdateAllEpisodesMonitoredBySeriesIDs(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesIDsParams{
		Monitored: monitored,
		Ids:       input.IDs,
	}); err != nil {
		return fmt.Errorf("failed to bulk update episode monitored: %w", err)
	}

	s.logger.Info().Int("count", len(input.IDs)).Bool("monitored", input.Monitored).Msg("Bulk updated series monitored status")

	if s.hub != nil {
		s.hub.Broadcast("library:updated", nil)
	}

	return nil
}

// DeleteSeries deletes a series and all its seasons/episodes.
func (s *Service) DeleteSeries(ctx context.Context, id int64, deleteFiles bool) error {
	series, err := s.GetSeries(ctx, id)
	if err != nil {
		return err
	}

	// Clean up download mappings for this series to prevent seeding torrents from re-triggering imports
	if err := s.queries.DeleteDownloadMappingsBySeriesID(ctx, sql.NullInt64{Int64: id, Valid: true}); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to delete download mappings for series")
	}

	// Clean up autosearch backoff records (must happen before episodes are deleted
	// since the episode query uses a subquery on the episodes table)
	if err := s.queries.DeleteAutosearchStatusForSeriesEpisodes(ctx, id); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to delete autosearch status for series episodes")
	}
	if err := s.queries.DeleteAutosearchStatus(ctx, sqlc.DeleteAutosearchStatusParams{ItemType: "series", ItemID: id}); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to delete autosearch status for series")
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

	// Dispatch notification
	if s.notifier != nil {
		s.notifier.DispatchSeriesDeleted(ctx, &SeriesNotificationInfo{
			ID:       series.ID,
			Title:    series.Title,
			Year:     series.Year,
			TvdbID:   series.TvdbID,
			TmdbID:   series.TmdbID,
			ImdbID:   series.ImdbID,
			Overview: series.Overview,
		}, deleteFiles, time.Now())
	}

	return nil
}

// Count returns the total number of series.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.queries.CountSeries(ctx)
}

// GetSeriesIDByTvdbID returns the internal series ID for a given TVDB ID.
func (s *Service) GetSeriesIDByTvdbID(ctx context.Context, tvdbID int64) (int64, error) {
	series, err := s.GetSeriesByTvdbID(ctx, int(tvdbID))
	if err != nil {
		return 0, err
	}
	return series.ID, nil
}

// rowToSeries converts a database row to a Series.
func (s *Service) rowToSeries(row *sqlc.Series) *Series {
	series := &Series{
		ID:               row.ID,
		Title:            row.Title,
		SortTitle:        row.SortTitle,
		Monitored:        row.Monitored == 1,
		SeasonFolder:     row.SeasonFolder == 1,
		ProductionStatus: row.ProductionStatus,
	}

	s.mapSeriesNullableFields(series, row)
	return series
}

func (s *Service) mapSeriesNullableFields(series *Series, row *sqlc.Series) {
	s.mapSeriesCoreFields(series, row)
	s.mapSeriesDisplayFields(series, row)
}

func (s *Service) mapSeriesCoreFields(series *Series, row *sqlc.Series) {
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
}

func (s *Service) mapSeriesDisplayFields(series *Series, row *sqlc.Series) {
	if row.AddedAt.Valid {
		series.AddedAt = row.AddedAt.Time
	}
	if row.UpdatedAt.Valid {
		series.UpdatedAt = row.UpdatedAt.Time
	}
	if row.Network.Valid {
		series.Network = row.Network.String
	}
	if row.NetworkLogoUrl.Valid {
		series.NetworkLogoURL = row.NetworkLogoUrl.String
	}
	if row.FormatType.Valid {
		series.FormatType = row.FormatType.String
	}
	if row.AddedBy.Valid {
		v := row.AddedBy.Int64
		series.AddedBy = &v
	}
}

// getSeriesRowToSeries converts a GetSeriesWithAddedByRow (with JOIN) to a Series.
func (s *Service) getSeriesRowToSeries(row *sqlc.GetSeriesWithAddedByRow) *Series {
	series := s.rowToSeries(&sqlc.Series{
		ID:               row.ID,
		Title:            row.Title,
		SortTitle:        row.SortTitle,
		Year:             row.Year,
		TvdbID:           row.TvdbID,
		TmdbID:           row.TmdbID,
		ImdbID:           row.ImdbID,
		Overview:         row.Overview,
		Runtime:          row.Runtime,
		Path:             row.Path,
		RootFolderID:     row.RootFolderID,
		QualityProfileID: row.QualityProfileID,
		Monitored:        row.Monitored,
		SeasonFolder:     row.SeasonFolder,
		ProductionStatus: row.ProductionStatus,
		Network:          row.Network,
		FormatType:       row.FormatType,
		AddedAt:          row.AddedAt,
		UpdatedAt:        row.UpdatedAt,
		NetworkLogoUrl:   row.NetworkLogoUrl,
		AddedBy:          row.AddedBy,
	})
	if row.AddedByUsername.Valid {
		series.AddedByUsername = row.AddedByUsername.String
	}
	return series
}

// enrichSeriesWithCounts populates the StatusCounts and air date fields on a series by querying episode statuses.
func (s *Service) enrichSeriesWithCounts(ctx context.Context, series *Series) {
	counts, err := s.queries.GetEpisodeStatusCountsBySeries(ctx, series.ID)
	if err != nil {
		return
	}
	series.StatusCounts = StatusCounts{
		Unreleased:  toInt(counts.Unreleased),
		Missing:     toInt(counts.Missing),
		Downloading: toInt(counts.Downloading),
		Failed:      toInt(counts.Failed),
		Upgradable:  toInt(counts.Upgradable),
		Available:   toInt(counts.Available),
		Total:       int(counts.Total),
	}
	series.FirstAired = toTimePtr(counts.FirstAired)
	series.LastAired = toTimePtr(counts.LastAired)
	series.NextAiring = toTimePtr(counts.NextAiring)
}

// toInt safely converts a COALESCE result (interface{}) to int.
func toInt(v interface{}) int {
	switch n := v.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

// toTimePtr converts a nullable aggregate date result to *time.Time.
func toTimePtr(v interface{}) *time.Time {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case time.Time:
		return &t
	case string:
		if parsed, err := parseAirDate(t); err == nil {
			return &parsed
		}
	}
	return nil
}

// computeEpisodeStatus determines the initial status for an episode based on its air date.
func computeEpisodeStatus(airDate *time.Time) string {
	if airDate == nil || airDate.After(time.Now()) {
		return "unreleased"
	}
	return "missing"
}

// GenerateSeriesPath generates a path for a series.
// Returns a path with forward slashes for consistency across platforms.
func GenerateSeriesPath(rootPath, title string) string {
	return filepath.ToSlash(filepath.Join(rootPath, title))
}

// GenerateSeasonPath generates a path for a season folder.
// Returns a path with forward slashes for consistency across platforms.
func GenerateSeasonPath(seriesPath string, seasonNumber int) string {
	return filepath.ToSlash(filepath.Join(seriesPath, fmt.Sprintf("Season %02d", seasonNumber)))
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// parseAirDate attempts to parse a date string in common formats.
func parseAirDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 +0000 UTC",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"January 2, 2006",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date: %s", s)
}

// SeasonMetadata represents season metadata from a provider.
type SeasonMetadata struct {
	SeasonNumber int
	Name         string
	Overview     string
	PosterURL    string
	AirDate      string
	Episodes     []EpisodeMetadata
}

// EpisodeMetadata represents episode metadata from a provider.
type EpisodeMetadata struct {
	EpisodeNumber int
	SeasonNumber  int
	Title         string
	Overview      string
	AirDate       string
	Runtime       int
}

// UpdateSeasonsFromMetadata updates all seasons and episodes from metadata.
func (s *Service) UpdateSeasonsFromMetadata(ctx context.Context, seriesID int64, seasons []SeasonMetadata) error {
	for _, seasonMeta := range seasons {
		_, err := s.queries.UpsertSeason(ctx, sqlc.UpsertSeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: int64(seasonMeta.SeasonNumber),
			Monitored:    1,
			Overview:     sql.NullString{String: seasonMeta.Overview, Valid: seasonMeta.Overview != ""},
			PosterUrl:    sql.NullString{String: seasonMeta.PosterURL, Valid: seasonMeta.PosterURL != ""},
		})
		if err != nil {
			s.logger.Warn().
				Err(err).
				Int64("seriesId", seriesID).
				Int("seasonNumber", seasonMeta.SeasonNumber).
				Msg("Failed to upsert season")
			continue
		}

		s.upsertEpisodesForSeason(ctx, seriesID, seasonMeta.Episodes)
	}

	s.logger.Info().
		Int64("seriesId", seriesID).
		Int("seasons", len(seasons)).
		Msg("Updated seasons from metadata")

	return nil
}

func (s *Service) upsertEpisodesForSeason(ctx context.Context, seriesID int64, episodes []EpisodeMetadata) {
	for _, epMeta := range episodes {
		var airDate sql.NullTime
		if epMeta.AirDate != "" {
			if t, err := parseAirDate(epMeta.AirDate); err == nil {
				airDate = sql.NullTime{Time: t, Valid: true}
			}
		}

		var airDatePtr *time.Time
		if airDate.Valid {
			airDatePtr = &airDate.Time
		}
		status := computeEpisodeStatus(airDatePtr)

		_, err := s.queries.UpsertEpisode(ctx, sqlc.UpsertEpisodeParams{
			SeriesID:      seriesID,
			SeasonNumber:  int64(epMeta.SeasonNumber),
			EpisodeNumber: int64(epMeta.EpisodeNumber),
			Title:         sql.NullString{String: epMeta.Title, Valid: epMeta.Title != ""},
			Overview:      sql.NullString{String: epMeta.Overview, Valid: epMeta.Overview != ""},
			AirDate:       airDate,
			Monitored:     1,
			Status:        status,
		})
		if err != nil {
			s.logger.Warn().
				Err(err).
				Int64("seriesId", seriesID).
				Int("seasonNumber", epMeta.SeasonNumber).
				Int("episodeNumber", epMeta.EpisodeNumber).
				Msg("Failed to upsert episode")
		}
	}
}

func (s *Service) createSeasonsAndEpisodes(ctx context.Context, seriesID int64, seasons []SeasonInput) {
	for _, seasonInput := range seasons {
		_, err := s.queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
			SeriesID:     seriesID,
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
			status := computeEpisodeStatus(episodeInput.AirDate)
			_, err := s.queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
				SeriesID:      seriesID,
				SeasonNumber:  int64(seasonInput.SeasonNumber),
				EpisodeNumber: int64(episodeInput.EpisodeNumber),
				Title:         sql.NullString{String: episodeInput.Title, Valid: episodeInput.Title != ""},
				Overview:      sql.NullString{String: episodeInput.Overview, Valid: episodeInput.Overview != ""},
				AirDate:       airDate,
				Monitored:     boolToInt(episodeInput.Monitored),
				Status:        status,
			})
			if err != nil {
				s.logger.Warn().Err(err).Int("episode", episodeInput.EpisodeNumber).Msg("Failed to create episode")
			}
		}
	}
}

func (s *Service) buildSeriesUpdateParams(id int64, current *Series, input *UpdateSeriesInput) sqlc.UpdateSeriesParams {
	title := resolveField(current.Title, input.Title)
	year := resolveField(current.Year, input.Year)
	tvdbID := resolveField(current.TvdbID, input.TvdbID)
	tmdbID := resolveField(current.TmdbID, input.TmdbID)
	imdbID := resolveField(current.ImdbID, input.ImdbID)
	overview := resolveField(current.Overview, input.Overview)
	runtime := resolveField(current.Runtime, input.Runtime)
	path := resolveField(current.Path, input.Path)
	rootFolderID := resolveField(current.RootFolderID, input.RootFolderID)
	qualityProfileID := resolveField(current.QualityProfileID, input.QualityProfileID)
	monitored := resolveField(current.Monitored, input.Monitored)
	seasonFolder := resolveField(current.SeasonFolder, input.SeasonFolder)
	productionStatus := resolveField(current.ProductionStatus, input.ProductionStatus)
	formatType := resolveField(current.FormatType, input.FormatType)
	network := resolveField(current.Network, input.Network)
	networkLogoURL := resolveField(current.NetworkLogoURL, input.NetworkLogoURL)

	return sqlc.UpdateSeriesParams{
		ID:               id,
		Title:            title,
		SortTitle:        generateSortTitle(title),
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
		ProductionStatus: productionStatus,
		Network:          sql.NullString{String: network, Valid: network != ""},
		FormatType:       sql.NullString{String: formatType, Valid: formatType != ""},
		NetworkLogoUrl:   sql.NullString{String: networkLogoURL, Valid: networkLogoURL != ""},
	}
}

func resolveField[T any](current T, input *T) T {
	if input != nil {
		return *input
	}
	return current
}

func (s *Service) cascadeMonitoringChanges(ctx context.Context, id int64, current *Series, input *UpdateSeriesInput) {
	if input.Monitored == nil || *input.Monitored == current.Monitored {
		return
	}

	monitoredInt := boolToInt(*input.Monitored)
	if err := s.queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: monitoredInt,
		SeriesID:  id,
	}); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to cascade monitoring to seasons")
	}
	if err := s.queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: monitoredInt,
		SeriesID:  id,
	}); err != nil {
		s.logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to cascade monitoring to episodes")
	}
}
