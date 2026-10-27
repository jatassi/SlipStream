package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world"
)

// siteEntry is what an Admin enters to add an Indexer for one of
// fixture-site's sites: a feed's address and API key, or a catalog site's
// settings and pasted cookie.
type siteEntry struct {
	Origin    string            `json:"origin"`
	Feed      string            `json:"feed,omitempty"`
	APIKey    string            `json:"apiKey,omitempty"`
	Settings  map[string]string `json:"settings,omitempty"`
	Cookie    string            `json:"cookie,omitempty"`
	UserAgent string            `json:"userAgent,omitempty"`
}

func siteCommand(args []string) int {
	fs := newFlagSet("site")
	var node string
	fs.StringVar(&node, "node", "", "")
	positional, err := parse(fs, args)
	if err != nil {
		return usage(err)
	}
	if len(positional) != 2 {
		return usage(errors.New("site: takes a run dir and a site's label"))
	}
	runDir, label := positional[0], positional[1]
	ctx, stop := interrupted()
	defer stop()
	return read(ctx, runDir, "site", args, func(context.Context) int {
		return printed("site", func() (any, error) {
			m, err := world.ReadLastManifest(runDir)
			if err != nil {
				return nil, err
			}
			return siteOf(m, label, node)
		})
	})
}

// siteOf is site label's entry in the run of m, its pasted cookie earned
// on Node node, the first Node when "".
func siteOf(m world.Manifest, label, node string) (siteEntry, error) {
	at, err := siteAddress(m)
	if err != nil {
		return siteEntry{}, err
	}
	if node == "" && len(m.Nodes) > 0 {
		node = m.Nodes[0].Name
	}
	s := scenario.Scenario{Seed: m.Scenario.Seed}
	e := siteEntry{Origin: site.Origin(at, label)}
	switch label {
	case site.Torznab, site.Prowlarr:
		e.Feed, e.APIKey = e.Origin+site.TorznabPath, site.APIKey(s)
	case site.Brook:
		e.Feed, e.Cookie = site.RSSURL(at, ""), site.Cookie(s)
	default:
		e.Settings, e.Cookie = site.Credentials(s, label, node)
		if e.Cookie != "" {
			e.UserAgent = site.BrowserUserAgent
		}
	}
	return e, nil
}

// siteAddress is where the run serves fixture-site, for the scheme of its
// sites' addresses.
func siteAddress(m world.Manifest) (fake.Address, error) {
	for _, f := range m.Fakes {
		if f.Name != site.Fake.Name {
			continue
		}
		u, err := url.Parse(f.URL)
		if err != nil {
			return fake.Address{}, fmt.Errorf("read %s's address: %w", f.Name, err)
		}
		return fake.Address{TLS: u.Scheme == "https"}, nil
	}
	return fake.Address{}, fmt.Errorf("the run has no %s", site.Fake.Name)
}
