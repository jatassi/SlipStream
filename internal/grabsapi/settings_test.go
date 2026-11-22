package grabsapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
)

// values is each setting's value by key, as GET /search-settings reads.
func values(t *testing.T, f *atticFixture) map[string]any {
	t.Helper()
	code, body := f.s.Call(t, http.MethodGet, "/api/v1/search-settings", "")
	var v struct {
		Settings []struct {
			Key   string `json:"key"`
			Value any    `json:"value"`
		} `json:"settings"`
	}
	if err := json.Unmarshal([]byte(body), &v); code != http.StatusOK || err != nil {
		t.Fatalf("getSearchSettings = %d %s", code, body)
	}
	out := map[string]any{}
	for _, s := range v.Settings {
		out[s.Key] = s.Value
	}
	return out
}

// defaults is every Automatic search setting at its default.
var defaults = map[string]any{
	"search.backlogOn": true, "search.backlogUpgrades": true, "tasks.backlog-search.cadence": 28800.0, "search.rssOn": true,
	"tasks.rss-sync.cadence": 900.0, "search.longestGap": 604800.0, "search.fullPace": 1209600.0, "search.stopCount": 3.0, "search.stopWindow": 604800.0,
	"indexers.searchWait": 120.0,
}

func TestASearchSettingsChangeOutsideARangeChangesNothing(t *testing.T) {
	f := searchable(t, site{})
	for _, refused := range []string{
		`{"values":{"search.longestGap":3600}}`,
		`{"values":{"search.longestGap":7862400}}`,
		`{"values":{"search.longestGap":86460}}`,
		`{"values":{"search.fullPace":7862400}}`,
		`{"values":{"search.fullPace":3600}}`,
		`{"values":{"search.stopCount":0}}`,
		`{"values":{"search.stopCount":21}}`,
		`{"values":{"search.stopWindow":60}}`,
		`{"values":{"search.stopWindow":2595600}}`,
		`{"values":{"indexers.searchWait":9}}`,
		`{"values":{"indexers.searchWait":601}}`,
		`{"values":{"tasks.rss-sync.cadence":540}}`,
		`{"values":{"tasks.rss-sync.cadence":7260}}`,
		`{"values":{"search.backlogOn":"off"}}`,
		`{"values":{"search.backlogOn":false,"indexers.searchWait":601}}`,
		`{"values":{"search.stopCount":5,"search.stopWindow":60}}`,
	} {
		if code, body := f.s.Call(t, http.MethodPatch, "/api/v1/search-settings", refused); code != http.StatusUnprocessableEntity {
			t.Errorf("PATCH %s = %d %s, want 422", refused, code, body)
		}
	}
	code, body := f.s.Call(t, http.MethodPatch, "/api/v1/search-settings", `{"values":{"search.rssOn":false,"tasks.other.cadence":60}}`)
	if want := `"code":"unknown_setting"`; code != http.StatusUnprocessableEntity || !strings.Contains(body, want) {
		t.Errorf("an unknown key = %d %s, want 422 %s", code, body, want)
	}
	if !reflect.DeepEqual(values(t, f), defaults) {
		t.Fatalf("after refused changes the settings read %v, want every default %v", values(t, f), defaults)
	}
}

func TestASearchSettingsChangeStoresEveryValueAndNullResetsIt(t *testing.T) {
	f := searchable(t, site{})
	if got := values(t, f); !reflect.DeepEqual(got, defaults) {
		t.Fatalf("getSearchSettings = %v, want %v", got, defaults)
	}
	change := `{"values":{"search.backlogOn":false,"search.backlogUpgrades":false,"search.rssOn":false,"search.longestGap":172800,` +
		`"search.fullPace":0,"search.stopCount":1,"search.stopWindow":3600,"indexers.searchWait":45,"tasks.rss-sync.cadence":1800}}`
	if code, body := f.s.Call(t, http.MethodPatch, "/api/v1/search-settings", change); code != http.StatusOK {
		t.Fatalf("PATCH %s = %d %s", change, code, body)
	}
	want := map[string]any{
		"search.backlogOn": false, "search.backlogUpgrades": false, "search.rssOn": false, "search.longestGap": 172800.0,
		"search.fullPace": 0.0, "search.stopCount": 1.0, "search.stopWindow": 3600.0, "indexers.searchWait": 45.0,
		"tasks.rss-sync.cadence": 1800.0,
	}
	got := values(t, f)
	for key, v := range want {
		if got[key] != v {
			t.Errorf("after the change %s = %v, want %v", key, got[key], v)
		}
	}
	reset := `{"values":{"search.backlogOn":null,"indexers.searchWait":null}}`
	if code, body := f.s.Call(t, http.MethodPatch, "/api/v1/search-settings", reset); code != http.StatusOK {
		t.Fatalf("PATCH %s = %d %s", reset, code, body)
	}
	if got := values(t, f); got["search.backlogOn"] != defaults["search.backlogOn"] || got["indexers.searchWait"] != defaults["indexers.searchWait"] || got["search.rssOn"] == defaults["search.rssOn"] {
		t.Fatalf("after the reset the settings read %v, want Backlog search on, a 120 s wait and RSS sync still off", got)
	}
}

func TestTheWaitForEachIndexerKeepsTheValueTheIndexersSectionStored(t *testing.T) {
	f := searchable(t, site{})
	stored, err := indexers.SearchWait.Parse(new(int64(300)))
	if err != nil {
		t.Fatal(err)
	}
	err = f.lib.Write(f.ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return indexers.SetSearchWaitIn(ctx, tx, stored)
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := values(t, f)["indexers.searchWait"]; got != 300.0 {
		t.Fatalf("indexers.searchWait = %v after the move, want the stored 300", got)
	}
}

func TestTheGapRefusalLandsOnTheFieldThatMoved(t *testing.T) {
	f := searchable(t, site{})
	patch := func(body string) (int, string) { return f.s.Call(t, http.MethodPatch, "/api/v1/search-settings", body) }
	if code, body := patch(`{"values":{"search.longestGap":86400}}`); code != http.StatusOK {
		t.Fatalf("a longest gap of 1 day over an 8 hour interval = %d %s", code, body)
	}
	for _, c := range []struct{ patch, field, words string }{
		{`{"values":{"tasks.backlog-search.cadence":172800}}`, "values.tasks.backlog-search.cadence",
			"Use 1 day or shorter. A file that keeps coming up empty is searched for at least every 1 day."},
		{`{"values":{"search.longestGap":86400,"tasks.backlog-search.cadence":172800}}`, "values.search.longestGap",
			"Use 2 days or longer. Backlog search runs every 2 days."},
	} {
		code, body := patch(c.patch)
		if code != http.StatusUnprocessableEntity || !strings.Contains(body, `"location":"body.`+c.field+`"`) ||
			!strings.Contains(body, c.words) {
			t.Errorf("PATCH %s = %d %s, want 422 on %s saying %q", c.patch, code, body, c.field, c.words)
		}
	}
	if code, body := patch(`{"values":{"tasks.backlog-search.cadence":86400}}`); code != http.StatusOK {
		t.Fatalf("an interval equal to the longest gap = %d %s", code, body)
	}
	if got := values(t, f); got["tasks.backlog-search.cadence"] != 86400.0 || got["search.longestGap"] != 86400.0 {
		t.Fatalf("after the changes the settings read %v, want both at 1 day", got)
	}
}
