// Package reachabilitytest is a Controller's Remote access over a scripted
// network: each Node answers Read, Resolve and Dial from its Site, the
// Media Servers from a fake, and units run on the real task runner. Run it
// inside a synctest bubble.
package reachabilitytest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	netv1 "github.com/jatassi/slipstream/internal/pb/slipstream/network/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/network/v1/networkv1connect"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reachability/judge/judgetest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

// Identity is what every Media Server answers a player with.
const Identity = "m1"

// Site is how one Node's network answers.
type Site struct {
	Facts   netmodel.NetworkFacts
	A, AAAA []netip.Addr
	// Player is how this Node's connection to the public address ends.
	Player judgetest.Outcome
	// Claims is the Node id its Read answer names, zero for its own.
	Claims role.NodeID
	// Stamp is the observed time its Read answer claims, nil for now.
	Stamp *time.Time
}

// Call is one request the Controller sent a Node.
type Call struct {
	Node   role.NodeID
	Method string
}

// World is atlas, the Controller, with the Nodes a judgetest shape pairs.
type World struct {
	Lib   *storage.DB[storage.Library]
	Reach *reachability.Reachability
	Tasks *tasks.Scheduler
	Admin identity.Admin
	Rules netmodel.Rules
	// Hub carries the events Remote access follows, for a test that
	// changes a record to publish.
	Hub *live.Hub

	mu      sync.Mutex
	sites   map[role.NodeID]Site
	away    map[role.NodeID]bool
	removed map[role.NodeID]bool
	serving []role.NodeID
	custom  []string
	calls   []Call
	wake    chan struct{}
}

var events = []live.Event{
	reachability.ChangedEvent, nodes.ChangedEvent, mediaservers.ChangedEvent, identity.AddressChangedEvent,
	tasks.ChangedEvent, tasks.ProgressEvent,
}

// Config is what a test chooses for a World: the Node's timings and the
// task runner's draw.
type Config struct {
	Timings reachability.Timings
	Rand    *rand.Rand
}

