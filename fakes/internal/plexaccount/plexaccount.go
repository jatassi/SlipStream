// Package plexaccount is the plex.tv account of one world: the fake-plex
// servers it owns, which fake-plex-tv lists in resources, and the server
// tokens fake-plex-tv issued, which fake-plex takes until a sign-out
// revokes them, and what each server publishes for use away from home:
// fake-plex answers it in /myplex/account and /:/prefs, and fake-plex-tv
// lists it in resources. Both fakes reach it through the world's
// fake.Shared.
package plexaccount

import (
	"net/netip"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
)

const key = "plexaccount"

// Server is a fake-plex server the account owns. Target is its fake's
// name, whose fronts are the server's connections.
type Server struct {
	Target     fault.Target
	Name       string
	Identifier string
	Version    string
	// Node is the Node the server runs on, whose site's address is its
	// public address.
	Node   string
	Remote Remote
}

// Remote is what a server publishes for use away from home.
type Remote struct {
	// On is Remote Access, PublishServerOnPlexOnlineKey.
	On bool
	// Custom is customConnections, which plex.tv also lists as connections.
	Custom []string
	// Published are the connections plex.tv lists beside each Node's
	// front.
	Published []string
	Mapping   Mapping
}

// Mapping is the state Plex's own check last left, in /myplex/account.
type Mapping struct {
	State   string
	Error   string
	Pending bool
	Public  netip.AddrPort
}

// MappingOf is a Scenario's mapping state as Plex reports it: mapped,
// failed, or pending, which the recorded run of 2026-11-24 saw as mapped
// while reachabilityPending was true. Empty is a server Plex never
// checked, which no run recorded and the fake reports as unknown.
func MappingOf(state string, public netip.AddrPort) Mapping {
	switch state {
	case "mapped", "failed":
		return Mapping{State: state, Public: public}
	case "pending":
		return Mapping{State: "mapped", Pending: true, Public: public}
	}
	return Mapping{State: "unknown"}
}

type Account struct {
	mu      sync.Mutex
	servers []Server
	// grants is each live server token's server identifier.
	grants map[string]string
}

// Of is shared's account. The world builds its fakes one after another,
// so the first to ask puts it.
func Of(shared *fake.Shared) *Account {
	if v, ok := shared.Get(key); ok {
		a, _ := v.(*Account)
		return a
	}
	a := &Account{grants: map[string]string{}}
	shared.Put(key, a)
	return a
}

// Claim makes srv one of the account's servers.
func (a *Account) Claim(srv Server) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.servers = append(a.servers, srv)
}

func (a *Account) Servers() []Server {
	a.mu.Lock()
	defer a.mu.Unlock()
	return slices.Clone(a.servers)
}

// Server is the server named identifier, false when the account owns
// none.
func (a *Account) Server(identifier string) (Server, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	i := slices.IndexFunc(a.servers, func(s Server) bool { return s.Identifier == identifier })
	if i < 0 {
		return Server{}, false
	}
	return a.servers[i], true
}

// SetMapping records m as the server named identifier's mapping state.
func (a *Account) SetMapping(identifier string, m Mapping) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if i := slices.IndexFunc(a.servers, func(s Server) bool { return s.Identifier == identifier }); i >= 0 {
		a.servers[i].Remote.Mapping = m
	}
}

// Grant makes token one the server named identifier takes.
func (a *Account) Grant(identifier, token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.grants[token] = identifier
}

func (a *Account) Revoke(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.grants, token)
}

// Accepts is whether the server named identifier takes token.
func (a *Account) Accepts(identifier, token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	id, ok := a.grants[token]
	return ok && id == identifier
}
