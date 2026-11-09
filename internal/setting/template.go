package setting

import (
	"encoding/json"
	"strings"
)

// Template is text that Check, which the owner supplies, accepts. A
// refusal's error text is the sentence the control shows. Tokens names the template field whose tokens the control
// offers. Plain is text that holds no tokens, such as a custom colon text.
type Template struct {
	Key, Label, Default string
	Tokens              string
	Check               func(string) error
	Plain               bool
}

func (t Template) Value(override *string) string {
	if override == nil {
		return t.Default
	}
	return *override
}

// Parse accepts text Check accepts, and with Plain no brace; nil resets.
func (t Template) Parse(text *string) (Stored[string], error) {
	if text == nil {
		return Stored[string]{}, nil
	}
	if t.Plain && strings.ContainsAny(*text, "{}") {
		return Stored[string]{}, refuse(t.Key, "This text can't hold a token. Remove each { and }.")
	}
	if err := t.Check(*text); err != nil {
		return Stored[string]{}, &RangeError{Key: t.Key, Sentence: err.Error()}
	}
	return override(*text), nil
}

func (t Template) ParseJSON(raw json.RawMessage) (Stored[string], error) {
	return parseJSON(raw, t.Key, "Text, as a string.", t.Parse)
}

// TemplateField is a Template on the wire.
type TemplateField struct {
	About
	Kind    string `json:"kind" enum:"template"`
	Value   string `json:"value"`
	Default string `json:"default"`
	Field   string `json:"field" doc:"The template field whose tokens the control offers"`
	Plain   bool   `json:"plain" doc:"The text holds no tokens"`
}

func (t Template) Field(override *string) TemplateField {
	return TemplateField{
		About:   about(t.Key, t.Label, override == nil),
		Kind:    "template",
		Value:   t.Value(override),
		Default: t.Default,
		Field:   t.Tokens,
		Plain:   t.Plain,
	}
}
