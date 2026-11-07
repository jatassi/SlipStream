// Package rtorrent is fake-rtorrent: rTorrent's XML-RPC behind nginx's
// scgi_pass at /RPC2, over the shared fake engine, one rTorrent per Node,
// each reached at that Node's front. Written from the rtorrent v0.9.8 and
// v0.16.27 sources of src/rpc/xmlrpc.cc (0.9.8, xmlrpc-c),
// xmlrpc_tinyxml2.cc and rpc_manager.cc (0.16), command_download.cc,
// command_file.cc, command_events.cc, core/download.cc and
// core/download_list.cc, and libtorrent's tracker_http.cc. 0.9.8 plays
// the xmlrpc-c build and 0.16 the tinyxml2 one, which differ in their
// faults for a hash rTorrent doesn't hold. Every call takes a target
// first: an info hash for a d.* or f.* call, "" otherwise. Every answer is
// HTTP 200, a fault included. A load.raw or load.raw_start is queued and
// the torrent shows from the next request on, as rTorrent loads it on a
// later pass of its main loop, and a duplicate or a load whose command
// fails loads nothing. d.erase never deletes data. nginx's auth_basic
// answers 401 to a request without the right username and password, with
// no lockout.
package rtorrent

import (
	"errors"
	"fmt"
	"io"
	"net/http"
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

// Versions are the versions this fake plays. The lowest is rTorrent's
// floor in SlipStream's kind facts.
var Versions = []string{"0.9.8", "0.16"}

const name = "rtorrent"

// Fake is the rTorrent an Admin adds on a Node, at that Node's front. Its
// address is typed, so Endpoints sets nothing.
var Fake = fake.Spec{
	Name:      name,
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.OnlineSuite("rtorrent"),
	Cues:      clientfake.Cues(name),
}

// Client is one Node's rTorrent and the nginx in front of it.
type Client struct {
	// Node is the name of the Node whose front reaches it.
	Node string
	// Version is one of Versions.
	Version string
	// Username and Password turn on nginx's auth_basic, and "" for both
	// leaves it off.
	Username, Password string
	// Dir is the real folder the files land in, and Folder rTorrent's
	// directory.default for it, such as /downloads.
	Dir, Folder string
	// Rate is how many bytes a second each torrent moves.
	Rate int64
	// Check is how long hashing a torrent takes, on its add and on a
	// d.check_hash, 0 for at once.
	Check time.Duration
}

// Playing is Fake playing clients, for a test that names them itself.
func Playing(clients ...Client) fake.Spec {
	spec := Fake
	spec.New = func(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, clients) }
	return spec
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) { return serve(env, nil) }

type version struct {
	client string
	style  style
	// untyped reads a <value> with bare text as a string, as xmlrpc-c does.
	untyped bool
	// notFound is the fault for a target hash rTorrent doesn't hold, and
	// localError the code a command's own error carries.
	notFound   fault
	localError int
	// unknown words the fault for a method rTorrent doesn't define.
	unknown func(method string) string
	// warns is libtorrent from 0.16, which reads a tracker's warning
	// message that says the torrent is unregistered as a failure.
	warns bool
}

var versions = map[string]version{
	"0.9.8": {
		client: "0.9.8", style: style{lines: true, faultI4: true}, untyped: true,
		notFound: fault{typeError, "Could not find info-hash."}, localError: parseError,
		unknown: func(m string) string { return "Method '" + m + "' not defined" },
	},
	"0.16": {
		client: "0.16.27", style: style{},
		notFound: fault{internalError, "invalid parameters: info-hash not found"}, localError: internalError,
		unknown: func(m string) string { return "method '" + m + "' not defined" },
		warns:   true,
	},
}

type server struct {
	clients map[string]*rtorrent
}

// rtorrent is one Node's rTorrent.
type rtorrent struct {
	Client
	version version
	engine  *torrentclient.Engine

	mu sync.Mutex
	// held is what rTorrent keeps of each torrent beside the engine.
	held map[bencode.InfoHash]*held
	// loads are the loads of the request being answered.
	loads []load
}

// held is a torrent's d.state, a d.pause that holds it while started, a
// d.check_hash's rehash and its metainfo's shape.
type held struct {
	started, paused, rehash bool
	multi                   bool
	pieceLength             int64
}

type load struct {
	metainfo []byte
	start    bool
	folder   string
	label    string
}

