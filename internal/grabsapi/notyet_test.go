package grabsapi_test

import (
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// notYet is every operation whose handler isn't built. The PR that builds
// one deletes its line.
var notYet = []struct{ op, method, path, body string }{
	{"listWanted", http.MethodGet, "/api/v1/wanted", ""},
}

func TestNotYetLedger(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	s := ownertest.Serve(t, lib, func(r *api.Router) { grabsapi.Routes(r, pipeline{}) })
	for _, c := range notYet {
		code, body := s.Call(t, c.method, c.path, c.body)
		if want := `{"status":501,"code":"not_yet","title":"SlipStream doesn't do this yet"}` + "\n"; code != http.StatusNotImplemented || body != want {
			t.Errorf("%s = %d %s, want 501 %s", c.op, code, body, want)
		}
	}
}
