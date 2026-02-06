package calendar

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// CalendarEvent represents a single calendar event.
type CalendarEvent struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	MediaType string `json:"mediaType"` // "movie" or "episode"
	EventType string `json:"eventType"` // "theatrical", "digital", "physical", "airDate"
	Date      string `json:"date"`      // YYYY-MM-DD
	Status    string `json:"status"`    // "missing", "available", "downloading"
	Monitored bool   `json:"monitored"`

	// Movie-specific
	TmdbID int `json:"tmdbId,omitempty"`
	Year   int `json:"year,omitempty"`

	// Episode-specific
	SeriesID      int64  `json:"seriesId,omitempty"`
	SeriesTitle   string `json:"seriesTitle,omitempty"`
	SeasonNumber  int    `json:"seasonNumber,omitempty"`
	EpisodeNumber int    `json:"episodeNumber,omitempty"`
	Network       string `json:"network,omitempty"`

	// Streaming services with early release (Apple TV+)
	EarlyAccess bool `json:"earlyAccess,omitempty"`
}

// Service provides calendar operations.
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	logger  zerolog.Logger
}

// NewService creates a new calendar service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		logger:  logger.With().Str("component", "calendar").Logger(),
	}
}

// SetDB updates the database connection used by this service.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// streamingServicesWithEarlyRelease lists networks that typically release content
// the night before the stated air date.
var streamingServicesWithEarlyRelease = map[string]bool{
	"Apple TV+": true,
	"Apple TV":  true,
}

// GetEvents returns calendar events for the specified date range.
func (s *Service) GetEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
	var events []CalendarEvent

	// Get movie events
	movieEvents, err := s.getMovieEvents(ctx, start, end)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get movie events")
		return nil, err
	}
	events = append(events, movieEvents...)

	// Get episode events
	episodeEvents, err := s.getEpisodeEvents(ctx, start, end)
	if err != nil {
		s.logger.Error().Err(err).Msg("Failed to get episode events")
		return nil, err
	}
	events = append(events, episodeEvents...)

	s.logger.Debug().
		Time("start", start).
		Time("end", end).
		Int("movieEvents", len(movieEvents)).
		Int("episodeEvents", len(episodeEvents)).
		Msg("Got calendar events")

	return events, nil
}

