package movies

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
	"github.com/slipstream/slipstream/internal/library/status"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/parseutil"
	"github.com/slipstream/slipstream/internal/pathutil"
	"github.com/slipstream/slipstream/internal/websocket"
)

// NotificationDispatcher defines the interface for movie notifications.
type NotificationDispatcher interface {
	DispatchMovieAdded(ctx context.Context, movie *MovieNotificationInfo, addedAt time.Time)
	DispatchMovieDeleted(ctx context.Context, movie *MovieNotificationInfo, deletedFiles bool, deletedAt time.Time)
}

// MovieNotificationInfo contains movie info for notifications.
type MovieNotificationInfo struct {
	ID        int64
	Title     string
	Year      int
	TmdbID    int
	ImdbID    string
	Overview  string
	PosterURL string
}

var (
	ErrMovieNotFound     = errors.New("movie not found")
	ErrMovieFileNotFound = errors.New("movie file not found")
	ErrInvalidMovie      = errors.New("invalid movie data")
	ErrDuplicateTmdbID   = errors.New("movie with this TMDB ID already exists")
)

// Service provides movie library operations.
type Service struct {
	module.BaseService
	fileDeleteHandler contracts.FileDeleteHandler
	notifier          NotificationDispatcher
}

// SetNotificationDispatcher sets the notification dispatcher for movie events.
func (s *Service) SetNotificationDispatcher(n NotificationDispatcher) {
	s.notifier = n
}

// SetFileDeleteHandler sets the handler for file deletion events.
// Req 12.1.1: Deleting file from slot does NOT trigger automatic search
func (s *Service) SetFileDeleteHandler(handler contracts.FileDeleteHandler) {
	s.fileDeleteHandler = handler
}

// isMovieReleased determines if a movie should be considered released based on
// the priority chain: digital → physical → theatrical + 90 days.
func isMovieReleased(digital, physical, theatrical sql.NullTime) bool {
	now := time.Now()
	if digital.Valid && !digital.Time.After(now) {
		return true
	}
	if physical.Valid && !physical.Time.After(now) {
		return true
	}
	if theatrical.Valid && !theatrical.Time.AddDate(0, 0, 90).After(now) {
		return true
	}
	return false
}

// NewService creates a new movie service.
func NewService(db *sql.DB, hub *websocket.Hub, logger *zerolog.Logger, qualityService *quality.Service, statusChangeLogger contracts.StatusChangeLogger) *Service {
	return &Service{
		BaseService: module.NewBaseService(db, hub, logger, qualityService, statusChangeLogger, "movies"),
	}
}

