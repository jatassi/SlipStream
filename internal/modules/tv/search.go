package tv

import (
	"context"

	"github.com/slipstream/slipstream/internal/decisioning"
	"github.com/slipstream/slipstream/internal/module"
	"github.com/slipstream/slipstream/internal/module/titleutil"
)

var _ module.SearchStrategy = (*Module)(nil)

// Categories returns all TV Newznab categories (5000-5999).
func (m *Module) Categories() []int {
	return []int{5000, 5010, 5020, 5030, 5040, 5045, 5060, 5070, 5080, 5090}
}

// DefaultSearchCategories returns the default TV search categories.
func (m *Module) DefaultSearchCategories() []int {
	return []int{5000, 5030, 5040, 5045, 5070, 5090}
}

// FilterRelease rejects releases that don't match the target episode/season.
func (m *Module) FilterRelease(_ context.Context, release *module.ReleaseForFilter, item module.SearchableItem) (reject bool, reason string) {
	if !release.IsTV {
		return true, "not TV content"
	}
	if !titleutil.TVTitlesMatch(release.Title, item.GetTitle()) {
		return true, "title mismatch"
	}

	seasonNumber := m.extractSeasonNumber(item)
	if reject, reason := m.checkSeasonMatch(release, seasonNumber); reject {
		return true, reason
	}

	switch item.GetMediaType() {
	case string(module.EntityEpisode):
		episodeNumber := m.extractEpisodeNumber(item)
		if reject, reason := m.checkEpisodeMatch(release, episodeNumber); reject {
			return true, reason
		}
	case string(module.EntitySeason):
		if !release.IsSeasonPack && !release.IsCompleteSeries {
			return true, "not a season pack"
		}
	}

	return false, ""
}

func (m *Module) checkSeasonMatch(release *module.ReleaseForFilter, seasonNumber int) (reject bool, reason string) {
	if seasonNumber <= 0 {
		return false, ""
	}
	if release.IsSeasonPack || release.IsCompleteSeries {
		if release.Season > 0 && release.Season != seasonNumber {
			if release.EndSeason > 0 && seasonNumber >= release.Season && seasonNumber <= release.EndSeason {
				return false, ""
			}
			return true, "wrong season pack"
		}
		return false, ""
	}
	if release.Season != seasonNumber {
		return true, "wrong season"
	}
	return false, ""
}

func (m *Module) checkEpisodeMatch(release *module.ReleaseForFilter, episodeNumber int) (reject bool, reason string) {
	if episodeNumber <= 0 {
		return false, ""
	}
	if release.IsSeasonPack || release.IsCompleteSeries {
		return true, "season pack during episode search"
	}
	if release.Episode > 0 && release.Episode != episodeNumber {
		if release.EndEpisode > 0 && episodeNumber >= release.Episode && episodeNumber <= release.EndEpisode {
			return false, ""
		}
		return true, "wrong episode"
	}
	return false, ""
}

// TitlesMatch checks if a release title matches a series title using TV-specific matching.
func (m *Module) TitlesMatch(releaseTitle, mediaTitle string) bool {
	return titleutil.TVTitlesMatch(releaseTitle, mediaTitle)
}

// BuildSearchCriteria constructs Newznab search criteria for a TV episode or season.
func (m *Module) BuildSearchCriteria(item module.SearchableItem) module.SearchCriteria {
	mediaType := item.GetMediaType()
	criteria := module.SearchCriteria{
		ModuleType:  module.TypeTV,
		Query:       item.GetTitle(),
		SearchType:  "tvsearch",
		ExternalIDs: item.GetExternalIDs(),
	}

	seasonNumber := m.extractSeasonNumber(item)
	episodeNumber := m.extractEpisodeNumber(item)

	switch mediaType {
	case string(module.EntityEpisode):
		criteria.Categories = m.Categories()
		criteria.Season = seasonNumber
		criteria.Episode = episodeNumber
	case string(module.EntitySeason):
		// Don't set categories or season for season pack searches
	}

	return criteria
}

// IsGroupSearchEligible checks if a season is eligible for season pack search.
func (m *Module) IsGroupSearchEligible(ctx context.Context, _ module.EntityType, seriesID int64, seasonNumber int, forUpgrade bool) bool {
	if forUpgrade {
		return decisioning.IsSeasonPackUpgradeEligible(ctx, m.queries, m.logger, seriesID, seasonNumber)
	}
	return decisioning.IsSeasonPackEligible(ctx, m.queries, m.logger, seriesID, seasonNumber)
}

// SuppressChildSearches returns episode IDs covered by a season pack grab.
func (m *Module) SuppressChildSearches(_ module.EntityType, _ int64, _ int) []int64 {
	return nil
}

func (m *Module) extractSeasonNumber(item module.SearchableItem) int {
	if v, ok := item.GetSearchParams().Extra["seasonNumber"].(int); ok {
		return v
	}
	return 0
}

func (m *Module) extractEpisodeNumber(item module.SearchableItem) int {
	if v, ok := item.GetSearchParams().Extra["episodeNumber"].(int); ok {
		return v
	}
	return 0
}
