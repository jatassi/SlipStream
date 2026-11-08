// Package transmission is fake-transmission: Transmission's RPC over the
// shared fake engine, one daemon per Node, each reached at that Node's
// front. It plays 4.0, which speaks only the legacy protocol, and 4.1,
// which speaks JSON-RPC 2.0 with snake_case names, tells the two apart by
// the request's jsonrpc key, and names its RPC version in the 409's
// X-Transmission-Rpc-Version. Written from 4.1.1's rpc-server.cc and
// rpcimpl.cc and checked against a 4.0.5 daemon: the anti-brute-force
// 403, the rpc_whitelist 403, Basic auth's 401, the host whitelist's 421
// and the session id's 409, in that order, then the RPC, which answers
// 200 even when it fails. Its two 403s have one body, as both versions'
// do.
package transmission

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/netip"
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
	"github.com/jatassi/slipstream/internal/node"
)

// Versions are the versions this fake plays. The lowest is Transmission's
// floor in SlipStream's kind facts.
var Versions = []string{"4.0", "4.1"}

// Fake is the Transmission an Admin adds on a Node, at that Node's front.
// Its address is typed, as the Challenge solver's is, so Endpoints sets
// nothing. Unreachable plays as a refuse fault on the Node's front.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("transmission"),
	Cues:      clientfake.Cues(name),
	Captures:  []fake.CaptureSpec{Added, Removed},
	Entered:   clientfake.Entered(name),
}

var (
	// Added records each torrent a daemon added, not one it already had.
	Added = fake.CaptureSpec{Name: "transmission-adds", Kind: "transmission.add"}
	// Removed records each torrent a daemon removed.
	Removed = fake.CaptureSpec{Name: "transmission-removes", Kind: "transmission.remove"}
)

// Add is an Added record's data, Folder as the daemon names it.
type Add struct {
	InfoHash string   `json:"infoHash"`
	Name     string   `json:"name"`
	Folder   string   `json:"folder"`
	Labels   []string `json:"labels"`
	Paused   bool     `json:"paused,omitempty"`
}

// Removal is a Removed record's data, with Data for a remove that deleted
// the torrent's files.
type Removal struct {
	InfoHash string `json:"infoHash"`
	Data     bool   `json:"data,omitempty"`
}

// Daemon is one Node's Transmission.
type Daemon struct {
	// Node is the name of the Node whose front reaches it.
	Node string
	// Version is one of Versions.
	Version string
	// Username and Password turn on rpc_authentication_required, and ""
	// for both leaves it off.
	Username, Password string
	// Whitelist is rpc_whitelist, the addresses allowed to connect, with *
	// for any number, nil for the whitelist off.
	Whitelist []string
	// Hosts is rpc_host_whitelist. The host check is on, as Transmission's
	// default is, and passes localhost and any IP address.
	Hosts []string
	// Lockout turns on anti_brute_force_enabled, which is off by default,
	// with this anti_brute_force_threshold: the failed sign-ins after
	// which every request is refused until the daemon restarts.
	Lockout int
	// Dir is the real folder the files land in, and Folder the daemon's
	// download_dir for it, such as /downloads or D:\Torrents.
	Dir, Folder string
	Windows     bool
	// Rate is how many bytes a second each torrent moves.
	Rate int64
	// Slots is download_queue_size, 0 for the queue off.
	Slots int
	// SeedRatio is seedRatioLimit, 0 for seedRatioLimited off.
	SeedRatio float64
	// Check is how long verifying a torrent takes, on its add and on a
	// recheck, 0 for at once.
	Check time.Duration
}

// Playing is Fake playing daemons, for a test that names them itself.
func Playing(daemons ...Daemon) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, daemons) }
	return spec
}

// newServer plays the Transmission of each Node the Scenario gives one.
func newServer(s scenario.Scenario, env fake.Env) (fake.Server, error) {
	var daemons []Daemon
	for _, p := range clientfake.Playing(s, name, env.Dir) {
		daemons = append(daemons, Daemon{
			Node: p.Node, Version: p.Version, Username: p.Username, Password: p.Password,
			Dir: p.Dir, Folder: p.Folder, Windows: p.Windows, Rate: clientfake.Rate,
		})
	}
	return serve(env, daemons)
}

type version struct {
	name, semver string
	rpc          int
	// json2 is a daemon that speaks JSON-RPC 2.0 beside the legacy
	// protocol.
	json2 bool
	// words spells a settings name in the daemon's own pages: snake_case
	// from 4.1, kebab-case before.
	words func(string) string
}

var versions = map[string]version{
	"4.0": {name: "4.0.5 (a6fe2a64aa)", semver: "5.3.0", rpc: 17, words: func(s string) string { return strings.ReplaceAll(s, "_", "-") }},
	"4.1": {name: "4.1.1 (a8bd3e2b)", semver: "6.0.1", rpc: 19, json2: true, words: func(s string) string { return s }},
}

const (
	name          = "transmission"
	sessionHeader = "X-Transmission-Session-Id"
	versionHeader = "X-Transmission-Rpc-Version"
	base          = "/transmission/"
)

type server struct {
	daemons map[string]*daemon
}

type captures struct {
	added, removed *fake.Capture
}

type daemon struct {
	Daemon
	version version
	engine  *torrentclient.Engine
	captures

	mu       sync.Mutex
	failures int
}

