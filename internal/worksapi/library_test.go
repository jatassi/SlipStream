package worksapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var today = time.Date(2026, time.June, 1, 12, 0, 0, 0, time.UTC)

// shelved is a server holding Targets HD and 2160p, a default Profile
// Archive wanting both, a Profile One wanting HD alone, and a default
// root.
type shelved struct {
	*server
	hd, uhd      targets.Target
	archive, one targets.Profile
	root         roots.Root
}

func stock(t *testing.T, rooted bool) *shelved {
	t.Helper()
	ctx := t.Context()
	s := &shelved{server: serveAt(t, func() time.Time { return today })}
	var err error
	if s.hd, err = s.goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{
		Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p}, {release.WEBDL1080p}}, Cutoff: release.Bluray1080p,
	}); err != nil {
		t.Fatal(err)
	}
	if s.uhd, err = s.goals.CreateTarget(ctx, identity.Admin{}, targets.Spec{Name: "2160p", Tiers: [][]release.Quality{{release.WEBDL2160p}}, Cutoff: release.WEBDL2160p}); err != nil {
		t.Fatal(err)
	}
	if s.archive, err = s.goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "Archive", ContentType: contenttypetest.ID, Entries: []targets.Entry{
		{Target: s.hd.ID, Sourcing: targets.Download}, {Target: s.uhd.ID, Sourcing: targets.Download},
	}}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.goals.MakeDefault(ctx, identity.Admin{}, s.archive.ID); err != nil {
		t.Fatal(err)
	}
	if s.one, err = s.goals.CreateProfile(ctx, identity.Admin{}, targets.ProfileSpec{Name: "One", ContentType: contenttypetest.ID, Entries: []targets.Entry{
		{Target: s.hd.ID, Sourcing: targets.Download},
	}}); err != nil {
		t.Fatal(err)
	}
	if rooted {
		s.root = rootstest.Root(t, s.lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	}
	return s
}

// add adds a Work labeled label with Things things in one Box, through
// addWork, and answers its tree.
func (s *shelved) add(t *testing.T, key, label, body string, things ...string) works.Tree {
	t.Helper()
	r := contenttypetest.Record{Tree: works.Fetched{Label: label, Children: []works.Entry{{Key: "b1", Label: "Box"}}}}
	for _, th := range things {
		r.Tree.Children[0].Children = append(r.Tree.Children[0].Children, works.Entry{Key: works.Key(th), Label: th})
	}
	return s.addRecord(t, key, r, body)
}

