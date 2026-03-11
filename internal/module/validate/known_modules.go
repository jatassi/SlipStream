package validate

import (
	"github.com/slipstream/slipstream/internal/module"
	moviemod "github.com/slipstream/slipstream/internal/modules/movie"
	"github.com/slipstream/slipstream/internal/modules/shared"
	tvmod "github.com/slipstream/slipstream/internal/modules/tv"
)

// KnownModules returns ModuleInfo for all built-in modules.
// These use lightweight Descriptor types and static data, avoiding the
// heavy runtime dependencies (DB, services) that full Module instances require.
func KnownModules() map[string]*ModuleInfo {
	movieDesc := &moviemod.Descriptor{}
	tvDesc := &tvmod.Descriptor{}

	return map[string]*ModuleInfo{
		"movie": {
			Descriptor:   movieDesc,
			QualityItems: shared.VideoQualityItems(),
			Events: []module.NotificationEvent{
				{ID: moviemod.EventMovieAdded, Label: "Movie Added", Description: "When a movie is added to the library"},
				{ID: moviemod.EventMovieDeleted, Label: "Movie Deleted", Description: "When a movie is removed"},
			},
			Categories: []int{2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060, 2070, 2080},
		},
		"tv": {
			Descriptor:   tvDesc,
			QualityItems: shared.VideoQualityItems(),
			Events: []module.NotificationEvent{
				{ID: tvmod.EventTVAdded, Label: "Series Added", Description: "When a series is added to the library"},
				{ID: tvmod.EventTVDeleted, Label: "Series Deleted", Description: "When a series is removed"},
			},
			Categories: []int{5000, 5010, 5020, 5030, 5040, 5045, 5060, 5070, 5080, 5090},
		},
	}
}

// AllModuleInfos returns all known ModuleInfo as a slice.
func AllModuleInfos() []*ModuleInfo {
	known := KnownModules()
	result := make([]*ModuleInfo, 0, len(known))
	for _, info := range known {
		result = append(result, info)
	}
	return result
}
