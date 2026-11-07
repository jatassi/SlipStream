package site_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
)

// portedWorld is fixture-site with orchard's movie on TMDB, so its
// release has an IMDb id, as seedbox reaches it.
func portedWorld(t *testing.T) catalogWorld {
	t.Helper()
	s := scenarioWithSeed()
	s.TMDB = &scenario.TMDB{Movies: []scenario.TMDBMovie{{ID: 9000001, IMDbID: "tt9000001", Title: "The Quiet Orchard"}}}
	return newWorld(t, faketest.InMemory, s)
}

// native is a native Indexer on the family site label, keeping the
// session each answer hands back.
type native struct {
	w        catalogWorld
	id       string
	label    string
	settings map[string]string
	session  *searching.Session
}

func (w catalogWorld) native(label string, settings map[string]string) *native {
	return &native{w: w, id: storage.NewUUID().String(), label: label, settings: settings}
}

// pasted is the session of a browser cookie an Admin pasted with userAgent.
func (ix *native) pasted(cookie, userAgent string) *native {
	ix.session = &searching.Session{UserAgent: userAgent, Pasted: true}
	for pair := range strings.SplitSeq(cookie, "; ") {
		name, value, _ := strings.Cut(pair, "=")
		ix.session.Cookies = append(ix.session.Cookies, searching.Cookie{Name: name, Value: value, Domain: site.Host(ix.label), Path: "/", HostOnly: true})
	}
	return ix
}

func (ix *native) ask(t *testing.T, call searching.Call) searching.Answer {
	t.Helper()
	job := searching.Job{
		ID: storage.NewUUID().String(), Indexer: ix.id, Kind: searching.Native, Port: ix.label, Address: site.Origin(ix.w.f.Address, ix.label) + "/",
		Settings: ix.settings, Session: ix.session, Call: call,
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

func (ix *native) movie(t *testing.T) searching.Answer {
	t.Helper()
	return ix.ask(t, searching.SearchCall{Query: searching.Query{Mode: "movie", Text: "The Quiet Orchard", IDs: map[string]string{"imdbid": "tt9000001"}}})
}

// stop is a's cause and the site's words, "" when it didn't stop.
func stop(a searching.Answer) string {
	var s *searching.Stopped
	if !errors.As(a.Outcome, &s) {
		return ""
	}
	return strings.TrimSuffix(string(s.Cause)+" "+s.Text, " ")
}

// grabs checks ix's movie search finds orchard alone, with its size,
// seeders and a date, and its fetch gets orchard's .torrent.
func grabs(t *testing.T, ix *native) {
	t.Helper()
	found := ix.movie(t)
	if titles(found) != orchard {
		t.Fatalf("%s's movie search = %s, want %s alone", ix.label, titles(found), orchard)
	}
	r := found.Releases[0]
	if r.Size != 6000 || r.Seeders != 12 || r.Published.IsZero() {
		t.Errorf("%s's Release = %+v, want 6000 B, 12 seeders and its date", ix.label, r)
	}
	if g := got(t, ix.ask(t, searching.FetchCall{Link: r.Link})); g != "torrent "+ix.w.torrent.String() {
		t.Errorf("%s's fetch = %s, want orchard's .torrent", ix.label, g)
	}
}

func signedInWith(t *testing.T, ix *native) []string { return signIns(t, ix.w.f, ix.label) }

func TestPassThePopcornTakesItsAPIUserAndKeyOnEveryRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := portedWorld(t)
		settings, _ := site.Credentials(w.s, site.PassThePopcorn, "seedbox")
		ix := w.native(site.PassThePopcorn, settings)
		if test := ix.ask(t, searching.TestCall{}); test.Report == nil || test.Report.Items != 2 || test.Session != nil {
			t.Fatalf("Test = %s, session %+v; want 2 items and no session", stated(test), test.Session)
		}
		grabs(t, ix)
		wrong := w.native(site.PassThePopcorn, map[string]string{"apiUser": site.Username, "apiKey": "nope"})
		if got := stop(wrong.ask(t, searching.TestCall{})); got != string(searching.AccountRefused) {
			t.Errorf("a Test with a wrong key = %q, want account_refused", got)
		}
	})
}

func TestTorrentBytesSignsInWithItsFormAgainOnceItsSessionExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := portedWorld(t)
		settings, _ := site.Credentials(w.s, site.TorrentBytes, "seedbox")
		ix := w.native(site.TorrentBytes, settings)
		if test := ix.ask(t, searching.TestCall{}); test.Report == nil || test.Report.Items != 2 || test.Session == nil {
			t.Fatalf("Test = %s, session %+v; want 2 items and a session", stated(test), test.Session)
		}
		grabs(t, ix)
		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.TorrentBytes})
		grabs(t, ix)
		if got, want := signedInWith(t, ix), []string{"seedbox form", "seedbox form"}; !slices.Equal(got, want) {
			t.Errorf("sign-ins = %v, want %v: the Test's and the one after the session expired", got, want)
		}
		wrong := w.native(site.TorrentBytes, map[string]string{"username": site.Username, "password": "nope"})
		if got, want := stop(wrong.ask(t, searching.TestCall{})), string(searching.CredentialsRejected)+" "+site.TorrentBytesRefused; got != want {
			t.Errorf("a Test with a wrong password = %q, want %q", got, want)
		}
	})
}

