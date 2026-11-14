package importsapi_test

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/storage"
)

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

func TestImportFolderRefusesAFolderOutsideTheNodesRootsAndDownloadFolders(t *testing.T) {
	s := serve(t)
	s.Expect(t, http.MethodPost, "/api/v1/imports", `{"nodeId":"`+s.node.String()+`","folder":"`+filepath.ToSlash(t.TempDir())+`"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"import_outside","title":"The folder isn't in one of the Node's library roots or download folders"}`)
}

func TestAFileActionItsRowDoesntOfferIsRefused(t *testing.T) {
	s := serve(t)
	id := s.pend(t, "Heat (1995)", nil, false, noMatch("Heat (1995)/a.mkv"))
	s.Expect(t, http.MethodPost, "/api/v1/imports/"+id+"/files/0", `{"action":"confirm"}`, http.StatusConflict,
		`{"status":409,"code":"import_not_offered","title":"The file doesn't offer that action now. Get the Import again to see what it offers."}`)
}