// getMovieEvents retrieves movie release events in the date range.
func (s *Service) getMovieEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
	startStr := start.Format("2006-01-02")
	endStr := end.Format("2006-01-02")

	rows, err := s.queries.GetMoviesInDateRange(ctx, sqlc.GetMoviesInDateRangeParams{
		FromReleaseDate:         sql.NullTime{Time: start, Valid: true},
		ToReleaseDate:           sql.NullTime{Time: end, Valid: true},
		FromPhysicalReleaseDate: sql.NullTime{Time: start, Valid: true},
		ToPhysicalReleaseDate:   sql.NullTime{Time: end, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	var events []CalendarEvent

	for _, row := range rows {
		// Check if movie has any files
		hasFile := false
		fileCount, err := s.queries.CountMovieFiles(ctx, row.ID)
		if err == nil && fileCount > 0 {
			hasFile = true
		}

		status := row.Status
		if hasFile {
			status = "available"
		}

		year := 0
		if row.Year.Valid {
			year = int(row.Year.Int64)
		}

		tmdbID := 0
		if row.TmdbID.Valid {
			tmdbID = int(row.TmdbID.Int64)
		}

		// Create event for digital/streaming release (ReleaseDate stores digital release)
		if row.ReleaseDate.Valid {
			dateStr := row.ReleaseDate.Time.Format("2006-01-02")
			if dateStr >= startStr && dateStr <= endStr {
				events = append(events, CalendarEvent{
					ID:        row.ID,
					Title:     row.Title,
					MediaType: "movie",
					EventType: "digital",
					Date:      dateStr,
					Status:    status,
					Monitored: row.Monitored == 1,
					TmdbID:    tmdbID,
					Year:      year,
				})
			}
		}

		// Create event for physical/Bluray release
		if row.PhysicalReleaseDate.Valid {
			dateStr := row.PhysicalReleaseDate.Time.Format("2006-01-02")
			if dateStr >= startStr && dateStr <= endStr {
				events = append(events, CalendarEvent{
					ID:        row.ID,
					Title:     row.Title,
					MediaType: "movie",
					EventType: "physical",
					Date:      dateStr,
					Status:    status,
					Monitored: row.Monitored == 1,
					TmdbID:    tmdbID,
					Year:      year,
				})
			}
		}
	}

	return events, nil
}

// seasonKey is used to group episodes by series, season, and date.
type seasonKey struct {
	SeriesID     int64
	SeasonNumber int64
	Date         string
}

// getEpisodeEvents retrieves episode air date events in the date range.
// If multiple episodes from the same season air on the same day, they are grouped
// into a single "Season X" entry (Netflix-style release handling).
func (s *Service) getEpisodeEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
	rows, err := s.queries.GetEpisodesInDateRange(ctx, sqlc.GetEpisodesInDateRangeParams{
		FromAirDate: sql.NullTime{Time: start, Valid: true},
		ToAirDate:   sql.NullTime{Time: end, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	// Group episodes by series+season+date to detect full season releases
	episodesBySeasonDate := make(map[seasonKey][]*sqlc.GetEpisodesInDateRangeRow)
	for _, row := range rows {
		key := seasonKey{
			SeriesID:     row.SeriesID,
			SeasonNumber: row.SeasonNumber,
			Date:         row.AirDate.Time.Format("2006-01-02"),
		}
		episodesBySeasonDate[key] = append(episodesBySeasonDate[key], row)
	}

	var events []CalendarEvent

	// Process grouped episodes
	for key, episodes := range episodesBySeasonDate {
		// If 3 or more episodes from the same season air on the same day,
		// treat it as a full season release
		if len(episodes) >= 3 {
			event := s.createSeasonReleaseEvent(ctx, key, episodes)
			events = append(events, event)
		} else {
			// Individual episode events
			for _, row := range episodes {
				event := s.createEpisodeEvent(ctx, row)
				events = append(events, event)
			}
		}
	}

	return events, nil
}

// createSeasonReleaseEvent creates a single calendar event for a full season release.
func (s *Service) createSeasonReleaseEvent(ctx context.Context, key seasonKey, episodes []*sqlc.GetEpisodesInDateRangeRow) CalendarEvent {
	if len(episodes) == 0 {
		return CalendarEvent{}
	}

	firstEp := episodes[0]

	// Check if any episode has files (season is "available" if all have files)
	availableCount := 0
	for _, ep := range episodes {
		files, err := s.queries.ListEpisodeFilesByEpisode(ctx, ep.ID)
		if err == nil && len(files) > 0 {
			availableCount++
		}
	}

	status := "missing"
	if availableCount == len(episodes) {
		status = "available"
	} else if availableCount > 0 {
		status = "downloading" // Partial - some episodes available
	}

	network := ""
	if firstEp.Network.Valid {
		network = firstEp.Network.String
	}

	earlyAccess := streamingServicesWithEarlyRelease[network]

	seriesTmdbID := 0
	if firstEp.SeriesTmdbID.Valid {
		seriesTmdbID = int(firstEp.SeriesTmdbID.Int64)
	}

	return CalendarEvent{
		ID:            firstEp.ID, // Use first episode's ID
		Title:         fmt.Sprintf("Season %d", key.SeasonNumber),
		MediaType:     "episode",
		EventType:     "airDate",
		Date:          key.Date,
		Status:        status,
		Monitored:     firstEp.Monitored == 1,
		SeriesID:      key.SeriesID,
		SeriesTitle:   firstEp.SeriesTitle,
		SeasonNumber:  int(key.SeasonNumber),
		EpisodeNumber: len(episodes), // Store episode count in EpisodeNumber field
		Network:       network,
		TmdbID:        seriesTmdbID,
		EarlyAccess:   earlyAccess,
	}
}

// createEpisodeEvent creates a calendar event for a single episode.
func (s *Service) createEpisodeEvent(ctx context.Context, row *sqlc.GetEpisodesInDateRangeRow) CalendarEvent {
	// Check if episode has any files
	hasFile := false
	files, err := s.queries.ListEpisodeFilesByEpisode(ctx, row.ID)
	if err == nil && len(files) > 0 {
		hasFile = true
	}

	status := "missing"
	if hasFile {
		status = "available"
	}

	network := ""
	if row.Network.Valid {
		network = row.Network.String
	}

	// Check for early access (streaming services like Apple TV+)
	earlyAccess := streamingServicesWithEarlyRelease[network]

	seriesTmdbID := 0
	if row.SeriesTmdbID.Valid {
		seriesTmdbID = int(row.SeriesTmdbID.Int64)
	}

	title := row.Title.String
	if title == "" {
		title = fmt.Sprintf("Episode %d", row.EpisodeNumber)
	}

	return CalendarEvent{
		ID:            row.ID,
		Title:         title,
		MediaType:     "episode",
		EventType:     "airDate",
		Date:          row.AirDate.Time.Format("2006-01-02"),
		Status:        status,
		Monitored:     row.Monitored == 1,
		SeriesID:      row.SeriesID,
		SeriesTitle:   row.SeriesTitle,
		SeasonNumber:  int(row.SeasonNumber),
		EpisodeNumber: int(row.EpisodeNumber),
		Network:       network,
		TmdbID:        seriesTmdbID,
		EarlyAccess:   earlyAccess,
	}
}
