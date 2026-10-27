package site_test

import (
	"crypto/sha1" //nolint:gosec // a definition's sha names its blob, as git does
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/site/definitions"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

const (
	orchard = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"
	meadow  = "Salt.Meadow.2020.1080p.WEB-DL.H264-FKW"
	// browser is the User-Agent of the browser an Admin pasted Cinder's
	// cookie from.
	browser = "Mozilla/5.0 (fixture)"
)

// catalogWorld is fixture-site as seedbox reaches it, listing a .torrent
// release and a magnet one, and a Node that reaches it.
type catalogWorld struct {
	f       faketest.Served
	n       searching.Remote
	s       scenario.Scenario
	torrent bencode.InfoHash
}

// served is how a test serves fixture-site: faketest.InMemory inside a
// bubble, or faketest.Loopback.
type served func(testing.TB, fake.Spec, scenario.Scenario, ...faketest.Option) faketest.Served

func newCatalogWorld(t *testing.T, serve served) catalogWorld {
	t.Helper()
	s := scenarioWithSeed()
	meta, hash, err := releases.Metainfo("orchard.mkv", []byte("the orchard payload"), "https://127.0.0.1:1/announce/built", false)
	if err != nil {
		t.Fatal(err)
	}
	catalog := &releases.Catalog{Passkey: "p4ss", Releases: []releases.Release{
		{Release: scenario.Release{Name: orchard, TMDBID: 9000001, Seeders: 12}, Metainfo: meta, InfoHash: hash, Size: 6000},
		{Release: scenario.Release{Name: meadow, TMDBID: 9000004, Seeders: 3, Link: scenario.MagnetLink}, InfoHash: bencode.InfoHash{1}, Size: 5000},
	}}
	fronts := fake.Fronts{{Target: "tracker", Node: "seedbox", Address: fake.Address{AddrPort: netip.MustParseAddrPort("127.0.0.1:7002"), TLS: true}}}
	f := serve(t, site.Fake, s, faketest.WithReleases(catalog), faketest.WithFronts(fronts), faketest.As("seedbox"))
	timings := searching.DefaultTimings()
	timings.Floor = 0
	n := searching.RemoteOf(searching.New(searching.Config{HTTP: f.Client, Timings: timings, Log: slog.New(slog.DiscardHandler)}))
	return catalogWorld{f: f, n: n, s: s, torrent: hash}
}

// indexer is a catalog Indexer on the site label, as its Controller asks
// for it, keeping the session each answer hands back.
type indexer struct {
	w        catalogWorld
	id       string
	label    string
	def      searching.Definition
	settings map[string]string
	session  *searching.Session
}

func (w catalogWorld) indexer(t *testing.T, label string, settings map[string]string) *indexer {
	t.Helper()
	yaml, err := fs.ReadFile(definitions.FS, label+".yml")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha1.Sum(yaml) //nolint:gosec // see the import
	return &indexer{
		w: w, id: storage.NewUUID().String(), label: label, settings: settings,
		def: searching.Definition{YAML: yaml, SHA: hex.EncodeToString(sum[:])},
	}
}

// ask runs call on the Node, polling until it answers.
func (ix *indexer) ask(t *testing.T, call searching.Call) searching.Answer {
	t.Helper()
	job := searching.Job{
		ID: storage.NewUUID().String(), Indexer: ix.id, Kind: searching.Catalog, Address: site.Origin(ix.w.f.Address, ix.label) + "/",
		Definition: &ix.def, Settings: ix.settings, Session: ix.session, Call: call,
	}
	for {
		poll, err := ix.w.n.Run(t.Context(), job)
		if err != nil {
			t.Fatalf("Run = %v", err)
		}
		if a := poll.Answer; a != nil {
			if a.Session != nil {
				ix.session = a.Session
			}
			return *a
		}
	}
}

func (ix *indexer) search(t *testing.T, text string) searching.Answer {
	t.Helper()
	return ix.ask(t, searching.SearchCall{Query: searching.Query{Mode: "search", Text: text}})
}

// titles are the titles of a's Releases, or its outcome.
func titles(a searching.Answer) string {
	if a.Outcome != nil {
		return stated(a)
	}
	var out []string
	for _, r := range a.Releases {
		out = append(out, r.Title)
	}
	return strings.Join(out, ", ")
}

// got is what a fetch answered: the info hash of its .torrent or its
// magnet link.
func got(t *testing.T, a searching.Answer) string {
	t.Helper()
	switch g := a.Got.(type) {
	case searching.Torrent:
		meta, err := bencode.DecodeMetainfo(g.Raw)
		if err != nil {
			t.Fatal(err)
		}
		return "torrent " + meta.InfoHash.String()
	case searching.Magnet:
		return "magnet " + g.Link.Reveal()
	}
	return stated(a)
}

// signIns are the sign-ins site accepted, by their method.
func signIns(t *testing.T, f faketest.Served, label string) []string {
	t.Helper()
	var out []string
	for _, r := range f.Records(t) {
		var in site.SignIn
		if r.Kind == site.SignedIn.Kind && json.Unmarshal(r.Data, &in) == nil && in.Site == label {
			out = append(out, r.Node+" "+in.Method)
		}
	}
	return out
}

// answered is the captcha a CaptchaCall answered, answered with text, or
// with the text it shows when text is empty.
func answered(t *testing.T, ix *indexer, text string) searching.TestCall {
	t.Helper()
	a := ix.ask(t, searching.CaptchaCall{})
	if a.Captcha == nil {
		t.Fatalf("Captcha = %s, want an image", stated(a))
	}
	if text == "" {
		shown, err := site.CaptchaText(a.Captcha.Image)
		if err != nil {
			t.Fatal(err)
		}
		text = shown
	}
	return searching.TestCall{Captcha: &searching.CaptchaAnswer{PreLogin: a.Captcha.PreLogin, Text: text}}
}

func TestLarkspurSignsInPastACaptchaAndThanksBeforeADownload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newCatalogWorld(t, faketest.InMemory)
		ix := w.indexer(t, site.Larkspur, map[string]string{"username": site.Username, "password": site.Password(w.s, site.Larkspur), "sort": "added"})
		if got, want := stated(ix.ask(t, searching.TestCall{})), "*searching.Stopped "; !strings.HasPrefix(got, want) || !strings.Contains(got, "captcha") {
			t.Errorf("a Test without an answer = %s, want a stop for the captcha", got)
		}
		if got := stated(ix.ask(t, answered(t, ix, "wrong"))); !strings.HasSuffix(got, ": "+site.WrongCaptcha) {
			t.Errorf("a Test with a wrong answer = %s, want the site's %q", got, site.WrongCaptcha)
		}
		wrong := w.indexer(t, site.Larkspur, map[string]string{"username": site.Username, "password": "nope", "sort": "added"})
		if got := stated(wrong.ask(t, answered(t, wrong, ""))); !strings.HasSuffix(got, ": "+site.WrongPassword) {
			t.Errorf("a Test with a wrong password = %s, want the site's %q", got, site.WrongPassword)
		}
		test := ix.ask(t, answered(t, ix, ""))
		if test.Report == nil || test.Report.Items != 2 || test.Session == nil {
			t.Fatalf("a Test with the shown text = %s, session %+v; want 2 items and a session", stated(test), test.Session)
		}
		found := ix.search(t, "orchard")
		if titles(found) != orchard {
			t.Fatalf("search = %s, want %s alone, past the garbled row", titles(found), orchard)
		}
		r := found.Releases[0]
		if r.Size != 6000 || r.Seeders != 12 || !slices.Contains(r.Categories, 2040) || r.Published.IsZero() {
			t.Errorf("the Release = %+v, want 6000 B, 12 seeders, Movies/HD and its date", r)
		}
		if g := got(t, ix.ask(t, searching.FetchCall{Link: r.Link})); g != "torrent "+w.torrent.String() {
			t.Errorf("fetch = %s, want the .torrent after thanks", g)
		}
		if got, want := signIns(t, w.f, site.Larkspur), []string{"seedbox form"}; !slices.Equal(got, want) {
			t.Errorf("sign-ins = %v, want %v", got, want)
		}
		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.Larkspur})
		if got := stated(ix.search(t, "orchard")); !strings.Contains(got, "captcha") {
			t.Errorf("a search once the session expired = %s, want a stop for the captcha, since a sign-in now needs one", got)
		}
	})
}

