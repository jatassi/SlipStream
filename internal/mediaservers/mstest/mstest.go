// Package mstest is a Controller's Media Servers over a test library:
// home, the Controller, and seedbox, a Worker that holds Media Server
// link, each reaching the fake servers a test serves at the hosts it
// names. Each Node answers with its real MediaServerLinkService, called in
// process rather than over the link. Make one inside synctest.Test.
package mstest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/providers/plextv"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// World is the Media Servers of home, with seedbox paired.
type World struct {
	Lib           *storage.DB[storage.Library]
	Sealer        *secret.Sealer
	Home, Seedbox role.NodeID
	Service       mediaservers.Service
	// Renewal is the Plex sign-in task, run as the scheduler runs it.
	Renewal tasks.Task
	ms      *mediaservers.MediaServers
	net     *network
	roots   *roots.Roots
	rehomed chan struct{}
	mu      sync.Mutex
	changed []mediaservers.Changed
}

// Schemas are the library schemas the Media Servers read.
func Schemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(), identity.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), roots.Schema(), mediaservers.Schema(),
	}
}

func New(t *testing.T) *World {
	t.Helper()
	w := &World{
		Lib: storagetest.Library(t, Schemas()...), Sealer: secrettest.Sealer(t),
		Home: role.NodeID{UUID: storage.NewUUID()}, Seedbox: role.NodeID{UUID: storage.NewUUID()},
		net: &network{served: map[string]http.RoundTripper{}}, rehomed: make(chan struct{}, 1),
	}
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return errors.Join(nodestest.Insert(ctx, tx, w.Home, "home"), nodestest.Insert(ctx, tx, w.Seedbox, "seedbox"),
			nodestest.Address(ctx, tx, w.Home, 0, HomeIP+":7878"), nodestest.Address(ctx, tx, w.Seedbox, 0, SeedboxIP+":7878"))
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := rootstest.Config(t, w.Lib, contenttypetest.ID)
	cfg.Describe = func(works.TypeID) (works.Description, bool) { return contenttypetest.Description, true }
	cfg.Changed, cfg.Contents = rootsChanged{w}, empty{}
	if w.roots, err = roots.Open(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(t.Output(), nil))
	// A route test, a sign-in and a library read are each one call.
	timings := link.Timings{Call: 10 * time.Second}
	ms := mediaservers.New(mediaservers.Config{
		Library: w.Lib, Sealer: w.Sealer, Roster: roster{w.Home, w.Seedbox}, Kinds: mediaserverlink.Kinds,
		HTTP: w.net.client(), Timings: timings, PlexTV: plexTV,
		Link: func(role.NodeID) mediaserverv1connect.MediaServerLinkServiceHandler {
			return mediaserverlink.NewLink(mediaserverlink.Config{Kinds: mediaserverlink.Kinds, HTTP: w.net.client(), Timings: timings, Log: log}).Service()
		},
		RootWords:    w.roots.WordsIn,
		RootsChanged: func(context.Context) <-chan struct{} { return w.rehomed },
		Tasks:        &cadences{},
		Events:       mediaservers.Events{Changed: changedEvents{w}},
		Touch:        func(...role.NodeID) {},
		Zone:         time.UTC,
		Log:          log,
	})
	w.ms, w.Service, w.Renewal = ms, ms.Service(), ms.RenewalTask()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = ms.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return w
}

// MediaServers is the Media Servers, as the API asks a Controller for them.
func (w *World) MediaServers() (mediaservers.Service, bool) { return w.Service, true }

// Jellyfin serves srv at host, such as 192.168.1.20:8096, in place of the
// server there before, with each library folder made in its media dir.
func (w *World) Jellyfin(t *testing.T, host string, srv jellyfin.Server) {
	t.Helper()
	media := t.TempDir()
	for _, l := range srv.Libraries {
		for _, f := range l.Folders {
			rel, err := filepath.Rel(srv.Root, f)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(media, rel), 0o700); err != nil {
				t.Fatal(err)
			}
		}
	}
	f := faketest.InMemory(t, jellyfin.Playing(srv), scenario.Scenario{Seed: "mstest"}, faketest.WithMedia(media))
	w.net.mu.Lock()
	defer w.net.mu.Unlock()
	w.net.served[host] = f.Client.Transport
}

// The link addresses of home and seedbox, where Plex lists a local
// connection to the Plex each runs.
const (
	HomeIP    = "192.168.1.10"
	SeedboxIP = "10.0.0.7"
)

// plexTV is where the World's plex.tv is served.
var plexTV = plextv.Endpoints{
	API:     endpoint.MustParseURL("https://plextv.fake.localhost"),
	Auth:    endpoint.MustParseURL("http://plextv-auth.fake.localhost/auth"),
	Desktop: endpoint.MustParseURL("https://app.plex.invalid/desktop"),
}

// Plex is plex.tv signing in to account, which owns srv, and srv served
// at home's and seedbox's addresses on port 32400, in that order in
// resources. Each connection is that Node's front, as in the world, so a
// fault rule's origin names the connection, whoever dials it.
type Plex struct {
	TV, Server faketest.Served
}

