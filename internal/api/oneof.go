package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// Variants names a oneOf and lists what it may hold: structs whose
// discriminator field, a string, carries a one-value enum tag.
type Variants interface {
	Variants() (discriminator string, variants []any)
}

// OneOf holds one of V's variants, given to NewOneOf. The document names
// its schema after V: a oneOf over V's variants, discriminated on V's
// property. Its JSON is the variant it holds.
type OneOf[V Variants] struct {
	value any
}

func NewOneOf[V Variants](variant any) OneOf[V] { return OneOf[V]{value: variant} }

func (o OneOf[V]) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

func (OneOf[V]) Schema(r huma.Registry) *huma.Schema {
	name := schemaName(reflect.TypeFor[V](), "")
	if _, done := r.Map()[name]; !done {
		var v V
		property, variants := v.Variants()
		union := &huma.Schema{Discriminator: &huma.Discriminator{PropertyName: property, Mapping: map[string]string{}}}
		for _, variant := range variants {
			t := reflect.TypeOf(variant)
			ref := r.Schema(t, true, "")
			union.OneOf = append(union.OneOf, ref)
			union.Discriminator.Mapping[discriminatorValue(t, property)] = ref.Ref
		}
		r.Map()[name] = union
	}
	return &huma.Schema{Ref: schemaPrefix + name}
}

func discriminatorValue(t reflect.Type, property string) string {
	for f := range t.Fields() {
		if name, _, _ := strings.Cut(f.Tag.Get("json"), ","); name != property {
			continue
		}
		if values := strings.Split(f.Tag.Get("enum"), ","); len(values) == 1 && values[0] != "" && f.Type.Kind() == reflect.String {
			return values[0]
		}
		break
	}
	panic(fmt.Sprintf("api: oneOf variant %s needs a string field %s with a one-value enum tag", t, property))
}
