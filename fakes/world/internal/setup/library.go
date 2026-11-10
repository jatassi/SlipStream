package setup

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/site/ports"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/ports/port"
)

// tvdbKey is the Admin's TVDB key a Scenario with a tvdb catalog sets,
// since TVDB answers nothing without a key and fake-tvdb takes any.
const tvdbKey = "fake-world-tvdb-key"

// Library creates the Scenario's library on its controller Node, over c
// with the Controller's API key: its Storage levels, the TVDB key when it
// has a tvdb catalog, the Custom Formats, Targets and Profiles as goals
// makes them, so a root's first scan adds Works under them, then each
// root's folder under mediaDir and the root on its Node, which shares the
// machine and so the media dir, then its seeding settings,
// then each Node's Challenge solver at the address solvers names for it,
// and each Node's Download Clients where clients says, and the Indexers on
// the site fakes at sites, as an Admin would.
func Library(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims, mediaDir string, sites map[string]fake.Address, solvers map[string]string, clients Clients) error {
	if err := StorageLevels(ctx, c, s, ready, claims); err != nil {
		return err
	}
	controller := controllerURL(s, ready)
	apiKey := claims.APIKey
	if s.TVDB != nil {
		if err := nodeapi.SetMetadataKey(ctx, c, controller, apiKey, "tvdb", tvdbKey); err != nil {
			return err
		}
	}
	if err := goals(ctx, c, s.Library, controller, apiKey); err != nil {
		return err
	}
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
	if sd := s.Library.Seeding; sd != nil {
		if err := nodeapi.SetSeeding(ctx, c, controller, apiKey, *sd); err != nil {
			return err
		}
	}
	for i, n := range s.Nodes {
		if n.ChallengeSolver {
			if err := nodeapi.SetChallengeSolver(ctx, c, controller, apiKey, claims.Members[i].ID, solvers[n.Name]); err != nil {
				return err
			}
		}
	}
	if err := addClients(ctx, c, s, claims, controller, clients); err != nil {
		return err
	}
	return addIndexers(ctx, c, s, claims, controller, sites[site.Fake.Name])
}

// Clients is where the Node named node reaches the fake Download Client of
// kind, and the folder that client downloads into as the Node sees it.
type Clients func(node, kind string) (url, folder string)

// addClients adds each Node's Download Clients as an Admin would, each at
// the address clients names, with the Client folder pairing its own name
// for its folder with the Node's when it says so, then sends the Node's New
// Grabs to the client that says so. Each client's folder is made first, as
// the client has it before anyone adds it.
func addClients(ctx context.Context, c *http.Client, s scenario.Scenario, claims Claims, controller string, clients Clients) error {
	for i, n := range s.Nodes {
		for _, dc := range n.DownloadClients {
			address, folder := clients(n.Name, dc.Kind)
			if err := addClient(ctx, c, claims, controller, claims.Members[i].ID, dc, address, folder); err != nil {
				return fmt.Errorf("node %s: %w", n.Name, err)
			}
		}
	}
	return nil
}

func addClient(ctx context.Context, c *http.Client, claims Claims, controller, nodeID string, dc scenario.DownloadClient, address, folder string) error {
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return fmt.Errorf("make the %s folder: %w", dc.Kind, err)
	}
	body := nodeapi.NewClient{
		NodeID: nodeID, Kind: dc.Kind, Name: dc.Name, Address: address,
		Username: dc.Username, Password: dc.Password, Label: dc.Label,
	}
	if dc.Pair {
		body.ClientFolders = []nodeapi.ClientFolder{{Client: dc.Folder, Node: folder}}
	}
	if dc.ImportDelay > 0 {
		body.ImportDelay = new(int64(dc.ImportDelay / time.Second))
	}
	id, err := nodeapi.CreateExternalClient(ctx, c, controller, claims.APIKey, body)
	if err != nil || !dc.GrabsTo {
		return err
	}
	return nodeapi.SetClientSettings(ctx, c, controller, claims.APIKey, nodeID, nodeapi.ClientSettings{GrabsTo: &id})
}

