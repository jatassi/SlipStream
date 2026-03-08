package calendar

import (
	"context"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/module"
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
	SeasonNumber  int    `json:"seasonNumber"`
	EpisodeNumber int    `json:"episodeNumber"`
	Network       string `json:"network,omitempty"`

	// Streaming services with early release (Apple TV+)
	EarlyAccess bool `json:"earlyAccess"`
}

// Service provides calendar operations.
type Service struct {
	logger   *zerolog.Logger
	registry *module.Registry
}

// NewService creates a new calendar service.
func NewService(registry *module.Registry, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "calendar").Logger()
	return &Service{
		registry: registry,
		logger:   &subLogger,
	}
}

// GetEvents returns calendar events for the specified date range.
// Events are fetched via module CalendarProvider implementations.
func (s *Service) GetEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
	var events []CalendarEvent
	for _, mod := range s.registry.All() {
		provider, ok := mod.(module.CalendarProvider)
		if !ok {
			continue
		}
		items, err := provider.GetItemsInDateRange(ctx, start, end)
		if err != nil {
			s.logger.Error().Err(err).Str("module", string(mod.ID())).Msg("Failed to get calendar items")
			continue
		}
		for i := range items {
			events = append(events, calendarItemToEvent(&items[i]))
		}
	}
	return events, nil
}

func calendarItemToEvent(item *module.CalendarItem) CalendarEvent {
	event := CalendarEvent{
		ID:        item.ID,
		Title:     item.Title,
		MediaType: string(item.EntityType),
		EventType: item.EventType,
		Date:      item.Date.Format("2006-01-02"),
		Status:    item.Status,
		Monitored: item.Monitored,
		Year:      item.Year,
	}

	if tmdb, ok := item.ExternalIDs["tmdb"]; ok {
		event.TmdbID, _ = strconv.Atoi(tmdb)
	}

	if item.ParentID != 0 {
		event.SeriesID = item.ParentID
		event.SeriesTitle = item.ParentTitle
	}

	if extra := item.Extra; extra != nil {
		if sn, ok := extra["seasonNumber"].(int); ok {
			event.SeasonNumber = sn
		}
		if en, ok := extra["episodeNumber"].(int); ok {
			event.EpisodeNumber = en
		}
		if net, ok := extra["network"].(string); ok {
			event.Network = net
		}
		if ea, ok := extra["earlyAccess"].(bool); ok {
			event.EarlyAccess = ea
		}
	}

	return event
}
