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
	"github.com/slipstream/slipstream/internal/module"
)

// WantedIndex provides fast lookup of wanted media items by various keys.
type WantedIndex struct {
	byTitle  map[string][]module.SearchableItem
	byImdbID map[string][]module.SearchableItem
	byTmdbID map[int][]module.SearchableItem
	byTvdbID map[int][]module.SearchableItem
}

// BuildWantedIndex creates a WantedIndex from legacy decisioning items.
// Converts them to module.SearchableItem internally.
func BuildWantedIndex(items []decisioning.SearchableItem) *WantedIndex {
	return BuildWantedIndexFromModuleItems(ConvertLegacyItems(items))
}

// BuildWantedIndexFromModuleItems creates a WantedIndex from module SearchableItems.
func BuildWantedIndexFromModuleItems(items []module.SearchableItem) *WantedIndex {
	idx := &WantedIndex{
		byTitle:  make(map[string][]module.SearchableItem),
		byImdbID: make(map[string][]module.SearchableItem),
		byTmdbID: make(map[int][]module.SearchableItem),
		byTvdbID: make(map[int][]module.SearchableItem),
	}

	for _, item := range items {
		idx.addItem(item)
	}

	return idx
}

func (idx *WantedIndex) addItem(item module.SearchableItem) {
	normalized := search.NormalizeTitle(item.GetTitle())
	if normalized != "" {
		idx.byTitle[normalized] = append(idx.byTitle[normalized], item)
	}
	extIDs := item.GetExternalIDs()
	if imdbID := extIDs["imdbId"]; imdbID != "" {
		idx.byImdbID[imdbID] = append(idx.byImdbID[imdbID], item)
	}
	if tmdbStr := extIDs["tmdbId"]; tmdbStr != "" {
		if tmdbID, err := strconv.Atoi(tmdbStr); err == nil && tmdbID != 0 {
			idx.byTmdbID[tmdbID] = append(idx.byTmdbID[tmdbID], item)
		}
	}
	if tvdbStr := extIDs["tvdbId"]; tvdbStr != "" {
		if tvdbID, err := strconv.Atoi(tvdbStr); err == nil && tvdbID != 0 {
			idx.byTvdbID[tvdbID] = append(idx.byTvdbID[tvdbID], item)
		}
	}
}

// MatchResult pairs a release with the wanted item it matched.
type MatchResult struct {
	Release    types.TorrentInfo
	WantedItem module.SearchableItem
	IsSeason   bool
}

// Matcher matches RSS releases against a WantedIndex.
type Matcher struct {
	index    *WantedIndex
	queries  *sqlc.Queries
	registry *module.Registry
	logger   *zerolog.Logger
}

