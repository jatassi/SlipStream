package api_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

type servedConn struct {
	net.Conn
	from role.NodeID
}

func (c servedConn) ServedFor() role.NodeID { return c.from }

func TestArrivalOfNamesTheListenerTheServingNodeThePeerAndTheSource(t *testing.T) {
	var proxy api.Proxy
	if err := proxy.UnmarshalText([]byte("127.0.0.1")); err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{TrustedProxies: []api.Proxy{proxy}})
	var got api.Arrival
	api.Register(r.Owner("fixture", api.Public), api.Public, api.Op{ID: "getFixture", Method: http.MethodGet, Path: "/fixture"},
		func(ctx context.Context, _ identity.Anyone, _ *struct{}) (*struct{}, error) {
			got = api.ArrivalOf(ctx)
			return &struct{}{}, nil
		})
	h := r.Handler()
	worker := role.NodeID{UUID: storage.NewUUID()}
	addr := netip.MustParseAddr
	for _, tc := range []struct {
		name    string
		conn    func(context.Context) context.Context
		from    string
		headers map[string]string
		want    api.Arrival
	}{
		{
			name: "a player on the bound listener",
			conn: func(ctx context.Context) context.Context { return api.OnListener(ctx, nil) },
			from: "203.0.113.5:50000",
			want: api.Arrival{Listener: api.Bound, Peer: addr("203.0.113.5"), Source: addr("203.0.113.5"), Host: "atlas.example.org"},
		},
		{
			name:    "a player behind a trusted proxy on the bound listener",
			conn:    func(ctx context.Context) context.Context { return api.OnListener(ctx, nil) },
			from:    "127.0.0.1:50000",
			headers: map[string]string{"X-Forwarded-For": "192.0.2.44"},
			want:    api.Arrival{Listener: api.Bound, Peer: addr("127.0.0.1"), Source: addr("192.0.2.44"), Host: "atlas.example.org"},
		},
		{
			name:    "an untrusted peer forging X-Forwarded-For",
			conn:    func(ctx context.Context) context.Context { return api.OnListener(ctx, nil) },
			from:    "[2001:db8::7]:50000",
			headers: map[string]string{"X-Forwarded-For": "192.0.2.44"},
			want:    api.Arrival{Listener: api.Bound, Peer: addr("2001:db8::7"), Source: addr("2001:db8::7"), Host: "atlas.example.org"},
		},
		{
			name:    "a player through a Serving Worker's Serve stream",
			conn:    func(ctx context.Context) context.Context { return api.OverLink(ctx, servedConn{from: worker}) },
			from:    "link",
			headers: map[string]string{"Forwarded": api.Forwarded(addr("198.51.100.7"))},
			want:    api.Arrival{Listener: api.Served, Via: worker, Source: addr("198.51.100.7"), Host: "atlas.example.org"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(tc.conn(t.Context()), http.MethodGet, "/api/v1/fixture", http.NoBody)
			req.Host, req.RemoteAddr = "atlas.example.org", tc.from
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			got = api.Arrival{}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Code != http.StatusNoContent || got != tc.want {
				t.Fatalf("status %d, ArrivalOf = %+v; want %+v", w.Code, got, tc.want)
			}
		})
	}
}
