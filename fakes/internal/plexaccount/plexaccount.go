// Package plexaccount is the plex.tv account of one world: the fake-plex
// servers it owns, which fake-plex-tv lists in resources, and the server
// tokens fake-plex-tv issued, which fake-plex takes until a sign-out
// revokes them. Both fakes reach it through the world's fake.Shared.
package plexaccount

import (
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
