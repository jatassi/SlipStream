package arrimport

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/library/movies"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
	"github.com/slipstream/slipstream/internal/library/tv"
	"github.com/slipstream/slipstream/internal/progress"
)

const previewStatusSkip = "skip"

// MovieService defines the interface for movie operations.
type MovieService interface {
	Create(ctx context.Context, input *movies.CreateMovieInput) (*movies.Movie, error)
	GetByTmdbID(ctx context.Context, tmdbID int) (*movies.Movie, error)
	AddFile(ctx context.Context, movieID int64, input *movies.CreateMovieFileInput) (*movies.MovieFile, error)
}

// TVService defines the interface for TV operations.
type TVService interface {
	CreateSeries(ctx context.Context, input *tv.CreateSeriesInput) (*tv.Series, error)
	GetSeriesByTvdbID(ctx context.Context, tvdbID int) (*tv.Series, error)
	GetEpisodeByNumber(ctx context.Context, seriesID int64, seasonNumber, episodeNumber int) (*tv.Episode, error)
	AddEpisodeFile(ctx context.Context, episodeID int64, input *tv.CreateEpisodeFileInput) (*tv.EpisodeFile, error)
}

// RootFolderService defines the interface for root folder operations.
type RootFolderService interface {
	List(ctx context.Context) ([]*RootFolder, error)
}

// RootFolder represents a SlipStream root folder.
type RootFolder struct {
	ID        int64
	Name      string
	Path      string
	MediaType string
}

// QualityService defines the interface for quality profile operations.
type QualityService interface {
	List(ctx context.Context) ([]*QualityProfile, error)
}

// QualityProfile represents a SlipStream quality profile.
type QualityProfile struct {
	ID   int64
	Name string
}

// SlotsService defines the interface for slot operations.
type SlotsService interface {
	IsMultiVersionEnabled(ctx context.Context) bool
	InitializeSlotAssignments(ctx context.Context, mediaType string, mediaID int64) error
	DetermineTargetSlot(ctx context.Context, parsed *scanner.ParsedMedia, mediaType string, mediaID int64) (*slots.SlotAssignment, error)
	AssignFileToSlot(ctx context.Context, mediaType string, mediaID, slotID, fileID int64) error
}

// Service manages library imports from external sources.
type Service struct {
	db                *sql.DB
	reader            Reader
	sourceType        SourceType
	movieService      MovieService
	tvService         TVService
	rootFolderService RootFolderService
	qualityService    QualityService
	slotsService      SlotsService
	progressManager   *progress.Manager
	hub               interface{ BroadcastJSON(v interface{}) }
	logger            *zerolog.Logger
	mu                sync.Mutex
}

// NewService creates a new library import service.
func NewService(
	db *sql.DB,
	movieService MovieService,
	tvService TVService,
	rootFolderService RootFolderService,
	qualityService QualityService,
	progressManager *progress.Manager,
	hub interface{ BroadcastJSON(v interface{}) },
	logger *zerolog.Logger,
) *Service {
	return &Service{
		db:                db,
		movieService:      movieService,
		tvService:         tvService,
		rootFolderService: rootFolderService,
		qualityService:    qualityService,
		progressManager:   progressManager,
		hub:               hub,
		logger:            logger,
	}
}

// SetSlotsService sets the optional slots service for multi-version support.
func (s *Service) SetSlotsService(svc SlotsService) {
	s.slotsService = svc
}

// Connect establishes a connection to the source application.
func (s *Service) Connect(ctx context.Context, cfg ConnectionConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	reader, err := NewReader(cfg)
	if err != nil {
		return fmt.Errorf("failed to create reader: %w", err)
	}

	if err := reader.Validate(ctx); err != nil {
		return fmt.Errorf("failed to validate connection: %w", err)
	}

	s.reader = reader
	s.sourceType = cfg.SourceType
	s.logger.Info().Str("sourceType", string(cfg.SourceType)).Msg("connected to source")

	return nil
}

// GetSourceRootFolders retrieves the list of root folders from the source.
func (s *Service) GetSourceRootFolders(ctx context.Context) ([]SourceRootFolder, error) {
	s.mu.Lock()
	reader := s.reader
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	return reader.ReadRootFolders(ctx)
}

// GetSourceQualityProfiles retrieves the list of quality profiles from the source.
func (s *Service) GetSourceQualityProfiles(ctx context.Context) ([]SourceQualityProfile, error) {
	s.mu.Lock()
	reader := s.reader
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	return reader.ReadQualityProfiles(ctx)
}

// Preview generates a preview of what will be imported without making changes.
func (s *Service) Preview(ctx context.Context, mappings ImportMappings) (*ImportPreview, error) {
	s.mu.Lock()
	reader := s.reader
	sourceType := s.sourceType
	s.mu.Unlock()

	if reader == nil {
		return nil, fmt.Errorf("not connected")
	}

	preview := &ImportPreview{
		Movies:  []MoviePreview{},
		Series:  []SeriesPreview{},
		Summary: ImportSummary{},
	}

	// Read and preview movies (only for Radarr)
	if sourceType == SourceTypeRadarr {
		if err := s.previewMovies(ctx, reader, preview); err != nil {
			return nil, fmt.Errorf("failed to preview movies: %w", err)
		}
	}

	// Read and preview series (only for Sonarr)
	if sourceType == SourceTypeSonarr {
		if err := s.previewSeries(ctx, reader, preview); err != nil {
			return nil, fmt.Errorf("failed to preview series: %w", err)
		}
	}

	return preview, nil
}