func (s *shelved) addRecord(t *testing.T, key string, r contenttypetest.Record, body string) works.Tree {
	t.Helper()
	s.fx.Put(works.Key(key), r)
	if !strings.Contains(body, `"monitor"`) {
		body += `,"monitor":false`
	}
	code, reply := s.call(t, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"`+key+`"`+body+`}`)
	if code != http.StatusOK {
		t.Fatalf("addWork %s = %d %s", key, code, reply)
	}
	var added struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal([]byte(reply), &added); err != nil {
		t.Fatal(err)
	}
	id, err := storage.ParseUUID(added.Work.ID)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := s.works.Get(t.Context(), identity.Admin{}, works.WorkID{UUID: id})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

// place records a file of quality at the Work's first Thing, filling
// fills, or a Kept file when fills is empty.
func (s *shelved) place(t *testing.T, w works.Tree, path string, q release.Quality, size int64, fills ...targets.TargetID) files.ID {
	t.Helper()
	return s.placeAt(t, w.Items[0], w, path, q, size, fills...)
}

func (s *shelved) placeAt(t *testing.T, it works.Item, w works.Tree, path string, q release.Quality, size int64, fills ...targets.TargetID) files.ID {
	t.Helper()
	f := files.File{WorkID: w.ID, ItemID: it.ID, Cut: release.Standard, RootID: s.root.ID, Path: path, Size: size, Quality: q}
	if len(fills) == 0 {
		f.Label, f.Extra = "HD", 1
	}
	err := s.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		f, err = files.Add(ctx, tx, f, fills)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return f.ID
}

func (s *shelved) grab(it works.Item, target targets.TargetID, g files.GrabCell) storage.UUID {
	g.Grab = storage.NewUUID()
	s.grabs[files.CellKey{Item: it.ID, Edition: targets.AnyEdition, Target: target}] = g
	return g.Grab
}

type row struct {
	ID          string          `json:"id"`
	Label       string          `json:"label"`
	Monitored   string          `json:"monitored"`
	ProfileID   string          `json:"profileId"`
	RootID      *string         `json:"rootId"`
	PosterURL   *string         `json:"posterUrl"`
	Rating      json.RawMessage `json:"rating"`
	AvailableOn *string         `json:"availableOn"`
	Bytes       int64           `json:"bytes"`
	Files       int             `json:"files"`
	Kept        int             `json:"kept"`
	Status      *string         `json:"status"`
	Cells       map[string]int  `json:"cells"`
	Failed      int             `json:"failed"`
	Items       int             `json:"items"`
	ItemStatus  map[string]int  `json:"itemStatus"`
	NeedsYou    int             `json:"needsYou"`
	Values      json.RawMessage `json:"values"`
}

func (s *shelved) rows(t *testing.T, query string) []row {
	t.Helper()
	code, body := s.call(t, http.MethodGet, "/api/v1/library"+query, "")
	var out struct {
		Rows []row `json:"rows"`
	}
	if err := json.Unmarshal([]byte(body), &out); code != http.StatusOK || err != nil {
		t.Fatalf("listLibrary%s = %d %s", query, code, body)
	}
	return out.Rows
}

func TestEachLibraryRowRollsUpItsCellsAndFiles(t *testing.T) {
	s := stock(t, true)
	photo := artwork.Ref{Source: "fixture", Shape: artwork.ShapePoster, Path: "/heat.jpg"}
	heat := s.addRecord(t, "heat", contenttypetest.Record{
		Tree: works.Fetched{
			Label: "Heat", Children: []works.Entry{{Key: "b1", Label: "Box", Children: []works.Entry{{Key: "c1", Label: "Heat"}}}},
			Ratings: []works.Rating{{Source: "imdb", Value: 8.3}, {Source: "tmdb", Value: 7.9}},
			Artwork: []works.Art{{Kind: "photo", Ref: photo}},
		},
		WorkFields: []works.FieldValue{works.TextField("origin", "Los Angeles"), works.TextsField("tags", []string{"Crime", "Drama"})},
	}, `,"monitor":true`)
	s.place(t, heat, "Heat/Heat - HD.mkv", release.Bluray1080p, 1000, s.hd.ID)
	s.place(t, heat, "Heat/Heat - 2160p.mkv", release.WEBDL2160p, 3000, s.uhd.ID)
	s.place(t, heat, "Heat/Heat - HD (extra 1).mkv", release.WEBDL1080p, 500)

	one := `,"monitor":true,"profileId":"` + s.one.ID.String() + `"`
	s.add(t, "arrival", "Arrival", one, "Arrival")
	brazil := s.add(t, "brazil", "Brazil", one, "Brazil")
	s.grab(brazil.Items[0], s.hd.ID, files.GrabCell{State: files.GrabDownloading})
	contact := s.add(t, "contact", "Contact", one, "Contact")
	s.grab(contact.Items[0], s.hd.ID, files.GrabCell{State: files.GrabFailed})
	dune := s.add(t, "dune", "Dune", one, "Dune")
	s.fx.SetTiming(dune.Items[0].ID, targets.Timing{Released: release.Known(works.Date{Year: 2026, Month: 12})})
	eraser := s.add(t, "eraser", "Eraser", one, "Eraser")
	s.place(t, eraser, "Eraser/Eraser - HD.mkv", release.WEBDL1080p, 700, s.hd.ID)
	fargo := s.add(t, "fargo", "Fargo", `,"profileId":"`+s.one.ID.String()+`"`, "Fargo")
	s.grab(fargo.Items[0], s.hd.ID, files.GrabCell{State: files.GrabFailed})

	got := s.rows(t, "")
	if len(got) != 7 || got[0].Label != "Arrival" || got[3].Label != "Dune" || got[5].Label != "Fargo" || got[6].Label != "Heat" {
		t.Fatalf("listLibrary answered %+v, want 7 rows by sort label", got)
	}
	str := func(s string) *string { return &s }
	root := str(s.root.ID.String())
	want := row{
		ID: heat.ID.String(), Label: "Heat", Monitored: "all", ProfileID: s.archive.ID.String(), RootID: root, PosterURL: str(minter.URL(photo)),
		Rating: json.RawMessage(`{"source":"tmdb","value":7.9,"scale":10}`), AvailableOn: str("2000-01-01"), Bytes: 4500, Files: 3, Kept: 1,
		Status: str("met"), Cells: map[string]int{"met": 2}, Items: 1, ItemStatus: map[string]int{"met": 1},
		Values: json.RawMessage(`{"origin":"Los Angeles","tags":["Crime","Drama"]}`),
	}
	if !reflect.DeepEqual(got[6], want) {
		t.Fatalf("Heat's row is\n%+v\nwant\n%+v", got[6], want)
	}
	for i, c := range []struct {
		label, status, monitored string
		failed, needsYou         int
		available                string
	}{
		{"Arrival", "missing", "all", 0, 0, "2000-01-01"},
		{"Brazil", "downloading", "all", 0, 0, "2000-01-01"},
		{"Contact", "failed", "all", 1, 1, "2000-01-01"},
		{"Dune", "unreleased", "all", 0, 0, "2026-12"},
		{"Eraser", "upgradable", "all", 0, 1, "2000-01-01"},
		{"Fargo", "failed", "none", 0, 0, "2000-01-01"},
	} {
		r := got[i]
		if r.Label != c.label || *r.Status != c.status || !reflect.DeepEqual(r.Cells, map[string]int{c.status: 1}) || r.Monitored != c.monitored ||
			r.Failed != c.failed || r.NeedsYou != c.needsYou || *r.AvailableOn != c.available || r.ProfileID != s.one.ID.String() ||
			*r.RootID != *root || r.PosterURL != nil || string(r.Rating) != "null" || string(r.Values) != "{}" {
			t.Fatalf("row %d is %+v, want %s %s, %s monitored, %d failed, %d Needs you, available %s, from One, in the root, with no poster, rating or values",
				i, r, c.label, c.status, c.monitored, c.failed, c.needsYou, c.available)
		}
	}

	named := s.rows(t, "?workIds="+heat.ID.String()+","+contact.ID.String()+","+storage.NewUUID().String())
	if len(named) != 2 || named[0].Label != "Contact" || !reflect.DeepEqual(named[1], want) {
		t.Fatalf("listLibrary for Heat, Contact and an id no Work has = %+v, want Contact's and Heat's rows", named)
	}
	if typed := s.rows(t, "?contentType=fixture"); len(typed) != 7 {
		t.Fatalf("listLibrary for the fixture type = %d rows, want 7", len(typed))
	}
	expect(t, s.server, http.MethodGet, "/api/v1/library?contentType=nosuch", "", http.StatusNotFound,
		`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)
}

