// Package proxy is fake-proxy: the Admin's own reverse proxy, such as a
// VPS in front of home, for S51 (Reachability diagnostic). The world
// terminates its TLS, so tls=expired, tls=wrong-name and status=401 are
// fault rules on it, set per tester by origin. It routes each request by
// Host to the Scenario's target, a Node's listener or a Media Server
// fake's front, with X-Forwarded-For set to the tester's address.
package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/netsite"
	"github.com/jatassi/slipstream/fakes/outside"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

var Fake = fake.Spec{
	Name:  string(outside.ProxyTarget),
	TLS:   fake.FrontTLS,
	Plays: netsite.Plays,
	New: func(s scenario.Scenario, env fake.Env) (fake.Server, error) {
		site, err := netsite.From(env, s)
		if err != nil {
			return nil, err
		}
		return fake.HTTP(env, fake.Routes{"proxy": {Pattern: "/", Handler: func(w http.ResponseWriter, r *http.Request) {
			svc, ok := site.Proxied(hostOf(r))
			if !ok {
				http.Error(w, "the proxy routes no such host", http.StatusNotFound)
				return
			}
			at, err := site.Where(svc)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadGateway)
				return
			}
			src := site.Source(fake.OriginOf(r.Context()))
			(&httputil.ReverseProxy{Transport: transport, Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(&url.URL{Scheme: "http", Host: at.String()})
				pr.Out.Host = pr.In.Host
				pr.Out.Header.Set("X-Forwarded-For", src.String())
			}}).ServeHTTP(w, r)
		}}}), nil
	},
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("the Admin's own proxy is theirs, and the Fake world's names under .localhost reach no real one"),
}

var transport = &http.Transport{DialContext: netsite.Dialer.DialContext}

func hostOf(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.Host); err == nil {
		return host
	}
	return r.Host
}
