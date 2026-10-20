package metadataapi_test

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/imagecdn"
	fakeomdb "github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketmdb "github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/ownertest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadataapi"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/tasks"
	"github.com/jatassi/slipstream/internal/works"
)

var self = role.NodeID{UUID: storage.NewUUID()}

type alone struct{}

func (alone) Self() role.NodeID { return self }

func (alone) Pick(role.Name, role.NodeID) (role.NodeID, bool) { return self, true }

func (alone) PickOther(role.Name, role.NodeID) (role.NodeID, bool) { return role.NodeID{}, false }

func (alone) Why(role.Name) string { return "" }

type registry struct{ m *metadata.Metadata }

func (r registry) Metadata() (*metadata.Metadata, bool) { return r.m, true }

func (registry) Tasks() (*tasks.Scheduler, bool) { return nil, false }

type changes struct {
	mu        sync.Mutex
	providers []string
}

func (c *changes) Publish(_ context.Context, v metadata.Changed) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.providers = append(c.providers, v.Provider)
}

type served struct {
	owner           *ownertest.Owner
	tmdb, omdb, cdn faketest.Served
	fetched         *fetches
	changed         *changes
}

// world is what a test sets before the Controller opens: how long a
// failed image fetch waits, and the faults fake-image-cdn plays.
type world struct {
	retry  time.Duration
	faults []fault.Rule
}

// serve serves the operations over fake-tmdb and fake-omdb, reached by a
// Controller that holds Metadata alone. Each fake has its own authority,
// so each provider has a client of its own.
func serve(t *testing.T) *served { return serveWorld(t, world{retry: time.Hour}) }

func serveWorld(t *testing.T, w world) *served {
	t.Helper()
	s := scenario.Scenario{Name: "x", Seed: "metadataapi"}
	srv := &served{
		tmdb:    faketest.Loopback(t, faketmdb.Fake, s),
		omdb:    faketest.Loopback(t, fakeomdb.Fake, s),
		cdn:     faketest.Loopback(t, imagecdn.Fake, s, faketest.WithFaults(w.faults...)),
		changed: &changes{},
	}
	srv.fetched = &fetches{next: srv.cdn.Client.Transport}
	lines := func(c *http.Client) *metadata.Lines {
		return metadata.NewLines(c, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	}
	tm := tmdb.New(lines(srv.tmdb.Client), tmdb.Endpoints{
		API:    endpoint.MustParseURL(srv.tmdb.Address.URL("/3")),
		Images: endpoint.MustParseURL(srv.cdn.Address.URL(imagecdn.Root)),
	}, tmdb.DefaultTimings())
	om := omdb.New(lines(srv.omdb.Client), omdb.Endpoints{API: endpoint.MustParseURL(srv.omdb.Address.URL(""))}, omdb.DefaultTimings())
	holder := metadata.NewHolder(metadata.HolderConfig{
		Types:   map[works.TypeID]metadata.Source{"pictures": pictures{tm.Images()}},
		Testers: []metadata.Tester{tm, om},
		Lines:   metadata.NewLines(&http.Client{Transport: srv.fetched}, "SlipStream/test", quick, rand.New(rand.NewPCG(1, 2))), //nolint:gosec // a fixed draw the test replays
	})
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), works.Schema(), metadata.Schema())
	timings := metadata.DefaultTimings()
	timings.ArtworkRetry = w.retry
	m, err := metadata.Open(t.Context(), metadata.Config{
		Library:    lib,
		Local:      storagetest.Local(t, artwork.Schema()),
		ArtworkDir: t.TempDir(),
		Minter:     minter,
		Sealer:     secrettest.Sealer(t),
		Roster:     alone{},
		Service:    func(role.NodeID) metadatav1connect.MetadataServiceHandler { return holder },
		Providers:  []metadata.Provider{tm.Provider(), om.Provider()},
		Changed:    srv.changed,
		Timings:    timings,
		Rand:       rand.New(rand.NewPCG(1, 2)), //nolint:gosec // a fixed draw the test replays
	})
	if err != nil {
		t.Fatal(err)
	}
	srv.owner = ownertest.Serve(t, lib, func(r *api.Router) { metadataapi.Routes(r, registry{m}) })
	return srv
}

var minter = artwork.NewMinter([]byte("metadataapi"))

