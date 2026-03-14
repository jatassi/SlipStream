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
	"github.com/slipstream/slipstream/internal/domain/contracts"
	"github.com/slipstream/slipstream/internal/library"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/metadata"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
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

// Service provides TV library operations.
type Service struct {
	module.BaseService
	fileDeleteHandler contracts.FileDeleteHandler
	notifier          NotificationDispatcher
	registry          *module.Registry
}

// SetRegistry sets the module registry for cascading monitoring changes.
func (s *Service) SetRegistry(r *module.Registry) {
	s.registry = r
}

// SetNotificationDispatcher sets the notification dispatcher for series events.
func (s *Service) SetNotificationDispatcher(n NotificationDispatcher) {
	s.notifier = n
}

// SetFileDeleteHandler sets the handler for file deletion events.
// Req 12.1.1: Deleting file from slot does NOT trigger automatic search
func (s *Service) SetFileDeleteHandler(handler contracts.FileDeleteHandler) {
	s.fileDeleteHandler = handler
}

// NewService creates a new TV service.
func NewService(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger, qualityService *quality.Service, statusChangeLogger contracts.StatusChangeLogger) *Service {
	return &Service{
		BaseService: module.NewBaseService(db, hub, logger, qualityService, statusChangeLogger, "tv"),
	}
}

// GetSeries retrieves a series by ID.
func (s *Service) GetSeries(ctx context.Context, id int64) (*Series, error) {
	row, err := s.Queries.GetSeriesWithAddedBy(ctx, id)
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
	row, err := s.Queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: int64(tvdbID), Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSeriesNotFound
		}
		return nil, fmt.Errorf("failed to get series: %w", err)
	}
	return s.rowToSeries(row), nil
}

// GetSeriesByPath retrieves a series by its filesystem path.
func (s *Service) GetSeriesByPath(ctx context.Context, path string) (*Series, error) {
	row, err := s.Queries.GetSeriesByPath(ctx, sql.NullString{String: path, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSeriesNotFound
		}
		return nil, fmt.Errorf("failed to get series by path: %w", err)
	}
	return s.rowToSeries(row), nil
}

// FindByTitle searches for a series by normalized title and optional year.
// Returns nil, nil if no match is found.
func (s *Service) FindByTitle(ctx context.Context, title string, year int) (*Series, error) {
	allSeries, err := s.ListSeries(ctx, ListSeriesOptions{})
	if err != nil {
		return nil, err
	}

	normalizedSearch := parseutil.NormalizeTitle(title)
	for _, series := range allSeries {
		if parseutil.NormalizeTitle(series.Title) == normalizedSearch {
			if year == 0 || series.Year == year {
				return series, nil
			}
		}
	}
	return nil, nil //nolint:nilnil // nil means no match found
}

