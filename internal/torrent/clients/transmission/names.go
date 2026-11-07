package transmission

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
)

// The legacy protocol spells methods and their arguments in kebab-case
// and a torrent's fields in camelCase. JSON-RPC 2.0 spells them all in
// snake_case, the spelling this package writes.

func legacy(method string) string { return strings.ReplaceAll(method, "_", "-") }

// field is a torrent field's name in the session's protocol.
func (c *client) field(snake string) string {
	if c.json2 {
		return snake
	}
	words := strings.Split(snake, "_")
	for i := 1; i < len(words); i++ {
		words[i] = strings.ToUpper(words[i][:1]) + words[i][1:]
	}
	return strings.Join(words, "")
}

func kebabKeys(args map[string]any) map[string]any {
	out := make(map[string]any, len(args))
	for k, v := range args {
		out[legacy(k)] = v
	}
	return out
}

// snakeKeys respells every object key in raw, kebab-case or camelCase,
// in snake_case.
func snakeKeys(raw json.RawMessage) (json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err //nolint:wrapcheck // the caller names the method
	}
	return json.Marshal(snakeValue(v)) //nolint:wrapcheck // the caller names the method
}

func snakeValue(v any) any {
	switch v := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for k, x := range v {
			out[snake(k)] = snakeValue(x)
		}
		return out
	case []any:
		for i, x := range v {
			v[i] = snakeValue(x)
		}
	}
	return v
}

func snake(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r == '-':
			b.WriteByte('_')
		case unicode.IsUpper(r):
			b.WriteByte('_')
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
