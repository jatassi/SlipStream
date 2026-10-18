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

// OneOf holds one of V's variants, given to NewOneOf or decoded from a
// request. The document names its schema after V: a oneOf over V's
// variants, discriminated on V's property. Its JSON is the variant it
// holds.
type OneOf[V Variants] struct {
	value any
}

func NewOneOf[V Variants](variant any) OneOf[V] { return OneOf[V]{value: variant} }

// Value is the variant o holds, a value of one of V's variant types.
func (o OneOf[V]) Value() any { return o.value }

// UnmarshalJSON decodes the variant b's discriminator names. Huma has
// already checked b against the union's schema.
func (o *OneOf[V]) UnmarshalJSON(b []byte) error {
	var v V
	property, variants := v.Variants()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(b, &fields); err != nil {
		return fmt.Errorf("api: decode a %s: %w", reflect.TypeFor[V](), err)
	}
	var kind string
	if err := json.Unmarshal(fields[property], &kind); err != nil {
		return fmt.Errorf("api: decode a %s's %s: %w", reflect.TypeFor[V](), property, err)
	}
	for _, variant := range variants {
		t := reflect.TypeOf(variant)
		if discriminatorValue(t, property) != kind {
			continue
		}
		decoded := reflect.New(t)
		if err := json.Unmarshal(b, decoded.Interface()); err != nil {
			return fmt.Errorf("api: decode a %s: %w", t, err)
		}
		o.value = decoded.Elem().Interface()
		return nil
	}
	return fmt.Errorf("api: no %s variant has %s %q", reflect.TypeFor[V](), property, kind)
}

func (o OneOf[V]) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(o.value)
	if err != nil {
		return nil, fmt.Errorf("api: encode %T: %w", o.value, err)
	}
	return b, nil
}

func (OneOf[V]) Schema(r huma.Registry) *huma.Schema {
	name := schemaName(reflect.TypeFor[V](), "")
	if _, done := r.Map()[name]; !done {
		var v V
		property, variants := v.Variants()
		union := &huma.Schema{Discriminator: &huma.Discriminator{PropertyName: property, Mapping: map[string]string{}}}
		for _, variant := range variants {
			t := reflect.TypeOf(variant)
			mustDocumentEmbedded(t, map[reflect.Type]bool{})
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