func TestQuarrySignsInWithItsKeyEachTimeAndKeepsNoSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newCatalogWorld(t, faketest.InMemory)
		ix := w.indexer(t, site.Quarry, map[string]string{"apikey": site.Key(w.s, site.Quarry)})
		test := ix.ask(t, searching.TestCall{})
		if test.Report == nil || test.Report.Items != 2 || test.Session != nil {
			t.Fatalf("Test = %s, session %+v; want 2 items and no session", stated(test), test.Session)
		}
		found := ix.search(t, "meadow")
		if titles(found) != meadow || found.Releases[0].Published.IsZero() || !slices.Contains(found.Releases[0].Categories, 2000) {
			t.Fatalf("search = %s %+v, want %s dated in Movies", titles(found), found.Releases, meadow)
		}
		orchardFound := ix.search(t, "quiet orchard")
		if g := got(t, ix.ask(t, searching.FetchCall{Link: orchardFound.Releases[0].Link})); g != "torrent "+w.torrent.String() {
			t.Errorf("fetch = %s, want the .torrent", g)
		}
		wrong := w.indexer(t, site.Quarry, map[string]string{"apikey": "nope"})
		if got := stated(wrong.ask(t, searching.TestCall{})); !strings.HasPrefix(got, "*searching.Stopped ") {
			t.Errorf("a Test with a wrong key = %s, want a stop", got)
		}
		if got := stated(wrong.search(t, "orchard")); !strings.HasPrefix(got, "*searching.Stopped ") {
			t.Errorf("a search with a wrong key = %s, want a stop", got)
		}
	})
}