// addIndexers adds the Scenario's Indexers as an Admin would, each on its
// site of fixture-site at at. Catalog Indexers wait for the Indexer
// catalog's first refresh.
func addIndexers(ctx context.Context, c *http.Client, s scenario.Scenario, claims Claims, controller string, at fake.Address) error {
	if slices.ContainsFunc(s.Library.Indexers, func(ix scenario.LibraryIndexer) bool { return ix.Kind == "catalog" }) {
		if err := catalogRefreshed(ctx, c, s, controller, claims.APIKey); err != nil {
			return err
		}
	}
	for _, ix := range s.Library.Indexers {
		body := nodeapi.NewIndexer{Kind: ix.Kind, Name: ix.Name, AllowFailover: ix.AllowFailover, Priority: ix.Priority}
		if ix.Home != "" {
			body.HomeNodeID = claims.Members[slices.IndexFunc(s.Nodes, func(n scenario.Node) bool { return n.Name == ix.Home })].ID
		}
		switch ix.Kind {
		case "torznab":
			body.Address, body.APIKey = site.Origin(at, ix.Site)+site.TorznabPath, site.APIKey(s)
		case "rss":
			body.Address, body.TrackerName, body.Media = site.RSSURL(at, ""), ix.Tracker, ix.Media
		case "catalog":
			if err := catalogBody(ctx, c, s, ix, controller, claims.APIKey, &body); err != nil {
				return err
			}
		case "native":
			if err := nativeBody(s, ix, at, &body); err != nil {
				return err
			}
		}
		if err := createIndexer(ctx, c, controller, claims.APIKey, body, ix.AutomaticSearch); err != nil {
			return err
		}
	}
	return nil
}

// createIndexer creates body's Indexer, then turns its automatic search
// as automatic says when it says, since createIndexer reads
// automaticSearch false as left out and takes the default, on.
func createIndexer(ctx context.Context, c *http.Client, controller, apiKey string, body nodeapi.NewIndexer, automatic *bool) error {
	id, err := nodeapi.CreateIndexer(ctx, c, controller, apiKey, body)
	if err != nil || automatic == nil {
		return err
	}
	return nodeapi.SetAutomaticSearch(ctx, c, controller, apiKey, id, *automatic)
}

// catalogBody fills in body what an Admin enters for catalog Indexer ix:
// its definition, the seed's credentials beside ix's settings, Cinder's
// pasted cookie, and the code its captcha shows when its sign-in shows one.
func catalogBody(ctx context.Context, c *http.Client, s scenario.Scenario, ix scenario.LibraryIndexer, controller, apiKey string, body *nodeapi.NewIndexer) error {
	values, cookie := site.Credentials(s, ix.Site, s.Home(ix))
	maps.Copy(values, ix.Settings)
	body.Definition, body.Values = cmp.Or(ix.Definition, ix.Site), values
	if cookie != "" {
		body.Cookie = &nodeapi.PastedCookie{Cookie: cookie, UserAgent: site.BrowserUserAgent}
	}
	captcha, err := nodeapi.SignInCaptcha(ctx, c, controller, apiKey, body.Definition)
	if err != nil || !captcha {
		return err
	}
	body.Captcha, err = answered(ctx, c, controller, apiKey, *body)
	return err
}

// nativeBody fills in body what an Admin enters for native Indexer ix:
// its port at its site's host, and the seed's credentials on a family
// site or the settings and pasted cookie of its port's fixture on any
// other, beside ix's settings.
func nativeBody(s scenario.Scenario, ix scenario.LibraryIndexer, at fake.Address, body *nodeapi.NewIndexer) error {
	body.Port, body.BaseAddress = ix.Port, site.Origin(at, ix.Site)+"/"
	values, cookie, userAgent := map[string]string{}, "", site.BrowserUserAgent
	if slices.Contains(site.Families, ix.Site) {
		values, cookie = site.Credentials(s, ix.Site, s.Home(ix))
	} else {
		f, err := ports.Embedded(port.ID(ix.Port))
		if err != nil {
			return err
		}
		maps.Copy(values, f.Settings)
		if f.Session != nil && f.Session.Pasted {
			cookie, userAgent = f.Session.Cookie, f.Session.UserAgent
		}
	}
	maps.Copy(values, ix.Settings)
	body.Values = values
	if cookie != "" {
		body.Cookie = &nodeapi.PastedCookie{Cookie: cookie, UserAgent: userAgent}
	}
	return nil
}

