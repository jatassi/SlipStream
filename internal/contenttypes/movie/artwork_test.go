package movie_test

import (
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/imagecdn"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/providers/omdb"
	"github.com/jatassi/slipstream/internal/providers/tmdb"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// paths records the path of each request a transport sends.
type paths struct {
	next http.RoundTripper
	mu   sync.Mutex
	got  []string
}

func (p *paths) RoundTrip(r *http.Request) (*http.Response, error) {
	p.mu.Lock()
	p.got = append(p.got, r.URL.Path)
	p.mu.Unlock()
	return p.next.RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

func (p *paths) sent() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Sorted(slices.Values(p.got))
}

func TestAddPrefetchesPosterMediumAndBackdropLarge(t *testing.T) {
	sv := serve(t, catalog(t))
	cdn := faketest.Loopback(t, imagecdn.Fake, catalog(t))
	images := &paths{next: cdn.Client.Transport}
	client := &http.Client{Transport: hosts{
		sv.tmdb.Address.AddrPort.String(): sv.tmdb.Client.Transport,
		sv.omdb.Address.AddrPort.String(): sv.omdb.Client.Transport,
		cdn.Address.AddrPort.String():     images,
	}}
	lines := metadata.NewLines(client, "SlipStream/test", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	src := movie.Def.OpenProvider(contenttype.ProviderDeps{
		TMDB: tmdb.New(lines, tmdb.Endpoints{API: endpoint.MustParseURL(sv.tmdb.Address.URL("/3")), Images: endpoint.MustParseURL(cdn.Address.URL(imagecdn.Root))}, tmdb.DefaultTimings()),
		OMDb: omdb.New(lines, omdb.Endpoints{API: endpoint.MustParseURL(sv.omdb.Address.URL(""))}, omdb.DefaultTimings()),
		Log:  slog.New(slog.DiscardHandler),
	}, nil)
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(movie.Def))...)
	meta := metadatatest.Artwork(t, lib, map[works.TypeID]metadata.Source{movie.Def.ID(): src}, client, metadata.DefaultTimings())
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{movie.Def.ID(): library()}, Timings: works.DefaultTimings(),
		Source: meta, Artwork: meta, Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}

	tree, _, err := w.Add(t.Context(), identity.Admin{}, movie.Def.ID(), "9000001", works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	poster, backdrop := tree.Artwork["poster"], tree.Artwork["backdrop"]
	want := slices.Sorted(slices.Values([]string{imagecdn.Root + "/w342" + poster.Path, imagecdn.Root + "/w1280" + backdrop.Path}))
	deadline := time.Now().Add(10 * time.Second)
	for len(images.sent()) < len(want) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := images.sent(); !slices.Equal(got, want) {
		t.Fatalf("after the add the image host was asked for %v, want the poster at medium and the backdrop at large: %v", got, want)
	}
	for _, at := range []struct {
		ref  artwork.Ref
		size artwork.Size
	}{{poster, artwork.SizeMedium}, {backdrop, artwork.SizeLarge}} {
		img, err := meta.Artwork(t.Context(), metadatatest.Minter.Token(at.ref), at.size)
		if err != nil || img.Type != "image/jpeg" || len(img.Body) == 0 {
			t.Fatalf("the %s %s answered %s of %d bytes, %v; want the JPEG prefetched", at.size, at.ref.Shape, img.Type, len(img.Body), err)
		}
	}
	if got := images.sent(); !slices.Equal(got, want) {
		t.Fatalf("serving the prefetched images asked the host for %v, want nothing past %v", got, want)
	}
}
