package targetsapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
)

const fanEdit = `{"key":"fan-edit","name":"Fan Edit","aliases":["Fan Edit","Fan Cut"],"origin":"custom"}`

func cutsBody(cuts ...string) string {
	return `{"kind":"cuts","cuts":["` + strings.Join(cuts, `","`) + `"]}`
}

func TestACustomCutIsKeyedByItsNameAndRefusesANameAnotherCutReads(t *testing.T) {
	s := serve(t)
	s.fx.SetCuts([]release.Cut{{Key: "superfan", Name: "Superfan", Aliases: []string{"Superfan", "Superfan Edition"}}})
	s.Expect(t, http.MethodPost, "/api/v1/cuts", `{"name":"Fan Edit","aliases":["Fan Cut","fan edit","FAN CUT"]}`, http.StatusCreated, fanEdit)
	if s.changed.n != 1 {
		t.Fatalf("creating a cut published %d times, want once", s.changed.n)
	}
	code, body := s.Call(t, http.MethodGet, "/api/v1/cuts", "")
	if code != http.StatusOK || !strings.HasSuffix(strings.TrimSpace(body), `"origin":"type","contentType":"fixture"},`+fanEdit+`]}`) {
		t.Fatalf("listCuts = %d %s, want the custom cut last", code, body)
	}
	taken := `{"status":409,"code":"cut_name_taken","title":"A cut already has that name, alias or key"}`
	for _, name := range []string{"Fan Edit", "fan-edit", "Fan Cut", "Extended Cut", "Superfan Edition", "Standard", "Any"} {
		s.Expect(t, http.MethodPost, "/api/v1/cuts", `{"name":"`+name+`"}`, http.StatusConflict, taken)
	}
	s.Expect(t, http.MethodPost, "/api/v1/cuts", `{"name":"Mine","aliases":["IMAX Edition"]}`, http.StatusConflict, taken)
	for _, body := range []string{`{"name":"!!!"}`, `{"name":"Mine","aliases":["--"]}`} {
		if code, reply := s.Call(t, http.MethodPost, "/api/v1/cuts", body); code != http.StatusUnprocessableEntity || !strings.Contains(reply, `"code":"invalid"`) {
			t.Errorf("POST /cuts %s = %d %s, want 422 invalid", body, code, reply)
		}
	}
}

func TestOnlyACustomCutChangesOrGoes(t *testing.T) {
	s := serve(t)
	s.fx.SetCuts([]release.Cut{{Key: "superfan", Name: "Superfan", Aliases: []string{"Superfan"}}})
	s.Call(t, http.MethodPost, "/api/v1/cuts", `{"name":"Fan Edit","aliases":["Fan Cut"]}`)
	fixed := `{"status":409,"code":"standard_is_fixed","title":"Standard can't be renamed or removed"}`
	notCustom := `{"status":409,"code":"cut_not_custom","title":"Only a custom cut can be changed or removed"}`
	missing := `{"status":404,"code":"cut_not_found","title":"No cut has that key"}`
	for _, c := range []struct{ key, want string }{
		{"standard", fixed}, {"extended", notCustom}, {"superfan", notCustom}, {"nope", missing},
	} {
		s.Expect(t, http.MethodPut, "/api/v1/cuts/"+c.key, `{"name":"Renamed"}`, statusOf(c.want), c.want)
		s.Expect(t, http.MethodDelete, "/api/v1/cuts/"+c.key, "", statusOf(c.want), c.want)
	}
	s.Expect(t, http.MethodPut, "/api/v1/cuts/fan-edit", `{"name":"Extended"}`, http.StatusConflict,
		`{"status":409,"code":"cut_name_taken","title":"A cut already has that name, alias or key"}`)
	code, body := s.Call(t, http.MethodPut, "/api/v1/cuts/fan-edit", `{"name":"Fan Cut","aliases":["Fan Edit"]}`)
	if want := `"cut":{"key":"fan-edit","name":"Fan Cut","aliases":["Fan Cut","Fan Edit"],"origin":"custom"}`; code != http.StatusOK || !strings.Contains(body, want) {
		t.Fatalf("renaming Fan Edit to its own alias = %d %s, want 200 with %s", code, body, want)
	}
	deleted(t, s, "fan-edit")
	s.Expect(t, http.MethodDelete, "/api/v1/cuts/fan-edit", "", http.StatusNotFound, missing)
}

func statusOf(problem string) int {
	switch {
	case strings.Contains(problem, `"status":404`):
		return http.StatusNotFound
	case strings.Contains(problem, `"status":422`):
		return http.StatusUnprocessableEntity
	}
	return http.StatusConflict
}

