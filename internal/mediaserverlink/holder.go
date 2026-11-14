package mediaserverlink

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/mediaserverlink/internal/sessions"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Holder is the Role's holder half as Run sees it.
type Holder = role.Holder[*msv1.Intent, *msv1.Facts, *msv1.Report]

// Control is the part of MediaServerControlService the holder calls.
type Control interface {
	MediaServerSecrets(ctx context.Context, req *msv1.MediaServerSecretsRequest) (*msv1.MediaServerSecretsResponse, error)
}

type Config struct {
	Kinds   link.Registry
	HTTP    *http.Client
	Timings link.Timings
	Log     *slog.Logger
}

// Link is the holder half on one Node: one watch per server of its intent,
// and each server's credential, kept in memory only.
type Link struct {
	cfg     Config
	mu      sync.Mutex
	control Control
	servers map[link.ServerID]target
	secrets map[link.ServerID]held
	// plays outlive a watch started again, so its sessions keep their
	// plays.
	plays map[link.ServerID]sessions.Diff
	facts map[link.ServerID]*msv1.ServerFacts
	sent  *msv1.Facts
}

// held is a credential and the settings version the Controller said it
// belongs to.
type held struct {
	cred    link.Credential
	version int64
}

func NewLink(cfg Config) *Link {
	l := &Link{cfg: cfg}
	l.forget()
	return l
}

// target is one server of the intent, as a watch dials it.
type target struct {
	ref          link.ServerRef
	name         string
	address      string
	deviceID     string
	monitorDelay time.Duration
	every        time.Duration
}

func targetsOf(kinds link.Registry, in *msv1.Intent) (map[link.ServerID]target, error) {
	out := map[link.ServerID]target{}
	for _, s := range in.GetServers() {
		u, err := storage.UUIDFromBytes(s.GetId())
		if err != nil {
			return nil, fmt.Errorf("read a Media Server id of the intent: %w", err)
		}
		kind := link.KindName(s.GetKind())
		if _, ok := kinds.Of(kind); !ok {
			return nil, fmt.Errorf("the intent's Media Server %s is of kind %q, which this Node doesn't have", s.GetName(), kind)
		}
		id := link.ServerID{UUID: u}
		out[id] = target{
			ref:      link.ServerRef{ID: id, Kind: kind, Identity: s.GetIdentity(), SettingsVersion: s.GetSettingsVersion()},
			name:     s.GetName(),
			address:  s.GetAddress(),
			deviceID: s.GetDeviceId(),

			monitorDelay: time.Duration(s.GetMonitorDelayMs()) * time.Millisecond,
			every:        time.Duration(in.GetSessionsEveryMs()) * time.Millisecond,
		}
	}
	return out, nil
}

// Run is the Def.Hold run: it keeps one watch per server of the intent,
// started again when the server's address, identity, credential or
// cadence changes and stopped when the server leaves the intent.
func (l *Link) Run(local *storage.DB[storage.Local], control Control) func(context.Context, Holder) error {
	return func(ctx context.Context, h Holder) error {
		l.mu.Lock()
		l.control = control
		l.mu.Unlock()
		watches := map[link.ServerID]*running{}
		defer func() {
			for _, w := range watches {
				w.stop()
			}
		}()
		for s := range h.Intent(ctx) {
			ts, err := targetsOf(l.cfg.Kinds, s.Intent)
			if err != nil {
				return err
			}
			l.follow(h, ts)
			l.reconcile(ctx, watches, ts, func(t target) *watch { return &watch{l: l, local: local, h: h, t: t} })
		}
		return nil
	}
}

// reconcile stops each watch whose server left ts or changed, and starts
// one for each server of ts with none.
func (l *Link) reconcile(ctx context.Context, watches map[link.ServerID]*running, ts map[link.ServerID]target, newWatch func(target) *watch) {
	for id, w := range watches {
		if t, ok := ts[id]; !ok || t != w.t {
			w.stop()
			delete(watches, id)
		}
	}
	for id, t := range ts {
		if _, ok := watches[id]; !ok {
			watches[id] = l.start(ctx, newWatch(t))
		}
	}
}

