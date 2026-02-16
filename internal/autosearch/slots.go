package autosearch

import (
	"context"
	"fmt"
	"sync"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/indexer/grab"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/quality"
	"github.com/slipstream/slipstream/internal/library/scanner"
	"github.com/slipstream/slipstream/internal/library/slots"
)

// SlotSearchResult extends SearchResult with slot information.
type SlotSearchResult struct {
	SearchResult
	SlotID        int64  `json:"slotId"`
	SlotNumber    int    `json:"slotNumber"`
	SlotName      string `json:"slotName"`
	IsSlotUpgrade bool   `json:"isSlotUpgrade"` // Req 11.1.2: whether grab is upgrade vs new fill
}

// MultiSlotSearchResult contains results from searching for multiple slots.
// Req 7.1.1: Auto-search with multiple empty monitored slots: search in parallel
type MultiSlotSearchResult struct {
	TotalSlots    int                `json:"totalSlots"`
	SearchedSlots int                `json:"searchedSlots"`
	Found         int                `json:"found"`
	Downloaded    int                `json:"downloaded"`
	Failed        int                `json:"failed"`
	Results       []SlotSearchResult `json:"results"`
}

// SetSlotsService sets the slots service for multi-version support.
func (s *Service) SetSlotsService(slotsService *slots.Service) {
	s.slotsService = slotsService
}

