package all

import (
	"slices"
	"strings"
	"testing"
	"unicode"

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
		ID: "lantern", Name: "Lantern", Description: "Lantern is a private Tracker for tests.", Privacy: port.Private,
		IndexerURLs:  []string{"https://lantern.test/"},
		Capabilities: port.Capabilities{SearchParams: []port.Param{port.Q}, Categories: port.Categories(port.Map(1, newznab.Movies, "Movies"))},
	}, port.Impl[settings]{Login: port.NoLogin{}, Basic: search, Parse: parse}))
	r := Registry()
	if e, ok := r.Lookup("lantern"); !ok || e.Key != "API key" || len(r.Entries()) != before+1 {
		t.Errorf("Lookup(lantern) = %+v, %v among %d entries", e, ok, len(r.Entries()))
	}
}

// glossaryTerms are the GLOSSARY.md terms a description may use, which
// keep their capital.
var glossaryTerms = []string{"tracker", "trackers", "release", "releases"}

func TestEveryPortDescribesItsSiteInOneSentenceOfGlossaryTerms(t *testing.T) {
	for _, e := range Registry().Entries() {
		words := strings.FieldsFunc(e.Description, func(r rune) bool { return !unicode.IsLetter(r) && r != '-' })
		if !strings.HasSuffix(e.Description, ".") {
			t.Errorf("%s: %q doesn't end with a period", e.ID, e.Description)
		}
		for _, w := range words {
			if slices.Contains(glossaryTerms, w) {
				t.Errorf("%s: %q writes the glossary term %q in lowercase", e.ID, e.Description, w)
			}
		}
		for _, p := range []port.Privacy{port.Private, port.SemiPrivate, port.Public} {
			if slices.Contains(words, string(p)) != (p == e.Privacy) {
				t.Errorf("%s: %q doesn't name its privacy, %s, alone", e.ID, e.Description, e.Privacy)
			}
		}
	}
}
