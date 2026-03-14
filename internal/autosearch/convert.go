package autosearch

import (
	"strconv"

	types "github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/module"
)

// convertModuleCriteria converts module.SearchCriteria to indexer types.SearchCriteria.
func convertModuleCriteria(mc *module.SearchCriteria) types.SearchCriteria {
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
