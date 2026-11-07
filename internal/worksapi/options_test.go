package worksapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/works"
)

func (s *server) addRecord(t *testing.T, key string) string {
	t.Helper()
	code, body := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"`+key+`","preset":"all"}`)
	if code != http.StatusOK {
		t.Fatalf("addWork %s = %d %s", key, code, body)
	}
	return uuids.FindString(body)
}

func TestSetWorkOptionsPreviewsThenChangesTheOrder(t *testing.T) {
	s := serve(t)
	r := record()
	r.Stacked = &works.Fetched{Label: r.Tree.Label, Children: []works.Entry{
		{Key: "s1", Label: "Top", Children: []works.Entry{{Key: "c2", Label: "Sock"}}},
	}}
	s.fx.Put("w1", r)
	id := s.addRecord(t, "w1")
	options := func(value string) string {
		return `"options":[{"key":"order","label":"Order","source":"database","applies":"live","isDefault":` + map[string]string{"shelf": "true", "stack": "false"}[value] +
			`,"pendingRestart":false,"kind":"choice","value":"` + value + `","running":"` + value + `","default":"shelf","options":[{"value":"shelf","label":"Shelf order"},{"value":"stack","label":"Stack order"}]}]`
	}

	expect(t, s, http.MethodPut, "/api/v1/works/"+id+"/options", `{"type":{"order":"stack"},"levelFolders":false,"preview":true}`, http.StatusOK,
		`{"levelFolders":false,`+options("stack")+`,"withdraws":1}`)
	_, work := s.call(t, http.MethodGet, "/api/v1/works/"+id, "")
	if !strings.Contains(work, `"levelFolders":true,`+options("shelf")) || strings.Contains(work, `"key":"s1"`) {
		t.Fatalf("after a preview the Work reads %s, want its shelf order and level folders kept", work)
	}

	expect(t, s, http.MethodPut, "/api/v1/works/"+id+"/options", `{"type":{"order":"stack"},"preview":false}`, http.StatusOK,
		`{"levelFolders":true,`+options("stack")+`,"withdraws":1}`)
	_, work = s.call(t, http.MethodGet, "/api/v1/works/"+id, "")
	if !strings.Contains(work, options("stack")) || !strings.Contains(work, `"key":"s1"`) {
		t.Fatalf("after the change the Work reads %s, want it in stack order", work)
	}
}

func TestSetWorkOptionsRefusesAnOrderTheSourceDoesNotList(t *testing.T) {
	s := serve(t)
	s.fx.Put("w1", record())
	id := s.addRecord(t, "w1")
	expect(t, s, http.MethodPut, "/api/v1/works/"+id+"/options", `{"type":{"order":"stack"},"preview":true}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"order_unavailable","title":"The metadata source doesn't list this Work in that order"}`)
	expect(t, s, http.MethodPut, "/api/v1/works/"+id+"/options", `{"type":{"shape":"round"},"preview":false}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"unknown_setting","title":"No setting has this key","detail":"No setting has the key shape."}`)
}
