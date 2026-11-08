package formats

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrNotJSON = errors.New("the text isn't Custom Format JSON")

// EmptyError is a format with no conditions. Upstream matches every name
// with one, so a save refuses it.
type EmptyError struct {
	Name string
}

func (e *EmptyError) Error() string {
	return fmt.Sprintf("the Custom Format \"%s\" has no conditions", e.Name)
}

type pasted struct {
	Name                            string            `json:"name"`
	IncludeCustomFormatWhenRenaming bool              `json:"includeCustomFormatWhenRenaming"`
	Specifications                  []pastedCondition `json:"specifications"`
}

type pastedCondition struct {
	Name           string          `json:"name"`
	Implementation string          `json:"implementation"`
	Negate         bool            `json:"negate"`
	Required       bool            `json:"required"`
	Fields         json.RawMessage `json:"fields"`
}

// apiField is one entry of Sonarr's API array form of fields, which also
// carries order, label, type and help text.
type apiField struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}

// Paste reads one format or a list, with fields as an object (TRaSH and
// export) or as Sonarr's API array. Unknown kinds and field names are a
// *KindError or *FieldError naming them, and a format with no conditions
// is an *EmptyError. Each Format's Source is its document, verbatim. The
// caller sets ID and Dialect.
func Paste(text []byte) ([]Format, error) {
	text = bytes.TrimSpace(text)
	docs := []json.RawMessage{text}
	if bytes.HasPrefix(text, []byte("[")) {
		if err := json.Unmarshal(text, &docs); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNotJSON, err)
		}
	}
	fs := make([]Format, 0, len(docs))
	for _, doc := range docs {
		f, err := pasteOne(doc)
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}
	return fs, nil
}

func pasteOne(doc json.RawMessage) (Format, error) {
	var p pasted
	if err := json.Unmarshal(doc, &p); err != nil {
		return Format{}, fmt.Errorf("%w: %w", ErrNotJSON, err)
	}
	if len(p.Specifications) == 0 {
		return Format{}, &EmptyError{Name: p.Name}
	}
	f := Format{Name: p.Name, IncludeWhenRenaming: p.IncludeCustomFormatWhenRenaming, Source: bytes.Clone(doc)}
	for _, s := range p.Specifications {
		c, err := pasteCondition(s)
		if err != nil {
			return Format{}, fmt.Errorf("read the Custom Format \"%s\": %w", p.Name, err)
		}
		f.Conditions = append(f.Conditions, c)
	}
	return f, nil
}

func pasteCondition(s pastedCondition) (Condition, error) {
	c := Condition{Name: s.Name, Kind: Kind(s.Implementation), Negate: s.Negate, Required: s.Required, Fields: s.Fields}
	k, ok := kindOf(c.Kind)
	if !ok {
		return Condition{}, &KindError{Name: s.Name, Kind: s.Implementation}
	}
	if bytes.HasPrefix(s.Fields, []byte("[")) {
		var list []apiField
		if err := json.Unmarshal(s.Fields, &list); err != nil {
			return Condition{}, &ValueError{Name: c.Name, Kind: c.Kind, Field: "fields", Value: string(s.Fields), Want: "an object or a list of fields"}
		}
		by := make(map[string]json.RawMessage, len(list))
		for _, af := range list {
			by[af.Name] = af.Value
		}
		object, err := json.Marshal(by)
		if err != nil {
			return Condition{}, fmt.Errorf("write the fields of condition \"%s\": %w", c.Name, err)
		}
		c.Fields = object
	}
	if _, err := k.fieldsOf(&c); err != nil {
		return Condition{}, err
	}
	return c, nil
}

// appOnly is each kind only one app has.
var appOnly = map[Kind]Dialect{
	Edition:         Radarr,
	QualityModifier: Radarr,
	Year:            Radarr,
	ReleaseType:     Sonarr,
}

// AppOf is the app formats were written for when their kinds settle it:
// Edition, QualityModifier or Year say Radarr, ReleaseType says Sonarr.
// Formats with kinds of both apps, or of neither, settle nothing.
func AppOf(fs []Format) (Dialect, bool) {
	var app Dialect
	for _, f := range fs {
		for _, c := range f.Conditions {
			d, ok := appOnly[c.Kind]
			if !ok {
				continue
			}
			if app != "" && app != d {
				return "", false
			}
			app = d
		}
	}
	return app, app != ""
}
