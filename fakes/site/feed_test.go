package site_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/searching"
)

func listed(a searching.Answer) []string {
	out := make([]string, len(a.Releases))
	for i, r := range a.Releases {
		out[i] = r.Title
	}
	return out
}

func TestTheNewestUploadsListNewestFirstAndPlayEachFeedCue(t *testing.T) {
	s := scenarioWithSeed()
	posted := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	catalog := &releases.Catalog{}
	for i := range 3 {
		catalog.Releases = append(catalog.Releases, releases.Release{
			Release: scenario.Release{Name: fmt.Sprintf("Orchard.%d", i), Published: posted.Add(time.Duration(i) * time.Hour)}, Size: 1000,
			InfoHash: bencode.InfoHash{byte(i + 1)}, Metainfo: []byte(torrent),
		})
	}
	f := faketest.Loopback(t, site.Fake, s, faketest.WithReleases(catalog), faketest.WithFronts(nil))
	key := site.APIKey(s)
	recent := func(offset int) searching.Answer {
		t.Helper()
		a := job(t, f, key, searching.RecentCall{Offset: offset})
		if a.Outcome != nil {
			t.Fatalf("a recent call at offset %d = %v", offset, a.Outcome)
		}
		return a
	}
	if got := listed(recent(0)); !slices.Equal(got, []string{"Orchard.2", "Orchard.1", "Orchard.0"}) {
		t.Fatalf("the newest uploads = %v, want newest first", got)
	}
	if got := listed(recent(2)); !slices.Equal(got, []string{"Orchard.0"}) {
		t.Fatalf("the newest uploads from offset 2 = %v, want the oldest", got)
	}

	f.Cue(t, site.Feed.Name, site.FeedArgs{Uploads: 2, NoPaging: true})
	first := listed(recent(0))
	if len(first) != 5 || !strings.HasPrefix(first[0], "Fixture.Upload.") || first[2] != "Orchard.2" {
		t.Fatalf("two uploads above the world's = %v", first)
	}
	if got := listed(recent(3)); !slices.Equal(got, first) {
		t.Fatalf("a feed that doesn't page lists %v at offset 3, want the newest page %v", got, first)
	}

	f.Cue(t, site.Feed.Name, site.FeedArgs{FutureDates: true, LongGUIDs: true})
	a := recent(0)
	if r := a.Releases[0]; !r.Published.After(time.Now()) || len(r.GUID) <= 512 {
		t.Fatalf("an item published %s with a %d-byte GUID, want a future date and a GUID over 512 bytes", r.Published, len(r.GUID))
	}

	f.Cue(t, site.Feed.Name, site.FeedArgs{OversizedPage: true})
	if n := len(recent(0).Releases); n <= 1000 {
		t.Fatalf("an oversized page lists %d items, want more than 1,000", n)
	}
}