// quick retries an image host at once, so a test sees every try.
var quick = func() metadata.Timings {
	t := metadata.DefaultTimings()
	t.RetryMin, t.RetryMax = time.Millisecond, time.Millisecond
	return t
}()

// pictures is a Content Type that only names TMDB's image host.
type pictures struct{ images artwork.Source }

func (pictures) Search(context.Context, metadata.Ask, string) ([]works.Candidate, error) {
	return nil, works.ErrNotFound
}

func (pictures) Fetch(context.Context, metadata.Ask, works.Key) (works.Fetched, error) {
	return works.Fetched{}, works.ErrNotFound
}

func (p pictures) Sources() []artwork.Source { return []artwork.Source{p.images} }

func (pictures) Changes(context.Context, metadata.Ask, time.Time, time.Time, int) (metadata.ChangesPage, error) {
	return metadata.ChangesPage{}, metadata.ErrNoChanges
}

// fetches records each path fetched from fake-image-cdn.
type fetches struct {
	next  http.RoundTripper
	mu    sync.Mutex
	paths []string
}

func (f *fetches) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	f.mu.Unlock()
	return f.next.RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

func (f *fetches) all() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.paths)
}

const (
	unusedTMDB = `{"id":"tmdb","label":"TMDB","summary":"This build has no TMDB key. Add your own.","key":"none","state":"unused","until":null,"details":{"lastSuccess":null,"lastError":null,"nodes":[]}}`
	unusedOMDb = `{"id":"omdb","label":"OMDb","summary":"This build has no OMDb key. Add your own.","key":"none","state":"unused","until":null,"details":{"lastSuccess":null,"lastError":null,"nodes":[]}}`
)

func TestAnAdminKeyIsTestedOnSaveAndShownInUse(t *testing.T) {
	s := serve(t)
	s.owner.Expect(t, http.MethodGet, "/api/v1/metadata/providers", "", http.StatusOK, `{"providers":[`+unusedTMDB+`,`+unusedOMDb+`]}`)

	s.tmdb.Cue(t, faketmdb.RevokeKey.Name, map[string]string{"key": "typo"})
	s.owner.Expect(t, http.MethodPut, "/api/v1/metadata/providers/tmdb/key", `{"key":"typo"}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"metadata_key_refused","title":"The metadata provider refused that key, so SlipStream didn't save it","detail":"TMDB refused the key: Invalid API key: You must be granted a valid key."}`)
	s.owner.Expect(t, http.MethodGet, "/api/v1/metadata/providers", "", http.StatusOK, `{"providers":[`+unusedTMDB+`,`+unusedOMDb+`]}`)

	s.owner.Expect(t, http.MethodPut, "/api/v1/metadata/providers/tmdb/key", `{"key":"mine"}`, http.StatusOK,
		`{"id":"tmdb","label":"TMDB","summary":"TMDB is working.","key":"yours","state":"working","until":null,"details":{"lastSuccess":"T","lastError":null,"nodes":[{"nodeId":"id1","reached":true,"at":"T"}]}}`)
	s.owner.Expect(t, http.MethodDelete, "/api/v1/metadata/providers/tmdb/key", "", http.StatusOK, unusedTMDB)
	s.owner.Expect(t, http.MethodDelete, "/api/v1/metadata/providers/tmdb/key", "", http.StatusOK, unusedTMDB)
	if want := []string{"tmdb", "tmdb"}; !slices.Equal(s.changed.providers, want) {
		t.Fatalf("metadata.changed fired for %v, want %v: once for the key saved, once for it cleared", s.changed.providers, want)
	}
}

func TestATestReadsTheProvidersStateWithItsTime(t *testing.T) {
	s := serve(t)
	s.omdb.Cue(t, fakeomdb.Cap.Name, map[string]any{})
	code, reply := s.owner.Call(t, http.MethodPost, "/api/v1/metadata/providers/omdb/test", "")
	midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour).Format(time.RFC3339)
	if code != http.StatusOK || !strings.Contains(reply, `"state":"out_of_allowance","until":"`+midnight+`"`) ||
		!strings.Contains(reply, `"summary":"OMDb is out of today's allowance. It resets on its own."`) {
		t.Fatalf("testMetadataProvider = %d %s, want OMDb out of its allowance until %s", code, reply, midnight)
	}

	s.tmdb.Cue(t, faketmdb.RevokeKey.Name, map[string]string{"key": ""})
	s.owner.Expect(t, http.MethodPost, "/api/v1/metadata/providers/tmdb/test", "", http.StatusOK,
		`{"id":"tmdb","label":"TMDB","summary":"This build has no TMDB key. Add your own.","key":"none","state":"key_refused","until":null,"details":{"lastSuccess":null,"lastError":{"words":"TMDB refused the key: Invalid API key: You must be granted a valid key.","at":"T"},"nodes":[{"nodeId":"id1","reached":true,"at":"T"}]}}`)
}

