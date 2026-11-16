package grabsapi_test

import (
	"github.com/jatassi/slipstream/internal/indexers"
	"net/http"
	"testing"
)

func TestSearchingAndWantedAnswerTheReadModel(t *testing.T) {
	f := searchable(t, site{})
	work, item := f.tree.ID.String(), f.tree.Items[0].ID
	expect(t, f.s, http.MethodGet, "/api/v1/works/"+work+"/searching", http.StatusOK,
		`{"groups":[{"key":"`+string(indexers.ItemGroup(item))+`","label":"The Attic","state":"idle","lane":null,"trigger":null,"again":false,`+
			`"behind":null,"lastSearchedAt":null,"lastResult":null,"lastFound":0,"lastGrabs":0,"nextSearchAt":null}],"waits":[],"stops":[],`+
			`"lead":{"groupKey":null,"of":0,"wait":null}}`)
	expect(t, f.s, http.MethodGet, "/api/v1/wanted", http.StatusOK,
		`{"status":{"searching":0,"queued":0,"nextBacklogAt":"T","limits":[]},"total":1,"leftGone":0,"rows":[{"work":{"id":"`+work+
			`","contentType":"fixture","label":"The Attic","posterUrl":null},"reason":"missing","others":{},"ranges":[{"first":"1","last":"1"}],`+
			`"lastSearchedAt":null,"lastResult":null,"nextSearchAt":"T","state":"idle"}],"next":null}`)
	expect(t, f.s, http.MethodGet, "/api/v1/wanted?cursor=nope", http.StatusBadRequest,
		`{"status":400,"code":"wanted_cursor_stale","title":"This list changed. Load Wanted again."}`)
}
