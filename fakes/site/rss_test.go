package site_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

func searchingNode(f faketest.Served) searching.Remote {
	timings := searching.DefaultTimings()
	timings.Floor = 0
	return searching.RemoteOf(searching.New(searching.Config{HTTP: f.Client, Timings: timings, Log: slog.New(slog.DiscardHandler)}))
}

// rssJob runs call on Brook's feed at address, through n.
func rssJob(t *testing.T, n searching.Remote, address string, session *searching.Session, reading searching.RSSReading, call searching.Call) searching.Answer {
	t.Helper()
	poll, err := n.Run(t.Context(), searching.Job{
		ID: storage.NewUUID().String(), Indexer: "brook", Kind: searching.RSS, Address: address, Session: session, RSS: reading, Call: call,
	})
	if err != nil || poll.Answer == nil {
		t.Fatalf("Run = %+v, %v; want an answer within the hold", poll, err)
	}
	return *poll.Answer
}

func stated(a searching.Answer) string {
	if a.Outcome != nil {
		return fmt.Sprintf("%T %v", a.Outcome, a.Outcome)
	}
	if r := a.Report; r != nil {
		if r.Refusal != "" {
			return "refused " + string(r.Refusal)
		}
		return fmt.Sprintf("%d items, %+v", r.Items, *r.RSS)
	}
	return fmt.Sprintf("%d Releases", len(a.Releases))
}

func TestEachRSSPlayLandsInItsTest(t *testing.T) {
	f := faketest.Loopback(t, spec(listing), scenarioWithSeed())
	n := searchingNode(f)
	for _, tc := range []struct {
		play        string
		allowNoSize bool
		want        string
	}{
		{"", false, "5 items, {Link:enclosure SizeFromDescription:false SeedersFromDescription:true AllowNoSize:false}"},
		{site.PlayNoItems, false, "refused no_items"},
		{site.PlayNoLink, false, "refused no_link"},
		{site.PlayRepeatedGUID, false, "refused repeated_guid"},
		{site.PlayNoSize, false, "refused no_size"},
		{site.PlayNoSize, true, "5 items, {Link:enclosure SizeFromDescription:false SeedersFromDescription:true AllowNoSize:true}"},
	} {
		got := stated(rssJob(t, n, site.RSSURL(f.Address, tc.play), nil, searching.RSSReading{AllowNoSize: tc.allowNoSize}, searching.TestCall{}))
		if got != tc.want {
			t.Errorf("a Test of the %q feed, allowing no size %t = %s, want %s", tc.play, tc.allowNoSize, got, tc.want)
		}
	}

	stale := &searching.Session{Cookies: []searching.Cookie{{Name: site.CookieName, Value: "stale", Domain: site.Host(site.Brook), HostOnly: true}}, Pasted: true, Earned: time.Now()}
	if got, want := stated(rssJob(t, n, site.RSSURL(f.Address, ""), stale, searching.RSSReading{}, searching.TestCall{})), "*searching.Stopped the site's cookie expired"; got != want {
		t.Errorf("a Test with a rejected cookie = %s, want %s", got, want)
	}

	for _, tc := range []struct {
		fault fault.Fault
		want  string
	}{
		{fault.Malformed{}, "*searching.Failing read the feed: failed to detect feed type"},
		{fault.Status{Code: 503}, "*searching.Failing the Indexer responded with status 503"},
	} {
		if _, err := f.Faults.Set(fault.Rule{ID: "feed", Target: fault.Target(site.Fake.Name), Route: "rss", Fault: tc.fault}); err != nil {
			t.Fatal(err)
		}
		other := searchingNode(f)
		if got := stated(rssJob(t, other, site.RSSURL(f.Address, ""), nil, searching.RSSReading{}, searching.TestCall{})); got != tc.want {
			t.Errorf("a Test of a feed playing %v = %s, want %s", tc.fault, got, tc.want)
		}
		f.Faults.Clear("feed")
	}
}

func TestBrooksRecentItemsAreReleasesAndFetch(t *testing.T) {
	f := faketest.Loopback(t, spec(listing), scenarioWithSeed())
	n := searchingNode(f)
	reading := searching.RSSReading{Link: "enclosure", SeedersFromDescription: true}
	a := rssJob(t, n, site.RSSURL(f.Address, ""), nil, reading, searching.RecentCall{})
	if a.Outcome != nil || len(a.Releases) != len(listing) {
		t.Fatalf("Recent = %s, want every listed release", stated(a))
	}
	var got []string
	for _, r := range a.Releases {
		got = append(got, fmt.Sprintf("%s %d %d %s %t %t %s", r.Title, r.Size, r.Seeders, r.InfoHash, r.Link.IsZero(), r.Magnet.IsZero(), r.Published.Format(time.RFC3339)))
	}
	var want []string
	for _, rel := range listing {
		magnet := rel.Link == scenario.MagnetLink
		want = append(want, fmt.Sprintf("%s %d %d %s %t %t 2026-06-01T12:00:00Z", rel.Name, rel.Size, rel.Seeders, rel.InfoHash, magnet, !magnet))
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Recent =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	fetched := rssJob(t, n, site.RSSURL(f.Address, ""), nil, reading, searching.FetchCall{Link: a.Releases[0].Link})
	if got, ok := fetched.Got.(searching.Torrent); fetched.Outcome != nil || !ok || string(got.Raw) != rewritten {
		t.Fatalf("Fetch = %+v, %v; want the .torrent %q", fetched.Got, fetched.Outcome, rewritten)
	}
}

// controller is an Indexers on one Node whose Searching service reaches f,
// carrying movies.
func controller(t *testing.T, f faketest.Served) *indexers.Indexers {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), indexers.Schema())
	self := role.NodeID{UUID: storage.NewUUID()}
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, self, "home")
	}); err != nil {
		t.Fatal(err)
	}
	n := searchingNode(f)
	return indexers.New(indexers.Config{
		Library:   lib,
		Sealer:    secrettest.Sealer(t),
		Roster:    indexerstest.Alone(self),
		Searching: func(role.NodeID) searching.Remote { return n },
		Changed:   discard[indexers.Changed]{},
		Status:    discard[indexers.StatusChanged]{},
		Trackers:  discard[indexers.TrackerChanged]{},
		Timings:   indexers.DefaultTimings(),
		Log:       slog.New(slog.DiscardHandler),
		Types:     map[works.TypeID]indexers.Facet{"movie": indexerstest.Facet{}},
	})
}

