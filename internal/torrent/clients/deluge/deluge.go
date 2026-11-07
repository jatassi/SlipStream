// Package deluge speaks Deluge's Web UI JSON-RPC at /json, 2.1 and later.
// A session signs in with auth.login and the password, which sets a
// _session_id cookie, then connects the web UI with web.connect to the
// chosen daemon, looked up by host and port in web.get_hosts at each
// connect, or to the only daemon it lists. It signs in and connects again
// on error code 1, no session, or 2, a method the web UI doesn't know,
// which every daemon method is until web.connect registers them. Before
// each poll it connects again a web UI that lost its daemon and, where the
// web UI lists more than one, checks it is connected to the chosen one.
package deluge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// limit bounds a response, which holds every torrent a poll asks for.
const limit = 32 << 20

// Open starts a session with the Deluge web UI at s.Address. It does no IO.
func Open(s clients.Session) clients.Client { return &client{s: s} }

type client struct {
	s       clients.Session
	cookies []*http.Cookie
	nextID  int
	// signedIn is a session signed in and connected to its daemon.
	signedIn bool
	version  clients.Version
	// host is the web UI's id for the daemon it connected to, and check a
	// web UI that lists more than one daemon and can say which it is
	// connected to.
	host  string
	check bool
}

// rpcError is a JSON-RPC error: 1 not authenticated, 2 unknown method, 3
// a web UI method's exception, 4 a daemon's failure and 5 a request the
// web UI couldn't decode.
type rpcError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("Deluge error %d: %s", e.Code, e.Message) }

const (
	notAuthenticated = 1
	unknownMethod    = 2
	daemonFailed     = 4
)

// failedWith is err as a daemon's failure whose words hold text.
func failedWith(err error, text string) bool {
	e, ok := errors.AsType[*rpcError](err)
	return ok && e.Code == daemonFailed && strings.Contains(e.Message, text)
}

