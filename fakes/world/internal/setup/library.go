package setup

import (
	"context"
	"errors"
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
// root on the Controller, then the Targets and Profiles as goals makes
// them, then the Indexers on the site fakes at sites, as an Admin would.
func Library(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, apiKey, mediaDir string, sites map[string]fake.Address) error {
	var controller string
	for i, n := range s.Nodes {
		if n.Setup == scenario.SetupController {
			controller = url(ready[i])
		}
	}
	self, err := controllerID(ctx, c, controller, apiKey)
	if err != nil {
		return err
	}
	for _, r := range s.Library.Roots {
		folder := filepath.Join(mediaDir, r.Folder)
		if err := os.MkdirAll(folder, 0o700); err != nil {
			return fmt.Errorf("make the %s root's folder: %w", r.ContentType, err)
		}
		if _, err := nodeapi.CreateRoot(ctx, c, controller, apiKey, r.ContentType, self, folder); err != nil {
			return err
		}
	}
	if err := goals(ctx, c, s.Library, controller, apiKey); err != nil {
		return err
	}
	for _, ix := range s.Library.Indexers {
		if _, err := nodeapi.CreateIndexer(ctx, c, controller, apiKey, ix.Name, site.URL(sites[ix.Site]), site.APIKey(s)); err != nil {
			return err
		}
	}
	return nil
}

func controllerID(ctx context.Context, c *http.Client, controller, apiKey string) (string, error) {
	rows, err := nodeapi.ListNodes(ctx, c, controller, apiKey)
	if err != nil {
		return "", err
	}
	for _, n := range rows {
		if n.Self {
			return n.ID, nil
		}
	}
	return "", errors.New("the Controller lists no row of its own")
}

// goals creates lib's Targets, replacing the rules of one whose name a
// Target has, such as the starter, and its Profiles, making each Profile
// that says so its Content Type's default.
func goals(ctx context.Context, c *http.Client, lib *scenario.Library, controller, apiKey string) error {
	targets, err := nodeapi.ListTargets(ctx, c, controller, apiKey)
	if err != nil {
		return err
	}
	for _, t := range lib.Targets {
		if id, made := targets[t.Name]; made {
			if err := nodeapi.UpdateTarget(ctx, c, controller, apiKey, id, t); err != nil {
				return err
			}
			continue
		}
		id, err := nodeapi.CreateTarget(ctx, c, controller, apiKey, t)
		if err != nil {
			return err
		}
		targets[t.Name] = id
	}
	for _, p := range lib.Profiles {
		ids := make([]string, len(p.Targets))
		for i, name := range p.Targets {
			ids[i] = targets[name]
		}
		id, err := nodeapi.CreateProfile(ctx, c, controller, apiKey, p.Name, p.ContentType, ids)
		if err != nil {
			return err
		}
		if p.Default {
			if err := nodeapi.MakeDefaultProfile(ctx, c, controller, apiKey, id); err != nil {
				return err
			}
		}
	}
	return nil
}
