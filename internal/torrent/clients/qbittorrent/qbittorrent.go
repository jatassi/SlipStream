// Package qbittorrent speaks qBittorrent's WebUI API v2, 4.6 and later. A
// session signs in with an API key as a Bearer header (5.2 and later), or
// with auth/login, which answers 200 "Ok." with an SID cookie before 5.2
// and 204 with a QBT_SID_<port> cookie from 5.2. It reads app/version and
// app/webapiVersion at each sign-in, and speaks stop and start from WebAPI
// 2.11.0, pause and resume before.
package qbittorrent

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

const (
	apiDir = "/api/v2/"
	// limit bounds a response, which holds at most one batch of torrents.
	limit = 32 << 20
)

// stopSince is the WebAPI version that renamed pause and resume to stop
// and start, the add parameter paused to stopped and the paused* states
// to stopped*.
var stopSince, _ = clients.ParseVersion("2.11.0")

// Open starts a session with the qBittorrent at s.Address. It does no IO.
func Open(s clients.Session) clients.Client { return open(s) }

func open(s clients.Session) *client {
	return &client{s: s, ran: map[bencode.InfoHash]clients.Run{}, trackers: map[bencode.InfoHash]trackers{}}
}

type client struct {
	s clients.Session
	// cookies are the session's, from auth/login or from a response to an
	// address qBittorrent lets in without signing in.
	cookies []*http.Cookie
	// version is app/version and api app/webapiVersion, read once after
	// each sign-in, while read is set.
	version, api clients.Version
	read         bool
	// ran is what each torrent was last seen or asked to do, since
	// qBittorrent reports a torrent it checks as checking whether it was
	// stopped or started.
	ran      map[bencode.InfoHash]clients.Run
	trackers map[bencode.InfoHash]trackers
}

// trackers is one torrent's tracker messages and when they were read.
type trackers struct {
	at       time.Time
	messages []clients.TrackerMessage
}

// request is one API call: its method, its path under /api/v2/, and its
// body and the body's content type, rebuilt for each send.
type request struct {
	method, call string
	query        url.Values
	body         func() (io.Reader, string, error)
}

func get(call string, query url.Values) request {
	return request{method: http.MethodGet, call: call, query: query}
}

func post(call string, form url.Values) request {
	return request{method: http.MethodPost, call: call, body: func() (io.Reader, string, error) {
		return strings.NewReader(form.Encode()), "application/x-www-form-urlencoded", nil
	}}
}

func (c *client) build(ctx context.Context, rq request) (*http.Request, error) {
	u := c.s.Address.Join(apiDir + rq.call)
	if len(rq.query) > 0 {
		u += "?" + rq.query.Encode()
	}
	var body io.Reader
	var contentType string
	if rq.body != nil {
		var err error
		if body, contentType, err = rq.body(); err != nil {
			return nil, fmt.Errorf("build %s: %w", rq.call, err)
		}
	}
	req, err := http.NewRequestWithContext(ctx, rq.method, u, body)
	if err != nil {
		return nil, fmt.Errorf("build %s: %w", rq.call, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.s.Secrets.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.s.Secrets.APIKey)
	}
	for _, k := range c.cookies {
		req.AddCookie(k)
	}
	return req, nil
}

// do sends rq. A 403 is a session that expired or never started: do
// signs in once and sends rq again, its body rebuilt. With an API key a
// 403 is the key refused. A 401 is the host or CSRF check, which
// qBittorrent runs before it looks at the session. It sends no Origin or
// Referer header, so the CSRF check passes.
func (c *client) do(ctx context.Context, rq request) (clients.Response, error) {
	for signedIn := false; ; signedIn = true {
		req, err := c.build(ctx, rq)
		if err != nil {
			return clients.Response{}, err
		}
		res, err := clients.Do(ctx, c.s, req, limit)
		if err != nil {
			return clients.Response{}, err
		}
		c.keep(res)
		switch res.Status {
		case http.StatusUnauthorized:
			host, _, _ := net.SplitHostPort(c.s.Address.HostPort())
			return clients.Response{}, &clients.Problem{Cause: clients.HostRefused, Host: host}
		case http.StatusForbidden:
			switch {
			case c.s.Secrets.APIKey != "":
				return clients.Response{}, &clients.Problem{Cause: clients.WrongAPIKey}
			case signedIn:
				return clients.Response{}, fmt.Errorf("the client refused %s on the session it just started", rq.call)
			}
			if err := c.login(ctx); err != nil {
				return clients.Response{}, err
			}
			continue
		}
		return res, nil
	}
}

func (c *client) keep(res clients.Response) {
	for _, k := range (&http.Response{Header: res.Header}).Cookies() {
		c.cookies = append(slices.DeleteFunc(c.cookies, func(old *http.Cookie) bool { return old.Name == k.Name }), k)
	}
}

// login is auth/login. A 403 from it is the ban. Bad credentials are 200
// "Fails." before 5.2 and 401 from 5.2, which is never the host check
// here, since do reached login past it.
func (c *client) login(ctx context.Context) error {
	req, err := c.build(ctx, post("auth/login", url.Values{"username": {c.s.Username}, "password": {c.s.Secrets.Password}}))
	if err != nil {
		return err
	}
	res, err := clients.Do(ctx, c.s, req, limit)
	if err != nil {
		return err
	}
	switch {
	case res.Status == http.StatusForbidden:
		return &clients.Problem{Cause: clients.Banned}
	case res.Status == http.StatusUnauthorized, res.Status == http.StatusOK && string(res.Body) == "Fails.":
		return &clients.Problem{Cause: clients.WrongCredentials}
	case res.Status == http.StatusOK, res.Status == http.StatusNoContent:
		c.keep(res)
		c.read = false
		return nil
	}
	return fmt.Errorf("the client responded to its sign-in with status %d", res.Status)
}

// text is a call whose 200 response is plain text.
func (c *client) text(ctx context.Context, rq request) (string, error) {
	res, err := c.do(ctx, rq)
	if err != nil {
		return "", err
	}
	if res.Status != http.StatusOK {
		return "", fmt.Errorf("the client responded to %s with status %d", rq.call, res.Status)
	}
	return string(res.Body), nil
}

// versions reads app/version and app/webapiVersion once a sign-in.
func (c *client) versions(ctx context.Context) error {
	if c.read {
		return nil
	}
	var v [2]clients.Version
	for i, call := range []string{"app/version", "app/webapiVersion"} {
		s, err := c.text(ctx, get(call, nil))
		if err != nil {
			return err
		}
		if v[i], err = clients.ParseVersion(strings.TrimSpace(s)); err != nil {
			return fmt.Errorf("read the client's %s: %w", call, err)
		}
	}
	c.version, c.api, c.read = v[0], v[1], true
	return nil
}

// renamed is a session past WebAPI 2.11.0's rename of pause to stop.
func (c *client) renamed() bool { return !c.api.Less(stopSince) }

func (c *client) Version(ctx context.Context) (clients.Version, error) {
	if err := c.versions(ctx); err != nil {
		return clients.Version{}, err
	}
	return c.version, nil
}

// Close signs out of a cookie session and drops the session.
func (c *client) Close(ctx context.Context) error {
	var err error
	if len(c.cookies) > 0 {
		var req *http.Request
		if req, err = c.build(ctx, post("auth/logout", nil)); err == nil {
			_, err = clients.Do(ctx, c.s, req, limit)
		}
	}
	*c = *open(c.s)
	return err
}
