package all

import (
	"testing"

	"github.com/jatassi/slipstream/internal/newznab"
	"github.com/jatassi/slipstream/internal/ports/port"
)

type settings struct{ port.APIKeyField }

func TestAPortAddedFromItsOwnFileJoinsTheRegistry(t *testing.T) {
	search := func(*port.Indexer[settings], port.SearchCriteria) (port.Chain, error) { return port.Chain{}, nil }
	parse := func(*port.Indexer[settings], *port.IndexerResponse) ([]port.TorrentInfo, error) { return nil, nil }
	before := len(Registry().Entries())
	t.Cleanup(func(saved []port.Port) func() { return func() { ports = saved } }(ports))
	add(port.Define(port.Entry{
		ID: "lantern", Name: "Lantern", Description: "Lantern is a private tracker for tests.", Privacy: port.Private,
		IndexerURLs:  []string{"https://lantern.test/"},
		Capabilities: port.Capabilities{SearchParams: []port.Param{port.Q}, Categories: port.Categories(port.Map(1, newznab.Movies, "Movies"))},
	}, port.Impl[settings]{Login: port.NoLogin{}, Basic: search, Parse: parse}))
	r := Registry()
	if e, ok := r.Lookup("lantern"); !ok || e.Key != "API key" || len(r.Entries()) != before+1 {
		t.Errorf("Lookup(lantern) = %+v, %v among %d entries", e, ok, len(r.Entries()))
	}
}
