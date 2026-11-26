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
	"maps"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/emby"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	fakeplex "github.com/jatassi/slipstream/fakes/plex"
	fakeplextv "github.com/jatassi/slipstream/fakes/plextv"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mail/mailtest"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/access"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/providers/plex"
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

// Timings are the Media Server link's shipped defaults, spelled out: only
// the Node's composition root reads its defaults.
var Timings = link.Timings{
	Call: 10 * time.Second, KeepAlive: 30 * time.Second, ReconnectMin: time.Second, ReconnectMax: time.Minute,
	RetryMin: 10 * time.Second, RetryMax: 15 * time.Minute, PinPoll: time.Second,
	ScanEndPoll: 250 * time.Millisecond, ScanEndPollMax: 5 * time.Second, ScanEndBound: 2 * time.Hour,
	MovedConfirm: 2 * time.Minute, MovedStep: 5 * time.Minute, StepPoll: 2 * time.Second,
	RenameSnapshot: 20 * time.Second, UpgradePairing: 2 * time.Minute, SectionFolders: 10, SessionCap: 100,
	IndexAge: 24 * time.Hour, SentAge: 7 * 24 * time.Hour, IndexPage: 500,
}

// World is the Media Servers of home, with seedbox paired.
type World struct {
	Lib    *storage.DB[storage.Library]
	Sealer *secret.Sealer
	// Identity is the Accounts, whose Signup links and deletes reach the
	// Media Servers through their Granter.
	Identity      *identity.Service
	Home, Seedbox role.NodeID
	Service       mediaservers.Service
	// Renewal is the Plex sign-in task, run as the scheduler runs it.
	Renewal tasks.Task
	// Mail captures every email the Media Servers send, waited or through
	// the Outbox.
	Mail     *mailtest.Sender
	ms       *mediaservers.MediaServers
	net      *network
	links    *links
	roots    *roots.Roots
	rehomed  chan struct{}
	mu       sync.Mutex
	changed  []mediaservers.Changed
	sessions []mediaservers.SessionsChanged
	// hub carries the events Health follows.
	hub         *live.Hub
	transitions []Transition
	files       map[works.WorkID][]files.File
	found       []mediaservers.Found
	// placed is the file whose bytes sit at each Rel of each root.
	placed map[queue.At]files.ID
	cfg    mediaservers.Config
	// halt stops the running Media Servers.
	halt   func()
	logged *logged
}

// Schemas are the library schemas the Media Servers read.
func Schemas() []storage.Schema[storage.Library] {
	return []storage.Schema[storage.Library]{
		secret.Schema(), identity.Schema(), mail.Schema(), nodes.Schema(), role.LibrarySchema(), works.Schema(), roots.Schema(), mediaservers.Schema(), health.Schema(),
	}
}

