package metadata_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// gateClient is a client of the door: its test reads /test, and as a
// provider half it fetches /fetch.
type gateClient struct {
	unlisted
	line *metadata.Line
	p    metadata.Provider
	base *url.URL
}

func (c gateClient) Provider() metadata.Provider { return c.p }

func (c gateClient) Test(ctx context.Context, ask metadata.Ask) error {
	return c.line.Get(ctx, ask, metadata.Request{URL: c.base.JoinPath("test")}, &body{})
}

func (gateClient) Search(context.Context, metadata.Ask, string) ([]works.Candidate, error) {
	return nil, nil
}

func (gateClient) Sources() []artwork.Source { return nil }

func (c gateClient) Fetch(ctx context.Context, ask metadata.Ask, _ works.Key, _ []byte) (works.Fetched, error) {
	if err := c.line.Get(ctx, ask, metadata.Request{URL: c.base.JoinPath("fetch")}, &body{}); err != nil {
		return works.Fetched{}, err
	}
	return works.Fetched{Label: "The Attic"}, nil
}

// gated is a Controller whose one holder reaches the door, beside TMDB,
// which takes no PIN.
type gated struct {
	cfg  metadata.Config
	m    *metadata.Metadata
	door *door
}

func openGated(t *testing.T, d *door) *gated {
	t.Helper()
	lines, base, _ := serve(t, d.handle)
	p := gate(base)
	c := gateClient{line: lines.Line(p), p: p, base: base}
	holder := metadata.NewHolder(metadata.HolderConfig{Types: map[works.TypeID]metadata.Source{"fixture": c}, Testers: []metadata.Tester{c}})
	g := &gated{door: d, cfg: metadata.Config{
		Library:    storagetest.Library(t, works.Schema(), metadata.Schema()),
		Sealer:     secrettest.Sealer(t),
		Roster:     roster{may: []role.NodeID{home}},
		Service:    func(role.NodeID) metadatav1connect.MetadataServiceHandler { return holder },
		Providers:  []metadata.Provider{p, tmdb},
		Changed:    &changes{},
		Timings:    retesting,
		Rand:       seeded(),
		Local:      storagetest.Local(t, artwork.Schema()),
		ArtworkDir: t.TempDir(),
	}}
	g.restart(t)
	return g
}

func (g *gated) restart(t *testing.T) {
	t.Helper()
	m, err := metadata.Open(t.Context(), g.cfg)
	if err != nil {
		t.Fatal(err)
	}
	g.m = m
}

func (g *gated) status(t *testing.T) metadata.ProviderStatus {
	t.Helper()
	return g.m.Providers(t.Context(), identity.Admin{})[0]
}

// fetched is the credentials behind the token a fetch sent.
func (g *gated) fetched(t *testing.T) credentials {
	t.Helper()
	if _, err := g.m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd}); err != nil {
		t.Fatal(err)
	}
	return g.door.lastGet()
}

func TestTheBundledKeyLogsInWithoutAPin(t *testing.T) {
	metadata.Bundle(t, "gate:shipped")
	g := openGated(t, newDoor(map[string]string{"shipped": "", "mine": "1234"}))
	if s := g.status(t); s.Key != metadata.KeyBundled || !s.TakesPin || s.PinSet {
		t.Fatalf("status = %+v, want SlipStream's key, a PIN offered and none set", s)
	}
	if got := g.fetched(t); got != (credentials{Key: "shipped"}) {
		t.Fatalf("a fetch logged in with %+v, want SlipStream's key and no PIN", got)
	}

	if _, err := g.m.SetKey(t.Context(), identity.Admin{}, "gate", metadata.NewKey{Key: "mine", Pin: "1234"}); err != nil {
		t.Fatal(err)
	}
	if l := g.door.loggedIn(); l[len(l)-1] != (credentials{Key: "mine", Pin: "1234"}) {
		t.Fatalf("the test before saving logged in with %+v, want the key and its PIN", l[len(l)-1])
	}
	g.restart(t)
	if s := g.status(t); s.Key != metadata.KeyYours || !s.PinSet {
		t.Fatalf("after a restart status = %+v, want your key with its PIN", s)
	}
	if got := g.fetched(t); got != (credentials{Key: "mine", Pin: "1234"}) {
		t.Fatalf("a fetch logged in with %+v, want your key and its PIN", got)
	}

	if s, err := g.m.ClearKey(t.Context(), identity.Admin{}, "gate"); err != nil || s.Key != metadata.KeyBundled || s.PinSet {
		t.Fatalf("ClearKey = %+v, %v; want SlipStream's key and no PIN", s, err)
	}
	if got := g.fetched(t); got != (credentials{Key: "shipped"}) {
		t.Fatalf("once cleared, a fetch logged in with %+v, want SlipStream's key and no PIN", got)
	}
	g.restart(t)
	if s := g.status(t); s.Key != metadata.KeyBundled || s.PinSet {
		t.Fatalf("after clearing and a restart status = %+v, want SlipStream's key and no PIN", s)
	}
}

func TestSetKeyRefusesAPinTheProviderRefuses(t *testing.T) {
	g := openGated(t, newDoor(map[string]string{"mine": "1234"}))
	_, err := g.m.SetKey(t.Context(), identity.Admin{}, "gate", metadata.NewKey{Key: "mine", Pin: "0000"})
	if se, ok := errors.AsType[*works.SourceError](err); !ok || !errors.Is(err, metadata.ErrPinRefused) || se.Source != "Gate" || se.Words != "refused the PIN" {
		t.Fatalf("SetKey with a wrong PIN = %v, want Gate's PIN refusal", err)
	}
	_, err = g.m.SetKey(t.Context(), identity.Admin{}, "gate", metadata.NewKey{Key: "typo", Pin: "1234"})
	if !errors.Is(err, metadata.ErrKeyRefused) {
		t.Fatalf("SetKey with a wrong key = %v, want ErrKeyRefused", err)
	}
	_, err = g.m.SetKey(t.Context(), identity.Admin{}, "tmdb", metadata.NewKey{Key: "mine", Pin: "1234"})
	if se, ok := errors.AsType[*works.SourceError](err); !ok || !errors.Is(err, metadata.ErrTakesNoPin) || se.Source != "TMDB" {
		t.Fatalf("SetKey with a PIN for TMDB = %v, want ErrTakesNoPin", err)
	}
	g.restart(t)
	if s := g.status(t); s.Key != metadata.KeyNone || s.PinSet {
		t.Fatalf("after refused keys and PINs status = %+v, want none kept", s)
	}
}

func TestARefusedPinReadsAsThePin(t *testing.T) {
	d := newDoor(map[string]string{"mine": "1234"})
	g := openGated(t, d)
	if _, err := g.m.SetKey(t.Context(), identity.Admin{}, "gate", metadata.NewKey{Key: "mine", Pin: "1234"}); err != nil {
		t.Fatal(err)
	}
	d.set(func(d *door) { d.accepts["mine"] = "5678" })
	d.expire()
	if _, err := g.m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd}); !errors.Is(err, works.ErrRefused) {
		t.Fatalf("a fetch once the PIN changed = %v, want ErrRefused", err)
	}
	s := g.status(t)
	if s.State != metadata.StatePinRefused || s.LastError != "Gate refused the PIN" || s.Summary != "Gate refused your PIN. Check it and save it again with your key, or remove your key." {
		t.Fatalf("status = %+v, want the PIN named as refused", s)
	}
}