func serve(env fake.Env, daemons []Daemon) (fake.Server, error) {
	var c captures
	var err error
	if c.added, err = env.Capture(Added.Name); err != nil {
		return nil, err
	}
	if c.removed, err = env.Capture(Removed.Name); err != nil {
		return nil, err
	}
	s := &server{daemons: map[string]*daemon{}}
	for _, d := range daemons {
		v, ok := versions[d.Version]
		if !ok {
			return nil, fmt.Errorf("fake-transmission plays %v, not %q", Versions, d.Version)
		}
		s.daemons[d.Node] = &daemon{Daemon: d, version: v, captures: c, engine: torrentclient.New(torrentclient.Config{
			Releases: env.Releases, Dir: d.Dir, Name: d.Folder, Windows: d.Windows, Rate: d.Rate, Slots: d.Slots, SeedRatio: d.SeedRatio, Check: d.Check,
		})}
	}
	engineOf := func(node string) (*torrentclient.Engine, bool) {
		d, ok := s.daemons[node]
		if !ok {
			return nil, false
		}
		return d.engine, true
	}
	if err := clientfake.OnCues(env, name, engineOf); err != nil {
		return nil, err
	}
	return fake.HTTP(env, fake.Routes{"rpc": {Pattern: "/", Handler: s.handle}}), nil
}

func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	d, ok := s.daemons[fake.OriginOf(r.Context())]
	if !ok {
		http.Error(w, "fake-transmission plays no Transmission for this Node.", http.StatusNotFound)
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	switch {
	case d.Lockout > 0 && d.failures >= d.Lockout, !d.addressAllowed(r.RemoteAddr):
		simple(w, http.StatusForbidden, "")
	case !d.authorized(r):
		d.failures++
		w.Header().Set("WWW-Authenticate", `Basic realm="Transmission"`)
		simple(w, http.StatusUnauthorized, "")
	default:
		d.route(w, r)
	}
}

// route is what follows sign-in: the web UI, the host check, the session
// id and the RPC, under /transmission/.
func (d *daemon) route(w http.ResponseWriter, r *http.Request) {
	words := d.version.words
	location, under := strings.CutPrefix(r.URL.RequestURI(), base)
	switch {
	case !under || location == "" || location == "web":
		w.Header().Set("Location", base+"web/")
		simple(w, http.StatusMovedPermanently, "")
	case strings.HasPrefix(location, "web/"):
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!DOCTYPE html><title>Transmission Web</title>"))
	case !d.hostAllowed(r.Host):
		simple(w, http.StatusMisdirectedRequest, "<p>Transmission received your request, but the hostname was unrecognized.</p>"+
			"<p>To fix this, choose one of the following options:<ul><li>Enable password authentication, then any hostname is allowed.</li>"+
			"<li>Add the hostname you want to use to the whitelist in settings.</li></ul></p>"+
			"<p>If you're editing settings.json, see the '"+words("rpc_host_whitelist")+"' and '"+words("rpc_host_whitelist_enabled")+"' entries.</p>"+
			`<p>This requirement has been added to help prevent <a href="https://en.wikipedia.org/wiki/DNS_rebinding">DNS Rebinding</a> attacks.</p>`)
	case r.Header.Get(sessionHeader) != d.session():
		w.Header().Set(sessionHeader, d.session())
		w.Header().Set("Access-Control-Expose-Headers", sessionHeader)
		if d.version.json2 {
			w.Header().Set(versionHeader, d.version.semver)
			w.Header().Set("Access-Control-Expose-Headers", sessionHeader+", "+versionHeader)
		}
		simple(w, http.StatusConflict, "<p>Your request had an invalid "+words("session_id")+" header.</p><p>To fix this, follow these steps:"+
			"<ol><li> When reading a response, get its "+sessionHeader+" header and remember it<li> Add the updated header to your outgoing requests"+
			"<li> When you get this 409 error message, resend your request with the updated header</ol></p>"+
			`<p>This requirement has been added to help prevent <a href="https://en.wikipedia.org/wiki/Cross-site_request_forgery">CSRF</a> attacks.</p>`+
			"<p><code>"+sessionHeader+": "+d.session()+"</code></p>")
	case strings.HasPrefix(location, "rpc"):
		d.rpc(w, r)
	default:
		simple(w, http.StatusNotFound, html.EscapeString(r.URL.RequestURI()))
	}
}

// simple is Transmission's send_simple_response.
func simple(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/html; charset=ISO-8859-1")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "<h1>%d: %s</h1>%s", status, http.StatusText(status), text)
}

func (d *daemon) addressAllowed(remote string) bool {
	if d.Whitelist == nil {
		return true
	}
	host, _, err := net.SplitHostPort(remote)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(d.Whitelist, func(pattern string) bool {
		ok, _ := path.Match(pattern, host)
		return ok
	})
}

func (d *daemon) auth() bool { return d.Username != "" || d.Password != "" }

func (d *daemon) authorized(r *http.Request) bool {
	if !d.auth() {
		return true
	}
	user, pass, ok := r.BasicAuth()
	if !ok || user != d.Username || pass != d.Password {
		return false
	}
	d.failures = 0
	return true
}

func (d *daemon) hostAllowed(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	_, ipErr := netip.ParseAddr(strings.Trim(host, "[]"))
	return d.auth() || ipErr == nil || host == "localhost" || slices.ContainsFunc(d.Hosts, func(pattern string) bool {
		ok, _ := path.Match(pattern, host)
		return ok
	})
}

// session is the daemon's session id, which changes when it restarts.
func (d *daemon) session() string {
	sum := sha256.Sum256([]byte(d.Node + "/" + strconv.Itoa(d.engine.Restarts())))
	return hex.EncodeToString(sum[:])[:48]
}

func unix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}
