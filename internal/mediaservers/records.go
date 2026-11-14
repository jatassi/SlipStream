package mediaservers

import (
	"context"
	"errors"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	"github.com/jatassi/slipstream/internal/nodes"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// Service is what the API calls.
func (m *MediaServers) Service() Service { return service{m: m} }

// service builds every operation but those Unbuilt still stands in for.
type service struct {
	Unbuilt
	m *MediaServers
}

func (s service) Settings() setting.Panel { return s.m.Settings() }

// route is one Node's view of a server: who calls it, where, with what.
type route struct {
	kind    link.Kind
	label   string
	node    role.NodeID
	name    string
	address string
	cred    link.Credential
	device  string
	// server names the server in copy: its name, or for a draft the kind
	// and address.
	server string
}

// reach reads the identity at r's address from r's Node with no
// credential, and refuses a version below the kind's floor.
func (m *MediaServers) reach(ctx context.Context, r route) (link.Identity, error) {
	id, err := m.reachFrom(ctx, r)
	if err != nil {
		m.cfg.Log.Info("a Media Server didn't respond to a test", "server", r.server, "from", r.name, "err", err.Error())
		return id, failure(err, r)
	}
	return id, floored(r, id)
}

// floored refuses a version below the kind's floor.
func floored(r route, id link.Identity) error {
	if floor := r.kind.About().Floor; id.Version.Compare(floor) < 0 {
		return &TooOldError{Label: r.label, Found: id.Version.String(), Floor: floor.String()}
	}
	return nil
}

func (m *MediaServers) reachFrom(ctx context.Context, r route) (link.Identity, error) {
	about := r.kind.About()
	resp, err := m.cfg.Link(r.node).Reach(ctx, &msv1.ReachRequest{Kind: string(about.Name), Address: r.address})
	if err != nil {
		return link.Identity{}, err
	}
	if err := mediaserverlink.ServerErrorOf(resp.GetProblem()); err != nil {
		return link.Identity{}, err
	}
	return mediaserverlink.IdentityOf(resp.GetIdentity())
}

// check proves r's credential from r's Node, the credential in the call,
// and refuses another server than identity when identity is set.
func (m *MediaServers) check(ctx context.Context, r route, identity string) (link.Checked, error) {
	c, err := m.checkFrom(ctx, r)
	switch {
	case err != nil:
		m.cfg.Log.Info("a Media Server failed a test", "server", r.server, "from", r.name, "err", err.Error())
		return c, failure(err, r)
	case identity != "" && c.Identity.ID != identity:
		return c, &OtherServerError{Label: r.label, Address: r.address}
	}
	return c, nil
}

func (m *MediaServers) checkFrom(ctx context.Context, r route) (link.Checked, error) {
	about := r.kind.About()
	resp, err := m.cfg.Link(r.node).Check(ctx, &msv1.CheckRequest{
		Kind: string(about.Name), Address: r.address, Credential: mediaserverlink.CredentialOnWire(r.cred), DeviceId: r.device,
	})
	if err != nil {
		return link.Checked{}, err
	}
	return mediaserverlink.CheckedOf(resp)
}

// direct is r's server as the Controller reaches it in process, at its
// fallback address.
func (m *MediaServers) direct(r route, identity string) link.Conn {
	ref := link.ServerRef{Kind: r.kind.About().Name, Identity: identity}
	return r.kind.Dial(link.Dialed{Server: ref, Address: r.address, Credential: r.cred, DeviceID: r.device, HTTP: m.cfg.HTTP, Timings: m.cfg.Timings})
}

// tested is both route tests of a server.
type tested struct {
	identity link.Identity
	checked  link.Checked
	reaching error
	// controller is the Controller's own test at the fallback address,
	// which decides whether a fallback is available.
	controller error
}

func (t tested) fallback() string {
	if t.controller == nil {
		return "available"
	}
	return "none"
}

// test reaches and checks the server from its reaching Node, then checks
// it from the Controller at fallback. identity is the saved server's, ""
// for a draft.
func (m *MediaServers) test(ctx context.Context, r, fallback route, identity string) tested {
	var t tested
	if t.identity, t.reaching = m.reach(ctx, r); t.reaching == nil {
		if identity != "" && t.identity.ID != identity {
			t.reaching = &OtherServerError{Label: r.label, Address: r.address}
		} else {
			t.checked, t.reaching = m.check(ctx, r, t.identity.ID)
		}
	}
	want := cmpOr(identity, t.identity.ID)
	if want == "" {
		t.controller = t.reaching
		return t
	}
	if _, err := m.direct(fallback, want).Check(ctx); err != nil {
		t.controller = failure(err, fallback)
	}
	return t
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// isLocalhost is whether address names the machine it is called from.
func isLocalhost(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// names is every Node's name by id, the removed ones flagged.
func (m *MediaServers) names(ctx context.Context, r storage.Reader[storage.Library]) (map[role.NodeID]nodes.Named, error) {
	all, err := nodes.Names(ctx, r)
	if err != nil {
		return nil, err
	}
	out := make(map[role.NodeID]nodes.Named, len(all))
	for _, n := range all {
		out[n.ID] = n
	}
	return out, nil
}

// placed checks a server's Nodes: each exists, and reaching holds Media
// Server link.
func (m *MediaServers) placed(ctx context.Context, reaching role.NodeID, runsOn *role.NodeID) (map[role.NodeID]nodes.Named, error) {
	var names map[role.NodeID]nodes.Named
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		names, err = m.names(ctx, r)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, n := range []*role.NodeID{&reaching, runsOn} {
		if n != nil {
			if named, ok := names[*n]; !ok || named.Removed {
				return nil, ErrNodeNotFound
			}
		}
	}
	if !m.cfg.Roster.Holds(role.MediaServerLink, reaching) {
		return nil, &NodeNotLinkedError{Node: names[reaching].Name}
	}
	return names, nil
}

// localOnWorker refuses a localhost address on a Worker with no fallback
// address, since the Controller can't use it.
func (m *MediaServers) localOnWorker(names map[role.NodeID]nodes.Named, reaching role.NodeID, address, fallback string) error {
	if reaching != m.cfg.Roster.Self() && fallback == "" && isLocalhost(address) {
		return &LocalhostOnWorkerError{Node: names[reaching].Name, Address: address}
	}
	return nil
}

// routes are a server's route from its reaching Node and from the
// Controller at its fallback address.
func (m *MediaServers) routes(k link.Kind, names map[role.NodeID]nodes.Named, reaching role.NodeID, address, fallback string, cred link.Credential, device, server string) (fromNode, fromController route) {
	label := k.About().Label
	self := m.cfg.Roster.Self()
	r := route{kind: k, label: label, node: reaching, name: names[reaching].Name, address: address, cred: cred, device: device, server: server}
	f := r
	f.node, f.name, f.address = self, names[self].Name, cmpOr(fallback, address)
	return r, f
}

// recordFetch records that node got server id's credential, unless node
// is the Controller, which holds every credential.
func (m *MediaServers) recordFetch(ctx context.Context, q *db.Queries, id link.ServerID, node role.NodeID, version int64) error {
	if node == m.cfg.Roster.Self() {
		return nil
	}
	return q.RecordFetch(ctx, db.RecordFetchParams{ServerID: id, NodeID: node, SettingsVersion: version, FetchedAt: storage.Now()})
}

func (m *MediaServers) changed(ctx context.Context, ids ...link.ServerID) {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	m.cfg.Events.Changed.Publish(ctx, Changed{ServerIDs: out})
	m.senders.wake(ids...)
}

// touchAll follows a change to every intent.
func (m *MediaServers) touchAll(ctx context.Context) {
	var reaching []role.NodeID
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		reaching, err = db.New(r).ListReachingNodes(ctx)
		return err
	})
	if err != nil {
		m.cfg.Log.Error("list the reaching Nodes to send them new intents", "err", err.Error())
		return
	}
	m.cfg.Touch(append(reaching, m.cfg.Roster.Self())...)
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func delayMs(k link.Kind, d time.Duration) *int64 {
	if k.About().Form == link.SignInPlexPIN {
		return nil
	}
	ms := d.Milliseconds()
	return &ms
}

func nodeOrZero(n *role.NodeID) role.NodeID {
	if n == nil {
		return role.NodeID{}
	}
	return *n
}

// nameTaken is whether a server other than except has name.
func nameTaken(ctx context.Context, q *db.Queries, name string, except link.ServerID) (bool, error) {
	all, err := q.ListServers(ctx)
	return slices.ContainsFunc(all, func(s db.MediaserversServer) bool { return s.Name == name && s.ID != except }), err
}

// Add tests a draft from its reaching Node and from the Controller,
// refuses one older than its kind's floor, and saves it with whether a
// fallback is available. A Plex draft takes its token and its addresses
// from the Plex account. The reaching Node held the credential for its
// test, so it is recorded as a holder.
func (s service) Add(ctx context.Context, d Draft) (ServerView, error) {
	k, ok := s.m.cfg.Kinds.Of(d.Kind)
	if !ok {
		return ServerView{}, ErrKindUnknown
	}
	names, err := s.m.placed(ctx, d.Reaching, d.RunsOn)
	if err != nil {
		return ServerView{}, err
	}
	err = s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		taken, err := nameTaken(ctx, db.New(r), d.Name, link.ServerID{})
		if taken {
			return ErrNameTaken
		}
		return err
	})
	if err != nil {
		return ServerView{}, err
	}
	var (
		device, identity string
		deviceColumn     *string
		cred             link.Credential
	)
	if k.About().Form == link.SignInPlexPIN {
		var a account
		if d, a, cred, err = s.m.plexDraft(ctx, d, names); err != nil {
			return ServerView{}, err
		}
		device, identity = a.clientID, d.PlexServer
	} else {
		device, cred = cmpOr(d.DeviceID, storage.NewUUID().String()), link.Credential{APIKey: d.APIKey}
		deviceColumn = &device
	}
	if err := s.m.localOnWorker(names, d.Reaching, d.Address, d.FallbackAddress); err != nil {
		return ServerView{}, err
	}
	r, f := s.m.routes(k, names, d.Reaching, d.Address, d.FallbackAddress, cred, device, k.About().Label+" at "+d.Address)
	t := s.m.test(ctx, r, f, identity)
	if t.reaching != nil {
		return ServerView{}, t.reaching
	}
	id := link.ServerID{UUID: storage.NewUUID()}
	sealed, err := sealCredential(s.m.cfg.Sealer, id, cred)
	if err != nil {
		return ServerView{}, err
	}
	err = s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		now := storage.Now()
		err := q.InsertServer(ctx, db.InsertServerParams{
			ID: id, Kind: d.Kind, Name: d.Name, Address: d.Address, FallbackAddress: optional(d.FallbackAddress),
			BrowserAddress: optional(d.BrowserAddress), Identity: t.identity.ID, Version: t.identity.Version.String(),
			MonitorDelayMs: delayMs(k, t.checked.MonitorDelay), ReachingNodeID: d.Reaching, RunsOnNodeID: nodeOrZero(d.RunsOn),
			DeviceID: deviceColumn, SecretsSealed: sealed, Fallback: t.fallback(), CheckedAt: now, At: now,
		})
		if err != nil {
			return err
		}
		return s.m.recordFetch(ctx, q, id, d.Reaching, 1)
	})
	if err != nil {
		return ServerView{}, err
	}
	s.m.cfg.Touch(d.Reaching)
	s.m.changed(ctx, id)
	return s.Get(ctx, id)
}

