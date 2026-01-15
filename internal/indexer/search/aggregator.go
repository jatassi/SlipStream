package search

import (
	"sort"
	"strings"

	"github.com/slipstream/slipstream/internal/indexer/types"
	"github.com/slipstream/slipstream/internal/library/scanner"
)

// aggregateResults combines results from multiple indexers.
func (s *Service) aggregateResults(results <-chan searchTaskResult) *SearchResult {
	allReleases := make([]types.ReleaseInfo, 0)
	errors := make([]SearchIndexerError, 0)
	indexersUsed := 0

	for result := range results {
		if result.Error != nil {
			errors = append(errors, SearchIndexerError{
				IndexerID:   result.IndexerID,
				IndexerName: result.IndexerName,
				Error:       result.Error.Error(),
			})
			continue
		}
		indexersUsed++
		allReleases = append(allReleases, result.Releases...)
	}

	// Deduplicate by GUID, prefer higher priority indexer (lower priority number)
	deduplicated := deduplicateReleases(allReleases)

	// Enrich with parsed quality info
	enrichWithQuality(deduplicated)

	// Sort: by publish date descending (newest first)
	sortReleases(deduplicated)

	return &SearchResult{
		Releases:      deduplicated,
		TotalResults:  len(deduplicated),
		IndexersUsed:  indexersUsed,
		IndexerErrors: errors,
	}
}

// aggregateTorrentResults combines torrent results from multiple indexers.
// If criteria is provided, results are filtered to match the search criteria.
func (s *Service) aggregateTorrentResults(results <-chan searchTaskResult, criteria *types.SearchCriteria) *TorrentSearchResult {
	allTorrents := make([]types.TorrentInfo, 0)
	errors := make([]SearchIndexerError, 0)
	indexersUsed := 0

	for result := range results {
		if result.Error != nil {
			errors = append(errors, SearchIndexerError{
				IndexerID:   result.IndexerID,
				IndexerName: result.IndexerName,
				Error:       result.Error.Error(),
			})
			continue
		}
		indexersUsed++
		s.logger.Info().
			Str("indexer", result.IndexerName).
			Int("results", len(result.Torrents)).
			Msg("Received results from indexer")
		allTorrents = append(allTorrents, result.Torrents...)
	}

	totalRaw := len(allTorrents)

	// Deduplicate by GUID or InfoHash
	deduplicated := deduplicateTorrents(allTorrents)
	afterDedup := len(deduplicated)

	// Enrich with parsed quality info
	enrichTorrentsWithQuality(deduplicated)

	// Filter by search criteria (title, season/episode for TV, year for movies)
	if criteria != nil {
		beforeFilter := len(deduplicated)
		deduplicated = FilterByCriteria(deduplicated, *criteria)
		s.logger.Info().
			Str("query", criteria.Query).
			Str("type", criteria.Type).
			Int("season", criteria.Season).
			Int("episode", criteria.Episode).
			Int("year", criteria.Year).
			Int("beforeFilter", beforeFilter).
			Int("afterFilter", len(deduplicated)).
			Int("filtered", beforeFilter-len(deduplicated)).
			Msg("Applied search criteria filter")
	}

	// Sort: by seeders descending (most seeds first)
	sortTorrents(deduplicated)

	s.logger.Info().
		Int("totalRaw", totalRaw).
		Int("afterDedup", afterDedup).
		Int("finalResults", len(deduplicated)).
		Int("indexersUsed", indexersUsed).
		Msg("Aggregation complete")

	return &TorrentSearchResult{
		Releases:      deduplicated,
		TotalResults:  len(deduplicated),
		IndexersUsed:  indexersUsed,
		IndexerErrors: errors,
	}
}

// deduplicateReleases removes duplicate releases based on GUID.
// When duplicates are found, keeps the one from the higher priority indexer.
func deduplicateReleases(releases []types.ReleaseInfo) []types.ReleaseInfo {
	if len(releases) == 0 {
		return releases
	}

	// Map to track best release for each GUID
	seen := make(map[string]int) // GUID -> index in result slice
	result := make([]types.ReleaseInfo, 0, len(releases))

	for _, release := range releases {
		// Normalize GUID for comparison
		guid := normalizeGUID(release.GUID)

		if existingIdx, exists := seen[guid]; exists {
			// Duplicate found - keep the one with higher priority (lower number)
			// Since we don't have priority in ReleaseInfo, we'll keep the first one
			// This assumes indexers are processed in priority order
			existing := result[existingIdx]
			if release.IndexerID < existing.IndexerID {
				result[existingIdx] = release
			}
		} else {
			seen[guid] = len(result)
			result = append(result, release)
		}
	}

	return result
}

