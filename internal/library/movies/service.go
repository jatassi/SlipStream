package movies

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/mediainfo"
	"github.com/slipstream/slipstream/internal/websocket"
)

// NotificationDispatcher defines the interface for movie notifications.
type NotificationDispatcher interface {
	DispatchMovieAdded(ctx context.Context, movie MovieNotificationInfo, addedAt time.Time)
	DispatchMovieDeleted(ctx context.Context, movie MovieNotificationInfo, deletedFiles bool, deletedAt time.Time)
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

// FileDeleteHandler is called when a file is deleted to update slot assignments.
type FileDeleteHandler interface {
	OnFileDeleted(ctx context.Context, mediaType string, fileID int64) error
}

// StatusChangeLogger logs status transition history events.
type StatusChangeLogger interface {
	LogStatusChanged(ctx context.Context, mediaType string, mediaID int64, from, to, reason string) error
}

// Service provides movie library operations.
type Service struct {
	db                 *sql.DB
	queries            *sqlc.Queries
	hub                *websocket.Hub
	logger             zerolog.Logger
	fileDeleteHandler  FileDeleteHandler
	statusChangeLogger StatusChangeLogger
	notifier           NotificationDispatcher
	qualityProfiles    *quality.Service
}

// SetNotificationDispatcher sets the notification dispatcher for movie events.
func (s *Service) SetNotificationDispatcher(n NotificationDispatcher) {
	s.notifier = n
}

// SetQualityService sets the quality profile service for quality evaluation.
func (s *Service) SetQualityService(qs *quality.Service) {
	s.qualityProfiles = qs
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
func NewService(db *sql.DB, hub *websocket.Hub, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		hub:     hub,
		logger:  logger.With().Str("component", "movies").Logger(),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// Get retrieves a movie by ID.
func (s *Service) Get(ctx context.Context, id int64) (*Movie, error) {
	row, err := s.queries.GetMovieWithAddedBy(ctx, id)
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
		s.logger.Warn().Err(err).Int64("movieId", id).Msg("Failed to get movie files")
	} else {
		movie.MovieFiles = files
		for _, f := range files {
			movie.SizeOnDisk += f.Size
		}
	}

	return movie, nil
}

// GetByTmdbID retrieves a movie by TMDB ID.
func (s *Service) GetByTmdbID(ctx context.Context, tmdbID int) (*Movie, error) {
	row, err := s.queries.GetMovieByTmdbID(ctx, sql.NullInt64{Int64: int64(tmdbID), Valid: true})
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
		rows, err = s.queries.SearchMovies(ctx, sqlc.SearchMoviesParams{
			SearchTerm: searchTerm,
			Lim:        int64(opts.PageSize),
			Off:        offset,
		})
	} else if opts.RootFolderID != nil {
		rows, err = s.queries.ListMoviesByRootFolder(ctx, sql.NullInt64{Int64: *opts.RootFolderID, Valid: true})
	} else if opts.Monitored != nil && *opts.Monitored {
		rows, err = s.queries.ListMonitoredMovies(ctx)
	} else {
		rows, err = s.queries.ListMoviesPaginated(ctx, sqlc.ListMoviesPaginatedParams{
			Limit:  int64(opts.PageSize),
			Offset: offset,
		})
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
	rows, err := s.queries.ListUnmatchedMoviesByRootFolder(ctx, sql.NullInt64{Int64: rootFolderID, Valid: true})
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
func (s *Service) Create(ctx context.Context, input CreateMovieInput) (*Movie, error) {
	if input.Title == "" {
		return nil, ErrInvalidMovie
	}

	// Check for duplicate TMDB ID
	if input.TmdbID > 0 {
		_, err := s.GetByTmdbID(ctx, input.TmdbID)
		if err == nil {
			return nil, ErrDuplicateTmdbID
		}
		if !errors.Is(err, ErrMovieNotFound) {
			return nil, err
		}
	}

	sortTitle := generateSortTitle(input.Title)

	// Generate path if not provided
	path := input.Path
	if path == "" && input.RootFolderID > 0 {
		// We'd need to get the root folder path here
		// For now, leave empty and let it be set later
	}

	// Parse release dates
	var releaseDate, physicalReleaseDate, theatricalReleaseDate sql.NullTime
	if input.ReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", input.ReleaseDate); err == nil {
			releaseDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if input.PhysicalReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", input.PhysicalReleaseDate); err == nil {
			physicalReleaseDate = sql.NullTime{Time: t, Valid: true}
		}
	}
	if input.TheatricalReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", input.TheatricalReleaseDate); err == nil {
			theatricalReleaseDate = sql.NullTime{Time: t, Valid: true}
		}
	}

	status := "unreleased"
	if isMovieReleased(releaseDate, physicalReleaseDate, theatricalReleaseDate) {
		status = "missing"
	}

	var addedBy sql.NullInt64
	if input.AddedBy != nil {
		addedBy = sql.NullInt64{Int64: *input.AddedBy, Valid: true}
	}

	row, err := s.queries.CreateMovie(ctx, sqlc.CreateMovieParams{
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
		Monitored:             boolToInt(input.Monitored),
		Status:                status,
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
	s.logger.Info().Int64("id", movie.ID).Str("title", movie.Title).Msg("Created movie")

	// Broadcast event
	if s.hub != nil {
		s.hub.Broadcast("movie:added", movie)
	}

	// Dispatch notification
	if s.notifier != nil {
		s.notifier.DispatchMovieAdded(ctx, MovieNotificationInfo{
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
func (s *Service) Update(ctx context.Context, id int64, input UpdateMovieInput) (*Movie, error) {
	s.logger.Debug().Int64("id", id).Msg("[UPDATE] Starting movie update")

	// Get current movie
	current, err := s.Get(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Int64("id", id).Msg("[UPDATE] Failed to get current movie")
		return nil, err
	}

	s.logger.Debug().
		Int64("id", id).
		Str("currentTitle", current.Title).
		Int("currentTmdbId", current.TmdbID).
		Str("currentImdbId", current.ImdbID).
		Msg("[UPDATE] Current movie state")

	// Apply updates
	title := current.Title
	if input.Title != nil {
		title = *input.Title
		s.logger.Debug().Str("newTitle", title).Msg("[UPDATE] Title will be updated")
	}
	sortTitle := generateSortTitle(title)

	year := current.Year
	if input.Year != nil {
		year = *input.Year
		s.logger.Debug().Int("newYear", year).Msg("[UPDATE] Year will be updated")
	}

	tmdbID := current.TmdbID
	if input.TmdbID != nil {
		tmdbID = *input.TmdbID
		s.logger.Debug().Int("newTmdbId", tmdbID).Msg("[UPDATE] TmdbID will be updated")
	}

	imdbID := current.ImdbID
	if input.ImdbID != nil {
		imdbID = *input.ImdbID
		s.logger.Debug().Str("newImdbId", imdbID).Msg("[UPDATE] ImdbID will be updated")
	}

	overview := current.Overview
	if input.Overview != nil {
		overview = *input.Overview
		s.logger.Debug().Int("overviewLen", len(overview)).Msg("[UPDATE] Overview will be updated")
	}

	runtime := current.Runtime
	if input.Runtime != nil {
		runtime = *input.Runtime
		s.logger.Debug().Int("newRuntime", runtime).Msg("[UPDATE] Runtime will be updated")
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

	studio := current.Studio
	if input.Studio != nil {
		studio = *input.Studio
	}
	tvdbID := current.TvdbID
	contentRating := current.ContentRating
	if input.ContentRating != nil {
		contentRating = *input.ContentRating
	}

	s.logger.Debug().
		Str("title", title).
		Int("year", year).
		Int("tmdbId", tmdbID).
		Str("imdbId", imdbID).
		Int("runtime", runtime).
		Msg("[UPDATE] Final values to be saved")

	// Handle release date updates
	var releaseDate, physicalReleaseDate, theatricalReleaseDate sql.NullTime
	if input.ReleaseDate != nil {
		if *input.ReleaseDate != "" {
			if t, err := time.Parse("2006-01-02", *input.ReleaseDate); err == nil {
				releaseDate = sql.NullTime{Time: t, Valid: true}
			}
		}
	} else if current.ReleaseDate != nil {
		releaseDate = sql.NullTime{Time: *current.ReleaseDate, Valid: true}
	}
	if input.PhysicalReleaseDate != nil {
		if *input.PhysicalReleaseDate != "" {
			if t, err := time.Parse("2006-01-02", *input.PhysicalReleaseDate); err == nil {
				physicalReleaseDate = sql.NullTime{Time: t, Valid: true}
			}
		}
	} else if current.PhysicalReleaseDate != nil {
		physicalReleaseDate = sql.NullTime{Time: *current.PhysicalReleaseDate, Valid: true}
	}
	if input.TheatricalReleaseDate != nil {
		if *input.TheatricalReleaseDate != "" {
			if t, err := time.Parse("2006-01-02", *input.TheatricalReleaseDate); err == nil {
				theatricalReleaseDate = sql.NullTime{Time: t, Valid: true}
			}
		}
	} else if current.TheatricalReleaseDate != nil {
		theatricalReleaseDate = sql.NullTime{Time: *current.TheatricalReleaseDate, Valid: true}
	}

	// Recalculate status if release dates changed and current status is unreleased/missing
	status := current.Status
	released := isMovieReleased(releaseDate, physicalReleaseDate, theatricalReleaseDate)
	if status == "unreleased" && released {
		status = "missing"
	} else if status == "missing" && !released {
		status = "unreleased"
	}

	row, err := s.queries.UpdateMovie(ctx, sqlc.UpdateMovieParams{
		ID:                    id,
		Title:                 title,
		SortTitle:             sortTitle,
		Year:                  sql.NullInt64{Int64: int64(year), Valid: year > 0},
		TmdbID:                sql.NullInt64{Int64: int64(tmdbID), Valid: tmdbID > 0},
		ImdbID:                sql.NullString{String: imdbID, Valid: imdbID != ""},
		Overview:              sql.NullString{String: overview, Valid: overview != ""},
		Runtime:               sql.NullInt64{Int64: int64(runtime), Valid: runtime > 0},
		Path:                  sql.NullString{String: path, Valid: path != ""},
		RootFolderID:          sql.NullInt64{Int64: rootFolderID, Valid: rootFolderID > 0},
		QualityProfileID:      sql.NullInt64{Int64: qualityProfileID, Valid: qualityProfileID > 0},
		Monitored:             boolToInt(monitored),
		Status:                status,
		ReleaseDate:           releaseDate,
		PhysicalReleaseDate:   physicalReleaseDate,
		TheatricalReleaseDate: theatricalReleaseDate,
		Studio:                sql.NullString{String: studio, Valid: studio != ""},
		TvdbID:                sql.NullInt64{Int64: int64(tvdbID), Valid: tvdbID > 0},
		ContentRating:         sql.NullString{String: contentRating, Valid: contentRating != ""},
	})
	if err != nil {
		s.logger.Error().Err(err).Int64("id", id).Msg("[UPDATE] Database update failed")
		return nil, fmt.Errorf("failed to update movie: %w", err)
	}

	movie := s.rowToMovie(row)
	s.logger.Info().
		Int64("id", id).
		Str("title", movie.Title).
		Int("tmdbId", movie.TmdbID).
		Str("imdbId", movie.ImdbID).
		Msg("[UPDATE] Movie updated successfully")

	// Broadcast event
	if s.hub != nil {
		s.hub.Broadcast("movie:updated", movie)
	}

	return movie, nil
}

// Delete deletes a movie.
func (s *Service) Delete(ctx context.Context, id int64, deleteFiles bool) error {
	movie, err := s.Get(ctx, id)
	if err != nil {
		return err
	}

	// Clean up download mappings for this movie to prevent seeding torrents from re-triggering imports
	if err := s.queries.DeleteDownloadMappingsByMovieID(ctx, sql.NullInt64{Int64: id, Valid: true}); err != nil {
		s.logger.Warn().Err(err).Int64("movieId", id).Msg("Failed to delete download mappings for movie")
	}

	// Delete movie files from database
	if err := s.queries.DeleteMovieFilesByMovie(ctx, id); err != nil {
		return fmt.Errorf("failed to delete movie files: %w", err)
	}

	// TODO: If deleteFiles is true, delete actual files from disk

	if err := s.queries.DeleteMovie(ctx, id); err != nil {
		return fmt.Errorf("failed to delete movie: %w", err)
	}

	s.logger.Info().Int64("id", id).Str("title", movie.Title).Msg("Deleted movie")

	// Broadcast event
	if s.hub != nil {
		s.hub.Broadcast("movie:deleted", map[string]int64{"id": id})
	}

	// Dispatch notification
	if s.notifier != nil {
		s.notifier.DispatchMovieDeleted(ctx, MovieNotificationInfo{
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

// GetFiles returns all files for a movie.
func (s *Service) GetFiles(ctx context.Context, movieID int64) ([]MovieFile, error) {
	rows, err := s.queries.ListMovieFiles(ctx, movieID)
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
// Returns nil, nil if no files exist.
func (s *Service) GetPrimaryFile(ctx context.Context, movieID int64) (*MovieFile, error) {
	files, err := s.GetFiles(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, nil
	}
	return &files[0], nil
}

// AddFile adds a file to a movie.
func (s *Service) AddFile(ctx context.Context, movieID int64, input CreateMovieFileInput) (*MovieFile, error) {
	movie, err := s.Get(ctx, movieID)
	if err != nil {
		return nil, err
	}

	qualityID := sql.NullInt64{}
	if input.QualityID != nil {
		qualityID = sql.NullInt64{Int64: *input.QualityID, Valid: true}
	}

	var row *sqlc.MovieFile

	// Use CreateMovieFileWithImportInfo when original path is provided (for import tracking)
	if input.OriginalPath != "" {
		row, err = s.queries.CreateMovieFileWithImportInfo(ctx, sqlc.CreateMovieFileWithImportInfoParams{
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
		row, err = s.queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
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

	status := "available"
	if qualityID.Valid && s.qualityProfiles != nil {
		if profile, profileErr := s.qualityProfiles.Get(ctx, movie.QualityProfileID); profileErr == nil {
			status = profile.StatusForQuality(int(qualityID.Int64))
		}
	}
	_ = s.queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
		ID:     movieID,
		Status: status,
	})

	file := s.rowToMovieFile(row)
	s.logger.Info().Int64("movieId", movieID).Str("path", input.Path).Msg("Added movie file")

	return &file, nil
}

// GetFileByPath retrieves a movie file by its path.
// Returns nil, nil if the file doesn't exist.
func (s *Service) GetFileByPath(ctx context.Context, path string) (*MovieFile, error) {
	row, err := s.queries.GetMovieFileByPath(ctx, path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get movie file by path: %w", err)
	}
	file := s.rowToMovieFile(row)
	return &file, nil
}

// RemoveFile removes a file from a movie.
// Req 12.1.1: Deleting file from slot does NOT trigger automatic search
// Req 12.1.2: Slot becomes empty; waits for next scheduled search
func (s *Service) RemoveFile(ctx context.Context, fileID int64) error {
	// Get file first
	row, err := s.queries.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMovieFileNotFound
		}
		return fmt.Errorf("failed to get movie file: %w", err)
	}

	// Clear slot assignment before deleting file (Req 12.1.1)
	if s.fileDeleteHandler != nil {
		if err := s.fileDeleteHandler.OnFileDeleted(ctx, "movie", fileID); err != nil {
			s.logger.Warn().Err(err).Int64("fileId", fileID).Msg("Failed to clear slot assignment")
		}
	}

	if err := s.queries.DeleteMovieFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete movie file: %w", err)
	}

	// Check if movie still has files; if none remain, transition to missing and unmonitor
	count, _ := s.queries.CountMovieFiles(ctx, row.MovieID)
	if count == 0 {
		movie, _ := s.queries.GetMovie(ctx, row.MovieID)
		oldStatus := ""
		if movie != nil {
			oldStatus = movie.Status
		}
		_ = s.queries.UpdateMovieStatusWithDetails(ctx, sqlc.UpdateMovieStatusWithDetailsParams{
			ID:     row.MovieID,
			Status: "missing",
		})
		_ = s.queries.UpdateMovieMonitored(ctx, sqlc.UpdateMovieMonitoredParams{
			ID:        row.MovieID,
			Monitored: 0,
		})
		if s.statusChangeLogger != nil && oldStatus != "" && oldStatus != "missing" {
			_ = s.statusChangeLogger.LogStatusChanged(ctx, "movie", row.MovieID, oldStatus, "missing", "File removed")
		}
		if s.hub != nil {
			_ = s.hub.Broadcast("movie:updated", map[string]any{"movieId": row.MovieID})
		}
	}

	s.logger.Info().Int64("fileId", fileID).Int64("movieId", row.MovieID).Msg("Removed movie file")
	return nil
}

// GetFileByID retrieves a movie file by its ID.
func (s *Service) GetFileByID(ctx context.Context, fileID int64) (*MovieFile, error) {
	row, err := s.queries.GetMovieFile(ctx, fileID)
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
	return s.queries.UpdateMovieFilePath(ctx, sqlc.UpdateMovieFilePathParams{
		Path: newPath,
		ID:   fileID,
	})
}

// UpdateFileMediaInfo updates the MediaInfo fields of a movie's primary file.
func (s *Service) UpdateFileMediaInfo(ctx context.Context, movieID int64, info *mediainfo.MediaInfo) error {
	return s.queries.UpdateMovieFileMediaInfo(ctx, sqlc.UpdateMovieFileMediaInfoParams{
		VideoCodec: sql.NullString{String: info.VideoCodec, Valid: info.VideoCodec != ""},
		AudioCodec: sql.NullString{String: info.AudioCodec, Valid: info.AudioCodec != ""},
		Resolution: sql.NullString{String: info.VideoResolution, Valid: info.VideoResolution != ""},
		MovieID:    movieID,
	})
}

// Count returns the total number of movies.
func (s *Service) Count(ctx context.Context) (int64, error) {
	return s.queries.CountMovies(ctx)
}

// rowToMovie converts a database row to a Movie.
func (s *Service) rowToMovie(row *sqlc.Movie) *Movie {
	m := &Movie{
		ID:        row.ID,
		Title:     row.Title,
		SortTitle: row.SortTitle,
		Monitored: row.Monitored == 1,
		Status:    row.Status,
	}

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

	return m
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

// rowToMovieFile converts a database row to a MovieFile.
func (s *Service) rowToMovieFile(row *sqlc.MovieFile) MovieFile {
	f := MovieFile{
		ID:      row.ID,
		MovieID: row.MovieID,
		Path:    row.Path,
		Size:    row.Size,
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

// GenerateMoviePath generates a path for a movie.
// Returns a path with forward slashes for consistency across platforms.
func GenerateMoviePath(rootPath, title string, year int) string {
	folderName := title
	if year > 0 {
		folderName = fmt.Sprintf("%s (%d)", title, year)
	}
	return rootPath + "/" + folderName
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
