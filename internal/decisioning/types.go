package decisioning

import (
	"context"

	"github.com/slipstream/slipstream/internal/module"
)

// MediaType represents the type of media being searched.
type MediaType string

const (
	MediaTypeMovie   MediaType = "movie"
	MediaTypeEpisode MediaType = "episode"
	MediaTypeSeason  MediaType = "season"
	MediaTypeSeries  MediaType = "series"
)

// DefaultStrategy is a pass-through SearchStrategy that accepts all releases.
// Used as a fallback when no module is registered for an item's type.
type DefaultStrategy struct{}

func (DefaultStrategy) Categories() []int              { return nil }
func (DefaultStrategy) DefaultSearchCategories() []int { return nil }
func (DefaultStrategy) TitlesMatch(_, _ string) bool   { return true }
func (DefaultStrategy) BuildSearchCriteria(item module.SearchableItem) module.SearchCriteria {
	return module.SearchCriteria{Query: item.GetTitle()}
}
func (DefaultStrategy) FilterRelease(_ context.Context, _ *module.ReleaseForFilter, _ module.SearchableItem) (reject bool, reason string) {
	return false, ""
}
func (DefaultStrategy) IsGroupSearchEligible(_ context.Context, _ module.EntityType, _ int64, _ int, _ bool) bool {
	return false
}
func (DefaultStrategy) SuppressChildSearches(_ module.EntityType, _ int64, _ int) []int64 {
	return nil
}

// FallbackReleaseParser is a minimal ReleaseParser that returns a ReleaseForFilter
// with only the raw title, size, and categories. Used when no module registry is available.
func FallbackReleaseParser(title string, size int64, categories []int) *module.ReleaseForFilter {
	return &module.ReleaseForFilter{
		Title:      title,
		Size:       size,
		Categories: categories,
	}
}