func New(t *testing.T) *World {
	t.Helper()
	w := &World{
		Lib: storagetest.Library(t, Schemas()...), Sealer: secrettest.Sealer(t),
		Home: role.NodeID{UUID: storage.NewUUID()}, Seedbox: role.NodeID{UUID: storage.NewUUID()},
		net: &network{served: map[string]http.RoundTripper{}, down: map[string]stopper{}}, rehomed: make(chan struct{}, 1), files: map[works.WorkID][]files.File{}, placed: map[queue.At]files.ID{}, hub: live.NewHub(hubEvents),
	}
	t.Cleanup(w.hub.Close)
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return errors.Join(nodestest.Insert(ctx, tx, w.Home, "home"), nodestest.Insert(ctx, tx, w.Seedbox, "seedbox"),
			nodestest.Address(ctx, tx, w.Home, 0, HomeIP+":7878"), nodestest.Address(ctx, tx, w.Seedbox, 0, SeedboxIP+":7878"))
	})
	if err != nil {
		t.Fatal(err)
	}
	cfg := rootstest.Config(t, w.Lib, contenttypetest.ID)
	cfg.Describe = func(works.TypeID) (works.Description, bool) { return contenttypetest.Description, true }
	cfg.Changed, cfg.Contents, cfg.Rehomed = rootsChanged{w}, empty{}, rehomed{w}
	if w.roots, err = roots.Open(t.Context(), cfg); err != nil {
		t.Fatal(err)
	}
	w.logged = &logged{}
	log := slog.New(slog.NewMultiHandler(slog.NewTextHandler(t.Output(), nil), slog.NewTextHandler(w.logged, &slog.HandlerOptions{Level: slog.LevelDebug})))
	w.links = newLinks(t, w, mediaserverlink.Config{Kinds: mediaserverlink.Kinds, HTTP: w.net.client(), Timings: Timings, Log: log})
	w.Mail = mailtest.New()
	outbox := mail.NewOutbox(w.Mail, log)
	t.Cleanup(func() { _ = outbox.Close(context.Background()) })
	w.cfg = mediaservers.Config{
		Library: w.Lib, Sealer: w.Sealer, Roster: roster{w.Home, w.Seedbox, w.links}, Kinds: mediaserverlink.Kinds,
		HTTP: w.net.client(), Timings: Timings, PlexTV: plexTV,
		Link:         w.links.service,
		RootWords:    w.roots.WordsIn,
		FileIDs:      w.fileIDs,
		FilePlaces:   w.filePlaces,
		WorkFiles:    w.workFiles,
		RootsChanged: func(context.Context) <-chan struct{} { return w.rehomed },
		Tasks:        &cadences{},
		Events: mediaservers.Events{
			Changed: changedEvents{w}, SessionsChanged: sessionsEvents{w}, Unhealthy: unhealthyEvents{w}, Recovered: recoveredEvents{w},
			ItemsFound: foundEvents{w}, AccessChanged: mediaservers.AccessChangedEvent.Bind(w.hub),
			MyAccessChanged: mediaservers.MyAccessChangedEvent.Bind(w.hub),
		},
		Touch: w.links.touch,
		Facts: w.links.factsOf,
		Zone:  time.UTC,
		Log:   log,
		Admins: func(ctx context.Context) ([]mail.Address, error) {
			return identity.EnabledAdminAddresses(ctx, w.Lib)
		},
		Mail:      w.Mail,
		Outbox:    outbox,
		PlexFacts: plextv.Recorded,
	}
	keys, err := secret.Open(t.Context(), w.Lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	if w.Identity, err = identity.New(t.Context(), identity.Config{Library: w.Lib, Pepper: keys.PINPepper(), Sealer: w.Sealer, Hub: identitytest.Hub(t)}); err != nil {
		t.Fatal(err)
	}
	address, err := identity.ParseAddress("https://media.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Identity.SetAddress(t.Context(), identity.Admin{}, address); err != nil {
		t.Fatal(err)
	}
	w.boot()
	t.Cleanup(func() { w.halt() })
	w.links.start(w.Home)
	w.links.start(w.Seedbox)
	return w
}

// boot makes the Controller's Media Servers over the World's library and
// runs them until halt.
func (w *World) boot() {
	ms := mediaservers.New(w.cfg)
	w.ms, w.Service, w.Renewal = ms, ms.Service(), ms.RenewalTask()
	w.Identity.SetGranter(ms.Granter())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = ms.Run(ctx)
	}()
	w.halt = func() {
		cancel()
		<-done
	}
}

// Restart stops the Controller's Media Servers as a Controller that stops
// does, with whatever is in memory lost, and starts them again over the
// same library. Run any call in flight to its end first.
func (w *World) Restart(t *testing.T) {
	t.Helper()
	w.halt()
	w.boot()
}

// logged is every log line of the World's Nodes, at debug.
type logged struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (l *logged) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf.Write(p)
	return len(p), nil
}

// Logged is every line the World's Media Servers and Media Server links
// logged so far, at debug.
func (w *World) Logged() string {
	w.logged.mu.Lock()
	defer w.logged.mu.Unlock()
	return w.logged.buf.String()
}

// Tap keeps the payload of every event the World publishes from now on.
func (w *World) Tap(ctx context.Context) *live.Tap { return w.hub.Tap(ctx, hubEvents...) }

// Member is a Member's Account and a fresh session of it.
type Member struct {
	ID      identity.AccountID
	Session string
}

// Member adds a Member whose email is name@example.test.
func (w *World) Member(t *testing.T, name string) Member {
	t.Helper()
	id := identitytest.Member(t, w.Lib, name+"@example.test")
	return Member{ID: id, Session: identitytest.Session(t, w.Lib, id)}
}

// MediaServers is the Media Servers, as the API asks a Controller for them.
func (w *World) MediaServers() (mediaservers.Service, bool) { return w.Service, true }

// Jellyfin serves srv at host, such as 192.168.1.20:8096, in place of the
// server there before, with each library folder made in its media dir and
// each of placed, a path under srv.Root, placed before it serves.
func (w *World) Jellyfin(t *testing.T, host string, srv jellyfin.Server, placed ...string) faketest.Served {
	t.Helper()
	return w.mediaBrowser(t, host, jellyfin.Playing(srv), srv, placed)
}

