package history

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
)

// Service provides history management functionality.
type Service struct {
	db      *sql.DB
	queries *sqlc.Queries
	logger  zerolog.Logger
}

// NewService creates a new history service.
func NewService(db *sql.DB, logger zerolog.Logger) *Service {
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		logger:  logger.With().Str("component", "history").Logger(),
	}
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

	return s.rowToEntry(row), nil
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

	// Use filtered query if filters provided
	if opts.EventType != "" || opts.MediaType != "" {
		eventFilter := opts.EventType
		mediaFilter := opts.MediaType

		rows, err = s.queries.ListHistoryFiltered(ctx, sqlc.ListHistoryFilteredParams{
			Column1:   eventFilter,
			EventType: eventFilter,
			Column3:   mediaFilter,
			MediaType: mediaFilter,
			Limit:     limit,
			Offset:    offset,
		})
		if err != nil {
			return nil, err
		}

		totalCount, err = s.queries.CountHistoryFiltered(ctx, sqlc.CountHistoryFilteredParams{
			Column1:   eventFilter,
			EventType: eventFilter,
			Column3:   mediaFilter,
			MediaType: mediaFilter,
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

// enrichEntry adds media title to the entry.
func (s *Service) enrichEntry(ctx context.Context, entry *Entry) {
	switch entry.MediaType {
	case MediaTypeMovie:
		movie, err := s.queries.GetMovie(ctx, entry.MediaID)
		if err == nil {
			entry.MediaTitle = movie.Title
		}
	case MediaTypeEpisode:
		episode, err := s.queries.GetEpisode(ctx, entry.MediaID)
		if err == nil {
			series, err := s.queries.GetSeries(ctx, episode.SeriesID)
			if err == nil {
				entry.MediaTitle = series.Title
				if episode.Title.Valid {
					entry.MediaTitle += " - " + episode.Title.String
				}
			}
		}
	}
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

// LogAutoSearchUpgrade logs a quality upgrade.
func (s *Service) LogAutoSearchUpgrade(ctx context.Context, mediaType MediaType, mediaID int64, quality string, data AutoSearchUpgradeData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal autosearch upgrade data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeAutoSearchUpgrade,
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

// LogImportStarted logs when an import process begins.
func (s *Service) LogImportStarted(ctx context.Context, mediaType MediaType, mediaID int64, data ImportEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal import started data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeImportStarted,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    data.Source,
		Quality:   data.Quality,
		Data:      dataMap,
	})
	return err
}

// LogImportCompleted logs a successful import.
func (s *Service) LogImportCompleted(ctx context.Context, mediaType MediaType, mediaID int64, data ImportEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal import completed data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeImportCompleted,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    data.Source,
		Quality:   data.Quality,
		Data:      dataMap,
	})
	return err
}

// LogImportFailed logs a failed import.
func (s *Service) LogImportFailed(ctx context.Context, mediaType MediaType, mediaID int64, data ImportEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal import failed data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeImportFailed,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    data.Source,
		Quality:   data.Quality,
		Data:      dataMap,
	})
	return err
}

// LogImportUpgrade logs an import that upgraded an existing file.
func (s *Service) LogImportUpgrade(ctx context.Context, mediaType MediaType, mediaID int64, data ImportEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal import upgrade data")
		dataMap = nil
	}

	_, err = s.Create(ctx, CreateInput{
		EventType: EventTypeImportUpgrade,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    data.Source,
		Quality:   data.Quality,
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
