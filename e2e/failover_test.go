package e2e

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// indexerView is what the failover tests read of an Indexer.
type indexerView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	HomeNodeID    string `json:"homeNodeId"`
	AllowFailover bool   `json:"allowFailover"`
	SignsIn       bool   `json:"signsIn"`
	Status        struct {
		State  string  `json:"state"`
		Cause  *string `json:"cause"`
		NodeID *string `json:"nodeId"`
	} `json:"status"`
	Waits *struct {
		Kind   string `json:"kind"`
		Lapsed bool   `json:"lapsed"`
	} `json:"waits"`
	Session *struct {
		NodeID string `json:"nodeId"`
	} `json:"session"`
}

// indexers is every Indexer by name.
func (w *walked) indexers(t *testing.T) map[string]indexerView {
	t.Helper()
	var list struct {
		Indexers []indexerView `json:"indexers"`
	}
	decode(t, "listIndexers", w.a.Admin("home", http.MethodGet, "/api/v1/indexers", ""), &list)
	out := map[string]indexerView{}
	for _, ix := range list.Indexers {
		out[ix.Name] = ix
	}
	return out
}

// searchAnother adds the movie key, monitored, and returns each Indexer's
// line in its finished search as its outcome and sentence, by name.
func (w *walked) searchAnother(t *testing.T, key string) map[string]string {
	t.Helper()
	first := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`","monitor":true}`), &added{})
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created {
		t.Fatalf("addWork %s created %t: %v; want a created Work", key, first.Created, err)
	}
	w.work = tr.ID
	lines := map[string]string{}
	for _, l := range w.searched(t, w.search(t)).Indexers {
		lines[l.Name] = *l.Outcome + ": " + l.Sentence
	}
	return lines
}

func expectLine(t *testing.T, when string, lines map[string]string, name, want string) {
	t.Helper()
	if lines[name] != want {
		t.Errorf("%s %s's line is %q, want %q", when, name, lines[name], want)
	}
}

// lapse sets the offline grace to its 1 min floor, stops seedbox and waits
// until Larkspur waits for it past the grace.
func (w *walked) lapse(t *testing.T) {
	t.Helper()
	if r := w.a.Admin("home", http.MethodPatch, "/api/v1/nodes/settings", `{"offlineGrace":60}`); r.Status != http.StatusOK {
		t.Fatalf("updateNodeSettings = %d %s, want 200", r.Status, r.Body)
	}
	act(t, w.run, "seedbox", world.StopNode)
	worldtest.Until(t, "Larkspur waits for seedbox past the grace", func() (bool, error) {
		waits := w.indexers(t)["Larkspur"].Waits
		return waits != nil && waits.Lapsed, nil
	})
}

// expectWaitingWarns waits until Health warns of each Indexer in signsIn
// waiting for seedbox past the grace, linking to its Allow failover, and of
// no other Indexer.
func (w *walked) expectWaitingWarns(t *testing.T, signsIn ...string) {
	t.Helper()
	ids := w.indexers(t)
	var wrong []string
	worldtest.Until(t, "Health warns of "+strings.Join(signsIn, ", ")+" waiting for seedbox and of no other Indexer", func() (bool, error) {
		items := healthItems(t, w.a, "indexers")
		wrong = nil
		for name, ix := range ids {
			_, warned := items["waiting."+ix.ID]
			if warned != slices.Contains(signsIn, name) {
				wrong = append(wrong, name)
			}
		}
		return len(wrong) == 0, nil
	})
	items := healthItems(t, w.a, "indexers")
	for _, name := range signsIn {
		ix := ids[name]
		it := items["waiting."+ix.ID]
		if it.Level != "warn" || it.Action == nil || it.Action.Href != "/settings/indexers/"+ix.ID+"#allow-failover" || it.Subject != name ||
			!strings.HasPrefix(it.Text, name+" is waiting for seedbox, which has been offline since ") ||
			!strings.HasSuffix(it.Text, ". Allow failover to search from another Node.") {
			t.Errorf("Health's item for %s waiting is %+v, want a Warn to allow failover linking to it", name, it)
		}
	}
}

