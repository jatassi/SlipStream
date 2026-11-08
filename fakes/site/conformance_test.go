package site_test

import (
	"testing"
	"testing/synctest"

	conformance "github.com/jatassi/slipstream/conformance/ports"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/site/ports"
	"github.com/jatassi/slipstream/internal/ports/all"
	"github.com/jatassi/slipstream/internal/ports/port"
	"github.com/jatassi/slipstream/internal/searching"
)

// TestThePublicPortsFixturesConform runs the online suite against each
// public port's fixture: its search and recent cases.
func TestThePublicPortsFixturesConform(t *testing.T) {
	for _, e := range all.Registry().Entries() {
		if e.Privacy != port.Public {
			continue
		}
		t.Run(string(e.ID), func(t *testing.T) {
			f, err := ports.Embedded(e.ID)
			if err != nil {
				t.Fatal(err)
			}
			search, ok := f.Case("search")
			if !ok {
				t.Fatal("the fixture has no search case")
			}
			q, ok := search.Call.Searching(ports.Address(e.ID)).(searching.SearchCall)
			if !ok {
				t.Fatal("the fixture's search case is no search")
			}
			synctest.Test(t, func(t *testing.T) {
				served := faketest.InMemory(t, fake.Spec{
					Name: site.Fake.Name, TLS: fake.FrontTLS, Online: site.Fake.Online, Cues: []fake.CueSpec{ports.PortCase},
					New: func(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
						routes := fake.Routes{}
						_, err := ports.Serve(env, all.Registry(), ports.Embedded, routes)
						return fake.HTTP(env, routes), err
					},
				}, scenario.Scenario{})
				conformance.Run(t, conformance.Target{
					Port:    e.ID,
					Address: ports.Address(e.ID),
					Client:  served.Client,
					Timings: searching.DefaultTimings(),
					Search:  q.Query,
					Cue: func(call string) {
						served.Cue(t, ports.PortCase.Name, ports.CaseArgs{Port: string(e.ID), Case: call})
					},
				})
			})
		})
	}
}
