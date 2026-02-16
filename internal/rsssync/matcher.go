package rsssync

import (
	"context"
	"strconv"
	"strings"

	"github.com/rs/zerolog"

	"github.com/slipstream/slipstream/internal/database/sqlc"
	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/indexer/search"
	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// WantedIndex provides fast lookup of wanted media items by various keys.
type WantedIndex struct {
	byTitle  map[string][]decisioning.SearchableItem
	byImdbID map[string][]decisioning.SearchableItem
	byTmdbID map[int][]decisioning.SearchableItem
	byTvdbID map[int][]decisioning.SearchableItem
}

// BuildWantedIndex creates a WantedIndex from a list of wanted items.
func BuildWantedIndex(items []decisioning.SearchableItem) *WantedIndex {
	idx := &WantedIndex{
		byTitle:  make(map[string][]decisioning.SearchableItem),
		byImdbID: make(map[string][]decisioning.SearchableItem),
		byTmdbID: make(map[int][]decisioning.SearchableItem),
		byTvdbID: make(map[int][]decisioning.SearchableItem),
	}

	for i := range items {
		item := &items[i]
		normalized := search.NormalizeTitle(item.Title)
		if normalized != "" {
			idx.byTitle[normalized] = append(idx.byTitle[normalized], *item)
		}
		if item.ImdbID != "" {
			idx.byImdbID[item.ImdbID] = append(idx.byImdbID[item.ImdbID], *item)
		}
		if item.TmdbID != 0 {
			idx.byTmdbID[item.TmdbID] = append(idx.byTmdbID[item.TmdbID], *item)
		}
		if item.TvdbID != 0 {
			idx.byTvdbID[item.TvdbID] = append(idx.byTvdbID[item.TvdbID], *item)
		}
	}

	return idx
}

// MatchResult pairs a release with the wanted item it matched.
type MatchResult struct {
	Release    types.TorrentInfo
	WantedItem decisioning.SearchableItem
	IsSeason   bool
}

// Matcher matches RSS releases against a WantedIndex.
type Matcher struct {
	index   *WantedIndex
	queries *sqlc.Queries
	logger  *zerolog.Logger
}

// NewMatcher creates a new Matcher.
func NewMatcher(index *WantedIndex, queries *sqlc.Queries, logger *zerolog.Logger) *Matcher {
	return &Matcher{
		index:   index,
		queries: queries,
		logger:  logger,
	}
}

func (m *Matcher) Match(ctx context.Context, release *types.TorrentInfo) []MatchResult {
	parsed := scanner.ParseFilename(release.Title)
	if parsed == nil || parsed.Title == "" {
		return nil
	}

	candidates := m.findCandidates(release, parsed)
	if len(candidates) == 0 {
		return nil
	}

	var results []MatchResult

	switch {
	case parsed.IsSeasonPack:
		results = m.matchSeasonPack(ctx, release, parsed, candidates)
	case parsed.IsTV:
		results = m.matchEpisode(release, parsed, candidates)
	default:
		results = m.matchMovie(release, parsed, candidates)
	}

	return results
}

func (m *Matcher) findCandidates(release *types.TorrentInfo, parsed *scanner.ParsedMedia) []decisioning.SearchableItem {
	extractExternalIDs(release)

	// ID-based lookup first
	if imdbID := m.extractImdbID(release); imdbID != "" {
		if items, ok := m.index.byImdbID[imdbID]; ok {
			return items
		}
	}
	if tmdbID := m.extractTmdbID(release); tmdbID != 0 {
		if items, ok := m.index.byTmdbID[tmdbID]; ok {
			return items
		}
	}
	if tvdbID := m.extractTvdbID(release); tvdbID != 0 {
		if items, ok := m.index.byTvdbID[tvdbID]; ok {
			return items
		}
	}

	// Title fallback
	normalizedTitle := search.NormalizeTitle(parsed.Title)
	if normalizedTitle == "" {
		return nil
	}

	if items, ok := m.index.byTitle[normalizedTitle]; ok {
		return items
	}

	return nil
}

func (m *Matcher) matchMovie(release *types.TorrentInfo, parsed *scanner.ParsedMedia, candidates []decisioning.SearchableItem) []MatchResult {
	matchedByID := release.ImdbID != 0 || release.TmdbID != 0 || release.TvdbID != 0
	var results []MatchResult
	for i := range candidates {
		item := &candidates[i]
		if item.MediaType != decisioning.MediaTypeMovie {
			continue
		}
		if !matchedByID && parsed.Year > 0 && item.Year > 0 && parsed.Year != item.Year {
			continue
		}
		results = append(results, MatchResult{
			Release:    *release,
			WantedItem: *item,
		})
	}
	return results
}

func (m *Matcher) matchEpisode(release *types.TorrentInfo, parsed *scanner.ParsedMedia, candidates []decisioning.SearchableItem) []MatchResult {
	var results []MatchResult
	for i := range candidates {
		item := &candidates[i]
		if item.MediaType != decisioning.MediaTypeEpisode {
			continue
		}
		if item.SeasonNumber != parsed.Season {
			continue
		}
		if item.EpisodeNumber != parsed.Episode {
			continue
		}
		results = append(results, MatchResult{
			Release:    *release,
			WantedItem: *item,
		})
	}
	return results
}

func (m *Matcher) matchSeasonPack(ctx context.Context, release *types.TorrentInfo, parsed *scanner.ParsedMedia, candidates []decisioning.SearchableItem) []MatchResult {
	var results []MatchResult

	for i := range candidates {
		item := &candidates[i]
		if item.MediaType == decisioning.MediaTypeSeason && item.SeasonNumber == parsed.Season {
			results = append(results, MatchResult{
				Release:    *release,
				WantedItem: *item,
				IsSeason:   true,
			})
		}
	}
	if len(results) > 0 {
		return results
	}

	type seriesSeason struct {
		seriesID int64
		season   int
	}
	checked := make(map[seriesSeason]bool)

	for i := range candidates {
		item := &candidates[i]
		if item.MediaType != decisioning.MediaTypeEpisode {
			continue
		}
		if item.SeasonNumber != parsed.Season {
			continue
		}

		key := seriesSeason{seriesID: item.SeriesID, season: parsed.Season}
		if checked[key] {
			continue
		}
		checked[key] = true

		seasonItem := m.checkSeasonPackEligibility(ctx, item, parsed.Season)
		if seasonItem == nil {
			continue
		}

		results = append(results, MatchResult{
			Release:    *release,
			WantedItem: *seasonItem,
			IsSeason:   true,
		})
	}
	return results
}

func (m *Matcher) checkSeasonPackEligibility(ctx context.Context, episodeItem *decisioning.SearchableItem, season int) *decisioning.SearchableItem {
	// Check all-missing eligibility
	allMissing := decisioning.IsSeasonPackEligible(ctx, m.queries, m.logger, episodeItem.SeriesID, season)
	if allMissing {
		return &decisioning.SearchableItem{
			MediaType:        decisioning.MediaTypeSeason,
			MediaID:          episodeItem.SeriesID,
			Title:            episodeItem.Title,
			TvdbID:           episodeItem.TvdbID,
			SeriesID:         episodeItem.SeriesID,
			SeasonNumber:     season,
			QualityProfileID: episodeItem.QualityProfileID,
			HasFile:          false,
		}
	}

	// Check all-upgradable eligibility
	allUpgradable := decisioning.IsSeasonPackUpgradeEligible(ctx, m.queries, m.logger, episodeItem.SeriesID, season)
	if allUpgradable {
		// Find the highest current quality across wanted episodes for this season
		highestQID := m.highestQualityForSeason(episodeItem.TvdbID, episodeItem.SeriesID, season)
		return &decisioning.SearchableItem{
			MediaType:        decisioning.MediaTypeSeason,
			MediaID:          episodeItem.SeriesID,
			Title:            episodeItem.Title,
			TvdbID:           episodeItem.TvdbID,
			SeriesID:         episodeItem.SeriesID,
			SeasonNumber:     season,
			QualityProfileID: episodeItem.QualityProfileID,
			HasFile:          true,
			CurrentQualityID: highestQID,
		}
	}

	return nil
}

// highestQualityForSeason finds the highest CurrentQualityID among wanted episodes
// for a given series+season in the WantedIndex.
func (m *Matcher) highestQualityForSeason(tvdbID int, seriesID int64, season int) int {
	if tvdbID != 0 {
		return highestQualityInItems(m.index.byTvdbID[tvdbID], seriesID, season)
	}

	highest := 0
	for _, items := range m.index.byTitle {
		if h := highestQualityInItems(items, seriesID, season); h > highest {
			highest = h
		}
	}
	return highest
}

func highestQualityInItems(items []decisioning.SearchableItem, seriesID int64, season int) int {
	highest := 0
	for i := range items {
		item := &items[i]
		if item.MediaType != decisioning.MediaTypeEpisode || item.SeriesID != seriesID || item.SeasonNumber != season {
			continue
		}
		if item.CurrentQualityID > highest {
			highest = item.CurrentQualityID
		}
	}
	return highest
}

func (m *Matcher) extractImdbID(release *types.TorrentInfo) string {
	if release.ImdbID != 0 {
		return "tt" + strconv.Itoa(release.ImdbID)
	}
	return ""
}

func (m *Matcher) extractTmdbID(release *types.TorrentInfo) int {
	return release.TmdbID
}

func (m *Matcher) extractTvdbID(release *types.TorrentInfo) int {
	return release.TvdbID
}

func itemKey(item *decisioning.SearchableItem) string {
	switch item.MediaType {
	case decisioning.MediaTypeMovie:
		return "movie:" + strconv.FormatInt(item.MediaID, 10)
	case decisioning.MediaTypeEpisode:
		return "episode:" + strconv.FormatInt(item.MediaID, 10)
	case decisioning.MediaTypeSeason:
		return "season:" + strconv.FormatInt(item.SeriesID, 10) + ":" + strconv.Itoa(item.SeasonNumber)
	default:
		return string(item.MediaType) + ":" + strconv.FormatInt(item.MediaID, 10)
	}
}

func seasonKeyForEpisode(item *decisioning.SearchableItem) string {
	return "season:" + strconv.FormatInt(item.SeriesID, 10) + ":" + strconv.Itoa(item.SeasonNumber)
}

func extractExternalIDs(release *types.TorrentInfo) {
	if release.ImdbID != 0 || release.InfoURL == "" {
		return
	}
	if v := parseImdbIDFromURL(release.InfoURL); v > 0 {
		release.ImdbID = v
	}
}

func parseImdbIDFromURL(url string) int {
	idx := strings.Index(url, "tt")
	if idx < 0 {
		return 0
	}
	end := idx + 2
	for end < len(url) && url[end] >= '0' && url[end] <= '9' {
		end++
	}
	if end <= idx+2 {
		return 0
	}
	v, err := strconv.Atoi(url[idx+2 : end])
	if err != nil {
		return 0
	}
	return v
}
