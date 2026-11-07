package metadataapi_test

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadataapi"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// door logs in key "mine" with PIN "1234" and answers /test to its token.
func door(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/login" {
		if r.Header.Get("Authorization") != "Bearer opened" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{}`))
		return
	}
	var in struct{ APIKey, Pin string }
	_ = json.NewDecoder(r.Body).Decode(&in)
	switch {
	case in.APIKey != "mine":
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unknown apikey"}`))
	case in.Pin != "1234":
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"wrong pin"}`))
	default:
		_, _ = w.Write([]byte(`{"token":"opened"}`))
	}
}

type gate struct {
	line *metadata.Line
	p    metadata.Provider
	base *url.URL
}

func newGate(t *testing.T) gate {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(door))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	p := metadata.Provider{
		ID: "gate", Label: "Gate", Rate: 20, Burst: 20, KeyPage: "https://gate.invalid/account",
		Auth: &metadata.Auth{
			Pin: true,
			Login: func(key, pin string) (*url.URL, []byte) {
				body, _ := json.Marshal(map[string]string{"apiKey": key, "pin": pin}) //nolint:errchkjson // strings always encode
				return base.JoinPath("login"), body
			},
			Token: func(body []byte) (string, time.Time, error) {
				var t struct{ Token string }
				err := json.Unmarshal(body, &t)
				return t.Token, time.Now().Add(time.Hour), err
			},
			Refused: func(_ int, body []byte) metadata.Credential {
				if strings.Contains(string(body), "pin") {
					return metadata.CredentialPin
				}
				return metadata.CredentialKey
			},
		},
	}
	lines := metadata.NewLines(srv.Client(), "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	return gate{line: lines.Line(p), p: p, base: base}
}

func (g gate) Provider() metadata.Provider { return g.p }

func (g gate) Test(ctx context.Context, ask metadata.Ask) error {
	return g.line.Get(ctx, ask, metadata.Request{URL: g.base.JoinPath("test")}, &struct{}{})
}

func TestPinRefusedAnswersItsCode(t *testing.T) {
	g := newGate(t)
	holder := metadata.NewHolder(metadata.HolderConfig{Testers: []metadata.Tester{g}})
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema(), metadata.Schema())
	m, err := metadata.Open(t.Context(), metadata.Config{
		Library:    lib,
		Local:      storagetest.Local(t, artwork.Schema()),
		ArtworkDir: t.TempDir(),
		Minter:     minter,
		Sealer:     secrettest.Sealer(t),
		Roster:     alone{},
		Service:    func(role.NodeID) metadatav1connect.MetadataServiceHandler { return holder },
		Providers:  []metadata.Provider{g.p},
		Changed:    &changes{},
		Timings:    metadata.DefaultTimings(),
		Rand:       rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw the test replays
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := ownertest.Serve(t, lib, func(r *api.Router) { metadataapi.Routes(r, registry{m}) })

	owner.Expect(t, http.MethodPut, "/api/v1/metadata/providers/gate/key", `{"key":"mine","pin":"0000"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"metadata_pin_refused","title":"The metadata provider refused that PIN, so SlipStream didn't save it","detail":"Gate refused the PIN. Check the PIN on your Gate account page."}`)
	owner.Expect(t, http.MethodPut, "/api/v1/metadata/providers/gate/key", `{"key":"typo","pin":"1234"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"metadata_key_refused","title":"The metadata provider refused that key, so SlipStream didn't save it","detail":"Gate refused the key: 401 Unauthorized"}`)

	code, reply := owner.Call(t, http.MethodPut, "/api/v1/metadata/providers/gate/key", `{"key":"mine","pin":"1234"}`)
	if code != http.StatusOK || !strings.Contains(reply, `"key":"yours","state":"working"`) || !strings.Contains(reply, `"takesPin":true,"pinSet":true`) {
		t.Fatalf("setMetadataKey with the right PIN = %d %s, want your key working with its PIN", code, reply)
	}
	code, reply = owner.Call(t, http.MethodDelete, "/api/v1/metadata/providers/gate/key", "")
	if code != http.StatusOK || !strings.Contains(reply, `"key":"none"`) || !strings.Contains(reply, `"takesPin":true,"pinSet":false`) {
		t.Fatalf("clearMetadataKey = %d %s, want no key and no PIN", code, reply)
	}
}

func TestAPinForAProviderThatTakesNoneIsAFieldError(t *testing.T) {
	s := serve(t)
	s.owner.Expect(t, http.MethodPut, "/api/v1/metadata/providers/tmdb/key", `{"key":"mine","pin":"1234"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"invalid","title":"Some fields have values SlipStream can't accept","errors":[{"location":"body.pin","message":"TMDB keys don't use a PIN. Leave the PIN out."}]}`)
	s.owner.Expect(t, http.MethodGet, "/api/v1/metadata/providers", "", http.StatusOK, `{"providers":[`+unusedTMDB+`,`+unusedOMDb+`]}`)
}
