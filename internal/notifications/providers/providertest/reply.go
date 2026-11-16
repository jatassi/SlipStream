package providertest

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// Now is the instant every classify case reads a reply at.
var Now = time.Date(2026, time.November, 16, 14, 2, 0, 0, time.UTC)

// Post is a reply of status with body and header given as name, value
// pairs.
func Post(status int, body string, header ...string) providers.Reply {
	h := http.Header{}
	for i := 0; i+1 < len(header); i += 2 {
		h.Add(header[i], header[i+1])
	}
	return providers.Reply{Post: &providers.Response{Status: status, Header: h, Body: []byte(body)}}
}

// ClassifyCase is one reply and the verdict it must read as.
type ClassifyCase struct {
	Name  string
	Reply providers.Reply
	Want  providers.Verdict
}

// Classify checks p reads each case's reply as its verdict.
func Classify(t *testing.T, p providers.Provider, cases []ClassifyCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			if got := p.Classify(c.Reply, Now); !reflect.DeepEqual(got, c.Want) {
				t.Errorf("Classify = %+v, want %+v", got, c.Want)
			}
		})
	}
}

// Raw is a values map of each key's value as JSON.
func Raw(kv map[string]any) providers.Values {
	out := providers.Values{}
	for k, v := range kv {
		b, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		out[k] = b
	}
	return out
}
