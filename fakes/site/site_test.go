package site_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
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
)

const (
	hash1080 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hash720  = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	info     = "d6:lengthi6e4:name5:a.mkv12:piece lengthi262144e6:pieces20:aaaaaaaaaaaaaaaaaaaae"
	torrent  = "d4:info" + info + "e"
	announce = "https://127.0.0.1:7000/announce/k"
	// rewritten is torrent as the site serves it, announcing to announce.
	rewritten = "d8:announce33:" + announce + "4:info" + info + "e"
)

var listing = []site.Release{
	{Name: "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW", Size: 6000, InfoHash: hash1080, TMDBID: 9000001, IMDbID: "tt9000001", Seeders: 12, Torrent: []byte(torrent)},
	{Name: "The.Quiet.Orchard.2021.720p.WEB-DL.H264-FKW", Size: 3000, InfoHash: hash720, TMDBID: 9000001, IMDbID: "tt9000001", Seeders: 40, Torrent: []byte("d4:infod4:name1:bee")},
	{Name: "Orchard.Road.2019.1080p.WEB-DL.H264-FKW", Size: 5000, InfoHash: strings.Repeat("c", 40), TMDBID: 9000002, IMDbID: "tt9000002", Seeders: 1, Torrent: []byte("d4:infod4:name1:cee")},
	{Name: "Salt.Meadow.2020.1080p.WEB-DL.H264-FKW", Size: 5000, InfoHash: strings.Repeat("e", 40), TMDBID: 9000004, IMDbID: "tt9000004", Seeders: 3, Link: scenario.MagnetLink},
	{Name: "Salt.Meadow.2020.720p.WEB-DL.H264-FKW", Size: 2000, InfoHash: strings.Repeat("f", 40), TMDBID: 9000004, IMDbID: "tt9000004", Seeders: 2, Link: scenario.RedirectLink},
}

func scenarioWithSeed() scenario.Scenario { return scenario.Scenario{Name: "x", Seed: "site-test"} }

func spec(rels []site.Release) fake.Spec {
	return fake.Spec{Name: site.Fake.Name, TLS: fake.FrontTLS, Online: site.Fake.Online, New: func(s scenario.Scenario, env fake.Env) (fake.Server, error) {
		return fake.HTTP(env, site.Routes(site.APIKey(s), rels, func(string) (string, error) { return announce, nil })), nil
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
			{api + "&t=movie&tmdbid=9000001", feed(item(origin, listing[0]), item(origin, listing[1]))},
			{api + "&t=movie&imdbid=tt9000002", feed(item(origin, listing[2]))},
			{api + "&t=movie&tmdbid=9000003", feed()},
			{api + "&t=search&q=quiet+ORCHARD+2021", feed(item(origin, listing[0]), item(origin, listing[1]))},
			{api + "&t=search&q=orchard+1080p", feed(item(origin, listing[0]), item(origin, listing[2]))},
			{api + "&t=search&q=orch", feed()},
			{api + "&t=search&q=orchard&offset=1&limit=1", feed(item(origin, listing[1]))},
			{api + "&t=search&q=orchard&cat=5000", feed()},
			{api + "&t=search&q=orchard&cat=5000,2040&offset=2", feed(item(origin, listing[2]))},
			{api + "&t=tvsearch&q=orchard", errorDoc("202", "No such function")},
			{site.TorznabPath + "/api?apikey=" + site.PlayKey(910) + "&t=caps", errorDoc("910", "API Disabled")},
			{site.TorznabPath + "/api?apikey=" + site.PlayKey(500) + "&t=search", errorDoc("500", "Request limit reached")},
			{site.TorznabPath + "/api?apikey=" + site.PlayKey(429) + "&t=search", "429 Too Many Requests Too Many Requests\n"},
			{site.TorznabPath + "/api?apikey=wrong&t=caps", errorDoc("100", "Invalid API Key")},
			{site.TorznabPath + "/api?t=caps", errorDoc("100", "Invalid API Key")},
			{"/download/" + hash1080 + ".torrent", "200 OK " + rewritten},
			{"/download/" + strings.Repeat("d", 40) + ".torrent", "404 Not Found 404 page not found\n"},
		} {
			if got := get(t, f, e.path); got != e.want {
				t.Errorf("GET %s =\n%s\nwant\n%s", e.path, got, e.want)
			}
		}
	}
	t.Run("loopback", func(t *testing.T) { check(t, faketest.Loopback(t, spec(listing), s)) })
	t.Run("in memory", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) { check(t, faketest.InMemory(t, spec(listing), s)) })
	})
}

