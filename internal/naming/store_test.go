package naming_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

const shelf works.TypeID = "shelf"

var boxes = naming.Declaration{
	Tokens: []naming.Token{
		{Name: "Shelf Title", Kind: naming.KindTitle, Scope: naming.WorkScope, Example: "Attic"},
		{Name: "box", Kind: naming.KindNumber, Scope: naming.LevelScope, Seq: naming.SeqOuter, Example: "1"},
		{Name: "thing", Kind: naming.KindNumber, Scope: naming.ItemScope, Seq: naming.SeqInner, Example: "1"},
		{Name: "Thing Title", Kind: naming.KindTitle, Scope: naming.ItemScope, Example: "Scarf"},
	},
	Folder: "{Shelf Title}",
	Levels: []string{"Box {box:00}"},
	Aside:  "Spares",
	Files: []naming.Variant{
		{Key: "standard", Label: "Thing file", File: "{Shelf Title} - B{box:00}T{thing:00} - {Thing Title} {Edition} {Target}", Anchor: naming.BlockAnchor()},
	},
	Samples: []naming.Sample{
		{Label: "A scarf", Work: naming.Set{"Shelf Title": naming.Text("Attic")}, Units: []naming.Unit{thing(1, "Scarf")}},
		{Label: "Two things", Work: naming.Set{"Shelf Title": naming.Text("Attic")}, Units: []naming.Unit{thing(1, "Scarf"), thing(2, "Hat")}},
		{Label: "Nothing known", Missing: true, Work: naming.Set{"Shelf Title": naming.Text("Attic")}, Units: []naming.Unit{{
			Level: naming.Set{"box": naming.Number(1)}, Item: naming.ItemValues{Set: naming.Set{"thing": naming.Number(1)}},
		}}},
	},
}

// thing is Thing n of box 1, titled title.
func thing(n int64, title string) naming.Unit {
	return naming.Unit{
		Level: naming.Set{"box": naming.Number(1)},
		Item:  naming.ItemValues{Set: naming.Set{"thing": naming.Number(n), "Thing Title": naming.Text(title)}},
	}
}

type facet struct{}

func (facet) Naming() naming.Declaration { return boxes }

func (facet) NamingValues(context.Context, storage.Reader[storage.Library], works.Among[works.WorkID]) (map[works.WorkID]naming.Values, error) {
	return nil, nil
}

type published struct{ events []naming.Changed }

func (p *published) Publish(_ context.Context, c naming.Changed) { p.events = append(p.events, c) }

type store struct {
	lib    *storage.DB[storage.Library]
	naming *naming.Naming
	sent   *published
}

func open(t *testing.T, budget int) store {
	t.Helper()
	lib := storagetest.Library(t, naming.Schema())
	sent := &published{}
	n := naming.New(naming.Config{
		Library: lib, Facets: map[works.TypeID]naming.Facet{shelf: facet{}},
		Describe: func(works.TypeID) (works.Description, bool) {
			return works.Description{
				Work: works.Noun{One: "Shelf", Many: "Shelves"}, Levels: []works.Noun{{One: "Box", Many: "Boxes"}},
				Aside: works.Noun{One: "Spare", Many: "Spares"}, Item: works.Noun{One: "Thing", Many: "Things"},
			}, true
		},
		Budget:  func(context.Context, storage.Reader[storage.Library], works.TypeID) (int, error) { return budget, nil },
		Changed: sent,
	})
	return store{lib: lib, naming: n, sent: sent}
}

func patch(values map[string]string) setting.Patch {
	p := setting.Patch{}
	for k, v := range values {
		p[k] = json.RawMessage(v)
	}
	return p
}

// name is the path s's stored scheme gives the first Thing of box 1 of
// Attic, a Standard file.
func (s store) name(t *testing.T) string {
	t.Helper()
	sc, err := s.naming.SchemeIn(t.Context(), s.lib, shelf)
	if err != nil {
		t.Fatal(err)
	}
	return named(t, sc)
}