// Get retrieves a movie by ID.
func (s *Service) Get(ctx context.Context, id int64) (*Movie, error) {
	row, err := s.Queries.GetMovieWithAddedBy(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMovieNotFound
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	movie := s.getMovieRowToMovie(row)

	// Get movie files
	files, err := s.GetFiles(ctx, id)
	if err != nil {
		s.Logger.Warn().Err(err).Int64("movieId", id).Msg("Failed to get movie files")
	} else {
		movie.MovieFiles = files
		for i := range files {
			f := &files[i]
			movie.SizeOnDisk += f.Size
		}
	}

	return movie, nil
}

// GetByTmdbID retrieves a movie by TMDB ID.
func (s *Service) GetByTmdbID(ctx context.Context, tmdbID int) (*Movie, error) {
	row, err := s.Queries.GetMovieByTmdbID(ctx, sql.NullInt64{Int64: int64(tmdbID), Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMovieNotFound
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}
	return s.rowToMovie(row), nil
}

// List returns movies with optional filtering.
func (s *Service) List(ctx context.Context, opts ListMoviesOptions) ([]*Movie, error) {
	var rows []*sqlc.Movie
	var err error

	switch {
	case opts.Search != "":
		searchTerm := "%" + opts.Search + "%"
		rows, err = s.Queries.SearchMovies(ctx, sqlc.SearchMoviesParams{
			SearchTerm: searchTerm,
			Lim:        1000,
			Off:        0,
		})
	case opts.RootFolderID != nil:
		rows, err = s.Queries.ListMoviesByRootFolder(ctx, sql.NullInt64{Int64: *opts.RootFolderID, Valid: true})
	case opts.Monitored != nil && *opts.Monitored:
		rows, err = s.Queries.ListMonitoredMovies(ctx)
	default:
		rows, err = s.Queries.ListMovies(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list movies: %w", err)
	}

	movies := make([]*Movie, len(rows))
	for i, row := range rows {
		movies[i] = s.rowToMovie(row)
	}
	return movies, nil
}

// ListUnmatchedByRootFolder returns movies without metadata (no TMDB ID) in a root folder.
func (s *Service) ListUnmatchedByRootFolder(ctx context.Context, rootFolderID int64) ([]*Movie, error) {
	rows, err := s.Queries.ListUnmatchedMoviesByRootFolder(ctx, sql.NullInt64{Int64: rootFolderID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list unmatched movies: %w", err)
	}

	movies := make([]*Movie, len(rows))
	for i, row := range rows {
		movies[i] = s.rowToMovie(row)
	}
	return movies, nil
}

// Create creates a new movie.
func (s *Service) Create(ctx context.Context, input *CreateMovieInput) (*Movie, error) {
	if input.Title == "" {
		return nil, ErrInvalidMovie
	}

	if err := s.checkDuplicateTmdbID(ctx, input.TmdbID); err != nil {
		return nil, err
	}

	sortTitle := module.GenerateSortTitle(input.Title)
	path := pathutil.NormalizePath(input.Path)

	releaseDate, physicalReleaseDate, theatricalReleaseDate := parseReleaseDates(input.ReleaseDate, input.PhysicalReleaseDate, input.TheatricalReleaseDate)

	st := status.Unreleased
	if isMovieReleased(releaseDate, physicalReleaseDate, theatricalReleaseDate) {
		st = status.Missing
	}

	var addedBy sql.NullInt64
	if input.AddedBy != nil {
		addedBy = sql.NullInt64{Int64: *input.AddedBy, Valid: true}
	}

	row, err := s.Queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:                 input.Title,
		SortTitle:             sortTitle,
		Year:                  sql.NullInt64{Int64: int64(input.Year), Valid: input.Year > 0},
		TmdbID:                sql.NullInt64{Int64: int64(input.TmdbID), Valid: input.TmdbID > 0},
		ImdbID:                sql.NullString{String: input.ImdbID, Valid: input.ImdbID != ""},
		Overview:              sql.NullString{String: input.Overview, Valid: input.Overview != ""},
		Runtime:               sql.NullInt64{Int64: int64(input.Runtime), Valid: input.Runtime > 0},
		Path:                  sql.NullString{String: path, Valid: path != ""},
		RootFolderID:          sql.NullInt64{Int64: input.RootFolderID, Valid: input.RootFolderID > 0},
		QualityProfileID:      sql.NullInt64{Int64: input.QualityProfileID, Valid: input.QualityProfileID > 0},
		Monitored:             input.Monitored,
		Status:                st,
		ReleaseDate:           releaseDate,
		PhysicalReleaseDate:   physicalReleaseDate,
		TheatricalReleaseDate: theatricalReleaseDate,
		Studio:                sql.NullString{String: input.Studio, Valid: input.Studio != ""},
		TvdbID:                sql.NullInt64{Int64: int64(input.TvdbID), Valid: input.TvdbID > 0},
		ContentRating:         sql.NullString{String: input.ContentRating, Valid: input.ContentRating != ""},
		AddedBy:               addedBy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create movie: %w", err)
	}

	movie := s.rowToMovie(row)
	s.Logger.Info().Int64("id", movie.ID).Str("title", movie.Title).Msg("Created movie")

	s.BroadcastEntity("movie", "movie", movie.ID, "added", movie)

	if s.notifier != nil {
		s.notifier.DispatchMovieAdded(ctx, &MovieNotificationInfo{
			ID:       movie.ID,
			Title:    movie.Title,
			Year:     movie.Year,
			TmdbID:   movie.TmdbID,
			ImdbID:   movie.ImdbID,
			Overview: movie.Overview,
		}, time.Now())
	}

	return movie, nil
}

// Update updates an existing movie.
func (s *Service) Update(ctx context.Context, id int64, input *UpdateMovieInput) (*Movie, error) {
	s.Logger.Debug().Int64("id", id).Msg("[UPDATE] Starting movie update")

	current, err := s.Get(ctx, id)
	if err != nil {
		s.Logger.Error().Err(err).Int64("id", id).Msg("[UPDATE] Failed to get current movie")
		return nil, err
	}

	params := s.buildMovieUpdateParams(id, current, input)

	row, err := s.Queries.UpdateMovie(ctx, params)
	if err != nil {
		s.Logger.Error().Err(err).Int64("id", id).Msg("[UPDATE] Database update failed")
		return nil, fmt.Errorf("failed to update movie: %w", err)
	}

	movie := s.rowToMovie(row)
	s.Logger.Info().
		Int64("id", id).
		Str("title", movie.Title).
		Int("tmdbId", movie.TmdbID).
		Str("imdbId", movie.ImdbID).
		Msg("[UPDATE] Movie updated successfully")

	s.BroadcastEntity("movie", "movie", movie.ID, "updated", movie)

	return movie, nil
}

// BulkUpdateMonitored updates the monitored flag for multiple movies at once.
func (s *Service) BulkUpdateMonitored(ctx context.Context, input BulkMonitorInput) error {
	if len(input.IDs) == 0 {
		return nil
	}

	if err := s.Queries.UpdateMoviesMonitoredByIDs(ctx, sqlc.UpdateMoviesMonitoredByIDsParams{
		Monitored: input.Monitored,
		Ids:       input.IDs,
	}); err != nil {
		return fmt.Errorf("failed to bulk update movie monitored: %w", err)
	}

	s.Logger.Info().Int("count", len(input.IDs)).Bool("monitored", input.Monitored).Msg("Bulk updated movie monitored status")

	s.Broadcast("library:updated", nil)

	return nil
}

// Delete deletes a movie.
func (s *Service) Delete(ctx context.Context, id int64, deleteFiles bool) error {
	movie, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Clean up all shared table records (download_mappings, queue_media, downloads,
	// history, autosearch_status, import_decisions, requests) for this movie.
	if err := module.DeleteEntity(ctx, s.DB, module.TypeMovie, module.EntityMovie, id); err != nil {
		s.Logger.Warn().Err(err).Int64("movieId", id).Msg("Failed to delete shared table records for movie")
	}

	// Delete files from disk before removing DB records
	if deleteFiles {
		if err := s.deleteMovieFilesFromDisk(ctx, id); err != nil {
			return err
		}
	}

	// Delete movie files from database
	if err := s.Queries.DeleteMovieFilesByMovie(ctx, id); err != nil {
		return fmt.Errorf("failed to delete movie files: %w", err)
	}

	if err := s.Queries.DeleteMovie(ctx, id); err != nil {
		return fmt.Errorf("failed to delete movie: %w", err)
	}

	s.Logger.Info().Int64("id", id).Str("title", movie.Title).Msg("Deleted movie")

	s.BroadcastEntity("movie", "movie", id, "deleted", nil)

	// Dispatch notification
	if s.notifier != nil {
		s.notifier.DispatchMovieDeleted(ctx, &MovieNotificationInfo{
			ID:       movie.ID,
			Title:    movie.Title,
			Year:     movie.Year,
			TmdbID:   movie.TmdbID,
			ImdbID:   movie.ImdbID,
			Overview: movie.Overview,
		}, deleteFiles, time.Now())
	}

	return nil
}

func (s *Service) deleteMovieFilesFromDisk(ctx context.Context, movieID int64) error {
	files, err := s.Queries.ListMovieFiles(ctx, movieID)
	if err != nil {
		return fmt.Errorf("failed to list movie files: %w", err)
	}

	paths := make([]string, 0, len(files))
	for _, f := range files {
		if f.Path != "" {
			paths = append(paths, f.Path)
		}
	}

	if err := library.CheckDeletable(paths); err != nil {
		return fmt.Errorf("cannot delete movie files: %w", err)
	}

	deleted, err := library.DeleteFiles(paths)
	if err != nil {
		return fmt.Errorf("failed to delete movie files from disk: %w", err)
	}
	if deleted > 0 {
		s.Logger.Info().Int("count", deleted).Int64("movieId", movieID).Msg("Deleted movie files from disk")
	}
	return nil
}

// GetFiles returns all files for a movie.
func (s *Service) GetFiles(ctx context.Context, movieID int64) ([]MovieFile, error) {
	rows, err := s.Queries.ListMovieFiles(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to list movie files: %w", err)
	}

	files := make([]MovieFile, len(rows))
	for i, row := range rows {
		files[i] = s.rowToMovieFile(row)
	}
	return files, nil
}

// GetPrimaryFile returns the primary (first) file for a movie.
// Returns sql.ErrNoRows if no files exist.
func (s *Service) GetPrimaryFile(ctx context.Context, movieID int64) (*MovieFile, error) {
	files, err := s.GetFiles(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, sql.ErrNoRows
	}
	return &files[0], nil
}

// AddFile adds a file to a movie.
func (s *Service) AddFile(ctx context.Context, movieID int64, input *CreateMovieFileInput) (*MovieFile, error) {
	movie, err := s.Get(ctx, movieID)
	if err != nil {
		return nil, err
	}

	input.Path = pathutil.NormalizePath(input.Path)

	qualityID := sql.NullInt64{}
	if input.QualityID != nil {
		qualityID = sql.NullInt64{Int64: *input.QualityID, Valid: true}
	}

	var row *sqlc.MovieFile

	// Use CreateMovieFileWithImportInfo when original path is provided (for import tracking)
	if input.OriginalPath != "" {
		row, err = s.Queries.CreateMovieFileWithImportInfo(ctx, sqlc.CreateMovieFileWithImportInfoParams{
			MovieID:          movieID,
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
		row, err = s.Queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
			MovieID:       movieID,
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
		return nil, fmt.Errorf("failed to create movie file: %w", err)
	}

	st := status.Available
	if qualityID.Valid && s.QualityProfiles != nil {
		if profile, profileErr := s.QualityProfiles.Get(ctx, movie.QualityProfileID); profileErr == nil {
			st = profile.StatusForQuality(int(qualityID.Int64))
		}
	}
	_ = s.Queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:     movieID,
		Status: st,
	})

	file := s.rowToMovieFile(row)
	s.Logger.Info().Int64("movieId", movieID).Str("path", input.Path).Msg("Added movie file")

	return &file, nil
}

// GetFileByPath retrieves a movie file by its path.
// Returns sql.ErrNoRows if the file doesn't exist.
func (s *Service) GetFileByPath(ctx context.Context, path string) (*MovieFile, error) {
	path = pathutil.NormalizePath(path)
	row, err := s.Queries.GetMovieFileByPath(ctx, path)
	if err != nil {
		return nil, err
	}
	file := s.rowToMovieFile(row)
	return &file, nil
}

// RemoveFile removes a file from a movie.
// Req 12.1.1: Deleting file from slot does NOT trigger automatic search
// Req 12.1.2: Slot becomes empty; waits for next scheduled search
func (s *Service) RemoveFile(ctx context.Context, fileID int64) error {
	row, err := s.Queries.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMovieFileNotFound
		}
		return fmt.Errorf("failed to get movie file: %w", err)
	}

	if s.fileDeleteHandler != nil {
		if err := s.fileDeleteHandler.OnFileDeleted(ctx, "movie", fileID); err != nil {
			s.Logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear slot assignment")
		}
	}

	if err := s.Queries.DeleteMovieFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete movie file: %w", err)
	}

	if err := s.Queries.DeleteImportDecisionsByExistingFile(ctx, sql.NullInt64{Int64: fileID, Valid: true}); err != nil {
		s.Logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear import decisions for removed file")
	}

	count, _ := s.Queries.CountMovieFiles(ctx, row.MovieID)
	if count == 0 {
		s.transitionMovieToMissingAfterFileRemoval(ctx, row.MovieID)
	}

	s.Logger.Info().Int64("fileId", fileID).Int64("movieId", row.MovieID).Msg("Removed movie file")
	return nil
}

// GetFileByID retrieves a movie file by its ID.
func (s *Service) GetFileByID(ctx context.Context, fileID int64) (*MovieFile, error) {
	row, err := s.Queries.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMovieFileNotFound
		}
		return nil, fmt.Errorf("failed to get movie file: %w", err)
	}
	file := s.rowToMovieFile(row)
	return &file, nil
}

