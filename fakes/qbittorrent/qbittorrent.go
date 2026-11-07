// Package qbittorrent is fake-qbittorrent: qBittorrent's WebUI API v2 over
// the shared fake engine, one client per Node, each reached at that Node's
// front. It plays 4.6 (WebAPI 2.9.3), 5.1 (2.11.4) and 5.2 (2.15.1).
// Written from the release-4.6.7, release-5.1.4 and release-5.2.3 sources
// of webapplication.cpp, authcontroller.cpp, torrentscontroller.cpp,
// appcontroller.cpp and serialize_torrent.cpp, in the order
// WebApplication::processRequest checks a request: the CSRF and host
// checks' 401, the session from its cookie or, on 5.2, an API key as a
// Bearer header, the 403 for an API call with no session, then the API.
// auth/login answers 200 "Ok." or "Fails." with an SID cookie before 5.2,
// and 204 or 401 with a QBT_SID_<port> cookie from 5.2. A banned address
// gets a 403 from it. 5.0 renamed pause and resume to stop and start, the
// paused add parameter to stopped and the paused* states to stopped*.
package qbittorrent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/clientfake"
	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// Versions are the versions this fake plays. The lowest is qBittorrent's
// floor in SlipStream's kind facts.
var Versions = []string{"4.6", "5.1", "5.2"}

// Fake is the qBittorrent an Admin adds on a Node, at that Node's front.
// Its address is typed, as the Challenge solver's is, so Endpoints sets
// nothing. Unreachable plays as a refuse fault on the Node's front.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("qbittorrent"),
	Cues:      clientfake.Cues(name),
}

// AtLimit is what qBittorrent does to a torrent that reaches its seeding
// limit, by max_ratio_act's values.
type AtLimit int

const (
	StopAtLimit              AtLimit = 0
	RemoveAtLimit            AtLimit = 1
	RemoveWithContentAtLimit AtLimit = 3
)

// Client is one Node's qBittorrent.
type Client struct {
	// Node is the name of the Node whose front reaches it.
	Node string
	// Version is one of Versions.
	Version string
	// Username and Password are the WebUI's. A Password of "" plays the
	// Node's address in the auth bypass subnets, so every request gets a
	// session without signing in.
	Username, Password string
	// APIKey is the WebUI API key, "" for none. Only 5.2 has one.
	APIKey string
	// Port is the WebUI port, which names 5.2's cookie, 8080 for 0.
	Port int
	// BanAfter is WebUIMaxAuthFailCount: the failed sign-ins from one
	// address after which it is banned for an hour, 0 for no ban.
	BanAfter int
	// SessionTimeout is WebUISessionTimeout, an hour for 0.
	SessionTimeout time.Duration
	// Domains is the host check's server domains, with * for any number
	// of characters. Nil is qBittorrent's default, *, which passes every
	// host. An IP address passes as qBittorrent's own address does.
	Domains []string
	// Categories are the categories it has when it starts, by name to
	// save path, "" for none.
	Categories map[string]string
	// Dir is the real folder the files land in, and Folder the client's
	// default save path for it, such as /downloads or D:\Torrents.
	Dir, Folder string
	Windows     bool
	// Rate is how many bytes a second each torrent moves.
	Rate int64
	// Slots is max_active_downloads with queueing on, 0 for queueing off.
	Slots int
	// SeedRatio is the global max_ratio, 0 for max_ratio_enabled off, and
	// AtLimit what reaching it does.
	SeedRatio float64
	AtLimit   AtLimit
	// Check is how long checking a torrent takes, on its add and on a
	// recheck, 0 for at once.
	Check time.Duration
	// MoveFor is how long a finished torrent reads moving, as with an
	// incomplete-files folder on, 0 for no move.
	MoveFor time.Duration
}

// Playing is Fake playing clients, for a test that names them itself.
func Playing(clients ...Client) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, clients) }
	return spec
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, nil) }

