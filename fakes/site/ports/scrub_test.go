package ports

import (
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/site/ports/internal/testport"
)

// TestScrubRefusesAFixtureHoldingEachKindOfRealValue is the scrub
// check's lint tier: testport's fixture, clean, with one real value put
// in.
func TestScrubRefusesAFixtureHoldingEachKindOfRealValue(t *testing.T) {
	for _, tc := range []struct {
		name  string
		put   func(f *Fixture)
		found string
	}{
		{"a passkey setting", func(f *Fixture) { f.Settings["passkey"] = "a1b2c3d4e5f6a7b8c9d0e1f2" }, `settings passkey: "a1b2c3d4e5f6a7b8c9d0e1f2" is a credential`},
		{"a short password", func(f *Fixture) { f.Settings["password"] = "hunter2" }, `settings password: "hunter2" is a credential`},
		{"a username", func(f *Fixture) { f.Settings["username"] = "jdoe" }, `settings: username is "jdoe", which isn't fixture-user`},
		{"a session cookie", func(f *Fixture) { f.Session.Cookie = "sid=8f14e45fceea167a5a36dedd4bea2543" }, `session cookie sid: "8f14e45fceea167a5a36dedd4bea2543" is a credential`},
		{"a bearer token", func(f *Fixture) { f.Session.Token = "eyJhbGciOiJIUzI1NiJ9" }, `session token: "eyJhbGciOiJIUzI1NiJ9" is a credential`},
		{"a cookie a response sets", func(f *Fixture) {
			f.Cases[0].Exchanges[0].Response.Headers["Set-Cookie"] = "uid=31337; Path=/"
		}, `case search exchange 1 response header Set-Cookie uid: "31337" is a credential`},
		{"an API key header", func(f *Fixture) {
			f.Cases[0].Exchanges[0].Request.Headers["X-Api-Key"] = "k3yk3yk3yk3y"
		}, `case search exchange 1 request header X-Api-Key: "k3yk3yk3yk3y" is a credential`},
		{"a passkey in a query", func(f *Fixture) {
			f.Cases[0].Exchanges[0].Request.Query["torrent_pass"] = "abcd"
		}, `case search exchange 1 query torrent_pass: "abcd" is a credential`},
		{"a username in a form", func(f *Fixture) {
			f.Cases[3].Exchanges[1].Request.Form["username"] = "jdoe"
		}, `case auth exchange 2 form: username is "jdoe"`},
		{"a key in a body", func(f *Fixture) {
			f.Bodies["search.json"] = []byte(`{"download":"/dl?key=9b74c9897bac770ffc029102a200c5de"}`)
		}, `bodies/search.json: "9b74c9897bac770ffc029102a200c5de" is shaped like a key`},
		{"an uploader in a body", func(f *Fixture) { f.Bodies["search.json"] = []byte(`{"Uploader": "SomeSeeder"}`) }, `bodies/search.json: Uploader is "SomeSeeder"`},
		{"a real title in a body", func(f *Fixture) {
			f.Bodies["search.json"] = []byte(`{"name":"Inception.2010.1080p.BluRay.x264-GRP"}`)
		}, `bodies/search.json: title "Inception.2010.1080p.BluRay.x264-GRP" isn't one of the release catalog's fictional titles`},
		{"a real title searched for", func(f *Fixture) { f.Cases[0].Call.Search.Text = "Inception" }, `case search search: title "Inception" isn't one`},
		{"an unminted info hash", func(f *Fixture) {
			f.Bodies["search.json"] = []byte(`{"infohash":"0123456789abcdef0123456789abcdef01234567"}`)
		}, `bodies/search.json: info hash 0123456789abcdef0123456789abcdef01234567 isn't one the fixture catalog minted`},
		{"a real title in a golden", func(f *Fixture) {
			f.Wants["search"] = []byte(`{"releases":[{"title":"Inception 2010 1080p"}]}`)
		}, `want/search.json: title "Inception 2010 1080p" isn't one`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := cleanFixture(t)
			tc.put(&f)
			found := Scrub(f)
			if !slices.ContainsFunc(found, func(s string) bool { return strings.HasPrefix(s, tc.found) }) {
				t.Errorf("Scrub = %q, want one starting %q", found, tc.found)
			}
		})
	}
}

func TestScrubPassesASubmitButtonNamedLogin(t *testing.T) {
	for value, refused := range map[string]bool{"Log in!": false, "jdoe": true} {
		f := cleanFixture(t)
		f.Cases[3].Exchanges[1].Request.Form["login"] = value
		found := Scrub(f)
		if got := slices.ContainsFunc(found, func(s string) bool { return strings.Contains(s, "login is") }); got != refused {
			t.Errorf("Scrub of form login %q = %q, want refused %v", value, found, refused)
		}
	}
}

func TestScrubPassesAnUpstreamConstantOnly(t *testing.T) {
	for path, refused := range map[string]bool{
		"/api_9djWe8Tb2NE3p6opyqnh/v1/browse.php": false,
		"/api_9djWe8Tb2NE3p6opyqnX/v1/browse.php": true,
	} {
		f := cleanFixture(t)
		f.Cases[0].Exchanges[0].Request.Path = path
		found := Scrub(f)
		if got := slices.ContainsFunc(found, func(s string) bool { return strings.Contains(s, "is shaped like a key") }); got != refused {
			t.Errorf("Scrub of path %q = %q, want refused %v", path, found, refused)
		}
	}
}

