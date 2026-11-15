package grabs

import (
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

func TestEveryTriggerHasARule(t *testing.T) {
	for _, tr := range searches.Triggers() {
		if _, ok := laneRules[tr]; !ok {
			t.Errorf("trigger %q has no lane rule", tr)
		}
	}
	if len(laneRules) != len(searches.Triggers()) {
		t.Errorf("laneRules has %d rows for %d triggers", len(laneRules), len(searches.Triggers()))
	}
}

func TestATriedReleaseIsKnownByItsInfoHashElseByItsIndexerAndGUID(t *testing.T) {
	site, other := indexers.ID{UUID: storage.NewUUID()}, indexers.ID{UUID: storage.NewUUID()}
	a, b := bencode.InfoHash([]byte("aaaaaaaaaaaaaaaaaaaa")), bencode.InfoHash([]byte("bbbbbbbbbbbbbbbbbbbb"))
	listed := func(on indexers.ID, hash *bencode.InfoHash, guid string) indexers.Release {
		return indexers.Release{Indexer: on, Release: searching.Release{InfoHash: hash, GUID: guid}}
	}
	for _, c := range []struct {
		name  string
		tried triedRelease
		r     indexers.Release
		want  bool
	}{
		{"the same info hash on another Indexer", triedRelease{hash: &a, indexer: site, guid: "g"}, listed(other, &a, "h"), true},
		{"another info hash with the same GUID", triedRelease{hash: &a, indexer: site, guid: "g"}, listed(site, &b, "g"), false},
		{"no info hash, the same Indexer and GUID", triedRelease{indexer: site, guid: "g"}, listed(site, &b, "g"), true},
		{"no info hash, the same GUID on another Indexer", triedRelease{indexer: site, guid: "g"}, listed(other, nil, "g"), false},
		{"no info hash and no GUID", triedRelease{indexer: site}, listed(site, nil, ""), false},
	} {
		if got := c.tried.is(c.r); got != c.want {
			t.Errorf("%s: is = %v, want %v", c.name, got, c.want)
		}
	}
}