func TestAnRSSFeedIsAddedWithItsCookieAndReading(t *testing.T) {
	s := scenarioWithSeed()
	s.Releases = []scenario.Release{{Name: "Harbor.Lights", Unlisted: true}}
	catalog := &releases.Catalog{Passkey: "p4ss", Releases: []releases.Release{
		{Release: scenario.Release{Name: "Quiet.Orchard"}, Size: 6000, InfoHash: bencode.InfoHash{1}, Metainfo: []byte(torrent)},
		{Release: scenario.Release{Name: "Salt.Meadow"}, Size: 5000, InfoHash: bencode.InfoHash{2}, Metainfo: []byte(torrent)},
		{Release: scenario.Release{Name: "Harbor.Lights"}, Size: 3000, InfoHash: bencode.InfoHash{3}, Metainfo: []byte(torrent)},
	}}
	f := faketest.Loopback(t, site.Fake, s, faketest.WithReleases(catalog), faketest.WithFronts(nil))
	x := controller(t, f)
	brook := indexers.Draft{
		Kind: indexers.RSS, Name: "Brook", Address: site.RSSURL(f.Address, ""), Tracker: "Brook", Media: []works.TypeID{"movie"},
		Cookie: &indexers.PastedCookie{Cookie: site.Cookie(s), UserAgent: "Pasted/1.0"}, On: true, RSS: true,
	}

	for _, tc := range []struct {
		edit func(*indexers.Draft)
		want string
	}{
		{func(d *indexers.Draft) { d.Media = nil }, indexers.ErrMedia.Error()},
		{func(d *indexers.Draft) { d.Media = []works.TypeID{"music"} }, indexers.ErrMedia.Error()},
		{func(d *indexers.Draft) { d.Cookie = &indexers.PastedCookie{Cookie: "not a cookie"} }, indexers.ErrCookie.Error()},
		{func(d *indexers.Draft) { d.Cookie.Cookie = site.CookieName + "=stale" }, "the Indexer's Test ended stopped"},
		{func(d *indexers.Draft) { d.Address = site.RSSURL(f.Address, site.PlayRepeatedGUID) }, "the feed can't be used: some items repeat the same GUID"},
	} {
		d := brook
		cookie := *brook.Cookie
		d.Cookie = &cookie
		tc.edit(&d)
		if _, err := x.Create(t.Context(), identity.Admin{}, d); err == nil || err.Error() != tc.want {
			t.Errorf("Create = %v, want %s", err, tc.want)
		}
	}
	d := brook
	d.Address = site.RSSURL(f.Address, site.PlayNoSize)
	_, err := x.Create(t.Context(), identity.Admin{}, d)
	var refused *indexers.FeedRefusedError
	if want := "Some items have no size. Turn on Allow items with no size to use the feed anyway."; !errors.As(err, &refused) || refused.Sentence != want {
		t.Fatalf("Create of a feed with no sizes = %v, want %q", err, want)
	}

	v, err := x.Create(t.Context(), identity.Admin{}, brook)
	if err != nil {
		t.Fatal(err)
	}
	feed, _ := v.Source.(indexers.Feed)
	if v.Kind != indexers.RSS || v.FeedHost != site.Host(site.Brook) || feed.Tracker != "Brook" || feed.Reading.Link != "enclosure" ||
		!feed.Reading.SeedersFromDescription || v.Session == nil || !v.Session.Pasted || v.Status.Success.IsZero() {
		t.Fatalf("the created feed = %+v, want an RSS feed on Brook's host read by its enclosures, its pasted session held and its Test passed", v)
	}

	before := len(rssJob(t, searchingNode(f), brook.Address, nil, feed.Reading, searching.RecentCall{}).Releases)
	f.Cue(t, site.RSSItem.Name, site.ListArgs{Release: "Harbor.Lights"})
	after := rssJob(t, searchingNode(f), brook.Address, nil, feed.Reading, searching.RecentCall{}).Releases
	if before != 2 || len(after) != 3 || after[2].Title != "Harbor.Lights" {
		t.Fatalf("the feed lists %d items before the cue and %+v after, want Harbor.Lights to appear", before, after)
	}
}
