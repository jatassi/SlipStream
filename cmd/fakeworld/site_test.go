package main

import (
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world"
)

func TestSiteOfNamesWhatAnAdminEnters(t *testing.T) {
	t.Parallel()
	m := world.Manifest{
		Scenario: world.ScenarioRef{Seed: "seed-1"},
		Nodes:    []world.Node{{Name: "home"}},
		Fakes:    []world.Fake{{Name: site.Fake.Name, URL: "https://fixture-site.fake.localhost:4443"}},
	}
	s := scenario.Scenario{Seed: "seed-1"}
	cases := []struct {
		label string
		want  siteEntry
	}{
		{site.Torznab, siteEntry{
			Origin: "https://torznab.fixture-site.fake.localhost", Feed: "https://torznab.fixture-site.fake.localhost" + site.TorznabPath, APIKey: site.APIKey(s),
		}},
		{site.Larkspur, siteEntry{
			Origin:   "https://larkspur.fixture-site.fake.localhost",
			Settings: map[string]string{"username": site.Username, "password": site.Password(s, site.Larkspur)},
		}},
	}
	for _, c := range cases {
		got, err := siteOf(m, c.label, "")
		if err != nil {
			t.Fatalf("siteOf(%s): %v", c.label, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("siteOf(%s) = %+v, want %+v", c.label, got, c.want)
		}
	}
	if _, err := siteOf(world.Manifest{}, site.Larkspur, ""); err == nil {
		t.Error("siteOf a run with no fixture-site succeeded")
	}
}
