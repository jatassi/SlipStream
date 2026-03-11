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

	// Extra holds module-specific fields (e.g., tmdbId, year, seriesId,
	// seasonNumber, episodeNumber, network, earlyAccess). Keyed by the same
	// JSON names the frontend expects so serialization is transparent.
	Extra map[string]any `json:"extra,omitempty"`
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
	for _, mod := range s.registry.Enabled() {
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
		Extra:     make(map[string]any),
	}

	if item.Year != 0 {
		event.Extra["year"] = item.Year
	}

	if tmdb, ok := item.ExternalIDs["tmdb"]; ok {
		if id, err := strconv.Atoi(tmdb); err == nil {
			event.Extra["tmdbId"] = id
		}
	}

	if item.ParentID != 0 {
		event.Extra["seriesId"] = item.ParentID
		event.Extra["seriesTitle"] = item.ParentTitle
	}

	// Forward all module-specific extra fields (seasonNumber, episodeNumber,
	// network, earlyAccess, etc.) directly into the event Extra map.
	for k, v := range item.Extra {
		event.Extra[k] = v
	}

	return event
}
