package subtitling

import (
	"github.com/jatassi/slipstream/internal/subtitles/cues"
	"github.com/jatassi/slipstream/internal/subtitles/timing"
)

// Spans are d's cue times as Timing sync reads them.
func Spans(d cues.Doc) []timing.Span {
	ts := d.Times()
	out := make([]timing.Span, len(ts))
	for i, t := range ts {
		out[i] = timing.Span{Start: t.Start, End: t.End, Skip: t.Skip}
	}
	return out
}

func times(spans []timing.Span) []cues.Times {
	out := make([]cues.Times, len(spans))
	for i, s := range spans {
		out[i] = cues.Times{Start: s.Start, End: s.End, Skip: s.Skip}
	}
	return out
}
