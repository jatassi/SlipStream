package requests

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rs/zerolog"
	"github.com/slipstream/slipstream/internal/database/sqlc"
)

type SlotInfo struct {
	SlotID    int64  `json:"slotId"`
	SlotName  string `json:"slotName"`
	HasFile   bool   `json:"hasFile"`
	QualityID *int64 `json:"qualityId,omitempty"`
}

type AvailabilityResult struct {
	InLibrary             bool       `json:"inLibrary"`
	ExistingSlots         []SlotInfo `json:"existingSlots,omitempty"`
	CanRequest            bool       `json:"canRequest"`
	ExistingRequestID     *int64     `json:"existingRequestId,omitempty"`
	ExistingRequestUserID *int64     `json:"existingRequestUserId,omitempty"`
	ExistingRequestStatus *string    `json:"existingRequestStatus,omitempty"`
	MediaID               *int64     `json:"mediaId,omitempty"`
	AddedAt               *string    `json:"addedAt,omitempty"`
}

type LibraryChecker struct {
	queries *sqlc.Queries
	logger  *zerolog.Logger
}

func NewLibraryChecker(queries *sqlc.Queries, logger *zerolog.Logger) *LibraryChecker {
	subLogger := logger.With().Str("component", "library-checker").Logger()
	return &LibraryChecker{
		queries: queries,
		logger:  &subLogger,
	}
}

func (c *LibraryChecker) SetDB(db *sql.DB) {
	c.queries = sqlc.New(db)
}

