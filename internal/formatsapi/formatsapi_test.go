package formatsapi_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/formatsapi"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
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
	"github.com/jatassi/slipstream/internal/transfers"
	"github.com/jatassi/slipstream/internal/works"
)

type goals struct {
	t *targets.Targets
	f *files.Files
	g *grabs.Grabs
}

func (g goals) Targets() (*targets.Targets, bool) { return g.t, true }

func (g goals) Files() (*files.Files, bool) { return g.f, true }

func (g goals) Grabs() (*grabs.Grabs, bool) { return g.g, true }

type discard[T any] struct{}

func (discard[T]) Publish(context.Context, T) {}

type served struct {
	*ownertest.Owner
	// starter is the id of the starter Target, which the fixture type's
	// default Profile holds.
	starter string
	// attic is the id of the one Work, The Attic (2020).
	attic string
}

// serve mounts the Custom Format and Target operations over a library
// holding one Work whose WEB-DL file fills the starter Target.
func serve(t *testing.T) *served {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), targets.Schema(), roots.Schema(),
		downloadclients.Schema(), grabs.Schema(), indexers.Schema(), files.Schema(), transfers.LibrarySchema())
	fx := contenttypetest.New()
	w, err := works.Open(ctx, works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{contenttypetest.ID: fx}, Source: fx.Source(), Artwork: fx,
		Changed: discard[works.Changed]{}, TypesChanged: discard[works.ContentTypesChanged]{},
	})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := targets.Open(ctx, targets.Config{
		Library: lib, Works: w, Types: map[works.TypeID]targets.Facet{contenttypetest.ID: fx}, Changed: discard[targets.Changed]{}, Formats: formats.DefaultTimings(),
	})
	if err != nil {
		t.Fatal(err)
	}
	shelf, err := files.Open(ctx, files.Config{
		Library: lib, Targets: svc, Grabs: grabs.GrabCells{InStep: func(role.NodeID) bool { return true }}, Naming: map[works.TypeID]naming.Facet{contenttypetest.ID: fx}, Today: time.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	g := grabs.New(grabs.Config{
		Library: lib, Works: w, Targets: svc, Files: shelf,
		Terms: map[works.TypeID]indexers.Facet{contenttypetest.ID: claims{w}}, Names: map[works.TypeID]release.Names{contenttypetest.ID: dated{}},
		Transfers: transfers.New(transfers.ControllerConfig{Library: lib}), Timings: grabs.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
	})
	s := &served{Owner: ownertest.Serve(t, lib, func(r *api.Router) {
		targetsapi.Routes(r, goals{svc, shelf, g})
		formatsapi.Routes(r, goals{svc, shelf, g})
	})}
	_, body := s.Call(t, http.MethodGet, "/api/v1/targets", "")
	s.starter = ownertest.FirstID(body)

	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{
		{Key: "b1", Label: "Winter", Children: []works.Entry{{Key: "c1", Label: "Scarf"}}},
	}}})
	tree, _, err := w.Add(ctx, identity.Admin{}, contenttypetest.ID, "w1", works.Adding{
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
	s.attic = tree.ID.String()
	root := rootstest.Root(t, lib, role.NodeID{UUID: storage.NewUUID()}, contenttypetest.ID, t.TempDir())
	starter, err := storage.ParseUUID(s.starter)
	if err != nil {
		t.Fatal(err)
	}
	err = lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: root.ID, Path: "The Attic/The Attic - Scarf - 1080p.mkv",
			Size: 2000 << 20, Quality: release.WEBDL1080p,
		}, []targets.TargetID{{UUID: starter}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var ids = regexp.MustCompile(`"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"`)

// call is the status and reply of a request, with every id in the reply
// written as "ID".
func (s *served) call(t *testing.T, method, path, body string) (status int, reply string) {
	t.Helper()
	status, reply = s.Call(t, method, path, body)
	return status, ids.ReplaceAllString(strings.TrimSuffix(reply, "\n"), `"ID"`)
}

func (s *served) expect(t *testing.T, method, path, body string, status int, want string) {
	t.Helper()
	if code, got := s.call(t, method, path, body); code != status || got != want {
		t.Fatalf("%s %s = %d\n%s\nwant %d\n%s", method, path, code, got, status, want)
	}
}

// scoreStarter gives the starter Target a score of 25 for format id.
func (s *served) scoreStarter(t *testing.T, id string) {
	t.Helper()
	_, body := s.Call(t, http.MethodGet, "/api/v1/targets/"+s.starter, "")
	var target map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &target); err != nil {
		t.Fatal(err)
	}
	for _, read := range []string{"id", "profiles", "contains", "createdAt"} {
		delete(target, read)
	}
	target["scores"] = json.RawMessage(`{"formats":[{"formatId":"` + id + `","score":25}],"minimum":0,"upgradeUntil":0,"increment":1}`)
	update, err := json.Marshal(target)
	if err != nil {
		t.Fatal(err)
	}
	if code, reply := s.Call(t, http.MethodPut, "/api/v1/targets/"+s.starter, string(update)); code != http.StatusOK {
		t.Fatalf("score the starter = %d %s", code, reply)
	}
}

const (
	hdrBody = `{"name":"HDR","includeWhenRenaming":true,"conditions":[` +
		`{"name":"HDR","kind":"ReleaseTitleSpecification","negate":false,"required":false,"fields":{"value":"\\bHDR\\b"}},` +
		`{"name":"Bluray","kind":"SourceSpecification","negate":false,"required":true,"fields":{"value":"bluray"}}]}`
	hdrConditions = `"conditions":[` +
		`{"name":"HDR","kind":"ReleaseTitleSpecification","negate":false,"required":false,"fields":{"value":"\\bHDR\\b"}},` +
		`{"name":"Bluray","kind":"SourceSpecification","negate":false,"required":true,"fields":{"value":"bluray"}}]`
	hdr       = `{"id":"ID","name":"HDR","includeWhenRenaming":true,"dialect":"slipstream",` + hdrConditions + `,"scoredBy":[],"slowReleases":0}`
	hdrScored = `{"id":"ID","name":"HDR","includeWhenRenaming":true,"dialect":"slipstream",` + hdrConditions + `,"scoredBy":[{"id":"ID","name":"1080p"}],"slowReleases":0}`
	noImpact  = `{"works":0,"ownList":0,"unwanted":{"files":0,"bytes":0},"renamed":0,"newlyWanted":0,"downloading":0}`
)

func TestCustomFormatsAreMadeReadChangedAndDeletedWithTheTargetsThatScoreThem(t *testing.T) {
	s := serve(t)
	s.expect(t, http.MethodPost, "/api/v1/custom-formats", hdrBody, http.StatusCreated, hdr)
	_, body := s.Call(t, http.MethodGet, "/api/v1/custom-formats", "")
	id := ownertest.FirstID(body)
	s.expect(t, http.MethodGet, "/api/v1/custom-formats", "", http.StatusOK, `{"formats":[`+hdr+`],"matchLimitMilliseconds":100}`)
	s.expect(t, http.MethodPost, "/api/v1/custom-formats", strings.Replace(hdrBody, `"HDR"`, `"hdr"`, 1), http.StatusConflict,
		`{"status":409,"code":"format_name_taken","title":"Another Custom Format has this name"}`)

	s.scoreStarter(t, id)
	s.expect(t, http.MethodGet, "/api/v1/custom-formats/"+id, "", http.StatusOK, hdrScored)

	renamed := strings.Replace(hdrBody, `"name":"HDR","include`, `"name":"HDR10","include`, 1)
	s.expect(t, http.MethodPut, "/api/v1/custom-formats/"+id+"?preview=true", renamed, http.StatusOK,
		`{"impact":`+noImpact+`,"format":`+hdrScored+`}`)
	s.expect(t, http.MethodGet, "/api/v1/custom-formats/"+id, "", http.StatusOK, hdrScored)
	s.expect(t, http.MethodPut, "/api/v1/custom-formats/"+id, renamed, http.StatusOK,
		`{"impact":`+noImpact+`,"format":`+strings.Replace(hdrScored, `"HDR","include`, `"HDR10","include`, 1)+`}`)

	s.expect(t, http.MethodDelete, "/api/v1/custom-formats/"+id+"?preview=true", "", http.StatusOK,
		`{"impact":`+noImpact+`,"scoredBy":[{"id":"ID","name":"1080p"}]}`)
	s.expect(t, http.MethodGet, "/api/v1/custom-formats", "", http.StatusOK,
		`{"formats":[`+strings.Replace(hdrScored, `"HDR","include`, `"HDR10","include`, 1)+`],"matchLimitMilliseconds":100}`)
	s.expect(t, http.MethodDelete, "/api/v1/custom-formats/"+id, "", http.StatusOK,
		`{"impact":`+noImpact+`,"scoredBy":[{"id":"ID","name":"1080p"}]}`)
	s.expect(t, http.MethodGet, "/api/v1/custom-formats/"+id, "", http.StatusNotFound,
		`{"status":404,"code":"format_not_found","title":"No Custom Format has that id"}`)
	if _, body := s.Call(t, http.MethodGet, "/api/v1/targets/"+s.starter, ""); !strings.Contains(body, `"scores":{"formats":[],`) {
		t.Errorf("the starter still scores the deleted format: %s", body)
	}
}

// trash is a TRaSH Guides document for format name, whose one condition
// is kind with fields.
func trash(name, kind, fields string) string {
	return `{"trash_id":"` + name + `-id","trash_scores":{"default":1750},"name":"` + name + `","includeCustomFormatWhenRenaming":false,` +
		`"specifications":[{"name":"` + name + `","implementation":"` + kind + `","negate":false,"required":true,"fields":` + fields + `}]}`
}

func pasteBody(text, extra string) string {
	b, err := json.Marshal(text)
	if err != nil {
		panic(err)
	}
	return `{"text":` + string(b) + extra + `}`
}

func TestAPasteAddsReplacesOrKeepsBothOfANameAndReadsInTheAppItWasWrittenFor(t *testing.T) {
	s := serve(t)
	_, body := s.Call(t, http.MethodPost, "/api/v1/custom-formats", hdrBody)
	id := ownertest.FirstID(body)
	text := `[` + trash("hdr", "QualityModifierSpecification", `{"value":5}`) + `,` + trash("Atmos", "ReleaseTitleSpecification", `{"value":"\\bATMOS\\b"}`) + `]`
	held := `{"id":"ID","name":"HDR","dialect":"slipstream"}`

	s.expect(t, http.MethodPost, "/api/v1/custom-formats/paste?preview=true", pasteBody(text, ""), http.StatusOK,
		`{"app":"radarr","formats":[{"name":"hdr","outcome":"keep_both","savedAs":"hdr (2)","sameName":`+held+`},`+
			`{"name":"Atmos","outcome":"add","savedAs":"Atmos"}],"impact":`+noImpact+`}`)
	s.expect(t, http.MethodPost, "/api/v1/custom-formats/paste?preview=true", pasteBody(text, `,"replace":["`+id+`"]`), http.StatusOK,
		`{"app":"radarr","formats":[{"name":"hdr","outcome":"replace","savedAs":"hdr","sameName":`+held+`},`+
			`{"name":"Atmos","outcome":"add","savedAs":"Atmos"}],"impact":`+noImpact+`}`)
	s.expect(t, http.MethodGet, "/api/v1/custom-formats", "", http.StatusOK, `{"formats":[`+hdr+`],"matchLimitMilliseconds":100}`)

	s.expect(t, http.MethodPost, "/api/v1/custom-formats/paste", pasteBody(text, `,"replace":["`+id+`"]`), http.StatusOK,
		`{"app":"radarr","formats":[{"name":"hdr","outcome":"replace","savedAs":"hdr","sameName":`+held+`},`+
			`{"name":"Atmos","outcome":"add","savedAs":"Atmos"}],"impact":`+noImpact+`}`)
	atmos := `{"id":"ID","name":"Atmos","includeWhenRenaming":false,"dialect":"radarr","conditions":[` +
		`{"name":"Atmos","kind":"ReleaseTitleSpecification","negate":false,"required":true,"fields":{"value":"\\bATMOS\\b"}}],"scoredBy":[],"slowReleases":0}`
	remux := func(name string) string {
		return `{"id":"ID","name":"` + name + `","includeWhenRenaming":false,"dialect":"radarr","conditions":[` +
			`{"name":"hdr","kind":"QualityModifierSpecification","negate":false,"required":true,"fields":{"value":5}}],"scoredBy":[],"slowReleases":0}`
	}
	s.expect(t, http.MethodGet, "/api/v1/custom-formats/"+id, "", http.StatusOK, remux("hdr"))

	s.expect(t, http.MethodPost, "/api/v1/custom-formats/paste", pasteBody(text, `,"app":"radarr"`), http.StatusOK,
		`{"app":"radarr","formats":[{"name":"hdr","outcome":"keep_both","savedAs":"hdr (2)","sameName":{"id":"ID","name":"hdr","dialect":"radarr"}},`+
			`{"name":"Atmos","outcome":"keep_both","savedAs":"Atmos (2)","sameName":{"id":"ID","name":"Atmos","dialect":"radarr"}}],"impact":`+noImpact+`}`)
	s.expect(t, http.MethodGet, "/api/v1/custom-formats", "", http.StatusOK,
		`{"formats":[`+atmos+`,`+strings.Replace(atmos, `"Atmos","include`, `"Atmos (2)","include`, 1)+`,`+remux("hdr")+`,`+remux("hdr (2)")+`],"matchLimitMilliseconds":100}`)

	s.expect(t, http.MethodPost, "/api/v1/custom-formats/paste", pasteBody(text, `,"replace":["`+s.starter+`"]`), http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","detail":"No pasted Custom Format has this one's name. Leave it out and paste again.",`+
			`"errors":[{"location":"body.replace[0]","message":"No pasted Custom Format has this one's name. Leave it out and paste again."}]}`)
}

func TestAPasteAsksForTheAppUnlessItsKindsSettleIt(t *testing.T) {
	s := serve(t)
	needed := `{"status":422,"code":"format_app_needed","title":"Say whether this was written for Radarr or Sonarr"}`
	titles := trash("Atmos", "ReleaseTitleSpecification", `{"value":"ATMOS"}`)
	both := `[` + trash("Pack", "ReleaseTypeSpecification", `{"value":3}`) + `,` + trash("Cut", "EditionSpecification", `{"value":"Cut"}`) + `]`
	for _, c := range []struct {
		name, body, app string
	}{
		{"a title alone settles nothing", pasteBody(titles, ""), ""},
		{"the answer reads a title alone", pasteBody(titles, `,"app":"sonarr"`), "sonarr"},
		{"edition means Radarr", pasteBody(trash("Cut", "EditionSpecification", `{"value":"Cut"}`), ""), "radarr"},
		{"year means Radarr", pasteBody(trash("Old", "YearSpecification", `{"min":1900,"max":1950}`), ""), "radarr"},
		{"release type means Sonarr", pasteBody(trash("Pack", "ReleaseTypeSpecification", `{"value":3}`), ""), "sonarr"},
		{"kinds of both apps settle nothing", pasteBody(both, ""), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, got := s.call(t, http.MethodPost, "/api/v1/custom-formats/paste?preview=true", c.body)
			if c.app == "" {
				if code != http.StatusUnprocessableEntity || got != needed {
					t.Fatalf("paste = %d %s, want %s", code, got, needed)
				}
				return
			}
			if code != http.StatusOK || !strings.HasPrefix(got, `{"app":"`+c.app+`",`) {
				t.Fatalf("paste = %d %s, want it read for %s", code, got, c.app)
			}
		})
	}
}