// Emby is Jellyfin for an Emby.
func (w *World) Emby(t *testing.T, host string, srv emby.Server, placed ...string) faketest.Served {
	t.Helper()
	return w.mediaBrowser(t, host, emby.Playing(srv), srv, placed)
}

func (w *World) mediaBrowser(t *testing.T, host string, spec fake.Spec, srv jellyfin.Server, placed []string) faketest.Served {
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
	for _, f := range placed {
		rel, err := filepath.Rel(srv.Root, f)
		at := filepath.Join(media, rel)
		if err == nil {
			err = os.MkdirAll(filepath.Dir(at), 0o700)
		}
		if err == nil {
			err = os.WriteFile(at, []byte(rel), 0o600)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	f := faketest.InMemory(t, spec, scenario.Scenario{Seed: "mstest"}, faketest.WithMedia(media))
	w.net.mu.Lock()
	defer w.net.mu.Unlock()
	w.net.served[host] = f.Client.Transport
	w.net.down[host] = stopper{faults: f.Faults, target: fault.Target(spec.Name)}
	return f
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
	Web:     endpoint.MustParseURL("https://plextv-web.fake.localhost"),
	Auth:    endpoint.MustParseURL("http://plextv-auth.fake.localhost/auth"),
	Desktop: endpoint.MustParseURL("https://app.plex.invalid/desktop"),
}

// Plex is plex.tv signing in to account, which owns srv, with members'
// accounts beside it, and srv served
// at home's and seedbox's addresses on port 32400, in that order in
// resources. Each connection is that Node's front, as in the world, so a
// fault rule's origin names the connection, whoever dials it.
type Plex struct {
	TV, Server faketest.Served
}

func (w *World) Plex(t *testing.T, srv fakeplex.Server, a fakeplextv.Account, members ...fakeplextv.Account) Plex {
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
	p.TV = faketest.InMemory(t, fakeplextv.Playing(a, members...), s, faketest.WithShared(shared), faketest.WithFronts(fronts))
	web := faketest.InMemory(t, fakeplextv.Web, s, faketest.WithShared(shared))
	w.net.mu.Lock()
	defer w.net.mu.Unlock()
	for host, node := range hosts {
		w.net.served[host] = p.Server.From(t, node).Transport
		w.net.down[host] = stopper{faults: p.Server.Faults, target: fault.Target(fakeplex.Fake.Name), origin: node}
	}
	w.net.served[plexTV.API.Hostname()] = p.TV.Client.Transport
	w.net.served[plexTV.Web.Hostname()] = web.Client.Transport
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

// ApproveAs signs username's Plex account in on the sign-in page at
// authURL, as a Member does on plex.tv.
func (p Plex) ApproveAs(t *testing.T, authURL, username string) {
	t.Helper()
	_, fragment, _ := strings.Cut(authURL, "#?")
	q, err := url.ParseQuery(fragment)
	if err != nil {
		t.Fatal(err)
	}
	p.TV.Cue(t, fakeplextv.Approve.Name, map[string]string{"code": q.Get("code"), "account": username})
}

// PlexTVToken is a token of username's Plex account, the Admin's for "",
// signed in on a device of their own.
func (w *World) PlexTVToken(t *testing.T, p Plex, username string) plextv.MemberToken {
	t.Helper()
	m := plextv.NewMember(plexTV, plex.Headers{ClientID: "elsewhere-" + username, Product: "Plex Web"}, w.net.client())
	pin, err := m.CreateLegacyPin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	p.TV.Cue(t, fakeplextv.Approve.Name, map[string]string{"code": pin.Code, "account": username})
	token, ok, err := m.PollLegacyPin(t.Context(), pin.ID)
	if err != nil || !ok {
		t.Fatalf("poll %q's PIN = %v, %v", username, ok, err)
	}
	return token
}

// PlexTVUser is the Plex account token signs in, false once plex.tv no
// longer holds token.
func (w *World) PlexTVUser(t *testing.T, token plextv.MemberToken) (plextv.User, bool) {
	t.Helper()
	m := plextv.NewMember(plexTV, plex.Headers{ClientID: "elsewhere", Product: "Plex Web"}, w.net.client())
	u, err := m.User(t.Context(), token)
	if plextv.Refused(err) {
		return plextv.User{}, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return u, true
}

// MemberPinToken is the token plex.tv gave account's device for its PIN
// pin, read as that device polls it.
func (w *World) MemberPinToken(t *testing.T, account identity.AccountID, pin int64) plextv.MemberToken {
	t.Helper()
	var p db.MediaserversMemberPlex
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		p, err = db.New(r).GetMemberPlex(ctx, account)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	m := plextv.NewMember(plexTV, plex.Headers{ClientID: p.ClientIdentifier, Product: "SlipStream"}, w.net.client())
	token, ok, err := m.PollLegacyPin(t.Context(), pin)
	if err != nil || !ok {
		t.Fatalf("poll PIN %d = %v, %v", pin, ok, err)
	}
	return token
}

// ShareOnPlexTV shares the Plex server machine with username's Plex
// account with every library, as the Admin does in Plex, and answers the
// share.
func (w *World) ShareOnPlexTV(t *testing.T, p Plex, machine, username string) plextv.ShareID {
	t.Helper()
	admin := w.PlexTVToken(t, p, "").Reveal()
	member, _ := w.PlexTVUser(t, w.PlexTVToken(t, p, username))
	c := plextv.New(plexTV, plex.Headers{ClientID: "elsewhere", Product: "Plex Web"}, nil, w.net.client())
	sections, err := c.Sections(t.Context(), admin, machine)
	if err != nil {
		t.Fatal(err)
	}
	share, err := c.Share(t.Context(), admin, machine, member.ID, slices.Collect(maps.Values(sections)))
	if err != nil {
		t.Fatal(err)
	}
	return share
}

// HomeOnPlexTV adds username's Plex account to the Admin's Plex Home, as
// the Admin does in Plex.
func (w *World) HomeOnPlexTV(t *testing.T, p Plex, username string) {
	t.Helper()
	admin := w.PlexTVToken(t, p, "").Reveal()
	member, _ := w.PlexTVUser(t, w.PlexTVToken(t, p, username))
	c := plextv.New(plexTV, plex.Headers{ClientID: "elsewhere", Product: "Plex Web"}, nil, w.net.client())
	if err := c.InviteHome(t.Context(), admin, plextv.Invitee{Who: plextv.Who{ID: member.ID, UUID: member.UUID}, Username: member.Username, Email: member.Email}); err != nil {
		t.Fatal(err)
	}
}

// UsePlexFacts restarts the Media Servers with f in place of what the
// recorded Plex run settled.
func (w *World) UsePlexFacts(t *testing.T, f plextv.Facts) {
	t.Helper()
	w.cfg.PlexFacts = f
	w.Restart(t)
}

// Shared moves access row id to shared on share, as the Admin's Share on
// Plex does when the Member's accept fails.
func (w *World) Shared(t *testing.T, id string, share plextv.ShareID) {
	t.Helper()
	row, err := storage.ParseUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	err = w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		a, err := q.GetAccess(ctx, mediaservers.AccessID{UUID: row})
		if err != nil {
			return err
		}
		_, err = q.MoveAccess(ctx, db.MoveAccessParams{
			ID: a.ID, FromState: access.Waiting, State: access.Shared, ChangedAt: storage.Now(), AccountID: a.AccountID,
			PlexID: a.PlexID, PlexUuid: a.PlexUuid, SharedServerID: &share, ByHand: a.ByHand, PlexHome: a.PlexHome,
		})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
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
	return w.RootOn(t, w.Home, name)
}

// RootOn makes a fresh folder named name a library root on node.
func (w *World) RootOn(t *testing.T, node role.NodeID, name string) roots.Root {
	t.Helper()
	v, err := w.roots.Create(t.Context(), identity.Admin{}, roots.NewRoot{ContentType: contenttypetest.ID, Node: node, Path: folder(t, name)})
	if err != nil {
		t.Fatal(err)
	}
	return v.Root
}

// Rehome moves root to a fresh folder named name on home, as an Admin
// does.
func (w *World) Rehome(t *testing.T, root roots.ID, name string) {
	t.Helper()
	to := folder(t, name)
	if _, err := w.roots.Rehome(t.Context(), identity.Admin{}, root, roots.Rehome{Path: &to}); err != nil {
		t.Fatal(err)
	}
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

func (e changedEvents) Publish(ctx context.Context, v mediaservers.Changed) {
	e.w.mu.Lock()
	e.w.changed = append(e.w.changed, v)
	e.w.mu.Unlock()
	mediaservers.ChangedEvent.Bind(e.w.hub).Publish(ctx, v)
}

type rehomed struct{ w *World }

func (r rehomed) Rehomed(ctx context.Context, tx *storage.Tx[storage.Library], root roots.ID) error {
	return r.w.ms.Rehomed(ctx, tx, root)
}

type rootsChanged struct{ w *World }

func (e rootsChanged) Publish(context.Context, roots.Changed) {
	select {
	case e.w.rehomed <- struct{}{}:
	default:
	}
}

// Place is a new file whose bytes sit at rel in root, as files records
// it.
func (w *World) Place(root roots.ID, rel storing.Rel) files.ID {
	w.mu.Lock()
	defer w.mu.Unlock()
	id := files.ID{UUID: storage.NewUUID()}
	w.placed[queue.At{Root: root, Rel: rel}] = id
	return id
}

func (w *World) fileIDs(_ context.Context, _ storage.Reader[storage.Library], root roots.ID, rels []storing.Rel) (map[storing.Rel]files.ID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := map[storing.Rel]files.ID{}
	for _, rel := range rels {
		if id, ok := w.placed[queue.At{Root: root, Rel: rel}]; ok {
			out[rel] = id
		}
	}
	return out, nil
}

func (w *World) filePlaces(_ context.Context, _ storage.Reader[storage.Library], root roots.ID, fs []files.ID) (map[files.ID]storing.Rel, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := map[files.ID]storing.Rel{}
	for at, id := range w.placed {
		if at.Root == root && slices.Contains(fs, id) {
			out[id] = at.Rel
		}
	}
	return out, nil
}

// Moves is the Controller's Media Servers, for the calls a Move makes.
func (w *World) Moves() *mediaservers.MediaServers { return w.ms }

// Exposure is what server id publishes, as S51 (Reachability diagnostic)
// reads it.
func (w *World) Exposure(ctx context.Context, id link.ServerID) (mediaservers.Exposure, error) {
	return w.ms.Exposure(ctx, id)
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

// network is the fakes served, by host, and how to stop each.
type network struct {
	mu     sync.Mutex
	served map[string]http.RoundTripper
	down   map[string]stopper
}

// stopper resets the connections that reach a fake at one host.
type stopper struct {
	faults *fault.Plan
	target fault.Target
	origin string
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

// roster is home's view of its Nodes: each holding Media Server link, and
// in step until a test takes its link down.
type roster struct {
	self, worker role.NodeID
	links        *links
}

func (r roster) Self() role.NodeID                      { return r.self }
func (r roster) Holding(role.Name) []role.NodeID        { return []role.NodeID{r.self, r.worker} }
func (r roster) Changed() <-chan struct{}               { return r.links.roster() }
func (r roster) InStep(node role.NodeID) bool           { return r.links.inStep(node) }
func (roster) Holds(name role.Name, _ role.NodeID) bool { return name == role.MediaServerLink }

func (r roster) Standing(node role.NodeID) (nodes.Standing, time.Time) {
	return r.links.standing(node)
}

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

// Tell has the Media Servers record cs in one library Write, as files
// does in the Write that records the bytes.
func (w *World) Tell(t *testing.T, cs ...files.PathChange) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.ms.Changed(ctx, tx, cs)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Prepare is files' once-per-pass call before the disk queue moves bytes.
func (w *World) Prepare(t *testing.T, rs ...files.Rename) {
	t.Helper()
	w.ms.Prepare(t.Context(), rs)
}

// Scanned marks every mapping scanned, as the fresh folder rule leaves a
// server that already lists content under its folders.
func (w *World) Scanned(t *testing.T) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).MarkShownScanned(ctx, storage.Now())
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Item is what server's path index holds at p, false for none.
type Item struct {
	Key, Top string
	Stale    bool
}

func (w *World) Item(t *testing.T, server string, p link.ServerPath) (Item, bool) {
	t.Helper()
	u, err := storage.ParseUUID(server)
	if err != nil {
		t.Fatal(err)
	}
	id := link.ServerID{UUID: u}
	var it db.MediaserversItem
	err = w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		it, err = db.New(r).GetItem(ctx, db.GetItemParams{ServerID: id, PathKey: p.Key()})
		return err
	})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return Item{}, false
	case err != nil:
		t.Fatal(err)
	}
	return Item{Key: it.ItemKey, Top: it.TopKey, Stale: it.Stale}, true
}

// Down stops the server at host answering, as a server that stopped
// does: every connection to it resets, an open socket too, until the func
// it returns starts it again.
func (w *World) Down(t *testing.T, host string) (up func()) {
	t.Helper()
	w.net.mu.Lock()
	s := w.net.down[host]
	w.net.mu.Unlock()
	id := "down " + host
	if _, err := s.faults.Set(fault.Rule{ID: id, Target: s.target, Origin: s.origin, Fault: fault.Reset{}}); err != nil {
		t.Fatal(err)
	}
	return func() { s.faults.Clear(id) }
}
