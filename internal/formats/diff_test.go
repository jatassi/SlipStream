package formats_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/release"
)

func TestDiff(t *testing.T) {
	before := format(1, formats.Radarr, title(`x264`), cond(formats.Resolution, map[string]any{"value": 1080}))
	after := format(1, formats.Radarr, title(`x264`), cond(formats.Resolution, map[string]any{"value": 2160}))
	at := func(res release.Resolution) release.Read {
		return release.Read{Given: "Film.x264", Facts: release.Facts{Resolution: release.Known(res)}}
	}
	e := formats.Diff(&before, &after, limit)
	cases := []struct {
		name string
		read release.Read
		want formats.Flip
	}{
		{"matched before only", at(release.R1080p), formats.Flip{Before: true}},
		{"matched after only", at(release.R2160p), formats.Flip{After: true}},
		{"matched neither", at(release.R720p), formats.Flip{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := e.Match(tc.read); got.Before != tc.want.Before || got.After != tc.want.After {
				t.Errorf("Match = %+v, want %+v", got, tc.want)
			}
		})
	}
	if got := formats.Diff(&before, nil, limit).Match(at(release.R1080p)); !got.Before || got.After {
		t.Errorf("a deleted format: %+v", got)
	}
}

func TestDiffReportsTheAfterVersionsTimeouts(t *testing.T) {
	short := formats.Timings{Match: 10 * time.Millisecond}
	before := format(1, formats.Radarr, title(slow))
	after := format(1, formats.Radarr, title(`x`), title(slow))
	got := formats.Diff(&before, &after, short).Match(aaa)
	if got.Before || got.After || !slices.Equal(got.Timeouts, []formats.Timeout{{Format: after.ID, Condition: 1}}) {
		t.Errorf("Match = %+v", got)
	}
}