func TestNeedsYouIsOneEntryPerItemAndEditionInRankOrder(t *testing.T) {
	s := stock(t, true)
	w := s.add(t, "mixed", "Mixed", `,"monitor":true`, "a", "b", "c", "d", "e", "f", "g")
	a, b, c, d, e, f, g := w.Items[0], w.Items[1], w.Items[2], w.Items[3], w.Items[4], w.Items[5], w.Items[6]
	failedA := s.grab(a, s.hd.ID, files.GrabCell{State: files.GrabFailed})
	for _, it := range []works.Item{b, c, e} {
		s.placeAt(t, it, w, "Mixed/"+it.Label+" - HD.mkv", release.WEBDL1080p, 1, s.hd.ID)
		s.placeAt(t, it, w, "Mixed/"+it.Label+" - 2160p.mkv", release.WEBDL2160p, 1, s.uhd.ID)
	}
	s.placeAt(t, d, w, "Mixed/d - HD.mkv", release.Bluray1080p, 1, s.hd.ID)
	s.placeAt(t, d, w, "Mixed/d - 2160p.mkv", release.WEBDL2160p, 1, s.uhd.ID)
	waits := s.grab(f, s.hd.ID, files.GrabCell{State: files.GrabDownloading, Waiting: "node_away"})
	failedG := s.grab(g, s.uhd.ID, files.GrabCell{State: files.GrabFailed})

	hd, uhd := s.hd.ID.String(), s.uhd.ID.String()
	want := `[` +
		`{"kind":"failed","itemIds":["` + a.ID.String() + `"],"edition":"any","targetId":"` + hd + `","waiting":["` + uhd + `"],"grabId":"` + failedA.String() + `"},` +
		`{"kind":"failed","itemIds":["` + g.ID.String() + `"],"edition":"any","targetId":"` + uhd + `","waiting":["` + hd + `"],"grabId":"` + failedG.String() + `"},` +
		`{"kind":"upgrade","itemIds":["` + b.ID.String() + `","` + c.ID.String() + `"],"edition":"any","upgrades":["` + hd + `"]},` +
		`{"kind":"upgrade","itemIds":["` + e.ID.String() + `"],"edition":"any","upgrades":["` + hd + `"]},` +
		`{"kind":"waiting","grabId":"` + waits.String() + `","code":"node_away"}]`
	if got := s.needsYou(t, w); got != want {
		t.Fatalf("Needs you is\n%s\nwant\n%s", got, want)
	}
	r := s.rows(t, "?workIds="+w.ID.String())
	if len(r) != 1 || r[0].NeedsYou != 5 || r[0].Failed != 2 || *r[0].Status != "missing" {
		t.Fatalf("the row is %+v, want 5 Needs you, 2 failed, missing", r)
	}
	if all := s.rows(t, ""); !reflect.DeepEqual(all, r) {
		t.Fatalf("the whole Library is %+v, want the one row read by id, %+v", all, r)
	}

	rootless := stock(t, false)
	lost := rootless.add(t, "lost", "Lost", `,"monitor":true`, "a")
	if got := rootless.needsYou(t, lost); got != `[{"kind":"no_root"}]` {
		t.Fatalf("a rootless monitored Work's Needs you is %s, want no_root", got)
	}
	idle := rootless.add(t, "idle", "Idle", "", "a")
	if got := rootless.needsYou(t, idle); got != `[]` {
		t.Fatalf("a rootless unmonitored Work's Needs you is %s, want none: it wants nothing", got)
	}
	if r := rootless.rows(t, "?workIds="+lost.ID.String()); len(r) != 1 || r[0].RootID != nil || r[0].NeedsYou != 1 {
		t.Fatalf("the rootless row is %+v, want no root and 1 Needs you", r)
	}
}