// UpdateMovieFilePath updates the path of a movie file.
func (s *Service) UpdateMovieFilePath(ctx context.Context, fileID int64, newPath string) error {
	return s.Queries.UpdateMovieFilePath(ctx, sqlc.UpdateMovieFilePathParams{
		Path: pathutil.NormalizePath(newPath),
		ID:   fileID,
	})
}

// UpdateFileMediaInfo updates the MediaInfo fields of a movie's primary file.
func (s *Service) UpdateFileMediaInfo(ctx context.Context, movieID int64, info *mediainfo.MediaInfo) error {
	return s.Queries.UpdateMovieFileMediaInfo(ctx, sqlc.UpdateMovieFileMediaInfoParams{
		VideoCodec: sql.NullString{String: info.VideoCodec, Valid: info.VideoCodec != ""},
		AudioCodec: sql.NullString{String: info.AudioCodec, Valid: info.AudioCodec != ""},
		Resolution: sql.NullString{String: info.VideoResolution, Valid: info.VideoResolution != ""},
		MovieID:    movieID,
	})
}

// FindByTitleAndYear finds a movie by normalized title and optional year.
// Returns nil if no match is found.
func (s *Service) FindByTitleAndYear(ctx context.Context, title string, year int) (*Movie, error) {
	allMovies, err := s.List(ctx, ListMoviesOptions{})
	if err != nil {
		return nil, err
	}

	normalizedSearch := parseutil.NormalizeTitle(title)
	for _, m := range allMovies {
		if parseutil.NormalizeTitle(m.Title) == normalizedSearch {
			if year == 0 || m.Year == year {
				return m, nil
			}
		}
	}
	return nil, nil //nolint:nilnil // nil movie with no error means "no match found"
}