// SearchMovieSlot searches for a specific slot for a movie.
// This allows per-slot auto search from the UI.
func (s *Service) SearchMovieSlot(ctx context.Context, movieID, slotID int64, source SearchSource) (*SlotSearchResult, error) {
	if s.slotsService == nil {
		return nil, fmt.Errorf("slots service not configured")
	}

	// Get slot info for the quality profile
	slotInfo, err := s.slotsService.GetMovieSlotSearchInfo(ctx, movieID, slotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot info: %w", err)
	}

	// Get movie details
	movie, err := s.queries.GetMovie(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	// Build searchable item with slot's quality profile
	item := s.movieToSearchableItem(ctx, movie)
	item.QualityProfileID = slotInfo.QualityProfileID
	item.TargetSlotID = &slotID

	// Use existing search infrastructure
	result, err := s.searchAndGrab(ctx, &item, source)
	if err != nil {
		return nil, err
	}

	return &SlotSearchResult{
		SearchResult:  *result,
		SlotID:        slotID,
		SlotNumber:    slotInfo.SlotNumber,
		SlotName:      slotInfo.SlotName,
		IsSlotUpgrade: slotInfo.NeedsUpgrade(),
	}, nil
}

// SearchEpisodeSlot searches for a specific slot for an episode.
// This allows per-slot auto search from the UI.
func (s *Service) SearchEpisodeSlot(ctx context.Context, episodeID, slotID int64, source SearchSource) (*SlotSearchResult, error) {
	if s.slotsService == nil {
		return nil, fmt.Errorf("slots service not configured")
	}

	// Get slot info for the quality profile
	slotInfo, err := s.slotsService.GetEpisodeSlotSearchInfo(ctx, episodeID, slotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slot info: %w", err)
	}

	// Get episode and series details
	episode, err := s.queries.GetEpisode(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	series, err := s.queries.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	// Build searchable item with slot's quality profile
	item := s.episodeToSearchableItem(ctx, episode, series)
	item.QualityProfileID = slotInfo.QualityProfileID
	item.TargetSlotID = &slotID

	// Use existing search infrastructure
	result, err := s.searchAndGrab(ctx, &item, source)
	if err != nil {
		return nil, err
	}

	return &SlotSearchResult{
		SearchResult:  *result,
		SlotID:        slotID,
		SlotNumber:    slotInfo.SlotNumber,
		SlotName:      slotInfo.SlotName,
		IsSlotUpgrade: slotInfo.NeedsUpgrade(),
	}, nil
}

// SearchMovieSlots searches for all monitored slots that need content for a movie.
// Req 7.1.1: Search in parallel for multiple empty monitored slots
// Req 7.1.2: May grab multiple releases simultaneously for different slots
// Req 7.1.3: Each slot's search is independent
// Req 8.1.3: Auto-search only runs for monitored slots
func (s *Service) SearchMovieSlots(ctx context.Context, movieID int64, source SearchSource) (*MultiSlotSearchResult, error) {
	if s.slotsService == nil || !s.slotsService.IsMultiVersionEnabled(ctx) {
		return s.fallbackToSingleMovieSearch(ctx, movieID, source)
	}

	slotsNeeding, err := s.slotsService.GetMovieSlotsNeedingSearch(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slots needing search: %w", err)
	}

	if len(slotsNeeding) == 0 {
		return &MultiSlotSearchResult{
			TotalSlots:    0,
			SearchedSlots: 0,
		}, nil
	}

	movie, err := s.queries.GetMovie(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("failed to get movie: %w", err)
	}

	return s.searchAllMovieSlots(ctx, movie, slotsNeeding, source), nil
}

func (s *Service) fallbackToSingleMovieSearch(ctx context.Context, movieID int64, source SearchSource) (*MultiSlotSearchResult, error) {
	result, err := s.SearchMovie(ctx, movieID, source)
	if err != nil {
		return nil, err
	}
	return &MultiSlotSearchResult{
		TotalSlots:    1,
		SearchedSlots: 1,
		Found:         boolToInt(result.Found),
		Downloaded:    boolToInt(result.Downloaded),
		Failed:        boolToInt(!result.Found && result.Error != ""),
		Results:       []SlotSearchResult{{SearchResult: *result}},
	}, nil
}

func (s *Service) searchAllMovieSlots(ctx context.Context, movie *sqlc.Movie, slotsNeeding []slots.SlotSearchInfo, source SearchSource) *MultiSlotSearchResult {
	var wg sync.WaitGroup
	resultChan := make(chan SlotSearchResult, len(slotsNeeding))

	for _, slotInfo := range slotsNeeding {
		wg.Add(1)
		go func(slot slots.SlotSearchInfo) {
			defer wg.Done()
			slotResult := s.searchForSlot(ctx, movie, &slot, source)
			resultChan <- slotResult
		}(slotInfo)
	}

	wg.Wait()
	close(resultChan)

	return s.collectSlotResults(resultChan, len(slotsNeeding))
}

func (s *Service) collectSlotResults(resultChan chan SlotSearchResult, totalSlots int) *MultiSlotSearchResult {
	result := &MultiSlotSearchResult{
		TotalSlots:    totalSlots,
		SearchedSlots: totalSlots,
		Results:       make([]SlotSearchResult, 0, totalSlots),
	}

	for slotResult := range resultChan {
		result.Results = append(result.Results, slotResult)
		if slotResult.Found {
			result.Found++
		}
		if slotResult.Downloaded {
			result.Downloaded++
		}
		if slotResult.Error != "" && !slotResult.Found {
			result.Failed++
		}
	}

	return result
}

// searchForSlot performs a search for a specific slot using its quality profile.
func (s *Service) searchForSlot(ctx context.Context, movie *sqlc.Movie, slot *slots.SlotSearchInfo, source SearchSource) SlotSearchResult {
	result := SlotSearchResult{
		SlotID:        slot.SlotID,
		SlotNumber:    slot.SlotNumber,
		SlotName:      slot.SlotName,
		IsSlotUpgrade: slot.NeedsUpgrade(),
	}

	profile, err := s.qualityService.Get(ctx, slot.QualityProfileID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("profileId", slot.QualityProfileID).
			Msg("Failed to get quality profile for slot")
		result.Error = "failed to get quality profile"
		return result
	}

	bestRelease := s.findBestReleaseForMovieSlot(ctx, movie, profile, slot, &result)
	if bestRelease == nil {
		return result
	}

	s.grabReleaseForMovieSlot(ctx, movie, slot, bestRelease, source, &result)
	return result
}

func (s *Service) findBestReleaseForMovieSlot(ctx context.Context, movie *sqlc.Movie, profile *quality.Profile, slot *slots.SlotSearchInfo, result *SlotSearchResult) *types.TorrentInfo {
	criteria := s.buildMovieSearchCriteria(movie)
	searchYear := 0
	if movie.Year.Valid {
		searchYear = int(movie.Year.Int64)
	}

	scoringParams := search.ScoredSearchParams{
		QualityProfile: profile,
		SearchYear:     searchYear,
	}

	searchResult, err := s.searchService.SearchTorrents(ctx, &criteria, &scoringParams)
	if err != nil {
		result.Error = fmt.Sprintf("search failed: %v", err)
		return nil
	}

	if len(searchResult.Releases) == 0 {
		s.logger.Debug().Str("title", movie.Title).Int64("slotId", slot.SlotID).
			Msg("No releases found for slot")
		return nil
	}

	bestRelease := s.selectBestReleaseForSlot(searchResult.Releases, profile, slot)
	if bestRelease == nil {
		s.logger.Debug().Str("title", movie.Title).Int64("slotId", slot.SlotID).
			Msg("No acceptable releases found for slot")
		return nil
	}

	result.Found = true
	result.Release = bestRelease

	s.logger.Info().Str("title", movie.Title).Int64("slotId", slot.SlotID).
		Str("slotName", slot.SlotName).Str("release", bestRelease.Title).
		Float64("score", bestRelease.Score).Msg("Selected best release for slot")

	return bestRelease
}

func (s *Service) buildMovieSearchCriteria(movie *sqlc.Movie) types.SearchCriteria {
	criteria := types.SearchCriteria{
		Query: movie.Title,
		Type:  "movie",
	}
	if movie.ImdbID.Valid {
		criteria.ImdbID = movie.ImdbID.String
	}
	if movie.TmdbID.Valid {
		criteria.TmdbID = int(movie.TmdbID.Int64)
	}
	if movie.Year.Valid {
		criteria.Year = int(movie.Year.Int64)
	}
	return criteria
}

func (s *Service) grabReleaseForMovieSlot(ctx context.Context, movie *sqlc.Movie, slot *slots.SlotSearchInfo, bestRelease *types.TorrentInfo, source SearchSource, result *SlotSearchResult) {
	grabReq := &grab.GrabRequest{
		Release:      &bestRelease.ReleaseInfo,
		MediaType:    string(MediaTypeMovie),
		MediaID:      movie.ID,
		TargetSlotID: &slot.SlotID,
		Source:       "auto-search",
	}

	grabResult, err := s.grabService.Grab(ctx, grabReq)
	if err != nil {
		result.Error = fmt.Sprintf("grab failed: %v", err)
		s.logAutoSearchFailed(ctx, &SearchableItem{
			MediaType: MediaTypeMovie,
			MediaID:   movie.ID,
			Title:     movie.Title,
		}, source, err.Error())
		return
	}

	result.Downloaded = grabResult.Success
	result.Upgraded = slot.NeedsUpgrade()
	result.ClientName = grabResult.ClientName
	result.DownloadID = grabResult.DownloadID

	if !grabResult.Success {
		result.Error = grabResult.Error
	} else {
		s.logSlotSearchSuccess(MediaTypeMovie, movie.ID, slot, bestRelease)
	}
}

// selectBestReleaseForSlot selects the best release for a specific slot.
// Req 7.2.1: When slot has file but finds better match (upgrade), replace
// Req 7.2.2: Standard upgrade behavior per slot
func (s *Service) selectBestReleaseForSlot(releases []types.TorrentInfo, profile *quality.Profile, slot *slots.SlotSearchInfo) *types.TorrentInfo {
	for i := range releases {
		release := &releases[i]

		// Skip if quality is not acceptable
		if release.ScoreBreakdown == nil || release.ScoreBreakdown.QualityID <= 0 {
			continue
		}

		if !profile.IsAcceptable(release.ScoreBreakdown.QualityID) {
			continue
		}

		// For upgrades, check if this is actually better than current
		if slot.HasFile() && slot.CurrentQualityID != nil {
			currentQuality := int(*slot.CurrentQualityID)
			if !profile.IsUpgrade(currentQuality, release.ScoreBreakdown.QualityID) {
				continue
			}
		}

		return release
	}

	return nil
}

// logSlotSearchSuccess logs a successful slot-specific auto-search.
func (s *Service) logSlotSearchSuccess(mediaType MediaType, mediaID int64, slot *slots.SlotSearchInfo, release *types.TorrentInfo) {
	if s.historyService == nil {
		return
	}

	// Log the success - the history service handles the details
	s.logger.Info().
		Str("mediaType", string(mediaType)).
		Int64("mediaId", mediaID).
		Int64("slotId", slot.SlotID).
		Str("slotName", slot.SlotName).
		Str("release", release.Title).
		Bool("isUpgrade", slot.NeedsUpgrade()).
		Msg("Slot search completed successfully")
}

// SearchEpisodeSlots searches for all monitored slots that need content for an episode.
func (s *Service) SearchEpisodeSlots(ctx context.Context, episodeID int64, source SearchSource) (*MultiSlotSearchResult, error) {
	if s.slotsService == nil || !s.slotsService.IsMultiVersionEnabled(ctx) {
		return s.fallbackToSingleEpisodeSearch(ctx, episodeID, source)
	}

	slotsNeeding, err := s.slotsService.GetEpisodeSlotsNeedingSearch(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slots needing search: %w", err)
	}

	if len(slotsNeeding) == 0 {
		return &MultiSlotSearchResult{
			TotalSlots:    0,
			SearchedSlots: 0,
		}, nil
	}

	episode, err := s.queries.GetEpisode(ctx, episodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get episode: %w", err)
	}

	series, err := s.queries.GetSeries(ctx, episode.SeriesID)
	if err != nil {
		return nil, fmt.Errorf("failed to get series: %w", err)
	}

	return s.searchAllEpisodeSlots(ctx, episode, series, slotsNeeding, source), nil
}

func (s *Service) fallbackToSingleEpisodeSearch(ctx context.Context, episodeID int64, source SearchSource) (*MultiSlotSearchResult, error) {
	result, err := s.SearchEpisode(ctx, episodeID, source)
	if err != nil {
		return nil, err
	}
	return &MultiSlotSearchResult{
		TotalSlots:    1,
		SearchedSlots: 1,
		Found:         boolToInt(result.Found),
		Downloaded:    boolToInt(result.Downloaded),
		Failed:        boolToInt(!result.Found && result.Error != ""),
		Results:       []SlotSearchResult{{SearchResult: *result}},
	}, nil
}

func (s *Service) searchAllEpisodeSlots(ctx context.Context, episode *sqlc.Episode, series *sqlc.Series, slotsNeeding []slots.SlotSearchInfo, source SearchSource) *MultiSlotSearchResult {
	var wg sync.WaitGroup
	resultChan := make(chan SlotSearchResult, len(slotsNeeding))

	for _, slotInfo := range slotsNeeding {
		wg.Add(1)
		go func(slot slots.SlotSearchInfo) {
			defer wg.Done()
			slotResult := s.searchEpisodeForSlot(ctx, episode, series, &slot, source)
			resultChan <- slotResult
		}(slotInfo)
	}

	wg.Wait()
	close(resultChan)

	return s.collectSlotResults(resultChan, len(slotsNeeding))
}

// searchEpisodeForSlot performs a search for a specific slot using its quality profile for an episode.
func (s *Service) searchEpisodeForSlot(ctx context.Context, episode *sqlc.Episode, series *sqlc.Series, slot *slots.SlotSearchInfo, _ SearchSource) SlotSearchResult {
	result := SlotSearchResult{
		SlotID:        slot.SlotID,
		SlotNumber:    slot.SlotNumber,
		SlotName:      slot.SlotName,
		IsSlotUpgrade: slot.NeedsUpgrade(),
	}

	profile, err := s.qualityService.Get(ctx, slot.QualityProfileID)
	if err != nil {
		s.logger.Warn().Err(err).Int64("profileId", slot.QualityProfileID).Msg("Failed to get quality profile for slot")
		result.Error = "failed to get quality profile"
		return result
	}

	bestRelease := s.findBestEpisodeRelease(ctx, episode, series, slot, profile)
	if bestRelease == nil {
		return result
	}

	result.Found = true
	result.Release = bestRelease
	s.grabEpisodeForSlot(ctx, &result, bestRelease, episode, series, slot)
	return result
}

func (s *Service) findBestEpisodeRelease(ctx context.Context, episode *sqlc.Episode, series *sqlc.Series, slot *slots.SlotSearchInfo, profile *quality.Profile) *types.TorrentInfo {
	criteria := types.SearchCriteria{
		Query:   series.Title,
		Type:    "tvsearch",
		Season:  int(episode.SeasonNumber),
		Episode: int(episode.EpisodeNumber),
	}
	if series.TvdbID.Valid {
		criteria.TvdbID = int(series.TvdbID.Int64)
	}

	scoringParams := search.ScoredSearchParams{
		QualityProfile: profile,
		SearchSeason:   int(episode.SeasonNumber),
		SearchEpisode:  int(episode.EpisodeNumber),
	}

	searchResult, err := s.searchService.SearchTorrents(ctx, &criteria, &scoringParams)
	if err != nil || len(searchResult.Releases) == 0 {
		return nil
	}

	return s.selectBestEpisodeReleaseForSlot(searchResult.Releases, profile, slot, int(episode.SeasonNumber), int(episode.EpisodeNumber))
}

func (s *Service) grabEpisodeForSlot(ctx context.Context, result *SlotSearchResult, bestRelease *types.TorrentInfo, episode *sqlc.Episode, series *sqlc.Series, slot *slots.SlotSearchInfo) {
	grabReq := &grab.GrabRequest{
		Release:      &bestRelease.ReleaseInfo,
		MediaType:    string(MediaTypeEpisode),
		MediaID:      episode.ID,
		SeriesID:     series.ID,
		SeasonNumber: int(episode.SeasonNumber),
		TargetSlotID: &slot.SlotID,
		Source:       "auto-search",
	}

	grabResult, err := s.grabService.Grab(ctx, grabReq)
	if err != nil {
		result.Error = fmt.Sprintf("grab failed: %v", err)
		return
	}

	result.Downloaded = grabResult.Success
	result.Upgraded = slot.NeedsUpgrade()
	result.ClientName = grabResult.ClientName
	result.DownloadID = grabResult.DownloadID

	if !grabResult.Success {
		result.Error = grabResult.Error
	}
}

// selectBestEpisodeReleaseForSlot selects the best release for a specific slot for an episode.
func (s *Service) selectBestEpisodeReleaseForSlot(releases []types.TorrentInfo, profile *quality.Profile, slot *slots.SlotSearchInfo, seasonNumber, episodeNumber int) *types.TorrentInfo {
	for i := range releases {
		release := &releases[i]

		// Parse to verify episode match
		parsed := scanner.ParseFilename(release.Title)
		if parsed.Season != seasonNumber || parsed.Episode != episodeNumber {
			continue
		}

		// Skip if quality is not acceptable
		if release.ScoreBreakdown == nil || release.ScoreBreakdown.QualityID <= 0 {
			continue
		}

		if !profile.IsAcceptable(release.ScoreBreakdown.QualityID) {
			continue
		}

		// For upgrades, check if this is actually better than current
		if slot.HasFile() && slot.CurrentQualityID != nil {
			currentQuality := int(*slot.CurrentQualityID)
			if !profile.IsUpgrade(currentQuality, release.ScoreBreakdown.QualityID) {
				continue
			}
		}

		return release
	}

	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