// NewMatcher creates a new Matcher.
func NewMatcher(index *WantedIndex, queries *sqlc.Queries, registry *module.Registry, logger *zerolog.Logger) *Matcher {
	return &Matcher{
		index:    index,
		queries:  queries,
		registry: registry,
		logger:   logger,
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

func (m *Matcher) findCandidates(release *types.TorrentInfo, parsed *scanner.ParsedMedia) []module.SearchableItem {
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

func (m *Matcher) matchMovie(release *types.TorrentInfo, parsed *scanner.ParsedMedia, candidates []module.SearchableItem) []MatchResult {
	matchedByID := release.ImdbID != 0 || release.TmdbID != 0 || release.TvdbID != 0
	var results []MatchResult
	for _, item := range candidates {
		if item.GetMediaType() != mediaTypeMovie {
			continue
		}
		sp := item.GetSearchParams()
		itemYear, _ := sp.Extra["year"].(int)
		if !matchedByID && parsed.Year > 0 && itemYear > 0 && parsed.Year != itemYear {
			continue
		}
		results = append(results, MatchResult{
			Release:    *release,
			WantedItem: item,
		})
	}
	return results
}

func (m *Matcher) matchEpisode(release *types.TorrentInfo, parsed *scanner.ParsedMedia, candidates []module.SearchableItem) []MatchResult {
	var results []MatchResult
	for _, item := range candidates {
		if item.GetMediaType() != mediaTypeEpisode {
			continue
		}
		sp := item.GetSearchParams()
		season, _ := sp.Extra["seasonNumber"].(int)
		episode, _ := sp.Extra["episodeNumber"].(int)
		if season != parsed.Season {
			continue
		}
		if episode != parsed.Episode {
			continue
		}
		results = append(results, MatchResult{
			Release:    *release,
			WantedItem: item,
		})
	}
	return results
}

func (m *Matcher) matchSeasonPack(ctx context.Context, release *types.TorrentInfo, parsed *scanner.ParsedMedia, candidates []module.SearchableItem) []MatchResult {
	var results []MatchResult

	for _, item := range candidates {
		if item.GetMediaType() != mediaTypeSeason {
			continue
		}
		sp := item.GetSearchParams()
		season, _ := sp.Extra["seasonNumber"].(int)
		if season == parsed.Season {
			results = append(results, MatchResult{
				Release:    *release,
				WantedItem: item,
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

	for _, item := range candidates {
		if item.GetMediaType() != mediaTypeEpisode {
			continue
		}
		sp := item.GetSearchParams()
		season, _ := sp.Extra["seasonNumber"].(int)
		if season != parsed.Season {
			continue
		}

		seriesID, _ := sp.Extra["seriesId"].(int64)
		key := seriesSeason{seriesID: seriesID, season: parsed.Season}
		if checked[key] {
			continue
		}
		checked[key] = true

		seasonItem := m.checkSeasonPackEligibility(ctx, item, seriesID, parsed.Season)
		if seasonItem == nil {
			continue
		}

		results = append(results, MatchResult{
			Release:    *release,
			WantedItem: seasonItem,
			IsSeason:   true,
		})
	}
	return results
}

func (m *Matcher) checkSeasonPackEligibility(ctx context.Context, episodeItem module.SearchableItem, seriesID int64, season int) module.SearchableItem {
	allMissing, allUpgradable := m.resolveSeasonEligibility(ctx, episodeItem, seriesID, season)

	if allMissing {
		return m.buildSyntheticSeasonItem(episodeItem, seriesID, season, false)
	}

	if allUpgradable {
		return m.buildSyntheticSeasonItem(episodeItem, seriesID, season, true)
	}

	return nil
}

func (m *Matcher) resolveSeasonEligibility(ctx context.Context, episodeItem module.SearchableItem, seriesID int64, season int) (allMissing, allUpgradable bool) {
	// Use module SearchStrategy if available
	if m.registry != nil {
		modType := module.Type(episodeItem.GetModuleType())
		if mod := m.registry.Get(modType); mod != nil {
			allMissing = mod.IsGroupSearchEligible(ctx, module.EntitySeries, seriesID, season, false)
			if !allMissing {
				allUpgradable = mod.IsGroupSearchEligible(ctx, module.EntitySeries, seriesID, season, true)
			}
			return
		}
	}

	// Fall back to legacy decisioning checks
	allMissing = decisioning.IsSeasonPackEligible(ctx, m.queries, m.logger, seriesID, season)
	if !allMissing {
		allUpgradable = decisioning.IsSeasonPackUpgradeEligible(ctx, m.queries, m.logger, seriesID, season)
	}
	return
}

func (m *Matcher) buildSyntheticSeasonItem(episodeItem module.SearchableItem, seriesID int64, season int, forUpgrade bool) module.SearchableItem {
	extra := map[string]any{
		"seriesId":     seriesID,
		"seasonNumber": season,
	}

	var currentQID *int64
	if forUpgrade {
		highestQID := m.highestQualityForSeason(episodeItem, seriesID, season)
		qid := int64(highestQID)
		currentQID = &qid
	}

	return module.NewWantedItem(
		module.Type(episodeItem.GetModuleType()),
		mediaTypeSeason,
		seriesID,
		episodeItem.GetTitle(),
		episodeItem.GetExternalIDs(),
		episodeItem.GetQualityProfileID(),
		currentQID,
		module.SearchParams{Extra: extra},
	)
}

// highestQualityForSeason finds the highest CurrentQualityID among wanted episodes
// for a given series+season in the WantedIndex.
func (m *Matcher) highestQualityForSeason(episodeItem module.SearchableItem, seriesID int64, season int) int {
	extIDs := episodeItem.GetExternalIDs()
	if tvdbStr := extIDs["tvdbId"]; tvdbStr != "" {
		if tvdbID, err := strconv.Atoi(tvdbStr); err == nil && tvdbID != 0 {
			return highestQualityInItems(m.index.byTvdbID[tvdbID], seriesID, season)
		}
	}

	highest := 0
	for _, items := range m.index.byTitle {
		if h := highestQualityInItems(items, seriesID, season); h > highest {
			highest = h
		}
	}
	return highest
}

func highestQualityInItems(items []module.SearchableItem, seriesID int64, season int) int {
	highest := 0
	for _, item := range items {
		if item.GetMediaType() != mediaTypeEpisode {
			continue
		}
		sp := item.GetSearchParams()
		itemSeriesID, _ := sp.Extra["seriesId"].(int64)
		itemSeason, _ := sp.Extra["seasonNumber"].(int)
		if itemSeriesID != seriesID || itemSeason != season {
			continue
		}
		if qid := item.GetCurrentQualityID(); qid != nil && int(*qid) > highest {
			highest = int(*qid)
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

func itemKey(item module.SearchableItem) string {
	sp := item.GetSearchParams()
	switch item.GetMediaType() {
	case mediaTypeMovie:
		return mediaTypeMovie + ":" + strconv.FormatInt(item.GetEntityID(), 10)
	case mediaTypeEpisode:
		return mediaTypeEpisode + ":" + strconv.FormatInt(item.GetEntityID(), 10)
	case mediaTypeSeason:
		seriesID, _ := sp.Extra["seriesId"].(int64)
		season, _ := sp.Extra["seasonNumber"].(int)
		return "season:" + strconv.FormatInt(seriesID, 10) + ":" + strconv.Itoa(season)
	default:
		return item.GetMediaType() + ":" + strconv.FormatInt(item.GetEntityID(), 10)
	}
}

func seasonKeyForItem(item module.SearchableItem) string {
	sp := item.GetSearchParams()
	seriesID, _ := sp.Extra["seriesId"].(int64)
	season, _ := sp.Extra["seasonNumber"].(int)
	return "season:" + strconv.FormatInt(seriesID, 10) + ":" + strconv.Itoa(season)
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

// ConvertLegacyItems converts legacy decisioning.SearchableItem to module.SearchableItem.
func ConvertLegacyItems(items []decisioning.SearchableItem) []module.SearchableItem {
	result := make([]module.SearchableItem, len(items))
	for i := range items {
		result[i] = convertLegacyItem(&items[i])
	}
	return result
}

func convertLegacyItem(item *decisioning.SearchableItem) module.SearchableItem {
	var modType module.Type
	switch item.MediaType {
	case decisioning.MediaTypeMovie:
		modType = module.TypeMovie
	default:
		modType = module.TypeTV
	}

	extIDs := make(map[string]string)
	if item.ImdbID != "" {
		extIDs["imdbId"] = item.ImdbID
	}
	if item.TmdbID != 0 {
		extIDs["tmdbId"] = strconv.Itoa(item.TmdbID)
	}
	if item.TvdbID != 0 {
		extIDs["tvdbId"] = strconv.Itoa(item.TvdbID)
	}

	extra := map[string]any{
		"year":          item.Year,
		"seriesId":      item.SeriesID,
		"seasonNumber":  item.SeasonNumber,
		"episodeNumber": item.EpisodeNumber,
	}
	if item.TargetSlotID != nil {
		extra["targetSlotId"] = item.TargetSlotID
	}

	var currentQID *int64
	if item.HasFile && item.CurrentQualityID > 0 {
		qid := int64(item.CurrentQualityID)
		currentQID = &qid
	}

	return module.NewWantedItem(
		modType,
		string(item.MediaType),
		item.MediaID,
		item.Title,
		extIDs,
		item.QualityProfileID,
		currentQID,
		module.SearchParams{Extra: extra},
	)
}

// moduleItemToLegacy converts a module.SearchableItem back to a decisioning.SearchableItem
// for use with legacy decisioning functions (SelectBestRelease, etc.).
func moduleItemToLegacy(item module.SearchableItem) decisioning.SearchableItem {
	sp := item.GetSearchParams()

	legacy := decisioning.SearchableItem{
		MediaType:        decisioning.MediaType(item.GetMediaType()),
		MediaID:          item.GetEntityID(),
		Title:            item.GetTitle(),
		QualityProfileID: item.GetQualityProfileID(),
	}

	applySearchParamsToLegacy(&legacy, sp)
	applyExternalIDsToLegacy(&legacy, item.GetExternalIDs())

	if qid := item.GetCurrentQualityID(); qid != nil {
		legacy.HasFile = true
		legacy.CurrentQualityID = int(*qid)
	}

	return legacy
}

func applySearchParamsToLegacy(legacy *decisioning.SearchableItem, sp module.SearchParams) {
	if v, ok := sp.Extra["year"].(int); ok {
		legacy.Year = v
	}
	if v, ok := sp.Extra["seriesId"].(int64); ok {
		legacy.SeriesID = v
	}
	if v, ok := sp.Extra["seasonNumber"].(int); ok {
		legacy.SeasonNumber = v
	}
	if v, ok := sp.Extra["episodeNumber"].(int); ok {
		legacy.EpisodeNumber = v
	}
	if v, ok := sp.Extra["targetSlotId"].(*int64); ok {
		legacy.TargetSlotID = v
	}
}

func applyExternalIDsToLegacy(legacy *decisioning.SearchableItem, extIDs map[string]string) {
	legacy.ImdbID = extIDs["imdbId"]
	if tmdbStr := extIDs["tmdbId"]; tmdbStr != "" {
		if tmdbID, err := strconv.Atoi(tmdbStr); err == nil {
			legacy.TmdbID = tmdbID
		}
	}
	if tvdbStr := extIDs["tvdbId"]; tvdbStr != "" {
		if tvdbID, err := strconv.Atoi(tvdbStr); err == nil {
			legacy.TvdbID = tvdbID
		}
	}
}