// Count returns the total number of movies.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.Queries.CountMovies(ctx)
}

// rowToMovie converts a database row to a Movie.
func (s *Service) rowToMovie(row *sqlc.Movie) *Movie {
	m := &Movie{
		ID:         row.ID,
		Title:      row.Title,
		SortTitle:  row.SortTitle,
		Monitored:  row.Monitored,
		Status:     row.Status,
		MovieFiles: []MovieFile{},
	}

	s.mapMovieNullableFields(m, row)
	return m
}

func (s *Service) mapMovieNullableFields(m *Movie, row *sqlc.Movie) {
	s.mapMovieCoreFields(m, row)
	s.mapMovieDatesAndStatus(m, row)
	s.mapMovieMetadataFields(m, row)
}

func (s *Service) mapMovieCoreFields(m *Movie, row *sqlc.Movie) {
	if row.Year.Valid {
		m.Year = int(row.Year.Int64)
	}
	if row.TmdbID.Valid {
		m.TmdbID = int(row.TmdbID.Int64)
	}
	if row.ImdbID.Valid {
		m.ImdbID = row.ImdbID.String
	}
	if row.Overview.Valid {
		m.Overview = row.Overview.String
	}
	if row.Runtime.Valid {
		m.Runtime = int(row.Runtime.Int64)
	}
	if row.Path.Valid {
		m.Path = row.Path.String
	}
	if row.RootFolderID.Valid {
		m.RootFolderID = row.RootFolderID.Int64
	}
	if row.QualityProfileID.Valid {
		m.QualityProfileID = row.QualityProfileID.Int64
	}
}

