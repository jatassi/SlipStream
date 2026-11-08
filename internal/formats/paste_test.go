package formats_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/formats"
)

func trashDoc(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "trash", path))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(b)
}

func list(docs ...[]byte) []byte {
	return append(append([]byte("[\n"), bytes.Join(docs, []byte(",\n"))...), "\n]"...)
}

type spec struct {
	Name               string `json:"name"`
	Implementation     string `json:"implementation"`
	ImplementationName string `json:"implementationName,omitempty"`
	Negate             bool   `json:"negate"`
	Required           bool   `json:"required"`
	Fields             any    `json:"fields"`
}

// rewrite is a document with its conditions edited and every other key
// kept.
func rewrite(t *testing.T, doc []byte, edit func([]spec) []spec) []byte {
	t.Helper()
	var f map[string]json.RawMessage
	if err := json.Unmarshal(doc, &f); err != nil {
		t.Fatal(err)
	}
	var specs []spec
	if err := json.Unmarshal(f["specifications"], &specs); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(edit(specs))
	if err != nil {
		t.Fatal(err)
	}
	f["specifications"] = b
	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func editFirst(t *testing.T, doc []byte, edit func(*spec)) []byte {
	t.Helper()
	return rewrite(t, doc, func(specs []spec) []spec {
		edit(&specs[0])
		return specs
	})
}

// apiForm is a document as Sonarr's /api/v3/customformat returns it, each
// condition's fields an array of described fields.
func apiForm(t *testing.T, doc []byte) []byte {
	t.Helper()
	return rewrite(t, doc, func(specs []spec) []spec {
		for i, s := range specs {
			byName, ok := s.Fields.(map[string]any)
			if !ok {
				t.Fatalf("condition %q has fields %v", s.Name, s.Fields)
			}
			var fields []any
			for name, value := range byName {
				fields = append(fields, map[string]any{
					"order": len(fields), "name": name, "label": name, "value": value, "type": "textbox", "advanced": false,
				})
			}
			specs[i].Fields = fields
			specs[i].ImplementationName = s.Implementation
		}
		return specs
	})
}

type pastedCondition struct {
	Name     string
	Kind     formats.Kind
	Negate   bool
	Required bool
	Fields   any
}

func conditionsOf(t *testing.T, f formats.Format) []pastedCondition {
	t.Helper()
	out := make([]pastedCondition, len(f.Conditions))
	for i, c := range f.Conditions {
		out[i] = pastedCondition{Name: c.Name, Kind: c.Kind, Negate: c.Negate, Required: c.Required}
		if err := json.Unmarshal(c.Fields, &out[i].Fields); err != nil {
			t.Fatalf("condition %q has fields %s: %v", c.Name, c.Fields, err)
		}
	}
	return out
}

var monoConditions = []pastedCondition{
	{"Mono", formats.ReleaseTitle, false, true, map[string]any{"value": `[^0-9]1[ .]0\b|\bMono\b|\[PCM \]`}},
	{"Not 3.0ch", formats.ReleaseTitle, true, true, map[string]any{"value": `[^0-9]3[ .]0\b`}},
	{"Not 4.0ch", formats.ReleaseTitle, true, true, map[string]any{"value": `[^0-9]4[ .]0\b`}},
	{"Not High Channel Count", formats.ReleaseTitle, true, true, map[string]any{"value": `[^0-9][5-9][ .][0-1]\b`}},
}

var packConditions = []pastedCondition{
	{"Season Packs", formats.ReleaseType, false, false, map[string]any{"value": float64(3)}},
}

func TestPaste(t *testing.T) {
	mono := trashDoc(t, "radarr/cf/10-mono.json")
	pack := trashDoc(t, "sonarr/cf/season-pack.json")
	monoAPI, packAPI := apiForm(t, mono), apiForm(t, pack)
	type want struct {
		name       string
		source     []byte
		conditions []pastedCondition
	}
	cases := []struct {
		name string
		text []byte
		want []want
	}{
		{"one TRaSH document", mono, []want{{"1.0 Mono", mono, monoConditions}}},
		{"one TRaSH document in a list", list(mono), []want{{"1.0 Mono", mono, monoConditions}}},
		{"a list of TRaSH documents", list(mono, pack), []want{
			{"1.0 Mono", mono, monoConditions},
			{"Season Pack", pack, packConditions},
		}},
		{"Sonarr's API array form", monoAPI, []want{{"1.0 Mono", monoAPI, monoConditions}}},
		{"a list in Sonarr's API array form", list(monoAPI, packAPI), []want{
			{"1.0 Mono", monoAPI, monoConditions},
			{"Season Pack", packAPI, packConditions},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs, err := formats.Paste(append([]byte("\n  "), tc.text...))
			if err != nil {
				t.Fatal(err)
			}
			if len(fs) != len(tc.want) {
				t.Fatalf("pasted %d formats, want %d", len(fs), len(tc.want))
			}
			for i, w := range tc.want {
				checkPasted(t, fs[i], w.name, w.source, w.conditions)
			}
		})
	}
}

func checkPasted(t *testing.T, f formats.Format, name string, source []byte, conditions []pastedCondition) {
	t.Helper()
	if f.Name != name {
		t.Errorf("pasted %q, want %q", f.Name, name)
	}
	if !bytes.Equal(f.Source, source) {
		t.Errorf("%q kept the source\n%s\nwant\n%s", f.Name, f.Source, source)
	}
	if got := conditionsOf(t, f); !reflect.DeepEqual(got, conditions) {
		t.Errorf("%q has conditions\n%+v\nwant\n%+v", f.Name, got, conditions)
	}
}

func TestPasteKeepsTheDocumentVerbatim(t *testing.T) {
	doc := trashDoc(t, "sonarr/cf/season-pack.json")
	fs, err := formats.Paste(list(doc))
	if err != nil {
		t.Fatal(err)
	}
	var source map[string]any
	if err := json.Unmarshal(fs[0].Source, &source); err != nil {
		t.Fatal(err)
	}
	if source["trash_id"] != "3bc5f395426614e155e585a2f056cdf1" {
		t.Errorf("trash_id = %v", source["trash_id"])
	}
	if !reflect.DeepEqual(source["trash_scores"], map[string]any{"default": float64(10)}) {
		t.Errorf("trash_scores = %v", source["trash_scores"])
	}
}

func TestPasteRefuses(t *testing.T) {
	mono := trashDoc(t, "radarr/cf/10-mono.json")
	pack := trashDoc(t, "sonarr/cf/season-pack.json")
	mediaInfo := editFirst(t, mono, func(s *spec) { s.Implementation = "MediaInfoSpecification" })
	regexField := editFirst(t, mono, func(s *spec) { s.Fields = map[string]any{"regex": `\bMono\b`} })
	noConditions := rewrite(t, mono, func([]spec) []spec { return []spec{} })
	cases := []struct {
		name string
		text []byte
		want error
		says string
	}{
		{
			"an unknown kind", mediaInfo,
			&formats.KindError{Name: "Mono", Kind: "MediaInfoSpecification"},
			`read the Custom Format "1.0 Mono": condition "Mono" has the kind MediaInfoSpecification, which isn't a Custom Format condition kind`,
		},
		{
			"an unknown kind in the second format of a list", list(pack, mediaInfo),
			&formats.KindError{Name: "Mono", Kind: "MediaInfoSpecification"}, "",
		},
		{
			"an unknown field", regexField,
			&formats.FieldError{Name: "Mono", Kind: formats.ReleaseTitle, Field: "regex"},
			`read the Custom Format "1.0 Mono": condition "Mono" has the field regex, which ReleaseTitleSpecification conditions don't have`,
		},
		{
			"an unknown field in Sonarr's API array form", apiForm(t, regexField),
			&formats.FieldError{Name: "Mono", Kind: formats.ReleaseTitle, Field: "regex"}, "",
		},
		{
			"a format with no conditions", noConditions,
			&formats.EmptyError{Name: "1.0 Mono"},
			`the Custom Format "1.0 Mono" has no conditions`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fs, err := formats.Paste(tc.text)
			if err == nil {
				t.Fatalf("pasted %d formats", len(fs))
			}
			got := reflect.New(reflect.TypeOf(tc.want))
			if !errors.As(err, got.Interface()) {
				t.Fatalf("Paste = %v, want a %T", err, tc.want)
			}
			if !reflect.DeepEqual(got.Elem().Interface(), tc.want) {
				t.Errorf("Paste = %#v, want %#v", got.Elem().Interface(), tc.want)
			}
			if tc.says != "" && err.Error() != tc.says {
				t.Errorf("Paste says %q, want %q", err, tc.says)
			}
		})
	}
}

