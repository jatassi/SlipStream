package subtitles

import "github.com/jatassi/slipstream/internal/history/entry"

// The kinds Subtitles records in History's Subtitles group, each in the
// Write that makes it true.
const (
	SubtitlesAdded       entry.Kind = "subtitles_added"
	SubtitleReplaced     entry.Kind = "subtitle_replaced"
	SubtitlesRemoved     entry.Kind = "subtitles_removed"
	SubtitleMarkedBad    entry.Kind = "subtitle_marked_bad"
	SubtitlesMissing     entry.Kind = "subtitles_missing"
	SubtitleTimingFailed entry.Kind = "subtitle_timing_failed"
	SubtitleProviderOut  entry.Kind = "subtitle_provider_out"
)

// Entries is the rows Subtitles records. Its Sidecar writes and its
// searches add them.
func Entries() []entry.Entry { return nil }
