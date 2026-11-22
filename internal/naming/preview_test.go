package naming_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
)

func field(t *testing.T, p naming.Preview, key string) naming.FieldPreview {
	t.Helper()
	i := slices.IndexFunc(p.Fields, func(f naming.FieldPreview) bool { return f.Key == key })
	if i < 0 {
		t.Fatalf("the preview has no %s", key)
	}
	return p.Fields[i]
}

func pathOf(t *testing.T, f naming.FieldPreview, label string) string {
	t.Helper()
	i := slices.IndexFunc(f.Samples, func(s naming.SamplePath) bool { return s.Label == label })
	if i < 0 {
		t.Fatalf("%s has no sample %q in %v", f.Key, label, f.Samples)
	}
	return f.Samples[i].Path
}

func TestAPreviewSaysWhatTheCheckRefusesUnderEachField(t *testing.T) {
	s := open(t, 1000)
	p, err := s.naming.Preview(t.Context(), shelf, patch(map[string]string{
		"shelf.naming.folder":        `"{Shelf Title"`,
		"shelf.naming.file.standard": `"{Shelf Title} - B{box:00}T{thing:00} - {Thing Title}"`,
	}))
	if err != nil {
		t.Fatal(err)
	}
	folder := field(t, p, "shelf.naming.folder")
	if want := []string{`{Shelf Title has no closing brace. Add "}" to close it.`}; !slices.Equal(folder.Problems, want) || folder.Samples != nil {
		t.Errorf("folder = %q with %v, want %q and no paths", folder.Problems, folder.Samples, want)
	}
	file := field(t, p, "shelf.naming.file.standard")
	want := []string{noTarget, "The file name has no Edition token. Add {Edition Tag} or {Edition} so each cut gets its own name."}
	if !slices.Equal(file.Problems, want) {
		t.Errorf("file problems = %q, want %q", file.Problems, want)
	}
	if got, want := pathOf(t, file, "A scarf with a cut"), "Attic/Box 01/Attic - B01T01 - Scarf.mkv"; got != want {
		t.Errorf("file sample under the stored folder = %q, want %q", got, want)
	}
}

func TestProblemsJudgeTheStoredTemplatesWithRenamingOn(t *testing.T) {
	s := open(t, 1000)
	if got, err := s.naming.Problems(t.Context(), shelf); err != nil || got != nil {
		t.Fatalf("problems of the defaults = %v, %v, want none", got, err)
	}
	err := s.naming.Panel(shelf).Apply(t.Context(), patch(map[string]string{"shelf.naming.renameFiles": `false`, "shelf.naming.file.standard": `"{Shelf Title} - B{box:00}T{thing:00} - {Edition}"`}))
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.naming.Problems(t.Context(), shelf)
	if err != nil {
		t.Fatal(err)
	}
	if want := []naming.Problem{{Key: "shelf.naming.file.standard", Sentence: noTarget}}; !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

func TestTheCatalogListsEachTemplatesTokensWithTheirModifierAndTags(t *testing.T) {
	s := open(t, 1000)
	fields, err := s.naming.Catalog(shelf)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, f := range fields {
		keys = append(keys, f.Key)
	}
	if want := []string{"shelf.naming.folder", "shelf.naming.levels.1", "shelf.naming.aside", "shelf.naming.file.standard"}; !slices.Equal(keys, want) {
		t.Fatalf("fields = %q, want %q", keys, want)
	}
	names := func(f naming.FieldTokens) []string {
		var out []string
		for _, e := range f.Tokens {
			out = append(out, e.Name)
		}
		return out
	}
	if got := names(fields[0]); slices.Contains(got, "box") || slices.Contains(got, "Target") || !slices.Contains(got, "Shelf CleanTitle") {
		t.Errorf("folder tokens = %q, want the Work's and its title forms only", got)
	}
	if got := names(fields[1]); !slices.Contains(got, "box") || slices.Contains(got, "thing") {
		t.Errorf("level tokens = %q, want the box and not the thing", got)
	}
	type offer struct {
		modifier naming.Modifier
		tags     []string
	}
	want := map[string]offer{
		"box":                      {modifier: naming.ModPad},
		"Thing Title":              {modifier: naming.ModCut},
		"Target":                   {},
		"Edition":                  {modifier: naming.ModCut, tags: []string{"edition"}},
		"Edition Tag":              {modifier: naming.ModCut},
		"Custom Format":            {modifier: naming.ModFormat},
		"Custom Formats":           {modifier: naming.ModFormats},
		"MediaInfo AudioLanguages": {modifier: naming.ModLanguages},
	}
	for _, e := range fields[3].Tokens {
		if w, ok := want[e.Name]; ok {
			if e.Modifier != w.modifier || !slices.Equal(e.Tags, w.tags) {
				t.Errorf("%s takes %d with tags %q, want %d with %q", e.Name, e.Modifier, e.Tags, w.modifier, w.tags)
			}
			delete(want, e.Name)
		}
	}
	if len(want) != 0 {
		t.Errorf("the file tokens lack %v", want)
	}
}
