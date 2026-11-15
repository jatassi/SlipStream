package grabsapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/grabsapi"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

func TestSearchingSeveralWorksTakesIdsOrAFilterAndAtMost5000Works(t *testing.T) {
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	s := ownertest.Serve(t, lib, func(r *api.Router) { grabsapi.Routes(r, pipeline{}) })
	ids := make([]string, 5001)
	for i := range ids {
		ids[i] = `"` + storage.NewUUID().String() + `"`
	}
	for _, c := range []struct{ name, body, code string }{
		{"both", `{"workIds":[],"wanted":{}}`, `"code":"invalid"`},
		{"neither", `{}`, `"code":"invalid"`},
		{"too many", `{"workIds":[` + strings.Join(ids, ",") + `]}`, `"code":"too_many_works"`},
	} {
		if code, body := s.Call(t, http.MethodPost, "/api/v1/works/search", c.body); code != http.StatusUnprocessableEntity || !strings.Contains(body, c.code) {
			t.Errorf("%s: %d %s, want 422 with %s", c.name, code, body, c.code)
		}
	}
}
