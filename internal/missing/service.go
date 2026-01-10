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
	ID             int64            `json:"id"`
	Title          string           `json:"title"`
	Year           int              `json:"year,omitempty"`
	TvdbID         int              `json:"tvdbId,omitempty"`
	TmdbID         int              `json:"tmdbId,omitempty"`
	ImdbID         string           `json:"imdbId,omitempty"`
	MissingCount   int              `json:"missingCount"`
	MissingSeasons []*MissingSeason `json:"missingSeasons"`
}

// MissingCounts returns counts for nav badge.
type MissingCounts struct {
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
