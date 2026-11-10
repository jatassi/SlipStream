package setting_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/setting"
)

type refusal string

func (r refusal) Error() string { return string(r) }

func noColon(text string) error {
	if strings.Contains(text, ":") {
		return refusal("The text has a colon. Remove it, because a file name can't hold one.")
	}
	return nil
}

var (
	fileTemplate = setting.Template{
		Key: "show.naming.file.standard", Label: "Standard file", Default: "{Title} - {Target}",
		Tokens: "file.standard", Check: noColon,
	}
	colonText = setting.Template{
		Key: "show.naming.colonCustom", Label: "Custom colon", Default: " - ",
		Tokens: "colonCustom", Check: noColon, Plain: true,
	}
)

func TestTemplateParseJSON(t *testing.T) {
	for _, tc := range []struct {
		name    string
		t       setting.Template
		raw     string
		want    string
		refusal string
	}{
		{"null resets", fileTemplate, `null`, "reset", ""},
		{"text the check accepts", fileTemplate, `"{Title} ({Year}) - {Target}"`, "{Title} ({Year}) - {Target}", ""},
		{"text the check refuses", fileTemplate, `"{Title}: {Target}"`, "", "The text has a colon. Remove it, because a file name can't hold one."},
		{"a number", fileTemplate, `7`, "", "Text, as a string."},
		{"plain text", colonText, `" ~ "`, " ~ ", ""},
		{"plain text with a brace", colonText, `"{Title}"`, "", "This text can't hold a token. Remove each { and }."},
		{"plain text the check refuses", colonText, `" : "`, "", "The text has a colon. Remove it, because a file name can't hold one."},
		{"null resets plain text", colonText, `null`, "reset", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := stored(tc.t.ParseJSON)(json.RawMessage(tc.raw))
			if tc.refusal != "" {
				if s := sentence(err); s != tc.refusal {
					t.Fatalf("refusal = %q, want %q", s, tc.refusal)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("ParseJSON(%s) = %q, %v, want %q", tc.raw, got, err, tc.want)
			}
		})
	}
}

func TestTemplateParseRefusalNamesItsKey(t *testing.T) {
	_, err := fileTemplate.Parse(ptr("{Title}: {Target}"))
	r, ok := errors.AsType[*setting.RangeError](err)
	if !ok || r.Key != fileTemplate.Key {
		t.Fatalf("Parse error = %v, want a RangeError for %s", err, fileTemplate.Key)
	}
}

func TestTemplateField(t *testing.T) {
	for _, tc := range []struct {
		name  string
		field setting.TemplateField
		want  string
	}{
		{
			"a template at its default", fileTemplate.Field(nil),
			`{"key":"show.naming.file.standard","label":"Standard file","source":"database","applies":"live","isDefault":true,"pendingRestart":false,"kind":"template","value":"{Title} - {Target}","default":"{Title} - {Target}","field":"file.standard","plain":false}`,
		},
		{
			"plain text, overridden", colonText.Field(ptr(" ~ ")),
			`{"key":"show.naming.colonCustom","label":"Custom colon","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"template","value":" ~ ","default":" - ","field":"colonCustom","plain":true}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, err := json.Marshal(tc.field)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.want {
				t.Fatalf("Field =\n%s\nwant\n%s", b, tc.want)
			}
		})
	}
}
