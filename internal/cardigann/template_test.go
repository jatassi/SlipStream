package cardigann

import (
	"slices"
	"strings"
	"testing"
)

func fixtureScope() scope {
	return scope{
		".Config.sitelink":  textValue("https://somesite.com/"),
		".Config.x":         textValue("1"),
		".Config.blank":     textValue("  "),
		".True":             textValue("True"),
		".False":            nullValue,
		".Today.Year":       textValue("2026"),
		".Categories":       listValue([]string{"tv", "movies"}),
		".NoCategories":     listValue(nil),
		".Query.Keywords":   textValue("abc def"),
		".Query.Injected":   textValue("{{ .True }}"),
		".Query.Brackets":   textValue("a}}b"),
		".Result.title":     textValue("Title"),
		".Result.title_alt": nullValue,
	}
}

func TestTemplateEval(t *testing.T) {
	tests := []struct {
		name, src, want string
	}{
		// ApplyGoTemplateTextFixture.cs
		{"range", "{{ range .Categories}}&categories[]={{.}}{{end}}", "&categories[]=tv&categories[]=movies"},
		{"range index", "{{ range $i, $e := .Categories}}&categories[{{$i}}]={{.}}{{end}}", "&categories[0]=tv&categories[1]=movies"},
		{"range named index", "{{ range $index, $element := .Categories}}&categories[{{$index}}]={{.}}+postIndex[{{$index}}]{{end}}", "&categories[0]=tv+postIndex[0]&categories[1]=movies+postIndex[1]"},
		{"re_replace", `{{ re_replace .Query.Keywords "[^a-zA-Z0-9]+" "%" }}`, "abc%def"},
		{"join", `{{ join .Categories ", " }}`, "tv, movies"},
		{"variable", "{{ .Today.Year }}", "2026"},
		{"if null", "{{if .False }}0{{else}}1{{end}}", "1"},
		{"if true", "{{if .True }}0{{else}}1{{end}}", "0"},

		{"null renders empty", "a{{ .False }}b", "ab"},
		{"text around", "x{{ .Config.x }}y", "x1y"},
		{"if without spaces", "{{if.True}}a{{end}}", "a"},
		{"if blank is false", "{{ if .Config.blank }}a{{ else }}b{{ end }}", "b"},
		{"if list", "{{ if .Categories }}a{{ end }}{{ if .NoCategories }}b{{ end }}", "a"},
		{"and false", "{{ if and .True .False }}y{{ else }}n{{ end }}", "n"},
		{"and true", "{{ if and .True .Config.x }}y{{ else }}n{{ end }}", "y"},
		{"or", "{{ if or .False .True }}y{{ else }}n{{ end }}", "y"},
		{"or value", "{{ or .False .Config.sitelink }}", "https://somesite.com/"},
		{"and value", "{{ and .True .Config.sitelink }}", "https://somesite.com/"},
		{"and first blank", "[{{ and .False .True }}]", "[]"},
		{"and skips literals", `{{ and "x" .Config.x }}`, "1"},
		{"eq literal", `{{ if eq .Config.x "1" }}y{{ else }}n{{ end }}`, "y"},
		{"eq renders True", `{{ eq .Config.x "1" }}`, "True"},
		{"eq differs", `[{{ eq .Config.x "2" }}]`, "[]"},
		{"ne", `{{ if ne .Config.x "2" }}y{{ else }}n{{ end }}`, "y"},
		{"eq null with null", "{{ if eq .False .Result.title_alt }}y{{ end }}", "y"},
		{"eq list reads as null", "{{ if eq .Categories .False }}y{{ end }}", "y"},
		{"nested call", `{{ if and (eq .Config.x "1") .True }}y{{ end }}`, "y"},
		{"prefix calls", `{{ if or eq .Config.x "2" .True }}y{{ end }}`, "y"},
		{"if in range", "{{ range .Categories }}{{ if .True }}<{{.}}>{{ end }}{{ end }}", "<tv><movies>"},
		{"range in if", "{{ if .True }}{{ range .Categories }}{{.}};{{ end }}{{ end }}", "tv;movies;"},
		{"multi-line", "a\n{{ if .True }}\nb\n{{ end }}", "a\n\nb\n"},
		{"substituted text isn't parsed again", "{{ .Query.Injected }}", "{{ .True }}"},
		{"re_replace pattern holding braces", `{{ re_replace .Query.Brackets "}}" "x" }}`, "axb"},
		{"re_replace group", `{{ re_replace .Query.Keywords "(\w+) (\w+)" "$2-$1" }}`, "def-abc"},
		{"no template", "plain", "plain"},
		{"lone braces", "a}}b", "a}}b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := parseTemplate(tt.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, err := tmpl.Eval(fixtureScope(), nil)
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTemplateModifier(t *testing.T) {
	tmpl, err := parseTemplate("q={{ .Query.Keywords }}&{{ range .Categories }}c={{.}}{{ end }}")
	if err != nil {
		t.Fatal(err)
	}
	got, err := tmpl.Eval(fixtureScope(), strings.ToUpper)
	if err != nil {
		t.Fatal(err)
	}
	if want := "q=ABC DEF&c=TVc=MOVIES"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateEvalErrors(t *testing.T) {
	tests := []struct{ name, src string }{
		{"missing variable", "{{ .Config.missing }}"},
		{"missing in if", "{{ if .Config.missing }}a{{ end }}"},
		{"list as text", "{{ .Categories }}"},
		{"range over text", "{{ range .Config.x }}{{.}}{{ end }}"},
		{"join over text", `{{ join .Config.x "," }}`},
		{"re_replace over a list", `{{ re_replace .Categories "a" "b" }}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := parseTemplate(tt.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got, err := tmpl.Eval(fixtureScope(), nil); err == nil {
				t.Errorf("got %q, want an error", got)
			}
		})
	}
}

func TestParseTemplateErrors(t *testing.T) {
	tests := []struct{ name, src string }{
		{"unclosed action", "{{ .True "},
		{"if without end", "{{ if .True }}a"},
		{"stray end", "a{{ end }}"},
		{"stray else", "a{{ else }}b"},
		{"elem outside range", "{{ . }}"},
		{"index outside range", "{{ $i }}"},
		{"unknown function", "{{ printf .True }}"},
		{"eq with one operand", "{{ eq .True }}"},
		{"and with only literals", `{{ and "a" "b" }}`},
		{"if on a literal", `{{ if "a" }}b{{ end }}`},
		{"range without a variable", "{{ range }}a{{ end }}"},
		{"range declaration without :=", "{{ range $i .Categories }}a{{ end }}"},
		{"unterminated string", `{{ eq .True "a }}`},
		{"empty action", "{{ }}"},
		{"bad re_replace pattern", `{{ re_replace .True "(" "" }}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseTemplate(tt.src); err == nil {
				t.Errorf("parse %q: want an error", tt.src)
			}
		})
	}
}

func TestTemplateVariants(t *testing.T) {
	tmpl, err := parseTemplate("a{{ if .X }}b{{ else }}c{{ end }}{{ .Y }}{{ range .L }}[{{.}}]{{ end }}")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := tmpl.variants("1", 64), []string{"ab1[1]", "ac1[1]"}; !slices.Equal(got, want) {
		t.Errorf("variants: got %q, want %q", got, want)
	}
	if got := tmpl.variants("1", 1); len(got) != 1 {
		t.Errorf("variants past the limit: got %d", len(got))
	}
	want := []string{".X", ".Y", ".L"}
	if got := tmpl.variables(); !slices.Equal(got, want) {
		t.Errorf("variables: got %q, want %q", got, want)
	}
}