func TestTheAPIKeyFollowsTheSeed(t *testing.T) {
	a, b := site.APIKey(scenario.Scenario{Seed: "one"}), site.APIKey(scenario.Scenario{Seed: "two"})
	if a == b || len(a) != 32 || a != site.APIKey(scenario.Scenario{Seed: "one"}) {
		t.Fatalf("keys %s and %s, want 32 hex digits that differ by seed and repeat for one", a, b)
	}
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

// created is the site served in f as an Indexer created on one Node that
// spaces nothing.
func created(t *testing.T) (faketest.Served, *indexers.Indexers, indexers.Indexer) {
	t.Helper()
	s := scenarioWithSeed()
	f := faketest.Loopback(t, spec(listing), s)
	timings := searching.DefaultTimings()
	timings.Floor = 0
	home := searching.RemoteOf(searching.New(searching.Config{HTTP: f.Client, Timings: timings, Log: slog.New(slog.DiscardHandler)}))
	lib := storagetest.Library(t, secret.Schema(), nodes.Schema(), indexers.Schema())
	self := role.NodeID{UUID: storage.NewUUID()}
	if err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, self, "home")
	}); err != nil {
		t.Fatal(err)
	}
	x := indexers.New(indexers.Config{
		Library:   lib,
		Sealer:    secrettest.Sealer(t),
		Roster:    indexerstest.Alone(self),
		Searching: func(role.NodeID) searching.Remote { return home },
		Changed:   discard[indexers.Changed]{},
		Status:    discard[indexers.StatusChanged]{},
		Timings:   indexers.DefaultTimings(),
		Log:       slog.New(slog.DiscardHandler),
	})
	v, err := x.Create(t.Context(), identity.Admin{}, indexers.Draft{Name: "Fixture Site", Address: site.URL(f.Address), APIKey: site.APIKey(s)})
	if err != nil {
		t.Fatal(err)
	}
	return f, x, v.Indexer
}

func TestTheSearchingServiceReadsTheSite(t *testing.T) {
	_, x, idx := created(t)
	if got, want := idx.Caps.Modes["movie"], []string{"q", "tmdbid", "imdbid"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("movie caps = %v, want %v", got, want)
	}

	a, err := x.Search(t.Context(), idx, indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"}, Query: "The Quiet Orchard 2021"})
	if err != nil || a.Outcome.Kind != indexers.Found {
		t.Fatalf("Search = %+v, %v; want found", a.Outcome, err)
	}
	found := a.Releases
	var got []string
	for _, r := range found {
		got = append(got, r.Title+" "+r.InfoHash.String()+" "+strconv.Itoa(r.Seeders)+" "+r.IDs["tmdbid"]+" "+r.IDs["imdbid"])
	}
	want := []string{
		"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW " + hash1080 + " 12 9000001 tt9000001",
		"The.Quiet.Orchard.2021.720p.WEB-DL.H264-FKW " + hash720 + " 40 9000001 tt9000001",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("releases =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	fetched, err := x.Fetch(t.Context(), idx, found[0].Link)
	if got, ok := fetched.(indexers.Torrent); err != nil || !ok || string(got.Raw) != rewritten {
		t.Fatalf("Fetch = %+v, %v, want the .torrent %q", fetched, err, rewritten)
	}
	m, err := x.Search(t.Context(), idx, indexers.Terms{Mode: "movie", IDs: map[string]string{"tmdbid": "9000004"}})
	magnets := m.Releases
	if err != nil || len(magnets) != 2 {
		t.Fatalf("Search for the magnet releases = %+v, %v; want two", magnets, err)
	}
	for i, want := range []string{strings.Repeat("e", 40), strings.Repeat("f", 40)} {
		link := magnets[i].Link
		if link.IsZero() {
			link = magnets[i].Magnet
		}
		fetched, err := x.Fetch(t.Context(), idx, link)
		got, ok := fetched.(indexers.Magnet)
		if err != nil || !ok || got.Link.InfoHash.String() != want || len(got.Link.Trackers) != 1 || got.Link.Trackers[0] != announce {
			t.Errorf("Fetch of %s = %+v, %v; want the magnet link of %s announcing to %s", magnets[i].Title, fetched, err, want, announce)
		}
	}
}

func TestTheSearchingServiceFetchesFromAnyHostAndNamesARefusal(t *testing.T) {
	f, x, idx := created(t)
	var asked string
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.String()
		_, _ = w.Write([]byte(rewritten))
	}))
	t.Cleanup(elsewhere.Close)
	fetched, err := x.Fetch(t.Context(), idx, searching.LinkOf(elsewhere.URL+"/a.torrent?id=1"))
	if got, ok := fetched.(indexers.Torrent); err != nil || !ok || string(got.Raw) != rewritten || asked != "/a.torrent?id=1" {
		t.Fatalf("a Fetch from another host = %+v, %v after asking for %s; want the .torrent asked for without the key", fetched, err, asked)
	}
	_, err = x.Fetch(t.Context(), idx, searching.LinkOf(f.Address.URL("/download/"+strings.Repeat("d", 40)+".torrent")))
	var unavailable *indexers.UnavailableError
	if !errors.As(err, &unavailable) || unavailable.Wait || err.Error() != "the Indexer is unavailable: the Indexer responded with status 404" {
		t.Fatalf("a missing torrent = %v, want an unavailable error that does not wait", err)
	}

	wrong := "wrong"
	_, err = x.Update(t.Context(), identity.Admin{}, idx.ID, indexers.Edit{APIKey: &wrong})
	var failed *indexers.TestFailedError
	want := "Fixture Site refused the API key with 'Invalid API Key'. Check the account, then test again."
	if !errors.As(err, &failed) || failed.Outcome.Cause != searching.AccountRefused || failed.Sentence != want {
		t.Fatalf("a wrong key = %v, want account_refused with %q", err, want)
	}
}

