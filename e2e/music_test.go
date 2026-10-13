package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/api/apitest"
)

const (
	lanterns  = "a1000000-0000-4000-8000-000000000001"
	marrowBay = "a1000000-0000-4000-8000-000000000002"
)

const musicType = `{"contentTypes":[{"id":"music","name":"Music","enabled":true,` +
	`"work":{"one":"Artist","many":"Artists"},"levels":[{"one":"Album","many":"Albums"}],"item":{"one":"Track","many":"Tracks"},` +
	`"fields":[{"key":"country","label":"Country","tier":"work","kind":"text"},` +
	`{"key":"released","label":"Released","tier":"level","depth":1,"kind":"date"},` +
	`{"key":"kind","label":"Kind","tier":"level","depth":1,"kind":"text"},` +
	`{"key":"disc","label":"Disc","tier":"item","kind":"number"},` +
	`{"key":"length","label":"Length","tier":"item","kind":"duration"}]}]}` + "\n"

func searchLanterns(workID string) worldtest.Reply {
	return worldtest.Reply{Status: 200, Body: `{"candidates":[{"key":"` + lanterns + `","label":"The Lanterns","fields":[{"key":"country","kind":"text","text":"GB"}],"workId":` + workID + `}]}` + "\n"}
}

func problem(status int, code, title string) worldtest.Reply {
	return worldtest.Reply{Status: status, Body: fmt.Sprintf(`{"status":%d,"code":%q,"title":%q}`, status, code, title) + "\n"}
}

type field struct {
	Key    string  `json:"key"`
	Kind   string  `json:"kind"`
	Text   string  `json:"text"`
	Number float64 `json:"number"`
	Date   string  `json:"date"`
	Ms     int64   `json:"ms"`
}

func (f field) String() string {
	return f.Key + "=" + map[string]string{"text": f.Text, "number": fmt.Sprint(f.Number), "date": f.Date, "duration": fmt.Sprint(f.Ms, "ms")}[f.Kind]
}

type treeNode struct {
	ID          string  `json:"id"`
	ParentID    *string `json:"parentId"`
	LevelID     *string `json:"levelId"`
	Depth       int     `json:"depth"`
	Key         string  `json:"key"`
	Label       string  `json:"label"`
	Position    int     `json:"position"`
	WithdrawnAt *string `json:"withdrawnAt"`
	Fields      []field `json:"fields"`
}

type tree struct {
	ID          string     `json:"id"`
	ContentType string     `json:"contentType"`
	Key         string     `json:"key"`
	Label       string     `json:"label"`
	SortLabel   string     `json:"sortLabel"`
	Fields      []field    `json:"fields"`
	Levels      []treeNode `json:"levels"`
	Items       []treeNode `json:"items"`
}

func (tr tree) outline(ids *[]string) string {
	name := func(id string) string {
		for i, seen := range *ids {
			if seen == id {
				return fmt.Sprintf("#%d", i+1)
			}
		}
		*ids = append(*ids, id)
		return fmt.Sprintf("#%d", len(*ids))
	}
	line := func(n treeNode) string {
		s := fmt.Sprintf("%s %d %s %v", name(n.ID), n.Position, n.Label, n.Fields)
		if n.WithdrawnAt != nil {
			s += " withdrawn"
		}
		return s
	}
	out := []string{fmt.Sprintf("%s %s %s (%s) %s %v", name(tr.ID), tr.ContentType, tr.Label, tr.SortLabel, tr.Key, tr.Fields)}
	for _, l := range tr.Levels {
		out = append(out, "  "+line(l))
		for _, it := range tr.Items {
			if it.LevelID != nil && *it.LevelID == l.ID {
				out = append(out, "    "+line(it))
			}
		}
	}
	return strings.Join(out, "\n")
}

type added struct {
	Created bool            `json:"created"`
	Work    json.RawMessage `json:"work"`
}

func decode[T any](t *testing.T, what string, r worldtest.Reply, v *T) *T {
	t.Helper()
	if r.Status != http.StatusOK {
		t.Fatalf("%s = %d %s, want 200", what, r.Status, r.Body)
	}
	if err := json.Unmarshal([]byte(r.Body), v); err != nil {
		t.Fatalf("%s = %s: %v", what, r.Body, err)
	}
	return v
}

