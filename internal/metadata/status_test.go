package metadata_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// host is TMDB's host as one Node reaches it: it refuses the keys in
// refused, a call with none included when "" is there, and answers 418,
// which reads as unreachable, while down.
type host struct {
	mu      sync.Mutex
	refused map[string]bool
	down    bool
}

func (h *host) set(down bool, refused ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.down, h.refused = down, map[string]bool{}
	for _, k := range refused {
		h.refused[k] = true
	}
}

func (h *host) handle(n int, w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch {
	case h.down:
		w.WriteHeader(http.StatusTeapot)
	case h.refused[r.URL.Query().Get("api_key")]:
		w.WriteHeader(http.StatusUnauthorized)
	default:
		ok(w, n)
	}
}

// probe is a TMDB client over a Line: its test reads /test, and as a
// provider half it fetches /fetch.
type probe struct {
	unlisted
	line *metadata.Line
	base *url.URL
}

func (probe) Provider() metadata.Provider { return tmdb }

func (p probe) Test(ctx context.Context, ask metadata.Ask) error {
	return p.line.Get(ctx, ask, metadata.Request{URL: p.base.JoinPath("test")}, &body{})
}

func (probe) Search(context.Context, metadata.Ask, string) ([]works.Candidate, error) {
	return nil, nil
}

func (probe) Sources() []artwork.Source { return nil }

func (p probe) Fetch(ctx context.Context, ask metadata.Ask, _ works.Key) (works.Fetched, error) {
	if err := p.line.Get(ctx, ask, metadata.Request{URL: p.base.JoinPath("fetch")}, &body{}); err != nil {
		return works.Fetched{}, err
	}
	return works.Fetched{Label: "The Attic"}, nil
}

type changes struct {
	mu        sync.Mutex
	providers []string
}

func (c *changes) Publish(_ context.Context, v metadata.Changed) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers = append(c.providers, v.Provider)
}

func (c *changes) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.providers)
}

// fleet is a Controller whose Nodes each reach TMDB through a host of
// their own.
type fleet struct {
	cfg     metadata.Config
	m       *metadata.Metadata
	roster  *roster
	hosts   map[role.NodeID]*host
	sent    map[role.NodeID]*sent
	changed *changes
}

var retesting = func() metadata.Timings {
	t := timings
	t.RetestMin, t.RetestMax = time.Minute, time.Hour
	return t
}()

func open(t *testing.T, r roster, nodes ...role.NodeID) *fleet {
	t.Helper()
	f := &fleet{roster: &r, hosts: map[role.NodeID]*host{}, sent: map[role.NodeID]*sent{}, changed: &changes{}}
	holders := map[role.NodeID]*metadata.Holder{}
	for _, n := range nodes {
		h := &host{refused: map[string]bool{}}
		lines, base, s := serve(t, h.handle)
		p := probe{line: lines.Line(tmdb), base: base}
		f.hosts[n], f.sent[n] = h, s
		holders[n] = metadata.NewHolder(metadata.HolderConfig{Types: map[works.TypeID]metadata.Source{"fixture": p}, Testers: []metadata.Tester{p}})
	}
	f.cfg = metadata.Config{
		Library:    storagetest.Library(t, works.Schema(), metadata.Schema()),
		Sealer:     secrettest.Sealer(t),
		Roster:     f.roster,
		Service:    func(n role.NodeID) metadatav1connect.MetadataServiceHandler { return holders[n] },
		Providers:  []metadata.Provider{tmdb},
		Changed:    f.changed,
		Timings:    retesting,
		Rand:       seeded(),
		Local:      storagetest.Local(t, artwork.Schema()),
		ArtworkDir: t.TempDir(),
	}
	f.restart(t)
	return f
}

// restart opens the Controller again over the same library.db.
func (f *fleet) restart(t *testing.T) {
	t.Helper()
	m, err := metadata.Open(t.Context(), f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.m = m
}

func (f *fleet) status(t *testing.T) metadata.ProviderStatus {
	t.Helper()
	all := f.m.Providers(t.Context(), identity.Admin{})
	if len(all) != 1 || all[0].Provider != "tmdb" {
		t.Fatalf("Providers = %+v, want TMDB alone", all)
	}
	return all[0]
}

func (f *fleet) fetch(t *testing.T) error {
	t.Helper()
	_, err := f.m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForAdd})
	return err
}

// lastKey is the api_key node's latest request carried, and whether it
// carried one.
func (f *fleet) lastKey(node role.NodeID) (string, bool) {
	s := f.sent[node]
	s.mu.Lock()
	defer s.mu.Unlock()
	q := s.reqs[len(s.reqs)-1].URL.Query()
	return q.Get("api_key"), q.Has("api_key")
}

