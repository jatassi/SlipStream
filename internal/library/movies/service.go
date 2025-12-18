package movies

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/websocket"
)

var (
	ErrMovieNotFound     = errors.New("movie not found")
	ErrMovieFileNotFound = errors.New("movie file not found")
	ErrInvalidMovie      = errors.New("invalid movie data")
	ErrDuplicateTmdbID   = errors.New("movie with this TMDB ID already exists")
)

// Service provides movie library operations.
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	hub     *websocket.Hub
	logger  zerolog.Logger
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

// Get retrieves a movie by ID.
func (s *Service) Get(ctx context.Context, id int64) (*Movie, error) {
	row, err := s.queries.GetMovie(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMovieNotFound
		}
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	movie := s.rowToMovie(row)

	// Get movie files
	files, err := s.GetFiles(ctx, id)
	if err != nil {
		s.logger.Warn().Err(err).Int64("movieId", id).Msg("Failed to get movie files")
	} else {
		movie.MovieFiles = files
		movie.HasFile = len(files) > 0
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
			Title:     searchTerm,
			SortTitle: searchTerm,
			Limit:     int64(opts.PageSize),
			Offset:    offset,
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
		// Check if has files
		count, _ := s.queries.CountMovieFiles(ctx, movies[i].ID)
		movies[i].HasFile = count > 0
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

	row, err := s.queries.CreateMovie(ctx, sqlc.CreateMovieParams{
		Title:            input.Title,
		SortTitle:        sortTitle,
		Year:             sql.NullInt64{Int64: int64(input.Year), Valid: input.Year > 0},
		TmdbID:           sql.NullInt64{Int64: int64(input.TmdbID), Valid: input.TmdbID > 0},
		ImdbID:           sql.NullString{String: input.ImdbID, Valid: input.ImdbID != ""},
		Overview:         sql.NullString{String: input.Overview, Valid: input.Overview != ""},
		Runtime:          sql.NullInt64{Int64: int64(input.Runtime), Valid: input.Runtime > 0},
		Path:             sql.NullString{String: path, Valid: path != ""},
		RootFolderID:     sql.NullInt64{Int64: input.RootFolderID, Valid: input.RootFolderID > 0},
		QualityProfileID: sql.NullInt64{Int64: input.QualityProfileID, Valid: input.QualityProfileID > 0},
		Monitored:        boolToInt(input.Monitored),
		Status:           "missing",
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

	return movie, nil
}

// Update updates an existing movie.
func (s *Service) Update(ctx context.Context, id int64, input UpdateMovieInput) (*Movie, error) {
	// Get current movie
	current, err := s.Get(ctx, id)
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

	row, err := s.queries.UpdateMovie(ctx, sqlc.UpdateMovieParams{
		ID:               id,
		Title:            title,
		SortTitle:        sortTitle,
		Year:             sql.NullInt64{Int64: int64(year), Valid: year > 0},
		TmdbID:           sql.NullInt64{Int64: int64(tmdbID), Valid: tmdbID > 0},
		ImdbID:           sql.NullString{String: imdbID, Valid: imdbID != ""},
		Overview:         sql.NullString{String: overview, Valid: overview != ""},
		Runtime:          sql.NullInt64{Int64: int64(runtime), Valid: runtime > 0},
		Path:             sql.NullString{String: path, Valid: path != ""},
		RootFolderID:     sql.NullInt64{Int64: rootFolderID, Valid: rootFolderID > 0},
		QualityProfileID: sql.NullInt64{Int64: qualityProfileID, Valid: qualityProfileID > 0},
		Monitored:        boolToInt(monitored),
		Status:           current.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update movie: %w", err)
	}

	movie := s.rowToMovie(row)
	s.logger.Info().Int64("id", id).Str("title", movie.Title).Msg("Updated movie")

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

// AddFile adds a file to a movie.
func (s *Service) AddFile(ctx context.Context, movieID int64, input CreateMovieFileInput) (*MovieFile, error) {
	// Verify movie exists
	_, err := s.Get(ctx, movieID)
	if err != nil {
		return nil, err
	}

	row, err := s.queries.CreateMovieFile(ctx, sqlc.CreateMovieFileParams{
		MovieID:    movieID,
		Path:       input.Path,
		Size:       input.Size,
		Quality:    sql.NullString{String: input.Quality, Valid: input.Quality != ""},
		VideoCodec: sql.NullString{String: input.VideoCodec, Valid: input.VideoCodec != ""},
		AudioCodec: sql.NullString{String: input.AudioCodec, Valid: input.AudioCodec != ""},
		Resolution: sql.NullString{String: input.Resolution, Valid: input.Resolution != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create movie file: %w", err)
	}

	// Update movie status
	_ = s.queries.UpdateMovieStatus(ctx, sqlc.UpdateMovieStatusParams{
		ID:     movieID,
		Status: "available",
	})

	file := s.rowToMovieFile(row)
	s.logger.Info().Int64("movieId", movieID).Str("path", input.Path).Msg("Added movie file")

	return &file, nil
}

// RemoveFile removes a file from a movie.
func (s *Service) RemoveFile(ctx context.Context, fileID int64) error {
	// Get file first
	row, err := s.queries.GetMovieFile(ctx, fileID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMovieFileNotFound
		}
		return fmt.Errorf("failed to get movie file: %w", err)
	}

	if err := s.queries.DeleteMovieFile(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete movie file: %w", err)
	}

	// Check if movie still has files
	count, _ := s.queries.CountMovieFiles(ctx, row.MovieID)
	if count == 0 {
		_ = s.queries.UpdateMovieStatus(ctx, sqlc.UpdateMovieStatusParams{
			ID:     row.MovieID,
			Status: "missing",
		})
	}

	s.logger.Info().Int64("fileId", fileID).Int64("movieId", row.MovieID).Msg("Removed movie file")
	return nil
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
	if row.Resolution.Valid {
		f.Resolution = row.Resolution.String
	}
	if row.CreatedAt.Valid {
		f.CreatedAt = row.CreatedAt.Time
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