type version struct {
	app, api string
	// stop is 5.0's rename, WebAPI 2.11.0: stop and start, the stopped add
	// parameter, stopped* states, and root_path in torrents/info.
	stop bool
	// v52 is 5.2's sign-in and answers: the QBT_SID_<port> cookie, 204 and
	// 401 from auth/login, API keys, 204 for a result with no data, JSON
	// or a 409 from torrents/add, and tracker status 5 for an error.
	v52 bool
}

var versions = map[string]version{
	"4.6": {app: "v4.6.7", api: "2.9.3"},
	"5.1": {app: "v5.1.4", api: "2.11.4", stop: true},
	"5.2": {app: "v5.2.3", api: "2.15.1", stop: true, v52: true},
}

const (
	name   = "qbittorrent"
	apiDir = "/api/v2/"
	banFor = time.Hour
)

type server struct {
	clients map[string]*client
}

type client struct {
	Client
	version version
	engine  *torrentclient.Engine

	mu         sync.Mutex
	restarts   int
	sessions   map[string]*session
	nextID     int
	failures   map[string]int
	banned     map[string]time.Time
	categories map[string]string
	announce   map[bencode.InfoHash]string
}

type session struct {
	id   string
	seen time.Time
	// key is a session an API key started, which never expires.
	key bool
}

func serve(env fake.Env, clients []Client) (fake.Server, error) {
	s := &server{clients: map[string]*client{}}
	for _, c := range clients {
		v, ok := versions[c.Version]
		if !ok {
			return nil, fmt.Errorf("fake-qbittorrent plays %v, not %q", Versions, c.Version)
		}
		if c.APIKey != "" && !v.v52 {
			return nil, fmt.Errorf("qBittorrent %s has no API key", c.Version)
		}
		s.clients[c.Node] = &client{
			Client: c, version: v,
			engine: torrentclient.New(torrentclient.Config{
				Releases: env.Releases, Dir: c.Dir, Name: c.Folder, Windows: c.Windows, Rate: c.Rate, Slots: c.Slots, SeedRatio: c.SeedRatio, Check: c.Check,
			}),
			sessions: map[string]*session{}, failures: map[string]int{}, banned: map[string]time.Time{},
			categories: map[string]string{}, announce: map[bencode.InfoHash]string{},
		}
		maps.Copy(s.clients[c.Node].categories, c.Categories)
	}
	engineOf := func(node string) (*torrentclient.Engine, bool) {
		c, ok := s.clients[node]
		if !ok {
			return nil, false
		}
		return c.engine, true
	}
	if err := clientfake.OnCues(env, name, engineOf); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{"webui": {Pattern: "/", Handler: s.handle}}), nil
}

