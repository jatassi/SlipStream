package e2e

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// TestANativeIndexerOfEachFamilySearchesAndGrabsThroughItsPort searches
// each of ported-sites' five movies with every native Indexer, the one
// of a new family each time at priority 10 above the others' 25. Each
// search finds the movie's release on all five, and its Grab comes from
// the Indexer at 10, fetched through its port, and imports.
func TestANativeIndexerOfEachFamilySearchesAndGrabsThroughItsPort(t *testing.T) {
	t.Parallel()
	w := begin(t, "ported-sites")
	ids := w.indexers(t)
	previous := ""
	for i, name := range []string{"PassThePopcorn", "Torrent Bytes", "AvistaZ", "IPTorrents", "NorBits"} {
		if previous != "" {
			w.prioritize(t, ids[previous].ID, 25)
			w.addWork(t, strconv.Itoa(9000001+i))
		}
		w.prioritize(t, ids[name].ID, 10)
		w.grabsFrom(t, name, ids[name].ID, len(ids))
		previous = name
	}
}

func (w *walked) prioritize(t *testing.T, id string, priority int) {
	t.Helper()
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/indexers/"+id, `{"priority":`+strconv.Itoa(priority)+`}`); r.Status != http.StatusOK {
		t.Fatalf("updateIndexer %s to priority %d = %d %s, want 200", id, priority, r.Status, r.Body)
	}
}

// addWork adds the movie key, monitored, as w's Work.
func (w *walked) addWork(t *testing.T, key string) {
	t.Helper()
	made := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(made.Work, &tr); err != nil || !made.Created {
		t.Fatalf("addWork %s created %t: %v; want a created Work", key, made.Created, err)
	}
	w.work = tr.ID
}

