package ports_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/site/ports"
)

func TestAPortsHostIsFixtureSitesHostOfItsID(t *testing.T) {
	if got, want := ports.Host("testport"), site.Host("testport"); got != want {
		t.Errorf("ports.Host = %s, site.Host = %s", got, want)
	}
	if !slices.Contains(site.Fake.Cues, ports.PortCase) {
		t.Errorf("fixture-site declares no %s cue", ports.PortCase.Name)
	}
}