// saved is a server row with its kind and credential.
type saved struct {
	row  db.MediaserversServer
	kind link.Kind
	cred link.Credential
	// device is the DeviceId the server knows SlipStream by: the Plex
	// account's client identifier for Plex.
	device string
}

func (m *MediaServers) load(ctx context.Context, r storage.Reader[storage.Library], id link.ServerID) (saved, error) {
	row, err := db.New(r).GetServer(ctx, id)
	if errors.Is(err, storage.ErrNoRows) {
		return saved{}, ErrNotFound
	}
	if err != nil {
		return saved{}, err
	}
	k, _ := m.cfg.Kinds.Of(row.Kind)
	cred, err := unsealCredential(m.cfg.Sealer, id, row.SecretsSealed)
	if err != nil {
		return saved{}, err
	}
	device, err := deviceOf(ctx, db.New(r), row)
	return saved{row: row, kind: k, cred: cred, device: device}, err
}

// deviceOf is the DeviceId row's server knows SlipStream by.
func deviceOf(ctx context.Context, q *db.Queries, row db.MediaserversServer) (string, error) {
	if row.DeviceID != nil {
		return *row.DeviceID, nil
	}
	a, err := q.GetPlexAccount(ctx)
	return a.ClientIdentifier, err
}

// loaded is server id and every Node's name, in one Read.
func (m *MediaServers) loaded(ctx context.Context, id link.ServerID) (saved, map[role.NodeID]nodes.Named, error) {
	var (
		sv    saved
		names map[role.NodeID]nodes.Named
	)
	err := m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		if sv, err = m.load(ctx, r, id); err != nil {
			return err
		}
		names, err = m.names(ctx, r)
		return err
	})
	return sv, names, err
}