func TestNoKeyMeansKeylessCalls(t *testing.T) {
	for _, tc := range []struct {
		name, bundle, sends string
		key                 metadata.KeyInUse
	}{
		{"a build with no key", "", "", metadata.KeyNone},
		{"a build with SlipStream's key", "tmdb:shipped,omdb:other", "shipped", metadata.KeyBundled},
	} {
		t.Run(tc.name, func(t *testing.T) {
			metadata.Bundle(t, tc.bundle)
			f := open(t, roster{may: []role.NodeID{home}}, home)
			check := func(when string, sends string, key metadata.KeyInUse) {
				t.Helper()
				if err := f.fetch(t); err != nil {
					t.Fatal(err)
				}
				if got, has := f.lastKey(home); got != sends || has != (sends != "") {
					t.Errorf("%s, a fetch sent api_key %q (present %v), want %q", when, got, has, sends)
				}
				if s := f.status(t); s.Key != key {
					t.Errorf("%s, status says key %v, want %v", when, s.Key, key)
				}
			}
			check("before any Admin key", tc.sends, tc.key)
			if _, err := f.m.SetKey(t.Context(), identity.Admin{}, "tmdb", "mine"); err != nil {
				t.Fatal(err)
			}
			check("with the Admin's key", "mine", metadata.KeyYours)
			f.restart(t)
			check("after a restart", "mine", metadata.KeyYours)
			if _, err := f.m.ClearKey(t.Context(), identity.Admin{}, "tmdb"); err != nil {
				t.Fatal(err)
			}
			check("once the Admin's key is cleared", tc.sends, tc.key)
		})
	}
}

func TestTheKeySetTimeIsKeptWithTheKey(t *testing.T) {
	f := open(t, roster{may: []role.NodeID{home}}, home)
	if got := f.status(t).KeySetAt; !got.IsZero() {
		t.Fatalf("with no Admin key the set time is %v, want none", got)
	}
	before := time.Now().Truncate(time.Microsecond)
	if _, err := f.m.SetKey(t.Context(), identity.Admin{}, "tmdb", "mine"); err != nil {
		t.Fatal(err)
	}
	set := f.status(t).KeySetAt
	if set.Before(before) || set.After(time.Now()) {
		t.Fatalf("the key's set time is %v, want between %v and now", set, before)
	}
	f.restart(t)
	if got := f.status(t).KeySetAt; !got.Equal(set) {
		t.Fatalf("after a restart the key's set time is %v, want %v", got, set)
	}
	if _, err := f.m.ClearKey(t.Context(), identity.Admin{}, "tmdb"); err != nil {
		t.Fatal(err)
	}
	if got := f.status(t).KeySetAt; !got.IsZero() {
		t.Fatalf("once the Admin's key is cleared its set time is %v, want none", got)
	}
}

func TestSetKeyRefusesAKeyTheProviderRefuses(t *testing.T) {
	f := open(t, roster{may: []role.NodeID{home}}, home)
	f.hosts[home].set(false, "typo")
	_, err := f.m.SetKey(t.Context(), identity.Admin{}, "tmdb", "typo")
	if se, ok := errors.AsType[*works.SourceError](err); !ok || !errors.Is(err, metadata.ErrKeyRefused) || se.Source != "TMDB" || se.Words != "refused the key: 401 Unauthorized" {
		t.Fatalf("SetKey(typo) = %v, want TMDB's refusal", err)
	}
	f.hosts[home].set(true)
	if _, err := f.m.SetKey(t.Context(), identity.Admin{}, "tmdb", "mine"); !errors.Is(err, works.ErrUnavailable) {
		t.Fatalf("SetKey while TMDB is unreachable = %v, want ErrUnavailable", err)
	}
	if _, err := f.m.SetKey(t.Context(), identity.Admin{}, "nope", "k"); !errors.Is(err, metadata.ErrNoProvider) {
		t.Fatalf("SetKey(nope) = %v, want ErrNoProvider", err)
	}
	f.restart(t)
	if s := f.status(t); s.Key != metadata.KeyNone || s.State != metadata.StateUnused || f.changed.count() != 0 {
		t.Fatalf("after refused and untested keys, status = %+v with %d events; want no key kept, unused and no event", s, f.changed.count())
	}

	f.hosts[home].set(false)
	s, err := f.m.SetKey(t.Context(), identity.Admin{}, "tmdb", "mine")
	if err != nil || s.Key != metadata.KeyYours || s.State != metadata.StateWorking || s.Summary != "TMDB is working." {
		t.Fatalf("SetKey(mine) = %+v, %v; want your key, working", s, err)
	}
	var sealed []byte
	err = f.cfg.Library.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, "SELECT api_key_sealed FROM metadata_key WHERE provider = 'tmdb'").Scan(&sealed)
	})
	if err != nil || len(sealed) == 0 || bytes.Contains(sealed, []byte("mine")) {
		t.Fatalf("library.db holds %q, %v; want the key sealed", sealed, err)
	}
	if f.changed.count() != 1 {
		t.Fatalf("metadata.changed fired %d times, want once for the new key", f.changed.count())
	}
}