func (s *Service) mapMovieMetadataFields(m *Movie, row *sqlc.Movie) {
	if row.Studio.Valid {
		m.Studio = row.Studio.String
	}
	if row.TvdbID.Valid {
		m.TvdbID = int(row.TvdbID.Int64)
	}
	if row.ContentRating.Valid {
		m.ContentRating = row.ContentRating.String
	}
	if row.AddedBy.Valid {
		v := row.AddedBy.Int64
		m.AddedBy = &v
	}
}

func (s *Service) mapMovieDatesAndStatus(m *Movie, row *sqlc.Movie) {
	if row.AddedAt.Valid {
		m.AddedAt = row.AddedAt.Time
	}
	if row.UpdatedAt.Valid {
		m.UpdatedAt = row.UpdatedAt.Time
	}
	if row.ReleaseDate.Valid {
		m.ReleaseDate = &row.ReleaseDate.Time
	}
	if row.PhysicalReleaseDate.Valid {
		m.PhysicalReleaseDate = &row.PhysicalReleaseDate.Time
	}
	if row.TheatricalReleaseDate.Valid {
		m.TheatricalReleaseDate = &row.TheatricalReleaseDate.Time
	}
	if row.StatusMessage.Valid {
		m.StatusMessage = &row.StatusMessage.String
	}
	if row.ActiveDownloadID.Valid {
		m.ActiveDownloadID = &row.ActiveDownloadID.String
	}
}

