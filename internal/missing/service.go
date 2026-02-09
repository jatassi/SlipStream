package missing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// Service provides operations for retrieving missing media.
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	logger  zerolog.Logger
}

// NewService creates a new missing service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		logger:  logger.With().Str("component", "missing").Logger(),
	}
}

// SetDB updates the database connection used by this service.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// MissingMovie represents a movie that is released but has no file.
type MissingMovie struct {
	ID                  int64      `json:"id"`
	Title               string     `json:"title"`
	Year                int        `json:"year,omitempty"`
	TmdbID              int        `json:"tmdbId,omitempty"`
	ImdbID              string     `json:"imdbId,omitempty"`
	ReleaseDate         *time.Time `json:"releaseDate,omitempty"`
	PhysicalReleaseDate *time.Time `json:"physicalReleaseDate,omitempty"`
	Path                string     `json:"path,omitempty"`
	QualityProfileID    int64      `json:"qualityProfileId"`
}

// MissingEpisode represents an episode that is released but has no file.
type MissingEpisode struct {
	ID            int64      `json:"id"`
	SeriesID      int64      `json:"seriesId"`
	SeasonNumber  int        `json:"seasonNumber"`
	EpisodeNumber int        `json:"episodeNumber"`
	Title         string     `json:"title"`
	AirDate       *time.Time `json:"airDate,omitempty"`
	SeriesTitle   string     `json:"seriesTitle"`
	SeriesTvdbID  int        `json:"seriesTvdbId,omitempty"`
	SeriesTmdbID  int        `json:"seriesTmdbId,omitempty"`
	SeriesImdbID  string     `json:"seriesImdbId,omitempty"`
	SeriesYear    int        `json:"seriesYear,omitempty"`
}

// MissingSeason groups missing episodes by season.
type MissingSeason struct {
	SeasonNumber    int               `json:"seasonNumber"`
	MissingEpisodes []*MissingEpisode `json:"missingEpisodes"`
}

// MissingSeries groups missing episodes by series with hierarchical structure.
type MissingSeries struct {
	ID               int64            `json:"id"`
	Title            string           `json:"title"`
	Year             int              `json:"year,omitempty"`
	TvdbID           int              `json:"tvdbId,omitempty"`
	TmdbID           int              `json:"tmdbId,omitempty"`
	ImdbID           string           `json:"imdbId,omitempty"`
	QualityProfileID int64            `json:"qualityProfileId"`
	MissingCount     int              `json:"missingCount"`
	MissingSeasons   []*MissingSeason `json:"missingSeasons"`
}

// MissingCounts returns counts for nav badge.
type MissingCounts struct {
	Movies   int64 `json:"movies"`
	Episodes int64 `json:"episodes"`
}

// UpgradableMovie represents a movie with a file below the quality cutoff.
type UpgradableMovie struct {
	ID                  int64      `json:"id"`
	Title               string     `json:"title"`
	Year                int        `json:"year,omitempty"`
	TmdbID              int        `json:"tmdbId,omitempty"`
	ImdbID              string     `json:"imdbId,omitempty"`
	ReleaseDate         *time.Time `json:"releaseDate,omitempty"`
	PhysicalReleaseDate *time.Time `json:"physicalReleaseDate,omitempty"`
	Path                string     `json:"path,omitempty"`
	QualityProfileID    int64      `json:"qualityProfileId"`
	CurrentQualityID    int        `json:"currentQualityId"`
}

// UpgradableEpisode represents an episode with a file below the quality cutoff.
type UpgradableEpisode struct {
	ID               int64      `json:"id"`
	SeriesID         int64      `json:"seriesId"`
	SeasonNumber     int        `json:"seasonNumber"`
	EpisodeNumber    int        `json:"episodeNumber"`
	Title            string     `json:"title"`
	AirDate          *time.Time `json:"airDate,omitempty"`
	SeriesTitle      string     `json:"seriesTitle"`
	SeriesTvdbID     int        `json:"seriesTvdbId,omitempty"`
	SeriesTmdbID     int        `json:"seriesTmdbId,omitempty"`
	SeriesImdbID     string     `json:"seriesImdbId,omitempty"`
	SeriesYear       int        `json:"seriesYear,omitempty"`
	CurrentQualityID int        `json:"currentQualityId"`
}

