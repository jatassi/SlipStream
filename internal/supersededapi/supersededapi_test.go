package supersededapi_test

import (
	"context"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/supersededapi"
)

// fence is a Controller fenced by a proof vps-relay carried, recording
// each Wipe.
type fence struct {
	f     nodes.Fence
	wiped []joincode.Code
}

func (f *fence) Fenced(context.Context) (nodes.Fence, bool, error) { return f.f, true, nil }
func (*fence) Name(nodes.ID) string                                { return "vps-relay" }
func (*fence) Stop(context.Context) error                          { return nodes.ErrNotTold }
func (*fence) Resume(context.Context) error                        { return nil }

func (f *fence) Wipe(_ context.Context, code joincode.Code) error {
	f.wiped = append(f.wiped, code)
	return nil
}

type node struct{ f *fence }

func (n node) Fence() (supersededapi.Fence, bool) { return n.f, true }

func serve(t *testing.T) (*ownertest.Owner, *fence) {
	t.Helper()
	addrs, err := link.ParseAddresses([]netip.AddrPort{netip.MustParseAddrPort("192.168.1.20:8485")})
	if err != nil {
		t.Fatal(err)
	}
	f := &fence{f: nodes.Fence{
		State: nodes.Proven, By: link.NewPrivateKey([32]byte{42}).Public(), Addresses: addrs,
		ByNode: role.NodeID{UUID: storage.NewUUID()}, At: time.Date(2026, 11, 14, 10, 12, 0, 0, time.UTC),
	}}
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	return ownertest.Serve(t, lib, func(r *api.Router) { supersededapi.Routes(r, node{f}) }), f
}

// TestTheSupersededPageReadsWhoReplacedThisController reads the fence as
// the page shows it: proven, when, the new Controller's host and the
// Worker that carried the proof.
func TestTheSupersededPageReadsWhoReplacedThisController(t *testing.T) {
	o, f := serve(t)
	want := `{"state":"proven","at":"2026-11-14T10:12:00Z","replacement":{"key":"` + f.f.By.Fingerprint() + `","address":"192.168.1.20"},"worker":"vps-relay"}` + "\n"
	if status, got := o.CallSignedIn(t, http.MethodGet, "/api/v1/superseded", ""); status != http.StatusOK || got != want {
		t.Fatalf("getSuperseded = %d %s, want 200 %s", status, got, want)
	}
}

// TestWipeRefusesADamagedJoinCodeBeforeAnythingChanges sends a mistyped
// Join Code, then a good one: the first answers 422 join_code_damaged and
// never reaches Wipe, the second wipes with it. The API key can't wipe.
func TestWipeRefusesADamagedJoinCodeBeforeAnythingChanges(t *testing.T) {
	o, f := serve(t)
	code, err := joincode.New(make([]byte, 32), [joincode.SecretSize]byte{7}, []netip.AddrPort{netip.MustParseAddrPort("192.168.1.20:8485")})
	if err != nil {
		t.Fatal(err)
	}
	damaged := code.String()[:len(code.String())-3] + "ZZZ"
	status, got := o.CallSignedIn(t, http.MethodPost, "/api/v1/superseded/wipe", `{"joinCode":"`+damaged+`"}`)
	if status != http.StatusUnprocessableEntity || len(f.wiped) != 0 {
		t.Fatalf("a damaged code = %d %s with %d wipes, want 422 and none", status, got, len(f.wiped))
	}
	if status, _ := o.Call(t, http.MethodPost, "/api/v1/superseded/wipe", `{"joinCode":"`+code.String()+`"}`); status != http.StatusUnauthorized && status != http.StatusForbidden {
		t.Fatalf("the API key's wipe = %d, want it refused", status)
	}
	if status, got := o.CallSignedIn(t, http.MethodPost, "/api/v1/superseded/wipe", `{"joinCode":"`+code.String()+`"}`); status != http.StatusAccepted || len(f.wiped) != 1 || f.wiped[0].String() != code.String() {
		t.Fatalf("a good code = %d %s, wipes %v; want 202 with the code", status, got, f.wiped)
	}
}

// TestStopWithNoClaimIsRefused stops a Controller no Worker told
// anything: 409 not_told.
func TestStopWithNoClaimIsRefused(t *testing.T) {
	o, _ := serve(t)
	want := `{"status":409,"code":"not_told","title":"No Worker said a restore replaced this Controller"}` + "\n"
	if status, got := o.CallSignedIn(t, http.MethodPost, "/api/v1/superseded/stop", ""); status != http.StatusConflict || got != want {
		t.Fatalf("stopSupersededController = %d %s, want 409 %s", status, got, want)
	}
}