func serve(env fake.Env, clients []Client) (fake.Server, error) {
	s := &server{clients: map[string]*rtorrent{}}
	for _, c := range clients {
		v, ok := versions[c.Version]
		if !ok {
			return nil, fmt.Errorf("fake-rtorrent plays %v, not %q", Versions, c.Version)
		}
		s.clients[c.Node] = &rtorrent{Client: c, version: v, held: map[bencode.InfoHash]*held{}, engine: torrentclient.New(torrentclient.Config{
			Releases: env.Releases, Dir: c.Dir, Name: c.Folder, Rate: c.Rate, Check: c.Check,
		})}
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
	return fake.HTTP(env, fake.Routes{"rpc": {Pattern: "/", Handler: s.handle}}), nil
}

// handle is nginx: its 404 for any path but /RPC2, auth_basic's 401, then
// scgi_pass to rTorrent.
func (s *server) handle(w http.ResponseWriter, r *http.Request) {
	c, ok := s.clients[fake.OriginOf(r.Context())]
	switch {
	case !ok:
		http.Error(w, "fake-rtorrent plays no rTorrent for this Node.", http.StatusNotFound)
	case r.URL.Path != "/RPC2":
		page(w, http.StatusNotFound, "404 Not Found")
	case !c.authorized(r):
		w.Header().Set("WWW-Authenticate", `Basic realm="rTorrent"`)
		page(w, http.StatusUnauthorized, "401 Authorization Required")
	default:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			page(w, http.StatusBadRequest, "400 Bad Request")
			return
		}
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write(c.process(body))
	}
}

// page is nginx's own error page.
func page(w http.ResponseWriter, status int, title string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "<html>\r\n<head><title>%s</title></head>\r\n<body>\r\n<center><h1>%s</h1></center>\r\n<hr><center>nginx/1.24.0</center>\r\n</body>\r\n</html>\r\n", title, title)
}

func (c *rtorrent) authorized(r *http.Request) bool {
	if c.Username == "" && c.Password == "" {
		return true
	}
	user, pass, ok := r.BasicAuth()
	return ok && user == c.Username && pass == c.Password
}

// process answers one XML-RPC request, then loads what it queued.
func (c *rtorrent) process(body []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	defer c.loadQueued()
	req, err := parseCall(body, c.version.untyped)
	if err != nil {
		return c.version.style.fault(c.asFault(err))
	}
	if req.method == "system.multicall" {
		res, err := c.multicall(req.params)
		if err != nil {
			return c.version.style.fault(c.asFault(err))
		}
		return c.version.style.response(res)
	}
	res, err := c.call(req.method, req.params)
	if err != nil {
		return c.version.style.fault(c.asFault(err))
	}
	return c.version.style.response(res)
}

func (c *rtorrent) asFault(err error) *fault {
	if f, ok := errors.AsType[*fault](err); ok {
		return f
	}
	return &fault{c.version.localError, err.Error()}
}

// multicall is system.multicall: an array of {methodName, params}, each
// answered as a one-element array or a fault struct.
func (c *rtorrent) multicall(params []any) ([]any, error) {
	if len(params) != 1 {
		return nil, &fault{parseError, "system.multicall takes one array"}
	}
	calls, ok := params[0].([]any)
	if !ok {
		return nil, &fault{typeError, "system.multicall takes one array"}
	}
	out := make([]any, len(calls))
	for i, raw := range calls {
		call, ok := raw.(map[string]any)
		method, named := call["methodName"].(string)
		if !ok || !named || method == "" {
			return nil, &fault{parseError, "multicall struct's first member must be methodName"}
		}
		args, _ := call["params"].([]any)
		res, err := c.call(method, args)
		if err != nil {
			out[i] = c.version.style.faultStruct(c.asFault(err))
			continue
		}
		out[i] = []any{res}
	}
	return out, nil
}

// call runs one method. Its first param is its target.
func (c *rtorrent) call(method string, params []any) (any, error) {
	if len(params) == 0 {
		return nil, &fault{typeError, "invalid parameters: too few"}
	}
	target, ok := params[0].(string)
	if !ok {
		return nil, &fault{typeError, "Unsupported target type found."}
	}
	args := params[1:]
	if run, ok := global[method]; ok {
		if target != "" {
			return nil, errWrongTarget
		}
		return run(c, args)
	}
	run, ok := onTorrent[method]
	if !ok {
		return nil, &fault{noSuchMethod, c.version.unknown(method)}
	}
	if target == "" {
		return nil, errWrongTarget
	}
	h, err := bencode.ParseInfoHash(target)
	if err != nil {
		return nil, &fault{c.version.notFound.code, c.version.notFound.message}
	}
	t, ok := c.engine.Get(h)
	if !ok {
		return nil, &fault{c.version.notFound.code, c.version.notFound.message}
	}
	return run(c, t, c.held[h], args)
}

// loadQueued loads the torrents the last request queued, as rTorrent's
// main loop does after it answered.
func (c *rtorrent) loadQueued() {
	for _, l := range c.loads {
		m, err := bencode.DecodeMetainfo(l.metainfo)
		if err != nil {
			continue
		}
		var labels []string
		if l.label != "" {
			labels = []string{l.label}
		}
		_, duplicate, err := c.engine.Add(l.metainfo, l.folder, labels, !l.start)
		if err != nil || duplicate {
			continue
		}
		c.held[m.InfoHash] = &held{started: l.start, multi: m.Info.Files[0].Path != nil, pieceLength: m.Info.PieceLength}
	}
	c.loads = nil
}

// directory is d.directory: the folder, with the torrent's name for a
// multi-file torrent, as d.directory.set adds it.
func directory(t torrentclient.Torrent, h *held) string {
	if !h.multi {
		return t.Folder
	}
	if strings.HasSuffix(t.Folder, "/") {
		return t.Folder + t.Name
	}
	return t.Folder + "/" + t.Name
}
