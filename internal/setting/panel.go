package setting

import (
	"context"
	"encoding/json"
	"strings"
)

// Patch is new values by setting key; JSON null resets one to its default.
type Patch map[string]json.RawMessage

// Panel is settings one owner stores: read as fields, changed by key, all
// or none, refusing a value with *RangeError and a key it has no setting
// for with *UnknownKeyError.
type Panel interface {
	Fields(ctx context.Context) ([]Field, error)
	Apply(ctx context.Context, p Patch) error
}

type UnknownKeyError struct{ Key string }

func (e *UnknownKeyError) Error() string { return e.Key + ": no setting has this key" }

func isNull(raw json.RawMessage) bool { return strings.TrimSpace(string(raw)) == "null" }

// parseJSON reads raw as the kind's wire value and hands it to parse, or
// refuses it with wrong when it is another JSON type. Null resets.
func parseJSON[W, T any](raw json.RawMessage, key, wrong string, parse func(*W) (Stored[T], error)) (Stored[T], error) {
	if isNull(raw) {
		return parse(nil)
	}
	var v W
	if err := json.Unmarshal(raw, &v); err != nil {
		return Stored[T]{}, refuse(key, "%s", wrong)
	}
	return parse(&v)
}

func (d Duration) ParseJSON(raw json.RawMessage) (Stored[int64], error) {
	return parseJSON(raw, d.Key, "Seconds, as a whole number.", d.Parse)
}

func (c Count) ParseJSON(raw json.RawMessage) (Stored[int64], error) {
	return parseJSON(raw, c.Key, "A whole number.", c.Parse)
}

func (c Choice[T]) ParseJSON(raw json.RawMessage) (Stored[T], error) {
	return parseJSON(raw, c.Key, "One of the options, as a word.", c.Parse)
}

func (s Switch) ParseJSON(raw json.RawMessage) (Stored[bool], error) {
	return parseJSON(raw, s.Key, "On or off: true or false.", func(b *bool) (Stored[bool], error) {
		if b == nil {
			return Stored[bool]{}, nil
		}
		return override(*b), nil
	})
}