func named(t *testing.T, sc *naming.Scheme) string {
	t.Helper()
	n, err := sc.Name(naming.Placing{
		Subject:      naming.Subject{Work: naming.Set{"Shelf Title": naming.Text("Attic")}, Units: []naming.Unit{thing(1, "Scarf")}},
		LevelFolders: true, Landing: naming.Rendered(), File: naming.FileFacts{Target: "1080p", Keep: "attic.scarf"}, Ext: ".mkv", Budget: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	return n.Rel
}

func refusedWith(t *testing.T, err error, want ...naming.Problem) {
	t.Helper()
	refused, ok := errors.AsType[*naming.RefusedError](err)
	if !ok {
		t.Fatalf("err = %v, want a *RefusedError", err)
	}
	if !slices.Equal(refused.Problems, want) {
		t.Errorf("problems = %q, want %q", refused.Problems, want)
	}
}

const (
	defaultName = "Attic/Box 01/Attic - B01T01 - Scarf 1080p.mkv"
	noTarget    = "The file name has no {Target}. Add {Target} so each file of a Work gets its own name."
)

func TestAStoredTemplateNamesTheNextFileAndAChangeIsPublished(t *testing.T) {
	s := open(t, 1000)
	if got := s.name(t); got != defaultName {
		t.Fatalf("with nothing stored, name = %q, want %q", got, defaultName)
	}
	err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.file.standard": `"{Shelf Title} - B{box:00}T{thing:00} - {Edition} {Target} {Thing Title}"`, "shelf.naming.levels.1": `"Box {box}"`}))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s.name(t), "Attic/Box 1/Attic - B01T01 - 1080p Scarf.mkv"; got != want {
		t.Errorf("name = %q, want %q", got, want)
	}
	if want := []naming.Changed{{ContentType: "shelf"}}; !slices.Equal(s.sent.events, want) {
		t.Errorf("published %v, want %v", s.sent.events, want)
	}
	if err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.file.standard": `null`, "shelf.naming.levels.1": `null`})); err != nil {
		t.Fatal(err)
	}
	if got := s.name(t); got != defaultName {
		t.Errorf("after null resets, name = %q, want %q", got, defaultName)
	}
}