// UpgradableSeason groups upgradable episodes by season.
type UpgradableSeason struct {
	SeasonNumber       int                  `json:"seasonNumber"`
	UpgradableEpisodes []*UpgradableEpisode `json:"upgradableEpisodes"`
}

// UpgradableSeries groups upgradable episodes by series.
type UpgradableSeries struct {
	ID                int64               `json:"id"`
	Title             string              `json:"title"`
	Year              int                 `json:"year,omitempty"`
	TvdbID            int                 `json:"tvdbId,omitempty"`
	TmdbID            int                 `json:"tmdbId,omitempty"`
	ImdbID            string              `json:"imdbId,omitempty"`
	QualityProfileID  int64               `json:"qualityProfileId"`
	UpgradableCount   int                 `json:"upgradableCount"`
	UpgradableSeasons []*UpgradableSeason `json:"upgradableSeasons"`
}

// UpgradableCounts returns counts of upgradable items.
type UpgradableCounts struct {
	Movies   int64 `json:"movies"`
	Episodes int64 `json:"episodes"`
}

// GetMissingMovies returns all movies that are released but have no files.
func (s *Service) GetMissingMovies(ctx context.Context) ([]*MissingMovie, error) {
	rows, err := s.queries.ListMissingMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list missing movies: %w", err)
	}

	movies := make([]*MissingMovie, len(rows))
	for i, row := range rows {
		movies[i] = s.rowToMissingMovie(row)
	}
	return movies, nil
}

// GetMissingSeries returns all series that have missing episodes, grouped hierarchically.
func (s *Service) GetMissingSeries(ctx context.Context) ([]*MissingSeries, error) {
	// Get all missing episodes
	rows, err := s.queries.ListMissingEpisodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list missing episodes: %w", err)
	}

	// Group episodes by series and season
	seriesMap := make(map[int64]*MissingSeries)
	seasonMap := make(map[int64]map[int]*MissingSeason) // seriesID -> seasonNumber -> season

	for _, row := range rows {
		seriesID := row.SeriesID
		seasonNumber := int(row.SeasonNumber)

		// Get or create series
		series, exists := seriesMap[seriesID]
		if !exists {
			series = &MissingSeries{
				ID:             seriesID,
				Title:          row.SeriesTitle,
				MissingSeasons: []*MissingSeason{},
			}
			if row.SeriesYear.Valid {
				series.Year = int(row.SeriesYear.Int64)
			}
			if row.SeriesTvdbID.Valid {
				series.TvdbID = int(row.SeriesTvdbID.Int64)
			}
			if row.SeriesTmdbID.Valid {
				series.TmdbID = int(row.SeriesTmdbID.Int64)
			}
			if row.SeriesImdbID.Valid {
				series.ImdbID = row.SeriesImdbID.String
			}
			if row.SeriesQualityProfileID.Valid {
				series.QualityProfileID = row.SeriesQualityProfileID.Int64
			}
			seriesMap[seriesID] = series
			seasonMap[seriesID] = make(map[int]*MissingSeason)
		}

		// Get or create season
		season, exists := seasonMap[seriesID][seasonNumber]
		if !exists {
			season = &MissingSeason{
				SeasonNumber:    seasonNumber,
				MissingEpisodes: []*MissingEpisode{},
			}
			seasonMap[seriesID][seasonNumber] = season
		}

		// Add episode
		episode := s.rowToMissingEpisode(row)
		season.MissingEpisodes = append(season.MissingEpisodes, episode)
		series.MissingCount++
	}

	// Build final result with sorted seasons
	result := make([]*MissingSeries, 0, len(seriesMap))
	for seriesID, series := range seriesMap {
		seasons := seasonMap[seriesID]
		series.MissingSeasons = make([]*MissingSeason, 0, len(seasons))
		for _, season := range seasons {
			series.MissingSeasons = append(series.MissingSeasons, season)
		}
		// Sort seasons by season number
		sortSeasons(series.MissingSeasons)
		result = append(result, series)
	}

	// Sort series by title
	sortSeries(result)

	return result, nil
}

// GetMissingCounts returns the count of missing movies and episodes.
func (s *Service) GetMissingCounts(ctx context.Context) (*MissingCounts, error) {
	movieCount, err := s.queries.CountMissingMovies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count missing movies: %w", err)
	}

	episodeCount, err := s.queries.CountMissingEpisodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count missing episodes: %w", err)
	}

	return &MissingCounts{
		Movies:   movieCount,
		Episodes: episodeCount,
	}, nil
}

