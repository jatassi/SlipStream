package setup

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
)

// Library creates the Scenario's library on its controller Node, over c
// with the Controller's API key: its Storage levels, each root's folder
// under mediaDir and the root on its Node, which shares the machine and so
// the media dir, then the Targets and Profiles as goals makes them, then
// the Indexers on the site fakes at sites, as an Admin would.
func Library(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims, mediaDir string, sites map[string]fake.Address) error {
	if err := StorageLevels(ctx, c, s, ready, claims); err != nil {
		return err
	}
	controller := controllerURL(s, ready)
	apiKey := claims.APIKey
	for _, r := range s.Library.Roots {
		folder := filepath.Join(mediaDir, r.Folder)
		if err := os.MkdirAll(folder, 0o700); err != nil {
			return fmt.Errorf("make the %s root's folder: %w", r.ContentType, err)
		}
		on := slices.IndexFunc(s.Nodes, func(n scenario.Node) bool { return n.Name == s.RootNode(r) })
		if _, err := nodeapi.CreateRoot(ctx, c, controller, apiKey, r.ContentType, claims.Members[on].ID, folder); err != nil {
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

// StorageLevels sets the Scenario's Storage levels on its controller Node,
// if it has any.
func StorageLevels(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims) error {
	if s.Library == nil || s.Library.StorageLevels == nil {
		return nil
	}
	return nodeapi.SetStorageLevels(ctx, c, controllerURL(s, ready), claims.APIKey, *s.Library.StorageLevels)
}

func controllerURL(s scenario.Scenario, ready []node.Ready) string {
	i := slices.IndexFunc(s.Nodes, func(n scenario.Node) bool { return n.Setup == scenario.SetupController })
	return url(ready[i])
}

const addsAtOnce = 8

// Works adds a Work for every movie fake-tmdb serves that TMDB still
// holds, as the Scenario's library works says, on its controller Node,
// and returns how many it added.
func Works(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims) (int, error) {
	w := s.Library.Works
	controller := controllerURL(s, ready)
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	slots := make(chan struct{}, addsAtOnce)
	var wg sync.WaitGroup
	added := 0
	for m := range s.TMDB.All() {
		if m.Deleted {
			continue
		}
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			break
		}
		added++
		wg.Go(func() {
			defer func() { <-slots }()
			if err := nodeapi.AddWork(ctx, c, controller, claims.APIKey, w.ContentType, strconv.FormatInt(m.ID, 10), w.Monitor); err != nil {
				cancel(err)
			}
		})
	}
	wg.Wait()
	if err := context.Cause(ctx); err != nil {
		return 0, fmt.Errorf("add the library's Works: %w", err)
	}
	return added, nil
}
