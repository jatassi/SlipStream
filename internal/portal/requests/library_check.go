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

type SeasonAvailability struct {
	SeasonNumber           int  `json:"seasonNumber"`
	Available              bool `json:"available"`
	HasAnyFiles            bool `json:"hasAnyFiles"`
	AiredEpisodesWithFiles int  `json:"airedEpisodesWithFiles"`
	TotalAiredEpisodes     int  `json:"totalAiredEpisodes"`
	TotalEpisodes          int  `json:"totalEpisodes"`
	Monitored              bool `json:"monitored"`
}

type CoveredSeason struct {
	RequestID  int64
	UserID     int64
	Status     string
	IsWatching bool
}

type AvailabilityResult struct {
	InLibrary                 bool                 `json:"inLibrary"`
	ExistingSlots             []SlotInfo           `json:"existingSlots,omitempty"`
	CanRequest                bool                 `json:"canRequest"`
	ExistingRequestID         *int64               `json:"existingRequestId,omitempty"`
	ExistingRequestUserID     *int64               `json:"existingRequestUserId,omitempty"`
	ExistingRequestStatus     *string              `json:"existingRequestStatus,omitempty"`
	ExistingRequestIsWatching *bool                `json:"existingRequestIsWatching,omitempty"`
	MediaID                   *int64               `json:"mediaId,omitempty"`
	AddedAt                   *string              `json:"addedAt,omitempty"`
	SeasonAvailability        []SeasonAvailability `json:"seasonAvailability,omitempty"`
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

func (c *LibraryChecker) CheckMovieAvailability(ctx context.Context, tmdbID int64, userQualityProfileID *int64, currentUserID ...int64) (*AvailabilityResult, error) {
	result := &AvailabilityResult{
		InLibrary:     false,
		ExistingSlots: []SlotInfo{},
		CanRequest:    true,
	}

	if err := c.checkMovieInLibrary(ctx, tmdbID, userQualityProfileID, result); err != nil {
		return nil, err
	}

	var uid int64
	if len(currentUserID) > 0 {
		uid = currentUserID[0]
	}
	if err := c.checkExistingMovieRequest(ctx, tmdbID, uid, result); err != nil {
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
	}

	hasFiles := c.movieHasFiles(ctx, movie.ID, result.ExistingSlots)
	switch {
	case !hasFiles:
		result.InLibrary = false
		result.CanRequest = true
	case len(slots) > 0:
		result.CanRequest = c.canRequestWithSlots(slots, userQualityProfileID)
	default:
		result.CanRequest = false
	}

	return nil
}

func (c *LibraryChecker) checkExistingMovieRequest(ctx context.Context, tmdbID, currentUserID int64, result *AvailabilityResult) error {
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

		if currentUserID > 0 {
			watchingVal, err := c.queries.IsWatchingRequest(ctx, sqlc.IsWatchingRequestParams{
				RequestID: existingReq.ID,
				UserID:    currentUserID,
			})
			if err == nil {
				isWatching := watchingVal != 0
				result.ExistingRequestIsWatching = &isWatching
			}
		}
	}

	return nil
}