func TestPasteRefusesTextThatIsNotJSON(t *testing.T) {
	for _, text := range []string{"1.0 Mono", "[1.0 Mono]", `{"name": 7}`} {
		if fs, err := formats.Paste([]byte(text)); err == nil {
			t.Errorf("Paste(%q) = %d formats", text, len(fs))
		}
	}
}

func TestAppOf(t *testing.T) {
	paste := func(t *testing.T, docs ...[]byte) []formats.Format {
		t.Helper()
		fs, err := formats.Paste(list(docs...))
		if err != nil {
			t.Fatal(err)
		}
		return fs
	}
	mono := trashDoc(t, "radarr/cf/10-mono.json")
	remux := trashDoc(t, "radarr/cf/french-remux-tier-01.json")
	pack := trashDoc(t, "sonarr/cf/season-pack.json")
	imax := trashDoc(t, "radarr/cf/imax.json")
	asEdition := editFirst(t, imax, func(s *spec) { s.Implementation = "EditionSpecification" })
	asYear := editFirst(t, mono, func(s *spec) {
		s.Implementation = "YearSpecification"
		s.Fields = map[string]any{"min": 1990, "max": 1999}
	})
	cases := []struct {
		name    string
		docs    [][]byte
		app     formats.Dialect
		settled bool
	}{
		{"a Quality modifier condition", [][]byte{remux}, formats.Radarr, true},
		{"an Edition condition", [][]byte{asEdition}, formats.Radarr, true},
		{"a Year condition", [][]byte{asYear}, formats.Radarr, true},
		{"a Release type condition", [][]byte{pack}, formats.Sonarr, true},
		{"kinds both apps have", [][]byte{mono}, "", false},
		{"one Radarr format among plain ones", [][]byte{mono, remux}, formats.Radarr, true},
		{"one Sonarr format among plain ones", [][]byte{mono, pack}, formats.Sonarr, true},
		{"a Radarr format and a Sonarr format", [][]byte{remux, pack}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, settled := formats.AppOf(paste(t, tc.docs...))
			if app != tc.app || settled != tc.settled {
				t.Errorf("AppOf = %q, %v, want %q, %v", app, settled, tc.app, tc.settled)
			}
		})
	}
}
