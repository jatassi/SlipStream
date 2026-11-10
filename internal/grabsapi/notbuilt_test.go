package grabsapi_test

import (
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

// TestTheseOperationsAreNotBuiltYet lists each operation of S32 (Manual
// search and grab) whose domain body a later change of that build writes.
// Each change deletes its operation's line, and the last deletes the test.
func TestTheseOperationsAreNotBuiltYet(t *testing.T) {
	f := searchable(t, site{})
	work, other := "/api/v1/works/"+f.tree.ID.String(), storage.NewUUID().String()
	notBuilt := `{"status":501,"code":"not_built","title":"This operation isn't built yet"}` + "\n"
	for _, op := range []struct{ id, method, path, body string }{
		{"getManualSearch", http.MethodGet, work + "/manual-search?pins=a,b", ""},
		{"listManualSearchReleases", http.MethodGet, work + "/manual-search/releases", ""},
		{"grabFromManualSearch", http.MethodPost, work + "/manual-search/grabs", `{"manualSearchId":"` + other + `","picks":[{"releaseId":"` + other + `.0","lifts":[]}]}`},
	} {
		if code, got := f.s.Call(t, op.method, op.path, op.body); code != http.StatusNotImplemented || got != notBuilt {
			t.Errorf("%s = %d %s, want 501 %s", op.id, code, got, notBuilt)
		}
	}
}