func TestStatusComesFromRealCallOutcomes(t *testing.T) {
	f := open(t, roster{may: []role.NodeID{home}}, home)
	if s := f.status(t); s.State != metadata.StateUnused || s.Summary != "This build has no TMDB key. Add your own." {
		t.Fatalf("before any call, status = %+v; want unused, saying the build has no key", s)
	}
	if err := f.fetch(t); err != nil {
		t.Fatal(err)
	}
	s := f.status(t)
	if s.State != metadata.StateWorking || s.LastSuccess.IsZero() || f.changed.count() != 1 {
		t.Fatalf("after a good fetch, status = %+v with %d events; want working since the call, one event", s, f.changed.count())
	}
	if err := f.fetch(t); err != nil || f.changed.count() != 1 {
		t.Fatalf("a second good fetch = %v with %d events; want no new event", err, f.changed.count())
	}

	f.hosts[home].set(false, "")
	if err := f.fetch(t); !errors.Is(err, works.ErrRefused) {
		t.Fatalf("fetch = %v, want ErrRefused", err)
	}
	s = f.status(t)
	if s.State != metadata.StateKeyRefused || s.LastError != "TMDB refused the key: 401 Unauthorized" || s.Summary != "This build has no TMDB key. Add your own." || f.changed.count() != 2 {
		t.Fatalf("after a refusal, status = %+v with %d events; want key refused in plain words", s, f.changed.count())
	}

	f.hosts[home].set(false)
	s, err := f.m.Test(t.Context(), identity.Admin{}, "tmdb")
	if err != nil || s.State != metadata.StateWorking || s.LastError == "" || f.changed.count() != 3 {
		t.Fatalf("Test = %+v, %v with %d events; want working again, the last error kept", s, err, f.changed.count())
	}
	if path := f.sent[home].reqs[len(f.sent[home].reqs)-1].URL.Path; path != "/test" {
		t.Fatalf("Test sent %s, want the provider's test call", path)
	}
}

func TestEachNodesReachIsShown(t *testing.T) {
	f := open(t, roster{may: []role.NodeID{box}}, home, box)
	f.hosts[box].set(true)
	if err := f.fetch(t); !errors.Is(err, works.ErrUnavailable) {
		t.Fatalf("fetch through box = %v, want ErrUnavailable", err)
	}
	if s := f.status(t); s.State != metadata.StateUnreachable || s.Summary != "SlipStream can't reach TMDB. SlipStream retries automatically. Check the network of each Node that can't reach it." {
		t.Fatalf("status = %+v, want unreachable", s)
	}
	f.roster.may = []role.NodeID{home, box}
	if err := f.fetch(t); err != nil {
		t.Fatal(err)
	}
	s := f.status(t)
	reached := map[role.NodeID]bool{}
	for _, n := range s.Nodes {
		reached[n.Node] = n.Reached
	}
	if s.State != metadata.StateWorking || len(s.Nodes) != 2 || !reached[home] || reached[box] {
		t.Fatalf("status = %+v, want working, home reaching TMDB and box not", s)
	}
}

func TestUnreachableRetestsBackOffFromAMinuteToAnHour(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := open(t, roster{may: []role.NodeID{home}}, home)
		ctx, stop := context.WithCancel(t.Context())
		done := make(chan error)
		go func() { done <- f.m.Run(ctx) }()
		defer func() {
			stop()
			if err := <-done; err != nil {
				t.Error(err)
			}
		}()
		f.hosts[home].set(true)
		failed := time.Now()
		if err := f.fetch(t); !errors.Is(err, works.ErrUnavailable) {
			t.Fatalf("fetch = %v, want ErrUnavailable", err)
		}
		time.Sleep(3*time.Hour + 5*time.Minute)
		synctest.Wait()
		var gaps []time.Duration
		last := failed
		s := f.sent[home]
		s.mu.Lock()
		for i, r := range s.reqs {
			if r.URL.Path == "/test" {
				gaps = append(gaps, s.at[i].Sub(last))
				last = s.at[i]
			}
		}
		s.mu.Unlock()
		want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 16 * time.Minute, 32 * time.Minute, time.Hour, time.Hour}
		if !slices.Equal(gaps, want) {
			t.Fatalf("retests came after %v, want %v", gaps, want)
		}

		f.hosts[home].set(false)
		time.Sleep(time.Hour)
		synctest.Wait()
		tests := s.count()
		if st := f.status(t); st.State != metadata.StateWorking {
			t.Fatalf("after TMDB came back, status = %+v, want working", st)
		}
		time.Sleep(24 * time.Hour)
		synctest.Wait()
		if s.count() != tests {
			t.Fatalf("%d more calls after TMDB answered, want no more retests", s.count()-tests)
		}
	})
}