func (s *Service) previewMovies(ctx context.Context, reader Reader, preview *ImportPreview) error {
	sourceMovies, err := reader.ReadMovies(ctx)
	if err != nil {
		return err
	}

	for i := range sourceMovies {
		moviePreview := MoviePreview{
			Title:   sourceMovies[i].Title,
			Year:    sourceMovies[i].Year,
			TmdbID:  sourceMovies[i].TmdbID,
			HasFile: sourceMovies[i].HasFile,
		}

		if sourceMovies[i].File != nil {
			moviePreview.Quality = sourceMovies[i].File.QualityName
		}

		if sourceMovies[i].TmdbID == 0 {
			moviePreview.Status = previewStatusSkip
			moviePreview.SkipReason = "no TMDB ID"
			preview.Movies = append(preview.Movies, moviePreview)
			preview.Summary.TotalMovies++
			preview.Summary.SkippedMovies++
			continue
		}

		_, err := s.movieService.GetByTmdbID(ctx, sourceMovies[i].TmdbID)
		if err != nil {
			errMsg := err.Error()
			if errMsg == "movie not found" {
				moviePreview.Status = "new"
				preview.Summary.NewMovies++
			} else {
				s.logger.Warn().Int("tmdbId", sourceMovies[i].TmdbID).Err(err).Msg("failed to check movie existence")
				moviePreview.Status = previewStatusSkip
				moviePreview.SkipReason = "error checking existence"
				preview.Summary.SkippedMovies++
			}
		} else {
			moviePreview.Status = "duplicate"
			preview.Summary.DuplicateMovies++
		}

		preview.Movies = append(preview.Movies, moviePreview)
		preview.Summary.TotalMovies++
		if sourceMovies[i].HasFile {
			preview.Summary.TotalFiles++
		}
	}

	return nil
}

func (s *Service) previewSeries(ctx context.Context, reader Reader, preview *ImportPreview) error {
	seriesList, err := reader.ReadSeries(ctx)
	if err != nil {
		return err
	}

	for i := range seriesList {
		seriesPreview := SeriesPreview{
			Title:  seriesList[i].Title,
			Year:   seriesList[i].Year,
			TvdbID: seriesList[i].TvdbID,
		}

		if seriesList[i].TvdbID == 0 {
			seriesPreview.Status = previewStatusSkip
			seriesPreview.SkipReason = "no TVDB ID"
			preview.Series = append(preview.Series, seriesPreview)
			preview.Summary.TotalSeries++
			preview.Summary.SkippedSeries++
			continue
		}

		episodes, err := reader.ReadEpisodes(ctx, seriesList[i].ID)
		if err != nil {
			s.logger.Warn().Int64("seriesId", seriesList[i].ID).Err(err).Msg("failed to read episodes")
			episodes = []SourceEpisode{}
		}
		seriesPreview.EpisodeCount = len(episodes)
		preview.Summary.TotalEpisodes += len(episodes)

		files, err := reader.ReadEpisodeFiles(ctx, seriesList[i].ID)
		if err != nil {
			s.logger.Warn().Int64("seriesId", seriesList[i].ID).Err(err).Msg("failed to read episode files")
			files = []SourceEpisodeFile{}
		}
		seriesPreview.FileCount = len(files)
		preview.Summary.TotalFiles += len(files)

		_, err = s.tvService.GetSeriesByTvdbID(ctx, seriesList[i].TvdbID)
		if err != nil {
			errMsg := err.Error()
			if errMsg == "series not found" {
				seriesPreview.Status = "new"
				preview.Summary.NewSeries++
			} else {
				s.logger.Warn().Int("tvdbId", seriesList[i].TvdbID).Err(err).Msg("failed to check series existence")
				seriesPreview.Status = previewStatusSkip
				seriesPreview.SkipReason = "error checking existence"
				preview.Summary.SkippedSeries++
			}
		} else {
			seriesPreview.Status = "duplicate"
			preview.Summary.DuplicateSeries++
		}

		preview.Series = append(preview.Series, seriesPreview)
		preview.Summary.TotalSeries++
	}

	return nil
}

// Execute starts the import process asynchronously.
// The actual import is handled by an Executor running in a goroutine.
// Progress is tracked via the progress manager and broadcast over WebSocket.
func (s *Service) Execute(ctx context.Context, mappings ImportMappings) error {
	s.mu.Lock()
	if s.reader == nil {
		s.mu.Unlock()
		return fmt.Errorf("not connected")
	}
	reader := s.reader
	sourceType := s.sourceType
	s.mu.Unlock()

	executor := NewExecutor(s.db, reader, sourceType, s.movieService, s.tvService, s.slotsService, s.progressManager, s.logger)
	go executor.Run(ctx, mappings)

	return nil
}

// Disconnect closes the connection to the source and clears session state.
func (s *Service) Disconnect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.reader != nil {
		if err := s.reader.Close(); err != nil {
			s.logger.Warn().Err(err).Msg("error closing reader")
		}
		s.reader = nil
	}

	s.sourceType = ""
	s.logger.Info().Msg("disconnected from source")

	return nil
}