func TestAnUnknownProviderIsNotFound(t *testing.T) {
	s := serve(t)
	notFound := `{"status":404,"code":"metadata_provider_not_found","title":"No metadata provider has that id"}`
	s.owner.Expect(t, http.MethodPut, "/api/v1/metadata/providers/imdb/key", `{"key":"k"}`, http.StatusNotFound, notFound)
	s.owner.Expect(t, http.MethodDelete, "/api/v1/metadata/providers/imdb/key", "", http.StatusNotFound, notFound)
	s.owner.Expect(t, http.MethodPost, "/api/v1/metadata/providers/imdb/test", "", http.StatusNotFound, notFound)
}

func TestMetadataSettingsReadAndChangeByKey(t *testing.T) {
	s := serve(t)
	type field struct {
		Key       string
		Kind      string
		Value     any
		IsDefault bool
		Options   []struct{ Value string }
	}
	read := func() []field {
		t.Helper()
		code, reply := s.owner.Call(t, http.MethodGet, "/api/v1/metadata/settings", "")
		var body struct{ Settings []field }
		if err := json.Unmarshal([]byte(reply), &body); code != http.StatusOK || err != nil {
			t.Fatalf("getMetadataSettings = %d %s", code, reply)
		}
		return body.Settings
	}
	got := read()
	keys := make([]string, len(got))
	for i, f := range got {
		keys[i] = f.Key
	}
	if want := []string{"metadata.language", "metadata.region", "metadata.cacheFor", "metadata.ratingsEvery", "metadata.artworkStore"}; !slices.Equal(keys, want) {
		t.Fatalf("settings = %v, want %v", keys, want)
	}
	if region := got[1]; region.Kind != "choice" || region.Value != "US" || len(region.Options) < 200 {
		t.Fatalf("region = %v, want a choice of the countries, US by default", region)
	}

	s.owner.Expect(t, http.MethodPatch, "/api/v1/metadata/settings", `{"values":{"metadata.language":"de-DE","metadata.cacheFor":86460}}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"setting_out_of_range","title":"The value is outside the setting's range","detail":"Between 0 s and 1 day."}`)
	s.owner.Expect(t, http.MethodPatch, "/api/v1/metadata/settings", `{"values":{"metadata.refresh.others":86400}}`, http.StatusUnprocessableEntity,
		`{"status":422,"code":"unknown_setting","title":"No setting has this key","detail":"No setting has the key metadata.refresh.others."}`)
	if got := read(); got[0].Value != "en-US" {
		t.Fatalf("after a refused patch the language is %v, want en-US kept", got[0].Value)
	}

	code, reply := s.owner.Call(t, http.MethodPatch, "/api/v1/metadata/settings", `{"values":{"metadata.language":"de-DE","metadata.region":"DE"}}`)
	if code != http.StatusOK || !strings.Contains(reply, `"key":"metadata.language","label":"Metadata language","source":"database","applies":"live","isDefault":false,"pendingRestart":false,"kind":"choice","value":"de-DE"`) {
		t.Fatalf("updateMetadataSettings = %d %s, want German saved", code, reply)
	}
	if code, reply := s.owner.Call(t, http.MethodPatch, "/api/v1/metadata/settings", `{"values":{"metadata.language":null}}`); code != http.StatusOK {
		t.Fatalf("resetting the language = %d %s", code, reply)
	}
	if got := read(); got[0].Value != "en-US" || !got[0].IsDefault || got[1].Value != "DE" {
		t.Fatalf("after resetting the language, settings = %v %v, want en-US by default and DE kept", got[0], got[1])
	}
}

func TestClearingTheCacheAnswersNoContent(t *testing.T) {
	s := serve(t)
	if code, reply := s.owner.Call(t, http.MethodDelete, "/api/v1/metadata/cache", ""); code != http.StatusNoContent || reply != "" {
		t.Fatalf("clearMetadataCache = %d %q, want 204 and no body", code, reply)
	}
}
