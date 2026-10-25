package servingapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/servingapi"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type node struct {
	facts    servingapi.Facts
	isWorker bool
}

func (n node) Worker() (servingapi.Facts, bool) { return n.facts, n.isWorker }

func serve(t *testing.T, n node) *ownertest.Owner {
	t.Helper()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	return ownertest.Serve(t, lib, func(r *api.Router) { servingapi.Routes(r, n) })
}

func TestAWorkerSaysWhatItIsAndCountsWhatItHolds(t *testing.T) {
	facts := servingapi.Facts{Name: "seedbox", Fingerprint: "7Q2M-0K4D-91XB-3FHA", Controller: "home", Holding: []role.Name{role.Searching, role.Downloading}, Seeding: 3}
	serve(t, node{facts, true}).Expect(t, http.MethodGet, "/api/v1/serving", "", http.StatusOK,
		`{"name":"seedbox","fingerprint":"7Q2M-0K4D-91XB-3FHA","controller":"home","holding":["Searching","Downloading"],"seeding":3}`)
	facts.Since = time.Date(2026, 10, 17, 21, 31, 0, 0, time.UTC)
	facts.Holding = nil
	status, got := serve(t, node{facts, true}).Call(t, http.MethodGet, "/api/v1/serving", "")
	if want := `{"name":"seedbox","fingerprint":"7Q2M-0K4D-91XB-3FHA","controller":"home","since":"2026-10-17T21:31:00Z","holding":[],"seeding":3}` + "\n"; status != http.StatusOK || got != want {
		t.Fatalf("getServing with the Controller unreachable = %d %s, want 200 %s", status, got, want)
	}
}

func TestOnlyAWorkerAnswers(t *testing.T) {
	serve(t, node{}).Expect(t, http.MethodGet, "/api/v1/serving", "", http.StatusConflict,
		`{"status":409,"code":"not_a_worker","title":"This Node isn't a Worker"}`)
}