// job runs call on the site at f as a Torznab Indexer with key, on a Node
// that spaces nothing.
func job(t *testing.T, f faketest.Served, key string, call searching.Call) searching.Answer {
	t.Helper()
	timings := searching.DefaultTimings()
	timings.Floor = 0
	node := searching.RemoteOf(searching.New(searching.Config{HTTP: f.Client, Timings: timings, Log: slog.New(slog.DiscardHandler)}))
	poll, err := node.Run(t.Context(), searching.Job{
		ID: storage.NewUUID().String(), Indexer: "fixture", Kind: searching.Torznab, Address: site.URL(f.Address), APIKey: key, Call: call,
	})
	if err != nil || poll.Answer == nil {
		t.Fatalf("Run = %+v, %v; want an answer within the hold", poll, err)
	}
	return *poll.Answer
}

func TestEachTorznabPlayLandsInItsOutcome(t *testing.T) {
	f := faketest.Loopback(t, spec(listing), scenarioWithSeed())
	search := searching.SearchCall{Query: searching.Query{Mode: "search", Text: "orchard"}}
	for _, tc := range []struct {
		code int
		want string
	}{
		{101, "*searching.Stopped the Indexer refused the account: Account suspended"},
		{910, "*searching.Stopped the Indexer refused the account: API Disabled"},
		{203, "*searching.Stopped the Indexer rejected the request: Function not available"},
		{500, "*searching.Limited the Indexer limited requests"},
		{429, "*searching.Limited the Indexer limited requests for 2s"},
		{300, "*searching.Failing the Indexer responded with error 300: No such item"},
	} {
		got := job(t, f, site.PlayKey(tc.code), search).Outcome
		if s := fmt.Sprintf("%T %v", got, got); s != tc.want {
			t.Errorf("a search refused with %d = %s, want %s", tc.code, s, tc.want)
		}
	}
}

func TestATorznabSearchPagesAndReadsEveryAttribute(t *testing.T) {
	s := scenarioWithSeed()
	many := make([]site.Release, 250)
	for i := range many {
		many[i] = site.Release{
			Name: fmt.Sprintf("Orchard.Part.%03d.1080p.WEB-DL", i), Size: 1000, InfoHash: fmt.Sprintf("%040x", i+1), TMDBID: 9000001, IMDbID: "tt9000001",
			Seeders: 7, Leechers: 3, Grabs: 40, Freeleech: i == 0, Torrent: []byte(torrent),
		}
	}
	f := faketest.Loopback(t, spec(many), s)
	a := job(t, f, site.APIKey(s), searching.SearchCall{Query: searching.Query{Mode: "movie", IDs: map[string]string{"tmdbid": "9000001"}}})
	if a.Outcome != nil || len(a.Releases) != len(many) || a.Started.IsZero() {
		t.Fatalf("a search over three pages = %d Releases, %v, started %v; want %d, ok, started", len(a.Releases), a.Outcome, a.Started, len(many))
	}
	for i, r := range a.Releases {
		if r.Title != many[i].Name {
			t.Fatalf("Release %d is %s, want %s in the site's order", i, r.Title, many[i].Name)
		}
	}
	got := a.Releases[0]
	want := fmt.Sprintf("%s 1000 2026-06-01T12:00:00Z 7/3/40 %s %v %v map[imdbid:tt9000001 tmdbid:9000001] {0 1} {1 48h0m0s} %s",
		many[0].Name, many[0].InfoHash, []int{2000}, true, f.Address.URL("/details/"+many[0].InfoHash))
	if s := fmt.Sprintf("%s %d %s %d/%d/%d %s %v %v %v %v %v %s", got.Title, got.Size, got.Published.Format(time.RFC3339), got.Seeders, got.Leechers, got.Grabs,
		got.InfoHash, got.Categories, got.Magnet.IsZero(), got.IDs, got.Factors, got.Floor, got.Details); s != want {
		t.Fatalf("the first Release =\n%s\nwant\n%s", s, want)
	}
}