func TestAFormatThatWontRunIsRefusedWithWhatToFix(t *testing.T) {
	s := serve(t)
	format := func(conditions string) string { return `{"name":"Broken","conditions":[` + conditions + `]}` }
	title := func(pattern string) string {
		return `{"name":"Not HDR10+","kind":"ReleaseTitleSpecification","negate":true,"required":false,"fields":{"value":"` + pattern + `"}}`
	}
	refusal := func(status int, code, title, detail, location string) string {
		return `{"status":` + map[int]string{400: "400", 422: "422"}[status] + `,"code":"` + code + `","title":"` + title + `","detail":"` + detail +
			`","errors":[{"location":"body.` + location + `","message":"` + detail + `"}]}`
	}
	const (
		notCompiled = "The pattern in this condition doesn't compile"
		empty       = "A Custom Format needs at least one condition"
		unknownKind = "SlipStream doesn't have this kind of condition"
	)
	for _, c := range []struct {
		name, path, body string
		status           int
		want             string
	}{
		{
			"a pattern refused at a position", "/api/v1/custom-formats", format(`{"name":"Atmos","kind":"ReleaseGroupSpecification","negate":false,"required":false,"fields":{"value":"x"}},` + title("ab**")),
			400, refusal(400, "format_pattern_not_compiled", notCompiled,
				`The pattern in \"Not HDR10+\" doesn't compile at position 4 (a repetition follows another repetition). Fix the pattern and save again.`, "conditions[1].fields.value"),
		},
		{
			"a pattern refused with no position", "/api/v1/custom-formats", format(title("(a")),
			400, refusal(400, "format_pattern_not_compiled", notCompiled,
				`The pattern in \"Not HDR10+\" doesn't compile (unterminated group). Fix the pattern and save again.`, "conditions[0].fields.value"),
		},
		{
			"a pasted pattern refused, naming its format", "/api/v1/custom-formats/paste", pasteBody(trash("Atmos", "ReleaseTitleSpecification", `{"value":"(a"}`), `,"app":"radarr"`),
			400, refusal(400, "format_pattern_not_compiled", notCompiled,
				`The pattern in \"Atmos\" of the Custom Format \"Atmos\" doesn't compile (unterminated group). Fix the pattern and paste again.`, "text"),
		},
		{
			"a format with no conditions", "/api/v1/custom-formats", format(""),
			422, refusal(422, "format_empty", empty, `The Custom Format \"Broken\" has no conditions. Add a condition and save again.`, "conditions"),
		},
		{
			"a pasted format with no conditions", "/api/v1/custom-formats/paste", pasteBody(`{"name":"Bare","specifications":[]}`, `,"app":"sonarr"`),
			422, refusal(422, "format_empty", empty, `The Custom Format \"Bare\" has no conditions. Add a condition to it or leave it out, and paste again.`, "text"),
		},
		{
			"a kind SlipStream doesn't have", "/api/v1/custom-formats", format(title("x") + `,{"name":"Mine","kind":"MineSpecification","negate":false,"required":false,"fields":{}}`),
			422, refusal(422, "format_kind_unknown", unknownKind, `The condition \"Mine\" has the kind MineSpecification. Pick one of the kinds the schema lists.`, "conditions[1].kind"),
		},
		{
			"a pasted kind SlipStream doesn't have", "/api/v1/custom-formats/paste", pasteBody(trash("Mine", "MineSpecification", `{}`), `,"app":"sonarr"`),
			422, refusal(422, "format_kind_unknown", unknownKind, `The condition \"Mine\" has the kind MineSpecification. Remove the condition and paste again.`, "text"),
		},
		{
			"a value the app doesn't have", "/api/v1/custom-formats/paste", pasteBody(trash("Odd", "SourceSpecification", `{"value":99}`), `,"app":"sonarr"`),
			422, refusal(422, "invalid", "Some fields have values SlipStream can't accept",
				`The condition \"Odd\" has the value 99, which isn't a Sonarr source value. Change it and paste again.`, "text"),
		},
		{
			"a field the kind doesn't have", "/api/v1/custom-formats", format(title("x") + `,{"name":"Big","kind":"SizeSpecification","negate":false,"required":false,"fields":{"min":1,"value":2}}`),
			422, refusal(422, "invalid", "Some fields have values SlipStream can't accept",
				`The condition \"Big\" has the field value, which size conditions don't have. Remove the field and save again.`, "conditions"),
		},
		{
			"text that isn't JSON", "/api/v1/custom-formats/paste", pasteBody(`{"name":`, `,"app":"sonarr"`),
			422, refusal(422, "invalid", "Some fields have values SlipStream can't accept",
				`The text isn't Custom Format JSON. Copy the whole format from TRaSH Guides, Radarr or Sonarr and paste again.`, "text"),
		},
	} {
		t.Run(c.name, func(t *testing.T) { s.expect(t, http.MethodPost, c.path, c.body, c.status, c.want) })
	}
	s.expect(t, http.MethodGet, "/api/v1/custom-formats", "", http.StatusOK, `{"formats":[],"matchLimitMilliseconds":100}`)
}