func (c *LibraryChecker) CheckSeriesAvailability(ctx context.Context, tvdbID, tmdbID int64, userQualityProfileID *int64) (*AvailabilityResult, error) {
	result := &AvailabilityResult{
		InLibrary:     false,
		ExistingSlots: []SlotInfo{},
		CanRequest:    true,
	}

	series := c.findSeries(ctx, tvdbID, tmdbID)
	if series != nil && series.ID > 0 {
		c.populateSeriesAvailability(ctx, series, result)
		// Use the DB series' TVDB ID for request lookup (may differ from search result)
		if series.TvdbID.Valid && series.TvdbID.Int64 > 0 {
			tvdbID = series.TvdbID.Int64
		}
	}

	if tvdbID > 0 {
		if err := c.checkExistingSeriesRequest(ctx, tvdbID, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// findSeries looks up a series by TVDB ID first, falling back to TMDB ID.
func (c *LibraryChecker) findSeries(ctx context.Context, tvdbID, tmdbID int64) *sqlc.Series {
	if tvdbID > 0 {
		series, err := c.queries.GetSeriesByTvdbID(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
		if err == nil && series.ID > 0 {
			return series
		}
	}
	if tmdbID > 0 {
		series, err := c.queries.GetSeriesByTmdbID(ctx, sql.NullInt64{Int64: tmdbID, Valid: true})
		if err == nil && series.ID > 0 {
			return series
		}
	}
	return nil
}

func (c *LibraryChecker) populateSeriesAvailability(ctx context.Context, series *sqlc.Series, result *AvailabilityResult) {
	result.MediaID = &series.ID
	if series.AddedAt.Valid {
		addedAtStr := series.AddedAt.Time.Format("2006-01-02T15:04:05Z")
		result.AddedAt = &addedAtStr
	}

	seasonAvail, err := c.getSeasonAvailability(ctx, series.ID)
	if err != nil {
		c.logger.Warn().Err(err).Int64("seriesID", series.ID).Msg("failed to get season availability")
	}
	result.SeasonAvailability = seasonAvail

	hasAnyFiles := false
	allAvailable := true
	for _, sa := range seasonAvail {
		if sa.HasAnyFiles {
			hasAnyFiles = true
		}
		if !sa.Available {
			allAvailable = false
		}
	}

	if hasAnyFiles {
		result.InLibrary = true
		result.CanRequest = !allAvailable
	}
}

func (c *LibraryChecker) checkExistingSeriesRequest(ctx context.Context, tvdbID int64, result *AvailabilityResult) error {
	existingReq, err := c.queries.GetActiveRequestByTvdbID(ctx, sqlc.GetActiveRequestByTvdbIDParams{
		TvdbID:    sql.NullInt64{Int64: tvdbID, Valid: true},
		MediaType: MediaTypeSeries,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil && existingReq != nil {
		result.ExistingRequestID = &existingReq.ID
		result.ExistingRequestUserID = &existingReq.UserID
		result.ExistingRequestStatus = &existingReq.Status
		if !result.InLibrary || !result.CanRequest {
			result.CanRequest = false
		}
	}
	return nil
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

func (c *LibraryChecker) movieHasFiles(ctx context.Context, movieID int64, slots []SlotInfo) bool {
	for _, slot := range slots {
		if slot.HasFile {
			return true
		}
	}
	count, err := c.queries.CountMovieFiles(ctx, movieID)
	if err != nil {
		return false
	}
	return count > 0
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

func (c *LibraryChecker) getSeasonAvailability(ctx context.Context, seriesID int64) ([]SeasonAvailability, error) {
	rows, err := c.queries.GetSeriesSeasonAvailabilitySummary(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	result := make([]SeasonAvailability, 0, len(rows))
	for _, row := range rows {
		airedEps := toInt(row.AiredEpisodes)
		airedWithFiles := toInt(row.AiredWithFiles)
		unairedMonitored := toInt(row.UnairedMonitored)
		totalEps := int(row.TotalEpisodes)
		unairedCount := totalEps - airedEps

		available := airedEps > 0 &&
			airedWithFiles == airedEps &&
			(unairedCount == 0 || unairedMonitored == unairedCount)

		result = append(result, SeasonAvailability{
			SeasonNumber:           int(row.SeasonNumber),
			Available:              available,
			HasAnyFiles:            airedWithFiles > 0,
			AiredEpisodesWithFiles: airedWithFiles,
			TotalAiredEpisodes:     airedEps,
			TotalEpisodes:          totalEps,
			Monitored:              row.Monitored == 1,
		})
	}

	return result, nil
}

func (c *LibraryChecker) GetSeasonAvailabilityMap(ctx context.Context, tvdbID, tmdbID int64) (map[int]SeasonAvailability, error) {
	series := c.findSeries(ctx, tvdbID, tmdbID)
	if series == nil {
		return nil, sql.ErrNoRows
	}

	avail, err := c.getSeasonAvailability(ctx, series.ID)
	if err != nil {
		return nil, err
	}

	m := make(map[int]SeasonAvailability, len(avail))
	for _, sa := range avail {
		m[sa.SeasonNumber] = sa
	}
	return m, nil
}

func (c *LibraryChecker) GetCoveredSeasons(ctx context.Context, tvdbID, tmdbID, currentUserID int64) (map[int]CoveredSeason, error) {
	tvdbID = c.resolveTvdbID(ctx, tvdbID, tmdbID)
	if tvdbID == 0 {
		return map[int]CoveredSeason{}, nil
	}
	reqs, err := c.queries.FindRequestsCoveringSeasons(ctx, sql.NullInt64{Int64: tvdbID, Valid: true})
	if err != nil {
		return nil, err
	}

	result := make(map[int]CoveredSeason)
	for _, r := range reqs {
		covered := c.toCoveredSeason(ctx, r, currentUserID)
		if r.MediaType == MediaTypeSeries {
			for _, sn := range seasonsFromJSON(r.RequestedSeasons) {
				result[int(sn)] = covered
			}
		} else if r.MediaType == MediaTypeSeason && r.SeasonNumber.Valid {
			result[int(r.SeasonNumber.Int64)] = covered
		}
	}

	return result, nil
}

func (c *LibraryChecker) resolveTvdbID(ctx context.Context, tvdbID, tmdbID int64) int64 {
	if tvdbID > 0 {
		return tvdbID
	}
	if tmdbID > 0 {
		series := c.findSeries(ctx, 0, tmdbID)
		if series != nil && series.TvdbID.Valid {
			return series.TvdbID.Int64
		}
	}
	return 0
}

func (c *LibraryChecker) toCoveredSeason(ctx context.Context, r *sqlc.Request, currentUserID int64) CoveredSeason {
	isWatching := false
	watchingVal, err := c.queries.IsWatchingRequest(ctx, sqlc.IsWatchingRequestParams{
		RequestID: r.ID,
		UserID:    currentUserID,
	})
	if err == nil {
		isWatching = watchingVal != 0
	}
	return CoveredSeason{
		RequestID:  r.ID,
		UserID:     r.UserID,
		Status:     r.Status,
		IsWatching: isWatching,
	}
}

type EpisodeAvailability struct {
	EpisodeNumber int  `json:"episodeNumber"`
	HasFile       bool `json:"hasFile"`
	Monitored     bool `json:"monitored"`
	Aired         bool `json:"aired"`
}

func (c *LibraryChecker) GetEpisodeAvailabilityForSeason(ctx context.Context, tvdbID, tmdbID int64, seasonNumber int) ([]EpisodeAvailability, error) {
	series := c.findSeries(ctx, tvdbID, tmdbID)
	if series == nil {
		return nil, sql.ErrNoRows
	}

	rows, err := c.queries.GetEpisodeAvailabilityForSeason(ctx, sqlc.GetEpisodeAvailabilityForSeasonParams{
		SeriesID:     series.ID,
		SeasonNumber: int64(seasonNumber),
	})
	if err != nil {
		return nil, err
	}

	result := make([]EpisodeAvailability, 0, len(rows))
	for _, row := range rows {
		result = append(result, EpisodeAvailability{
			EpisodeNumber: int(row.EpisodeNumber),
			HasFile:       row.HasFile != 0,
			Monitored:     row.Monitored != 0,
			Aired:         row.Aired != 0,
		})
	}
	return result, nil
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int64:
		return int(val)
	case float64:
		return int(val)
	case int:
		return val
	default:
		return 0
	}
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