func (m *MediaServers) routesOf(sv saved, names map[role.NodeID]nodes.Named) (fromNode, fromController route) {
	return m.routes(sv.kind, names, sv.row.ReachingNodeID, sv.row.Address, deref(sv.row.FallbackAddress), sv.cred, sv.device, sv.row.Name)
}

// Update changes a saved server. A change to its address, fallback
// address, reaching Node or API key tests it again first, as Add does,
// and the server must still be the one saved. A change the reaching Node
// reads moves the settings version, so it fetches the credential again.
func (s service) Update(ctx context.Context, id link.ServerID, p Patch) (ServerView, error) {
	sv, _, err := s.m.loaded(ctx, id)
	if err != nil {
		return ServerView{}, err
	}
	row := sv.row
	next, cred := p.onto(row, sv.cred)
	runsOn := &next.RunsOnNodeID
	if runsOn.IsZero() {
		runsOn = nil
	}
	names, err := s.m.placed(ctx, next.ReachingNodeID, runsOn)
	if err != nil {
		return ServerView{}, err
	}
	if err := s.m.localOnWorker(names, next.ReachingNodeID, next.Address, deref(next.FallbackAddress)); err != nil {
		return ServerView{}, err
	}
	routed := next.Address != row.Address || deref(next.FallbackAddress) != deref(row.FallbackAddress) ||
		next.ReachingNodeID != row.ReachingNodeID || cred != sv.cred
	if routed {
		r, f := s.m.routes(sv.kind, names, next.ReachingNodeID, next.Address, deref(next.FallbackAddress), cred, sv.device, next.Name)
		t := s.m.test(ctx, r, f, row.Identity)
		if t.reaching != nil {
			return ServerView{}, t.reaching
		}
		next.Version, next.MonitorDelayMs = t.identity.Version.String(), delayMs(sv.kind, t.checked.MonitorDelay)
		next.Fallback, next.CheckedAt = t.fallback(), storage.Now()
		next.SettingsVersion++
	}
	sealed := row.SecretsSealed
	if cred != sv.cred {
		if sealed, err = sealCredential(s.m.cfg.Sealer, id, cred); err != nil {
			return ServerView{}, err
		}
	}
	err = s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		taken, err := nameTaken(ctx, q, next.Name, id)
		switch {
		case err != nil:
			return err
		case taken:
			return ErrNameTaken
		}
		err = q.UpdateServer(ctx, db.UpdateServerParams{
			ID: id, Name: next.Name, Address: next.Address, FallbackAddress: next.FallbackAddress, BrowserAddress: next.BrowserAddress,
			Identity: next.Identity, Version: next.Version, MonitorDelayMs: next.MonitorDelayMs, ReachingNodeID: next.ReachingNodeID,
			RunsOnNodeID: next.RunsOnNodeID, SecretsSealed: sealed, SettingsVersion: next.SettingsVersion, Fallback: next.Fallback,
			CheckedAt: next.CheckedAt, At: storage.Now(),
		})
		if err != nil || !routed {
			return err
		}
		return s.m.recordFetch(ctx, q, id, next.ReachingNodeID, next.SettingsVersion)
	})
	if err != nil {
		return ServerView{}, err
	}
	s.m.cfg.Touch(row.ReachingNodeID, next.ReachingNodeID)
	s.m.changed(ctx, id)
	return s.Get(ctx, id)
}