func TestAnEditionListNamesOnlyCutsItsTypeCanHold(t *testing.T) {
	s, ids := shelved(t)
	missing := `{"status":422,"code":"cut_not_found","title":"No cut has that key"}`
	named := strings.Replace(profileBody("Cuts", ids["web"]), `{"kind":"any"}`, cutsBody("standard", "nope"), 1)
	s.Expect(t, http.MethodPost, "/api/v1/profiles", named, http.StatusUnprocessableEntity, missing)
	s.Expect(t, http.MethodPut, "/api/v1/profiles/"+ids["small"], named, http.StatusUnprocessableEntity, missing)
	s.Expect(t, http.MethodPut, "/api/v1/works/"+ids["work"]+"/wants", `{"profileId":"`+ids["small"]+`","editions":`+cutsBody("nope")+`}`, http.StatusUnprocessableEntity, missing)
	s.fx.SetCuts([]release.Cut{{Key: "superfan", Name: "Superfan", Aliases: []string{"Superfan"}}})
	extended := strings.Replace(profileBody("Cuts", ids["web"]), `{"kind":"any"}`, cutsBody("standard", "superfan", "extended"), 1)
	code, body := s.Call(t, http.MethodPost, "/api/v1/profiles", extended)
	if want := `"editions":` + cutsBody("standard", "superfan", "extended"); code != http.StatusCreated || !strings.Contains(body, want) {
		t.Fatalf("createProfile naming the type's cut = %d %s, want 201 with %s", code, body, want)
	}
}

func TestACutHeldByAListOrAFileStays(t *testing.T) {
	s, ids := shelved(t)
	s.Call(t, http.MethodPost, "/api/v1/cuts", `{"name":"Fan Edit"}`)
	inUse := func(detail string) string {
		return `{"status":409,"code":"cut_in_use","title":"Edition lists or files hold the cut","detail":"` + detail + `"}`
	}
	if code, body := s.Call(t, http.MethodPut, "/api/v1/profiles/"+ids["small"], strings.Replace(profileBody("Small", ids["web"]), `{"kind":"any"}`, cutsBody("standard", "fan-edit"), 1)); code != http.StatusOK {
		t.Fatalf("updateProfile naming Fan Edit = %d %s", code, body)
	}
	s.Expect(t, http.MethodDelete, "/api/v1/cuts/fan-edit", "", http.StatusConflict, inUse("Small uses it"))
	code, body := s.Call(t, http.MethodPut, "/api/v1/works/"+ids["work"]+"/wants?unwanted=keep", `{"profileId":"`+ids["fixture"]+`","editions":`+cutsBody("fan-edit")+`}`)
	if code != http.StatusOK {
		t.Fatalf("setWorkWants naming Fan Edit = %d %s", code, body)
	}
	s.Expect(t, http.MethodDelete, "/api/v1/cuts/fan-edit", "", http.StatusConflict, inUse("Small and 1 Work's own list use it"))
	s.Call(t, http.MethodPut, "/api/v1/profiles/"+ids["small"], profileBody("Small", ids["web"]))
	s.Call(t, http.MethodPut, "/api/v1/works/"+ids["work"]+"/wants", `{"profileId":"`+ids["fixture"]+`","editions":null}`)
	cutAll(t, s, "fan-edit")
	s.Expect(t, http.MethodDelete, "/api/v1/cuts/fan-edit", "", http.StatusConflict, inUse("1 file uses it"))
	cutAll(t, s, "standard")
	deleted(t, s, "fan-edit")
}

func TestRenamingACutPreviewsThenRenamesItsFiles(t *testing.T) {
	s, ids := shelved(t)
	s.Call(t, http.MethodPost, "/api/v1/cuts", `{"name":"Fan Edit"}`)
	cutAll(t, s, "fan-edit")
	before := s.seen(t, ids)
	rename := `{"name":"Fan Favorite","aliases":["Fan Edit"]}`
	code, body := s.Call(t, http.MethodPut, "/api/v1/cuts/fan-edit?preview=true", rename)
	want := `{"works":1,"ownList":0,"unwanted":{"files":0,"bytes":0},"renamed":1,"newlyWanted":0,"downloading":0}`
	if got := impactOf(t, "previewing the rename", code, body); got != want {
		t.Fatalf("previewing the rename = %s, want %s", got, want)
	}
	if after := s.seen(t, ids); after != before {
		t.Fatalf("a preview saved:\n%s\nwant\n%s", after, before)
	}
	code, body = s.Call(t, http.MethodPut, "/api/v1/cuts/fan-edit", rename)
	if got := impactOf(t, "renaming the cut", code, body); got != want {
		t.Fatalf("renaming the cut = %s, want %s", got, want)
	}
	if !strings.Contains(body, `"cut":{"key":"fan-edit","name":"Fan Favorite","aliases":["Fan Favorite","Fan Edit"],"origin":"custom"}`) {
		t.Fatalf("renaming the cut answered %s, want the cut renamed", body)
	}
	var path string
	if err := s.lib.QueryRowContext(t.Context(), `SELECT path FROM files_file`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if want := "The Attic/The Attic - Scarf - {edition-Fan Favorite} 1080p.mkv"; path != want {
		t.Fatalf("the file of the cut is at %q, want %q", path, want)
	}
}

func deleted(t *testing.T, s *served, key string) {
	t.Helper()
	if code, body := s.Call(t, http.MethodDelete, "/api/v1/cuts/"+key, ""); code != http.StatusNoContent {
		t.Fatalf("DELETE /cuts/%s = %d %s, want 204", key, code, body)
	}
}

func cutAll(t *testing.T, s *served, cut string) {
	t.Helper()
	if err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, `UPDATE files_file SET cut = ?`, cut)
		return err
	}); err != nil {
		t.Fatal(err)
	}
}
