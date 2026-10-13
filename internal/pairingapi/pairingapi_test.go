package pairingapi_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/pairingapi"
	"github.com/jatassi/slipstream/internal/system"
)

// golden is the SLP1 golden code, which parses.
const golden = "SLP1-CC6WT-AB6RG-SPD48-JAH4B-QCJV9-XJ6AS-K7D1M-PMTVC-DNQ6Y-W3HE9-SG213-Z0000-58D1S-QG852-G"

// unclaimed is a Node with no library.db: Unclaimed, with no identity
// service.
type unclaimed struct{}

func (unclaimed) Claim(context.Context) (system.Claim, error) { return system.Unclaimed, nil }

func (unclaimed) Identity() (*identity.Service, bool) { return nil, false }

// refusing is a Node whose every Join is refused as kind.
type refusing struct{ kind pairing.Kind }

func (r refusing) Join(context.Context, joincode.Code) (pairing.Joined, error) {
	return pairing.Joined{}, &pairing.Error{Kind: r.kind, Reason: "refused"}
}

func TestClaimWorkerProblemCodes(t *testing.T) {
	for _, tc := range []struct {
		code string
		kind pairing.Kind
		want string
	}{
		{golden[:len(golden)-1] + "H", "", problem(422, "join_code_damaged", "The Join Code is mistyped, cut short or not a SlipStream code")},
		{golden, pairing.CodeUnknown, problem(403, "join_code_unknown", "The Controller does not know this Join Code")},
		{golden, pairing.CodeExpired, problem(403, "join_code_expired", "The Join Code has expired")},
		{golden, pairing.CodeUsed, problem(403, "join_code_used", "The Join Code was already used")},
		{golden, pairing.Unreachable, problem(502, "controller_unreachable", "The Controller did not answer")},
		{golden, pairing.KeyMismatch, problem(502, "controller_key_mismatch", "The Controller's key does not match the Join Code")},
		{golden, pairing.IsController, problem(409, "already_claimed", "This Node is already set up")},
		{golden, pairing.RestartRequired, problem(409, "already_claimed", "This Node is already set up")},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			log := slog.New(slog.DiscardHandler)
			gate, setupCode, err := identity.NewGate(t.Context(), unclaimed{}.Claim, log)
			if err != nil {
				t.Fatal(err)
			}
			r := api.NewRouter(api.Services{Node: unclaimed{}, Gate: gate, Web: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, Log: log})
			pairingapi.Routes(r, refusing{tc.kind})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/setup/worker", strings.NewReader(`{"code":"`+tc.code+`"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Setup-Code", setupCode)
			w := httptest.NewRecorder()
			r.Handler().ServeHTTP(w, req)
			if got := w.Body.String(); got != tc.want {
				t.Fatalf("claimWorker = %d %s, want %s", w.Code, got, tc.want)
			}
		})
	}
}

func problem(status int, code, title string) string {
	return fmt.Sprintf(`{"status":%d,"code":%q,"title":%q}`, status, code, title) + "\n"
}
