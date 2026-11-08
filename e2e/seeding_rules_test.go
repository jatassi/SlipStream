package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

type seedingRule struct {
	Clears      string   `json:"clears"`
	Ratio       *float64 `json:"ratio"`
	SeedSeconds *int64   `json:"seedSeconds"`
	Source      string   `json:"source"`
}

type ruledTracker struct {
	ID   string       `json:"id"`
	Name string       `json:"name"`
	Rule *seedingRule `json:"rule"`
}

// ruled reads every Tracker by name.
func (w *walked) ruled(t *testing.T) map[string]ruledTracker {
	t.Helper()
	var list struct {
		Trackers []ruledTracker `json:"trackers"`
	}
	decode(t, "listTrackers", w.a.Admin("home", http.MethodGet, "/api/v1/trackers", ""), &list)
	out := map[string]ruledTracker{}
	for _, tr := range list.Trackers {
		out[tr.Name] = tr
	}
	return out
}

// addMovie adds the movie key monitored and makes it w's Work.
func (w *walked) addMovie(t *testing.T, key string) {
	t.Helper()
	a := decode(t, "addWork", w.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"`+key+`","preset":"all"}`), &added{})
	var tr tree
	if err := json.Unmarshal(a.Work, &tr); err != nil || !a.Created {
		t.Fatalf("addWork %s created %t, work %s: %v; want a created Work", key, a.Created, a.Work, err)
	}
	w.work = tr.ID
}

// grabWork searches w's Work and waits for its Grab to import.
func (w *walked) grabWork(t *testing.T) string {
	t.Helper()
	s := w.searched(t, w.search(t))
	if s.Outcome == nil || *s.Outcome != "grabbed" {
		t.Fatalf("the search ended %+v, want a Grab", s)
	}
	w.ended(t, *s.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v, want imported", w.grab.Stage, w.grab.Failure)
	}
	return *s.GrabID
}

func expectRule(t *testing.T, when string, tr ruledTracker, want string) {
	t.Helper()
	got, err := json.Marshal(tr.Rule)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s %s's rule is %s, want %s", when, tr.Name, got, want)
	}
}

// TestSeedingGoalsReadEachSiteShapeOfRuleAndAnAdminRuleSticksThroughANewGrab
// grabs The Quiet Orchard from heron, which states a seed time, and Harbor
// Lights from tamarack, which states a ratio or a seed time. Seeding goals
// reads each rule from its Grab. An Admin rule set on Tamarack before its
// Grab outlasts the rule that Grab states, and deleting it puts that rule
// in force.
func TestSeedingGoalsReadEachSiteShapeOfRuleAndAnAdminRuleSticksThroughANewGrab(t *testing.T) {
	t.Parallel()
	w := begin(t, "seeding")
	before := w.ruled(t)
	heron, tamarack := before["Heron"], before["Tamarack"]
	expectRule(t, "before any Grab", heron, "null")
	expectRule(t, "before any Grab", tamarack, "null")

	w.grabWork(t)
	w.runTask(t, "seeding-goals")
	expectRule(t, "once Orchard's Grab is read", w.ruled(t)["Heron"], `{"clears":"time","ratio":null,"seedSeconds":180,"source":"releases"}`)

	admin := `{"clears":"ratio","ratio":2,"seedSeconds":null}`
	if r := w.a.Admin("home", http.MethodPut, "/api/v1/trackers/"+tamarack.ID+"/rule", admin); r.Status != http.StatusOK {
		t.Fatalf("setTrackerRule = %d %s, want 200", r.Status, r.Body)
	}
	w.addMovie(t, "9000002")
	w.grabWork(t)
	w.runTask(t, "seeding-goals")
	expectRule(t, "after Harbor's Grab", w.ruled(t)["Tamarack"], `{"clears":"ratio","ratio":2,"seedSeconds":null,"source":"admin"}`)

	cleared := decode(t, "clearTrackerRule", w.a.Admin("home", http.MethodDelete, "/api/v1/trackers/"+tamarack.ID+"/rule", ""), &ruledTracker{})
	expectRule(t, "once the Admin's rule is deleted", *cleared, `{"clears":"either","ratio":1,"seedSeconds":180,"source":"releases"}`)
	expectRule(t, "after both Grabs", w.ruled(t)["Heron"], `{"clears":"time","ratio":null,"seedSeconds":180,"source":"releases"}`)
}

type timedTorrent struct {
	Seeding struct {
		State string `json:"state"`
		Time  struct {
			Value int64  `json:"value"`
			Rule  *int64 `json:"rule"`
		} `json:"time"`
	} `json:"seeding"`
}

// TestATimeRuleIsMetOnceTheTrackerCreditsItsSeedTimeAndTheMargin grabs The
// Quiet Orchard from heron, whose rule is 3 minutes of seeding. Its window
// owes until the seed time seedbox counts from heron's answers passes the
// rule and the 1-minute margin, and then reads met.
func TestATimeRuleIsMetOnceTheTrackerCreditsItsSeedTimeAndTheMargin(t *testing.T) {
	t.Parallel()
	w := begin(t, "seeding")
	torrent := *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+w.grabWork(t), ""), &trackedGrab{}).TorrentID
	w.torrentIn(t, torrent, "seeding")
	var got timedTorrent
	decode(t, "getTorrent", w.a.Admin("home", http.MethodGet, "/api/v1/torrents/"+torrent, ""), &got)
	if got.Seeding.State != "owing" || got.Seeding.Time.Rule == nil || *got.Seeding.Time.Rule != 240 {
		t.Fatalf("seeding Orchard reads %+v, want owing heron's 3 minutes and the 1-minute margin", got.Seeding)
	}

	worldtest.Until(t, "Orchard's window is met", func() (bool, error) {
		decode(t, "getTorrent", w.a.Admin("home", http.MethodGet, "/api/v1/torrents/"+torrent, ""), &got)
		return got.Seeding.State == "met", nil
	})
	if got.Seeding.Time.Value < 240 {
		t.Fatalf("met Orchard reads %d seconds of seeding, want at least the rule's and the margin's 240", got.Seeding.Time.Value)
	}
}
