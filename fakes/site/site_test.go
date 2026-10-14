package site_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/pb/slipstream/search/v1/searchv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

const (
	hash1080 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hash720  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	torrent  = "d4:infod6:lengthi6e4:name5:a.mkv12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaaee"
)

var releases = []site.Release{
	{Name: "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW", Size: 6000, InfoHash: hash1080, TMDBID: 9000001, IMDbID: "tt9000001", Seeders: 12, Torrent: []byte(torrent)},
	{Name: "The.Quiet.Orchard.2021.720p.WEB-DL.H264-FKW", Size: 3000, InfoHash: hash720, TMDBID: 9000001, IMDbID: "tt9000001", Seeders: 40, Torrent: []byte("d4:infod4:name1:bee")},
	{Name: "Orchard.Road.2019.1080p.WEB-DL.H264-FKW", Size: 5000, InfoHash: strings.Repeat("c", 40), TMDBID: 9000002, IMDbID: "tt9000002", Seeders: 1, Torrent: []byte("d4:infod4:name1:cee")},
}

func scenarioWithSeed() scenario.Scenario { return scenario.Scenario{Name: "x", Seed: "site-test"} }

func spec(rels []site.Release) fake.Spec {
	return fake.Spec{Name: site.Fake.Name, TLS: true, New: func(s scenario.Scenario, _ fake.Env) (fake.Server, error) {
		return fake.HTTP(site.Handler(site.APIKey(s), rels)), nil
	}}
}

func get(t *testing.T, f faketest.Served, path string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL(path), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.Status + " " + string(body)
}