func TestTheSchemaListsEachAppsKindsAndTheirValuesInWords(t *testing.T) {
	s := serve(t)
	code, body := s.Call(t, http.MethodGet, "/api/v1/custom-formats/schema", "")
	var got struct {
		Dialects []struct {
			Dialect string `json:"dialect"`
			Kinds   []struct {
				Kind   string            `json:"kind"`
				Words  string            `json:"words"`
				Values []json.RawMessage `json:"values"`
			} `json:"kinds"`
		} `json:"dialects"`
	}
	if code != http.StatusOK || json.Unmarshal([]byte(body), &got) != nil {
		t.Fatalf("customFormatSchema = %d %s", code, body)
	}
	kinds := map[string][]string{}
	values := map[string][]string{}
	for _, d := range got.Dialects {
		for _, k := range d.Kinds {
			kinds[d.Dialect] = append(kinds[d.Dialect], k.Words)
			for _, v := range k.Values {
				values[d.Dialect+" "+k.Kind] = append(values[d.Dialect+" "+k.Kind], string(v))
			}
		}
	}
	for d, want := range map[string]string{
		"slipstream": "Release title, Edition, Language, Indexer flag, Source, Resolution, Quality modifier, Size, Release group, Release type, Year",
		"radarr":     "Release title, Edition, Language, Indexer flag, Source, Resolution, Quality modifier, Size, Release group, Year",
		"sonarr":     "Release title, Language, Indexer flag, Source, Resolution, Size, Release group, Release type",
	} {
		if have := strings.Join(kinds[d], ", "); have != want {
			t.Errorf("%s kinds = %s, want %s", d, have, want)
		}
	}
	for key, want := range map[string]string{
		"radarr SourceSpecification":       `{"value":9,"words":"BLURAY"}`,
		"sonarr SourceSpecification":       `{"value":7,"words":"BlurayRaw"}`,
		"slipstream SourceSpecification":   `{"value":"webdl","words":"WEB-DL"}`,
		"sonarr ReleaseTypeSpecification":  `{"value":3,"words":"SeasonPack"}`,
		"slipstream LanguageSpecification": `{"value":"original","words":"Original"}`,
		"radarr IndexerFlagSpecification":  `{"value":1,"words":"G_Freeleech"}`,
	} {
		if !slices.Contains(values[key], want) {
			t.Errorf("%s values = %v, want one %s", key, values[key], want)
		}
	}
	if v := values["radarr ReleaseTitleSpecification"]; len(v) != 0 {
		t.Errorf("a pattern kind lists values %v", v)
	}
}