// catalogRefreshed waits for the Indexer catalog's first refresh, which
// the Controller runs at start, on the Admin's event stream.
func catalogRefreshed(ctx context.Context, c *http.Client, s scenario.Scenario, controller, apiKey string) error {
	if s.Admin == nil {
		return errors.New("wait for the Indexer catalog: the Scenario has no Admin to read the event stream as")
	}
	session, err := nodeapi.SignIn(ctx, c, controller, s.Admin.Email, s.Admin.PIN)
	if err != nil {
		return err
	}
	if err := nodeapi.AwaitCatalog(ctx, c, controller, apiKey, session); err != nil {
		return err
	}
	return nodeapi.SignOut(ctx, c, controller, session)
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

// goals creates lib's Custom Formats, then its Targets, replacing the
// rules of one whose name a Target has, such as the starter, and its
// Profiles, making each Profile that says so its Content Type's default.
func goals(ctx context.Context, c *http.Client, lib *scenario.Library, controller, apiKey string) error {
	formats, err := createFormats(ctx, c, lib.Formats, controller, apiKey)
	if err != nil {
		return err
	}
	targets, err := nodeapi.ListTargets(ctx, c, controller, apiKey)
	if err != nil {
		return err
	}
	for _, t := range lib.Targets {
		if id, made := targets[t.Name]; made {
			if err := nodeapi.UpdateTarget(ctx, c, controller, apiKey, id, t, formats); err != nil {
				return err
			}
			continue
		}
		id, err := nodeapi.CreateTarget(ctx, c, controller, apiKey, t, formats)
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
		id, err := nodeapi.CreateProfile(ctx, c, controller, apiKey, p.Name, p.ContentType, ids, p.Cuts)
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

// createFormats creates formats and returns their ids by name.
func createFormats(ctx context.Context, c *http.Client, formats []scenario.LibraryFormat, controller, apiKey string) (map[string]string, error) {
	ids := map[string]string{}
	for _, f := range formats {
		id, err := nodeapi.CreateFormat(ctx, c, controller, apiKey, f)
		if err != nil {
			return nil, err
		}
		ids[f.Name] = id
	}
	return ids, nil
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

// Works adds a Work for every record of the Scenario's library works'
// Content Type its fake serves, as library works says, on its controller
// Node, and returns how many it added: each movie TMDB still holds, or
// each series TVDB lists.
func Works(ctx context.Context, c *http.Client, s scenario.Scenario, ready []node.Ready, claims Claims) (int, error) {
	w := s.Library.Works
	controller := controllerURL(s, ready)
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	slots := make(chan struct{}, addsAtOnce)
	var wg sync.WaitGroup
	added := 0
	for key := range keysOf(s, w.ContentType) {
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
			if err := nodeapi.AddWork(ctx, c, controller, claims.APIKey, w.ContentType, strconv.FormatInt(key, 10), w.Preset); err != nil {
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

// keysOf is the key of each record of contentType the Scenario's fake
// serves that its source still holds.
func keysOf(s scenario.Scenario, contentType string) iter.Seq[int64] {
	return func(yield func(int64) bool) {
		if contentType == "series" {
			for series := range s.TVDB.All() {
				if !yield(series.ID) {
					return
				}
			}
			return
		}
		for m := range s.TMDB.All() {
			if !m.Deleted && !yield(m.ID) {
				return
			}
		}
	}
}