// ListSeries returns series with optional filtering.
func (s *Service) ListSeries(ctx context.Context, opts ListSeriesOptions) ([]*Series, error) {
	var rows []*sqlc.Series
	var err error

	switch {
	case opts.Search != "":
		searchTerm := "%" + opts.Search + "%"
		rows, err = s.Queries.SearchSeries(ctx, sqlc.SearchSeriesParams{
			SearchTerm: searchTerm,
			Lim:        1000,
			Off:        0,
		})
	case opts.RootFolderID != nil:
		rows, err = s.Queries.ListSeriesByRootFolder(ctx, sql.NullInt64{Int64: *opts.RootFolderID, Valid: true})
	case opts.Monitored != nil && *opts.Monitored:
		rows, err = s.Queries.ListMonitoredSeries(ctx)
	default:
		rows, err = s.Queries.ListSeries(ctx)
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
	rows, err := s.Queries.ListUnmatchedSeriesByRootFolder(ctx, sql.NullInt64{Int64: rootFolderID, Valid: true})
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

	sortTitle := module.GenerateSortTitle(input.Title)

	productionStatus := input.ProductionStatus
	if productionStatus == "" {
		productionStatus = "continuing"
	}

	var addedBy sql.NullInt64
	if input.AddedBy != nil {
		addedBy = sql.NullInt64{Int64: *input.AddedBy, Valid: true}
	}

	row, err := s.Queries.CreateSeries(ctx, sqlc.CreateSeriesParams{
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
		Monitored:        input.Monitored,
		SeasonFolder:     input.SeasonFolder,
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

	s.Logger.Info().Int64("id", series.ID).Str("title", series.Title).Msg("Created series")

	s.BroadcastEntity("tv", "series", series.ID, "added", series)

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

	row, err := s.Queries.UpdateSeries(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update series: %w", err)
	}

	if input.Monitored != nil && *input.Monitored != current.Monitored && s.registry != nil {
		if err := module.CascadeMonitoredForModule(ctx, s.registry, module.TypeTV, module.EntitySeries, id, *input.Monitored); err != nil {
			s.Logger.Warn().Err(err).Int64("seriesId", id).Msg("cascade monitoring failed")
		}
	}

	series := s.rowToSeries(row)
	s.Logger.Info().Int64("id", id).Str("title", series.Title).Msg("Updated series")

	s.BroadcastEntity("tv", "series", series.ID, "updated", series)

	return series, nil
}

// BulkUpdateSeriesMonitored updates the monitored flag for multiple series,
// cascading the change to all their seasons and episodes.
func (s *Service) BulkUpdateSeriesMonitored(ctx context.Context, input BulkSeriesMonitorInput) error {
	if len(input.IDs) == 0 {
		return nil
	}

	if err := s.Queries.UpdateSeriesMonitoredByIDs(ctx, sqlc.UpdateSeriesMonitoredByIDsParams{
		Monitored: input.Monitored,
		Ids:       input.IDs,
	}); err != nil {
		return fmt.Errorf("failed to bulk update series monitored: %w", err)
	}

	if s.registry != nil {
		for _, id := range input.IDs {
			if err := module.CascadeMonitoredForModule(ctx, s.registry, module.TypeTV, module.EntitySeries, id, input.Monitored); err != nil {
				s.Logger.Warn().Err(err).Int64("seriesId", id).Msg("cascade monitoring failed")
			}
		}
	}

	s.Logger.Info().Int("count", len(input.IDs)).Bool("monitored", input.Monitored).Msg("Bulk updated series monitored status")

	s.Broadcast("library:updated", nil)

	return nil
}

// DeleteSeries deletes a series and all its seasons/episodes.
func (s *Service) DeleteSeries(ctx context.Context, id int64, deleteFiles bool) error {
	series, err := s.GetSeries(ctx, id)
	if err != nil {
		return err
	}

	s.cleanupSeriesRelatedData(ctx, id)

	// Delete files from disk before removing DB records
	if deleteFiles {
		if err := s.deleteSeriesFilesFromDisk(ctx, id); err != nil {
			return err
		}
	}

	if err := s.deleteSeriesDBRecords(ctx, id); err != nil {
		return err
	}

	s.Logger.Info().Int64("id", id).Str("title", series.Title).Msg("Deleted series")

	s.BroadcastEntity("tv", "series", id, "deleted", nil)

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

func (s *Service) cleanupSeriesRelatedData(ctx context.Context, id int64) {
	// Episode-level autosearch uses a subquery on the episodes table, so it must
	// run before episodes are deleted from the DB.
	if err := s.Queries.DeleteAutosearchStatusForSeriesEpisodes(ctx, id); err != nil {
		s.Logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to delete autosearch status for series episodes")
	}

	// Bulk-delete episode-level shared table records using subqueries against the
	// episodes table (must run before episodes are deleted). These replace the
	// ON DELETE CASCADE that was previously on download_mappings/queue_media FKs.
	episodeSubquery := "SELECT id FROM episodes WHERE series_id = ?"
	episodeDeletes := []string{
		"DELETE FROM queue_media WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id IN (" + episodeSubquery + ")",
		"DELETE FROM download_mappings WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id IN (" + episodeSubquery + ")",
		"DELETE FROM downloads WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id IN (" + episodeSubquery + ")",
		"DELETE FROM history WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id IN (" + episodeSubquery + ")",
		"DELETE FROM import_decisions WHERE module_type = 'tv' AND entity_type = 'episode' AND entity_id IN (" + episodeSubquery + ")",
	}
	for _, q := range episodeDeletes {
		if _, err := s.DB.ExecContext(ctx, q, id); err != nil {
			s.Logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to delete episode-level shared records")
		}
	}

	// Clean up series-level shared table records (download_mappings, queue_media,
	// downloads, history, autosearch_status, import_decisions, requests).
	if err := module.DeleteEntity(ctx, s.DB, module.TypeTV, module.EntitySeries, id); err != nil {
		s.Logger.Warn().Err(err).Int64("seriesId", id).Msg("Failed to delete shared table records for series")
	}
}

func (s *Service) deleteSeriesDBRecords(ctx context.Context, id int64) error {
	if err := s.Queries.DeleteEpisodesBySeries(ctx, id); err != nil {
		return fmt.Errorf("failed to delete episodes: %w", err)
	}
	if err := s.Queries.DeleteSeasonsBySeries(ctx, id); err != nil {
		return fmt.Errorf("failed to delete seasons: %w", err)
	}
	if err := s.Queries.DeleteSeries(ctx, id); err != nil {
		return fmt.Errorf("failed to delete series: %w", err)
	}
	return nil
}

func (s *Service) deleteSeriesFilesFromDisk(ctx context.Context, seriesID int64) error {
	files, err := s.Queries.ListEpisodeFilesBySeries(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("failed to list episode files: %w", err)
	}

	paths := make([]string, 0, len(files))
	for _, f := range files {
		if f.Path != "" {
			paths = append(paths, f.Path)
		}
	}

	if err := library.CheckDeletable(paths); err != nil {
		return fmt.Errorf("cannot delete episode files: %w", err)
	}

	deleted, err := library.DeleteFiles(paths)
	if err != nil {
		return fmt.Errorf("failed to delete episode files from disk: %w", err)
	}
	if deleted > 0 {
		s.Logger.Info().Int("count", deleted).Int64("seriesId", seriesID).Msg("Deleted episode files from disk")
	}
	return nil
}

// Count returns the total number of series.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.Queries.CountSeries(ctx)
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
		Monitored:        row.Monitored,
		SeasonFolder:     row.SeasonFolder,
		ProductionStatus: row.ProductionStatus,
		Seasons:          []Season{},
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
	counts, err := s.Queries.GetEpisodeStatusCountsBySeries(ctx, series.ID)
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

// ConvertSeasonResults converts metadata season results to the library SeasonMetadata type.
func ConvertSeasonResults(seasonResults []metadata.SeasonResult) []SeasonMetadata {
	seasonMeta := make([]SeasonMetadata, len(seasonResults))
	for i, sr := range seasonResults {
		episodes := make([]EpisodeMetadata, len(sr.Episodes))
		for j, ep := range sr.Episodes {
			episodes[j] = EpisodeMetadata{
				EpisodeNumber: ep.EpisodeNumber,
				SeasonNumber:  ep.SeasonNumber,
				Title:         ep.Title,
				Overview:      ep.Overview,
				AirDate:       ep.AirDate,
				Runtime:       ep.Runtime,
			}
		}
		seasonMeta[i] = SeasonMetadata{
			SeasonNumber: sr.SeasonNumber,
			Name:         sr.Name,
			Overview:     sr.Overview,
			PosterURL:    sr.PosterURL,
			AirDate:      sr.AirDate,
			Episodes:     episodes,
		}
	}
	return seasonMeta
}

// UpdateSeasonsFromMetadata updates all seasons and episodes from metadata.
func (s *Service) UpdateSeasonsFromMetadata(ctx context.Context, seriesID int64, seasons []SeasonMetadata) error {
	for _, seasonMeta := range seasons {
		_, err := s.Queries.UpsertSeason(ctx, sqlc.UpsertSeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: int64(seasonMeta.SeasonNumber),
			Monitored:    true,
			Overview:     sql.NullString{String: seasonMeta.Overview, Valid: seasonMeta.Overview != ""},
			PosterUrl:    sql.NullString{String: seasonMeta.PosterURL, Valid: seasonMeta.PosterURL != ""},
		})
		if err != nil {
			s.Logger.Warn().
				Err(err).
				Int64("seriesId", seriesID).
				Int("seasonNumber", seasonMeta.SeasonNumber).
				Msg("Failed to upsert season")
			continue
		}

		s.upsertEpisodesForSeason(ctx, seriesID, seasonMeta.Episodes)
	}

	s.Logger.Info().
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

		_, err := s.Queries.UpsertEpisode(ctx, sqlc.UpsertEpisodeParams{
			SeriesID:      seriesID,
			SeasonNumber:  int64(epMeta.SeasonNumber),
			EpisodeNumber: int64(epMeta.EpisodeNumber),
			Title:         sql.NullString{String: epMeta.Title, Valid: epMeta.Title != ""},
			Overview:      sql.NullString{String: epMeta.Overview, Valid: epMeta.Overview != ""},
			AirDate:       airDate,
			Monitored:     true,
			Status:        status,
		})
		if err != nil {
			s.Logger.Warn().
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
		_, err := s.Queries.CreateSeason(ctx, sqlc.CreateSeasonParams{
			SeriesID:     seriesID,
			SeasonNumber: int64(seasonInput.SeasonNumber),
			Monitored:    seasonInput.Monitored,
		})
		if err != nil {
			s.Logger.Warn().Err(err).Int("season", seasonInput.SeasonNumber).Msg("Failed to create season")
			continue
		}

		for _, episodeInput := range seasonInput.Episodes {
			var airDate sql.NullTime
			if episodeInput.AirDate != nil {
				airDate = sql.NullTime{Time: *episodeInput.AirDate, Valid: true}
			}
			status := computeEpisodeStatus(episodeInput.AirDate)
			_, err := s.Queries.CreateEpisode(ctx, sqlc.CreateEpisodeParams{
				SeriesID:      seriesID,
				SeasonNumber:  int64(seasonInput.SeasonNumber),
				EpisodeNumber: int64(episodeInput.EpisodeNumber),
				Title:         sql.NullString{String: episodeInput.Title, Valid: episodeInput.Title != ""},
				Overview:      sql.NullString{String: episodeInput.Overview, Valid: episodeInput.Overview != ""},
				AirDate:       airDate,
				Monitored:     episodeInput.Monitored,
				Status:        status,
			})
			if err != nil {
				s.Logger.Warn().Err(err).Int("episode", episodeInput.EpisodeNumber).Msg("Failed to create episode")
			}
		}
	}
}

func (s *Service) buildSeriesUpdateParams(id int64, current *Series, input *UpdateSeriesInput) sqlc.UpdateSeriesParams {
	title := module.ResolveField(current.Title, input.Title)
	year := module.ResolveField(current.Year, input.Year)
	tvdbID := module.ResolveField(current.TvdbID, input.TvdbID)
	tmdbID := module.ResolveField(current.TmdbID, input.TmdbID)
	imdbID := module.ResolveField(current.ImdbID, input.ImdbID)
	overview := module.ResolveField(current.Overview, input.Overview)
	runtime := module.ResolveField(current.Runtime, input.Runtime)
	path := module.ResolveField(current.Path, input.Path)
	rootFolderID := module.ResolveField(current.RootFolderID, input.RootFolderID)
	qualityProfileID := module.ResolveField(current.QualityProfileID, input.QualityProfileID)
	monitored := module.ResolveField(current.Monitored, input.Monitored)
	seasonFolder := module.ResolveField(current.SeasonFolder, input.SeasonFolder)
	productionStatus := module.ResolveField(current.ProductionStatus, input.ProductionStatus)
	formatType := module.ResolveField(current.FormatType, input.FormatType)
	network := module.ResolveField(current.Network, input.Network)
	networkLogoURL := module.ResolveField(current.NetworkLogoURL, input.NetworkLogoURL)

	return sqlc.UpdateSeriesParams{
		ID:               id,
		Title:            title,
		SortTitle:        module.GenerateSortTitle(title),
		Year:             sql.NullInt64{Int64: int64(year), Valid: year > 0},
		TvdbID:           sql.NullInt64{Int64: int64(tvdbID), Valid: tvdbID > 0},
		TmdbID:           sql.NullInt64{Int64: int64(tmdbID), Valid: tmdbID > 0},
		ImdbID:           sql.NullString{String: imdbID, Valid: imdbID != ""},
		Overview:         sql.NullString{String: overview, Valid: overview != ""},
		Runtime:          sql.NullInt64{Int64: int64(runtime), Valid: runtime > 0},
		Path:             sql.NullString{String: path, Valid: path != ""},
		RootFolderID:     sql.NullInt64{Int64: rootFolderID, Valid: rootFolderID > 0},
		QualityProfileID: sql.NullInt64{Int64: qualityProfileID, Valid: qualityProfileID > 0},
		Monitored:        monitored,
		SeasonFolder:     seasonFolder,
		ProductionStatus: productionStatus,
		Network:          sql.NullString{String: network, Valid: network != ""},
		FormatType:       sql.NullString{String: formatType, Valid: formatType != ""},
		NetworkLogoUrl:   sql.NullString{String: networkLogoURL, Valid: networkLogoURL != ""},
	}
}

// CascadeSeriesMonitored propagates a monitored value from a series to all its seasons and episodes.
func (s *Service) CascadeSeriesMonitored(ctx context.Context, seriesID int64, monitored bool) error {
	if err := s.Queries.UpdateSeasonMonitoredBySeries(ctx, sqlc.UpdateSeasonMonitoredBySeriesParams{
		Monitored: monitored,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to cascade monitoring to seasons: %w", err)
	}
	if err := s.Queries.UpdateAllEpisodesMonitoredBySeries(ctx, sqlc.UpdateAllEpisodesMonitoredBySeriesParams{
		Monitored: monitored,
		SeriesID:  seriesID,
	}); err != nil {
		return fmt.Errorf("failed to cascade monitoring to episodes: %w", err)
	}
	return nil
}

// CascadeSeasonMonitored propagates a monitored value from a season to all its episodes.
func (s *Service) CascadeSeasonMonitored(ctx context.Context, seasonID int64, monitored bool) error {
	season, err := s.Queries.GetSeason(ctx, seasonID)
	if err != nil {
		return fmt.Errorf("failed to get season: %w", err)
	}
	if err := s.Queries.UpdateEpisodesMonitoredBySeason(ctx, sqlc.UpdateEpisodesMonitoredBySeasonParams{
		Monitored:    monitored,
		SeriesID:     season.SeriesID,
		SeasonNumber: season.SeasonNumber,
	}); err != nil {
		return fmt.Errorf("failed to cascade monitoring to episodes: %w", err)
	}
	return nil
}