// follow makes ts the servers this Node reaches, and forgets the
// credential, plays and facts of each server that left. The Controller
// ends the plays of a server it names another reaching Node for.
func (l *Link) follow(h Holder, ts map[link.ServerID]target) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.servers = ts
	for id := range l.secrets {
		if _, ok := ts[id]; !ok {
			delete(l.secrets, id)
		}
	}
	for id := range l.facts {
		if _, ok := ts[id]; !ok {
			delete(l.plays, id)
			delete(l.facts, id)
		}
	}
	l.setFacts(h)
}

// Release forgets every credential, play and fact. The holder keeps
// nothing durable.
func (l *Link) Release(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.forget()
	return nil
}

func (l *Link) forget() {
	l.control, l.servers, l.secrets = nil, map[link.ServerID]target{}, map[link.ServerID]held{}
	l.plays, l.facts, l.sent = map[link.ServerID]sessions.Diff{}, map[link.ServerID]*msv1.ServerFacts{}, &msv1.Facts{}
}

func (l *Link) diff(id link.ServerID) sessions.Diff {
	l.mu.Lock()
	defer l.mu.Unlock()
	if d, ok := l.plays[id]; ok {
		return d
	}
	return sessions.New(id)
}

// keep records id's plays and facts once their reports are queued, while
// this Node still reaches id.
func (l *Link) keep(h Holder, id link.ServerID, d sessions.Diff, f *msv1.ServerFacts) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.servers[id]; !ok {
		return
	}
	l.plays[id], l.facts[id] = d, f
	l.setFacts(h)
}

// setFacts sets every server's facts, in id order, when they changed.
// The caller holds l.mu.
func (l *Link) setFacts(h Holder) {
	next := &msv1.Facts{}
	for _, id := range slices.SortedFunc(maps.Keys(l.facts), func(a, b link.ServerID) int { return a.Compare(b.UUID) }) {
		next.Servers = append(next.Servers, l.facts[id])
	}
	if proto.Equal(next, l.sent) {
		return
	}
	l.sent = next
	h.SetFacts(next)
}

// target is the intent's server id, false for one this Node doesn't reach.
func (l *Link) target(id link.ServerID) (target, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	t, ok := l.servers[id]
	return t, ok
}

// credential is ref's credential, fetched from the Controller unless the
// one kept is at ref's settings version or newer. It keeps the version the
// Controller answered, which may be newer than ref's.
func (l *Link) credential(ctx context.Context, ref link.ServerRef) (held, error) {
	l.mu.Lock()
	h, ok := l.secrets[ref.ID]
	control := l.control
	l.mu.Unlock()
	if ok && h.version >= ref.SettingsVersion {
		return h, nil
	}
	resp, err := control.MediaServerSecrets(ctx, &msv1.MediaServerSecretsRequest{ServerId: ref.ID.Bytes()})
	if err != nil {
		return held{}, fmt.Errorf("fetch the credential of Media Server %s: %w", ref.ID, err)
	}
	h = held{cred: CredentialOf(resp.GetCredential()), version: resp.GetSettingsVersion()}
	l.mu.Lock()
	if _, reached := l.servers[ref.ID]; reached {
		l.secrets[ref.ID] = h
	}
	l.mu.Unlock()
	return h, nil
}

func (l *Link) dial(t target, h held) link.Conn {
	k, _ := l.cfg.Kinds.Of(t.ref.Kind)
	ref := t.ref
	ref.SettingsVersion = h.version
	return k.Dial(link.Dialed{
		Server: ref, Address: t.address, Credential: h.cred, DeviceID: t.deviceID,
		MonitorDelay: t.monitorDelay, HTTP: l.cfg.HTTP, Timings: l.cfg.Timings,
	})
}

// running is one server's watch.
type running struct {
	t      target
	cancel context.CancelFunc
	done   chan struct{}
}

func (r *running) stop() {
	r.cancel()
	<-r.done
}

func (l *Link) start(ctx context.Context, w *watch) *running {
	ctx, cancel := context.WithCancel(ctx)
	r := &running{t: w.t, cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(r.done)
		w.supervise(ctx)
	}()
	return r
}
