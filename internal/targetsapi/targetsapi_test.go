package targetsapi_test

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
	"github.com/jatassi/slipstream/internal/works"
)

type published struct{ n int }

func (p *published) Publish(context.Context, targets.Changed) { p.n++ }

type goals struct {
	t *targets.Targets
	f *files.Files
}

func (g goals) Targets() (*targets.Targets, bool) { return g.t, true }

func (g goals) Files() (*files.Files, bool) { return g.f, true }

type served struct {
	*ownertest.Owner
	lib     *storage.DB[storage.Library]
	changed *published
	works   *works.Works
	fx      *contenttypetest.Fixture
	// starter and fixture are the ids of the starter Target and the
	// fixture type's default Profile, which Open made.
	starter, fixture string
}

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

func serve(t *testing.T) *served {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(), grabs.Schema(), files.Schema())
	fx := contenttypetest.New()
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	changed := &published{}
	svc, err := targets.Open(t.Context(), targets.Config{Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: changed})
	if err != nil {
		t.Fatal(err)
	}
	if changed.n != 1 {
		t.Fatalf("Open on a fresh library published %d times, want once", changed.n)
	}
	changed.n = 0
	shelf := files.New(files.Config{
		Library: lib, Works: w, Targets: svc, Grabs: grabs.GrabCells{}, Naming: map[works.TypeID]naming.Facet{contenttypetest.ID: fx}, Today: time.Now,
	})
	s := &served{Owner: ownertest.Serve(t, lib, func(r *api.Router) { targetsapi.Routes(r, goals{svc, shelf}) }), lib: lib, changed: changed, works: w, fx: fx}
	_, body := s.Call(t, http.MethodGet, "/api/v1/targets", "")
	s.starter = ownertest.FirstID(body)
	_, body = s.Call(t, http.MethodGet, "/api/v1/profiles", "")
	s.fixture = ownertest.FirstID(body)
	return s
}

// starterRules is the starter Target's rules on the wire.
const starterRules = `"tiers":[["bluray-1080p","remux-1080p"],["hdtv-1080p","webrip-1080p","webdl-1080p"],["bluray-720p"],["hdtv-720p","webrip-720p","webdl-720p"]],` +
	`"cutoff":"bluray-1080p","filters":[],"sizes":[{"quality":"bluray-1080p","min":50.8,"max":null},{"quality":"remux-1080p","min":102,"max":null},` +
	`{"quality":"hdtv-1080p","min":33.8,"max":null},{"quality":"webrip-1080p","min":12.5,"max":null},{"quality":"webdl-1080p","min":12.5,"max":null},` +
	`{"quality":"bluray-720p","min":25.7,"max":null},{"quality":"hdtv-720p","min":17.1,"max":null},{"quality":"webrip-720p","min":12.5,"max":null},` +
	`{"quality":"webdl-720p","min":12.5,"max":null}]`

// profileBody is a createProfile body for Profile name of the fixture
// type holding the Targets with ids, downloaded.
func profileBody(name string, ids ...string) string {
	var entries []string
	for _, id := range ids {
		entries = append(entries, `{"targetId":"`+id+`","sourcing":"download"}`)
	}
	return `{"name":"` + name + `","contentType":"fixture","targets":[` + strings.Join(entries, ",") + `],"editions":{"kind":"any"}}`
}

