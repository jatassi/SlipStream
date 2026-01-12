package renamer

import (
	"fmt"
	"sort"
	"strings"
)

// MultiEpisodeStyle defines how to format multi-episode filenames.
type MultiEpisodeStyle string

const (
	// StyleExtend: S01E01-02-03
	StyleExtend MultiEpisodeStyle = "extend"
	// StyleDuplicate: S01E01.S01E02.S01E03
	StyleDuplicate MultiEpisodeStyle = "duplicate"
	// StyleRepeat: S01E01E02E03
	StyleRepeat MultiEpisodeStyle = "repeat"
	// StyleScene: S01E01-E02-E03
	StyleScene MultiEpisodeStyle = "scene"
	// StyleRange: S01E01-03
	StyleRange MultiEpisodeStyle = "range"
	// StylePrefixedRange: S01E01-E03
	StylePrefixedRange MultiEpisodeStyle = "prefixed_range"
)

// FormatMultiEpisode formats a multi-episode identifier based on the style.
// padding specifies the number of digits for episode numbers (typically 2).
func FormatMultiEpisode(season int, episodes []int, style MultiEpisodeStyle, padding int) string {
	if len(episodes) == 0 {
		return ""
	}

	if padding < 1 {
		padding = 2
	}

	// Sort episodes to ensure consistent ordering
	sorted := make([]int, len(episodes))
	copy(sorted, episodes)
	sort.Ints(sorted)

	// Single episode - no special formatting needed
	if len(sorted) == 1 {
		return formatSeasonEpisode(season, sorted[0], padding)
	}

	switch style {
	case StyleExtend:
		return formatExtend(season, sorted, padding)
	case StyleDuplicate:
		return formatDuplicate(season, sorted, padding)
	case StyleRepeat:
		return formatRepeat(season, sorted, padding)
	case StyleScene:
		return formatScene(season, sorted, padding)
	case StyleRange:
		return formatRange(season, sorted, padding)
	case StylePrefixedRange:
		return formatPrefixedRange(season, sorted, padding)
	default:
		return formatExtend(season, sorted, padding)
	}
}

// formatSeasonEpisode formats a single season+episode identifier.
func formatSeasonEpisode(season, episode, padding int) string {
	epFormat := fmt.Sprintf("%%0%dd", padding)
	return fmt.Sprintf("S%02dE"+epFormat, season, episode)
}

// formatEpisodeOnly formats just the episode number with padding.
func formatEpisodeOnly(episode, padding int) string {
	return fmt.Sprintf("%0*d", padding, episode)
}

// formatExtend: S01E01-02-03
func formatExtend(season int, episodes []int, padding int) string {
	var parts []string
	parts = append(parts, formatSeasonEpisode(season, episodes[0], padding))

	for i := 1; i < len(episodes); i++ {
		parts = append(parts, formatEpisodeOnly(episodes[i], padding))
	}

	return strings.Join(parts, "-")
}

// formatDuplicate: S01E01.S01E02.S01E03
func formatDuplicate(season int, episodes []int, padding int) string {
	var parts []string
	for _, ep := range episodes {
		parts = append(parts, formatSeasonEpisode(season, ep, padding))
	}
	return strings.Join(parts, ".")
}

// formatRepeat: S01E01E02E03
func formatRepeat(season int, episodes []int, padding int) string {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("S%02d", season))

	epFormat := fmt.Sprintf("E%%0%dd", padding)
	for _, ep := range episodes {
		result.WriteString(fmt.Sprintf(epFormat, ep))
	}

	return result.String()
}

// formatScene: S01E01-E02-E03
func formatScene(season int, episodes []int, padding int) string {
	var parts []string
	parts = append(parts, formatSeasonEpisode(season, episodes[0], padding))

	epFormat := fmt.Sprintf("E%%0%dd", padding)
	for i := 1; i < len(episodes); i++ {
		parts = append(parts, fmt.Sprintf(epFormat, episodes[i]))
	}

	return strings.Join(parts, "-")
}

// formatRange: S01E01-03 (only if episodes are consecutive)
func formatRange(season int, episodes []int, padding int) string {
	if !isConsecutive(episodes) {
		// Fall back to extend style for non-consecutive episodes
		return formatExtend(season, episodes, padding)
	}

	first := episodes[0]
	last := episodes[len(episodes)-1]

	return fmt.Sprintf("S%02dE%0*d-%0*d", season, padding, first, padding, last)
}

// formatPrefixedRange: S01E01-E03 (only if episodes are consecutive)
func formatPrefixedRange(season int, episodes []int, padding int) string {
	if !isConsecutive(episodes) {
		// Fall back to scene style for non-consecutive episodes
		return formatScene(season, episodes, padding)
	}

	first := episodes[0]
	last := episodes[len(episodes)-1]

	return fmt.Sprintf("S%02dE%0*d-E%0*d", season, padding, first, padding, last)
}

// isConsecutive checks if all episodes are consecutive numbers.
func isConsecutive(episodes []int) bool {
	if len(episodes) < 2 {
		return true
	}

	for i := 1; i < len(episodes); i++ {
		if episodes[i] != episodes[i-1]+1 {
			return false
		}
	}

	return true
}

// FormatMultiEpisodeTitles concatenates multiple episode titles.
func FormatMultiEpisodeTitles(titles []string, separator string) string {
	if len(titles) == 0 {
		return ""
	}

	if separator == "" {
		separator = " + "
	}

	// Filter out empty titles
	nonEmpty := make([]string, 0, len(titles))
	for _, t := range titles {
		if t = strings.TrimSpace(t); t != "" {
			nonEmpty = append(nonEmpty, t)
		}
	}

	if len(nonEmpty) == 0 {
		return ""
	}

	return strings.Join(nonEmpty, separator)
}

// ParseMultiEpisodeStyle converts a string to MultiEpisodeStyle.
func ParseMultiEpisodeStyle(s string) MultiEpisodeStyle {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "extend":
		return StyleExtend
	case "duplicate":
		return StyleDuplicate
	case "repeat":
		return StyleRepeat
	case "scene":
		return StyleScene
	case "range":
		return StyleRange
	case "prefixed_range", "prefixedrange":
		return StylePrefixedRange
	default:
		return StyleExtend
	}
}

// GetMultiEpisodeStyleOptions returns all available multi-episode styles.
func GetMultiEpisodeStyleOptions() []struct {
	Value   MultiEpisodeStyle
	Label   string
	Example string
} {
	return []struct {
		Value   MultiEpisodeStyle
		Label   string
		Example string
	}{
		{StyleExtend, "Extend", "S01E01-02-03"},
		{StyleDuplicate, "Duplicate", "S01E01.S01E02"},
		{StyleRepeat, "Repeat", "S01E01E02E03"},
		{StyleScene, "Scene", "S01E01-E02-E03"},
		{StyleRange, "Range", "S01E01-03"},
		{StylePrefixedRange, "Prefixed Range", "S01E01-E03"},
	}
}