// send is one JSON-RPC request, its result decoded into out. Deluge
// answers HTTP 200 with the error in the body.
func (c *client) send(ctx context.Context, method string, params []any, out any) error {
	c.nextID++
	body, err := json.Marshal(map[string]any{"method": method, "params": params, "id": c.nextID})
	if err != nil {
		return fmt.Errorf("build %s: %w", method, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.s.Address.Join(""), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, k := range c.cookies {
		req.AddCookie(k)
	}
	res, err := clients.Do(ctx, c.s, req, limit)
	if err != nil {
		return err
	}
	for _, k := range (&http.Response{Header: res.Header}).Cookies() {
		c.cookies = append(slices.DeleteFunc(c.cookies, func(old *http.Cookie) bool { return old.Name == k.Name }), k)
	}
	if res.Status != http.StatusOK {
		return fmt.Errorf("the client responded to %s with status %d", method, res.Status)
	}
	var reply struct {
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
	}
	if err := json.Unmarshal(res.Body, &reply); err != nil {
		return fmt.Errorf("read the client's %s: %w", method, err)
	}
	if reply.Error != nil {
		return reply.Error
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(reply.Result, out); err != nil {
		return fmt.Errorf("read the client's %s: %w", method, err)
	}
	return nil
}

// call is a method on a signed-in session. Code 1 or 2 signs in and
// connects again, then sends the method once more. Code 2 again for a
// Label plugin method is the plugin off.
func (c *client) call(ctx context.Context, method string, params []any, out any) error {
	if !c.signedIn {
		if _, err := c.signIn(ctx); err != nil {
			return err
		}
	}
	err := c.send(ctx, method, params, out)
	if e, ok := errors.AsType[*rpcError](err); !ok || e.Code != notAuthenticated && e.Code != unknownMethod {
		return err
	}
	if _, err := c.signIn(ctx); err != nil {
		return err
	}
	err = c.send(ctx, method, params, out)
	if e, ok := errors.AsType[*rpcError](err); ok && e.Code == unknownMethod && strings.HasPrefix(method, "label.") {
		return &clients.Problem{Cause: clients.LabelPluginOff}
	}
	return err
}

// signIn is auth.login, then the connect. It answers the daemons the web
// UI lists once it has read them.
func (c *client) signIn(ctx context.Context) ([]clients.Daemon, error) {
	c.signedIn = false
	var ok bool
	if err := c.send(ctx, "auth.login", []any{c.s.Secrets.Password}, &ok); err != nil {
		return nil, err
	}
	if !ok {
		return nil, &clients.Problem{Cause: clients.WrongCredentials}
	}
	return c.connect(ctx)
}

// connect is web.connect to the session's daemon, looked up in
// web.get_hosts, then daemon.get_version.
func (c *client) connect(ctx context.Context) ([]clients.Daemon, error) {
	var hosts [][]json.RawMessage
	if err := c.send(ctx, "web.get_hosts", []any{}, &hosts); err != nil {
		return nil, err
	}
	daemons := make([]clients.Daemon, len(hosts))
	ids := make([]string, len(hosts))
	for i, h := range hosts {
		if len(h) < 3 || json.Unmarshal(h[0], &ids[i]) != nil || json.Unmarshal(h[1], &daemons[i].Host) != nil || json.Unmarshal(h[2], &daemons[i].Port) != nil {
			return nil, errors.New("read the client's web.get_hosts: a host isn't an id, a host and a port")
		}
	}
	at := 0
	switch {
	case c.s.Daemon != nil:
		at = slices.Index(daemons, *c.s.Daemon)
		if at < 0 {
			return daemons, &clients.Problem{Cause: clients.WrongDaemon}
		}
	case len(daemons) == 0:
		return daemons, errors.New("the web UI lists no daemon")
	case len(daemons) > 1:
		return daemons, &clients.Problem{Cause: clients.DaemonNeeded}
	}
	var methods []string
	if err := c.send(ctx, "web.connect", []any{ids[at]}, &methods); err != nil {
		return daemons, err
	}
	d := daemons[at]
	if methods == nil {
		return daemons, &clients.Problem{Cause: clients.NoResponse, Err: fmt.Errorf("the web UI couldn't connect to the daemon at %s:%d", d.Host, d.Port)}
	}
	var version string
	if err := c.send(ctx, "daemon.get_version", []any{}, &version); err != nil {
		return daemons, err
	}
	v, err := clients.ParseVersion(version)
	if err != nil {
		return daemons, fmt.Errorf("read the client's daemon.get_version: %w", err)
	}
	_, byAddress := netip.ParseAddr(d.Host)
	c.version, c.host, c.check, c.signedIn = v, ids[at], len(daemons) > 1 && byAddress == nil, true
	return daemons, nil
}

// onDaemon connects again a web UI that lost its daemon. A web UI that
// lists more than one daemon and is connected to another one than the
// session's is WrongDaemon. Deluge 2.1 reads Connected only for a daemon
// it lists by IP address, so a daemon listed by name goes unchecked.
func (c *client) onDaemon(ctx context.Context) error {
	var connected bool
	if err := c.call(ctx, "web.connected", []any{}, &connected); err != nil {
		return err
	}
	if !connected {
		_, err := c.connect(ctx)
		return err
	}
	if !c.check {
		return nil
	}
	var status []json.RawMessage
	if err := c.call(ctx, "web.get_host_status", []any{c.host}, &status); err != nil {
		return err
	}
	var state string
	if len(status) < 2 || json.Unmarshal(status[1], &state) != nil {
		return errors.New("read the client's web.get_host_status: no status")
	}
	if state != "Connected" {
		return &clients.Problem{Cause: clients.WrongDaemon}
	}
	return nil
}

func (c *client) Version(ctx context.Context) (clients.Version, error) {
	if !c.signedIn {
		if _, err := c.signIn(ctx); err != nil {
			return clients.Version{}, err
		}
	}
	return c.version, nil
}

// Close ends the session with auth.delete_session and drops it.
func (c *client) Close(ctx context.Context) error {
	var err error
	if c.signedIn {
		err = c.send(ctx, "auth.delete_session", []any{}, nil)
	}
	*c = client{s: c.s}
	return err
}
