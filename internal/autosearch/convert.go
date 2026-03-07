package autosearch

import (
	"strconv"

	"github.com/slipstream/slipstream/internal/decisioning"
	types "github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/module"
)

// convertModuleCriteria converts module.SearchCriteria to indexer types.SearchCriteria.
func convertModuleCriteria(mc *module.SearchCriteria) types.SearchCriteria { //nolint:unused // used when module registry is fully active
	ic := types.SearchCriteria{
		ModuleType: string(mc.ModuleType),
		Query:      mc.Query,
		Type:       mc.SearchType,
		Categories: mc.Categories,
		Year:       mc.Year,
		Season:     mc.Season,
		Episode:    mc.Episode,
	}
	if v, ok := mc.ExternalIDs["imdbId"]; ok {
		ic.ImdbID = v
	}
	if v, ok := mc.ExternalIDs["tmdbId"]; ok {
		if id, err := strconv.Atoi(v); err == nil {
			ic.TmdbID = id
		}
	}
	if v, ok := mc.ExternalIDs["tvdbId"]; ok {
		if id, err := strconv.Atoi(v); err == nil {
			ic.TvdbID = id
		}
	}
	return ic
}

// moduleItemToSearchableItem converts a module.SearchableItem to the legacy
// decisioning.SearchableItem struct. Temporary bridge for Phase 5.
func moduleItemToSearchableItem(item module.SearchableItem) decisioning.SearchableItem {
	result := decisioning.SearchableItem{
		MediaType: decisioning.MediaType(item.GetMediaType()),
		MediaID:   item.GetEntityID(),
		Title:     item.GetTitle(),
	}

	result.QualityProfileID = item.GetQualityProfileID()
	if qid := item.GetCurrentQualityID(); qid != nil {
		result.HasFile = true
		result.CurrentQualityID = int(*qid)
	}

	applyExternalIDs(&result, item.GetExternalIDs())
	applySearchParams(&result, item.GetSearchParams())

	return result
}

func applyExternalIDs(result *decisioning.SearchableItem, ids map[string]string) {
	if v, ok := ids["imdbId"]; ok {
		result.ImdbID = v
	}
	if v, ok := ids["tmdbId"]; ok {
		if id, err := strconv.Atoi(v); err == nil {
			result.TmdbID = id
		}
	}
	if v, ok := ids["tvdbId"]; ok {
		if id, err := strconv.Atoi(v); err == nil {
			result.TvdbID = id
		}
	}
}

func applySearchParams(result *decisioning.SearchableItem, params module.SearchParams) {
	if v, ok := params.Extra["year"].(int); ok {
		result.Year = v
	}
	if v, ok := params.Extra["seriesId"].(int64); ok {
		result.SeriesID = v
	}
	if v, ok := params.Extra["seasonNumber"].(int); ok {
		result.SeasonNumber = v
	}
	if v, ok := params.Extra["episodeNumber"].(int); ok {
		result.EpisodeNumber = v
	}
}
