package search

import (
	"context"
	"sort"
	"time"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/indexer/scoring"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// ModeProvider provides information about the current indexer mode.
type ModeProvider interface {
	IsProwlarrMode(ctx context.Context) (bool, error)
}

// ProwlarrSearcher provides search functionality through Prowlarr.
type ProwlarrSearcher interface {
	Search(ctx context.Context, criteria types.SearchCriteria) ([]types.TorrentInfo, error)
}

// Router routes searches to the appropriate backend based on the current indexer mode.
// It implements SearchService to provide a drop-in replacement for the standard service.
type Router struct {
	slipstreamService *Service
	prowlarrSearcher  ProwlarrSearcher
	modeProvider      ModeProvider
	logger            zerolog.Logger
}

// NewRouter creates a new search router.
func NewRouter(slipstreamService *Service, logger zerolog.Logger) *Router {
	return &Router{
		slipstreamService: slipstreamService,
		logger:            logger.With().Str("component", "search-router").Logger(),
	}
}

// SetProwlarrSearcher sets the Prowlarr search provider.
func (r *Router) SetProwlarrSearcher(searcher ProwlarrSearcher) {
	r.prowlarrSearcher = searcher
}

// SetModeProvider sets the mode provider.
func (r *Router) SetModeProvider(provider ModeProvider) {
	r.modeProvider = provider
}

// Search routes the search to the appropriate backend.
func (r *Router) Search(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error) {
	isProwlarr, err := r.isProwlarrMode(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to check indexer mode, defaulting to SlipStream")
		isProwlarr = false
	}

	if isProwlarr && r.prowlarrSearcher != nil {
		return r.searchViaProwlarr(ctx, criteria)
	}

	return r.slipstreamService.Search(ctx, criteria)
}

// SearchTorrents routes the torrent search to the appropriate backend with scoring.
func (r *Router) SearchTorrents(ctx context.Context, criteria types.SearchCriteria, params ScoredSearchParams) (*TorrentSearchResult, error) {
	isProwlarr, err := r.isProwlarrMode(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to check indexer mode, defaulting to SlipStream")
		isProwlarr = false
	}

	r.logger.Info().
		Bool("isProwlarrMode", isProwlarr).
		Bool("hasProwlarrSearcher", r.prowlarrSearcher != nil).
		Str("query", criteria.Query).
		Str("type", criteria.Type).
		Msg("SearchTorrents routing decision")

	if isProwlarr && r.prowlarrSearcher != nil {
		return r.searchTorrentsViaProwlarr(ctx, criteria, params)
	}

	return r.slipstreamService.SearchTorrents(ctx, criteria, params)
}

// SearchMovies routes movie searches to the appropriate backend.
func (r *Router) SearchMovies(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error) {
	isProwlarr, err := r.isProwlarrMode(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to check indexer mode, defaulting to SlipStream")
		isProwlarr = false
	}

	if isProwlarr && r.prowlarrSearcher != nil {
		criteria.Type = "movie"
		return r.searchViaProwlarr(ctx, criteria)
	}

	return r.slipstreamService.SearchMovies(ctx, criteria)
}

// SearchTV routes TV searches to the appropriate backend.
func (r *Router) SearchTV(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error) {
	isProwlarr, err := r.isProwlarrMode(ctx)
	if err != nil {
		r.logger.Warn().Err(err).Msg("Failed to check indexer mode, defaulting to SlipStream")
		isProwlarr = false
	}

	if isProwlarr && r.prowlarrSearcher != nil {
		criteria.Type = "tvsearch"
		return r.searchViaProwlarr(ctx, criteria)
	}

	return r.slipstreamService.SearchTV(ctx, criteria)
}

// isProwlarrMode checks if Prowlarr mode is active.
func (r *Router) isProwlarrMode(ctx context.Context) (bool, error) {
	if r.modeProvider == nil {
		return false, nil
	}
	return r.modeProvider.IsProwlarrMode(ctx)
}

// searchViaProwlarr executes a search through Prowlarr and converts to SearchResult.
func (r *Router) searchViaProwlarr(ctx context.Context, criteria types.SearchCriteria) (*SearchResult, error) {
	r.logger.Debug().
		Str("query", criteria.Query).
		Str("type", criteria.Type).
		Msg("Routing search through Prowlarr")

	torrents, err := r.prowlarrSearcher.Search(ctx, criteria)
	if err != nil {
		return &SearchResult{
			Releases:     []types.ReleaseInfo{},
			TotalResults: 0,
			IndexersUsed: 0,
			IndexerErrors: []SearchIndexerError{{
				IndexerID:   0,
				IndexerName: "Prowlarr",
				Error:       err.Error(),
			}},
		}, nil
	}

	// Convert TorrentInfo to ReleaseInfo
	releases := make([]types.ReleaseInfo, len(torrents))
	for i, t := range torrents {
		releases[i] = t.ReleaseInfo
	}

	return &SearchResult{
		Releases:     releases,
		TotalResults: len(releases),
		IndexersUsed: 1,
	}, nil
}

// searchTorrentsViaProwlarr executes a torrent search through Prowlarr with scoring.
func (r *Router) searchTorrentsViaProwlarr(ctx context.Context, criteria types.SearchCriteria, params ScoredSearchParams) (*TorrentSearchResult, error) {
	r.logger.Info().
		Str("query", criteria.Query).
		Str("type", criteria.Type).
		Int("season", criteria.Season).
		Int("episode", criteria.Episode).
		Msg("Routing torrent search through Prowlarr")

	torrents, err := r.prowlarrSearcher.Search(ctx, criteria)
	if err != nil {
		r.logger.Error().Err(err).Msg("Prowlarr search failed")
		return &TorrentSearchResult{
			Releases:     []types.TorrentInfo{},
			TotalResults: 0,
			IndexersUsed: 0,
			IndexerErrors: []SearchIndexerError{{
				IndexerID:   0,
				IndexerName: "Prowlarr",
				Error:       err.Error(),
			}},
		}, nil
	}

	r.logger.Info().Int("rawResults", len(torrents)).Msg("Prowlarr returned results")

	if len(torrents) == 0 {
		return &TorrentSearchResult{
			Releases:     []types.TorrentInfo{},
			TotalResults: 0,
			IndexersUsed: 1,
		}, nil
	}

	// Enrich with parsed quality info
	r.enrichTorrentsWithQuality(torrents)

	// Filter by search criteria
	preFilterCount := len(torrents)
	torrents = r.filterByCriteria(torrents, criteria)

	r.logger.Info().
		Int("preFilter", preFilterCount).
		Int("postFilter", len(torrents)).
		Msg("Filtered Prowlarr results by criteria")

	// Score torrents using SlipStream's scoring algorithm
	r.scoreTorrents(torrents, params)

	r.logger.Info().
		Int("results", len(torrents)).
		Msg("Prowlarr torrent search completed")

	return &TorrentSearchResult{
		Releases:     torrents,
		TotalResults: len(torrents),
		IndexersUsed: 1,
	}, nil
}

// enrichTorrentsWithQuality parses quality info from torrent titles.
func (r *Router) enrichTorrentsWithQuality(torrents []types.TorrentInfo) {
	for i := range torrents {
		parsed := scanner.ParseFilename(torrents[i].Title)
		torrents[i].Quality = parsed.Quality
		torrents[i].Source = parsed.Source
		torrents[i].Resolution = r.qualityToResolution(parsed.Quality)
		torrents[i].Languages = parsed.Languages
	}
}

// qualityToResolution converts a quality string to a resolution integer.
func (r *Router) qualityToResolution(quality string) int {
	switch quality {
	case "2160p":
		return 2160
	case "1080p":
		return 1080
	case "720p":
		return 720
	case "480p":
		return 480
	default:
		return 0
	}
}

// filterByCriteria filters torrent results based on search criteria.
func (r *Router) filterByCriteria(torrents []types.TorrentInfo, criteria types.SearchCriteria) []types.TorrentInfo {
	if criteria.Query == "" {
		return torrents
	}

	return FilterByCriteria(torrents, criteria)
}

// scoreTorrents applies SlipStream's scoring algorithm to Prowlarr results.
func (r *Router) scoreTorrents(torrents []types.TorrentInfo, params ScoredSearchParams) {
	if params.QualityProfile == nil {
		// Sort by seeders if no quality profile
		sort.Slice(torrents, func(i, j int) bool {
			return torrents[i].Seeders > torrents[j].Seeders
		})
		return
	}

	// Build scoring context (Prowlarr results don't have indexer priorities)
	scoringCtx := scoring.ScoringContext{
		QualityProfile:    params.QualityProfile,
		SearchYear:        params.SearchYear,
		SearchSeason:      params.SearchSeason,
		SearchEpisode:     params.SearchEpisode,
		IndexerPriorities: make(map[int64]int),
		Now:               time.Now(),
	}

	// Score and sort torrents
	scorer := scoring.NewDefaultScorer()
	scorer.ScoreTorrents(torrents, scoringCtx)
}

// GetSlipStreamService returns the underlying SlipStream search service.
// This is useful for components that need direct access to the service.
func (r *Router) GetSlipStreamService() *Service {
	return r.slipstreamService
}