// onto is row and cred with the patch's fields.
func (p Patch) onto(row db.MediaserversServer, cred link.Credential) (db.MediaserversServer, link.Credential) {
	if p.Name != nil {
		row.Name = *p.Name
	}
	if p.Address != nil {
		row.Address = *p.Address
	}
	if p.FallbackAddress != nil {
		row.FallbackAddress = p.FallbackAddress.To
	}
	if p.BrowserAddress != nil {
		row.BrowserAddress = p.BrowserAddress.To
	}
	if p.Reaching != nil {
		row.ReachingNodeID = *p.Reaching
	}
	if p.RunsOn != nil {
		row.RunsOnNodeID = nodeOrZero(p.RunsOn.To)
	}
	if p.APIKey != nil {
		cred.APIKey = *p.APIKey
	}
	return row, cred
}

// Remove forgets the server with its changes, mappings, index, play
// history and credential holders.
func (s service) Remove(ctx context.Context, id link.ServerID) error {
	var reaching role.NodeID
	err := s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		row, err := q.GetServer(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		reaching = row.ReachingNodeID
		_, err = q.DeleteServer(ctx, id)
		return err
	})
	if err != nil {
		return err
	}
	s.m.cfg.Touch(reaching)
	s.m.changed(ctx, id)
	return nil
}

