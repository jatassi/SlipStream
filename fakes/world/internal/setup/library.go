package setup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
)

// Library creates the Scenario's library on its controller Node, over c
// with the Controller's API key: each root's folder under mediaDir and the
// root, then the Targets, then the Profiles, then the Indexers on the
// site fakes at sites, as an Admin would.
func Library(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, apiKey, mediaDir string, sites map[string]fake.Address) error {
	var controller string
	for i, n := range s.Nodes {
		if n.Setup == scenario.SetupController {
			controller = url(ready[i])
		}
	}
	for _, r := range s.Library.Roots {
		folder := filepath.Join(mediaDir, r.Folder)
		if err := os.MkdirAll(folder, 0o700); err != nil {
			return fmt.Errorf("make the %s root's folder: %w", r.ContentType, err)
		}
		if _, err := nodeapi.CreateRoot(ctx, c, controller, apiKey, r.ContentType, folder); err != nil {
			return err
		}
	}
	targets := map[string]string{}
	for _, t := range s.Library.Targets {
		id, err := nodeapi.CreateTarget(ctx, c, controller, apiKey, t.Name, t.Resolutions)
		if err != nil {
			return err
		}
		targets[t.Name] = id
	}
	for _, p := range s.Library.Profiles {
		ids := make([]string, len(p.Targets))
		for i, name := range p.Targets {
			ids[i] = targets[name]
		}
		if _, err := nodeapi.CreateProfile(ctx, c, controller, apiKey, p.Name, p.ContentType, ids); err != nil {
			return err
		}
	}
	for _, ix := range s.Library.Indexers {
		if _, err := nodeapi.CreateIndexer(ctx, c, controller, apiKey, ix.Name, site.URL(sites[ix.Site]), site.APIKey(s)); err != nil {
			return err
		}
	}
	return nil
}