func (c *LibraryChecker) CheckMovieAvailability(ctx context.Context, tmdbID int64, userQualityProfileID *int64) (*AvailabilityResult, error) {
	result := &AvailabilityResult{
		InLibrary:     false,
		ExistingSlots: []SlotInfo{},
		CanRequest:    true,
	}

	if err := c.checkMovieInLibrary(ctx, tmdbID, userQualityProfileID, result); err != nil {
		return nil, err
	}

	if err := c.checkExistingMovieRequest(ctx, tmdbID, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *LibraryChecker) checkMovieInLibrary(ctx context.Context, tmdbID int64, userQualityProfileID *int64, result *AvailabilityResult) error {
	movie, err := c.queries.GetMovieByTmdbID(ctx, sql.NullInt64{Int64: tmdbID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if movie.ID == 0 {
		return nil
	}

	result.InLibrary = true
	result.MediaID = &movie.ID

	if movie.AddedAt.Valid {
		addedAtStr := movie.AddedAt.Time.Format("2006-01-02T15:04:05Z")
		result.AddedAt = &addedAtStr
	}

	slots, err := c.getMovieSlots(ctx, movie.ID)
	if err != nil {
		c.logger.Warn().Err(err).Int64("movieID", movie.ID).Msg("failed to get movie slots")
	} else {
		result.ExistingSlots = slots
		result.CanRequest = c.canRequestWithSlots(slots, userQualityProfileID)
	}

	return nil
}

func (c *LibraryChecker) checkExistingMovieRequest(ctx context.Context, tmdbID int64, result *AvailabilityResult) error {
	existingReq, err := c.queries.GetActiveRequestByTmdbID(ctx, sqlc.GetActiveRequestByTmdbIDParams{
		TmdbID:    sql.NullInt64{Int64: tmdbID, Valid: true},
		MediaType: MediaTypeMovie,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if existingReq != nil {
		result.ExistingRequestID = &existingReq.ID
		result.ExistingRequestUserID = &existingReq.UserID
		result.ExistingRequestStatus = &existingReq.Status
		result.CanRequest = false
	}

	return nil
}

func (c *LibraryChecker) CheckSeriesAvailability(ctx context.Context, tvdbID int64, userQualityProfileID *int64) (*AvailabilityResult, error) {
	result := &AvailabilityResult{
		InLibrary:     false,
		ExistingSlots: []SlotInfo{},
		CanRequest:    true,
	}

	series, err := c.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// Only mark as in library if we found a series (no error and valid ID)
	if err == nil && series.ID > 0 {
		result.InLibrary = true
		result.MediaID = &series.ID
		if series.AddedAt.Valid {
			addedAtStr := series.AddedAt.Time.Format("2006-01-02T15:04:05Z")
			result.AddedAt = &addedAtStr
		}
		result.CanRequest = c.canRequestWithSlots(result.ExistingSlots, userQualityProfileID)
	}

	// Use GetActiveRequestByTvdbID to find any active request (including 'available' status)
	existingReq, err := c.queries.GetActiveRequestByTvdbID(ctx, sqlc.GetActiveRequestByTvdbIDParams{
		TvdbID:    sql.NullInt64{Int64: tvdbID, Valid: true},
		MediaType: MediaTypeSeries,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil && existingReq != nil {
		result.ExistingRequestID = &existingReq.ID
		result.ExistingRequestUserID = &existingReq.UserID
		result.ExistingRequestStatus = &existingReq.Status
		result.CanRequest = false
	}

	return result, nil
}

func (c *LibraryChecker) CheckSeasonAvailability(ctx context.Context, tvdbID, seasonNumber int64, userQualityProfileID *int64) (*AvailabilityResult, error) {
	result := &AvailabilityResult{
		InLibrary:     false,
		ExistingSlots: []SlotInfo{},
		CanRequest:    true,
	}

	if err := c.checkSeasonInLibrary(ctx, tvdbID, seasonNumber, userQualityProfileID, result); err != nil {
		return nil, err
	}

	if err := c.checkExistingSeasonRequest(ctx, tvdbID, seasonNumber, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *LibraryChecker) checkSeasonInLibrary(ctx context.Context, tvdbID, seasonNumber int64, userQualityProfileID *int64, result *AvailabilityResult) error {
	series, err := c.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if series.ID == 0 {
		return nil
	}

	season, err := c.queries.GetSeasonByNumber(ctx, sqlc.GetSeasonByNumberParams{
		SeriesID:     series.ID,
		SeasonNumber: seasonNumber,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if season.ID > 0 {
		result.InLibrary = true
		result.MediaID = &season.ID
		result.CanRequest = c.canRequestWithSlots(result.ExistingSlots, userQualityProfileID)
	}

	return nil
}

func (c *LibraryChecker) checkExistingSeasonRequest(ctx context.Context, tvdbID, seasonNumber int64, result *AvailabilityResult) error {
	existingReq, err := c.queries.GetActiveRequestByTvdbIDAndSeason(ctx, sqlc.GetActiveRequestByTvdbIDAndSeasonParams{
		TvdbID:       sql.NullInt64{Int64: tvdbID, Valid: true},
		SeasonNumber: sql.NullInt64{Int64: seasonNumber, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if existingReq != nil {
		result.ExistingRequestID = &existingReq.ID
		result.ExistingRequestUserID = &existingReq.UserID
		result.ExistingRequestStatus = &existingReq.Status
		result.CanRequest = false
	}

	return nil
}

func (c *LibraryChecker) CheckEpisodeAvailability(ctx context.Context, tvdbID, seasonNumber, episodeNumber int64, userQualityProfileID *int64) (*AvailabilityResult, error) {
	result := &AvailabilityResult{
		InLibrary:     false,
		ExistingSlots: []SlotInfo{},
		CanRequest:    true,
	}

	if err := c.checkEpisodeInLibrary(ctx, tvdbID, seasonNumber, episodeNumber, userQualityProfileID, result); err != nil {
		return nil, err
	}

	if err := c.checkExistingEpisodeRequest(ctx, tvdbID, seasonNumber, episodeNumber, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *LibraryChecker) checkEpisodeInLibrary(ctx context.Context, tvdbID, seasonNumber, episodeNumber int64, userQualityProfileID *int64, result *AvailabilityResult) error {
	series, err := c.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if series.ID == 0 {
		return nil
	}

	episode, err := c.queries.GetEpisodeByNumber(ctx, sqlc.GetEpisodeByNumberParams{
		SeriesID:      series.ID,
		SeasonNumber:  seasonNumber,
		EpisodeNumber: episodeNumber,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if episode.ID == 0 {
		return nil
	}

	result.InLibrary = true
	result.MediaID = &episode.ID

	slots, err := c.getEpisodeSlots(ctx, episode.ID)
	if err != nil {
		c.logger.Warn().Err(err).Int64("episodeID", episode.ID).Msg("failed to get episode slots")
	} else {
		result.ExistingSlots = slots
		result.CanRequest = c.canRequestWithSlots(slots, userQualityProfileID)
	}

	return nil
}

func (c *LibraryChecker) checkExistingEpisodeRequest(ctx context.Context, tvdbID, seasonNumber, episodeNumber int64, result *AvailabilityResult) error {
	existingReq, err := c.queries.GetActiveRequestByTvdbIDAndEpisode(ctx, sqlc.GetActiveRequestByTvdbIDAndEpisodeParams{
		TvdbID:        sql.NullInt64{Int64: tvdbID, Valid: true},
		SeasonNumber:  sql.NullInt64{Int64: seasonNumber, Valid: true},
		EpisodeNumber: sql.NullInt64{Int64: episodeNumber, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if existingReq != nil {
		result.ExistingRequestID = &existingReq.ID
		result.ExistingRequestUserID = &existingReq.UserID
		result.ExistingRequestStatus = &existingReq.Status
		result.CanRequest = false
	}

	return nil
}

func (c *LibraryChecker) getMovieSlots(ctx context.Context, movieID int64) ([]SlotInfo, error) {
	assignments, err := c.queries.ListMovieSlotAssignments(ctx, movieID)
	if err != nil {
		return nil, err
	}

	slots := make([]SlotInfo, 0, len(assignments))
	for _, a := range assignments {
		info := SlotInfo{
			SlotID:   a.SlotID,
			SlotName: a.SlotName,
			HasFile:  a.FileID.Valid,
		}
		if a.QualityProfileID.Valid {
			info.QualityID = &a.QualityProfileID.Int64
		}
		slots = append(slots, info)
	}

	return slots, nil
}

func (c *LibraryChecker) getEpisodeSlots(ctx context.Context, episodeID int64) ([]SlotInfo, error) {
	assignments, err := c.queries.ListEpisodeSlotAssignments(ctx, episodeID)
	if err != nil {
		return nil, err
	}

	slots := make([]SlotInfo, 0, len(assignments))
	for _, a := range assignments {
		info := SlotInfo{
			SlotID:   a.SlotID,
			SlotName: a.SlotName,
			HasFile:  a.FileID.Valid,
		}
		if a.QualityProfileID.Valid {
			info.QualityID = &a.QualityProfileID.Int64
		}
		slots = append(slots, info)
	}

	return slots, nil
}

func (c *LibraryChecker) canRequestWithSlots(slots []SlotInfo, userQualityProfileID *int64) bool {
	if len(slots) == 0 {
		return true
	}

	if userQualityProfileID == nil {
		for _, slot := range slots {
			if slot.HasFile {
				return false
			}
		}
		return true
	}

	for _, slot := range slots {
		if slot.QualityID != nil && *slot.QualityID == *userQualityProfileID {
			if slot.HasFile {
				return false
			}
		}
	}

	return true
}
