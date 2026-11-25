// Package plextv is the wire client for plex.tv: the device's Ed25519 key
// and its public JWK, the strong PIN on clients.plex.tv and its poll with a
// device JWT, the account from /api/v2/user, the servers from resources,
// and the renewal of the 7-day Plex JWT with a nonce. Every call carries
// the X-Plex-* headers with the stored client identifier and asks for
// JSON. Written against plex.tv as the recorded run of 2026-11-14 met it:
// the PIN poll needs no nonce, the PIN answers authToken and the renewal
// auth_token, and a server's resources token works only on that server.
// The Admin's sharing calls and a Member's own device are as the Plex run
// of 2026-11-25 met them (recorded/plex-members-run-2026-11-25.md).
package plextv

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

type Endpoints struct {
	API endpoint.URL `yaml:"api"`
	// Web is plex.tv, where the run of 2026-11-25 measured the share POST.
	Web endpoint.URL `yaml:"web"`
	// Auth is the sign-in page, opened in the browser only.
	Auth endpoint.URL `yaml:"auth"`
	// Desktop is Plex's web app, which Open in links to in the browser only.
	Desktop endpoint.URL `yaml:"desktop"`
}

func DefaultEndpoints() Endpoints {
	return Endpoints{
		API:     endpoint.MustParseURL("https://clients.plex.tv"),
		Web:     endpoint.MustParseURL("https://plex.tv"),
		Auth:    endpoint.MustParseURL("https://app.plex.tv/auth"),
		Desktop: endpoint.MustParseURL("https://app.plex.tv/desktop"),
	}
}

// deviceJWTLife is how long a device JWT is good for, as PLEXAPI and the
// recorded run sign it.
const deviceJWTLife = 5 * time.Minute

// renewScope is the scope a renewal asks for, as PLEXAPI asks it.
const renewScope = "username,email,friendly_name"

// Client is plex.tv for one device: its key signs every device JWT, and
// h.ClientID is the client identifier plex.tv knows the key by.
type Client struct {
	t   transport
	key ed25519.PrivateKey
}

func New(e Endpoints, h plex.Headers, key ed25519.PrivateKey, c *http.Client) *Client {
	return &Client{t: transport{e: e, h: h, http: c}, key: key}
}

// transport is plex.tv's wire for one client identifier, which the
// Admin's Client and a Member's client share.
type transport struct {
	e    Endpoints
	h    plex.Headers
	http *http.Client
}

// JWK is the device key's public half as plex.tv takes it.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

func (c *Client) jwk() JWK {
	pub, _ := c.key.Public().(ed25519.PublicKey)
	x := base64.RawURLEncoding.EncodeToString(pub)
	return JWK{Kty: "OKP", Crv: "Ed25519", X: x, Kid: keyID(x), Alg: "EdDSA", Use: "sig"}
}