// GetUpgradableMovies returns all movies with files below the quality cutoff.
func (s *Service) GetUpgradableMovies(ctx context.Context) ([]*UpgradableMovie, error) {
	rows, err := s.queries.ListUpgradableMoviesWithQuality(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list upgradable movies: %w", err)
	}

	movies := make([]*UpgradableMovie, len(rows))
	for i, row := range rows {
		movies[i] = s.rowToUpgradableMovie(row)
	}
	return movies, nil
}

// GetUpgradableSeries returns all series with upgradable episodes, grouped hierarchically.
func (s *Service) GetUpgradableSeries(ctx context.Context) ([]*UpgradableSeries, error) {
	rows, err := s.queries.ListUpgradableEpisodesWithQuality(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list upgradable episodes: %w", err)
	}

	seriesMap := make(map[int64]*UpgradableSeries)
	seasonMap := make(map[int64]map[int]*UpgradableSeason)

	for _, row := range rows {
		seriesID := row.SeriesID
		seasonNumber := int(row.SeasonNumber)

		series, exists := seriesMap[seriesID]
		if !exists {
			series = &UpgradableSeries{
				ID:                seriesID,
				Title:             row.SeriesTitle,
				UpgradableSeasons: []*UpgradableSeason{},
			}
			if row.SeriesYear.Valid {
				series.Year = int(row.SeriesYear.Int64)
			}
			if row.SeriesTvdbID.Valid {
				series.TvdbID = int(row.SeriesTvdbID.Int64)
			}
			if row.SeriesTmdbID.Valid {
				series.TmdbID = int(row.SeriesTmdbID.Int64)
			}
			if row.SeriesImdbID.Valid {
				series.ImdbID = row.SeriesImdbID.String
			}
			if row.SeriesQualityProfileID.Valid {
				series.QualityProfileID = row.SeriesQualityProfileID.Int64
			}
			seriesMap[seriesID] = series
			seasonMap[seriesID] = make(map[int]*UpgradableSeason)
		}

		season, exists := seasonMap[seriesID][seasonNumber]
		if !exists {
			season = &UpgradableSeason{
				SeasonNumber:       seasonNumber,
				UpgradableEpisodes: []*UpgradableEpisode{},
			}
			seasonMap[seriesID][seasonNumber] = season
		}

		episode := s.rowToUpgradableEpisode(row)
		season.UpgradableEpisodes = append(season.UpgradableEpisodes, episode)
		series.UpgradableCount++
	}

	result := make([]*UpgradableSeries, 0, len(seriesMap))
	for seriesID, series := range seriesMap {
		seasons := seasonMap[seriesID]
		series.UpgradableSeasons = make([]*UpgradableSeason, 0, len(seasons))
		for _, season := range seasons {
			series.UpgradableSeasons = append(series.UpgradableSeasons, season)
		}
		sortUpgradableSeasons(series.UpgradableSeasons)
		result = append(result, series)
	}

	sortUpgradableSeries(result)
	return result, nil
}

// GetUpgradableCounts returns counts of upgradable movies and episodes.
func (s *Service) GetUpgradableCounts(ctx context.Context) (*UpgradableCounts, error) {
	movieCount, err := s.queries.CountMovieUpgradeCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count upgradable movies: %w", err)
	}

	episodeCount, err := s.queries.CountEpisodeUpgradeCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count upgradable episodes: %w", err)
	}

	return &UpgradableCounts{
		Movies:   movieCount,
		Episodes: episodeCount,
	}, nil
}