func TestARefusedPatchStoresNothingAndNamesEachKey(t *testing.T) {
	s := open(t, 1000)
	err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.levels.1": `"Box {box:00}"`, "shelf.naming.file.standard": `"{Shelf Title} - B{box:00}T{thing:00} - {Edition}"`}))
	refusedWith(t, err, naming.Problem{Key: "shelf.naming.file.standard", Sentence: noTarget})
	err = s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.folder": `"{Shelf Title"`, "shelf.naming.colon": `"sideways"`}))
	refusedWith(t, err,
		naming.Problem{Key: "shelf.naming.folder", Sentence: `{Shelf Title has no closing brace. Add "}" to close it.`},
		naming.Problem{Key: "shelf.naming.colon", Sentence: "Choose Delete or Dash (-) or Space dash ( -) or Space dash space ( - ) or Smart or Custom."},
	)
	if got := s.name(t); got != defaultName {
		t.Errorf("name = %q, want the default %q", got, defaultName)
	}
	if len(s.sent.events) != 0 {
		t.Errorf("published %v for refused patches", s.sent.events)
	}
}

func TestTurningRenamingOnChecksTheStoredTemplates(t *testing.T) {
	s := open(t, 1000)
	err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.renameFiles": `false`, "shelf.naming.file.standard": `"{Shelf Title} - B{box:00}T{thing:00} - {Edition}"`}))
	if err != nil {
		t.Fatalf("with renaming off, a file template without {Target} is refused: %v", err)
	}
	if got, want := s.name(t), "Attic/Box 01/attic.scarf.mkv"; got != want {
		t.Errorf("with renaming off, name = %q, want %q", got, want)
	}
	err = s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.renameFiles": `true`}))
	refusedWith(t, err, naming.Problem{Key: "shelf.naming.file.standard", Sentence: noTarget})
}

func TestAKeyTheTypeHasNoSettingForIsUnknown(t *testing.T) {
	s := open(t, 1000)
	err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.levels.2": `"Shelf {box}"`, "shelf.naming.levels.1": `"Box {box}"`}))
	if unknown, ok := errors.AsType[*setting.UnknownKeyError](err); !ok || unknown.Key != "shelf.naming.levels.2" {
		t.Fatalf("err = %v, want shelf.naming.levels.2 unknown", err)
	}
	if got := s.name(t); got != defaultName {
		t.Errorf("name = %q, want the default %q", got, defaultName)
	}
}

func TestTheSaveCheckNamesSamplesWithinTheSmallestRootBudget(t *testing.T) {
	s := open(t, 30)
	err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.colon": `"dash"`}))
	refusedWith(t, err, naming.Problem{
		Key:      "shelf.naming.file.standard",
		Sentence: `For the sample "A scarf" with a Standard file, the path in the root is 39 bytes long and the limit there is 30 bytes. Shorten the template.`,
	})
}

func TestADraftNamesWithoutStoring(t *testing.T) {
	s := open(t, 1000)
	sc, err := s.naming.DraftIn(t.Context(), s.lib, shelf, patch(map[string]string{"shelf.naming.folder": `"Shelf {Shelf Title}"`}))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := named(t, sc), "Shelf Attic/Box 01/Attic - B01T01 - Scarf 1080p.mkv"; got != want {
		t.Errorf("draft name = %q, want %q", got, want)
	}
	if got := s.name(t); got != defaultName {
		t.Errorf("stored name = %q, want the default %q", got, defaultName)
	}
	_, err = s.naming.DraftIn(t.Context(), s.lib, shelf, patch(map[string]string{"shelf.naming.file.standard": `"{Shelf Title} - B{box:00}T{thing:00} - {Edition}"`}))
	refusedWith(t, err, naming.Problem{Key: "shelf.naming.file.standard", Sentence: noTarget})
}

func TestTheFieldsListEveryKeyInCardOrderWithItsValue(t *testing.T) {
	s := open(t, 1000)
	if err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.multiEpisode": `"range"`})); err != nil {
		t.Fatal(err)
	}
	fields, err := s.naming.Panel(shelf).Fields(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	type row struct{ key, label, value string }
	var got []row
	for _, f := range fields {
		b, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		var w struct {
			Key, Label string
			Value      json.RawMessage
		}
		if err := json.Unmarshal(b, &w); err != nil {
			t.Fatal(err)
		}
		got = append(got, row{w.Key, w.Label, string(w.Value)})
	}
	want := []row{
		{"shelf.naming.renameFiles", "Rename files", `true`},
		{"shelf.naming.folder", "Shelf folder", `"{Shelf Title}"`},
		{"shelf.naming.levels.1", "Box folder", `"Box {box:00}"`},
		{"shelf.naming.aside", "Spares folder", `"Spares"`},
		{"shelf.naming.file.standard", "Thing file", `"{Shelf Title} - B{box:00}T{thing:00} - {Thing Title} {Edition} {Target}"`},
		{"shelf.naming.colon", "Colon replacement", `"smart"`},
		{"shelf.naming.colonCustom", "Custom colon replacement", `""`},
		{"shelf.naming.multiEpisode", "Multi-thing style", `"range"`},
	}
	if !slices.Equal(got, want) {
		t.Errorf("fields =\n%v\nwant\n%v", got, want)
	}
}

func TestTheCustomColonTextNamesFilesAndIsRefusedWhenANameCantHoldIt(t *testing.T) {
	s := open(t, 1000)
	err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.colon": `"custom"`, "shelf.naming.colonCustom": `" ?"`}))
	refusedWith(t, err, naming.Problem{Key: "shelf.naming.colonCustom", Sentence: `The colon text can't contain '?' because a name on Windows can't hold it. Remove it.`})
	if err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.colon": `"custom"`, "shelf.naming.colonCustom": `" ~"`})); err != nil {
		t.Fatal(err)
	}
	sc, err := s.naming.SchemeIn(t.Context(), s.lib, shelf)
	if err != nil {
		t.Fatal(err)
	}
	n, err := sc.Name(naming.Placing{
		Subject:      naming.Subject{Work: naming.Set{"Shelf Title": naming.Text("Attic: Loft")}, Units: []naming.Unit{thing(1, "Scarf")}},
		LevelFolders: true, Landing: naming.Rendered(), File: naming.FileFacts{Target: "1080p"}, Ext: ".mkv", Budget: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "Attic ~ Loft/Box 01/Attic ~ Loft - B01T01 - Scarf 1080p.mkv"; n.Rel != want {
		t.Errorf("name = %q, want %q", n.Rel, want)
	}
}
