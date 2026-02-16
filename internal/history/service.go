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
	Broadcast(msgType string, payload interface{})
}

// Service provides history management functionality.
type Service struct {
	db          *sql.DB
	queries     *sqlc.Queries
	logger      *zerolog.Logger
	broadcaster Broadcaster
}

// NewService creates a new history service.
func NewService(db *sql.DB, logger *zerolog.Logger) *Service {
	subLogger := logger.With().Str("component", "history").Logger()
	return &Service{
		db:      db,
		queries: sqlc.New(db),
		logger:  &subLogger,
	}
}

// SetDB updates the database connection used by this service.
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
	s.queries = sqlc.New(db)
}

// SetBroadcaster sets the WebSocket broadcaster for real-time history events.
func (s *Service) SetBroadcaster(broadcaster Broadcaster) {
	s.broadcaster = broadcaster
}

// Create creates a new history entry.
func (s *Service) Create(ctx context.Context, input *CreateInput) (*Entry, error) {
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
		s.broadcaster.Broadcast("history:added", entry)
	}

	return entry, nil
}

// List lists history entries with pagination and filtering.
func (s *Service) List(ctx context.Context, opts *ListOptions) (*ListResponse, error) {
	normalized := normalizeListOptions(opts)
	offset := int64((normalized.Page - 1) * normalized.PageSize)
	limit := int64(normalized.PageSize)

	hasFilters := normalized.EventType != "" || normalized.MediaType != "" || normalized.After != "" || normalized.Before != ""

	var rows []*sqlc.History
	var totalCount int64
	var err error

	if hasFilters {
		rows, totalCount, err = s.listWithFilters(ctx, &normalized, limit, offset)
	} else {
		rows, totalCount, err = s.listAll(ctx, limit, offset)
	}

	if err != nil {
		return nil, err
	}

	entries := make([]*Entry, 0, len(rows))
	for _, row := range rows {
		entry := s.rowToEntry(row)
		s.enrichEntry(ctx, entry)
		entries = append(entries, entry)
	}

	totalPages := int(totalCount) / normalized.PageSize
	if int(totalCount)%normalized.PageSize > 0 {
		totalPages++
	}

	return &ListResponse{
		Items:      entries,
		Page:       normalized.Page,
		PageSize:   normalized.PageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
	}, nil
}

func normalizeListOptions(opts *ListOptions) ListOptions {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PageSize < 1 {
		opts.PageSize = 50
	}
	if opts.PageSize > 100 {
		opts.PageSize = 100
	}
	return *opts
}

func (s *Service) listAll(ctx context.Context, limit, offset int64) ([]*sqlc.History, int64, error) {
	rows, err := s.queries.ListHistoryPaginated(ctx, sqlc.ListHistoryPaginatedParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, err
	}

	totalCount, err := s.queries.CountHistory(ctx)
	if err != nil {
		return nil, 0, err
	}

	return rows, totalCount, nil
}