// New is a World with each Node of b's facts paired and answering as its
// Site. atlas is the Controller.
func New(t *testing.T, b *judgetest.B, cfg Config) *World {
	t.Helper()
	f := b.Facts()
	w := &World{
		Lib:     storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), works.Schema(), roots.Schema(), mediaservers.Schema(), tasks.Schema(), reachability.Schema()),
		Rules:   f.Rules,
		Hub:     live.NewHub(events),
		sites:   map[role.NodeID]Site{},
		away:    map[role.NodeID]bool{},
		removed: map[role.NodeID]bool{},
		wake:    make(chan struct{}),
	}
	t.Cleanup(w.Hub.Close)
	if f.Server != nil {
		w.custom = f.Server.Custom
	}
	for id, nf := range f.Nodes {
		s := Site{Facts: nf}
		for _, d := range f.DNS {
			if d.By == id {
				s.A, s.AAAA = d.A, d.AAAA
			}
		}
		w.sites[id] = s
	}
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, id := range []role.NodeID{judgetest.Atlas, judgetest.Den, judgetest.Seedbox, judgetest.VPS} {
			if _, ok := w.sites[id]; ok || id == judgetest.Atlas {
				if err := nodestest.Insert(ctx, tx, id, judgetest.Names[id]); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var scheduler *tasks.Scheduler
	w.Reach = reachability.New(reachability.Config{
		Library: w.Lib, Hub: w.Hub, Log: slog.New(slog.NewTextHandler(t.Output(), nil)), Roster: roster{w},
		Tasks: late{&scheduler},
		Network: func(id role.NodeID) networkv1connect.NodeNetworkServiceHandler {
			return node{w, id}
		},
		MediaServers: servers{w},
		Rules:        w.Rules,
		LinkSource:   func(_ string, remote netip.AddrPort) netip.Addr { return remote.Addr() },
		WebPort:      func() uint16 { return 8484 },
		Timings:      cfg.Timings,
		Zone:         time.UTC,
		Changed:      reachability.ChangedEvent.Bind(w.Hub),
		Now:          time.Now,
	})
	network := netmodel.Def.Serve(w.Reach.Intent, w.Reach.Apply)
	role.NewControls(role.ControlsConfig{Library: w.Lib}, network)
	w.Reach.Serve(network)
	scheduler, err = tasks.Open(t.Context(), tasks.Config{
		Library: w.Lib, Catalog: []tasks.Task{w.Reach.Task()}, Roster: taskRoster{w}, Zone: time.UTC,
		Rand: cfg.Rand, Hub: w.Hub, Changed: tasks.ChangedEvent.Bind(w.Hub), Progressed: tasks.ProgressEvent.Bind(w.Hub),
		Reconciling: func(context.Context, storage.Reader[storage.Library]) (bool, error) { return false, nil },
		Log:         slog.New(slog.NewTextHandler(t.Output(), nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	w.Tasks = scheduler
	run(t, scheduler.Run)
	run(t, w.Reach.Run)
	return w
}

func run(t *testing.T, fn func(context.Context) error) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = fn(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

// Removed marks node removed, as the roster does once its row is a
// tombstone.
func (w *World) Removed(t *testing.T, id role.NodeID) {
	t.Helper()
	w.mu.Lock()
	w.removed[id] = true
	w.mu.Unlock()
	w.changed()
	nodes.ChangedEvent.Bind(w.Hub).Publish(t.Context(), nodes.Changed{})
}

// Away takes node out of step, or brings it back.
func (w *World) Away(id role.NodeID, away bool) {
	w.mu.Lock()
	w.away[id] = away
	w.mu.Unlock()
	w.changed()
}

// Serves marks the Nodes that serve the UI and portal.
func (w *World) Serves(ids ...role.NodeID) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.serving = ids
}

// Site changes how node answers from now on.
func (w *World) Site(id role.NodeID, fn func(*Site)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	s := w.sites[id]
	fn(&s)
	w.sites[id] = s
}

// Calls is every request sent so far, then forgets them.
func (w *World) Calls() []Call {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := w.calls
	w.calls = nil
	return out
}

// Intent is what node is told to watch now.
func (w *World) Intent(t *testing.T, id role.NodeID) *netv1.Intent {
	t.Helper()
	var in *netv1.Intent
	err := w.Lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		var err error
		in, err = w.Reach.Intent(ctx, rd, id)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return in
}

// Page reads Remote access.
func (w *World) Page(t *testing.T) reachability.Page {
	t.Helper()
	p, err := w.Reach.RemoteAccess(t.Context(), w.Admin)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// Route is the route of subject on the page.
func (w *World) Route(t *testing.T, subject string) reachability.Route {
	t.Helper()
	for _, rt := range w.Page(t).Routes {
		if rt.Subject == subject {
			return rt
		}
	}
	t.Fatalf("no route of %s", subject)
	return reachability.Route{}
}

func (w *World) changed() {
	w.mu.Lock()
	defer w.mu.Unlock()
	close(w.wake)
	w.wake = make(chan struct{})
}

func (w *World) inStep(id role.NodeID) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, paired := w.sites[id]
	return (paired || id == judgetest.Atlas) && !w.away[id] && !w.removed[id]
}

func (w *World) site(id role.NodeID, method string) (Site, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls = append(w.calls, Call{Node: id, Method: method})
	if w.away[id] || w.removed[id] {
		return Site{}, fmt.Errorf("%s is away", judgetest.Names[id])
	}
	return w.sites[id], nil
}

type node struct {
	w  *World
	id role.NodeID
}

func (n node) Read(_ context.Context, _ *netv1.ReadRequest) (*netv1.ReadResponse, error) {
	s, err := n.w.site(n.id, "Read")
	if err != nil {
		return nil, err
	}
	nf := s.Facts
	nf.At = time.Now()
	if s.Stamp != nil {
		nf.At = *s.Stamp
	}
	if !s.Claims.IsZero() {
		nf.Node = s.Claims
	}
	return &netv1.ReadResponse{Facts: netmodel.FactsOnWire(nf)}, nil
}

func (n node) Resolve(_ context.Context, _ *netv1.ResolveRequest) (*netv1.ResolveResponse, error) {
	s, err := n.w.site(n.id, "Resolve")
	if err != nil {
		return nil, err
	}
	resp := &netv1.ResolveResponse{}
	for _, a := range s.A {
		resp.A = append(resp.A, a.AsSlice())
	}
	for _, a := range s.AAAA {
		resp.Aaaa = append(resp.Aaaa, a.AsSlice())
	}
	return resp, nil
}

// Dial answers as the Site's Player outcome. A connection to the UI and
// portal fetches the echo from the tester's public address.
func (n node) Dial(ctx context.Context, req *netv1.DialRequest) (*netv1.DialResponse, error) {
	s, err := n.w.site(n.id, "Dial")
	if err != nil {
		return nil, err
	}
	addrs := s.A
	if req.GetFamily() == netmodel.FamilyOnWire(netmodel.IPv6) {
		addrs = s.AAAA
	}
	if a := req.GetAddr(); a != nil {
		x, _ := netip.AddrFromSlice(a)
		addrs = []netip.Addr{x}
	}
	resp := &netv1.DialResponse{}
	for _, a := range addrs {
		resp.Attempts = append(resp.Attempts, n.attempt(ctx, s, req, a))
	}
	return resp, nil
}

// attempt is one address's connection ending as the Site's Player says.
// Only a connection that reads the page fetches the echo.
func (n node) attempt(ctx context.Context, s Site, req *netv1.DialRequest, a netip.Addr) *netv1.Attempt {
	at := &netv1.Attempt{Addr: a.AsSlice(), Stage: netv1.Stage_STAGE_HTTP, Status: 200, Body: []byte(`{"MediaContainer":{"machineIdentifier":"` + Identity + `"}}`)}
	if req.GetScheme() == netv1.Scheme_SCHEME_HTTPS {
		name := req.GetHost()
		if name == "" {
			name = req.GetSni()
		}
		at.Tls = &netv1.Certificate{Names: []string{name}, NotAfterMs: judgetest.Expires.AddDate(1, 0, 0).UnixMilli()}
	}
	unread := func(stage netv1.Stage) *netv1.Attempt {
		at.Stage, at.Status, at.Body = stage, 0, nil
		return at
	}
	switch s.Player {
	case judgetest.Passed:
	case judgetest.Closed:
		at.Tls = nil
		return unread(netv1.Stage_STAGE_RESOLVED)
	case judgetest.Expired:
		at.Tls.Problem, at.Tls.NotAfterMs = netv1.TLSProblem_TLS_PROBLEM_EXPIRED, judgetest.Expires.UnixMilli()
		return unread(netv1.Stage_STAGE_TLS)
	case judgetest.WrongName:
		at.Tls.Problem, at.Tls.Names = netv1.TLSProblem_TLS_PROBLEM_WRONG_NAME, []string{"other.example.org"}
		return unread(netv1.Stage_STAGE_TLS)
	case judgetest.NoResponse:
		return unread(netv1.Stage_STAGE_TLS)
	case judgetest.SignInPage:
		at.Status = 401
	case judgetest.Something:
		at.Body = []byte("<html>")
	}
	if id, ok := strings.CutPrefix(req.GetPath(), reachability.EchoPath); ok && s.Player == judgetest.Passed {
		answer, _ := n.w.Reach.Echo(ctx, id, api.Arrival{Source: n.w.Rules.Public4(s.Facts.Observed())})
		at.Body = []byte(answer)
	}
	return at
}

type roster struct{ w *World }

func (r roster) Self() role.NodeID          { return judgetest.Atlas }
func (r roster) InStep(id role.NodeID) bool { return r.w.inStep(id) }
func (r roster) Serving(role.Name) []role.NodeID {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	return r.w.serving
}

func (r roster) Remote(id role.NodeID) (netip.AddrPort, bool, bool) {
	if id == judgetest.Atlas || !r.w.inStep(id) {
		return netip.AddrPort{}, false, false
	}
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	return netip.AddrPortFrom(r.w.Rules.Public4(r.w.sites[id].Facts.Observed()), 7878), false, true
}

func (r roster) Names(context.Context) (map[role.NodeID]nodes.Named, error) {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	out := map[role.NodeID]nodes.Named{judgetest.Atlas: {ID: judgetest.Atlas, Name: "atlas"}}
	for id := range r.w.sites {
		out[id] = nodes.Named{ID: id, Name: judgetest.Names[id], Removed: r.w.removed[id]}
	}
	return out, nil
}

type taskRoster struct{ w *World }

func (r taskRoster) Self() role.NodeID { return judgetest.Atlas }

func (r taskRoster) Standing(id role.NodeID) (nodes.Standing, time.Time) {
	if r.w.inStep(id) {
		return nodes.InStep, time.Time{}
	}
	return nodes.Offline, time.Time{}
}

func (r taskRoster) Route(_ role.Name, home role.NodeID) (role.NodeID, error) { return home, nil }

func (r taskRoster) Get(_ context.Context, id role.NodeID) (nodes.Node, error) {
	return nodes.Node{ID: id, Name: judgetest.Names[id], Self: id == judgetest.Atlas}, nil
}

func (r taskRoster) Names(ctx context.Context) (map[role.NodeID]nodes.Named, error) {
	return roster(r).Names(ctx)
}

func (r taskRoster) Changed() <-chan struct{} {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	return r.w.wake
}

type servers struct{ w *World }

// Exposure answers each Plex's facts as the shape publishes them, with
// remote access on.
func (s servers) Exposure(ctx context.Context, id link.ServerID) (mediaservers.Exposure, error) {
	var x mediaservers.Exposure
	err := s.w.Lib.Read(ctx, func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		recs, err := mediaservers.ServersIn(ctx, rd)
		if err != nil {
			return err
		}
		for _, rec := range recs {
			if rec.ID == id {
				x = mediaservers.Exposure{Server: id, Kind: rec.Kind, Name: rec.Name, Reaching: rec.Reaching, RunsOn: rec.RunsOn, At: time.Now()}
				return nil
			}
		}
		return errors.New("no such Media Server")
	})
	if err != nil {
		return x, err
	}
	if s.w.inStep(x.Reaching) {
		x.Exposed = &link.Exposed{Local: netip.AddrPortFrom(judgetest.AtlasLAN, 32400), Remote: link.Yes, Custom: s.w.custom}
	}
	return x, nil
}

type late struct{ s **tasks.Scheduler }

func (l late) Start(ctx context.Context, st tasks.Start) (tasks.Started, error) {
	return (*l.s).Start(ctx, st)
}