// Test runs both route tests and stores whether a fallback is available.
func (s service) Test(ctx context.Context, id link.ServerID) (Tested, error) {
	sv, names, err := s.m.loaded(ctx, id)
	if err != nil {
		return Tested{}, err
	}
	r, f := s.m.routesOf(sv, names)
	t := s.m.test(ctx, r, f, sv.row.Identity)
	version, delay := sv.row.Version, sv.row.MonitorDelayMs
	if t.reaching == nil {
		version, delay = t.identity.Version.String(), delayMs(sv.kind, t.checked.MonitorDelay)
	}
	err = s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		if err := q.SetTested(ctx, db.SetTestedParams{ID: id, Version: version, MonitorDelayMs: delay, Fallback: t.fallback(), CheckedAt: storage.Now()}); err != nil {
			return err
		}
		if t.reaching == nil {
			return s.m.recordFetch(ctx, q, id, sv.row.ReachingNodeID, sv.row.SettingsVersion)
		}
		return nil
	})
	if err != nil {
		return Tested{}, err
	}
	s.m.changed(ctx, id)
	return Tested{Reaching: routeTest(r, t.reaching), Controller: routeTest(f, t.controller), Fallback: t.fallback()}, nil
}

func routeTest(r route, err error) RouteTest {
	if err == nil {
		return RouteTest{OK: true, Words: r.name + " reached " + r.server + "."}
	}
	w, _ := Words(err)
	return RouteTest{Words: w}
}