// cleanFixture is testport's fixture, which Scrub passes, with maps the
// cases above write to.
func cleanFixture(t *testing.T) Fixture {
	t.Helper()
	f, err := Embedded(testport.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found := Scrub(f); len(found) > 0 {
		t.Fatalf("testport's fixture doesn't pass: %q", found)
	}
	x := &f.Cases[0].Exchanges[0]
	x.Response.Headers["Set-Cookie"] = ""
	x.Request.Headers["X-Api-Key"] = "{{apiKey}}"
	return f
}

func TestTheRewriteTurnsARecordingIntoWhatScrubPasses(t *testing.T) {
	ptp := `{"Movies":[{"GroupId":"148131","Title":"The Night Of","Year":"2016","Cover":"https:\/\/passthepopcorn.me\/c.jpg",` +
		`"Torrents":[{"Id":452135,"ReleaseName":"The.Night.Of.S01.BluRay.AAC2.0.x264-DEPTH","InfoHash":"AB5C9A64E0E0A3B7C8C0E7D3F1D2C4B5A6978899"},` +
		`{"Id":465090,"ReleaseName":"The.Night.Of.2016.S01.REPACK.576p.BluRay.x264-HANDJOB","Uploader":"someone"}]}],"AuthKey":"9b74c9897bac770ffc029102a200c5de"}`
	ipt := `<item><title>24 S03E12 720p WEBRip h264-DRAWER</title>` +
		`<link>http://iptorrents.com/download.php/1234/24.S03E12.720p.WEBRip.h264-DRAWER.torrent?torrent_pass=abcd</link>` +
		`<pubDate>Mon, 12 May 2014 19:06:24 +0000</pubDate></item>` +
		`<item><title>Rosemary&#39;s Baby S01E01 Part 1 1080p WEB-DL DD5 1 H 264-BS</title></item>`
	r := &Rewriter{
		Host: Host("lantern"), Hosts: []string{"passthepopcorn.me", "iptorrents.com"},
		Replace: map[string]string{"9b74c9897bac770ffc029102a200c5de": "{{authKey}}"},
		Titles:  []string{"24 S03E12 720p WEBRip h264-DRAWER", "Rosemary's Baby S01E01 Part 1 1080p WEB-DL DD5 1 H 264-BS"},
	}
	f := Fixture{Bodies: map[string][]byte{}}
	for _, b := range []struct{ name, raw string }{{"ipt.xml", ipt}, {"ptp.json", ptp}} {
		out, err := r.Rewrite([]byte(b.raw))
		if err != nil {
			t.Fatal(err)
		}
		f.Bodies[b.name] = out
	}
	if found := Scrub(f); len(found) > 0 {
		t.Errorf("Scrub = %q over\n%s\n%s", found, f.Bodies["ptp.json"], f.Bodies["ipt.xml"])
	}
	got := string(f.Bodies["ptp.json"]) + string(f.Bodies["ipt.xml"])
	for _, want := range []string{
		`"Title":"Juniper Falls"`, `"ReleaseName":"Juniper.Falls.S01.BluRay`, `"ReleaseName":"Juniper.Falls.2016.S01.REPACK`,
		`"InfoHash":"` + strings.ToUpper(Mint(0)) + `"`, `"Uploader":"fixture-user"`, `"AuthKey":"{{authKey}}"`, `https:\/\/lantern.fixture-site.fake.localhost\/c.jpg`,
		`<title>Cedar Hollow S03E12 720p WEBRip h264-DRAWER</title>`, `/Cedar.Hollow.S03E12.720p.WEBRip.h264-DRAWER.torrent`,
		"19:06:24", `<title>Harbor Lights S01E01 Part 1`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the rewrite holds no %s:\n%s", want, got)
		}
	}
}

func TestAnEpisodeATagOrAnAuthorEndsATitlesStem(t *testing.T) {
	for title, want := range map[string]string{
		"Harbor.Lights.E05.1080p.WEB-DL":                "Harbor Lights",
		"Harbor Lights E1234 720p":                      "Harbor Lights",
		"Harbor.Lights.S02E05.1080p":                    "Harbor Lights",
		"Evermore.1080p":                                "Evermore",
		"[SubsPlease] Harbor Lights - 03 (1080p)":       "Harbor Lights",
		"[SubsPlease] Salt Meadow - Movie (720p)":       "Salt Meadow",
		"Harbor Lights - Second Tide 2021":              "Harbor Lights - Second Tide",
		"The Quiet Orchard by Ada Fernsby [ENG / EPUB]": "The Quiet Orchard",
		"The Quiet Orchard [FRA]":                       "The Quiet Orchard",
	} {
		if got := Stem(title); got != want {
			t.Errorf("Stem(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestAGazelleMusicTitlesStemIsItsAlbum(t *testing.T) {
	for title, want := range map[string]string{
		"Juniper & Vale - The Quiet Orchard (2021) [EP] [FLAC Lossless / WEB]": "The Quiet Orchard",
		"The Lantern Choir - Cedar Hollow (1969) [Album] [MP3 V2 (VBR) / BD]":  "Cedar Hollow",
		"Harbor.Lights.2020.1080p.WEB-DL-GROUP":                                "Harbor Lights",
	} {
		if got := Stem(title); got != want {
			t.Errorf("Stem(%q) = %q, want %q", title, got, want)
		}
	}
}

func TestTheRewriteSwapsATitlesSlugInALink(t *testing.T) {
	r := &Rewriter{Host: Host("lantern")}
	out, err := r.Rewrite([]byte(`<a href="/torrent/7-the-night-of-2016-s01">The.Night.Of.2016.S01.1080p.BluRay</a>`))
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	name := got[strings.Index(got, ">")+1 : strings.LastIndex(got, "<")]
	slug := strings.ToLower(strings.ReplaceAll(Stem(name), " ", "-"))
	if strings.Contains(got, "the-night-of") || !strings.Contains(got, "/torrent/7-"+slug+"-2016-s01") {
		t.Errorf("the rewrite = %s, want the link's slug to be %s", got, slug)
	}
}