func TestAnnounceRewriteKeepsTheInfoHash(t *testing.T) {
	s := scenarioWithSeed()
	meta, hash, err := releases.Metainfo("orchard.mkv", []byte("the orchard payload"), "https://127.0.0.1:1/announce/built", false)
	if err != nil {
		t.Fatal(err)
	}
	catalog := &releases.Catalog{Passkey: "p4ss", Releases: []releases.Release{{Release: scenario.Release{Name: "orchard"}, Metainfo: meta, InfoHash: hash}}}
	fronts := fake.Fronts{
		{Target: "tracker", Node: "home", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:7001"), TLS: true}},
		{Target: "tracker", Node: "seedbox", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:7002"), TLS: true}},
	}
	f := faketest.Loopback(t, site.Fake, s, faketest.WithReleases(catalog), faketest.WithFronts(fronts), faketest.As("seedbox"))
	status, body, _ := strings.Cut(get(t, f, "/download/"+hash.String()+".torrent"), " OK ")
	if status != "200" {
		t.Fatalf("GET the .torrent = %s %s", status, body)
	}
	served, err := bencode.DecodeMetainfo([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	if want := "https://127.0.0.1:7002/announce/p4ss"; served.Announce != want {
		t.Fatalf("the .torrent announces to %s, want seedbox's tracker front %s", served.Announce, want)
	}
	if served.InfoHash != hash {
		t.Fatalf("the served info hash is %s, want the catalog's %s", served.InfoHash, hash)
	}
}

func TestAMagnetLinkNamesTheReadersTrackerFront(t *testing.T) {
	s := scenarioWithSeed()
	catalog := &releases.Catalog{Passkey: "p4ss", Releases: []releases.Release{
		{Release: scenario.Release{Name: "Harbor.Lights", Link: scenario.MagnetLink}, InfoHash: bencode.InfoHash{1}},
		{Release: scenario.Release{Name: "Salt.Meadow", Link: scenario.MagnetLink, Trackerless: true, Public: true}, InfoHash: bencode.InfoHash{2}},
		{Release: scenario.Release{Name: "Quiet.Orchard", Link: scenario.RedirectLink}, InfoHash: bencode.InfoHash{3}},
	}}
	fronts := fake.Fronts{
		{Target: "tracker", Node: "home", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:7001"), TLS: true}},
		{Target: "tracker", Node: "seedbox", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:7002"), TLS: true}},
	}
	f := faketest.Loopback(t, site.Fake, s, faketest.WithReleases(catalog), faketest.WithFronts(fronts), faketest.As("seedbox"))
	origin := f.Address.URL("")
	harbor := "magnet:?xt=urn:btih:" + bencode.InfoHash{1}.String() + "&amp;dn=Harbor.Lights&amp;tr=https%3A%2F%2F127.0.0.1%3A7002%2Fannounce%2Fp4ss"
	meadow := "magnet:?xt=urn:btih:" + bencode.InfoHash{2}.String() + "&amp;dn=Salt.Meadow"
	orchard := "magnet:?xt=urn:btih:" + bencode.InfoHash{3}.String() + "&amp;dn=Quiet.Orchard&amp;tr=https%3A%2F%2F127.0.0.1%3A7002%2Fannounce%2Fp4ss"
	body := get(t, f, site.TorznabPath+"/api?apikey="+site.APIKey(s)+"&t=search&q=")
	for _, want := range []string{
		"<link>" + harbor + "</link>",
		`<enclosure url="` + harbor + `" length="0" type="application/x-bittorrent" />`,
		`<torznab:attr name="magneturl" value="` + harbor + `" />`,
		"<link>" + meadow + "</link>",
		`<torznab:attr name="magneturl" value="` + meadow + `" />`,
		"<link>" + origin + "/download/" + bencode.InfoHash{3}.String() + ".torrent</link>",
		`<torznab:attr name="magneturl" value="` + orchard + `" />`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the feed\n%s\nholds no %s", body, want)
		}
	}

	noFollow := *f.Client
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL("/download/"+bencode.InfoHash{3}.String()+".torrent"), http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := noFollow.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if want := strings.ReplaceAll(orchard, "&amp;", "&"); res.StatusCode != http.StatusFound || res.Header.Get("Location") != want {
		t.Fatalf("GET the redirect release's .torrent = %d to %q, want 302 to %s", res.StatusCode, res.Header.Get("Location"), want)
	}
}
