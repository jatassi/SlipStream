package naming_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/naming"
)

func TestEveryTrashTemplateParses(t *testing.T) {
	files, err := filepath.Glob("testdata/trash/*/naming/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no TRaSH naming files: %v", err)
	}
	n := 0
	for _, file := range files {
		var tree any
		if err := json.Unmarshal(read(t, file), &tree); err != nil {
			t.Fatal(file, err)
		}
		walkStrings(tree, func(template string) {
			n++
			got, err := naming.Parse(template)
			if err != nil {
				t.Errorf("%s: Parse(%q): %v", file, template, err)
			} else if got.String() != template {
				t.Errorf("%s: Parse(%q).String() = %q", file, template, got.String())
			}
		})
	}
	if n < 30 {
		t.Errorf("parsed %d TRaSH templates; the snapshot holds more", n)
	}
}

// refusedArr are the upstream templates outside SlipStream's grammar:
// Sonarr's "\" between folders and Radarr's tag wrapped in braces.
var refusedArr = map[string]bool{
	`{Series TitleFirstCharacter}\{Series Title}`: true,
	`{Series Year}\{Series Title}`:                true,
	"{Movie Title} {{imdb-{ImdbId}}}":             true,
	"{Movie Title} {{imdbid-{ImdbId}}}":           true,
	"{Movie Title} {{tmdb-{TmdbId}}}":             true,
	"{Movie Title} {{edition-{Edition Tags}}}":    true,
}

func TestArrTemplatesParseOrAreRefused(t *testing.T) {
	files, err := filepath.Glob("testdata/arr/*/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no arr case files: %v", err)
	}
	seen := map[string]bool{}
	for _, file := range files {
		var f struct {
			Cases []struct{ Template, Source string }
		}
		if err := json.Unmarshal(read(t, file), &f); err != nil {
			t.Fatal(file, err)
		}
		for _, c := range f.Cases {
			_, err := naming.Parse(c.Template)
			var pe *naming.ParseError
			switch {
			case refusedArr[c.Template]:
				seen[c.Template] = true
				if !errors.As(err, &pe) {
					t.Errorf("%s: Parse(%q) = %v; want a *ParseError", c.Source, c.Template, err)
				}
			case err != nil:
				t.Errorf("%s: Parse(%q): %v", c.Source, c.Template, err)
			}
		}
	}
	for template := range refusedArr {
		if !seen[template] {
			t.Errorf("no arr case has %q any more", template)
		}
	}
}

func read(t *testing.T, file string) []byte {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func walkStrings(v any, f func(string)) {
	switch v := v.(type) {
	case string:
		f(v)
	case map[string]any:
		for _, e := range v {
			walkStrings(e, f)
		}
	case []any:
		for _, e := range v {
			walkStrings(e, f)
		}
	}
}