// deduplicateTorrents removes duplicate torrents based on GUID or InfoHash.
// When duplicates are found, keeps the one with the most seeders.
func deduplicateTorrents(torrents []types.TorrentInfo) []types.TorrentInfo {
	if len(torrents) == 0 {
		return torrents
	}

	// Map to track best torrent for each unique identifier
	// Use InfoHash if available, otherwise GUID
	seen := make(map[string]int) // identifier -> index in result slice
	result := make([]types.TorrentInfo, 0, len(torrents))

	for _, torrent := range torrents {
		// Use InfoHash if available (more reliable), otherwise use GUID
		var identifier string
		if torrent.InfoHash != "" {
			identifier = "hash:" + strings.ToLower(torrent.InfoHash)
		} else {
			identifier = "guid:" + normalizeGUID(torrent.GUID)
		}

		if existingIdx, exists := seen[identifier]; exists {
			// Duplicate found - keep the one with more seeders
			existing := result[existingIdx]
			if torrent.Seeders > existing.Seeders {
				result[existingIdx] = torrent
			}
		} else {
			seen[identifier] = len(result)
			result = append(result, torrent)
		}
	}

	return result
}

// normalizeGUID normalizes a GUID for comparison.
func normalizeGUID(guid string) string {
	// Trim whitespace and convert to lowercase
	return strings.ToLower(strings.TrimSpace(guid))
}

// sortReleases sorts releases by publish date descending (newest first).
func sortReleases(releases []types.ReleaseInfo) {
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].PublishDate.After(releases[j].PublishDate)
	})
}

// sortTorrents sorts torrents by seeders descending (most seeds first).
func sortTorrents(torrents []types.TorrentInfo) {
	sort.Slice(torrents, func(i, j int) bool {
		// Primary sort: seeders descending
		if torrents[i].Seeders != torrents[j].Seeders {
			return torrents[i].Seeders > torrents[j].Seeders
		}
		// Secondary sort: size descending (prefer larger files for same seeders)
		return torrents[i].Size > torrents[j].Size
	})
}

// FilterByQuality filters releases based on quality criteria.
// This is a placeholder for future quality profile integration.
func FilterByQuality(releases []types.ReleaseInfo, minSize, maxSize int64) []types.ReleaseInfo {
	if minSize == 0 && maxSize == 0 {
		return releases
	}

	filtered := make([]types.ReleaseInfo, 0, len(releases))
	for _, release := range releases {
		if minSize > 0 && release.Size < minSize {
			continue
		}
		if maxSize > 0 && release.Size > maxSize {
			continue
		}
		filtered = append(filtered, release)
	}
	return filtered
}

// FilterTorrentsByQuality filters torrents based on quality criteria.
func FilterTorrentsByQuality(torrents []types.TorrentInfo, minSeeders int, minSize, maxSize int64) []types.TorrentInfo {
	filtered := make([]types.TorrentInfo, 0, len(torrents))
	for _, torrent := range torrents {
		if minSeeders > 0 && torrent.Seeders < minSeeders {
			continue
		}
		if minSize > 0 && torrent.Size < minSize {
			continue
		}
		if maxSize > 0 && torrent.Size > maxSize {
			continue
		}
		filtered = append(filtered, torrent)
	}
	return filtered
}

// FilterFreeleech returns only freeleech torrents (downloadVolumeFactor == 0).
func FilterFreeleech(torrents []types.TorrentInfo) []types.TorrentInfo {
	filtered := make([]types.TorrentInfo, 0)
	for _, torrent := range torrents {
		if torrent.DownloadVolumeFactor == 0 {
			filtered = append(filtered, torrent)
		}
	}
	return filtered
}

// enrichWithQuality parses quality info from release titles using the scanner parser.
func enrichWithQuality(releases []types.ReleaseInfo) {
	for i := range releases {
		parsed := scanner.ParseFilename(releases[i].Title)
		releases[i].Quality = parsed.Quality
		releases[i].Source = parsed.Source
		releases[i].Resolution = qualityToResolution(parsed.Quality)
	}
}

// enrichTorrentsWithQuality parses quality info from torrent titles using the scanner parser.
func enrichTorrentsWithQuality(torrents []types.TorrentInfo) {
	for i := range torrents {
		parsed := scanner.ParseFilename(torrents[i].Title)
		torrents[i].Quality = parsed.Quality
		torrents[i].Source = parsed.Source
		torrents[i].Resolution = qualityToResolution(parsed.Quality)
	}
}

