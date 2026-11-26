package e2e

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type playbackOutcome struct {
	Kind    string `json:"kind"`
	Version *struct {
		SeparateItem bool `json:"separateItem"`
	} `json:"version"`
	Suggestion *struct {
		Name string `json:"name"`
	} `json:"suggestion"`
	Target *struct {
		Name string `json:"name"`
	} `json:"target"`
	Works []struct {
		State string `json:"state"`
		Fact  *struct {
			Fact string `json:"fact"`
		} `json:"fact"`
	} `json:"works"`
}

type playbackReport struct {
	Plex      bool `json:"plex"`
	Unmatched int  `json:"unmatched"`
	Players   []struct {
		Key       string `json:"key"`
		Unmatched int    `json:"unmatched"`
		Groups    []struct {
			Dimensions []struct {
				Dimension string `json:"dimension"`
			} `json:"dimensions"`
			Outcomes []playbackOutcome `json:"outcomes"`
		} `json:"groups"`
	} `json:"players"`
}

// TestThePlaybackWorldPlaysEveryOutcomeAtLaunch launches the playback
// Scenario and reads each server's Playback report: every outcome kind,
// each Work state of an existing Target, the plays no root covers and
// Plex's play with no reason.
func TestThePlaybackWorldPlaysEveryOutcomeAtLaunch(t *testing.T) {
	run := worldtest.LaunchE2E(t, "playback")
	a := api(t, run)
	var servers struct {
		Servers []idName `json:"servers"`
	}
	decode(t, "listMediaServers", a.Admin("home", http.MethodGet, "/api/v1/media-servers", ""), &servers)
	want := map[string][]string{
		"Living room Jellyfin": {
			"bedroom-tv: hdr suggest 1080p HDR10",
			"den-tv: hdr existing 1080p SDR AAC [no_file not_in_profile not_listed not_proven(bitrate)]",
			"hallway-stick: subtitles no_target",
			"kitchen-ipad: video_codec unproven",
			"kitchen-ipad: 1 unmatched",
			"living-room-tv: hdr version in the item",
			"report: 1 unmatched",
		},
		"Den Emby":    {"lounge-tv: hdr version as its own item"},
		"Office Plex": {"office-pc: unknown no_target", "report: plex"},
	}
	for _, srv := range servers.Servers {
		t.Run(srv.Name, func(t *testing.T) {
			var r playbackReport
			decode(t, "getMediaServerPlayback", a.Admin("home", http.MethodGet, "/api/v1/media-servers/"+srv.ID+"/playback", ""), &r)
			slices.Sort(want[srv.Name])
			if got := linesOf(r); !slices.Equal(got, want[srv.Name]) {
				t.Errorf("the report reads\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want[srv.Name], "\n"))
			}
		})
	}
}

// linesOf is one line per outcome of r, each naming its player, its
// group's dimensions and what it names, sorted.
func linesOf(r playbackReport) []string {
	var out []string
	if r.Plex {
		out = append(out, "report: plex")
	}
	if r.Unmatched > 0 {
		out = append(out, fmt.Sprintf("report: %d unmatched", r.Unmatched))
	}
	for _, p := range r.Players {
		if p.Unmatched > 0 {
			out = append(out, fmt.Sprintf("%s: %d unmatched", p.Key, p.Unmatched))
		}
		for _, g := range p.Groups {
			var dims []string
			for _, d := range g.Dimensions {
				dims = append(dims, d.Dimension)
			}
			for _, o := range g.Outcomes {
				out = append(out, p.Key+": "+strings.Join(dims, " ")+" "+o.Kind+namedBy(o))
			}
		}
	}
	slices.Sort(out)
	return out
}

// namedBy is what o names: where its version is, its suggestion's name,
// or its Target and each Work's state, sorted.
func namedBy(o playbackOutcome) string {
	switch {
	case o.Version != nil && o.Version.SeparateItem:
		return " as its own item"
	case o.Version != nil:
		return " in the item"
	case o.Suggestion != nil:
		return " " + o.Suggestion.Name
	case o.Target != nil:
		var states []string
		for _, w := range o.Works {
			state := w.State
			if w.Fact != nil {
				state += "(" + w.Fact.Fact + ")"
			}
			states = append(states, state)
		}
		slices.Sort(states)
		return fmt.Sprintf(" %s %v", o.Target.Name, states)
	}
	return ""
}