func (s *Service) listWithFilters(ctx context.Context, opts *ListOptions, limit, offset int64) ([]*sqlc.History, int64, error) {
	afterTime := parseTimeFilter(opts.After)
	beforeTime := parseTimeFilter(opts.Before)

	rows, err := s.queries.ListHistoryFiltered(ctx, sqlc.ListHistoryFilteredParams{
		Column1:     opts.EventType,
		Column2:     sql.NullString{String: opts.EventType, Valid: opts.EventType != ""},
		Column3:     opts.MediaType,
		MediaType:   opts.MediaType,
		Column5:     opts.After,
		CreatedAt:   afterTime,
		Column7:     opts.Before,
		CreatedAt_2: beforeTime,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		return nil, 0, err
	}

	totalCount, err := s.queries.CountHistoryFiltered(ctx, sqlc.CountHistoryFilteredParams{
		Column1:     opts.EventType,
		Column2:     sql.NullString{String: opts.EventType, Valid: opts.EventType != ""},
		Column3:     opts.MediaType,
		MediaType:   opts.MediaType,
		Column5:     opts.After,
		CreatedAt:   afterTime,
		Column7:     opts.Before,
		CreatedAt_2: beforeTime,
	})
	if err != nil {
		return nil, 0, err
	}

	return rows, totalCount, nil
}

func parseTimeFilter(filter string) sql.NullTime {
	if filter == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse(time.RFC3339, filter)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
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
	s.enrichMediaInfo(ctx, entry)
	s.enrichQualityInfo(entry)
}

func (s *Service) enrichMediaInfo(ctx context.Context, entry *Entry) {
	switch entry.MediaType {
	case MediaTypeMovie:
		s.enrichMovieInfo(ctx, entry)
	case MediaTypeEpisode:
		s.enrichEpisodeInfo(ctx, entry)
	}
}

func (s *Service) enrichMovieInfo(ctx context.Context, entry *Entry) {
	movie, err := s.queries.GetMovie(ctx, entry.MediaID)
	if err != nil {
		return
	}

	entry.MediaTitle = movie.Title
	if movie.Year.Valid && movie.Year.Int64 > 0 {
		year := movie.Year.Int64
		entry.Year = &year
	}
}

func (s *Service) enrichEpisodeInfo(ctx context.Context, entry *Entry) {
	episode, err := s.queries.GetEpisode(ctx, entry.MediaID)
	if err != nil {
		return
	}

	seriesID := episode.SeriesID
	entry.SeriesID = &seriesID

	series, err := s.queries.GetSeries(ctx, seriesID)
	if err != nil {
		return
	}

	entry.MediaTitle = series.Title
	entry.MediaQualifier = fmt.Sprintf("S%02dE%02d", episode.SeasonNumber, episode.EpisodeNumber)
}

func (s *Service) enrichQualityInfo(entry *Entry) {
	if !shouldEnrichQuality(entry) {
		return
	}

	backfillMissingQualityFields(entry)
}

func shouldEnrichQuality(entry *Entry) bool {
	if entry.EventType != EventTypeImported || entry.Data == nil {
		return false
	}

	isUpgrade, _ := entry.Data["isUpgrade"].(bool)
	if !isUpgrade {
		return false
	}

	_, hasPrev := entry.Data["previousQuality"]
	_, hasNew := entry.Data["newQuality"]
	return !hasPrev || !hasNew
}

func backfillMissingQualityFields(entry *Entry) {
	_, hasPrev := entry.Data["previousQuality"]
	if !hasPrev {
		tryBackfillPreviousQuality(entry)
	}

	_, hasNew := entry.Data["newQuality"]
	if !hasNew {
		tryBackfillNewQuality(entry)
	}
}

func tryBackfillPreviousQuality(entry *Entry) {
	prev, _ := entry.Data["previousFile"].(string)
	if prev == "" {
		return
	}

	if name := qualityNameFromPath(prev); name != "" {
		entry.Data["previousQuality"] = name
	}
}

func tryBackfillNewQuality(entry *Entry) {
	dest, _ := entry.Data["destinationPath"].(string)
	if dest == "" {
		return
	}

	if name := qualityNameFromPath(dest); name != "" {
		entry.Data["newQuality"] = name
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
func (s *Service) LogAutoSearchDownload(ctx context.Context, mediaType MediaType, mediaID int64, quality string, data *AutoSearchDownloadData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal autosearch download data")
		dataMap = nil
	}

	_, err = s.Create(ctx, &CreateInput{
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

	_, err = s.Create(ctx, &CreateInput{
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

	_, err = s.Create(ctx, &CreateInput{
		EventType: EventTypeStatusChanged,
		MediaType: mediaType,
		MediaID:   mediaID,
		Data:      dataMap,
	})
	return err
}

// LogSlotAssigned logs when a file is assigned to a slot.
func (s *Service) LogSlotAssigned(ctx context.Context, mediaType MediaType, mediaID int64, data SlotEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal slot assigned data")
		dataMap = nil
	}

	_, err = s.Create(ctx, &CreateInput{
		EventType: EventTypeSlotAssigned,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    "slot",
		Data:      dataMap,
	})
	return err
}

// LogSlotReassigned logs when a file is moved to a different slot.
func (s *Service) LogSlotReassigned(ctx context.Context, mediaType MediaType, mediaID int64, data SlotEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal slot reassigned data")
		dataMap = nil
	}

	_, err = s.Create(ctx, &CreateInput{
		EventType: EventTypeSlotReassigned,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    "slot",
		Data:      dataMap,
	})
	return err
}

// LogSlotUnassigned logs when a file is removed from a slot.
func (s *Service) LogSlotUnassigned(ctx context.Context, mediaType MediaType, mediaID int64, data SlotEventData) error {
	dataMap, err := ToJSON(data)
	if err != nil {
		s.logger.Warn().Err(err).Msg("Failed to marshal slot unassigned data")
		dataMap = nil
	}

	_, err = s.Create(ctx, &CreateInput{
		EventType: EventTypeSlotUnassigned,
		MediaType: mediaType,
		MediaID:   mediaID,
		Source:    "slot",
		Data:      dataMap,
	})
	return err
}
