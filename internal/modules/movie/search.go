package movie

import (
	"context"

	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/titleutil"
)

var _ module.SearchStrategy = (*Module)(nil)

// Categories returns all movie Newznab categories (2000-2999).
func (m *Module) Categories() []int {
	return []int{2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060, 2070, 2080}
}

// DefaultSearchCategories returns the default movie search categories.
func (m *Module) DefaultSearchCategories() []int {
	return []int{2000, 2030, 2040, 2045, 2050, 2080}
}

// FilterRelease rejects releases that don't match the target movie.
func (m *Module) FilterRelease(_ context.Context, release *module.ReleaseForFilter, item module.SearchableItem) (reject bool, reason string) {
	if release.IsTV {
		return true, "is TV content, not movie"
	}
	if !titleutil.TitlesMatch(release.Title, item.GetTitle()) {
		return true, "title mismatch"
	}
	year := item.GetSearchParams().Extra["year"]
	if yearInt, ok := year.(int); ok && yearInt > 0 && release.Year > 0 {
		diff := yearInt - release.Year
		if diff < 0 {
			diff = -diff
		}
		if diff > 1 {
			return true, "year mismatch (>1 year difference)"
		}
	}
	return false, ""
}

// TitlesMatch checks if a release title matches a movie title.
func (m *Module) TitlesMatch(releaseTitle, mediaTitle string) bool {
	return titleutil.TitlesMatch(releaseTitle, mediaTitle)
}

// BuildSearchCriteria constructs Newznab search criteria for a movie.
func (m *Module) BuildSearchCriteria(item module.SearchableItem) module.SearchCriteria {
	criteria := module.SearchCriteria{
		ModuleType:  module.TypeMovie,
		Query:       item.GetTitle(),
		SearchType:  "movie",
		Categories:  m.Categories(),
		ExternalIDs: item.GetExternalIDs(),
	}

	if year, ok := item.GetSearchParams().Extra["year"].(int); ok && year > 0 {
		criteria.Year = year
	}

	return criteria
}

// IsGroupSearchEligible — movies have no hierarchy, always false.
func (m *Module) IsGroupSearchEligible(_ context.Context, _ module.EntityType, _ int64, _ int, _ bool) bool {
	return false
}

// SuppressChildSearches — movies have no children.
func (m *Module) SuppressChildSearches(_ module.EntityType, _ int64, _ int) []int64 {
	return nil
}
