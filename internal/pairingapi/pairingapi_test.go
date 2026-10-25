package pairingapi_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/pairing"
	"github.com/jatassi/slipstream/internal/pairingapi"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
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

func (r refusing) Join(context.Context, joincode.Code, func(pairing.Pending)) (pairing.Joined, error) {
	return pairing.Joined{}, &pairing.Error{Kind: r.kind, Reason: "refused"}
}

func (refusing) Pending() (pairing.Pending, bool) { return pairing.Pending{}, false }

func TestClaimWorkerProblemCodes(t *testing.T) {
	for _, tc := range []struct {
		code string
		kind pairing.Kind
		want string
	}{
		{golden[:len(golden)-1] + "H", "", problem(422, "join_code_damaged", "The Join Code is mistyped, cut short or not a SlipStream code")},
		{golden, pairing.CodeUnknown, problem(403, "join_code_unknown", "The Controller has no such Join Code")},
		{golden, pairing.CodeExpired, problem(403, "join_code_expired", "The Join Code has expired")},
		{golden, pairing.CodeUsed, problem(403, "join_code_used", "The Join Code was already used")},
		{golden, pairing.Unaddressed, problem(409, "node_unaddressed", "The Controller is unreachable, and this Node's link has no address for it to dial")},
		{golden, pairing.KeyMismatch, problem(502, "controller_key_mismatch", "The Controller's key doesn't match the Join Code")},
		{golden, pairing.IsController, problem(409, "already_claimed", "This Node is already set up")},
		{golden, pairing.RestartRequired, problem(409, "restart_required", "A running Worker re-pairs only at start")},
		{golden, pairing.JoinPending, problem(409, "join_pending", "A Join waits for its Controller to dial this Node")},
	} {
		t.Run(string(tc.kind), func(t *testing.T) {
			call := setup(t, refusing{tc.kind})
			if code, got := call(http.MethodPost, `{"code":"`+tc.code+`"}`); got != tc.want {
				t.Fatalf("claimWorker = %d %s, want %s", code, got, tc.want)
			}
		})
	}
}

// setup serves pairingapi on an Unclaimed Node over j, and calls
// /setup/worker with its Setup code.
func setup(t *testing.T, j pairingapi.Joiner) func(method, body string) (int, string) {
	t.Helper()
	log := slog.New(slog.DiscardHandler)
	gate, setupCode, err := identity.NewGate(t.Context(), unclaimed{}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: unclaimed{}, Gate: gate, Web: fstest.MapFS{"index.html": {Data: []byte("<!doctype html>")}}, Log: log})
	pairingapi.Routes(r, j)
	h := r.Handler()
	return func(method, body string) (int, string) {
		req := httptest.NewRequestWithContext(t.Context(), method, "/api/v1/setup/worker", strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("X-Setup-Code", setupCode)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}
}

// waiting is a Node whose Join awaits its Controller's dial until joined
// is closed, then joins as joinedAs.
type waiting struct {
	mu       *sync.Mutex
	pend     *pairing.Pending
	joined   chan struct{}
	joinedAs pairing.Joined
}

func (w waiting) Join(_ context.Context, _ joincode.Code, awaiting func(pairing.Pending)) (pairing.Joined, error) {
	p, _ := w.Pending()
	awaiting(p)
	<-w.joined
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pend.State, w.pend.Joined = pairing.JoinedState, &w.joinedAs
	return w.joinedAs, nil
}

func (w waiting) Pending() (pairing.Pending, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return *w.pend, true
}

func TestAJoinAwaitingTheDialAnswers202(t *testing.T) {
	addr, err := joincode.ParseNodeAddress("SLN1-40GJ4-8S44M-K2EA1-958NJ-RB9E5-WR32C-HK6GT-KCDSR-74X3P-F1X7R-ZG216-B01RG-J895T-WRNRQ-0")
	if err != nil {
		t.Fatal(err)
	}
	until := time.Date(2026, 10, 17, 22, 31, 0, 0, time.UTC)
	id, err := storage.ParseUUID("0192f3a0-0000-7000-8000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	w := waiting{
		mu:       &sync.Mutex{},
		pend:     &pairing.Pending{State: pairing.AwaitingDial, Address: addr, Until: until},
		joined:   make(chan struct{}),
		joinedAs: pairing.Joined{NodeID: role.NodeID{UUID: id}, ControllerName: "home", Link: netip.MustParseAddrPort("203.0.113.1:50000")},
	}
	call := setup(t, w)
	awaitingDial := `{"state":"awaiting-dial","nodeAddress":"` + addr.String() + `","until":"2026-10-17T22:31:00Z"}` + "\n"
	if code, got := call(http.MethodPost, `{"code":"`+golden+`"}`); code != http.StatusAccepted || got != awaitingDial {
		t.Fatalf("claimWorker = %d %s, want 202 %s", code, got, awaitingDial)
	}
	if code, got := call(http.MethodGet, ""); code != http.StatusOK || got != awaitingDial {
		t.Fatalf("getWorkerClaim = %d %s, want 200 %s", code, got, awaitingDial)
	}
	close(w.joined)
	want := `{"state":"joined","nodeId":"0192f3a0-0000-7000-8000-000000000001","controller":{"name":"home","key":"0000-0000-0000-0000","link":"203.0.113.1:50000"},"nodeAddress":"` + addr.String() + `","until":"2026-10-17T22:31:00Z"}` + "\n"
	for {
		code, got := call(http.MethodGet, "")
		if code != http.StatusOK {
			t.Fatalf("getWorkerClaim = %d %s", code, got)
		}
		if got == want {
			return
		}
		if got != awaitingDial {
			t.Fatalf("getWorkerClaim = %s, want %s", got, want)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestGetWorkerClaimWithNoPendingJoin(t *testing.T) {
	if code, got := setup(t, refusing{})(http.MethodGet, ""); got != problem(404, "no_join_pending", "No Join has awaited its Controller's dial") {
		t.Fatalf("getWorkerClaim = %d %s", code, got)
	}
}

func problem(status int, code, title string) string {
	return fmt.Sprintf(`{"status":%d,"code":%q,"title":%q}`, status, code, title) + "\n"
}
