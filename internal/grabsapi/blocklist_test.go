package grabsapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
)

func TestTheBlocklistListsAndRemovesThroughTheAPI(t *testing.T) {
	f := searchable(t, site{})
	h := bencode.InfoHash([]byte("aaaaaaaaaaaaaaaaaaaa"))
	grabstest.Blocked(t, f.lib, grabs.BlockEntry{
		InfoHash: h, Title: "The.Attic.2020.1080p.BluRay.x264-GRP", Size: 4096, Work: f.tree.ID, Cause: grabs.CauseMarkedFailed, By: &grabs.Actor{},
	})

	code, got := f.s.Call(t, http.MethodGet, "/api/v1/blocklist?q=attic", "")
	for _, want := range []string{`"total":1`, `"infoHash":"` + h.String() + `"`, `"cause":"marked_failed"`, `"because":"Marked failed through the API key."`, `"title":"The Attic"`} {
		if code != http.StatusOK || !strings.Contains(got, want) {
			t.Fatalf("listBlocklist = %d %s, want 200 with %s", code, got, want)
		}
	}
	if code, got := f.s.Call(t, http.MethodGet, "/api/v1/blocklist?q=cellar", ""); code != http.StatusOK || got != `{"entries":[],"total":0}`+"\n" {
		t.Fatalf("listBlocklist?q=cellar = %d %s, want no entries", code, got)
	}

	for range 2 {
		if code, got := f.s.Call(t, http.MethodPost, "/api/v1/blocklist/remove", `{"infoHashes":["`+h.String()+`"]}`); code != http.StatusNoContent {
			t.Fatalf("removeFromBlocklist = %d %s, want 204", code, got)
		}
		<-f.blocklisted
	}
	if _, got := f.s.Call(t, http.MethodGet, "/api/v1/blocklist", ""); got != `{"entries":[],"total":0}`+"\n" {
		t.Fatalf("listBlocklist after remove = %s, want no entries", got)
	}
}
