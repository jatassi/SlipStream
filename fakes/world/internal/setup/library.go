package setup

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"

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
	return addIndexers(ctx, c, s, claims, controller, sites[site.Fake.Name])
}

// addIndexers adds the Scenario's Indexers as an Admin would, each on its
// site of fixture-site at at. A catalog Indexer waits for the Indexer
// catalog's first refresh, and one whose sign-in shows a captcha is added
// with the code its image shows.
func addIndexers(ctx context.Context, c *http.Client, s scenario.Scenario, claims Claims, controller string, at fake.Address) error {
	refreshed := false
	for _, ix := range s.Library.Indexers {
		body := nodeapi.NewIndexer{Kind: ix.Kind, Name: ix.Name, AllowFailover: ix.AllowFailover}
		if ix.Home != "" {
			body.HomeNodeID = claims.Members[slices.IndexFunc(s.Nodes, func(n scenario.Node) bool { return n.Name == ix.Home })].ID
		}
		switch ix.Kind {
		case "torznab":
			body.Address, body.APIKey = site.URL(at), site.APIKey(s)
		case "rss":
			body.Address, body.TrackerName, body.Media = site.RSSURL(at, ""), ix.Tracker, ix.Media
		case "catalog":
			if !refreshed {
				if err := catalogRefreshed(ctx, c, controller, claims.APIKey); err != nil {
					return err
				}
				refreshed = true
			}
			values, cookie := site.Credentials(s, ix.Site, s.Home(ix))
			maps.Copy(values, ix.Settings)
			body.Definition, body.Values = cmp.Or(ix.Definition, ix.Site), values
			if cookie != "" {
				body.Cookie = &nodeapi.PastedCookie{Cookie: cookie, UserAgent: site.BrowserUserAgent}
			}
			captcha, err := nodeapi.SignInCaptcha(ctx, c, controller, claims.APIKey, body.Definition)
			if err != nil {
				return err
			}
			if captcha {
				if body.Captcha, err = answered(ctx, c, controller, claims.APIKey, body); err != nil {
					return err
				}
			}
		}
		if _, err := nodeapi.CreateIndexer(ctx, c, controller, claims.APIKey, body); err != nil {
			return err
		}
	}
	return nil
}

// catalogRefreshed waits for the Indexer catalog's first refresh, which
// the Controller runs at start.
func catalogRefreshed(ctx context.Context, c *http.Client, controller, apiKey string) error {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		done, err := nodeapi.CatalogRefreshed(ctx, c, controller, apiKey)
		if err != nil || done {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for the Indexer catalog's first refresh: %w", ctx.Err())
		case <-tick.C:
		}
	}
}

// answered is the answer to a captcha of ix's sign-in: the code its image
// shows, as an Admin types it.
func answered(ctx context.Context, c *http.Client, controller, apiKey string, ix nodeapi.NewIndexer) (*nodeapi.CaptchaAnswer, error) {
	attempt, img, err := nodeapi.IndexerCaptcha(ctx, c, controller, apiKey, ix)
	if err != nil {
		return nil, err
	}
	code, err := site.CaptchaText(img)
	if err != nil {
		return nil, fmt.Errorf("read Indexer %s's captcha: %w", ix.Name, err)
	}
	return &nodeapi.CaptchaAnswer{Attempt: attempt, Answer: code}, nil
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