// cinderSession is the session an Admin pasted from a browser that cleared
// Cinder's challenge as origin.
func cinderSession(s scenario.Scenario, origin string) *searching.Session {
	name, value, _ := strings.Cut(site.SessionCookie(s), "=")
	host := site.Host(site.Cinder)
	return &searching.Session{
		Cookies: []searching.Cookie{
			{Name: name, Value: value, Domain: host, Path: "/", HostOnly: true},
			{Name: site.ClearanceCookie, Value: site.Clearance(s, origin, browser), Domain: host, Path: "/", HostOnly: true},
		},
		UserAgent: browser, Pasted: true, Earned: time.Now(),
	}
}

func TestCinderTakesAPastedCookieBehindItsChallenge(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newCatalogWorld(t, faketest.InMemory)
		ix := w.indexer(t, site.Cinder, nil)
		ix.session = cinderSession(w.s, "seedbox")
		if test := ix.ask(t, searching.TestCall{}); test.Report == nil || test.Report.Items != 2 {
			t.Fatalf("Test = %s, want 2 items", stated(test))
		}
		found := ix.search(t, "orchard")
		if titles(found) != orchard || found.Releases[0].InfoHash == nil || *found.Releases[0].InfoHash != w.torrent {
			t.Fatalf("search = %s %+v, want %s with its info hash", titles(found), found.Releases, orchard)
		}
		want := "magnet magnet:?xt=urn:btih:" + w.torrent.String() + "&dn=" + orchard
		if g := got(t, ix.ask(t, searching.FetchCall{Link: found.Releases[0].Link})); g != want {
			t.Errorf("fetch = %s, want %s", g, want)
		}

		elsewhere := w.indexer(t, site.Cinder, nil)
		elsewhere.session = cinderSession(w.s, "home")
		if got, want := stated(elsewhere.search(t, "orchard")), "*searching.Stopped "; !strings.HasPrefix(got, want) || !strings.Contains(got, "challenge") {
			t.Errorf("a search with another Node's clearance = %s, want a stop at the challenge", got)
		}

		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.Cinder})
		if got := stated(ix.search(t, "orchard")); !strings.Contains(got, "cookie") {
			t.Errorf("a search once the pasted cookie expired = %s, want a stop for the cookie", got)
		}
	})
}

