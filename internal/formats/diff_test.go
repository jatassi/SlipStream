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
		return release.Read{Release: "Film.x264", Facts: release.Facts{Resolution: release.Known(res)}}
	}
	e := formats.Diff([]formats.Versions{{Before: &before, After: &after}}, limit)
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
			if got := e.Match(tc.read)[0]; got.Before != tc.want.Before || got.After != tc.want.After {
				t.Errorf("Match = %+v, want %+v", got, tc.want)
			}
		})
	}
	if got := formats.Diff([]formats.Versions{{Before: &before}}, limit).Match(at(release.R1080p))[0]; !got.Before || got.After {
		t.Errorf("a deleted format: %+v", got)
	}
}

func TestDiffReportsTheAfterVersionsTimeouts(t *testing.T) {
	short := formats.Timings{Match: 10 * time.Millisecond}
	before := format(1, formats.Radarr, title(slow))
	after := format(1, formats.Radarr, title(`x`), title(slow))
	got := formats.Diff([]formats.Versions{{Before: &before, After: &after}}, short).Match(aaa)[0]
	if got.Before || got.After || !slices.Equal(got.Timeouts, []formats.Timeout{{Format: after.ID, Condition: 1}}) {
		t.Errorf("Match = %+v", got)
	}
}

func TestAnEditReadsOnlyTheNameWhenBothVersionsTestOnlyTheTitle(t *testing.T) {
	titleOnly := format(1, formats.Radarr, title(`x264`))
	byGroup := format(1, formats.Radarr, title(`x264`), cond(formats.ReleaseGroup, map[string]any{"value": "^GRP$"}))
	for _, c := range []struct {
		name          string
		before, after *formats.Format
		want          bool
	}{
		{"title conditions on both sides", &titleOnly, &titleOnly, true},
		{"a deleted title-only format", &titleOnly, nil, true},
		{"a group condition added", &titleOnly, &byGroup, false},
		{"a group condition removed", &byGroup, &titleOnly, false},
	} {
		if got := formats.Diff([]formats.Versions{{Before: c.before, After: c.after}}, limit).NameOnly(); got != c.want {
			t.Errorf("%s: NameOnly = %t, want %t", c.name, got, c.want)
		}
	}
}

func TestDiffFlipsEachFormatOfOneChangeOnItsOwn(t *testing.T) {
	hdr := format(1, formats.Radarr, title(`\bHDR\b`))
	dv := format(2, formats.Radarr, title(`\bDV\b`))
	widened := format(2, formats.Radarr, title(`\b(DV|HDR)\b`))
	e := formats.Diff([]formats.Versions{{Before: &hdr}, {Before: &dv, After: &widened}, {After: &hdr}}, limit)
	got := e.Match(release.Read{Release: "Film.2160p.HDR"})
	want := []formats.Flip{{Before: true}, {After: true}, {After: true}}
	if len(got) != len(want) {
		t.Fatalf("Match = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i].Before != want[i].Before || got[i].After != want[i].After {
			t.Errorf("format %d: Match = %+v, want %+v", i, got[i], want[i])
		}
	}
}
