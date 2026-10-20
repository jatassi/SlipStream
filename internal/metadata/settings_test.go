package metadata_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

type typeChanges struct {
	mu    sync.Mutex
	types []string
}

func (c *typeChanges) Publish(_ context.Context, v works.TypeChanged) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.types = append(c.types, v.ContentType)
}

func (c *typeChanges) take() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	got := c.types
	c.types = nil
	return got
}

// cached is a provider half whose fetch the Node's cache may answer.
type cached struct {
	unlisted
	line *metadata.Line
	base *url.URL
	h    *heard
}

func (cached) Sources() []artwork.Source { return nil }

func (cached) Search(context.Context, metadata.Ask, string) ([]works.Candidate, error) {
	return nil, nil
}

func (c cached) Fetch(ctx context.Context, ask metadata.Ask, key works.Key) (works.Fetched, error) {
	c.h.add("fetch "+string(key), ask)
	if err := c.line.Get(ctx, ask, metadata.Request{URL: c.base.JoinPath("record", string(key)), Cache: true}, &body{}); err != nil {
		return works.Fetched{}, err
	}
	return works.Fetched{Label: "The Attic"}, nil
}

// shop is a Controller over a library.db whose metadata changes for two
// Content Types, with each Node in nodes reaching TMDB through a host of
// its own and answering only while the roster lists it.
type shop struct {
	m       *metadata.Metadata
	lib     *storage.DB[storage.Library]
	local   *storage.DB[storage.Local]
	artwork string
	roster  *roster
	sent    map[role.NodeID]*sent
	heard   *heard
	changed *typeChanges
}