func checkOutline(t *testing.T, what string, tr tree, ids *[]string, want string) {
	t.Helper()
	if got := tr.outline(ids); got != want {
		t.Fatalf("%s:\n%s\nwant\n%s", what, got, want)
	}
}

func event(t *testing.T, s *apitest.Stream, name, data string) {
	t.Helper()
	named := regexp.MustCompile(`(?m)^event: ` + regexp.QuoteMeta(name) + `$`)
	want := regexp.MustCompile(`^id: \d+\n(retry: \d+\n)?event: ` + regexp.QuoteMeta(name) + `\ndata: ` + regexp.QuoteMeta(data) + `\n$`)
	ctx := worldtest.Context(t)
	for {
		msg, err := s.Next(ctx)
		if err != nil {
			t.Fatalf("the stream ended before %s: %v", name, err)
		}
		if !named.MatchString(msg) {
			continue
		}
		if !want.MatchString(msg) {
			t.Fatalf("the stream read %q, want event %s with data %s", msg, name, data)
		}
		return
	}
}

var refreshedAt = regexp.MustCompile(`"refreshedAt":"[^"]+"`)

func TestMusic(t *testing.T) {
	t.Parallel()
	run := worldtest.LaunchE2E(t, "music")
	a := api(t, run)
	session := a.SignIn("home")

	worldtest.Expect(t, "listContentTypes", a.Signed("home", session, http.MethodGet, "/api/v1/content-types", ""), worldtest.Reply{Status: 200, Body: musicType})

	worldtest.Expect(t, "searchCatalog for The Lanterns", a.Admin("home", http.MethodGet, "/api/v1/content-types/music/catalog?query=Lanterns", ""), searchLanterns("null"))

	stream := a.Stream("home", session)
	first := decode(t, "addWork", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"music","key":"`+lanterns+`"}`), &added{})
	var ids []string
	var tr tree
	if err := json.Unmarshal(first.Work, &tr); err != nil || !first.Created {
		t.Fatalf("addWork created %t, work %s: %v", first.Created, first.Work, err)
	}
	checkOutline(t, "the added tree", tr, &ids, strings.Join([]string{
		"#1 music The Lanterns (Lanterns, The) " + lanterns + " [country=GB]",
		"  #2 1 Harbor Lights [released=2001-03-12 kind=Album]",
		"    #3 1 Tidewater [disc=1 length=205000ms]",
		"    #4 2 Paper Boats [disc=1 length=242000ms]",
		"    #5 3 Low Bridge [disc=1 length=178000ms]",
		"  #6 2 Night Ferry [released=2003 kind=EP]",
		"    #7 1 Night Ferry [disc=1 length=190000ms]",
		"    #8 2 Signal Fires [disc=1 length=224000ms]",
	}, "\n"))
	event(t, stream, "works.changed", `{"workId":"`+tr.ID+`"}`)

	again := decode(t, "addWork again", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"music","key":"`+lanterns+`"}`), &added{})
	if again.Created || string(again.Work) != string(first.Work) {
		t.Fatalf("addWork again created %t with work\n%s\nwant false with\n%s", again.Created, again.Work, first.Work)
	}
	worldtest.Expect(t, "searchCatalog once added", a.Admin("home", http.MethodGet, "/api/v1/content-types/music/catalog?query=Lanterns", ""), searchLanterns(`"`+tr.ID+`"`))

	refresh := "/api/v1/works/" + tr.ID + "/refresh"
	refreshed := a.Admin("home", http.MethodPost, refresh, "")
	checkOutline(t, "the refreshed tree", *decode(t, "refreshWork", refreshed, &tree{}), &ids, strings.Join([]string{
		"#1 music The Lanterns (Lanterns, The) " + lanterns + " [country=GB]",
		"  #2 1 Harbor Lights [released=2001-03-12 kind=Album]",
		"    #3 1 Tidewater [disc=1 length=205000ms]",
		"    #4 2 Paper Ships [disc=1 length=242000ms]",
		"    #9 3 Lantern Song [disc=1 length=301000ms]",
		"    #5 3 Low Bridge [disc=1 length=178000ms] withdrawn",
		"  #6 2 Night Ferry [released=2003 kind=EP]",
		"    #7 1 Night Ferry [disc=1 length=190000ms]",
		"    #8 2 Signal Fires [disc=1 length=224000ms]",
	}, "\n"))
	twice := a.Admin("home", http.MethodPost, refresh, "")
	if twice.Status != http.StatusOK || refreshedAt.ReplaceAllString(twice.Body, "") != refreshedAt.ReplaceAllString(refreshed.Body, "") {
		t.Fatalf("a second refreshWork = %d\n%s\nwant the first's body apart from refreshedAt\n%s", twice.Status, twice.Body, refreshed.Body)
	}

	addBay := `{"contentType":"music","key":"` + marrowBay + `"}`
	worldtest.Expect(t, "addWork for Marrow Bay", a.Admin("home", http.MethodPost, "/api/v1/works", addBay), problem(502, "metadata_unavailable", "The metadata source didn't answer. Try again later."))
	bay := decode(t, "addWork for Marrow Bay again", a.Admin("home", http.MethodPost, "/api/v1/works", addBay), &added{})
	var bayTree tree
	if err := json.Unmarshal(bay.Work, &bayTree); err != nil || !bay.Created {
		t.Fatalf("addWork for Marrow Bay again created %t, work %s: %v", bay.Created, bay.Work, err)
	}
	checkOutline(t, "Marrow Bay's tree", bayTree, &ids, strings.Join([]string{
		"#10 music Marrow Bay (Marrow Bay) " + marrowBay + " [country=NZ]",
		"  #11 1 Saltgrass [released=2019-11 kind=Album]",
		"    #12 1 Saltgrass [disc=1]",
	}, "\n"))
	worldtest.Expect(t, "addWork for an unknown MBID", a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"music","key":"ffffffff-0000-4000-8000-000000000000"}`),
		problem(404, "metadata_not_found", "The metadata source has no record with that key"))

	worldtest.Expect(t, "turning music off while it holds Works", a.Admin("home", http.MethodPut, "/api/v1/content-types/music/enabled", `{"enabled":false}`),
		problem(409, "content_type_in_use", "Remove its Works before turning this Content Type off"))
	t.Run("a fresh run with no Works", turnOffAndOn)
}

func turnOffAndOn(t *testing.T) {
	s, err := scenario.Load("music")
	if err != nil {
		t.Fatal(err)
	}
	s.Name, s.Music.Faults = "music-empty", nil
	run := worldtest.LaunchE2EScenario(t, s)
	a := api(t, run)
	session := a.SignIn("home")
	stream := a.Stream("home", session)
	musicOff := strings.Replace(strings.TrimSuffix(strings.TrimPrefix(musicType, `{"contentTypes":[`), "]}\n"), `"enabled":true`, `"enabled":false`, 1) + "\n"

	worldtest.Expect(t, "turning music off", a.Admin("home", http.MethodPut, "/api/v1/content-types/music/enabled", `{"enabled":false}`), worldtest.Reply{Status: 200, Body: musicOff})
	event(t, stream, "works.content-types-changed", `{"contentType":"music"}`)
	worldtest.Expect(t, "searchCatalog while off", a.Admin("home", http.MethodGet, "/api/v1/content-types/music/catalog?query=Lanterns", ""),
		problem(409, "content_type_disabled", "That Content Type is turned off"))

	worldtest.Expect(t, "turning music on", a.Admin("home", http.MethodPut, "/api/v1/content-types/music/enabled", `{"enabled":true}`),
		worldtest.Reply{Status: 200, Body: strings.Replace(musicOff, `"enabled":false`, `"enabled":true`, 1)})
	event(t, stream, "works.content-types-changed", `{"contentType":"music"}`)
	worldtest.Expect(t, "searchCatalog once on again", a.Admin("home", http.MethodGet, "/api/v1/content-types/music/catalog?query=Lanterns", ""), searchLanterns("null"))
}
