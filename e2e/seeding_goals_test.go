package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

const orchardRelease = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"

// grabbedTorrent grabs w's Work and answers the Torrent its Grab
// imported from, once its Node reports it seeding.
func (w *walked) grabbedTorrent(t *testing.T) string {
	t.Helper()
	torrent := *decode(t, "getGrab", w.a.Admin("home", http.MethodGet, "/api/v1/grabs/"+w.grabWork(t), ""), &trackedGrab{}).TorrentID
	w.torrentIn(t, torrent, "seeding")
	return torrent
}

// gone waits until the Controller no longer holds Torrent id: its Node
// removed it and confirmed.
func (w *walked) gone(t *testing.T, what, id string) {
	t.Helper()
	worldtest.Until(t, what, func() (bool, error) {
		return w.a.Admin("home", http.MethodGet, "/api/v1/torrents/"+id, "").Status == http.StatusNotFound, nil
	})
}

// stoppedSeeding waits for fake-tracker's record of release from the
// announce that stopped seeding it, the first that reads it inactive.
func (w *walked) stoppedSeeding(t *testing.T, release string) tracker.History {
	t.Helper()
	var stopped tracker.History
	worldtest.Until(t, "fake-tracker records "+release+" stopped", func() (bool, error) {
		recs, err := events.Select(w.m.RunDir, events.Query{Kind: tracker.Seeding.Kind, Match: map[string]string{"release": release}})
		if err != nil {
			return false, err
		}
		for _, r := range recs {
			if err := json.Unmarshal(r.Data, &stopped); err != nil {
				return false, fmt.Errorf("decode a seeding record: %w", err)
			}
			if !stopped.Active {
				return true, nil
			}
		}
		return false, nil
	})
	return stopped
}

// TestAGoalRemovesATorrentOnceItsRuleAndMarginAreMetAndARegrabMeetsItAgain
// gives Heron a goal of 1 minute of seeding then remove, far short of its
// 3-minute rule and the 1-minute margin. While The Quiet Orchard owes
// Heron, removing it is previewed as a hit and run. The goal removes it
// only once its window is met, and fake-tracker had credited the rule's 3
// minutes when it stopped. Grabbed again, the new Torrent carries the seed
// time it had, so its goal removes it again.
func TestAGoalRemovesATorrentOnceItsRuleAndMarginAreMetAndARegrabMeetsItAgain(t *testing.T) {
	t.Parallel()
	w := begin(t, "seeding")
	heron := w.ruled(t)["Heron"].ID
	if r := w.a.Admin("home", http.MethodPut, "/api/v1/trackers/"+heron+"/goal", `{"combine":"first","ratio":null,"seedSeconds":60,"action":"remove"}`); r.Status != http.StatusOK {
		t.Fatalf("setTrackerGoal = %d %s, want 200", r.Status, r.Body)
	}
	torrent := w.grabbedTorrent(t)

	var dry struct {
		Preview struct {
			HitAndRun []struct {
				TorrentID string `json:"torrentId"`
				Tracker   *struct {
					Name string `json:"name"`
				} `json:"tracker"`
			} `json:"hitAndRun"`
		} `json:"preview"`
	}
	decode(t, "torrentActions", w.a.Admin("home", http.MethodPost, "/api/v1/torrents/actions", `{"action":"remove","ids":["`+torrent+`"],"dryRun":true}`), &dry)
	if h := dry.Preview.HitAndRun; len(h) != 1 || h[0].TorrentID != torrent || h[0].Tracker == nil || h[0].Tracker.Name != "Heron" {
		t.Fatalf("removing Orchard while it owes Heron previews hit and runs %+v, want Orchard's on Heron", h)
	}

	w.gone(t, "Orchard's goal removes it", torrent)
	if win := w.window(t, heron); win.State != "met" {
		t.Fatalf("once its goal removed it Orchard's window is %+v, want met", win)
	}
	if h := w.stoppedSeeding(t, orchardRelease); h.SeedSeconds < 180 || h.Owes || h.HitAndRun {
		t.Fatalf("when Orchard stopped seeding fake-tracker recorded %+v, want at least Heron's 180 seconds and nothing owed", h)
	}

	if r := w.a.Admin("home", http.MethodPost, "/api/v1/works/delete", `{"workIds":["`+w.work+`"],"files":"delete"}`); r.Status != http.StatusOK {
		t.Fatalf("deleteWorks = %d %s, want 200", r.Status, r.Body)
	}
	w.addMovie(t, "9000001")
	again := w.grabbedTorrent(t)
	if again == torrent {
		t.Fatalf("grabbing Orchard again reused Torrent %s, want a new one", torrent)
	}
	w.gone(t, "Orchard's goal removes it again", again)
}