// getMovieRowToMovie converts a GetMovieWithAddedByRow (with JOIN) to a Movie.
func (s *Service) getMovieRowToMovie(row *sqlc.GetMovieWithAddedByRow) *Movie {
	m := s.rowToMovie(&sqlc.Movie{
		ID:                    row.ID,
		Title:                 row.Title,
		SortTitle:             row.SortTitle,
		Year:                  row.Year,
		TmdbID:                row.TmdbID,
		ImdbID:                row.ImdbID,
		Overview:              row.Overview,
		Runtime:               row.Runtime,
		Path:                  row.Path,
		RootFolderID:          row.RootFolderID,
		QualityProfileID:      row.QualityProfileID,
		Monitored:             row.Monitored,
		Status:                row.Status,
		ActiveDownloadID:      row.ActiveDownloadID,
		StatusMessage:         row.StatusMessage,
		ReleaseDate:           row.ReleaseDate,
		PhysicalReleaseDate:   row.PhysicalReleaseDate,
		AddedAt:               row.AddedAt,
		UpdatedAt:             row.UpdatedAt,
		TheatricalReleaseDate: row.TheatricalReleaseDate,
		Studio:                row.Studio,
		TvdbID:                row.TvdbID,
		ContentRating:         row.ContentRating,
		AddedBy:               row.AddedBy,
	})
	if row.AddedByUsername.Valid {
		m.AddedByUsername = row.AddedByUsername.String
	}
	return m
}

func (s *Service) checkDuplicateTmdbID(ctx context.Context, tmdbID int) error {
	if tmdbID <= 0 {
		return nil
	}
	_, err := s.GetByTmdbID(ctx, tmdbID)
	if err == nil {
		return ErrDuplicateTmdbID
	}
	if !errors.Is(err, ErrMovieNotFound) {
		return err
	}
	return nil
}