func must[T any](t *testing.T) func(T, error) T {
	return func(v T, err error) T {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
}

var times = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T[0-9:.]+Z"`)

// get compares the reply to a GET of path with want, in which each @name
// stands for the id ids names.
func get(t *testing.T, s *served, path, want string, ids map[string]string) {
	t.Helper()
	for name, id := range ids {
		want = strings.ReplaceAll(want, "@"+name, id)
	}
	code, got := s.Call(t, http.MethodGet, path, "")
	if got = times.ReplaceAllString(strings.TrimSuffix(got, "\n"), `"T"`); code != http.StatusOK || got != want {
		t.Fatalf("GET %s = %d\n%s\nwant 200\n%s", path, code, got, want)
	}
}

const (
	hdBody = `{"name":"HD","tiers":[["bluray-1080p","remux-1080p"],["webdl-1080p","webrip-1080p"]],"cutoff":"remux-1080p",` +
		`"filters":[{"attribute":"language","mode":"not","values":["deu"]},{"attribute":"hdr","mode":"only","values":["SDR","HDR10"]}],` +
		`"sizes":[{"quality":"webdl-1080p","min":10,"max":40}]}`
	hdRules = `"tiers":[["bluray-1080p","remux-1080p"],["webdl-1080p","webrip-1080p"]],"cutoff":"bluray-1080p",` +
		`"filters":[{"attribute":"hdr","mode":"only","values":["SDR","HDR10"]},{"attribute":"language","mode":"not","values":["de"]}],` +
		`"sizes":[{"quality":"bluray-1080p","min":50.8,"max":null},{"quality":"remux-1080p","min":102,"max":null},` +
		`{"quality":"webdl-1080p","min":10,"max":40},{"quality":"webrip-1080p","min":12.5,"max":null}]`
	webBody = `{"name":"Web","tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p",` +
		`"filters":[{"attribute":"hdr","mode":"only","values":["SDR"]},{"attribute":"language","mode":"not","values":["de","fr"]}],` +
		`"sizes":[{"quality":"webdl-1080p","min":12,"max":30}]}`
	webRules = `"tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p",` +
		`"filters":[{"attribute":"hdr","mode":"only","values":["SDR"]},{"attribute":"language","mode":"not","values":["de","fr"]}],` +
		`"sizes":[{"quality":"webdl-1080p","min":12,"max":30}]`
)

func TestTargetsAreCreatedReadDuplicatedAndDeletedWithTheirRules(t *testing.T) {
	s := serve(t)
	ids := map[string]string{"starter": s.starter, "fixture": s.fixture}
	_, body := s.Call(t, http.MethodPost, "/api/v1/targets", hdBody)
	ids["hd"] = ownertest.FirstID(body)
	get(t, s, "/api/v1/targets/"+ids["hd"],
		`{"id":"@hd","name":"HD",`+hdRules+`,"profiles":[],"contains":[],"createdAt":"T"}`, ids)
	_, body = s.Call(t, http.MethodPost, "/api/v1/targets", webBody)
	ids["web"] = ownertest.FirstID(body)
	get(t, s, "/api/v1/targets/"+ids["web"],
		`{"id":"@web","name":"Web",`+webRules+`,"profiles":[],"contains":[],"createdAt":"T"}`, ids)

	_, body = s.Call(t, http.MethodPost, "/api/v1/targets/"+ids["hd"]+"/duplicate", "")
	ids["copy"] = ownertest.FirstID(body)
	get(t, s, "/api/v1/targets/"+ids["copy"],
		`{"id":"@copy","name":"HD copy",`+hdRules+`,"profiles":[],"contains":["@hd","@web"],"createdAt":"T"}`, ids)
	_, body = s.Call(t, http.MethodPost, "/api/v1/targets/"+ids["hd"]+"/duplicate", "")
	ids["again"] = ownertest.FirstID(body)

	_, body = s.Call(t, http.MethodPost, "/api/v1/profiles", profileBody("Standard", ids["hd"], ids["web"]))
	ids["standard"] = ownertest.FirstID(body)
	_, body = s.Call(t, http.MethodPost, "/api/v1/profiles", profileBody("Small", ids["web"]))
	ids["small"] = ownertest.FirstID(body)
	get(t, s, "/api/v1/targets", `{"targets":[`+
		`{"id":"@starter","name":"1080p",`+starterRules+`,"profiles":[{"id":"@fixture","name":"Fixture"}],"contains":[],"createdAt":"T"},`+
		`{"id":"@hd","name":"HD",`+hdRules+`,"profiles":[{"id":"@standard","name":"Standard"}],"contains":["@copy","@again","@web"],"createdAt":"T"},`+
		`{"id":"@copy","name":"HD copy",`+hdRules+`,"profiles":[],"contains":["@hd","@again","@web"],"createdAt":"T"},`+
		`{"id":"@again","name":"HD copy 2",`+hdRules+`,"profiles":[],"contains":["@hd","@copy","@web"],"createdAt":"T"},`+
		`{"id":"@web","name":"Web",`+webRules+`,"profiles":[{"id":"@small","name":"Small"},{"id":"@standard","name":"Standard"}],"contains":[],"createdAt":"T"}]}`, ids)

	s.Expect(t, http.MethodDelete, "/api/v1/targets/"+ids["hd"], "", http.StatusConflict,
		`{"status":409,"code":"target_in_use","title":"Profiles hold the Target","detail":"Standard uses it"}`)
	s.Expect(t, http.MethodDelete, "/api/v1/targets/"+ids["web"], "", http.StatusConflict,
		`{"status":409,"code":"target_in_use","title":"Profiles hold the Target","detail":"Small and Standard use it"}`)
	if code, body := s.Call(t, http.MethodDelete, "/api/v1/targets/"+ids["copy"], ""); code != http.StatusNoContent || body != "" {
		t.Fatalf("deleteTarget = %d %q, want 204", code, body)
	}
	s.Expect(t, http.MethodGet, "/api/v1/targets/"+ids["copy"], "", http.StatusNotFound,
		`{"status":404,"code":"target_not_found","title":"No Target has that id"}`)
	if s.changed.n != 7 {
		t.Fatalf("targets.changed published %d times, want 7", s.changed.n)
	}
}

func TestEachTargetsErrorAnswersItsProblem(t *testing.T) {
	s := serve(t)
	_, body := s.Call(t, http.MethodPost, "/api/v1/targets", webBody)
	id := ownertest.FirstID(body)
	none := storage.NewUUID().String()
	for _, c := range []struct {
		method, path, body string
		status             int
		want               string
	}{
		{http.MethodPost, "/api/v1/targets", webBody, http.StatusConflict, `{"status":409,"code":"target_name_taken","title":"A Target already has that name"}`},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"]],"cutoff":"bluray-1080p"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"quality_not_in_target","title":"The Target doesn't accept that quality","detail":"the Target doesn't accept that quality: the cutoff bluray-1080p"}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p","sizes":[{"quality":"cam","min":0,"max":null}]}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"quality_not_in_target","title":"The Target doesn't accept that quality","detail":"the Target doesn't accept that quality: a size for cam"}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1440p"]],"cutoff":"webdl-1080p"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"body.tiers[0][0]","message":"\"webdl-1440p\" is not a quality"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"],[]],"cutoff":"webdl-1080p"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"tiers are missing, empty or repeat a quality: tier 2 is empty","errors":[{"location":"body.tiers","message":"tiers are missing, empty or repeat a quality: tier 2 is empty"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"],["webdl-1080p"]],"cutoff":"webdl-1080p"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"tiers are missing, empty or repeat a quality: webdl-1080p twice","errors":[{"location":"body.tiers","message":"tiers are missing, empty or repeat a quality: webdl-1080p twice"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p","sizes":[{"quality":"webdl-1080p","min":20,"max":10}]}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"a size is negative or its maximum is below its minimum: webdl-1080p","errors":[{"location":"body.sizes","message":"a size is negative or its maximum is below its minimum: webdl-1080p"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets",
			`{"name":"X","tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p","filters":[{"attribute":"video_codec","mode":"only","values":["TrueHD"]},{"attribute":"video_codec","mode":"not","values":["x264"]}],"sizes":[{"quality":"webdl-1080p","min":1,"max":null},{"quality":"webdl-1080p","min":2,"max":null}]}`,
			http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[` +
				`{"location":"body.filters[0].values","message":"a filter is Any with values, Only or Not with none, or names a value twice: \"TrueHD\" is not a video codec"},` +
				`{"location":"body.filters[1].attribute","message":"the attribute is listed twice"},` +
				`{"location":"body.sizes[1].quality","message":"the quality is listed twice"}]}`,
		},
		{http.MethodGet, "/api/v1/targets/" + none, "", http.StatusNotFound, `{"status":404,"code":"target_not_found","title":"No Target has that id"}`},
		{http.MethodPost, "/api/v1/targets/" + none + "/duplicate", "", http.StatusNotFound, `{"status":404,"code":"target_not_found","title":"No Target has that id"}`},
		{http.MethodDelete, "/api/v1/targets/" + none, "", http.StatusNotFound, `{"status":404,"code":"target_not_found","title":"No Target has that id"}`},
		{
			http.MethodPost, "/api/v1/profiles", strings.Replace(profileBody("Standard", id), "fixture", "ghost", 1), http.StatusNotFound,
			`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", profileBody("Standard", none), http.StatusUnprocessableEntity,
			`{"status":422,"code":"target_not_found","title":"No Target has that id"}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", profileBody("Fixture"), http.StatusConflict,
			`{"status":409,"code":"profile_name_taken","title":"A Profile already has that name"}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", `{"name":"Twice","contentType":"fixture","targets":[{"targetId":"` + id + `","sourcing":"download"},{"targetId":"` + strings.ToUpper(id) + `","sourcing":"download"}],"editions":{"kind":"any"}}`,
			http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"a Profile holds a Target twice","errors":[{"location":"body.targets","message":"a Profile holds a Target twice"}]}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targets":[{"targetId":"` + id + `","sourcing":"derive"}],"editions":{"kind":"any"}}`,
			http.StatusUnprocessableEntity, `@invalid`,
		},
		{
			http.MethodPost, "/api/v1/profiles", profileBody("Standard", id), http.StatusCreated,
			`{"id":"id1","name":"Standard","contentType":"fixture","default":false,"targets":[{"targetId":"id2","sourcing":"download"}],"editions":{"kind":"any"},"works":0}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", profileBody("Standard", id), http.StatusConflict,
			`{"status":409,"code":"profile_name_taken","title":"A Profile already has that name"}`,
		},
		{http.MethodGet, "/api/v1/profiles/" + none, "", http.StatusNotFound, `{"status":404,"code":"profile_not_found","title":"No Profile has that id"}`},
		{http.MethodPost, "/api/v1/profiles/" + none + "/default", "", http.StatusNotFound, `{"status":404,"code":"profile_not_found","title":"No Profile has that id"}`},
		{http.MethodGet, "/api/v1/works/" + none + "/wants", "", http.StatusNotFound, `{"status":404,"code":"work_not_found","title":"No Work has that id"}`},
	} {
		if c.want == "@invalid" {
			if code, _ := s.Call(t, c.method, c.path, c.body); code != c.status {
				t.Errorf("%s %s %s = %d, want %d", c.method, c.path, c.body, code, c.status)
			}
			continue
		}
		s.Expect(t, c.method, c.path, c.body, c.status, c.want)
	}
	if s.changed.n != 2 {
		t.Fatalf("targets.changed published %d times, want 2", s.changed.n)
	}
}

func TestListQualitiesAnswersTheTableAndThePresets(t *testing.T) {
	s := serve(t)
	q := func(key, name, source, res string, floor string) string {
		r := `"` + res + `"`
		if res == "" {
			r = "null"
		}
		return `{"key":"` + key + `","name":"` + name + `","source":"` + source + `","resolution":` + r + `,"cinema":` +
			map[bool]string{true: "true", false: "false"}[source == "cinema"] + `,"floor":` + floor + `}`
	}
	qualities := strings.Join([]string{
		q("sdtv", "SDTV", "tv", "480p", "0"), q("dvd", "DVD", "dvd", "480p", "0"), q("webrip-480p", "WEBRip-480p", "webrip", "480p", "0"),
		q("hdtv-720p", "HDTV-720p", "hdtv", "720p", "17.1"), q("webrip-720p", "WEBRip-720p", "webrip", "720p", "12.5"),
		q("webdl-720p", "WEBDL-720p", "webdl", "720p", "12.5"), q("bluray-720p", "Bluray-720p", "bluray", "720p", "25.7"),
		q("hdtv-1080p", "HDTV-1080p", "hdtv", "1080p", "33.8"), q("webrip-1080p", "WEBRip-1080p", "webrip", "1080p", "12.5"),
		q("webdl-1080p", "WEBDL-1080p", "webdl", "1080p", "12.5"), q("bluray-1080p", "Bluray-1080p", "bluray", "1080p", "50.8"),
		q("remux-1080p", "Remux-1080p", "remux", "1080p", "102"), q("hdtv-2160p", "HDTV-2160p", "hdtv", "2160p", "85"),
		q("webrip-2160p", "WEBRip-2160p", "webrip", "2160p", "34.5"), q("webdl-2160p", "WEBDL-2160p", "webdl", "2160p", "34.5"),
		q("bluray-2160p", "Bluray-2160p", "bluray", "2160p", "102"), q("remux-2160p", "Remux-2160p", "remux", "2160p", "187.4"),
		q("cam", "CAM", "cinema", "", "0"),
	}, ",")
	presets := `{"key":"disc-or-resolution","name":"Upgrade for disc or resolution","tiers":[` +
		`["bluray-2160p","remux-2160p"],["hdtv-2160p","webrip-2160p","webdl-2160p"],["bluray-1080p","remux-1080p"],["hdtv-1080p","webrip-1080p","webdl-1080p"],` +
		`["bluray-720p"],["hdtv-720p","webrip-720p","webdl-720p"],["dvd"],["sdtv","webrip-480p"],["cam"]]},` +
		`{"key":"anything-better","name":"Upgrade for anything better","tiers":[` +
		`["remux-2160p"],["bluray-2160p"],["webdl-2160p"],["webrip-2160p"],["hdtv-2160p"],["remux-1080p"],["bluray-1080p"],["webdl-1080p"],["webrip-1080p"],["hdtv-1080p"],` +
		`["bluray-720p"],["webdl-720p"],["webrip-720p"],["hdtv-720p"],["webrip-480p"],["dvd"],["sdtv"],["cam"]]},` +
		`{"key":"resolution-only","name":"Upgrade for resolution only","tiers":[` +
		`["hdtv-2160p","webrip-2160p","webdl-2160p","bluray-2160p","remux-2160p"],["hdtv-1080p","webrip-1080p","webdl-1080p","bluray-1080p","remux-1080p"],` +
		`["hdtv-720p","webrip-720p","webdl-720p","bluray-720p"],["sdtv","dvd","webrip-480p"],["cam"]]}`
	s.Expect(t, http.MethodGet, "/api/v1/qualities", "", http.StatusOK, `{"qualities":[`+qualities+`],"presets":[`+presets+`]}`)
}

func TestProfilesAreListedWithTheirWorksMadeDefaultAndEstimated(t *testing.T) {
	ctx := t.Context()
	s := serve(t)
	_, body := s.Call(t, http.MethodPost, "/api/v1/targets", webBody)
	web := ownertest.FirstID(body)
	_, body = s.Call(t, http.MethodPost, "/api/v1/profiles", profileBody("Small", web, s.starter))
	small := ownertest.FirstID(body)
	_, body = s.Call(t, http.MethodPost, "/api/v1/profiles", profileBody("Empty"))
	empty := ownertest.FirstID(body)
	ids := map[string]string{"starter": s.starter, "fixture": s.fixture, "web": web, "small": small, "empty": empty}

	s.fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	tree, _, err := s.works.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{
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
	ids["work"] = tree.ID.String()
	s.fx.SetTiming(tree.Items[0].ID, targets.Timing{Released: release.Known(contenttypetest.Released), Runtime: release.Known(100 * time.Minute)})
	root := rootstest.Root(t, s.lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	err = s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: root.ID, Path: "a.mkv", Size: 2000 << 20, Quality: release.WEBDL1080p,
		}, []targets.TargetID{{UUID: must[storage.UUID](t)(storage.ParseUUID(s.starter))}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	entry := func(id string) string { return `{"targetId":"@` + id + `","sourcing":"download"}` }
	fixture := `{"id":"@fixture","name":"Fixture","contentType":"fixture","default":true,"targets":[` + entry("starter") + `],"editions":{"kind":"any"},"works":1}`
	get(t, s, "/api/v1/profiles", `{"profiles":[`+
		`{"id":"@empty","name":"Empty","contentType":"fixture","default":false,"targets":[],"editions":{"kind":"any"},"works":0},`+fixture+`,`+
		`{"id":"@small","name":"Small","contentType":"fixture","default":false,"targets":[`+entry("web")+`,`+entry("starter")+`],"editions":{"kind":"any"},"works":0}]}`, ids)
	get(t, s, "/api/v1/profiles/"+s.fixture, strings.TrimSuffix(fixture, "}")+`,"estimates":[{"targetId":"@starter","mibPerMinute":20,"bytes":2097152000}]}`, ids)
	get(t, s, "/api/v1/profiles/"+small, `{"id":"@small","name":"Small","contentType":"fixture","default":false,"targets":[`+entry("web")+`,`+entry("starter")+`],`+
		`"editions":{"kind":"any"},"works":0,"estimates":[{"targetId":"@web","mibPerMinute":null,"bytes":null},{"targetId":"@starter","mibPerMinute":20,"bytes":0}]}`, ids)
	get(t, s, "/api/v1/works/"+ids["work"]+"/wants", `{"profileId":"@fixture","profileName":"Fixture","editions":{"kind":"any"},"from":"profile"}`, ids)

	published := s.changed.n
	code, body := s.Call(t, http.MethodPost, "/api/v1/profiles/"+small+"/default", "")
	if code != http.StatusOK || !strings.Contains(body, `"name":"Small","contentType":"fixture","default":true`) {
		t.Fatalf("makeDefaultProfile = %d %s, want Small, now default", code, body)
	}
	s.Call(t, http.MethodPost, "/api/v1/profiles/"+small+"/default", "")
	if n := s.changed.n - published; n != 1 {
		t.Fatalf("making Small default twice published %d times, want once", n)
	}
	get(t, s, "/api/v1/profiles/"+s.fixture, strings.Replace(strings.TrimSuffix(fixture, "}"), `"default":true`, `"default":false`, 1)+
		`,"estimates":[{"targetId":"@starter","mibPerMinute":20,"bytes":2097152000}]}`, ids)
	get(t, s, "/api/v1/works/"+ids["work"]+"/wants", `{"profileId":"@fixture","profileName":"Fixture","editions":{"kind":"any"},"from":"profile"}`, ids)
}

func TestListCutsAnswersTheBuiltInCutsThenEachContentTypes(t *testing.T) {
	s := serve(t)
	s.fx.SetCuts([]release.Cut{{Key: "superfan", Name: "Superfan", Aliases: []string{"Superfan", "Superfan Edition"}}})
	builtIn := `{"key":"standard","name":"Standard","aliases":[],"origin":"builtin"},` +
		`{"key":"theatrical","name":"Theatrical","aliases":["Theatrical","Theatrical Cut","Theatrical Edition","Theatrical Version"],"origin":"builtin"},` +
		`{"key":"extended","name":"Extended","aliases":["Extended","Extended Cut","Extended Edition","Extended Version"],"origin":"builtin"},` +
		`{"key":"directors-cut","name":"Director's Cut","aliases":["Director's Cut","Director.s.Cut","Director Cut","Director's Edition","Directors Edition"],"origin":"builtin"},` +
		`{"key":"unrated","name":"Unrated","aliases":["Unrated"],"origin":"builtin"},` +
		`{"key":"uncut","name":"Uncut","aliases":["Uncut"],"origin":"builtin"},` +
		`{"key":"imax","name":"IMAX","aliases":["IMAX","IMAX Edition"],"origin":"builtin"},` +
		`{"key":"remastered","name":"Remastered","aliases":["Remastered"],"origin":"builtin"},` +
		`{"key":"special-edition","name":"Special Edition","aliases":["Special Edition"],"origin":"builtin"},` +
		`{"key":"final-cut","name":"Final Cut","aliases":["Final Cut"],"origin":"builtin"},` +
		`{"key":"ultimate","name":"Ultimate","aliases":["Ultimate","Ultimate Cut","Ultimate Edition"],"origin":"builtin"}`
	s.Expect(t, http.MethodGet, "/api/v1/cuts", "", http.StatusOK, `{"cuts":[`+builtIn+
		`,{"key":"superfan","name":"Superfan","aliases":["Superfan","Superfan Edition"],"origin":"type","contentType":"fixture"}]}`)
}
