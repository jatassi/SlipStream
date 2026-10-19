package node

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/node/internal/config"
)

type trackerEndpoints struct {
	Announce endpoint.URL        `yaml:"announce"`
	STUN     []endpoint.HostPort `yaml:"stun"`
}

type censusTree struct {
	Seed      string `yaml:"seed"`
	Endpoints struct {
		Tracker trackerEndpoints `yaml:"tracker"`
	} `yaml:"endpoints"`
}

func censusDefaults() censusTree {
	var t censusTree
	t.Endpoints.Tracker = trackerEndpoints{
		Announce: endpoint.MustParseURL("https://tracker.example.org/announce"),
		STUN:     []endpoint.HostPort{endpoint.MustParseHostPort("stun.example.net:3478"), endpoint.MustParseHostPort("stun2.example.net:3478")},
	}
	return t
}

func TestCensusListsEndpointsStillAtADefaultHost(t *testing.T) {
	defaults := censusDefaults()
	for _, c := range []struct {
		environ []string
		want    []string
	}{
		{nil, []string{"endpoints.tracker.announce", "endpoints.tracker.stun"}},
		{
			[]string{"SLIPSTREAM_ENDPOINTS_TRACKER_ANNOUNCE=http://127.0.0.1:9000/announce", "SLIPSTREAM_ENDPOINTS_TRACKER_STUN=127.0.0.1:3478,stun2.example.net:3478"},
			[]string{"endpoints.tracker.stun"},
		},
		{
			[]string{"SLIPSTREAM_ENDPOINTS_TRACKER_ANNOUNCE=https://tracker.example.org:8443/fake", "SLIPSTREAM_ENDPOINTS_TRACKER_STUN=127.0.0.1:3478"},
			[]string{"endpoints.tracker.announce"},
		},
	} {
		tree := censusDefaults()
		if _, err := config.Load(&tree, t.TempDir(), c.environ); err != nil {
			t.Fatal(err)
		}
		if got := census(&tree, &defaults); !slices.Equal(got, c.want) {
			t.Errorf("census with %v = %v, want %v", c.environ, got, c.want)
		}
	}
}

func TestCensusOfTheDefaultsListsTheCoreEndpoints(t *testing.T) {
	if got, want := Census(Defaults(noTypes, install.Dev), noTypes, install.Dev), []string{"endpoints.tmdb.api", "endpoints.tmdb.images", "endpoints.omdb.api"}; !slices.Equal(got, want) {
		t.Fatalf("census = %#v, want %#v", got, want)
	}
}
