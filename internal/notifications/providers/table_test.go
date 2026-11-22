package providers_test

import (
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestEachProviderBuildsEachMessageAsItsGolden(t *testing.T) {
	for _, p := range providers.Table {
		t.Run(string(p.ID), func(t *testing.T) {
			c, ok := providertest.Cases[p.ID]
			if !ok {
				t.Fatalf("%s has no fixture connection", p.ID)
			}
			providertest.Golden(t, p, c)
		})
	}
}

func TestTheTableListsElevenProvidersWithDistinctFields(t *testing.T) {
	var ids []providers.ID
	for _, p := range providers.Table {
		ids = append(ids, p.ID)
		keys := map[string]bool{}
		for _, f := range p.Fields {
			if keys[f.Key] {
				t.Errorf("%s has two fields %s", p.ID, f.Key)
			}
			keys[f.Key] = true
			if f.Shown == providers.ShownHost && (!f.Secret || f.Kind != providers.URL) {
				t.Errorf("%s.%s shows a host but isn't a secret URL", p.ID, f.Key)
			}
		}
		if p.Build == nil || p.Classify == nil || p.Host == nil || p.Label == "" {
			t.Errorf("%s is missing part of its row", p.ID)
		}
	}
	if want := []providers.ID{"discord", "telegram", "slack", "pushover", "gotify", "ntfy", "pushbullet", "apprise", "email", "webhook", "script"}; !slices.Equal(ids, want) {
		t.Errorf("Table = %v, want %v", ids, want)
	}
}

func TestTheFixedHostsAreTheEndpoints(t *testing.T) {
	e := providers.DefaultEndpoints()
	for id, want := range map[providers.ID]string{"telegram": "https://api.telegram.org", "pushover": "https://api.pushover.net", "pushbullet": "https://api.pushbullet.com", "ntfy": "https://ntfy.sh"} {
		p, _ := providers.ByID(id)
		if host, ok := p.Host(providers.Values{}, e); !ok || host.String() != want {
			t.Errorf("%s host = %v, want %s", id, host, want)
		}
	}
	for _, id := range []providers.ID{"email", "script"} {
		p, _ := providers.ByID(id)
		if _, ok := p.Host(providers.Values{}, e); ok {
			t.Errorf("%s names a host", id)
		}
	}
}
