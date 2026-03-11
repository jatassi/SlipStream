package module

import "strconv"

// Accessor helpers for SearchableItem — extract commonly needed fields
// from the generic interface without callers needing to know about
// SearchParams.Extra or external ID map keys.

// ItemYear returns the year from a SearchableItem's search params.
func ItemYear(item SearchableItem) int {
	v, _ := item.GetSearchParams().Extra["year"].(int)
	return v
}

// ItemSeriesID returns the series ID from a SearchableItem's search params.
func ItemSeriesID(item SearchableItem) int64 {
	v, _ := item.GetSearchParams().Extra["seriesId"].(int64)
	return v
}

// ItemSeasonNumber returns the season number from a SearchableItem's search params.
func ItemSeasonNumber(item SearchableItem) int {
	v, _ := item.GetSearchParams().Extra["seasonNumber"].(int)
	return v
}

// ItemEpisodeNumber returns the episode number from a SearchableItem's search params.
func ItemEpisodeNumber(item SearchableItem) int {
	v, _ := item.GetSearchParams().Extra["episodeNumber"].(int)
	return v
}

// ItemHasFile returns true if the item has a file (CurrentQualityID is set).
func ItemHasFile(item SearchableItem) bool {
	return item.GetCurrentQualityID() != nil
}

// ItemCurrentQualityID returns the current quality ID, or 0 if unset.
func ItemCurrentQualityID(item SearchableItem) int {
	if qid := item.GetCurrentQualityID(); qid != nil {
		return int(*qid)
	}
	return 0
}

// ItemTargetSlotID returns the target slot ID from search params, or nil.
func ItemTargetSlotID(item SearchableItem) *int64 {
	v, _ := item.GetSearchParams().Extra["targetSlotId"].(*int64)
	return v
}

// CloneWithSlotOverrides creates a new WantedItem based on an existing SearchableItem
// but with the quality profile ID and target slot ID overridden for slot-specific searching.
func CloneWithSlotOverrides(item SearchableItem, qualityProfileID int64, targetSlotID *int64) SearchableItem {
	sp := item.GetSearchParams()
	extra := make(map[string]any, len(sp.Extra)+1)
	for k, v := range sp.Extra {
		extra[k] = v
	}
	if targetSlotID != nil {
		extra["targetSlotId"] = targetSlotID
	}
	return NewWantedItem(
		Type(item.GetModuleType()),
		item.GetMediaType(),
		item.GetEntityID(),
		item.GetTitle(),
		item.GetExternalIDs(),
		qualityProfileID,
		item.GetCurrentQualityID(),
		SearchParams{Extra: extra},
	)
}

// ItemImdbID returns the IMDB ID string from external IDs.
func ItemImdbID(item SearchableItem) string {
	return item.GetExternalIDs()["imdbId"]
}

// ItemTmdbID returns the TMDB ID as int from external IDs.
func ItemTmdbID(item SearchableItem) int {
	if s := item.GetExternalIDs()["tmdbId"]; s != "" {
		if id, err := strconv.Atoi(s); err == nil {
			return id
		}
	}
	return 0
}

// ItemTvdbID returns the TVDB ID as int from external IDs.
func ItemTvdbID(item SearchableItem) int {
	if s := item.GetExternalIDs()["tvdbId"]; s != "" {
		if id, err := strconv.Atoi(s); err == nil {
			return id
		}
	}
	return 0
}