// ConfirmIdentity accepts the server that responds at the address now as
// the saved one, once it passes the floor and accepts the credential.
func (s service) ConfirmIdentity(ctx context.Context, id link.ServerID) (ServerView, error) {
	sv, names, err := s.m.loaded(ctx, id)
	if err != nil {
		return ServerView{}, err
	}
	r, _ := s.m.routesOf(sv, names)
	found, err := s.m.reach(ctx, r)
	if err != nil {
		return ServerView{}, err
	}
	c, err := s.m.check(ctx, r, found.ID)
	if err != nil {
		return ServerView{}, err
	}
	row := sv.row
	err = s.m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		err := q.UpdateServer(ctx, db.UpdateServerParams{
			ID: id, Name: row.Name, Address: row.Address, FallbackAddress: row.FallbackAddress, BrowserAddress: row.BrowserAddress,
			Identity: found.ID, Version: found.Version.String(), MonitorDelayMs: delayMs(sv.kind, c.MonitorDelay),
			ReachingNodeID: row.ReachingNodeID, RunsOnNodeID: row.RunsOnNodeID, SecretsSealed: row.SecretsSealed,
			SettingsVersion: row.SettingsVersion + 1, Fallback: row.Fallback, CheckedAt: row.CheckedAt, At: storage.Now(),
		})
		if err != nil {
			return err
		}
		return s.m.recordFetch(ctx, q, id, row.ReachingNodeID, row.SettingsVersion+1)
	})
	if err != nil {
		return ServerView{}, err
	}
	s.m.cfg.Touch(row.ReachingNodeID)
	s.m.changed(ctx, id)
	return s.Get(ctx, id)
}

// SignIn signs in once as a Jellyfin or Emby administrator and creates a
// key named SlipStream. The Controller signs in when it reaches the
// address. Otherwise the Node does, and the password is in that one call.
func (s service) SignIn(ctx context.Context, d SignInDraft) (SignedIn, error) {
	k, ok := s.m.cfg.Kinds.Of(d.Kind)
	if !ok || k.About().Form != link.SignInKeyOrAdmin {
		return SignedIn{}, ErrKindUnknown
	}
	names, err := s.m.placed(ctx, d.Node, nil)
	if err != nil {
		return SignedIn{}, err
	}
	device := storage.NewUUID().String()
	server := k.About().Label + " at " + d.Address
	r, f := s.m.routes(k, names, d.Node, d.Address, "", link.Credential{}, device, server)
	if id, err := s.m.direct(f, "").Reach(ctx); err == nil {
		if err := floored(f, id); err != nil {
			return SignedIn{}, err
		}
		cred, err := s.m.direct(f, "").SignIn(ctx, d.User, d.Password)
		if err != nil {
			return SignedIn{}, signInFailure(err, f)
		}
		return SignedIn{APIKey: cred.APIKey, DeviceID: device}, nil
	}
	if _, err := s.m.reach(ctx, r); err != nil {
		return SignedIn{}, err
	}
	resp, err := s.m.cfg.Link(d.Node).SignIn(ctx, &msv1.SignInRequest{
		Kind: string(d.Kind), Address: d.Address, DeviceId: device, User: d.User, Password: d.Password,
	})
	if err == nil {
		err = mediaserverlink.ServerErrorOf(resp.GetProblem())
	}
	if err != nil {
		return SignedIn{}, signInFailure(err, r)
	}
	return SignedIn{APIKey: mediaserverlink.CredentialOf(resp.GetCredential()).APIKey, DeviceID: device}, nil
}

func signInFailure(err error, r route) error {
	if p, ok := link.ServerProblem(err); ok && p.Reason == link.ReasonRefused {
		return &SignInRefusedError{Server: r.server, With: RefusedPassword}
	}
	return failure(err, r)
}

// CredentialHolders is every Node that fetched the server's credential,
// newest first.
func (s service) CredentialHolders(ctx context.Context, id link.ServerID) ([]HolderView, error) {
	out := []HolderView{}
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		q := db.New(r)
		row, err := q.GetServer(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		names, err := s.m.names(ctx, r)
		if err != nil {
			return err
		}
		holders, err := q.ListHolders(ctx, id)
		for _, h := range holders {
			out = append(out, HolderView{
				Node: NodeRef{ID: h.NodeID.String(), Name: names[h.NodeID].Name}, FetchedAt: h.FetchedAt.Time(), Current: h.NodeID == row.ReachingNodeID,
			})
		}
		return err
	})
	return out, err
}
