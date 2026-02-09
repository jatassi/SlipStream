package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/indexer/scoring"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// Broadcaster defines the interface for sending WebSocket messages.
type Broadcaster interface {
	Broadcast(msgType string, payload interface{}) error
}

// Service provides history management functionality.
type Service struct {
	db          *sql.DB
	queries     *sqlc.Queries
	logger      zerolog.Logger
	broadcaster Broadcaster
}

// NewService creates a new history service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		logger:  logger.With().Str("component", "history").Logger(),
	}
}

// SetDB updates the database connection used by this service.
// This is called when switching between production and development databases.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// SetBroadcaster sets the WebSocket broadcaster for real-time history events.
func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

// Create creates a new history entry.
func (s *Service) Create(ctx context.Context, input CreateInput) (*Entry, error) {
	var dataJSON sql.NullString
	if input.Data != nil {
		bytes, err := json.Marshal(input.Data)
		if err != nil {
			return nil, err
		}
		dataJSON = sql.NullString{String: string(bytes), Valid: true}
	}

	row, err := s.queries.CreateHistoryEntry(ctx, sqlc.CreateHistoryEntryParams{
		EventType: string(input.EventType),
		MediaType: string(input.MediaType),
		MediaID:   input.MediaID,
		Source:    sql.NullString{String: input.Source, Valid: input.Source != ""},
		Quality:   sql.NullString{String: input.Quality, Valid: input.Quality != ""},
		Data:      dataJSON,
	})
	if err != nil {
		return nil, err
	}

	entry := s.rowToEntry(row)
	s.enrichEntry(ctx, entry)

	if s.broadcaster != nil {
		_ = s.broadcaster.Broadcast("history:added", entry)
	}

	return entry, nil
}

