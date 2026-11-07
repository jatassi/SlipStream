// Package deluge is fake-deluge: Deluge 2.1's Web UI JSON-RPC at /json over
// the shared fake engine, one web UI per Node, each reached at that Node's
// front and listing one or more daemons with their own engines. Written
// from the deluge-2.1.1 sources of deluge/ui/web/json_api.py, auth.py,
// deluge/ui/hostlist.py, deluge/core/core.py, torrent.py, torrentmanager.py
// and the Label plugin's core.py. A request must be a POST with
// application/json, or it gets a 405 or code 5. Every answer is HTTP 200
// with {"result","error","id"}. A method the web UI doesn't know is code 2,
// which a core.* or label.* method is until web.connect registers the
// daemon's methods. A call without a session is code 1. auth.login answers
// true with a _session_id cookie, or false, and there is no lockout. A
// daemon's own failure is code 4, in the words of the Twisted Failure that
// carries it. web.get_host_status reads Connected only for the daemon the
// web UI is connected to, and only when the web UI lists it by IP address,
// since 2.1.1 compares the listed host with its own resolved address.
package deluge

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"net/http"
	"net/netip"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/clientfake"
	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// Versions are the versions this fake plays. The lowest is Deluge's floor
// in SlipStream's kind facts.
var Versions = []string{"2.1"}

const (
	name = "deluge"
	// switchCue connects a Node's web UI to another of its daemons, as a
	// person does in the web UI's connection manager.
	switchCue = name + "-switch"
	// disconnectCue disconnects a Node's web UI from its daemon, as a
	// person does in the connection manager. The web UI keeps the daemon's
	// methods, so a call to one waits for an answer that never comes.
	disconnectCue = name + "-disconnect"
)

// Fake is the Deluge an Admin adds on a Node, at that Node's front. Its
// address is typed, so Endpoints sets nothing. Besides the shared cues,
// deluge-switch {node, host, port} connects the web UI to another daemon
// and deluge-disconnect {node} disconnects it. deluge-restart restarts the
// web UI, ending every session and its daemon connection.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("deluge"),
	Cues:      append(clientfake.Cues(name), fake.CueSpec{Name: switchCue}, fake.CueSpec{Name: disconnectCue}),
}

// Client is one Node's Deluge web UI and the daemons it lists.
type Client struct {
	// Node is the name of the Node whose front reaches it.
	Node string
	// Version is one of Versions.
	Version  string
	Password string
	// SessionTimeout is web.conf's session_timeout, an hour for 0. A call
	// with the session refreshes it.
	SessionTimeout time.Duration
	// Daemons are the web UI's host list, in order. The shared cues play
	// on the first.
	Daemons []Daemon
	// AutoConnect is web.conf's default_daemon by its place in Daemons from
	// 1, which the web UI connects to when it starts, 0 for none.
	AutoConnect int
}

// Daemon is one deluged and its core.conf.
type Daemon struct {
	Host string
	Port int
	// Dir is the real folder the files land in, and Folder the client's
	// download_location for it, such as /downloads or D:\Torrents.
	Dir, Folder string
	Windows     bool
	// Rate is how many bytes a second each torrent moves.
	Rate int64
	// Slots is max_active_downloading, 0 for no queue.
	Slots int
	// SeedRatio is stop_seed_ratio with stop_seed_at_ratio on, 0 for off,
	// and RemoveAtRatio is remove_seed_at_ratio, which removes a torrent
	// there and keeps its files.
	SeedRatio     float64
	RemoveAtRatio bool
	// MoveTo is move_completed_path with move_completed on, "" for off,
	// and MoveFor how long the move reads Moving. MoveTo is under Folder.
	MoveTo  string
	MoveFor time.Duration
	// LabelOff is the Label plugin disabled.
	LabelOff bool
	// Labels are the labels the daemon has when it starts, by name to the
	// move_completed_path the label applies, "" for none.
	Labels map[string]string
	// Check is how long checking a torrent takes, on its add and on a
	// recheck, 0 for at once.
	Check time.Duration
}

// Playing is Fake playing clients, for a test that names them itself.
func Playing(clients ...Client) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, clients) }
	return spec
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, nil) }

// version is the daemon.get_version of each of Versions.
var versions = map[string]string{"2.1": "2.1.1"}

type server struct {
	clients map[string]*web
}

// web is one Node's deluge-web.
type web struct {
	Client
	version string
	daemons []*daemon

	mu       sync.Mutex
	restarts int
	nextID   int
	sessions map[string]time.Time
	// connected is the daemon the web UI is connected to, and methods the
	// one whose methods it registered at its last connect, -1 for none.
	connected, methods int
}

type daemon struct {
	Daemon
	id      string
	version string
	engine  *torrentclient.Engine
	// labels are the Label plugin's, by name to the move path it applies.
	labels map[string]string
	// moveTo is each torrent's move_completed_path while its
	// move_completed is on.
	moveTo map[bencode.InfoHash]string
	// paused are the torrents added paused or paused since, which read
	// Paused while the engine checks them.
	paused map[bencode.InfoHash]bool
}

