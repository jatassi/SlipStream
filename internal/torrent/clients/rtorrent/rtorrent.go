// Package rtorrent speaks rTorrent's XML-RPC over HTTP, usually nginx's
// /RPC2 in front of rTorrent's SCGI port, 0.9.8 and later. Every call
// takes a target first: the info hash for a torrent's d.* and f.* calls,
// the empty string otherwise. Basic auth is nginx's, and rTorrent holds
// no session, so a session is only the address and the credentials.
package rtorrent

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// limit bounds a response, which holds every torrent a poll asks for.
const limit = 32 << 20

// Open starts a session with the rTorrent at s.Address. It does no IO.
func Open(s clients.Session) clients.Client { return &client{s: s} }

type client struct {
	s clients.Session
	// version is the version the last Test or Version read, zero before.
	version clients.Version
}

// send is one XML-RPC request. A 401 is nginx refusing the username and
// password.
func (c *client) send(ctx context.Context, method string, params []any) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.s.Address.String(), bytes.NewReader(encode(method, params)))
	if err != nil {
		return nil, fmt.Errorf("build %s: %w", method, err)
	}
	req.Header.Set("Content-Type", "text/xml")
	if c.s.Username != "" || c.s.Secrets.Password != "" {
		req.SetBasicAuth(c.s.Username, c.s.Secrets.Password)
	}
	res, err := clients.Do(ctx, c.s, req, limit)
	if err != nil {
		return nil, err
	}
	switch res.Status {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, &clients.Problem{Cause: clients.WrongCredentials}
	default:
		return nil, fmt.Errorf("the client responded to %s with status %d", method, res.Status)
	}
	v, err := decode(res.Body)
	if _, ok := errors.AsType[*fault](err); err != nil && !ok {
		return nil, fmt.Errorf("read the client's %s: %w", method, err)
	}
	return v, err
}

// multicall is one system.multicall: each call's value, or its *fault.
func (c *client) multicall(ctx context.Context, calls []call) ([]any, error) {
	list := make([]any, len(calls))
	for i, k := range calls {
		list[i] = k
	}
	v, err := c.send(ctx, "system.multicall", []any{list})
	if err != nil {
		return nil, err
	}
	results, ok := v.([]any)
	if !ok || len(results) != len(calls) {
		return nil, fmt.Errorf("the client responded to a system.multicall of %d calls with %v", len(calls), v)
	}
	for i, r := range results {
		if f, ok := asFault(r); ok {
			results[i] = f
			continue
		}
		one, ok := r.([]any)
		if !ok || len(one) != 1 {
			return nil, fmt.Errorf("the client responded to %s in a system.multicall with %v", calls[i].method, r)
		}
		results[i] = one[0]
	}
	return results, nil
}

// missing is a fault for a target hash rTorrent doesn't hold, in the
// words of xmlrpc-c's build before 0.16 and of RpcManager from 0.16.
func missing(err error) bool {
	f, ok := errors.AsType[*fault](err)
	return ok && (f.Message == "Could not find info-hash." || f.Message == "invalid parameters: info-hash not found")
}

func target(h bencode.InfoHash) string { return strings.ToUpper(h.String()) }

func (c *client) text(ctx context.Context, method string) (string, error) {
	v, err := c.send(ctx, method, []any{""})
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("the client responded to %s with %v, not a string", method, v)
	}
	return s, nil
}

func (c *client) readVersion(ctx context.Context) error {
	s, err := c.text(ctx, "system.client_version")
	if err != nil {
		return err
	}
	v, err := clients.ParseVersion(s)
	if err != nil {
		return fmt.Errorf("read the client's version: %w", err)
	}
	c.version = v
	return nil
}

// Test reads the version. rTorrent takes any label, and a Test changes
// nothing.
func (c *client) Test(ctx context.Context, _ clients.Label) (clients.Tested, error) {
	if err := c.readVersion(ctx); err != nil {
		return clients.Tested{}, err
	}
	if c.version.Less(clients.RTorrent.Facts().Floor) {
		return clients.Tested{}, &clients.Problem{Cause: clients.VersionTooOld, Version: c.version}
	}
	return clients.Tested{Version: c.version}, nil
}

func (c *client) Version(ctx context.Context) (clients.Version, error) {
	if c.version == (clients.Version{}) {
		if err := c.readVersion(ctx); err != nil {
			return clients.Version{}, err
		}
	}
	return c.version, nil
}

// DefaultFolder is directory.default, since rTorrent has no label folders.
func (c *client) DefaultFolder(ctx context.Context, _ clients.Label) (string, error) {
	return c.text(ctx, "directory.default")
}

// Close drops what the session read. rTorrent has no sign-out.
func (c *client) Close(context.Context) error {
	*c = client{s: c.s}
	return nil
}