func (w *World) Plex(t *testing.T, srv fakeplex.Server, a fakeplextv.Account) Plex {
	t.Helper()
	s := scenario.Scenario{Seed: "mstest"}
	shared := fake.NewShared()
	p := Plex{Server: faketest.InMemory(t, fakeplex.Playing(srv), s, faketest.WithShared(shared), faketest.WithOrigins("home", "seedbox"))}
	fronts := fake.Fronts{}
	hosts := map[string]string{}
	for _, f := range []struct{ node, ip string }{{"home", HomeIP}, {"seedbox", SeedboxIP}} {
		at := netip.MustParseAddrPort(f.ip + ":32400")
		fronts = append(fronts, fake.Front{Target: "plex", Node: f.node, Address: fake.Address{AddrPort: at}})
		hosts[at.String()] = f.node
	}
	p.TV = faketest.InMemory(t, fakeplextv.Playing(a), s, faketest.WithShared(shared), faketest.WithFronts(fronts))
	w.net.mu.Lock()
	defer w.net.mu.Unlock()
	for host, node := range hosts {
		w.net.served[host] = p.Server.From(t, node).Transport
	}
	w.net.served[plexTV.API.Hostname()] = p.TV.Client.Transport
	return p
}

// Approve signs the account in on the sign-in page at authURL, as the
// Admin does on plex.tv.
func (p Plex) Approve(t *testing.T, authURL string) {
	t.Helper()
	_, fragment, _ := strings.Cut(authURL, "#?")
	q, err := url.ParseQuery(fragment)
	if err != nil {
		t.Fatal(err)
	}
	p.TV.Cue(t, fakeplextv.Approve.Name, map[string]string{"code": q.Get("code")})
}

// Intent is node's intent as the Role sends it.
func (w *World) Intent(t *testing.T, node role.NodeID) *msv1.Intent {
	t.Helper()
	var in *msv1.Intent
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		in, err = w.ms.Intent(ctx, r, node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return in
}

// Root makes a fresh folder named name a library root on home.
func (w *World) Root(t *testing.T, name string) roots.Root {
	t.Helper()
	v, err := w.roots.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: contenttypetest.ID, Node: w.Home, Path: folder(t, name)})
	if err != nil {
		t.Fatal(err)
	}
	return v.Root
}

// Rehome moves root to a fresh folder named name on home, as an Admin
// does, and waits until the Media Servers follow it.
func (w *World) Rehome(t *testing.T, root roots.ID, name string) {
	t.Helper()
	to := folder(t, name)
	if _, err := w.roots.Rehome(t.Context(), identity.Admin{}, root, roots.Rehome{Path: &to}); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

func folder(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Changed is every media-servers.changed published so far.
func (w *World) Changed() []mediaservers.Changed {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]mediaservers.Changed(nil), w.changed...)
}

type changedEvents struct{ w *World }

func (e changedEvents) Publish(_ context.Context, v mediaservers.Changed) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	e.w.changed = append(e.w.changed, v)
}

type rootsChanged struct{ w *World }

func (e rootsChanged) Publish(context.Context, roots.Changed) {
	select {
	case e.w.rehomed <- struct{}{}:
	default:
	}
}

// empty is roots that record no files.
type empty struct{}

func (empty) Use(context.Context, storage.Reader[storage.Library], []roots.ID) (map[roots.ID]roots.Use, error) {
	return map[roots.ID]roots.Use{}, nil
}

func (empty) HasFiles(context.Context, storage.Reader[storage.Library], works.WorkID) (bool, error) {
	return false, nil
}

func (empty) Proof(context.Context, storage.Reader[storage.Library], roots.ID) ([]string, error) {
	return nil, nil
}

func (empty) Recorded(context.Context, storage.Reader[storage.Library], roots.ID) ([]storing.Recorded, error) {
	return nil, nil
}

func (empty) Forget(context.Context, *storage.Tx[storage.Library], roots.ID, []storing.Rel) error {
	return nil
}

func (empty) ForgetGone(context.Context, *storage.Tx[storage.Library], roots.ID) error { return nil }

func (empty) ForgetGoneOf(context.Context, *storage.Tx[storage.Library], works.WorkID) error {
	return nil
}

// network is the fakes served, by host.
type network struct {
	mu     sync.Mutex
	served map[string]http.RoundTripper
}

var errNoAnswer = errors.New("no answer")

func (n *network) client() *http.Client {
	return &http.Client{Transport: roundTrip(func(req *http.Request) (*http.Response, error) {
		n.mu.Lock()
		rt, ok := n.served[req.URL.Host]
		n.mu.Unlock()
		if !ok {
			return nil, errNoAnswer
		}
		return rt.RoundTrip(req)
	})}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// roster is home's view of its Nodes: each in step, holding Media Server
// link.
type roster struct{ self, worker role.NodeID }

func (r roster) Self() role.NodeID                      { return r.self }
func (r roster) Holding(role.Name) []role.NodeID        { return []role.NodeID{r.self, r.worker} }
func (roster) Changed() <-chan struct{}                 { return nil }
func (roster) InStep(role.NodeID) bool                  { return true }
func (roster) Holds(name role.Name, _ role.NodeID) bool { return name == role.MediaServerLink }

// cadences is the renewal task's cadence as tasks would store it.
type cadences struct {
	mu    sync.Mutex
	every *int64
}

func (c *cadences) Every(context.Context, tasks.TaskID) (*int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.every, nil
}

func (c *cadences) SetCadence(_ context.Context, _ tasks.TaskID, v setting.Stored[int64]) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.every = v.Override()
	return nil
}
