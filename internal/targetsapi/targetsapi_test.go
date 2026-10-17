package targetsapi_test

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/targetsapi"
	"github.com/jatassi/slipstream/internal/works"
)

type published struct{ n int }

func (p *published) Publish(context.Context, targets.Changed) { p.n++ }

type goals struct{ t *targets.Targets }

func (g goals) Targets() (*targets.Targets, bool) { return g.t, true }

func serve(t *testing.T) (*ownertest.Owner, *published) {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema(), targets.Schema())
	changed := &published{}
	svc := targets.New(targets.Config{Library: lib, Types: []works.TypeID{contenttypetest.ID}, Changed: changed})
	return ownertest.Serve(t, lib, func(r *api.Router) { targetsapi.Routes(r, goals{svc}) }), changed
}

var times = regexp.MustCompile(`"\d{4}-\d{2}-\d{2}T[0-9:.]+Z"`)

// get compares the reply to a GET of path with want, in which each @name
// stands for the id ids names.
func get(t *testing.T, s *ownertest.Owner, path string, want string, ids map[string]string) {
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
	s, changed := serve(t)
	ids := map[string]string{}
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

	_, body = s.Call(t, http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["`+ids["hd"]+`","`+ids["web"]+`"]}`)
	ids["standard"] = ownertest.FirstID(body)
	_, body = s.Call(t, http.MethodPost, "/api/v1/profiles", `{"name":"Small","contentType":"fixture","targetIds":["`+ids["web"]+`"]}`)
	ids["small"] = ownertest.FirstID(body)
	get(t, s, "/api/v1/targets", `{"targets":[`+
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
	if changed.n != 7 {
		t.Fatalf("targets.changed published %d times, want 7", changed.n)
	}
}

func TestEachTargetsErrorAnswersItsProblem(t *testing.T) {
	s, changed := serve(t)
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
			`{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.tiers[0][0]","message":"\"webdl-1440p\" is not a quality"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"],[]],"cutoff":"webdl-1080p"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields are invalid","detail":"tiers are missing, empty or repeat a quality: tier 2 is empty","errors":[{"location":"body.tiers","message":"tiers are missing, empty or repeat a quality: tier 2 is empty"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"],["webdl-1080p"]],"cutoff":"webdl-1080p"}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields are invalid","detail":"tiers are missing, empty or repeat a quality: webdl-1080p twice","errors":[{"location":"body.tiers","message":"tiers are missing, empty or repeat a quality: webdl-1080p twice"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets", `{"name":"X","tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p","sizes":[{"quality":"webdl-1080p","min":20,"max":10}]}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields are invalid","detail":"a size is negative or its maximum is below its minimum: webdl-1080p","errors":[{"location":"body.sizes","message":"a size is negative or its maximum is below its minimum: webdl-1080p"}]}`,
		},
		{
			http.MethodPost, "/api/v1/targets",
			`{"name":"X","tiers":[["webdl-1080p"]],"cutoff":"webdl-1080p","filters":[{"attribute":"video_codec","mode":"only","values":["TrueHD"]},{"attribute":"video_codec","mode":"not","values":["x264"]}],"sizes":[{"quality":"webdl-1080p","min":1,"max":null},{"quality":"webdl-1080p","min":2,"max":null}]}`,
			http.StatusUnprocessableEntity,
			`{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[` +
				`{"location":"body.filters[0].values","message":"a filter is Any with values, Only or Not with none, or names a value twice: \"TrueHD\" is not a video codec"},` +
				`{"location":"body.filters[1].attribute","message":"the attribute is listed twice"},` +
				`{"location":"body.sizes[1].quality","message":"the quality is listed twice"}]}`,
		},
		{http.MethodGet, "/api/v1/targets/" + none, "", http.StatusNotFound, `{"status":404,"code":"target_not_found","title":"No Target has that id"}`},
		{http.MethodPost, "/api/v1/targets/" + none + "/duplicate", "", http.StatusNotFound, `{"status":404,"code":"target_not_found","title":"No Target has that id"}`},
		{http.MethodDelete, "/api/v1/targets/" + none, "", http.StatusNotFound, `{"status":404,"code":"target_not_found","title":"No Target has that id"}`},
		{
			http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"ghost","targetIds":["` + id + `"]}`, http.StatusNotFound,
			`{"status":404,"code":"content_type_not_found","title":"No Content Type has that id"}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["` + none + `"]}`, http.StatusUnprocessableEntity,
			`{"status":422,"code":"target_not_found","title":"No Target has that id"}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["` + id + `"]}`, http.StatusCreated,
			`{"id":"id1","name":"Standard","contentType":"fixture","default":true,"targetIds":["id2"]}`,
		},
		{
			http.MethodPost, "/api/v1/profiles", `{"name":"Standard","contentType":"fixture","targetIds":["` + id + `"]}`, http.StatusConflict,
			`{"status":409,"code":"profile_name_taken","title":"A Profile already has that name"}`,
		},
	} {
		s.Expect(t, c.method, c.path, c.body, c.status, c.want)
	}
	if changed.n != 2 {
		t.Fatalf("targets.changed published %d times, want 2", changed.n)
	}
}

func TestListQualitiesAnswersTheTableAndThePresets(t *testing.T) {
	s, _ := serve(t)
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
