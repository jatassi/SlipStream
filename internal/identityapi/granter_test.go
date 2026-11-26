package identityapi_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing/fstest"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

// offersSchema is the stand-in owner's table, which its Redeemed writes
// inside identity's redemption Write.
var offersSchema = storage.NewSchema[storage.Library]("offers", fstest.MapFS{
	"schema/00001_offers.sql": {Data: []byte("-- +goose Up\nCREATE TABLE offers_offer (account_id BLOB NOT NULL, server TEXT NOT NULL) STRICT;\n")},
})

type server struct {
	name, kind string
	shown      bool
	plex       bool
	// refuses fails Redeemed after the offers before it are written.
	refuses bool
}

// granter is a stand-in Media Servers owner. It offers the servers it
// holds and records what identity hands it.
type granter struct {
	lib *storage.DB[storage.Library]

	mu      sync.Mutex
	servers map[storage.UUID]server
	// afters is the offers each after saw committed.
	afters []int
	// steps is Leaving and done, each with whether the Account was there.
	steps         []string
	refuseLeaving bool
}

var _ identity.Granter = (*granter)(nil)

func newGranter(lib *storage.DB[storage.Library]) *granter {
	return &granter{lib: lib, servers: map[storage.UUID]server{}}
}

func (g *granter) add(s server) storage.UUID {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := storage.NewUUID()
	g.servers[id] = s
	return id
}

func (g *granter) change(id storage.UUID, f func(*server)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	s := g.servers[id]
	f(&s)
	g.servers[id] = s
}

func (g *granter) CheckGrants(_ context.Context, _ storage.Reader[storage.Library], gs identity.Grants) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, sg := range gs.MediaServers {
		s, ok := g.servers[sg.Server]
		if !ok {
			return &identity.NotGrantableError{Why: identity.Gone}
		}
		if !s.shown {
			return &identity.NotGrantableError{Name: s.name, Why: identity.NotShown}
		}
		if sg.PlexHome && !s.plex {
			return &identity.NotGrantableError{Name: s.name, Why: identity.NotPlexWay}
		}
	}
	return nil
}

func (g *granter) Describe(_ context.Context, _ storage.Reader[storage.Library], gs identity.Grants) (identity.Described, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var d identity.Described
	for _, sg := range gs.MediaServers {
		s, ok := g.servers[sg.Server]
		d.MediaServers = append(d.MediaServers, identity.GrantedServer{
			Server: sg.Server, Name: s.name, Kind: s.kind, Present: ok, Shown: s.shown, PlexHome: sg.PlexHome,
		})
	}
	return d, nil
}

func (g *granter) Redeemed(ctx context.Context, tx *storage.Tx[storage.Library], r identity.Redemption) (func(context.Context), error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, sg := range r.Grants.MediaServers {
		s := g.servers[sg.Server]
		if !s.shown {
			continue
		}
		if s.refuses {
			return nil, errors.New(s.name + " refused the offer")
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO offers_offer (account_id, server) VALUES (?, ?)", r.Account.UUID, s.name); err != nil {
			return nil, err
		}
	}
	return func(ctx context.Context) {
		n := offered(ctx, g.lib)
		g.mu.Lock()
		defer g.mu.Unlock()
		g.afters = append(g.afters, len(n))
	}, nil
}

func (g *granter) Leaving(ctx context.Context, account identity.AccountID) (func(), error) {
	g.step(ctx, "leaving", account)
	if g.refuseLeaving {
		return nil, errors.New("leaving refused")
	}
	if err := g.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM offers_offer WHERE account_id = ?", account.UUID)
		return err
	}); err != nil {
		return nil, err
	}
	return func() { g.step(ctx, "done", account) }, nil
}

func (g *granter) step(ctx context.Context, name string, account identity.AccountID) {
	var n int
	if err := g.lib.QueryRowContext(ctx, "SELECT COUNT(*) FROM identity_account WHERE id = ?", account.UUID).Scan(&n); err != nil {
		panic(err)
	}
	if n == 1 {
		name += " with the Account"
	} else {
		name += " without the Account"
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.steps = append(g.steps, name)
}

func (n *node) afters() []int {
	n.grants.mu.Lock()
	defer n.grants.mu.Unlock()
	return slices.Clone(n.grants.afters)
}

func (n *node) steps() []string {
	n.grants.mu.Lock()
	defer n.grants.mu.Unlock()
	return slices.Clone(n.grants.steps)
}

// offered is every server offered to anyone, as committed.
func (n *node) offered() []string { return offered(n.t.Context(), n.lib) }

func offered(ctx context.Context, lib *storage.DB[storage.Library]) []string {
	rows, err := lib.QueryContext(ctx, "SELECT server FROM offers_offer ORDER BY server")
	if err != nil {
		panic(err)
	}
	defer func() { _ = rows.Close() }()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			panic(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return out
}
