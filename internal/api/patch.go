package api

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// Patch is one member of a PATCH body with three states: absent leaves
// the value alone, null resets it to its default, and a value sets it.
// Tag its field omitempty so it may be absent.
type Patch[T any] struct {
	sent  bool
	value *T
}

func (p Patch[T]) Sent() bool { return p.sent }

// Value is the value sent, nil for null.
func (p Patch[T]) Value() *T { return p.value }

func (p *Patch[T]) UnmarshalJSON(b []byte) error {
	p.sent = true
	if string(b) == "null" {
		p.value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("api: decode %T: %w", v, err)
	}
	p.value = &v
	return nil
}

func (Patch[T]) Schema(r huma.Registry) *huma.Schema { return nullableSchema[T](r) }

// Nullable is a response member that is a value or null. Huma can't mark a
// pointer to a struct nullable, so its schema is T's or null.
type Nullable[T any] struct{ value *T }

// Some is a Nullable holding v.
func Some[T any](v T) Nullable[T] { return Nullable[T]{value: &v} }

func (n Nullable[T]) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(n.value)
	if err != nil {
		return nil, fmt.Errorf("api: encode %T: %w", n.value, err)
	}
	return b, nil
}

func (Nullable[T]) Schema(r huma.Registry) *huma.Schema { return nullableSchema[T](r) }

func nullableSchema[T any](r huma.Registry) *huma.Schema {
	s := r.Schema(reflect.TypeFor[T](), true, "")
	if s.Ref != "" {
		return &huma.Schema{AnyOf: []*huma.Schema{s, {Type: "null"}}}
	}
	nullable := *s
	nullable.Nullable = true
	return &nullable
}
