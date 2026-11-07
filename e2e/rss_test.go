package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

func TestHomeAddsBrooksRSSFeedWithItsCookie(t *testing.T) {
	t.Parallel()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	run := worldtest.LaunchScenario(t, s)
	a := api(t, run)
	feed := "https://" + site.Host(site.Brook) + site.RSSPath
	body := func(address, cookie string) string {
		b, err := json.Marshal(map[string]any{
			"kind": "rss", "name": "Brook", "address": address, "trackerName": "Brook", "media": []string{"movie"},
			"cookie": map[string]string{"cookie": cookie, "userAgent": "Pasted/1.0"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	for _, tc := range []struct{ address, cookie, want string }{
		{feed + "?play=" + site.PlayRepeatedGUID, site.Cookie(s), `{"status":422,"code":"indexer_test_failed","title":"The Indexer's Test failed","detail":"Some items repeat the same GUID."}`},
		{feed, site.CookieName + "=stale", `{"status":422,"code":"indexer_test_failed","title":"The Indexer's Test failed","detail":"Brook no longer accepts the browser cookie. Paste a new one and test again."}`},
	} {
		worldtest.Expect(t, "createIndexer", a.Admin("home", http.MethodPost, "/api/v1/indexers", body(tc.address, tc.cookie)), worldtest.Reply{Status: 422, Body: tc.want + "\n"})
	}

	var created struct {
		Kind     string   `json:"kind"`
		Media    []string `json:"media"`
		FeedHost string   `json:"feedHost"`
		Sentence string   `json:"sentence"`
		Session  *struct {
			Pasted bool `json:"pasted"`
		} `json:"session"`
	}
	reply := a.Admin("home", http.MethodPost, "/api/v1/indexers", body(feed, site.Cookie(s)))
	if reply.Status != http.StatusCreated {
		t.Fatalf("createIndexer = %d %s, want 201", reply.Status, reply.Body)
	}
	if err := json.Unmarshal([]byte(reply.Body), &created); err != nil {
		t.Fatal(err)
	}
	if created.Kind != "rss" || len(created.Media) != 1 || created.Media[0] != "movie" || created.FeedHost != site.Host(site.Brook) ||
		created.Sentence != "Brook is working." || created.Session == nil || !created.Session.Pasted {
		t.Fatalf("createIndexer = %s, want Brook's RSS feed carrying movies, working, with its pasted session", reply.Body)
	}
}
