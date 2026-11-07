package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/danielgtaylor/huma/v2"
)

// Values names a string enum and lists its values, for a list that lives
// in Go code rather than in an enum tag.
type Values interface {
	Values() []string
}

// Enum is one of V's values. The document names its schema after V, so the
// web's union of them follows V's list.
type Enum[V Values] struct {
	value string
}

// NewEnum is value as an Enum. It panics unless V lists value.
func NewEnum[V Values](value string) Enum[V] {
	var v V
	if !slices.Contains(v.Values(), value) {
		panic(fmt.Sprintf("api: %q is not a %s", value, reflect.TypeFor[V]()))
	}
	return Enum[V]{value: value}
}

func (e Enum[V]) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(e.value)
	if err != nil {
		return nil, fmt.Errorf("api: encode a %s: %w", reflect.TypeFor[V](), err)
	}
	return b, nil
}

func (Enum[V]) Schema(r huma.Registry) *huma.Schema {
	name := schemaName(reflect.TypeFor[V](), "")
	if _, done := r.Map()[name]; !done {
		var v V
		s := &huma.Schema{Type: huma.TypeString}
		for _, value := range v.Values() {
			s.Enum = append(s.Enum, value)
		}
		r.Map()[name] = s
	}
	return &huma.Schema{Ref: schemaPrefix + name}
}

// Counts is a count for each of V's values, a value with none left out.
// The document names its schema after V.
type Counts[V Values] map[string]int

func (Counts[V]) Schema(r huma.Registry) *huma.Schema {
	name := schemaName(reflect.TypeFor[V](), "") + "Counts"
	if _, done := r.Map()[name]; !done {
		var v V
		s := &huma.Schema{Type: huma.TypeObject, Properties: map[string]*huma.Schema{}, AdditionalProperties: false}
		for _, value := range v.Values() {
			s.Properties[value] = &huma.Schema{Type: huma.TypeInteger, Minimum: new(1.0)}
		}
		r.Map()[name] = s
	}
	return &huma.Schema{Ref: schemaPrefix + name}
}