// keyID is the RFC 7638 thumbprint of an Ed25519 key, which plex.tv takes
// as the kid.
func keyID(x string) string {
	sum := sha256.Sum256([]byte(`{"crv":"Ed25519","kty":"OKP","x":"` + x + `"}`))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// deviceClaims are a device JWT's claims: the audience, issuer and
// five-minute life every one carries, and a renewal's nonce and scope.
type deviceClaims struct {
	Aud   string `json:"aud"`
	Iss   string `json:"iss"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
	Nonce string `json:"nonce,omitempty"`
	Scope string `json:"scope,omitempty"`
}

type jwsHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

// sign is a compact EdDSA JWS under the device key, with nonce and scope
// for a renewal and neither for a PIN poll.
func (c *Client) sign(nonce, scope string) (string, error) {
	now := time.Now()
	header, err := json.Marshal(jwsHeader{Alg: "EdDSA", Kid: c.jwk().Kid, Typ: "JWT"})
	if err != nil {
		return "", fmt.Errorf("encode the device JWT header: %w", err)
	}
	payload, err := json.Marshal(deviceClaims{Aud: "plex.tv", Iss: c.t.h.ClientID, Iat: now.Unix(), Exp: now.Add(deviceJWTLife).Unix(), Nonce: nonce, Scope: scope})
	if err != nil {
		return "", fmt.Errorf("encode the device JWT claims: %w", err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return input + "." + base64.RawURLEncoding.EncodeToString(ed25519.Sign(c.key, []byte(input))), nil
}

// Error is a response other than 2xx.
type Error struct {
	Status int
	// RetryAfter is the wait a 429 asked for, zero when it named none.
	RetryAfter time.Duration
	// Detail is plex.tv's first error message, or the start of the body.
	Detail string
}

func (e *Error) Error() string {
	return fmt.Sprintf("plex.tv responded with status %d: %s", e.Status, e.Detail)
}

func status(err error) int {
	e, ok := errors.AsType[*Error](err)
	if !ok {
		return 0
	}
	return e.Status
}

// Refused is whether plex.tv refused the account's sign-in: a 401 to a
// JWT it no longer takes, or a 422 to a renewal after sign-out.
func Refused(err error) bool {
	s := status(err)
	return s == http.StatusUnauthorized || s == http.StatusUnprocessableEntity
}

// Declined is whether plex.tv said no to a sharing call, with its reason
// in Error.Detail: a Home invite without Plex Pass or into a full Home is
// 403, a second share to one account 400.
func Declined(err error) bool {
	s := status(err)
	return s == http.StatusBadRequest || s == http.StatusForbidden || s == http.StatusUnprocessableEntity
}

// Expired is whether a PIN poll found the PIN gone: past its expiresIn,
// or never created.
func Expired(err error) bool { return status(err) == http.StatusNotFound }

// RateLimited is plex.tv's 429 and the wait it asked for, zero when it
// named none.
func RateLimited(err error) (time.Duration, bool) {
	e, ok := errors.AsType[*Error](err)
	if !ok || e.Status != http.StatusTooManyRequests {
		return 0, false
	}
	return e.RetryAfter, true
}

// request is one call to route: the values of its path's {name}s in
// pairs, the query, the token, and a body sent as JSON.
type request struct {
	route route
	vars  []string
	query url.Values
	token string
	body  any
}

// do sends req and decodes the response into out, nil for a response with
// no body.
func (t transport) do(ctx context.Context, req request, out any) error {
	r := req.route
	ref, err := url.Parse(strings.NewReplacer(req.vars...).Replace(r.path))
	if err != nil {
		return fmt.Errorf("plex.tv %s %s: %w", r.method, r.path, err)
	}
	u := r.host(t.e).JoinPath(ref.Path)
	q := ref.Query()
	for k, v := range req.query {
		q[k] = v
	}
	u.RawQuery = q.Encode()
	var in io.Reader = http.NoBody
	if req.body != nil {
		b, err := json.Marshal(req.body)
		if err != nil {
			return fmt.Errorf("plex.tv %s %s: %w", r.method, ref.Path, err)
		}
		in = bytes.NewReader(b)
	}
	hr, err := http.NewRequestWithContext(ctx, r.method, u.String(), in)
	if err != nil {
		return fmt.Errorf("plex.tv %s %s: %w", r.method, ref.Path, err)
	}
	t.header(hr.Header, r.format)
	if req.body != nil {
		hr.Header.Set("Content-Type", "application/json")
	}
	if req.token != "" {
		hr.Header.Set("X-Plex-Token", req.token)
	}
	resp, err := t.http.Do(hr)
	if err != nil {
		return fmt.Errorf("plex.tv %s %s: %w", r.method, ref.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("plex.tv %s %s: read the response: %w", r.method, ref.Path, err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("plex.tv %s %s: %w", r.method, ref.Path, responseError(resp, raw))
	}
	if out == nil {
		return nil
	}
	decode := json.Unmarshal
	if r.format == formatXML {
		decode = xml.Unmarshal
	}
	if err := decode(raw, out); err != nil {
		return fmt.Errorf("plex.tv %s %s: decode the response: %w", r.method, ref.Path, err)
	}
	return nil
}

func (t transport) header(h http.Header, f format) {
	h.Set("Accept", map[format]string{formatJSON: "application/json", formatXML: "application/xml"}[f])
	for k, v := range map[string]string{
		"X-Plex-Client-Identifier": t.h.ClientID,
		"X-Plex-Product":           t.h.Product,
		"X-Plex-Version":           t.h.Version,
		"X-Plex-Platform":          t.h.Platform,
		"X-Plex-Platform-Version":  t.h.PlatformVersion,
		"X-Plex-Device":            t.h.Device,
		"X-Plex-Device-Name":       t.h.DeviceName,
	} {
		if v != "" {
			h.Set(k, v)
		}
	}
}

func responseError(resp *http.Response, raw []byte) *Error {
	e := &Error{Status: resp.StatusCode}
	if s, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
		e.RetryAfter = time.Duration(s) * time.Second
	}
	var b struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(raw, &b) == nil && len(b.Errors) > 0 {
		e.Detail = b.Errors[0].Message
		return e
	}
	e.Detail = strings.TrimSpace(string(raw))
	if len(e.Detail) > 200 {
		e.Detail = e.Detail[:200]
	}
	return e
}

// Pin is a strong PIN, the code the Admin approves on the auth page.
type Pin struct {
	ID        int64
	Code      string
	ExpiresAt time.Time
}

type pinBody struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
	AuthToken *string   `json:"authToken"`
}

// CreatePin is a new strong PIN tied to the device's public key.
func (c *Client) CreatePin(ctx context.Context) (Pin, error) {
	var out pinBody
	err := c.t.do(ctx, request{route: routes[pin], body: map[string]any{"jwk": c.jwk(), "strong": true}}, &out)
	return Pin{ID: out.ID, Code: out.Code, ExpiresAt: out.ExpiresAt}, err
}

// AuthURL is the page where the Admin approves code. Afterwards plex.tv
// opens forward again with no query string, so the page there can't learn
// the PIN from its address.
func (c *Client) AuthURL(code, forward string) string { return c.t.authURL(code, forward) }

func (t transport) authURL(code, forward string) string {
	q := url.Values{"clientID": {t.h.ClientID}, "code": {code}, "forwardUrl": {forward}, "context[device][product]": {t.h.Product}}
	return t.e.Auth.String() + "#?" + q.Encode()
}

// PollPin is the Plex JWT once the Admin approved the PIN, and false while
// it waits. A PIN past its life is an error Expired reads.
func (c *Client) PollPin(ctx context.Context, id int64) (Token, bool, error) {
	device, err := c.sign("", "")
	if err != nil {
		return Token{}, false, err
	}
	var out pinBody
	if err := c.t.do(ctx, request{route: routes[pollPin], vars: pinVars(id), query: url.Values{"deviceJWT": {device}}}, &out); err != nil {
		return Token{}, false, err
	}
	if out.AuthToken == nil {
		return Token{}, false, nil
	}
	t, err := parseToken(*out.AuthToken)
	return t, err == nil, err
}

// Token is a Plex JWT, which only plex.tv takes, and when it expires.
type Token struct {
	JWT     string
	Expires time.Time
}

// parseToken reads exp from a Plex JWT. plex.tv signed it, and only
// plex.tv checks the signature.
func parseToken(jwt string) (Token, error) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return Token{}, errors.New("plex.tv sent a token that isn't a JWT")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Token{}, fmt.Errorf("decode the Plex JWT claims: %w", err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return Token{}, fmt.Errorf("decode the Plex JWT claims: %w", err)
	}
	return Token{JWT: jwt, Expires: time.Unix(claims.Exp, 0)}, nil
}

func pinVars(id int64) []string { return []string{"{pin}", strconv.FormatInt(id, 10)} }

// User is the signed-in account.
type User struct {
	ID       AccountID
	UUID     string
	Username string
	Email    string
	Title    string
	PlexPass bool
}

func (c *Client) User(ctx context.Context, jwt string) (User, error) { return c.t.user(ctx, jwt) }

func (t transport) user(ctx context.Context, token string) (User, error) {
	var out struct {
		ID           AccountID `json:"id"`
		UUID         string    `json:"uuid"`
		Username     string    `json:"username"`
		Email        string    `json:"email"`
		Title        string    `json:"title"`
		Subscription struct {
			Active bool `json:"active"`
		} `json:"subscription"`
	}
	err := t.do(ctx, request{route: routes[user], token: token}, &out)
	return User{ID: out.ID, UUID: out.UUID, Username: out.Username, Email: out.Email, Title: out.Title, PlexPass: out.Subscription.Active}, err
}

// Server is a Plex Media Server the account can reach.
type Server struct {
	Name string
	// Identifier is the server's machineIdentifier.
	Identifier string
	Version    string
	Owned      bool
	// Token is the server's own accessToken. The server takes it and the
	// account's JWT it refuses, while plex.tv refuses it.
	Token string
	// PublicAddress is the address plex.tv last saw the server at, and
	// PublicAddressMatches whether this client's is the same.
	PublicAddress        string
	PublicAddressMatches bool
	// Connections are in plex.tv's order, relays included.
	Connections []Connection
}

type Connection struct {
	URI *url.URL
	// Protocol, Address and Port are what plex.tv lists apart from URI:
	// Address is an IP, IPv6 included, or a custom URL's host name.
	Protocol string
	Address  string
	Port     uint16
	Local    bool
	Relay    bool
	IPv6     bool
}

// Servers are the account's servers from resources, asked with HTTPS,
// relay and IPv6 connections, its other devices left out.
func (c *Client) Servers(ctx context.Context, jwt string) ([]Server, error) {
	var out []struct {
		Name                 string `json:"name"`
		ProductVersion       string `json:"productVersion"`
		ClientIdentifier     string `json:"clientIdentifier"`
		Provides             string `json:"provides"`
		Owned                bool   `json:"owned"`
		AccessToken          string `json:"accessToken"`
		PublicAddress        string `json:"publicAddress"`
		PublicAddressMatches bool   `json:"publicAddressMatches"`
		Connections          []struct {
			URI      string `json:"uri"`
			Protocol string `json:"protocol"`
			Address  string `json:"address"`
			Port     uint16 `json:"port"`
			Local    bool   `json:"local"`
			Relay    bool   `json:"relay"`
			IPv6     bool   `json:"IPv6"`
		} `json:"connections"`
	}
	q := url.Values{"includeHttps": {"1"}, "includeRelay": {"1"}, "includeIPv6": {"1"}}
	if err := c.t.do(ctx, request{route: routes[resources], query: q, token: jwt}, &out); err != nil {
		return nil, err
	}
	var servers []Server
	for _, r := range out {
		if !strings.Contains(r.Provides, "server") {
			continue
		}
		s := Server{
			Name: r.Name, Identifier: r.ClientIdentifier, Version: r.ProductVersion, Owned: r.Owned, Token: r.AccessToken,
			PublicAddress: r.PublicAddress, PublicAddressMatches: r.PublicAddressMatches,
		}
		for _, conn := range r.Connections {
			u, err := url.Parse(conn.URI)
			if err != nil {
				return nil, fmt.Errorf("plex.tv listed server %s with connection %q: %w", r.Name, conn.URI, err)
			}
			s.Connections = append(s.Connections, Connection{
				URI: u, Protocol: conn.Protocol, Address: conn.Address, Port: conn.Port, Local: conn.Local, Relay: conn.Relay, IPv6: conn.IPv6,
			})
		}
		servers = append(servers, s)
	}
	return servers, nil
}

// Renew is a new Plex JWT for the device: a nonce, then the token
// exchange signed with the device key. It works after the last JWT
// expired, and plex.tv refuses it with a 422 once the Admin signed the
// device out.
func (c *Client) Renew(ctx context.Context) (Token, error) {
	var nonce struct {
		Nonce string `json:"nonce"`
	}
	if err := c.t.do(ctx, request{route: routes[authNonce]}, &nonce); err != nil {
		return Token{}, err
	}
	var out struct {
		AuthToken string `json:"auth_token"`
	}
	jwt, err := c.sign(nonce.Nonce, renewScope)
	if err != nil {
		return Token{}, err
	}
	if err := c.t.do(ctx, request{route: routes[authToken], body: map[string]string{"jwt": jwt}}, &out); err != nil {
		return Token{}, err
	}
	return parseToken(out.AuthToken)
}
