package targetsapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// shelved is s with a Work of one Item wanting from the fixture type's
// default Profile, whose 2000 MiB WEB-DL fills the starter, and a Profile
// Small holding a Target Web of WEB-DL alone. ids names them all.
func shelved(t *testing.T) (s *served, ids map[string]string) {
	t.Helper()
	ctx := t.Context()
	s = serve(t)
	s.fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	tree, _, err := s.works.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{
		Monitor: true,
		Also: func(ctx context.Context, tx *storage.Tx[storage.Library], w works.Summary) error {
			if err := targets.Assign(ctx, tx, w, nil); err != nil {
				return err
			}
			return roots.Assign(ctx, tx, w, nil)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	root := rootstest.Root(t, s.lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	err = s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: root.ID, Path: "The Attic/The Attic - Scarf - 1080p.mkv",
			Size: 2000 << 20, Quality: release.WEBDL1080p,
		}, []targets.TargetID{{UUID: must[storage.UUID](t)(storage.ParseUUID(s.starter))}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	ids = map[string]string{"starter": s.starter, "fixture": s.fixture, "work": tree.ID.String()}
	_, body := s.Call(t, http.MethodPost, "/api/v1/targets", webBody)
	ids["web"] = ownertest.FirstID(body)
	_, body = s.Call(t, http.MethodPost, "/api/v1/profiles", profileBody("Small", ids["web"]))
	ids["small"] = ownertest.FirstID(body)
	s.changed.n = 0
	return s, ids
}

func fill(s string, ids map[string]string) string {
	for name, id := range ids {
		s = strings.ReplaceAll(s, "@"+name, id)
	}
	return s
}

// seen is every read an Admin has of what a Change could alter.
func (s *served) seen(t *testing.T, ids map[string]string) string {
	t.Helper()
	var out []string
	for _, path := range []string{"/api/v1/targets", "/api/v1/profiles", "/api/v1/works/" + ids["work"] + "/wants"} {
		code, body := s.Call(t, http.MethodGet, path, "")
		if code != http.StatusOK {
			t.Fatalf("GET %s = %d %s", path, code, body)
		}
		out = append(out, body)
	}
	var paths []string
	rows, err := s.lib.QueryContext(t.Context(), `SELECT path || ' ' || label || ' ' || extra FROM files_file ORDER BY path`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return strings.Join(append(out, paths...), "")
}

type changeReply struct {
	Impact json.RawMessage `json:"impact"`
}

func impactOf(t *testing.T, what string, code int, body string) string {
	t.Helper()
	var r changeReply
	if code != http.StatusOK || json.Unmarshal([]byte(body), &r) != nil {
		t.Fatalf("%s = %d %s, want 200 with an impact", what, code, body)
	}
	return string(r.Impact)
}

func TestEveryChangePreviewsWithoutSavingThenSavesWhatItPreviewed(t *testing.T) {
	s, ids := shelved(t)
	starterAs := func(name string) string { return `{"name":"` + name + `",` + starterRules + `}` }
	impact := func(works, renamed, newlyWanted int) string {
		return `{"works":` + strconv.Itoa(works) + `,"ownList":0,"unwanted":{"files":0,"bytes":0},"renamed":` + strconv.Itoa(renamed) +
			`,"newlyWanted":` + strconv.Itoa(newlyWanted) + `,"downloading":0}`
	}
	for _, c := range []struct {
		name, method, path, body, impact, after string
	}{
		{"renaming the starter", http.MethodPut, "/api/v1/targets/@starter", starterAs("Starter"), impact(1, 1, 0), `"name":"Starter"`},
		{
			"putting Web first in the default Profile", http.MethodPut, "/api/v1/profiles/@fixture", profileBody("Fixture", ids["web"], ids["starter"]),
			impact(1, 1, 0), `"targets":[{"targetId":"@web","sourcing":"download"},{"targetId":"@starter","sourcing":"download"}]`,
		},
		{
			"giving the Work its own list", http.MethodPut, "/api/v1/works/@work/wants", `{"editions":{"kind":"any"}}`,
			impact(0, 0, 0), `"profileId":"@fixture","profileName":"Fixture","editions":{"kind":"any"},"from":"work"`,
		},
		{"deleting Small into the default", http.MethodDelete, "/api/v1/profiles/@small?moveTo=@fixture", "", impact(0, 0, 0), `"name":"Fixture"`},
		{
			"resetting the Work's list", http.MethodPut, "/api/v1/works/@work/wants", `{"editions":null}`,
			impact(0, 0, 0), `"from":"profile"`,
		},
	} {
		path, body := fill(c.path, ids), fill(c.body, ids)
		before := s.seen(t, ids)
		join := "?"
		if strings.Contains(path, "?") {
			join = "&"
		}
		code, reply := s.Call(t, c.method, path+join+"preview=true", body)
		preview := impactOf(t, c.name+" previewed", code, reply)
		if after := s.seen(t, ids); after != before || s.changed.n != 0 {
			t.Fatalf("previewing %s published %d times and changed\n%s\nto\n%s", c.name, s.changed.n, before, after)
		}
		code, reply = s.Call(t, c.method, path, body)
		if saved := impactOf(t, c.name, code, reply); preview != c.impact || saved != preview {
			t.Fatalf("%s: preview %s, save %s, want both %s", c.name, preview, saved, c.impact)
		}
		if !strings.Contains(reply, fill(c.after, ids)) || s.changed.n != 1 {
			t.Fatalf("%s answered %s and published %d times, want %s once", c.name, reply, s.changed.n, fill(c.after, ids))
		}
		s.changed.n = 0
	}
	get(t, s, "/api/v1/works/"+ids["work"]+"/wants", `{"profileId":"@fixture","profileName":"Fixture","editions":{"kind":"any"},"from":"profile"}`, ids)
}

func TestASaveDeletingMoreThanItsPreviewConfirmedIsRefused(t *testing.T) {
	s, ids := shelved(t)
	empty := fill(profileBody("Fixture"), ids)
	path := "/api/v1/profiles/" + ids["fixture"]
	unwanted := `{"works":1,"ownList":0,"unwanted":{"files":1,"bytes":2097152000},"renamed":0,"newlyWanted":0,"downloading":0}`
	code, reply := s.Call(t, http.MethodPut, path+"?preview=true&unwanted=keep", empty)
	if got := impactOf(t, "emptying the default Profile previewed", code, reply); got != unwanted {
		t.Fatalf("the preview = %s, want %s", got, unwanted)
	}
	before := s.seen(t, ids)
	s.Expect(t, http.MethodPut, path+"?confirm=0", empty, http.StatusConflict,
		`{"status":409,"code":"impact_changed","title":"The change now does more than its preview showed","detail":"The save now leaves 1 file unwanted, and the preview showed 0. Preview the change again."}`)
	if after := s.seen(t, ids); after != before || s.changed.n != 0 {
		t.Fatalf("the refused save published %d times and changed\n%s\nto\n%s", s.changed.n, before, after)
	}
	code, reply = s.Call(t, http.MethodPut, path+"?unwanted=keep&confirm=0", empty)
	if got := impactOf(t, "emptying the default Profile, keeping its file", code, reply); got != unwanted {
		t.Fatalf("the save keeping = %s, want %s: keeping deletes nothing, so confirm=0 holds", got, unwanted)
	}
	if got := s.seen(t, ids); !strings.Contains(got, "The Attic/The Attic - Scarf - 1080p (extra 1).mkv 1080p 1") {
		t.Fatalf("after keeping, the files are %s, want the WEB-DL Kept under its label", got)
	}
	code, reply = s.Call(t, http.MethodPut, path+"?confirm=1", fill(profileBody("Fixture", ids["starter"]), ids))
	if got := impactOf(t, "putting the starter back", code, reply); !strings.Contains(got, `"renamed":1`) {
		t.Fatalf("putting the starter back = %s, want the Kept file renamed back", got)
	}
	if got := s.seen(t, ids); !strings.Contains(got, "The Attic/The Attic - Scarf - 1080p.mkv 1080p 0") {
		t.Fatalf("after putting the starter back, the files are %s, want the WEB-DL filling it again under its name", got)
	}
}

func TestEachChangesErrorAnswersItsProblem(t *testing.T) {
	s, ids := shelved(t)
	ids["none"] = storage.NewUUID().String()
	notFound := func(what string) string {
		return `{"status":404,"code":"` + what + `_not_found","title":"No ` + strings.ToUpper(what[:1]) + what[1:] + ` has that id"}`
	}
	for _, c := range []struct {
		method, path, body string
		status             int
		want               string
	}{
		{http.MethodPut, "/api/v1/targets/@none", webBody, http.StatusNotFound, notFound("target")},
		{http.MethodPut, "/api/v1/targets/@starter", webBody, http.StatusConflict, `{"status":409,"code":"target_name_taken","title":"A Target already has that name"}`},
		{
			http.MethodPut, "/api/v1/targets/@starter", `{"name":"1080p","tiers":[["webdl-1080p"]],"cutoff":"bluray-1080p"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"quality_not_in_target","title":"The Target doesn't accept that quality","detail":"the Target doesn't accept that quality: the cutoff bluray-1080p"}`,
		},
		{http.MethodPut, "/api/v1/profiles/@none", profileBody("Small"), http.StatusNotFound, notFound("profile")},
		{http.MethodPut, "/api/v1/profiles/@fixture", profileBody("Small"), http.StatusConflict, `{"status":409,"code":"profile_name_taken","title":"A Profile already has that name"}`},
		{http.MethodPut, "/api/v1/profiles/@fixture", profileBody("Fixture", "@none"), http.StatusUnprocessableEntity, `{"status":422,"code":"target_not_found","title":"No Target has that id"}`},
		{
			http.MethodPut, "/api/v1/profiles/@fixture", strings.Replace(profileBody("Fixture"), `"fixture"`, `"music"`, 1), http.StatusUnprocessableEntity,
			`{"status":422,"code":"profile_wrong_type","title":"The Profile is for another Content Type"}`,
		},
		{http.MethodDelete, "/api/v1/profiles/@fixture?moveTo=@small", "", http.StatusConflict, `{"status":409,"code":"profile_is_default","title":"The default Profile can't be deleted"}`},
		{http.MethodDelete, "/api/v1/profiles/@none?moveTo=@small", "", http.StatusNotFound, notFound("profile")},
		{http.MethodDelete, "/api/v1/profiles/@small?moveTo=@none", "", http.StatusUnprocessableEntity, `{"status":422,"code":"profile_not_found","title":"No Profile has that id"}`},
		{
			http.MethodDelete, "/api/v1/profiles/@small?moveTo=@small", "", http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"query.moveTo","message":"a Profile's Works can't move to the Profile being deleted"}]}`,
		},
		{http.MethodDelete, "/api/v1/profiles/@small", "", http.StatusUnprocessableEntity, `@invalid`},
		{http.MethodPut, "/api/v1/works/@none/wants", `{"editions":null}`, http.StatusNotFound, `{"status":404,"code":"work_not_found","title":"No Work has that id"}`},
		{http.MethodPut, "/api/v1/works/@work/wants?unwanted=maybe", `{"editions":null}`, http.StatusUnprocessableEntity, `@invalid`},
		{http.MethodPut, "/api/v1/works/@work/wants", `{"editions":{"kind":"cuts"}}`, http.StatusUnprocessableEntity, `@invalid`},
		{http.MethodPut, "/api/v1/works/@work/wants", `{"editions":{"kind":"bogus"}}`, http.StatusUnprocessableEntity, `@invalid`},
		{http.MethodPut, "/api/v1/works/@work/wants", `{"editions":{"kind":"cuts","cuts":[]}}`, http.StatusUnprocessableEntity, `@invalid`},
		{http.MethodPut, "/api/v1/works/@work/wants", `{"editions":{"kind":"cuts","cuts":["a","a"]}}`, http.StatusUnprocessableEntity, `@invalid`},
	} {
		path, body := fill(c.path, ids), fill(c.body, ids)
		if c.want == "@invalid" {
			if code, reply := s.Call(t, c.method, path, body); code != c.status || !strings.Contains(reply, `"code":"invalid"`) {
				t.Errorf("%s %s %s = %d %s, want %d invalid", c.method, path, body, code, reply, c.status)
			}
			continue
		}
		s.Expect(t, c.method, path, body, c.status, c.want)
	}
	if s.changed.n != 0 {
		t.Fatalf("targets.changed published %d times, want none", s.changed.n)
	}
}