func parseReleaseDates(release, physical, theatrical string) (releaseDate, physicalReleaseDate, theatricalReleaseDate sql.NullTime) {
	if release != "" {
		if t, err := time.Parse("2006-01-02", release); err == nil {
			releaseDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if physical != "" {
		if t, err := time.Parse("2006-01-02", physical); err == nil {
			physicalReleaseDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if theatrical != "" {
		if t, err := time.Parse("2006-01-02", theatrical); err == nil {
			theatricalReleaseDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	return releaseDate, physicalReleaseDate, theatricalReleaseDate
}

func (s *Service) buildMovieUpdateParams(id int64, current *Movie, input *UpdateMovieInput) sqlc.UpdateMovieParams {
	title := module.ResolveField(current.Title, input.Title)
	year := module.ResolveField(current.Year, input.Year)
	tmdbID := module.ResolveField(current.TmdbID, input.TmdbID)
	imdbID := module.ResolveField(current.ImdbID, input.ImdbID)
	overview := module.ResolveField(current.Overview, input.Overview)
	runtime := module.ResolveField(current.Runtime, input.Runtime)
	path := module.ResolveField(current.Path, input.Path)
	rootFolderID := module.ResolveField(current.RootFolderID, input.RootFolderID)
	qualityProfileID := module.ResolveField(current.QualityProfileID, input.QualityProfileID)
	monitored := module.ResolveField(current.Monitored, input.Monitored)
	studio := module.ResolveField(current.Studio, input.Studio)

	releaseDate := s.parseOrKeepDate(input.ReleaseDate, current.ReleaseDate)
	physicalReleaseDate := s.parseOrKeepDate(input.PhysicalReleaseDate, current.PhysicalReleaseDate)
	theatricalReleaseDate := s.parseOrKeepDate(input.TheatricalReleaseDate, current.TheatricalReleaseDate)

	st := current.Status
	released := isMovieReleased(releaseDate, physicalReleaseDate, theatricalReleaseDate)
	if st == status.Unreleased && released {
		st = status.Missing
	} else if st == status.Missing && !released {
		st = status.Unreleased
	}

	return sqlc.UpdateMovieParams{
		ID:                    id,
		Title:                 title,
		SortTitle:             module.GenerateSortTitle(title),
		Year:                  sql.NullInt64{Int64: int64(year), Valid: year > 0},
		TmdbID:                sql.NullInt64{Int64: int64(tmdbID), Valid: tmdbID > 0},
		ImdbID:                sql.NullString{String: imdbID, Valid: imdbID != ""},
		Overview:              sql.NullString{String: overview, Valid: overview != ""},
		Runtime:               sql.NullInt64{Int64: int64(runtime), Valid: runtime > 0},
		Path:                  sql.NullString{String: path, Valid: path != ""},
		RootFolderID:          sql.NullInt64{Int64: rootFolderID, Valid: rootFolderID > 0},
		QualityProfileID:      sql.NullInt64{Int64: qualityProfileID, Valid: qualityProfileID > 0},
		Monitored:             monitored,
		Status:                st,
		ReleaseDate:           releaseDate,
		PhysicalReleaseDate:   physicalReleaseDate,
		TheatricalReleaseDate: theatricalReleaseDate,
		Studio:                sql.NullString{String: studio, Valid: studio != ""},
		TvdbID:                sql.NullInt64{Int64: int64(current.TvdbID), Valid: current.TvdbID > 0},
		ContentRating:         sql.NullString{String: current.ContentRating, Valid: current.ContentRating != ""},
	}
}

func (s *Service) parseOrKeepDate(inputDate *string, currentDate *time.Time) sql.NullTime {
	if inputDate != nil {
		if *inputDate != "" {
			if t, err := time.Parse("2006-01-02", *inputDate); err == nil {
				return sql.NullTime{Time: t, Valid: true}
			}
		}
		return sql.NullTime{}
	}
	if currentDate != nil {
		return sql.NullTime{Time: *currentDate, Valid: true}
	}
	return sql.NullTime{}
}

func (s *Service) transitionMovieToMissingAfterFileRemoval(ctx context.Context, movieID int64) {
	var logStatusChange func(ctx context.Context, entityType string, entityID int64, oldStatus, newStatus, reason string) error
	if s.StatusChangeLogger != nil {
		logStatusChange = s.StatusChangeLogger.LogStatusChanged
	}
	var broadcastUpdate func()
	if s.Hub != nil {
		broadcastUpdate = func() {
			s.BroadcastEntity("movie", "movie", movieID, "updated", nil)
		}
	}

	module.TransitionToMissingAfterFileRemoval(ctx, &module.FileRemovalTransitionParams{
		ModuleType: module.TypeMovie,
		EntityType: module.EntityMovie,
		EntityID:   movieID,
		Logger:     s.Logger,
		GetCurrentStatus: func(ctx context.Context, entityID int64) (string, error) {
			movie, err := s.Queries.GetMovie(ctx, entityID)
			if err != nil {
				return "", err
			}
			return movie.Status, nil
		},
		SetMissingAndUnmonitor: func(ctx context.Context, entityID int64) error {
			_ = s.Queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
				ID:     entityID,
				Status: status.Missing,
			})
			_ = s.Queries.UpdateMovieMonitored(ctx, sqlc.UpdateMovieMonitoredParams{
				ID:        entityID,
				Monitored: false,
			})
			return nil
		},
		LogStatusChange: logStatusChange,
		BroadcastUpdate: broadcastUpdate,
	})
}

// rowToMovieFile converts a database row to a MovieFile.
func (s *Service) rowToMovieFile(row *sqlc.MovieFile) MovieFile {
	return MovieFile{
		ID:            row.ID,
		MovieID:       row.MovieID,
		Path:          row.Path,
		Size:          row.Size,
		Quality:       module.NullStr(row.Quality),
		VideoCodec:    module.NullStr(row.VideoCodec),
		AudioCodec:    module.NullStr(row.AudioCodec),
		AudioChannels: module.NullStr(row.AudioChannels),
		DynamicRange:  module.NullStr(row.DynamicRange),
		Resolution:    module.NullStr(row.Resolution),
		CreatedAt:     module.NullTime(row.CreatedAt),
		SlotID:        module.NullInt64Ptr(row.SlotID),
	}
}

// GenerateMoviePath generates a path for a movie.
// Returns a path with forward slashes for consistency across platforms.
func GenerateMoviePath(rootPath, title string, year int) string {
	folderName := title
	if year > 0 {
		folderName = fmt.Sprintf("%s (%d)", title, year)
	}
	return filepath.ToSlash(filepath.Join(rootPath, folderName))
}