func (s *shelved) needsYou(t *testing.T, w works.Tree) string {
	t.Helper()
	code, body := s.call(t, http.MethodGet, "/api/v1/works/"+w.ID.String()+"/cells", "")
	var out struct {
		NeedsYou json.RawMessage `json:"needsYou"`
	}
	if err := json.Unmarshal([]byte(body), &out); code != http.StatusOK || err != nil {
		t.Fatalf("getWorkCells = %d %s", code, body)
	}
	return string(out.NeedsYou)
}

func TestListWorkFilesListsFillingFilesInProfileOrderThenKeptOnes(t *testing.T) {
	s := stock(t, true)
	heat := s.add(t, "heat", "Heat", `,"monitor":true`, "Heat")
	uhd := s.place(t, heat, "Heat/Heat - 2160p.mkv", release.WEBDL2160p, 3000, s.uhd.ID)
	kept := s.place(t, heat, "Heat/Heat - HD (extra 1).mkv", release.WEBDL1080p, 500)
	hd := s.place(t, heat, "Heat/Heat - HD.mkv", release.Bluray1080p, 1000, s.hd.ID)
	item := heat.Items[0].ID.String()
	expect(t, s.server, http.MethodGet, "/api/v1/works/"+heat.ID.String()+"/files", "", http.StatusOK, `{"folder":"Heat","files":[`+
		`{"id":"`+hd.String()+`","path":"Heat/Heat - HD.mkv","size":1000,"quality":"bluray-1080p","probe":{"width":0,"height":0},"folder":"Heat","kept":false,`+
		`"fills":[{"itemId":"`+item+`","edition":"any","targetId":"`+s.hd.ID.String()+`"}]},`+
		`{"id":"`+uhd.String()+`","path":"Heat/Heat - 2160p.mkv","size":3000,"quality":"webdl-2160p","probe":{"width":0,"height":0},"folder":"Heat","kept":false,`+
		`"fills":[{"itemId":"`+item+`","edition":"any","targetId":"`+s.uhd.ID.String()+`"}]},`+
		`{"id":"`+kept.String()+`","path":"Heat/Heat - HD (extra 1).mkv","size":500,"quality":"webdl-1080p","probe":{"width":0,"height":0},"folder":"Heat","kept":true,"fills":[]}]}`)

	arrival := s.add(t, "arrival", "Arrival", "", "Arrival")
	expect(t, s.server, http.MethodGet, "/api/v1/works/"+arrival.ID.String()+"/files", "", http.StatusOK, `{"folder":"Arrival","files":[]}`)
	expect(t, s.server, http.MethodGet, "/api/v1/works/"+storage.NewUUID().String()+"/files", "", http.StatusNotFound,
		`{"status":404,"code":"work_not_found","title":"No Work has that id"}`)
}