// TestALoginSessionIndexerWaitsForItsLapsedHomeUntilItAllowsFailover
// stops seedbox, the home of four catalog Indexers, past the grace while
// home holds Searching. Quarry, which keeps no session, fails over to
// home; Larkspur, Marsh and Cinder wait for seedbox and Health warns of
// each. Once they allow failover, Marsh signs in and searches from home,
// while Larkspur's captcha and Cinder's browser challenge on home make
// them wait for seedbox again, and the next search leaves them waiting
// without asking home.
func TestALoginSessionIndexerWaitsForItsLapsedHomeUntilItAllowsFailover(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	home := w.a.Node("home").NodeID
	w.lapse(t)
	w.expectWaitingWarns(t, "Larkspur", "Marsh", "Cinder")

	lines := w.searchAnother(t, "9000002")
	expectLine(t, "with seedbox lapsed", lines, "Quarry", "found: Found 1 Release")
	expectLine(t, "with seedbox lapsed", lines, "Fixture Site", "found: Found 1 Release")
	for _, name := range []string{"Larkspur", "Marsh", "Cinder"} {
		expectLine(t, "with seedbox lapsed", lines, name, "waiting: Waiting for seedbox")
	}

	before := w.seq(t)
	ids := w.indexers(t)
	for _, name := range []string{"Larkspur", "Marsh", "Cinder"} {
		if r := w.a.Admin("home", http.MethodPatch, "/api/v1/indexers/"+ids[name].ID, `{"allowFailover":true}`); r.Status != http.StatusOK {
			t.Fatalf("updateIndexer %s allowing failover = %d %s, want 200", name, r.Status, r.Body)
		}
	}
	w.expectWaitingWarns(t)

	lines = w.searchAnother(t, "9000003")
	expectLine(t, "once they allow failover", lines, "Marsh", "found: Found 1 Release")
	expectLine(t, "once they allow failover", lines, "Larkspur", "waiting: Waiting for seedbox")
	expectLine(t, "once they allow failover", lines, "Cinder", "waiting: Waiting for seedbox")
	if n := w.recorded(t, site.SignedIn.Kind, site.Marsh, before); n != 1 {
		t.Errorf("Marsh signed in from home %d times, want once", n)
	}
	ids = w.indexers(t)
	for name, cause := range map[string]string{"Larkspur": "captcha_needed", "Cinder": "challenge_unsolved"} {
		st := ids[name].Status
		if st.State != "home_only" || st.Cause == nil || *st.Cause != cause || st.NodeID == nil || *st.NodeID != home {
			t.Errorf("%s's status = %+v, want home_only for %s met on home", name, st, cause)
		}
	}

	again, asked := w.seq(t), w.queries(t, ids["Larkspur"].ID)+w.queries(t, ids["Cinder"].ID)
	lines = w.searchAnother(t, "9000004")
	expectLine(t, "while they wait for seedbox", lines, "Marsh", "found: Found 1 Release")
	expectLine(t, "while they wait for seedbox", lines, "Larkspur", "waiting: Waiting for seedbox")
	expectLine(t, "while they wait for seedbox", lines, "Cinder", "waiting: Waiting for seedbox")
	if n := w.queries(t, ids["Larkspur"].ID) + w.queries(t, ids["Cinder"].ID) - asked; n != 0 {
		t.Errorf("the search after asked Larkspur and Cinder %d more times, want none while they wait for seedbox", n)
	}
	if n := w.recorded(t, site.SignedIn.Kind, site.Marsh, again); n != 0 {
		t.Errorf("Marsh signed in from home %d more times, want none: its session on home holds", n)
	}
}

// queries counts Indexer id's query rows.
func (w *walked) queries(t *testing.T, id string) int {
	t.Helper()
	var list struct {
		Queries []queryView `json:"queries"`
	}
	decode(t, "listIndexerQueries", w.a.Admin("home", http.MethodGet, "/api/v1/indexers/"+id+"/queries", ""), &list)
	return len(list.Queries)
}

// TestWhileTheControllerOnlyCoversSearchingASessionWaitsAndAWriteOffDeletesSeedboxsSessions
// takes Searching from home, so its feeds move to seedbox, then stops
// seedbox past the grace. home covers Searching: Quarry and Fixture Site
// search from it, and the Indexers that sign in wait. Writing seedbox off
// moves every Indexer to home and deletes the sessions they held there.
func TestWhileTheControllerOnlyCoversSearchingASessionWaitsAndAWriteOffDeletesSeedboxsSessions(t *testing.T) {
	t.Parallel()
	w := begin(t, "indexers")
	home, box := w.a.Node("home").NodeID, w.a.Node("seedbox").NodeID
	w.a.SetRoles("home", home, "storage", "downloading", "converting", "subtitles", "metadata", "media-server-link", "notifying", "serving")
	signedIn := []string{"Larkspur", "Marsh", "Cinder"}
	for name, ix := range w.indexers(t) {
		if ix.HomeNodeID != box {
			t.Errorf("%s is homed on %s once home let Searching go, want seedbox %s", name, ix.HomeNodeID, box)
		}
		if slices.Contains(signedIn, name) && (ix.Session == nil || ix.Session.NodeID != box) {
			t.Errorf("%s's session = %+v, want one on seedbox", name, ix.Session)
		}
	}
	w.lapse(t)
	w.expectWaitingWarns(t, signedIn...)

	lines := w.searchAnother(t, "9000002")
	expectLine(t, "with home covering", lines, "Quarry", "found: Found 1 Release")
	expectLine(t, "with home covering", lines, "Fixture Site", "found: Found 1 Release")
	for _, name := range signedIn {
		expectLine(t, "with home covering", lines, name, "waiting: Waiting for seedbox")
	}

	worldtest.Expect(t, "removeNode seedbox writing its work off", w.a.Admin("home", http.MethodDelete, "/api/v1/nodes/"+box+"?writeOff=true", ""), worldtest.Reply{Status: http.StatusNoContent})
	for name, ix := range w.indexers(t) {
		if ix.HomeNodeID != home || ix.Session != nil {
			t.Errorf("%s after seedbox's write-off is homed on %s with session %+v, want home and no session", name, ix.HomeNodeID, ix.Session)
		}
	}
}

// seq is the newest record of the run's event log.
func (w *walked) seq(t *testing.T) uint64 {
	t.Helper()
	recs, err := events.Read(w.m.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	return recs[len(recs)-1].Seq
}

// recorded counts the records of kind for label that fixture-site wrote
// on home's front after seq.
func (w *walked) recorded(t *testing.T, kind, label string, after uint64) int {
	t.Helper()
	recs, err := events.Select(w.m.RunDir, events.Query{Kind: kind, Node: "home", Match: map[string]string{"site": label}, After: after})
	if err != nil {
		t.Fatal(err)
	}
	return len(recs)
}