func TestAvistaZSignsInToABearerTokenAndAgainOnceItExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := portedWorld(t)
		settings, _ := site.Credentials(w.s, site.AvistaZ, "seedbox")
		ix := w.native(site.AvistaZ, settings)
		if test := ix.ask(t, searching.TestCall{}); test.Report == nil || test.Report.Items != 2 || test.Session == nil || test.Session.Token == "" {
			t.Fatalf("Test = %s, session %+v; want 2 items and a bearer token", stated(test), test.Session)
		}
		grabs(t, ix)
		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.AvistaZ})
		grabs(t, ix)
		if got, want := signedInWith(t, ix), []string{"seedbox token", "seedbox token"}; !slices.Equal(got, want) {
			t.Errorf("sign-ins = %v, want %v: the Test's and the one after the token expired", got, want)
		}
		settings["pid"] = "nope"
		if got, want := stop(w.native(site.AvistaZ, settings).ask(t, searching.TestCall{})), string(searching.CredentialsRejected)+" "+site.AvistaZRefused; got != want {
			t.Errorf("a Test with a wrong PID = %q, want %q", got, want)
		}
	})
}

func TestIPTorrentsTakesItsPastedCookieOnlyFromItsBrowser(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := portedWorld(t)
		settings, cookie := site.Credentials(w.s, site.IPTorrents, "seedbox")
		ix := w.native(site.IPTorrents, settings).pasted(cookie, site.BrowserUserAgent)
		if test := ix.ask(t, searching.TestCall{}); test.Report == nil || test.Report.Items != 2 {
			t.Fatalf("Test = %s, want 2 items", stated(test))
		}
		grabs(t, ix)
		other := w.native(site.IPTorrents, settings).pasted(cookie, "Mozilla/5.0 (another browser)")
		if got := stop(other.ask(t, searching.TestCall{})); got != string(searching.CookieExpired) {
			t.Errorf("a Test with the cookie from another browser = %q, want cookie_expired", got)
		}
		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.IPTorrents})
		if got := stop(ix.movie(t)); got != string(searching.CookieExpired) {
			t.Errorf("a search once the cookie expired = %q, want cookie_expired", got)
		}
	})
}

func TestNorBitsSignsInWithItsAuthenticatorKeysCodeOrOneTypedForATest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := portedWorld(t)
		settings, _ := site.Credentials(w.s, site.NorBits, "seedbox")
		keyed := w.native(site.NorBits, settings)
		if test := keyed.ask(t, searching.TestCall{}); test.Report == nil || test.Report.Items != 2 || test.Session == nil {
			t.Fatalf("a Test with the authenticator key = %s, session %+v; want 2 items and a session", stated(test), test.Session)
		}
		grabs(t, keyed)
		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.NorBits})
		grabs(t, keyed)

		delete(settings, "authenticatorKey")
		typed := w.native(site.NorBits, settings)
		if got := stop(typed.ask(t, searching.TestCall{})); got != string(searching.CodeNeeded) {
			t.Errorf("a Test with neither a key nor a typed code = %q, want code_needed", got)
		}
		if got := stop(typed.ask(t, searching.TestCall{Code: "000000"})); got != string(searching.CredentialsRejected) {
			t.Errorf("a Test with a wrong code = %q, want credentials_rejected", got)
		}
		if test := typed.ask(t, searching.TestCall{Code: site.TwoFactorCode(w.s, time.Now())}); test.Report == nil || test.Report.Items != 2 || test.Session == nil {
			t.Fatalf("a Test with the typed code = %s, session %+v; want 2 items and a session", stated(test), test.Session)
		}
		grabs(t, typed)
		w.f.Cue(t, site.Expire.Name, site.SiteArgs{Site: site.NorBits})
		if got := stop(typed.movie(t)); got != string(searching.CodeNeeded) {
			t.Errorf("a search once the typed code's session expired = %q, want code_needed", got)
		}
		if got, want := signedInWith(t, typed), slices.Repeat([]string{"seedbox code"}, 3); !slices.Equal(got, want) {
			t.Errorf("sign-ins = %v, want %v: the key's two and the typed code's", got, want)
		}
	})
}

func TestTheLeafCoversTheHostsUnderEveryPortedSite(t *testing.T) {
	names := site.Fake.LeafNames()
	for _, e := range all.Registry().Entries() {
		if want := "*." + site.Host(string(e.ID)); !slices.Contains(names, want) {
			t.Errorf("fixture-site's leaf names %v, want %s too", names, want)
		}
	}
}
