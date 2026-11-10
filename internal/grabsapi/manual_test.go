package grabsapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/storage"
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

func TestGetManualSearchPlansTheWorksManualSearch(t *testing.T) {
	f, _, _ := mirrored(t, "The.Attic.1080p.WEB", "The.Attic.720p.WEB", false, indexers.Edit{})
	path := "/api/v1/works/" + f.tree.ID.String() + "/manual-search"
	if code, got := f.s.Call(t, http.MethodGet, path, ""); code != http.StatusNotFound || !strings.Contains(got, `"code":"no_manual_search"`) {
		t.Fatalf("getManualSearch before any = %d %s, want 404 no_manual_search", code, got)
	}
	if code, got := f.s.Call(t, http.MethodPost, path, ""); code != http.StatusAccepted {
		t.Fatalf("startManualSearch = %d %s, want 202", code, got)
	}
	f.running(t)
	type view struct {
		Searching bool   `json:"searching"`
		Line      string `json:"line"`
		Plan      struct {
			Headline string `json:"headline"`
			Rows     []struct {
				Key  string `json:"key"`
				Pick *struct {
					Release struct {
						Title string `json:"title"`
					} `json:"release"`
				} `json:"pick"`
			} `json:"rows"`
		} `json:"plan"`
		Folded struct {
			Line string `json:"line"`
		} `json:"folded"`
		ReleaseCount int `json:"releaseCount"`
	}
	var v view
	for v.Line == "" || v.Searching {
		<-f.searched
		code, got := f.s.Call(t, http.MethodGet, path+"?pins=not-a-pin", "")
		if err := json.Unmarshal([]byte(got), &v); err != nil || code != http.StatusOK {
			t.Fatalf("getManualSearch = %d %s, want 200", code, got)
		}
	}
	if v.Line != "SlipStream searched 2 Indexers." || v.Plan.Headline != "1 Grab gets you everything." || len(v.Plan.Rows) != 1 ||
		v.Plan.Rows[0].Pick == nil || v.Plan.Rows[0].Pick.Release.Title != "The.Attic.1080p.WEB" || v.ReleaseCount != 2 ||
		v.Folded.Line != "SlipStream wouldn't grab 1 more Release." {
		t.Fatalf("getManualSearch = %+v, want one row grabbing Better Site's 1080p and the 720p folded", v)
	}
	code, got := f.s.Call(t, http.MethodGet, path+"/releases", "")
	var list struct {
		Releases []struct {
			Rank *int `json:"rank"`
			Grab struct {
				Grabbable bool   `json:"grabbable"`
				Refusal   string `json:"refusal"`
			} `json:"grab"`
		} `json:"releases"`
	}
	if err := json.Unmarshal([]byte(got), &list); err != nil || code != http.StatusOK || len(list.Releases) != 2 ||
		list.Releases[0].Rank == nil || !list.Releases[0].Grab.Grabbable || list.Releases[1].Rank != nil || list.Releases[1].Grab.Refusal == "" {
		t.Fatalf("listManualSearchReleases = %d %s, want the 1080p ranked and grabbable, then the 720p refused", code, got)
	}
}

func TestGrabFromManualSearchAnswersEachPick(t *testing.T) {
	f, _, _ := mirrored(t, "The.Attic.1080p.WEB", "The.Attic.720p.WEB", false, indexers.Edit{})
	path := "/api/v1/works/" + f.tree.ID.String() + "/manual-search"
	code, got := f.s.Call(t, http.MethodPost, path, "")
	var started struct {
		ManualSearchID string `json:"manualSearchId"`
	}
	if err := json.Unmarshal([]byte(got), &started); err != nil || code != http.StatusAccepted {
		t.Fatalf("startManualSearch = %d %s, want 202", code, got)
	}
	f.running(t)
	var v struct {
		Searching bool `json:"searching"`
		Plan      struct {
			Rows []struct {
				Pick *struct {
					Release struct {
						ID string `json:"id"`
					} `json:"release"`
				} `json:"pick"`
			} `json:"rows"`
		} `json:"plan"`
	}
	for v.Plan.Rows == nil || v.Searching {
		<-f.searched
		if code, got := f.s.Call(t, http.MethodGet, path, ""); code != http.StatusOK || json.Unmarshal([]byte(got), &v) != nil {
			t.Fatalf("getManualSearch = %d %s, want 200", code, got)
		}
	}
	release := v.Plan.Rows[0].Pick.Release.ID
	grab := func(manual string) (int, string) {
		return f.s.Call(t, http.MethodPost, path+"/grabs", `{"manualSearchId":"`+manual+`","picks":[{"releaseId":"`+release+`","lifts":[]},{"releaseId":"`+release+`","lifts":[]}]}`)
	}
	code, got = grab(started.ManualSearchID)
	var out struct {
		Results []struct {
			GrabID  *string         `json:"grabId"`
			Problem json.RawMessage `json:"problem"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(got), &out); err != nil || code != http.StatusOK || len(out.Results) != 2 || out.Results[0].GrabID == nil ||
		out.Results[1].GrabID == nil || *out.Results[0].GrabID != *out.Results[1].GrabID || string(out.Results[0].Problem) != "null" {
		t.Fatalf("grabFromManualSearch = %d %s, want one Grab answering both picks", code, got)
	}
	if code, got := grab(storage.NewUUID().String()); code != http.StatusConflict || !strings.Contains(got, `"code":"manual_search_replaced"`) {
		t.Fatalf("grabFromManualSearch of another search = %d %s, want 409 manual_search_replaced", code, got)
	}
}