func item(origin string, r site.Release) string {
	link := origin + "/download/" + r.InfoHash + ".torrent"
	return `    <item>
      <title>` + r.Name + `</title>
      <guid>` + link + `</guid>
      <size>` + itoa(r.Size) + `</size>
      <link>` + link + `</link>
      <enclosure url="` + link + `" length="` + itoa(r.Size) + `" type="application/x-bittorrent" />
      <torznab:attr name="seeders" value="` + itoa(r.Seeders) + `" />
      <torznab:attr name="infohash" value="` + r.InfoHash + `" />
      <torznab:attr name="tmdbid" value="` + itoa(r.TMDBID) + `" />
      <torznab:attr name="imdbid" value="` + r.IMDbID + `" />
    </item>
`
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func feed(items ...string) string {
	return `200 OK <?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed">
  <channel>
    <title>fixture-site</title>
` + strings.Join(items, "") + "  </channel>\n</rss>\n"
}

func errorDoc(code, description string) string {
	return `200 OK <?xml version="1.0" encoding="UTF-8"?>
<error code="` + code + `" description="` + description + `" />
`
}

func TestTheSiteAnswersAsATorznabIndexerDoes(t *testing.T) {
	s := scenarioWithSeed()
	key := site.APIKey(s)
	check := func(t *testing.T, f faketest.Served) {
		origin := f.Address.URL("")
		api := site.TorznabPath + "/api?apikey=" + key
		for _, e := range []struct{ path, want string }{
			{api + "&t=movie&tmdbid=9000001", feed(item(origin, releases[0]), item(origin, releases[1]))},
			{api + "&t=movie&imdbid=tt9000002", feed(item(origin, releases[2]))},
			{api + "&t=movie&tmdbid=9000003", feed()},
			{api + "&t=search&q=quiet+ORCHARD+2021", feed(item(origin, releases[0]), item(origin, releases[1]))},
			{api + "&t=search&q=orchard+1080p", feed(item(origin, releases[0]), item(origin, releases[2]))},
			{api + "&t=search&q=orch", feed()},
			{api + "&t=tvsearch&q=orchard", errorDoc("202", "No such function")},
			{site.TorznabPath + "/api?apikey=wrong&t=caps", errorDoc("100", "Invalid API Key")},
			{site.TorznabPath + "/api?t=caps", errorDoc("100", "Invalid API Key")},
			{"/download/" + hash1080 + ".torrent", "200 OK " + torrent},
			{"/download/" + strings.Repeat("d", 40) + ".torrent", "404 Not Found 404 page not found\n"},
		} {
			if got := get(t, f, e.path); got != e.want {
				t.Errorf("GET %s =\n%s\nwant\n%s", e.path, got, e.want)
			}
		}
	}
	t.Run("loopback", func(t *testing.T) { check(t, faketest.Loopback(t, spec(releases), s)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { check(t, faketest.InMemory(t, spec(releases), s)) })
	})
}

func TestTheAPIKeyFollowsTheSeed(t *testing.T) {
	a, b := site.APIKey(scenario.Scenario{Seed: "one"}), site.APIKey(scenario.Scenario{Seed: "two"})
	if a == b || len(a) != 32 || a != site.APIKey(scenario.Scenario{Seed: "one"}) {
		t.Fatalf("keys %s and %s, want 32 hex digits that differ by seed and repeat for one", a, b)
	}
}

type alone struct{ self role.NodeID }

func (a alone) Self() role.NodeID { return a.self }

func (a alone) HoldersByPairing(context.Context, role.Name) ([]role.NodeID, error) {
	return []role.NodeID{a.self}, nil
}

type discard struct{}

func (discard) Publish(context.Context, indexers.Changed) {}

func TestTheSearchingServiceReadsTheSite(t *testing.T) {
	s := scenarioWithSeed()
	f := faketest.Loopback(t, spec(releases), s)
	home := indexers.NewHome(f.Client, indexers.Timings{Request: 5 * time.Second})
	x := indexers.New(indexers.Config{
		Library:   storagetest.Library(t, indexers.Schema()),
		Roster:    alone{role.NodeID{UUID: storage.NewUUID()}},
		Searching: func(role.NodeID) searchv1connect.SearchingServiceHandler { return home },
		Changed:   discard{},
	})
	idx, err := x.Create(t.Context(), identity.Admin{}, indexers.Draft{Name: "Fixture Site", URL: site.URL(f.Address), APIKey: site.APIKey(s)})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := idx.Caps.Modes["movie"], []string{"q", "tmdbid", "imdbid"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("movie caps = %v, want %v", got, want)
	}

	items, err := x.Search(t.Context(), idx, indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"}, Query: "The Quiet Orchard 2021"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range items {
		got = append(got, it.Title+" "+it.InfoHash+" "+itoa(it.Seeders)+" "+it.IDs["tmdbid"]+" "+it.IDs["imdbid"])
	}
	want := []string{
		"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW " + hash1080 + " 12 9000001 tt9000001",
		"The.Quiet.Orchard.2021.720p.WEB-DL.H264-FKW " + hash720 + " 40 9000001 tt9000001",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("items =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	b, err := x.FetchTorrent(t.Context(), idx, items[0].Link)
	if err != nil || string(b) != torrent {
		t.Fatalf("FetchTorrent = %q, %v, want %q", b, err, torrent)
	}
	_, err = x.FetchTorrent(t.Context(), idx, "https://elsewhere.test/download/"+hash1080+".torrent")
	var unavailable *indexers.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Wait || err.Error() != "the Indexer is unavailable: invalid_argument: the torrent's URL is not on the Indexer's host" {
		t.Fatalf("an off-host FetchTorrent = %v, want an unavailable error that does not wait", err)
	}
	_, err = x.FetchTorrent(t.Context(), idx, f.Address.URL("/download/"+strings.Repeat("d", 40)+".torrent"))
	if err == nil || err.Error() != "the Indexer is unavailable: aborted: the Indexer answered status 404" {
		t.Fatalf("a missing torrent = %v", err)
	}

	idx.APIKey = "wrong"
	_, err = x.Search(t.Context(), idx, indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": "9000001"}})
	var refused *indexers.RefusedError
	if !errors.As(err, &refused) || err.Error() != "the Indexer refused: error 100: Invalid API Key" {
		t.Fatalf("a wrong key = %v, want refused 100", err)
	}
}
