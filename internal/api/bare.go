package api

import (
	"encoding/json"
	"fmt"

	"github.com/danielgtaylor/huma/v2"
)

// Bare is a number, a string or a list of strings with no envelope, for a
// value whose reader knows its kind from elsewhere, such as a descriptor
// it already holds.
type Bare struct{ value any }

func BareNumber(n float64) Bare { return Bare{value: n} }

func BareString(s string) Bare { return Bare{value: s} }

func BareStrings(ss []string) Bare { return Bare{value: append([]string{}, ss...)} }

func (b Bare) MarshalJSON() ([]byte, error) {
	out, err := json.Marshal(b.value)
	if err != nil {
		return nil, fmt.Errorf("api: encode %T: %w", b.value, err)
	}
	return out, nil
}

func (Bare) Schema(r huma.Registry) *huma.Schema {
	const name = "Bare"
	if _, done := r.Map()[name]; !done {
		r.Map()[name] = &huma.Schema{OneOf: []*huma.Schema{
			{Type: huma.TypeNumber},
			{Type: huma.TypeString},
			{Type: huma.TypeArray, Items: &huma.Schema{Type: huma.TypeString}},
		}}
	}
	return &huma.Schema{Ref: schemaPrefix + name}
}
