package importsapi_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

func TestUnbuiltOperationsAnswer501(t *testing.T) {
	s := serve(t)
	id := storage.NewUUID().String()
	for _, op := range []struct{ method, path, body string }{
		{http.MethodPost, "/imports", `{"nodeId":"` + id + `","folder":"/downloads/Heat"}`},
		{http.MethodPost, "/imports/" + id + "/files/0", `{"action":"confirm"}`},
		{http.MethodPost, "/imports/confirm", `{"importIds":["` + id + `"]}`},
		{http.MethodPost, "/imports/move-files", `{"importIds":["` + id + `"],"on":true}`},
	} {
		code, reply := s.Call(t, op.method, "/api/v1"+op.path, op.body)
		if code != http.StatusNotImplemented || !strings.Contains(reply, `"code":"import_not_built"`) {
			t.Errorf("%s %s = %d %s, want 501 import_not_built until its lane builds it", op.method, op.path, code, reply)
		}
	}
}

func TestListImportsPagesThroughTheWaitingImports(t *testing.T) {
	s := serve(t)
	for _, folder := range []string{"Alien (1979)", "Brazil (1985)", "Heat (1995)"} {
		s.pend(t, folder, nil, false, noMatch(folder+"/a.mkv"))
	}
	page := func(query string) (folders string, next *string) {
		t.Helper()
		code, reply := s.Call(t, http.MethodGet, "/api/v1/imports"+query, "")
		var l struct {
			Imports []struct {
				Folder string `json:"folder"`
			} `json:"imports"`
			Next *string `json:"next"`
		}
		if err := json.Unmarshal([]byte(reply), &l); code != http.StatusOK || err != nil {
			t.Fatalf("listImports%s = %d %s", query, code, reply)
		}
		names := make([]string, len(l.Imports))
		for i, im := range l.Imports {
			names[i] = im.Folder
		}
		return strings.Join(names, ","), l.Next
	}
	got, next := page("?waiting=true&limit=2")
	if got != "Alien (1979),Brazil (1985)" || next == nil {
		t.Fatalf("listImports?limit=2 lists %q with next %v, want the first two folders and a next", got, next)
	}
	if got, next := page("?waiting=true&limit=2&after=" + *next); got != "Heat (1995)" || next != nil {
		t.Errorf("listImports after the first page lists %q with next %v, want the last folder and no next", got, next)
	}
	if got := s.entries(t, "?work="+storage.NewUUID().String()); !strings.Contains(got, `"imports":[]`) {
		t.Errorf("listImports of a Work no Import goes into = %s, want none", got)
	}
	s.Expect(t, http.MethodGet, "/api/v1/imports?after="+storage.NewUUID().String(), "", http.StatusConflict,
		`{"status":409,"code":"import_list_moved","title":"The Import the last page ended at is no longer listed. List again from the first page."}`)
}