// grabsFrom searches w's Work, which each of the indexers Indexers finds
// once, and checks its Grab came from Indexer id, named name, and
// imported.
func (w *walked) grabsFrom(t *testing.T, name, id string, indexers int) {
	t.Helper()
	found := w.searched(t, w.search(t))
	for _, l := range found.Indexers {
		if l.Outcome == nil || *l.Outcome != "found" || l.Releases != 1 {
			t.Errorf("%s's search ended %v with %d Releases: %s, want its one release found", l.Name, l.Outcome, l.Releases, l.Sentence)
		}
	}
	if len(found.Indexers) != indexers || found.GrabID == nil {
		t.Fatalf("the search = %+v, want a line for each of the %d Indexers and a Grab", found, indexers)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("%s's Grab ended %s with %+v after %v, want imported", name, w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	if from := w.grabbedFrom(t, w.grab.ID); from != id {
		t.Errorf("the Grab came from Indexer %s, want %s's %s", from, name, id)
	}
}

// portPin is what the ported sites tests read of a native Indexer's port.
type portPin struct {
	Port struct {
		BaseAddress string            `json:"baseAddress"`
		Values      map[string]string `json:"values"`
		Sealed      []string          `json:"sealed"`
	} `json:"port"`
}

// refusedBecause checks r is a failed Test for cause that says detail.
func refusedBecause(t *testing.T, what string, r worldtest.Reply, cause, detail string) {
	t.Helper()
	var p struct {
		Code   string `json:"code"`
		Detail string `json:"detail"`
		Cause  string `json:"cause"`
	}
	if err := json.Unmarshal([]byte(r.Body), &p); err != nil || r.Status != http.StatusUnprocessableEntity || p.Code != "indexer_test_failed" || p.Cause != cause || p.Detail != detail {
		t.Fatalf("%s = %d %s, want a failed Test for %s saying %q", what, r.Status, r.Body, cause, detail)
	}
}

// expectStopItem waits for Indexer id, named name, to raise its stop's
// Fail saying text and linking to its page at fragment.
func (w *walked) expectStopItem(t *testing.T, id, name, text, fragment string) {
	t.Helper()
	worldtest.Until(t, name+"'s stop raises its item", func() (bool, error) {
		_, open := healthItems(t, w.a, "indexers")["stopped."+id]
		return open, nil
	})
	if it := healthItems(t, w.a, "indexers")["stopped."+id]; it.Level != "fail" || it.Subject != name || it.Text != text ||
		it.Action == nil || it.Action.Href != "/settings/indexers/"+id+fragment {
		t.Errorf("%s's stop item is %+v, want a Fail saying %q linking to its page at %q", name, it, text, fragment)
	}
}

// TestNorBitsMakesItsOwnCodeFromTheKeyOrStopsForOneTypedOnItsTest adds
// a second NorBits on seedbox with the authenticator key left out, whose
// Test needs the current code and saves with it. Once NorBits signs every
// session out, the next search signs the first in again with a code made
// from its key, and stops the second, which raises its Fail. Its Test
// needs the code again, and a Test with it clears the stop.
func TestNorBitsMakesItsOwnCodeFromTheKeyOrStopsForOneTypedOnItsTest(t *testing.T) {
	t.Parallel()
	w := begin(t, "ported-sites")
	keyed := w.indexers(t)["NorBits"].ID
	saved := decode(t, "getIndexer", w.a.Admin("home", http.MethodGet, "/api/v1/indexers/"+keyed, ""), &portPin{}).Port
	if !slices.Contains(saved.Sealed, "authenticatorKey") || saved.Values["authenticatorKey"] != "" {
		t.Errorf("NorBits' port = %+v, want the authenticator key sealed and never shown", saved)
	}
	needsCode := "NorBits by code needs a two-factor code to sign in again. Type the current code and test again."
	draft := `{"kind":"native","name":"NorBits by code","port":"norbits","baseAddress":"` + saved.BaseAddress + `","homeNodeId":"` + w.a.Node("seedbox").NodeID +
		`","values":{"username":"` + site.Username + `","password":"` + site.Password(w.s, site.NorBits) + `"}`
	refusedBecause(t, "testIndexer with no code", w.a.Admin("home", http.MethodPost, "/api/v1/indexers/test", draft+"}"), "code_needed", needsCode)
	reply := w.a.Admin("home", http.MethodPost, "/api/v1/indexers", draft+`,"code":"`+site.TwoFactorCode(w.s, time.Now())+`"}`)
	if reply.Status != http.StatusCreated {
		t.Fatalf("createIndexer with the code = %d %s, want 201", reply.Status, reply.Body)
	}
	reply.Status = http.StatusOK
	typed := decode(t, "createIndexer", reply, &struct {
		ID string `json:"id"`
		portPin
	}{})
	if !slices.Equal(typed.Port.Sealed, []string{"password"}) {
		t.Errorf("NorBits by code seals %v, want only its password", typed.Port.Sealed)
	}

	cue(t, w.run, site.Expire.Name, `{"site":"`+site.NorBits+`"}`)
	lines := w.searchAnother(t, "9000002")
	expectLine(t, "once NorBits signs every session out", lines, "NorBits", "found: Found 1 Release")
	expectLine(t, "once NorBits signs every session out", lines, "NorBits by code", "stopped: Stopped because signing in again needs a two-factor code.")
	w.expectStopItem(t, typed.ID, "NorBits by code", "NorBits by code needs a two-factor code to sign in again. Open the Indexer and type the current code.", "#code")

	path := "/api/v1/indexers/" + typed.ID + "/test"
	refusedBecause(t, "testIndexer NorBits by code with no code", w.a.Admin("home", http.MethodPost, path, ""), "code_needed", needsCode)
	if r := w.a.Admin("home", http.MethodPost, path, `{"code":"`+site.TwoFactorCode(w.s, time.Now())+`"}`); r.Status != http.StatusOK {
		t.Fatalf("testIndexer NorBits by code with the code = %d %s, want 200", r.Status, r.Body)
	}
	worldtest.Until(t, "the passing Test clears the stop", func() (bool, error) {
		_, open := healthItems(t, w.a, "indexers")["stopped."+typed.ID]
		return !open, nil
	})
}

// TestAPortWithNoSessionFailsOverWhileOneThatSignsInWaitsUntilItAllowsFailover
// moves PassThePopcorn to seedbox beside IPTorrents and NorBits, then
// stops seedbox past the grace. PassThePopcorn, which sends its key on
// every request, searches from home, while IPTorrents and NorBits wait
// for seedbox. Once NorBits allows failover it signs in from home with
// its authenticator key.
func TestAPortWithNoSessionFailsOverWhileOneThatSignsInWaitsUntilItAllowsFailover(t *testing.T) {
	t.Parallel()
	w := begin(t, "ported-sites")
	ids := w.indexers(t)
	w.patchIndexer(t, ids["PassThePopcorn"].ID, `{"homeNodeId":"`+w.a.Node("seedbox").NodeID+`"}`)
	w.lapse(t, "NorBits")
	w.expectWaitingWarns(t, "IPTorrents", "NorBits")

	lines := w.searchAnother(t, "9000002")
	expectLine(t, "with seedbox lapsed", lines, "PassThePopcorn", "found: Found 1 Release")
	expectLine(t, "with seedbox lapsed", lines, "IPTorrents", "waiting: Waiting for seedbox")
	expectLine(t, "with seedbox lapsed", lines, "NorBits", "waiting: Waiting for seedbox")

	before := w.seq(t)
	w.patchIndexer(t, ids["NorBits"].ID, `{"allowFailover":true}`)
	w.expectWaitingWarns(t, "IPTorrents")
	lines = w.searchAnother(t, "9000003")
	expectLine(t, "once NorBits allows failover", lines, "NorBits", "found: Found 1 Release")
	expectLine(t, "once NorBits allows failover", lines, "IPTorrents", "waiting: Waiting for seedbox")
	if n := w.recorded(t, site.SignedIn.Kind, site.NorBits, before); n != 1 {
		t.Errorf("NorBits signed in from home %d times, want once", n)
	}
}

// TestAPortSignsInAgainAtASignedOutPageStopsAtADeadCookieAndBacksOffAnOutage
// signs TorrentBytes out and stops IPTorrents' pasted cookie. The next
// search signs Torrent Bytes in again once and finds the Release, and
// stops IPTorrents, which raises its Fail. AvistaZ in maintenance answers
// 503, and its line backs off. PassThePopcorn's search limit starts at
// its port's 150 an hour.
func TestAPortSignsInAgainAtASignedOutPageStopsAtADeadCookieAndBacksOffAnOutage(t *testing.T) {
	t.Parallel()
	w := begin(t, "ported-sites")
	ids := w.indexers(t)
	if s := w.standing(t, ids["PassThePopcorn"].ID).Searches; s.Count != 150 || s.Per != "hour" {
		t.Errorf("PassThePopcorn's search limit is %+v, want its port's 150 an hour", s)
	}
	before := w.seq(t)
	cue(t, w.run, site.Expire.Name, `{"site":"`+site.TorrentBytes+`"}`)
	cue(t, w.run, site.Expire.Name, `{"site":"`+site.IPTorrents+`"}`)
	cue(t, w.run, site.Maintenance.Name, `{"site":"`+site.AvistaZ+`","on":true}`)

	lines := w.searchAnother(t, "9000002")
	expectLine(t, "once Torrent Bytes is signed out", lines, "Torrent Bytes", "found: Found 1 Release")
	if n := w.recorded(t, site.SignedIn.Kind, site.TorrentBytes, before); n != 1 {
		t.Errorf("Torrent Bytes signed in %d times at the signed-out page, want once", n)
	}
	expectLine(t, "once its cookie stops working", lines, "IPTorrents", "stopped: Stopped because the browser cookie is no longer accepted.")
	w.expectStopItem(t, ids["IPTorrents"].ID, "IPTorrents", "IPTorrents no longer accepts the browser cookie. Paste a new one and that browser's User-Agent on the Indexer.", "")
	if line := lines["AvistaZ"]; !strings.HasPrefix(line, "failing: Isn't responding. SlipStream retries at ") {
		t.Errorf("AvistaZ's line in maintenance is %q, want failing until its backoff ends", line)
	}
}

// TestANativeIndexersEditTestsTheChangeFirstAndKeepsASealedSettingSentEmpty
// edits Torrent Bytes' password to a wrong one, which its Test refuses
// and the edit doesn't save, then sends it empty, which keeps the saved
// one: signed out, Torrent Bytes signs in again with it. A cookie pasted
// with another browser's User-Agent fails IPTorrents' Test, and the
// cookie with its own browser's saves.
func TestANativeIndexersEditTestsTheChangeFirstAndKeepsASealedSettingSentEmpty(t *testing.T) {
	t.Parallel()
	w := begin(t, "ported-sites")
	ids := w.indexers(t)
	bytes, ipt := "/api/v1/indexers/"+ids["Torrent Bytes"].ID, "/api/v1/indexers/"+ids["IPTorrents"].ID
	refusedBecause(t, "updateIndexer Torrent Bytes with a wrong password", w.a.Admin("home", http.MethodPatch, bytes, `{"values":{"password":"wrong"}}`),
		"credentials_rejected", "Torrent Bytes refused the sign-in with '"+site.TorrentBytesRefused+"'. Check the username, password and any code, then test again.")
	w.patchIndexer(t, ids["Torrent Bytes"].ID, `{"values":{"password":""}}`)

	cookie := func(userAgent string) string {
		return `{"cookie":{"cookie":"` + site.IPTorrentsCookie(w.s) + `","userAgent":"` + userAgent + `"}}`
	}
	refusedBecause(t, "updateIndexer IPTorrents with another browser's User-Agent", w.a.Admin("home", http.MethodPatch, ipt, cookie("Mozilla/5.0 (another)")),
		"cookie_expired", "IPTorrents no longer accepts the browser cookie. Paste a new one with that browser's User-Agent and test again.")
	w.patchIndexer(t, ids["IPTorrents"].ID, cookie(site.BrowserUserAgent))

	cue(t, w.run, site.Expire.Name, `{"site":"`+site.TorrentBytes+`"}`)
	lines := w.searchAnother(t, "9000002")
	expectLine(t, "after the edits", lines, "Torrent Bytes", "found: Found 1 Release")
	expectLine(t, "after the edits", lines, "IPTorrents", "found: Found 1 Release")
}