func TestGetWorkAnswersItsProfileAndRoot(t *testing.T) {
	s := stock(t, true)
	heat := s.add(t, "heat", "Heat", "", "Heat")
	code, body := s.call(t, http.MethodGet, "/api/v1/works/"+heat.ID.String(), "")
	var got struct {
		Links   []any `json:"links"`
		Profile struct {
			ID, Name string
		} `json:"profile"`
		Root *struct {
			ID     string `json:"id"`
			NodeID string `json:"nodeId"`
			Path   string `json:"path"`
		} `json:"root"`
	}
	if err := json.Unmarshal([]byte(body), &got); code != http.StatusOK || err != nil || got.Links == nil || len(got.Links) != 0 ||
		got.Profile.ID != s.archive.ID.String() || got.Profile.Name != "Archive" ||
		got.Root == nil || got.Root.ID != s.root.ID.String() || got.Root.NodeID != s.root.Node.String() || got.Root.Path != s.root.Path {
		t.Fatalf("getWork = %d %s, want no links, Archive and the root %+v", code, body, s.root)
	}

	rootless := stock(t, false)
	lost := rootless.add(t, "lost", "Lost", "", "Lost")
	if code, body := rootless.call(t, http.MethodGet, "/api/v1/works/"+lost.ID.String(), ""); code != http.StatusOK || !strings.HasSuffix(body, `"root":null}`+"\n") {
		t.Fatalf("getWork of a rootless Work = %d %s, want root null", code, body)
	}
}

func (s *shelved) search(t *testing.T, query string) (ids, labels []string) {
	t.Helper()
	code, body := s.call(t, http.MethodGet, "/api/v1/works/search?"+query, "")
	var out struct {
		WorkIDs []string `json:"workIds"`
		Rows    []row    `json:"rows"`
	}
	if err := json.Unmarshal([]byte(body), &out); code != http.StatusOK || err != nil {
		t.Fatalf("searchWorks?%s = %d %s", query, code, body)
	}
	for _, r := range out.Rows {
		labels = append(labels, r.Label)
	}
	return out.WorkIDs, labels
}