func TestMarshSignsInAgainOnceItsSessionExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newCatalogWorld(t, faketest.InMemory)
		ix := w.indexer(t, site.Marsh, map[string]string{"username": site.Username, "password": site.Password(w.s, site.Marsh)})
		found := ix.search(t, "orchard")
		if titles(found) != orchard {
			t.Fatalf("search = %s, want %s", titles(found), orchard)
		}
		r := found.Releases[0]
		if r.Size != 6000 || r.Seeders != 12 || !slices.Contains(r.Categories, 2000) || r.Published.Format(time.DateOnly) != "2026-06-01" {
			t.Errorf("the Release = %+v, want 6000 B, 12 seeders, the windows-1251 category as Movies and its day's date", r)
		}
		first := ix.session
		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.Marsh})
		time.Sleep(time.Second)
		if titles(ix.search(t, "orchard")) != orchard || !ix.session.Earned.After(first.Earned) {
			t.Errorf("a search once the session expired = %s, session earned %v after %v; want a new session", titles(found), ix.session.Earned, first.Earned)
		}
		if got, want := signIns(t, w.f, site.Marsh), []string{"seedbox post", "seedbox post"}; !slices.Equal(got, want) {
			t.Errorf("sign-ins = %v, want %v", got, want)
		}
		if g := got(t, ix.ask(t, searching.FetchCall{Link: r.Link})); g != "torrent "+w.torrent.String() {
			t.Errorf("fetch = %s, want the .torrent", g)
		}
		if got := stated(ix.search(t, "or")); !strings.HasPrefix(got, "*searching.Failing ") || !strings.Contains(got, site.MarshTooShort) {
			t.Errorf("a two-letter search = %s, want the site's %q", got, site.MarshTooShort)
		}
		wrong := w.indexer(t, site.Marsh, map[string]string{"username": site.Username, "password": "nope"})
		if got := stated(wrong.ask(t, searching.TestCall{})); !strings.HasSuffix(got, ": "+site.WrongPassword) {
			t.Errorf("a Test with a wrong password = %s, want the site's %q", got, site.WrongPassword)
		}
	})
}

// Ferry runs on the wall clock: its regular expressions start regexp2's
// timeout clock, a goroutine that would outlive a bubble.
func TestFerryListsMagnetLinksAndSkipsAMirrorThatServesAPage(t *testing.T) {
	w := newCatalogWorld(t, faketest.Loopback)
	ix := w.indexer(t, site.Ferry, map[string]string{"passkey": site.Key(w.s, site.Ferry)})
	found := ix.search(t, "Quiet ORCHARD")
	if titles(found) != orchard {
		t.Fatalf("search = %s, want %s", titles(found), orchard)
	}
	r := found.Releases[0]
	if r.Size != 6000 || r.Seeders != 12 || r.Leechers != 0 || !strings.HasPrefix(r.Magnet.Reveal(), "magnet:?xt=urn:btih:"+w.torrent.String()) ||
		time.Since(r.Published).Round(time.Hour) != 2*time.Hour {
		t.Errorf("the Release = %+v, want 6000 B, 12/0 peers, its magnet link and two hours old", r)
	}
	if g := got(t, ix.ask(t, searching.FetchCall{Link: r.Link})); g != "torrent "+w.torrent.String() {
		t.Errorf("fetch = %s, want the .torrent the second link names", g)
	}
	wrong := w.indexer(t, site.Ferry, map[string]string{"passkey": "nope"})
	if got := stated(wrong.ask(t, searching.TestCall{})); !strings.HasPrefix(got, "*searching.Stopped ") {
		t.Errorf("a Test with a wrong passkey = %s, want a stop", got)
	}
}

func TestACatalogSiteInMaintenanceIsFailing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newCatalogWorld(t, faketest.InMemory)
		ix := w.indexer(t, site.Quarry, map[string]string{"apikey": site.Key(w.s, site.Quarry)})
		w.f.Cue(t, site.Maintenance.Name, site.SiteArgs{Site: site.Quarry, On: true})
		if got := stated(ix.search(t, "orchard")); !strings.HasPrefix(got, "*searching.Failing ") {
			t.Errorf("a search in maintenance = %s, want failing", got)
		}
		w.f.Cue(t, site.Maintenance.Name, site.SiteArgs{Site: site.Quarry})
		if got := titles(ix.search(t, "orchard")); got != orchard {
			t.Errorf("a search after maintenance = %s, want %s", got, orchard)
		}
	})
}