// qualityToResolution converts a quality string to a resolution integer.
func qualityToResolution(quality string) int {
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

// FilterByCriteria filters torrent results based on search criteria.
// For TV searches: validates content type, title match, season, and episode.
// For movie searches: validates content type, title match, and year (±1 tolerance).
func FilterByCriteria(torrents []types.TorrentInfo, criteria types.SearchCriteria) []types.TorrentInfo {
	if criteria.Query == "" {
		return torrents
	}

	filtered := make([]types.TorrentInfo, 0, len(torrents))

	for _, torrent := range torrents {
		parsed := scanner.ParseFilename(torrent.Title)

		switch criteria.Type {
		case "tvsearch":
			if getTVFilterReason(*parsed, criteria) != "" {
				continue
			}
		case "movie":
			if getMovieFilterReason(*parsed, criteria) != "" {
				continue
			}
		}

		filtered = append(filtered, torrent)
	}

	return filtered
}

// getTVFilterReason returns the reason a TV result was filtered, or empty string if it passes.
func getTVFilterReason(parsed scanner.ParsedMedia, criteria types.SearchCriteria) string {
	if !parsed.IsTV {
		return "not TV content"
	}
	if !TitlesMatch(parsed.Title, criteria.Query) {
		return "title mismatch: '" + parsed.Title + "' != '" + criteria.Query + "'"
	}
	if criteria.Season > 0 {
		if parsed.IsSeasonPack || parsed.IsCompleteSeries {
			if parsed.Season > 0 && parsed.Season != criteria.Season {
				if parsed.EndSeason > 0 {
					if criteria.Season < parsed.Season || criteria.Season > parsed.EndSeason {
						return "season not in pack range"
					}
				} else {
					return "wrong season pack"
				}
			}
		} else {
			if parsed.Season != criteria.Season {
				return "wrong season"
			}
		}
	}
	if criteria.Episode > 0 {
		if parsed.IsSeasonPack || parsed.IsCompleteSeries {
			return ""
		}
		if parsed.Episode > 0 && parsed.Episode != criteria.Episode {
			if parsed.EndEpisode > 0 {
				if criteria.Episode < parsed.Episode || criteria.Episode > parsed.EndEpisode {
					return "episode not in multi-episode range"
				}
			} else {
				return "wrong episode"
			}
		}
	}
	return ""
}

// getMovieFilterReason returns the reason a movie result was filtered, or empty string if it passes.
func getMovieFilterReason(parsed scanner.ParsedMedia, criteria types.SearchCriteria) string {
	if parsed.IsTV {
		return "is TV content, not movie"
	}
	if !TitlesMatch(parsed.Title, criteria.Query) {
		return "title mismatch: '" + parsed.Title + "' != '" + criteria.Query + "'"
	}
	if criteria.Year > 0 && parsed.Year > 0 {
		yearDiff := criteria.Year - parsed.Year
		if yearDiff < 0 {
			yearDiff = -yearDiff
		}
		if yearDiff > 1 {
			return "year mismatch (>1 year difference)"
		}
	}
	return ""
}

// matchesTVCriteria checks if a parsed media matches TV search criteria.
func matchesTVCriteria(parsed scanner.ParsedMedia, criteria types.SearchCriteria) bool {
	// Must be TV content
	if !parsed.IsTV {
		return false
	}

	// Title must match exactly
	if !TitlesMatch(parsed.Title, criteria.Query) {
		return false
	}

	// Season validation (if specified)
	if criteria.Season > 0 {
		// For season packs, check if season matches
		if parsed.IsSeasonPack || parsed.IsCompleteSeries {
			// Season pack: check if requested season is within range
			if parsed.Season > 0 && parsed.Season != criteria.Season {
				// Check if it's a multi-season pack containing our season
				if parsed.EndSeason > 0 {
					if criteria.Season < parsed.Season || criteria.Season > parsed.EndSeason {
						return false
					}
				} else {
					return false
				}
			}
		} else {
			// Single episode: season must match
			if parsed.Season != criteria.Season {
				return false
			}
		}
	}

	// Episode validation (if specified)
	if criteria.Episode > 0 {
		// Season packs always pass episode check (they contain all episodes)
		if parsed.IsSeasonPack || parsed.IsCompleteSeries {
			return true
		}

		// For individual episodes, check exact match or multi-episode range
		if parsed.Episode > 0 && parsed.Episode != criteria.Episode {
			// Check if it's a multi-episode release containing our episode
			if parsed.EndEpisode > 0 {
				if criteria.Episode < parsed.Episode || criteria.Episode > parsed.EndEpisode {
					return false
				}
			} else {
				return false
			}
		}
	}

	return true
}

// matchesMovieCriteria checks if a parsed media matches movie search criteria.
func matchesMovieCriteria(parsed scanner.ParsedMedia, criteria types.SearchCriteria) bool {
	// Must be movie content (not TV)
	if parsed.IsTV {
		return false
	}

	// Title must match exactly
	if !TitlesMatch(parsed.Title, criteria.Query) {
		return false
	}

	// Year validation with ±1 tolerance
	if criteria.Year > 0 && parsed.Year > 0 {
		yearDiff := criteria.Year - parsed.Year
		if yearDiff < 0 {
			yearDiff = -yearDiff
		}
		if yearDiff > 1 {
			return false
		}
	}

	return true
}