// List lists history entries with pagination and filtering.
func (s *Service) List(ctx context.Context, opts ListOptions) (*ListResponse, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 50
	}
	if opts.PageSize > 100 {
		opts.PageSize = 100
	}

	offset := int64((opts.Page - 1) * opts.PageSize)
	limit := int64(opts.PageSize)

	var rows []*sqlc.History
	var err error
	var totalCount int64

	hasFilters := opts.EventType != "" || opts.MediaType != "" || opts.After != "" || opts.Before != ""

	if hasFilters {
		eventFilter := opts.EventType
		mediaFilter := opts.MediaType
		afterFilter := opts.After
		beforeFilter := opts.Before

		afterTime := sql.NullTime{}
		if afterFilter != "" {
			if t, err := time.Parse(time.RFC3339, afterFilter); err == nil {
				afterTime = sql.NullTime{Time: t, Valid: true}
			}
		}
		beforeTime := sql.NullTime{}
		if beforeFilter != "" {
			if t, err := time.Parse(time.RFC3339, beforeFilter); err == nil {
				beforeTime = sql.NullTime{Time: t, Valid: true}
			}
		}

		rows, err = s.queries.ListHistoryFiltered(ctx, sqlc.ListHistoryFilteredParams{
			Column1:     eventFilter,
			EventType:   eventFilter,
			Column3:     mediaFilter,
			MediaType:   mediaFilter,
			Column5:     afterFilter,
			CreatedAt:   afterTime,
			Column7:     beforeFilter,
			CreatedAt_2: beforeTime,
			Limit:       limit,
			Offset:      offset,
		})
		if err != nil {
			return nil, err
		}

		totalCount, err = s.queries.CountHistoryFiltered(ctx, sqlc.CountHistoryFilteredParams{
			Column1:     eventFilter,
			EventType:   eventFilter,
			Column3:     mediaFilter,
			MediaType:   mediaFilter,
			Column5:     afterFilter,
			CreatedAt:   afterTime,
			Column7:     beforeFilter,
			CreatedAt_2: beforeTime,
		})
		if err != nil {
			return nil, err
		}
	} else {
		rows, err = s.queries.ListHistoryPaginated(ctx, sqlc.ListHistoryPaginatedParams{
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, err
		}

		totalCount, err = s.queries.CountHistory(ctx)
		if err != nil {
			return nil, err
		}
	}

	entries := make([]*Entry, 0, len(rows))
	for _, row := range rows {
		entry := s.rowToEntry(row)
		s.enrichEntry(ctx, entry)
		entries = append(entries, entry)
	}

	totalPages := int(totalCount) / opts.PageSize
	if int(totalCount)%opts.PageSize > 0 {
		totalPages++
	}

	return &ListResponse{
		Items:      entries,
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

// ListByMedia lists history for a specific media item.
func (s *Service) ListByMedia(ctx context.Context, mediaType MediaType, mediaID int64) ([]*Entry, error) {
	rows, err := s.queries.ListHistoryByMedia(ctx, sqlc.ListHistoryByMediaParams{
		MediaType: string(mediaType),
		MediaID:   mediaID,
	})
	if err != nil {
		return nil, err
	}

	entries := make([]*Entry, 0, len(rows))
	for _, row := range rows {
		entry := s.rowToEntry(row)
		s.enrichEntry(ctx, entry)
		entries = append(entries, entry)
	}

	return entries, nil
}

// DeleteAll deletes all history entries.
func (s *Service) DeleteAll(ctx context.Context) error {
	return s.queries.DeleteAllHistory(ctx)
}

// rowToEntry converts a database row to an Entry.
func (s *Service) rowToEntry(row *sqlc.History) *Entry {
	entry := &Entry{
		ID:        row.ID,
		EventType: EventType(row.EventType),
		MediaType: MediaType(row.MediaType),
		MediaID:   row.MediaID,
	}

	if row.Source.Valid {
		entry.Source = row.Source.String
	}
	if row.Quality.Valid {
		entry.Quality = row.Quality.String
	}
	if row.Data.Valid {
		var data map[string]any
		if err := json.Unmarshal([]byte(row.Data.String), &data); err == nil {
			entry.Data = data
		}
	}
	if row.CreatedAt.Valid {
		entry.CreatedAt = row.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
	}

	return entry
}

// enrichEntry adds media title, series ID, and year to the entry.
func (s *Service) enrichEntry(ctx context.Context, entry *Entry) {
	switch entry.MediaType {
	case MediaTypeMovie:
		movie, err := s.queries.GetMovie(ctx, entry.MediaID)
		if err == nil {
			entry.MediaTitle = movie.Title
			if movie.Year.Valid && movie.Year.Int64 > 0 {
				year := movie.Year.Int64
				entry.Year = &year
			}
		}
	case MediaTypeEpisode:
		episode, err := s.queries.GetEpisode(ctx, entry.MediaID)
		if err == nil {
			seriesID := episode.SeriesID
			entry.SeriesID = &seriesID
			series, err := s.queries.GetSeries(ctx, seriesID)
			if err == nil {
				epCode := fmt.Sprintf("S%02dE%02d", episode.SeasonNumber, episode.EpisodeNumber)
				entry.MediaTitle = series.Title + " - " + epCode
				if episode.Title.Valid && episode.Title.String != "" {
					entry.MediaTitle += " - " + episode.Title.String
				}
			}
		}
	}

	// Backfill quality names for imported upgrade events that lack them
	if entry.EventType == EventTypeImported && entry.Data != nil {
		isUpgrade, _ := entry.Data["isUpgrade"].(bool)
		_, hasPrev := entry.Data["previousQuality"]
		_, hasNew := entry.Data["newQuality"]
		if isUpgrade && (!hasPrev || !hasNew) {
			if !hasPrev {
				if prev, _ := entry.Data["previousFile"].(string); prev != "" {
					if name := qualityNameFromPath(prev); name != "" {
						entry.Data["previousQuality"] = name
					}
				}
			}
			if !hasNew {
				if dest, _ := entry.Data["destinationPath"].(string); dest != "" {
					if name := qualityNameFromPath(dest); name != "" {
						entry.Data["newQuality"] = name
					}
				}
			}
		}
	}
}

// qualityNameFromPath derives a quality name (e.g. "Bluray-1080p") from a file path.
func qualityNameFromPath(path string) string {
	parsed := scanner.ParsePath(path)
	res, _ := strconv.Atoi(strings.TrimSuffix(parsed.Quality, "p"))
	if result := scoring.MatchQuality(parsed.Source, res); result.Quality != nil {
		return result.Quality.Name
	}
	return ""
}

// LogAutoSearchDownload logs a successful autosearch download.
func (s *Service) LogAutoSearchDownload(ctx context.Context, mediaType MediaType, mediaID int64, quality string, data AutoSearchDownloadData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal autosearch download data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeAutoSearchDownload,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    data.Source,
		Quality:   quality,
		Data:      dataMap,
	})
	return err
}

// LogAutoSearchFailed logs an autosearch failure (not "not found").
func (s *Service) LogAutoSearchFailed(ctx context.Context, mediaType MediaType, mediaID int64, data AutoSearchFailedData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal autosearch failed data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeAutoSearchFailed,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    data.Source,
		Data:      dataMap,
	})
	return err
}


// LogStatusChanged logs a status transition not covered by existing event types.
func (s *Service) LogStatusChanged(ctx context.Context, mediaType MediaType, mediaID int64, data StatusChangedData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal status changed data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeStatusChanged,
		MediaType: mediaType,
		MediaID:   mediaID,
		Data:      dataMap,
	})
	return err
}

// LogSlotAssigned logs when a file is assigned to a slot.
// Req 17.1.1: Log all slot-related events
func (s *Service) LogSlotAssigned(ctx context.Context, mediaType MediaType, mediaID int64, data SlotEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal slot assigned data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeSlotAssigned,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    "slot",
		Data:      dataMap,
	})
	return err
}

// LogSlotReassigned logs when a file is moved to a different slot.
// Req 17.1.1: Log all slot-related events
func (s *Service) LogSlotReassigned(ctx context.Context, mediaType MediaType, mediaID int64, data SlotEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal slot reassigned data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeSlotReassigned,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    "slot",
		Data:      dataMap,
	})
	return err
}

// LogSlotUnassigned logs when a file is removed from a slot.
// Req 17.1.1: Log all slot-related events
func (s *Service) LogSlotUnassigned(ctx context.Context, mediaType MediaType, mediaID int64, data SlotEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal slot unassigned data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeSlotUnassigned,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    "slot",
		Data:      dataMap,
	})
	return err
}
