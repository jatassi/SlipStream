package namingapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/namingapi"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

type discard struct{}

func (discard) Publish(context.Context, naming.Changed) {}

type node struct {
	n  *naming.Naming
	f  *files.Files
	im *imports.Imports
}

func (n node) Naming() (*naming.Naming, bool)    { return n.n, true }
func (n node) Files() (*files.Files, bool)       { return n.f, n.f != nil }
func (n node) Imports() (*imports.Imports, bool) { return n.im, n.im != nil }

// serve serves the naming of the Fixture type, whose roots hold paths of
// up to 1,000 bytes.
func serve(t *testing.T) *ownertest.Owner {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), naming.Schema())
	n := naming.New(naming.Config{
		Library: lib, Facets: map[works.TypeID]naming.Facet{contenttypetest.ID: contenttypetest.New()},
		Describe: func(works.TypeID) (works.Description, bool) { return contenttypetest.Description, true },
		Budget:   func(context.Context, storage.Reader[storage.Library], works.TypeID) (int, error) { return 1000, nil },
		Changed:  discard{},
	})
	return ownertest.Serve(t, lib, func(r *api.Router) { namingapi.Routes(r, node{n: n}) })
}

func call[T any](t *testing.T, o *ownertest.Owner, method, path, body string, status int) T {
	t.Helper()
	code, reply := o.Call(t, method, path, body)
	if code != status {
		t.Fatalf("%s %s = %d %s, want %d", method, path, code, reply, status)
	}
	var out T
	if err := json.Unmarshal([]byte(reply), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

type setting struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type problem struct{ Key, Sentence string }

type typeNaming struct {
	ContentType string
	Settings    []setting
	Problems    []problem
}

// folderOf is tn's Work folder template.
func folderOf(t *testing.T, tn typeNaming) string {
	t.Helper()
	i := slices.IndexFunc(tn.Settings, func(s setting) bool { return s.Key == "fixture.naming.folder" })
	if i < 0 {
		t.Fatalf("%s has no folder setting", tn.ContentType)
	}
	return string(tn.Settings[i].Value)
}

const (
	fileKey  = "fixture.naming.file.standard"
	noTarget = "The file name has no {Target}. Add {Target} so each file of a Work gets its own name."
)

func TestASavedTemplateListsWithTheTypesOtherSettings(t *testing.T) {
	o := serve(t)
	list := call[struct{ ContentTypes []typeNaming }](t, o, http.MethodGet, "/api/v1/naming", "", http.StatusOK)
	if len(list.ContentTypes) != 1 || list.ContentTypes[0].ContentType != "fixture" || len(list.ContentTypes[0].Problems) != 0 {
		t.Fatalf("listNaming = %+v, want the Fixture with no problems", list)
	}
	if got, want := folderOf(t, list.ContentTypes[0]), `"{Shelf}"`; got != want {
		t.Errorf("folder = %s, want the default %s", got, want)
	}
	saved := call[typeNaming](t, o, http.MethodPatch, "/api/v1/naming/fixture", `{"values":{"fixture.naming.folder":"Shelf {Shelf}","fixture.naming.renameFiles":false}}`, http.StatusOK)
	if got, want := folderOf(t, saved), `"Shelf {Shelf}"`; got != want {
		t.Errorf("saved folder = %s, want %s", got, want)
	}
	call[typeNaming](t, o, http.MethodPatch, "/api/v1/naming/fixture", `{"values":{"`+fileKey+`":"{Shelf} - {Thing} - {Edition Tag}"}}`, http.StatusOK)
	list = call[struct{ ContentTypes []typeNaming }](t, o, http.MethodGet, "/api/v1/naming", "", http.StatusOK)
	if got, want := list.ContentTypes[0].Problems, []problem{{fileKey, noTarget}}; !slices.Equal(got, want) {
		t.Errorf("with renaming off and no {Target}, problems = %v, want %v", got, want)
	}
}

func TestARefusedSaveNamesEachProblemAtItsKeyAndStoresNothing(t *testing.T) {
	o := serve(t)
	got := call[struct {
		Code   string
		Errors []api.FieldError
	}](t, o, http.MethodPatch, "/api/v1/naming/fixture", `{"values":{"fixture.naming.folder":"Shelf {Shelf}","`+fileKey+`":"{Shelf} - {Thing}"}}`, http.StatusUnprocessableEntity)
	want := []api.FieldError{
		{Location: "body.values." + fileKey, Message: noTarget},
		{Location: "body.values." + fileKey, Message: "The file name has no Edition token. Add {Edition Tag} or {Edition} so each cut gets its own name."},
		{Location: "body.values." + fileKey, Message: "' - ' doesn't come right after {Thing}. Put it there, because Plex, Jellyfin and Emby show an Item's files as one item only when every name does."},
	}
	if got.Code != "naming_template_refused" || !slices.Equal(got.Errors, want) {
		t.Errorf("setNaming = %s %v, want naming_template_refused %v", got.Code, got.Errors, want)
	}
	list := call[struct{ ContentTypes []typeNaming }](t, o, http.MethodGet, "/api/v1/naming", "", http.StatusOK)
	if got, want := folderOf(t, list.ContentTypes[0]), `"{Shelf}"`; got != want {
		t.Errorf("after a refused save, folder = %s, want %s", got, want)
	}
}

func TestAnUnknownKeyOrTypeIsRefused(t *testing.T) {
	o := serve(t)
	got := call[struct{ Code string }](t, o, http.MethodPatch, "/api/v1/naming/fixture", `{"values":{"fixture.naming.levels.2":"{Box}"}}`, http.StatusUnprocessableEntity)
	if got.Code != "unknown_setting" {
		t.Errorf("setNaming of levels.2 = %s, want unknown_setting", got.Code)
	}
	got = call[struct{ Code string }](t, o, http.MethodPost, "/api/v1/naming/fixture/preview", `{"draft":{"fixture.naming.levels.2":"{Box}"}}`, http.StatusUnprocessableEntity)
	if got.Code != "unknown_setting" {
		t.Errorf("previewNaming of levels.2 = %s, want unknown_setting", got.Code)
	}
	if got := call[struct{ Code string }](t, o, http.MethodGet, "/api/v1/naming/attic/tokens", "", http.StatusNotFound); got.Code != "content_type_not_found" {
		t.Errorf("listNamingTokens of attic = %s, want content_type_not_found", got.Code)
	}
	if got := call[struct{ Code string }](t, o, http.MethodPatch, "/api/v1/naming/attic", `{"values":{}}`, http.StatusNotFound); got.Code != "content_type_not_found" {
		t.Errorf("setNaming of attic = %s, want content_type_not_found", got.Code)
	}
}

func TestTheTokensOfEachTemplateCarryWhatTheBuilderShows(t *testing.T) {
	o := serve(t)
	type token struct {
		Name, Kind, Scope, Group, Example, About string
		Aliases, Tags                            []string
		Modifier                                 *string
	}
	got := call[struct {
		Fields []struct {
			Key    string
			Tokens []token
		}
	}](t, o, http.MethodGet, "/api/v1/naming/fixture/tokens", "", http.StatusOK)
	var keys []string
	for _, f := range got.Fields {
		keys = append(keys, f.Key)
	}
	if want := []string{"fixture.naming.folder", "fixture.naming.levels.1", "fixture.naming.aside", fileKey}; !slices.Equal(keys, want) {
		t.Fatalf("fields = %q, want %q", keys, want)
	}
	file := got.Fields[3].Tokens
	i := slices.IndexFunc(file, func(tk token) bool { return tk.Name == "Edition" })
	if i < 0 {
		t.Fatalf("the file tokens lack Edition: %v", file)
	}
	edition := file[i]
	if edition.Kind != "text" || edition.Scope != "file" || edition.Group != "target" || edition.Example != "Director's Cut" ||
		edition.Modifier == nil || *edition.Modifier != "cut" || !slices.Equal(edition.Tags, []string{"edition"}) || edition.Aliases == nil {
		t.Errorf("Edition = %+v, want a text token of the file grouped under target, cut, tagged edition", edition)
	}
	if slices.ContainsFunc(got.Fields[0].Tokens, func(tk token) bool { return tk.Scope != "work" }) {
		t.Errorf("folder tokens = %v, want the Work's only", got.Fields[0].Tokens)
	}
}

func TestAPreviewNamesTheSamplesOfADraftAndStoresNothing(t *testing.T) {
	o := serve(t)
	type field struct {
		Key      string
		Samples  []struct{ Label, Path string }
		Problems []string
	}
	got := call[struct{ Fields []field }](t, o, http.MethodPost, "/api/v1/naming/fixture/preview",
		`{"draft":{"fixture.naming.folder":"Shelf {Shelf}","`+fileKey+`":"{Shelf} - {Thing} - {Edition Tag}"}}`, http.StatusOK)
	byKey := map[string]field{}
	for _, f := range got.Fields {
		byKey[f.Key] = f
	}
	folder := byKey["fixture.naming.folder"]
	if len(folder.Samples) == 0 || folder.Samples[0].Label != "A Thing" || folder.Samples[0].Path != "Shelf Attic/Box 1/Attic - Scarf.mkv" || len(folder.Problems) != 0 {
		t.Errorf("folder = %+v, want A Thing at Shelf Attic/Box 1/Attic - Scarf.mkv and no problems", folder)
	}
	aside := byKey["fixture.naming.aside"]
	if len(aside.Samples) != 1 || aside.Samples[0].Path != "Shelf Attic/Spare boxes/Attic - Odd sock.mkv" {
		t.Errorf("aside = %+v, want the spare Thing in Spare boxes", aside)
	}
	file := byKey[fileKey]
	if !slices.Equal(file.Problems, []string{noTarget}) {
		t.Errorf("file problems = %q, want %q", file.Problems, noTarget)
	}
	if colon := byKey["fixture.naming.colon"]; colon.Samples == nil || len(colon.Samples) != 0 {
		t.Errorf("colon samples = %v, want an empty list", colon.Samples)
	}
	list := call[struct{ ContentTypes []typeNaming }](t, o, http.MethodGet, "/api/v1/naming", "", http.StatusOK)
	if got, want := folderOf(t, list.ContentTypes[0]), `"{Shelf}"`; got != want {
		t.Errorf("after a preview, folder = %s, want %s", got, want)
	}
}