func serve(env fake.Env, clients []Client) (fake.Server, error) {
	s := &server{clients: map[string]*web{}}
	for _, c := range clients {
		v, ok := versions[c.Version]
		if !ok {
			return nil, fmt.Errorf("fake-deluge plays %v, not %q", Versions, c.Version)
		}
		w := &web{Client: c, version: v, sessions: map[string]time.Time{}, connected: -1, methods: -1}
		for _, d := range c.Daemons {
			sum := sha256.Sum256([]byte(d.Host + ":" + strconv.Itoa(d.Port)))
			labels := maps.Clone(d.Labels)
			if labels == nil {
				labels = map[string]string{}
			}
			w.daemons = append(w.daemons, &daemon{
				Daemon: d, id: hex.EncodeToString(sum[:16]), version: v, labels: labels, moveTo: map[bencode.InfoHash]string{}, paused: map[bencode.InfoHash]bool{},
				engine: torrentclient.New(torrentclient.Config{
					Releases: env.Releases, Dir: d.Dir, Name: d.Folder, Windows: d.Windows, Rate: d.Rate, Slots: d.Slots, SeedRatio: d.SeedRatio, Check: d.Check,
				}),
			})
		}
		w.start()
		s.clients[c.Node] = w
	}
	if err := s.onCues(env); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{
		"json":  {Pattern: "/json", Handler: s.handle},
		"webui": {Pattern: "/", Handler: s.page},
	}), nil
}

type switchArgs struct {
	Node string `json:"node"`
	Host string `json:"host"`
	Port int    `json:"port"`
}

type nodeArgs struct {
	Node string `json:"node"`
}

func (s *server) onCues(env fake.Env) error {
	engineOf := func(node string) (*torrentclient.Engine, bool) {
		w, ok := s.clients[node]
		if !ok || len(w.daemons) == 0 {
			return nil, false
		}
		return w.daemons[0].engine, true
	}
	if err := clientfake.OnCues(env, name, engineOf); err != nil {
		return err
	}
	webOf := func(node string) (*web, error) {
		w, ok := s.clients[node]
		if !ok {
			return nil, fmt.Errorf("fake deluge plays no web UI on Node %q", node)
		}
		return w, nil
	}
	if err := fake.OnCue(env, switchCue, func(_ context.Context, a switchArgs) (any, error) {
		w, err := webOf(a.Node)
		if err != nil {
			return nil, err
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		for i, d := range w.daemons {
			if d.Host == a.Host && d.Port == a.Port {
				w.connect(i)
				return a, nil
			}
		}
		return nil, fmt.Errorf("the web UI on Node %q lists no daemon %s:%d", a.Node, a.Host, a.Port)
	}); err != nil {
		return err
	}
	return fake.OnCue(env, disconnectCue, func(_ context.Context, a nodeArgs) (any, error) {
		w, err := webOf(a.Node)
		if err != nil {
			return nil, err
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		w.connected = -1
		return a, nil
	})
}

// start is the web UI starting: no session, no daemon, then the
// default_daemon's connect.
func (w *web) start() {
	clear(w.sessions)
	w.connected, w.methods = -1, -1
	if w.AutoConnect > 0 {
		w.connect(w.AutoConnect - 1)
	}
}

func (w *web) connect(i int) { w.connected, w.methods = i, i }

func (w *web) timeout() time.Duration {
	if w.SessionTimeout == 0 {
		return time.Hour
	}
	return w.SessionTimeout
}

// sweep is what happened since the last request: a restart of the web UI,
// moves whose time has come, and each daemon's removes at its seed ratio.
func (w *web) sweep() {
	if len(w.daemons) > 0 {
		if n := w.daemons[0].engine.Restarts(); n != w.restarts {
			w.restarts = n
			w.start()
		}
	}
	now := time.Now()
	for _, d := range w.daemons {
		for _, t := range d.engine.List(nil) {
			if to, ok := d.moveTo[t.Hash]; ok && t.Done == t.Size && !now.Before(t.CompletedAt.Add(d.MoveFor)) {
				if d.engine.Move(t.Hash, to) == nil {
					delete(d.moveTo, t.Hash)
				}
			}
			if t.SeedGoalMet && d.RemoveAtRatio {
				_ = d.engine.Remove(t.Hash, false)
				delete(d.moveTo, t.Hash)
			}
		}
	}
}

// page is the web UI's own page, which any method may fetch.
func (s *server) page(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.clients[fake.OriginOf(r.Context())]; !ok {
		http.Error(w, "fake-deluge plays no Deluge for this Node.", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!DOCTYPE html><title>Deluge: Web UI 2.1.1</title>"))
}

// newSession is auth._create_session: a sha256 hex id and the sum of its
// characters as its checksum.
func (w *web) newSession(rw http.ResponseWriter) {
	w.nextID++
	sum := sha256.Sum256([]byte(w.Node + "/" + strconv.Itoa(w.restarts) + "/" + strconv.Itoa(w.nextID)))
	id := hex.EncodeToString(sum[:])
	w.sessions[id] = time.Now().Add(w.timeout())
	w.setCookie(rw, id+strconv.Itoa(checksum(id)))
}

func (w *web) setCookie(rw http.ResponseWriter, value string) {
	http.SetCookie(rw, &http.Cookie{Name: "_session_id", Value: value, Path: "/json", Expires: time.Now().Add(w.timeout())}) //nolint:gosec // Deluge's own cookie
}

func checksum(id string) int {
	n := 0
	for _, r := range id {
		n += int(r)
	}
	return n
}

// session is check_request: the request's session while it hasn't
// expired, refreshed, or "" for none.
func (w *web) session(rw http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("_session_id")
	if err != nil || len(cookie.Value) < 5 {
		return ""
	}
	id := cookie.Value[:len(cookie.Value)-4]
	if sum, err := strconv.Atoi(cookie.Value[len(cookie.Value)-4:]); err != nil || sum != checksum(id) {
		return ""
	}
	expires, ok := w.sessions[id]
	if !ok {
		return ""
	}
	if time.Now().After(expires) {
		delete(w.sessions, id)
		return ""
	}
	w.sessions[id] = time.Now().Add(w.timeout())
	w.setCookie(rw, cookie.Value)
	return id
}

func ipAddress(host string) bool {
	_, err := netip.ParseAddr(host)
	return err == nil
}