func openShop(t *testing.T, nodes ...role.NodeID) *shop {
	t.Helper()
	s := &shop{
		lib:   storagetest.Library(t, works.Schema(), metadata.Schema()),
		local: storagetest.Local(t, artwork.Schema()), artwork: t.TempDir(),
		roster: &roster{may: nodes}, sent: map[role.NodeID]*sent{}, heard: &heard{}, changed: &typeChanges{},
	}
	holders := map[role.NodeID]*metadata.Holder{}
	for _, n := range nodes {
		lines, base, got := serve(t, func(n int, w http.ResponseWriter, _ *http.Request) { ok(w, n) })
		s.sent[n] = got
		holders[n] = metadata.NewHolder(metadata.HolderConfig{Types: map[works.TypeID]metadata.Source{"fixture": cached{line: lines.Line(tmdb), base: base, h: s.heard}}})
	}
	m, err := metadata.Open(t.Context(), metadata.Config{
		Library: s.lib,
		Sealer:  secrettest.Sealer(t),
		Roster:  s.roster,
		Service: func(n role.NodeID) metadatav1connect.MetadataServiceHandler {
			if !slices.Contains(s.roster.may, n) {
				return unreachable{node: names[n], h: s.heard}
			}
			return holders[n]
		},
		Providers:   []metadata.Provider{tmdb},
		Changed:     &changes{},
		Timings:     timings,
		Rand:        seeded(),
		Types:       []works.TypeID{"fixture", "shelf"},
		TypeChanged: s.changed,
		Local:       s.local,
		ArtworkDir:  s.artwork,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.m = m
	return s
}

func (s *shop) set(t *testing.T, values map[string]any) {
	t.Helper()
	p := setting.Patch{}
	for k, v := range values {
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		p[k] = raw
	}
	if err := s.m.Settings().Apply(t.Context(), p); err != nil {
		t.Fatalf("Apply(%v) = %v", values, err)
	}
}

func (s *shop) fetch(t *testing.T) metadata.Ask {
	t.Helper()
	if _, err := s.m.Fetch(t.Context(), "fixture", "k1", works.Fetching{Purpose: works.ForPreview}); err != nil {
		t.Fatal(err)
	}
	s.heard.mu.Lock()
	defer s.heard.mu.Unlock()
	return s.heard.asks[len(s.heard.asks)-1]
}

// refreshAll is when each Content Type last asked for every Work to be
// refreshed, by type.
func (s *shop) refreshAll(t *testing.T) map[string]int64 {
	t.Helper()
	var fixture, shelf int64
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT
			coalesce(max(CASE content_type WHEN 'fixture' THEN refresh_all_at END), 0),
			coalesce(max(CASE content_type WHEN 'shelf' THEN refresh_all_at END), 0)
			FROM metadata_type`).Scan(&fixture, &shelf)
	})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]int64{}
	for typ, at := range map[string]int64{"fixture": fixture, "shelf": shelf} {
		if at != 0 {
			out[typ] = at
		}
	}
	return out
}

func TestLanguageOrRegionMakesEveryWorkDue(t *testing.T) {
	s := openShop(t, home)
	if ask := s.fetch(t); ask.Language != "en-US" || ask.Region != "US" {
		t.Fatalf("by default a call asks in %s for %s, want en-US for US", ask.Language, ask.Region)
	}
	if due := s.refreshAll(t); len(due) != 0 {
		t.Fatalf("before any change, %v asked for a refresh of every Work, want none", due)
	}

	s.set(t, map[string]any{"metadata.cacheFor": 600, "metadata.ratingsEvery": 86400})
	if due := s.refreshAll(t); len(due) != 0 {
		t.Fatalf("after a cache and ratings change, %v asked for a refresh of every Work, want none", due)
	}

	before := storage.Now().Time().UnixMicro()
	s.set(t, map[string]any{"metadata.language": "fr-FR"})
	due := s.refreshAll(t)
	if len(due) != 2 || due["fixture"] < before || due["shelf"] < before {
		t.Fatalf("after a new language, refresh all = %v, want every Content Type since %d", due, before)
	}
	if ask := s.fetch(t); ask.Language != "fr-FR" || ask.Region != "US" {
		t.Fatalf("after a new language a call asks in %s for %s, want fr-FR for US", ask.Language, ask.Region)
	}

	s.set(t, map[string]any{"metadata.language": "fr-FR"})
	if again := s.refreshAll(t); again["fixture"] != due["fixture"] {
		t.Fatalf("the same language again moved refresh all from %d to %d, want it kept", due["fixture"], again["fixture"])
	}

	s.set(t, map[string]any{"metadata.region": "FR"})
	if after := s.refreshAll(t); after["fixture"] <= due["fixture"] || after["shelf"] <= due["shelf"] {
		t.Fatalf("after a new region, refresh all = %v, want later than %v", after, due)
	}
	if ask := s.fetch(t); ask.Region != "FR" {
		t.Fatalf("after a new region a call asks for %s, want FR", ask.Region)
	}
}

func TestRegionChangeSendsOneTypeChangedPerType(t *testing.T) {
	s := openShop(t, home)
	region := func() string {
		t.Helper()
		var got string
		err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			var err error
			got, err = metadata.Region(ctx, r)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := region(); got != "US" {
		t.Fatalf("a library half reads region %s by default, want US", got)
	}

	s.set(t, map[string]any{"metadata.region": "GB"})
	if got := s.changed.take(); !slices.Equal(got, []string{"fixture", "shelf"}) {
		t.Fatalf("a new region announced %v, want one works.type-changed per Content Type", got)
	}
	if got := region(); got != "GB" {
		t.Fatalf("a library half reads region %s, want GB", got)
	}

	s.set(t, map[string]any{"metadata.region": "GB", "metadata.language": "en-GB", "metadata.cacheFor": 0})
	if got := s.changed.take(); len(got) != 0 {
		t.Fatalf("the same region with a new language announced %v, want nothing", got)
	}

	s.set(t, map[string]any{"metadata.region": nil})
	if got := s.changed.take(); !slices.Equal(got, []string{"fixture", "shelf"}) || region() != "US" {
		t.Fatalf("resetting the region announced %v and reads %s, want every type and US", got, region())
	}
}

func TestASettingsPatchIsAllOrNone(t *testing.T) {
	s := openShop(t, home)
	for _, tc := range []struct {
		name  string
		patch setting.Patch
		check func(error) bool
	}{
		{"a region nobody has", setting.Patch{"metadata.language": json.RawMessage(`"de-DE"`), "metadata.region": json.RawMessage(`"ZZ"`)}, isRange},
		{"a cache past a day", setting.Patch{"metadata.language": json.RawMessage(`"de-DE"`), "metadata.cacheFor": json.RawMessage(`90000`)}, isRange},
		{"ratings more than daily", setting.Patch{"metadata.ratingsEvery": json.RawMessage(`3600`)}, isRange},
		{"a refresh tier", setting.Patch{"metadata.language": json.RawMessage(`"de-DE"`), "metadata.refresh.others": json.RawMessage(`86400`)}, isUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := s.m.Settings().Apply(t.Context(), tc.patch); !tc.check(err) {
				t.Fatalf("Apply = %v, want it refused", err)
			}
			if ask := s.fetch(t); ask.Language != "en-US" {
				t.Fatalf("after a refused patch a call asks in %s, want en-US kept", ask.Language)
			}
		})
	}

	if err := s.m.Tuning().Apply(t.Context(), setting.Patch{"metadata.refresh.others": json.RawMessage(`604800`)}); err != nil {
		t.Fatal(err)
	}
	fields, err := s.m.Tuning().Fields(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if others, ok := fields[2].(setting.DurationField); len(fields) != 4 || !ok || others.Key != "metadata.refresh.others" || others.Value != 604800 {
		t.Fatalf("Tuning = %+v, want the three tiers and the changes switch, every other Work weekly", fields)
	}
}

func isRange(err error) bool {
	_, ok := errors.AsType[*setting.RangeError](err)
	return ok
}

func isUnknown(err error) bool {
	_, ok := errors.AsType[*setting.UnknownKeyError](err)
	return ok
}

func TestASmallerArtworkStoreTrimsTheStoreAtOnce(t *testing.T) {
	s := openShop(t)
	filler, err := artwork.OpenStore(t.Context(), s.artwork, s.local, func(context.Context) (int64, error) { return 1 << 20, nil })
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		svg := `<svg xmlns="http://www.w3.org/2000/svg"><!--` + strings.Repeat(strconv.Itoa(i), 40<<20) + `--></svg>`
		w := artwork.Want{Ref: artwork.Ref{Source: "tmdb-images", Shape: artwork.ShapePoster, Path: strconv.Itoa(i)}, Size: artwork.SizeMedium}
		if _, err := filler.Put(t.Context(), w, artwork.Image{Type: "image/svg+xml", Body: []byte(svg)}); err != nil {
			t.Fatal(err)
		}
	}
	if n, size := stored(t, s.artwork); n != 3 {
		t.Fatalf("before the change the store holds %d files of %d bytes, want all 3", n, size)
	}

	s.set(t, map[string]any{metadata.ArtworkStore.Key: 100})

	if n, size := stored(t, s.artwork); n != 2 || size > 100<<20 {
		t.Errorf("saved at 100 MB the store holds %d files of %d bytes, want 2 within 100 MB", n, size)
	}
}

func stored(t *testing.T, dir string) (n int, size int64) {
	t.Helper()
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", d.Name(), err)
		}
		n, size = n+1, size+info.Size()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return n, size
}

func TestClearingTheCacheReachesAHolderOfflineAtTheTime(t *testing.T) {
	s := openShop(t, home, box)
	via := func(n role.NodeID) {
		t.Helper()
		s.roster.may = []role.NodeID{n}
		s.fetch(t)
	}
	calls := func(n role.NodeID) int { return s.sent[n].count() }

	via(home)
	via(box)
	via(home)
	via(box)
	if calls(home) != 1 || calls(box) != 1 {
		t.Fatalf("TMDB heard home %d and box %d times, want once each, the second fetch cached", calls(home), calls(box))
	}

	s.roster.may = []role.NodeID{home}
	s.m.ClearCache(identity.Admin{})
	via(home)
	if calls(home) != 2 {
		t.Fatalf("after clearing, TMDB heard home %d times, want a second call", calls(home))
	}

	via(box)
	if calls(box) != 2 {
		t.Fatalf("box, offline when the cache was cleared, reached TMDB %d times, want a second call", calls(box))
	}
	via(box)
	if calls(box) != 2 {
		t.Fatalf("box reached TMDB %d times, want its new answer cached", calls(box))
	}

	s.set(t, map[string]any{"metadata.cacheFor": 0})
	via(box)
	if calls(box) != 3 {
		t.Fatalf("with no cache time, box reached TMDB %d times, want every fetch sent", calls(box))
	}
}
