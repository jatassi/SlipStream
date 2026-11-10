package grabsapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
)

func TestStartManualSearchAnswersTheWorksSearchUntilSearchAgain(t *testing.T) {
	f, _, _ := mirrored(t, "The.Attic.1080p.WEB", "The.Attic.720p.WEB", false, indexers.Edit{})
	f.storageNodes.lapsed.Store(true) // an offline root doesn't stop a Manual search
	path := "/api/v1/works/" + f.tree.ID.String() + "/manual-search"
	start := func(body string, want int) string {
		t.Helper()
		code, got := f.s.Call(t, http.MethodPost, path, body)
		var started struct {
			ManualSearchID string `json:"manualSearchId"`
		}
		if err := json.Unmarshal([]byte(got), &started); err != nil || code != want || started.ManualSearchID == "" {
			t.Fatalf("startManualSearch %s = %d %s, want %d with an id", body, code, got, want)
		}
		return started.ManualSearchID
	}
	first := start("", http.StatusAccepted)
	if ev := <-f.searched; ev.ManualSearchID == nil || *ev.ManualSearchID != first || ev.WorkID != f.tree.ID.String() {
		t.Fatalf("starting published %+v, want the run of Manual search %s", ev, first)
	}
	if again := start("{}", http.StatusOK); again != first {
		t.Fatalf("the same scope answered %s, want the Work's Manual search %s", again, first)
	}
	if again := start(`{"again":true}`, http.StatusAccepted); again == first {
		t.Fatal("Search again answered the Manual search it replaces")
	}
	if typed := start(`{"query":"attic remux"}`, http.StatusAccepted); typed == first {
		t.Fatal("a typed query answered the Manual search without one")
	}
}
