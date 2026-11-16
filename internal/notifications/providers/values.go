package providers

import (
	"encoding/json"
	"net/url"

	"github.com/jatassi/slipstream/internal/node/endpoint"
)

// Header is one row of a Headers field.
type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// value is key's raw value from the settings or the opened secrets. A key
// is in one or the other, never both.
func (p Payload) value(key string) json.RawMessage {
	if v, ok := p.Secrets[key]; ok {
		return v
	}
	return p.Settings[key]
}

func (p Payload) text(key string) string { return decode[string](p.value(key)) }

func (p Payload) list(key string) []string { return decode[[]string](p.value(key)) }

func (p Payload) number(key string) int { return decode[int](p.value(key)) }

func (p Payload) on(key string) bool { return decode[bool](p.value(key)) }

func (p Payload) headers(key string) []Header { return decode[[]Header](p.value(key)) }

// decode is raw as T, and T's zero value when raw is absent. The API
// checked every value against its Field before it was stored.
func decode[T any](raw json.RawMessage) T {
	var v T
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &v)
	}
	return v
}

// enteredHost is the host of the URL the Admin entered as key. values are
// the settings with the opened secrets, since a webhook URL is secret.
func enteredHost(key string) func(values Values, _ Endpoints) (*url.URL, bool) {
	return func(values Values, _ Endpoints) (*url.URL, bool) {
		u, err := url.Parse(decode[string](values[key]))
		if err != nil || u.Host == "" {
			return nil, false
		}
		return &url.URL{Scheme: u.Scheme, Host: u.Host}, true
	}
}

func fixedHost(pick func(Endpoints) endpoint.URL) func(Values, Endpoints) (*url.URL, bool) {
	return func(_ Values, e Endpoints) (*url.URL, bool) {
		u := pick(e).JoinPath()
		return &url.URL{Scheme: u.Scheme, Host: u.Host}, true
	}
}

func noHost(Values, Endpoints) (*url.URL, bool) { return nil, false }
