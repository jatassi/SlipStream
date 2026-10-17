package api_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

func TestAZeroLevelIsNoLevel(t *testing.T) {
	r := api.NewRouter(api.Services{})
	o := r.Owner("fixture", api.Public)
	handler := func(context.Context, identity.Admin, *struct{}) (*struct{}, error) { return nil, nil }
	for _, tc := range []struct {
		register func()
		want     string
	}{
		{func() {
			api.Register(o, api.Level[identity.Admin]{}, api.Op{ID: "getFixture", Method: http.MethodGet, Path: "/fixture"}, handler)
		}, "getFixture: a zero api.Level; use api.Public, api.Setup, api.Bearer, api.Account, api.Confirmed, api.Admin or api.Reconfirmed"},
		{func() { r.Owner("nodes", api.Level[identity.Admin]{}) }, "api: owner nodes: its floor is a zero api.Level; use api.Public, api.Setup, api.Bearer, api.Account, api.Confirmed, api.Admin or api.Reconfirmed"},
	} {
		if got := panicOf(tc.register); got != tc.want {
			t.Errorf("panic = %q, want %q", got, tc.want)
		}
	}
}

func panicOf(f func()) (msg string) {
	defer func() { msg = fmt.Sprint(recover()) }()
	f()
	return "<nil>"
}
