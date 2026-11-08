package release

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Fact is a value that may be unknown. Its zero value is unknown. An
// unknown fact passes every hard filter, so a title that says nothing
// never blocks a grab.
type Fact[T any] struct {
	value T
	known bool
}

func Known[T any](v T) Fact[T] { return Fact[T]{value: v, known: true} }

func (f Fact[T]) Get() (v T, known bool) { return f.value, f.known }

// MarshalJSON is f's value, or null when unknown, so a Read stores as JSON.
func (f Fact[T]) MarshalJSON() ([]byte, error) {
	if !f.known {
		return []byte("null"), nil
	}
	b, err := json.Marshal(f.value)
	if err != nil {
		return nil, fmt.Errorf("marshal a fact: %w", err)
	}
	return b, nil
}

func (f *Fact[T]) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		*f = Fact[T]{}
		return nil
	}
	var v T
	if err := json.Unmarshal(b, &v); err != nil {
		return fmt.Errorf("unmarshal a fact: %w", err)
	}
	*f = Known(v)
	return nil
}