func TestSearchWorksMatchesEveryTitleIgnoringCaseAndAccents(t *testing.T) {
	ctx := t.Context()
	s := stock(t, true)
	amelie := s.addRecord(t, "amelie", contenttypetest.Record{
		Tree:   works.Fetched{Label: "Amélie", SortLabel: "Amelie", Children: []works.Entry{{Key: "b1", Label: "Box", Children: []works.Entry{{Key: "c1", Label: "Amélie"}}}}},
		Titles: []string{"Le Fabuleux Destin d'Amélie Poulain", "The Fabulous Destiny of Amélie Poulain"},
	}, "")
	heat := s.add(t, "heat", "Heat", "", "Heat")
	wave := s.add(t, "wave", "Heat Wave", "", "Heat Wave")
	ids := func(ws ...works.Tree) []string {
		out := make([]string, len(ws))
		for i, w := range ws {
			out[i] = w.ID.String()
		}
		return out
	}
	for _, c := range []struct {
		query  string
		ids    []string
		labels []string
	}{
		{"q=amelie", ids(amelie), []string{"Amélie"}},
		{"q=AMÉLIE", ids(amelie), []string{"Amélie"}},
		{"q=fabuleux+dest", ids(amelie), []string{"Amélie"}},
		{"q=amelie+po", ids(amelie), []string{"Amélie"}},
		{"q=heat", ids(heat, wave), []string{"Heat", "Heat Wave"}},
		{"q=heat&limit=1", ids(heat, wave), []string{"Heat"}},
		{"q=heat&limit=0", ids(heat, wave), nil},
		{"q=wave+heat", ids(wave), []string{"Heat Wave"}},
		{"q=heat&contentType=fixture", ids(heat, wave), []string{"Heat", "Heat Wave"}},
		{"q=eat", nil, nil},
		{`q="OR"+NEAR(`, nil, nil},
		{"q=%21%21", nil, nil},
	} {
		gotIDs, gotLabels := s.search(t, c.query)
		if !slices.Equal(gotIDs, c.ids) || !slices.Equal(gotLabels, c.labels) {
			t.Fatalf("searchWorks?%s = %v %v, want %v %v", c.query, gotIDs, gotLabels, c.ids, c.labels)
		}
	}
	expect(t, s.server, http.MethodGet, "/api/v1/works/search?q=heat&contentType=nosuch", "", http.StatusNotFound,
		`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`)

	s.fx.Put("amelie", contenttypetest.Record{
		Tree:   works.Fetched{Label: "Amélie", SortLabel: "Amelie", Children: []works.Entry{{Key: "b1", Label: "Box", Children: []works.Entry{{Key: "c1", Label: "Amélie"}}}}},
		Titles: []string{"Die fabelhafte Welt der Amélie"},
	})
	if code, body := s.call(t, http.MethodPost, "/api/v1/works/refresh", `{"workIds":["`+amelie.ID.String()+`"]}`); code != http.StatusOK || !strings.Contains(body, `"outcome":"done"`) {
		t.Fatalf("refreshWorks = %d %s", code, body)
	}
	if got, _ := s.search(t, "q=fabelhafte"); !slices.Equal(got, ids(amelie)) {
		t.Fatalf("after a refresh brought a new title, searching it answers %v, want Amélie", got)
	}
	if got, _ := s.search(t, "q=fabuleux"); len(got) != 0 {
		t.Fatalf("after a refresh dropped a title, searching it answers %v, want none", got)
	}

	err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if _, err := tx.ExecContext(ctx, "DELETE FROM works_work WHERE id = ?", heat.ID); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "DELETE FROM works_title WHERE work_id = ?", wave.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.search(t, "q=heat"); len(got) != 0 {
		t.Fatalf("with Heat deleted and Heat Wave out of the index, searching heat answers %v, want none", got)
	}
	if _, err := works.Open(ctx, works.Config{Library: s.lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: s.fx}, Source: s.fx.Source(), Artwork: s.fx}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.search(t, "q=heat"); !slices.Equal(got, ids(wave)) {
		t.Fatalf("once the library opens again, searching heat answers %v, want Heat Wave caught up and Heat gone", got)
	}
}