// endpoint is an API call's handler and the method it takes, "" for GET
// or POST.
type endpoint struct {
	method string
	serve  func(c *client, w http.ResponseWriter, r *http.Request, s *session)
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	c, ok := s.clients[fake.OriginOf(r.Context())]
	if !ok {
		http.Error(w, "fake-qbittorrent plays no qBittorrent for this Node.", http.StatusNotFound)
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweep()
	key := c.version.v52 && bearer(r) != ""
	if (!key && crossSite(r)) || !c.hostAllowed(r.Host) {
		text(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	ses := c.session(w, r, key)
	call, api := strings.CutPrefix(r.URL.Path, apiDir)
	switch {
	case !api && key:
		text(w, http.StatusNotFound, "Not Found")
		return
	case !api:
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!DOCTYPE html><title>qBittorrent WebUI</title>"))
		return
	case ses == nil && call != "auth/login":
		text(w, http.StatusForbidden, "Forbidden")
		return
	case key && strings.HasPrefix(call, "auth/"):
		text(w, http.StatusForbidden, "Forbidden")
		return
	}
	e, ok := c.endpoints()[call]
	switch {
	case !ok:
		text(w, http.StatusNotFound, c.notFound())
	case e.method != "" && r.Method != e.method, r.Method != http.MethodGet && r.Method != http.MethodPost:
		text(w, http.StatusMethodNotAllowed, "Method Not Allowed")
	default:
		e.serve(c, w, r, ses)
	}
}

// sweep is what happened in the client since the last request: a restart
// ends every session and ban, and a remove action at the seeding limit
// removes the torrents the engine stopped there.
func (c *client) sweep() {
	if n := c.engine.Restarts(); n != c.restarts {
		c.restarts = n
		clear(c.sessions)
		clear(c.failures)
		clear(c.banned)
	}
	if c.AtLimit == StopAtLimit {
		return
	}
	for _, t := range c.engine.List(nil) {
		if t.SeedGoalMet {
			_ = c.engine.Remove(t.Hash, c.AtLimit == RemoveWithContentAtLimit)
		}
	}
}

// crossSite is qBittorrent's CSRF check: an Origin, or else a Referer,
// whose host isn't the request's. A request with neither passes.
func crossSite(r *http.Request) bool {
	for _, h := range []string{"Origin", "Referer"} {
		if v := r.Header.Get(h); v != "" {
			u, err := url.Parse(v)
			return err != nil || !strings.EqualFold(u.Host, r.Host)
		}
	}
	return false
}

func (c *client) hostAllowed(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	if _, err := netip.ParseAddr(strings.Trim(host, "[]")); err == nil || c.Domains == nil {
		return true
	}
	return slices.ContainsFunc(c.Domains, func(pattern string) bool {
		ok, _ := path.Match(strings.ToLower(pattern), strings.ToLower(host))
		return ok
	})
}

func bearer(r *http.Request) string {
	scheme, key, ok := strings.Cut(r.Header.Get("Authorization"), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(key)
}

func (c *client) cookieName() string {
	if !c.version.v52 {
		return "SID"
	}
	port := c.Port
	if port == 0 {
		port = 8080
	}
	return "QBT_SID_" + strconv.Itoa(port)
}

func (c *client) timeout() time.Duration {
	if c.SessionTimeout == 0 {
		return time.Hour
	}
	return c.SessionTimeout
}

// session is the request's session: an API key's, its cookie's while it
// hasn't expired, or a new one for an address that needs no sign-in.
func (c *client) session(w http.ResponseWriter, r *http.Request, key bool) *session {
	now := time.Now()
	if key {
		if c.APIKey == "" || bearer(r) != c.APIKey {
			return nil
		}
		s, ok := c.sessions[c.APIKey]
		if !ok {
			s = &session{id: c.APIKey, key: true}
			c.sessions[s.id] = s
		}
		s.seen = now
		return s
	}
	if cookie, err := r.Cookie(c.cookieName()); err == nil {
		if s, ok := c.sessions[cookie.Value]; ok && !s.key {
			if now.Sub(s.seen) <= c.timeout() {
				s.seen = now
				return s
			}
			delete(c.sessions, s.id)
		}
	}
	if c.Password == "" {
		return c.start(w)
	}
	return nil
}

func (c *client) start(w http.ResponseWriter) *session {
	c.nextID++
	sum := sha256.Sum256([]byte(c.Node + "/" + strconv.Itoa(c.restarts) + "/" + strconv.Itoa(c.nextID)))
	s := &session{id: hex.EncodeToString(sum[:16]), seen: time.Now()}
	c.sessions[s.id] = s
	cookie := &http.Cookie{Name: c.cookieName(), Value: s.id, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode} //nolint:gosec // qBittorrent's own cookie, Secure only behind HTTPS
	if c.version.v52 {
		cookie.Expires, cookie.SameSite = time.Now().Add(c.timeout()), http.SameSiteLaxMode
	}
	http.SetCookie(w, cookie)
	return s
}

func (c *client) notFound() string {
	if c.version.v52 {
		return "Endpoint does not exist"
	}
	return "Not Found"
}

func text(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body)) //nolint:gosec // a text/plain body
}

// done is an action's empty result: 200 with no body, as setResult of an
// empty string answers in every version.
func done(w http.ResponseWriter) { text(w, http.StatusOK, "") }

func remote(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