// rowToMissingMovie converts a database row to a MissingMovie.
func (s *Service) rowToMissingMovie(row *sqlc.Movie) *MissingMovie {
	m := &MissingMovie{
		ID:    row.ID,
		Title: row.Title,
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
	if row.Path.Valid {
		m.Path = row.Path.String
	}
	if row.ReleaseDate.Valid {
		m.ReleaseDate = &row.ReleaseDate.Time
	}
	if row.PhysicalReleaseDate.Valid {
		m.PhysicalReleaseDate = &row.PhysicalReleaseDate.Time
	}
	if row.QualityProfileID.Valid {
		m.QualityProfileID = row.QualityProfileID.Int64
	}

	return m
}

// rowToMissingEpisode converts a database row to a MissingEpisode.
func (s *Service) rowToMissingEpisode(row *sqlc.ListMissingEpisodesRow) *MissingEpisode {
	e := &MissingEpisode{
		ID:            row.ID,
		SeriesID:      row.SeriesID,
		SeasonNumber:  int(row.SeasonNumber),
		EpisodeNumber: int(row.EpisodeNumber),
		SeriesTitle:   row.SeriesTitle,
	}

	if row.Title.Valid {
		e.Title = row.Title.String
	}
	if row.AirDate.Valid {
		e.AirDate = &row.AirDate.Time
	}
	if row.SeriesTvdbID.Valid {
		e.SeriesTvdbID = int(row.SeriesTvdbID.Int64)
	}
	if row.SeriesTmdbID.Valid {
		e.SeriesTmdbID = int(row.SeriesTmdbID.Int64)
	}
	if row.SeriesImdbID.Valid {
		e.SeriesImdbID = row.SeriesImdbID.String
	}
	if row.SeriesYear.Valid {
		e.SeriesYear = int(row.SeriesYear.Int64)
	}

	return e
}

func (s *Service) rowToUpgradableMovie(row *sqlc.ListUpgradableMoviesWithQualityRow) *UpgradableMovie {
	m := &UpgradableMovie{
		ID:    row.ID,
		Title: row.Title,
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
	if row.Path.Valid {
		m.Path = row.Path.String
	}
	if row.ReleaseDate.Valid {
		m.ReleaseDate = &row.ReleaseDate.Time
	}
	if row.PhysicalReleaseDate.Valid {
		m.PhysicalReleaseDate = &row.PhysicalReleaseDate.Time
	}
	if row.QualityProfileID.Valid {
		m.QualityProfileID = row.QualityProfileID.Int64
	}
	if row.CurrentQualityID.Valid {
		m.CurrentQualityID = int(row.CurrentQualityID.Int64)
	}
	return m
}

func (s *Service) rowToUpgradableEpisode(row *sqlc.ListUpgradableEpisodesWithQualityRow) *UpgradableEpisode {
	e := &UpgradableEpisode{
		ID:            row.ID,
		SeriesID:      row.SeriesID,
		SeasonNumber:  int(row.SeasonNumber),
		EpisodeNumber: int(row.EpisodeNumber),
		SeriesTitle:   row.SeriesTitle,
	}
	if row.Title.Valid {
		e.Title = row.Title.String
	}
	if row.AirDate.Valid {
		e.AirDate = &row.AirDate.Time
	}
	if row.SeriesTvdbID.Valid {
		e.SeriesTvdbID = int(row.SeriesTvdbID.Int64)
	}
	if row.SeriesTmdbID.Valid {
		e.SeriesTmdbID = int(row.SeriesTmdbID.Int64)
	}
	if row.SeriesImdbID.Valid {
		e.SeriesImdbID = row.SeriesImdbID.String
	}
	if row.SeriesYear.Valid {
		e.SeriesYear = int(row.SeriesYear.Int64)
	}
	if row.CurrentQualityID.Valid {
		e.CurrentQualityID = int(row.CurrentQualityID.Int64)
	}
	return e
}

// sortSeasons sorts seasons by season number.
func sortSeasons(seasons []*MissingSeason) {
	for i := 0; i < len(seasons)-1; i++ {
		for j := i + 1; j < len(seasons); j++ {
			if seasons[i].SeasonNumber > seasons[j].SeasonNumber {
				seasons[i], seasons[j] = seasons[j], seasons[i]
			}
		}
	}
}

// sortSeries sorts series by title.
func sortSeries(series []*MissingSeries) {
	for i := 0; i < len(series)-1; i++ {
		for j := i + 1; j < len(series); j++ {
			if series[i].Title > series[j].Title {
				series[i], series[j] = series[j], series[i]
			}
		}
	}
}

func sortUpgradableSeasons(seasons []*UpgradableSeason) {
	for i := 0; i < len(seasons)-1; i++ {
		for j := i + 1; j < len(seasons); j++ {
			if seasons[i].SeasonNumber > seasons[j].SeasonNumber {
				seasons[i], seasons[j] = seasons[j], seasons[i]
			}
		}
	}
}

func sortUpgradableSeries(series []*UpgradableSeries) {
	for i := 0; i < len(series)-1; i++ {
		for j := i + 1; j < len(series); j++ {
			if series[i].Title > series[j].Title {
				series[i], series[j] = series[j], series[i]
			}
		}
	}
}
