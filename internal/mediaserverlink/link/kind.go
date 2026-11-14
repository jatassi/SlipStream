// Package link is the kind-neutral vocabulary of Media Servers: the Kind
// and Conn a product package implements, the Jobs the Controller plans and
// a reaching Node runs one step at a time, the one path type a server sees,
// and the rules every kind's Plan shares. It is a leaf: the kind packages,
// the holder and the Controller all import it, and it imports none of them.
package link

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

// KindName is a Media Server product as it is stored, sent and shown in the
// API: "plex", "jellyfin" or "emby".
type KindName string

// SignIn is the form the Add sheet shows for a kind.
type SignIn string

const (
	SignInPlexPIN    SignIn = "plex_pin"
	SignInKeyOrAdmin SignIn = "api_key_or_admin"
)

// About is what the API and the web learn about a kind, so neither lists
// kinds.
type About struct {
	Name  KindName
	Label string
	Form  SignIn
	// Browser is whether the kind takes a browser address for Open in.
	Browser bool
	// Floor is the oldest version Save accepts.
	Floor Version
}

// Kind is everything SlipStream knows about one Media Server product.
type Kind interface {
	About() About
	// Plan turns a server's due rows into Jobs, in the order they run. Pure.
	Plan(d Due) []Job
	// Dial opens a connection to one server. It does no IO.
	Dial(d Dialed) Conn
	// OpenIn is the address of item on the server identity names, in the
	// server's web app at app.
	OpenIn(app, identity, item string) string
}

// Registry is every kind, in the order the API lists them.
type Registry []Kind

func (r Registry) Of(n KindName) (Kind, bool) {
	i := slices.IndexFunc(r, func(k Kind) bool { return k.About().Name == n })
	if i < 0 {
		return nil, false
	}
	return r[i], true
}

func (r Registry) Names() []KindName {
	out := make([]KindName, len(r))
	for i, k := range r {
		out[i] = k.About().Name
	}
	return out
}

func (r Registry) Abouts() []About {
	out := make([]About, len(r))
	for i, k := range r {
		out[i] = k.About()
	}
	return out
}

// Dialed is one server as a Node dials it.
type Dialed struct {
	// Server is zero for a draft that Reach, Check or SignIn tests before
	// Save.
	Server     ServerRef
	Address    string
	Credential Credential
	// DeviceID is the server's Jellyfin or Emby DeviceId, or the Plex
	// account's client identifier.
	DeviceID     string
	MonitorDelay time.Duration
	HTTP         *http.Client
	Timings      Timings
}

// Checked is what Check read with the credential.
type Checked struct {
	Identity Identity
	// MonitorDelay is Jellyfin's or Emby's library monitor delay, zero on
	// Plex.
	MonitorDelay time.Duration
	Libraries    []Library
}

// Conn is one server as one Node reaches it.
type Conn interface {
	// Reach reads the server's identity with no credential: Plex /identity,
	// Jellyfin and Emby /System/Info/Public.
	Reach(ctx context.Context) (Identity, error)
	// Check reads identity, version and libraries with the credential, which
	// proves the server accepts it.
	Check(ctx context.Context) (Checked, error)
	// SignIn creates a key named SlipStream with an administrator's name and
	// password, signs that session out and keeps neither. Jellyfin and Emby
	// only.
	SignIn(ctx context.Context, user, password string) (Credential, error)
	// Advance runs exactly one step of j.
	Advance(ctx context.Context, j Job) (Advanced, error)
	// Watch keeps the push socket open and polls sessions every every until
	// ctx ends, calling seen for each poll and each push.
	Watch(ctx context.Context, every time.Duration, seen func(Seen)) error
}

// Identity is who responds at an address.
type Identity struct {
	// ID is Plex's machineIdentifier or the Jellyfin or Emby server Id.
	ID      string
	Name    string
	Version Version
}

// Credential is a server's secret: Plex's per-server resources token, or a
// Jellyfin or Emby API key. It never appears in a log or an error.
type Credential struct {
	Token  string
	APIKey string
}

func (Credential) LogValue() slog.Value { return slog.StringValue("[credential]") }

func (Credential) String() string { return "[credential]" }

func (Credential) GoString() string { return "[credential]" }
