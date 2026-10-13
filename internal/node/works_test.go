package node

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestAControllerBuildsItsContentTypesOnce(t *testing.T) {
	fx := contenttypetest.New()
	fx.Put("w1", contenttypetest.Record{Tree: works.Fetched{Label: "The Attic", Children: []works.Entry{{Key: "b1", Label: "Winter"}}}})
	env, _ := testEnv(t)
	env.Types = contenttype.NewSet(fx.Def())
	listeners := env.Environ
	env.Environ = slices.Concat(listeners, []string{"SLIPSTREAM_ENDPOINTS_CONTENT_TYPES_FIXTURE_API=https://127.0.0.1:9/v1"})
	n := startTest(t, env)
	if opens := fx.Opens(); len(opens) != 0 {
		t.Fatalf("an Unclaimed Node built the fixture %d times", len(opens))
	}
	c := client{t: t, c: &http.Client{}, base: "http://" + n.Ready().Listeners.HTTP.String()}
	claimed := c.claim(n.Ready().SetupCode, "1234")
	var res struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal([]byte(claimed.body), &res); err != nil || claimed.status != http.StatusCreated {
		t.Fatalf("claimController = %d %s", claimed.status, claimed.body)
	}
	opens := fx.Opens()
	if len(opens) != 1 {
		t.Fatalf("the Controller built the fixture %d times, want once", len(opens))
	}
	if o := opens[0]; o.Endpoints.API.String() != "https://127.0.0.1:9/v1" || o.Deps.UserAgent != "SlipStream/test ( github.com/jatassi/SlipStream )" || o.Deps.HTTP == nil {
		t.Fatalf("opened with %+v, want the loaded api, the User-Agent and a client", o)
	}
	r := c.do(call{method: http.MethodGet, path: "/api/v1/content-types", session: sessionOf(t, claimed)})
	var list struct {
		ContentTypes []struct {
			ID      string `json:"id"`
			Enabled bool   `json:"enabled"`
		} `json:"contentTypes"`
	}
	if err := json.Unmarshal([]byte(r.body), &list); err != nil || r.status != http.StatusOK || len(list.ContentTypes) != 1 || list.ContentTypes[0].ID != "fixture" || !list.ContentTypes[0].Enabled {
		t.Fatalf("listContentTypes = %d %s, want fixture on", r.status, r.body)
	}
	if r := c.admin(res.APIKey, http.MethodPost, "/api/v1/works", `{"contentType":"fixture","key":"w1"}`); r.status != http.StatusOK {
		t.Fatalf("addWork = %d %s", r.status, r.body)
	}
	n.Stop()
	waitDone(t, n)

	env.Types, env.Environ = contenttype.NewSet(), listeners
	_, err := Start(t.Context(), env)
	var refused *storage.RefusedError
	if want := "library.db holds Works of Content Type fixture, which this binary does not register"; !errors.As(err, &refused) || err.Error() != want || ExitCode(err) != ExitLibraryRefused {
		t.Fatalf("start without the fixture = %v, want %q with exit %d", err, want, ExitLibraryRefused)
	}
}
