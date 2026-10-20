package music_test

import (
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/contenttypes/music"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/coverart"
	"github.com/jatassi/slipstream/fakes/contenttypes/music/musicmeta"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/metadata/metadatatest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/works"
)

// hosts sends each request through the transport of the fake at its host.
type hosts map[string]http.RoundTripper

func (h hosts) RoundTrip(r *http.Request) (*http.Response, error) {
	return h[r.URL.Host].RoundTrip(r) //nolint:wrapcheck // a RoundTripper hands its transport's error on unchanged
}

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
	return slices.Clone(p.got)
}

func TestMusicCoverArtUsesTheSameStore(t *testing.T) {
	meta := faketest.Loopback(t, musicmeta.Fake, catalog(t))
	covers := faketest.Loopback(t, coverart.Fake, catalog(t))
	fetched := &paths{next: covers.Client.Transport}
	client := &http.Client{Transport: hosts{meta.Address.AddrPort.String(): meta.Client.Transport, covers.Address.AddrPort.String(): fetched}}
	lines := metadata.NewLines(client, "SlipStream/test ( github.com/jatassi/SlipStream )", metadata.DefaultTimings(), rand.New(rand.NewPCG(1, 2))) //nolint:gosec // a fixed draw the test replays
	src := music.Def.OpenProvider(contenttype.ProviderDeps{Lines: lines, Log: slog.New(slog.DiscardHandler)},
		&music.Endpoints{Metadata: endpoint.MustParseURL(meta.Address.URL("/ws/2")), CoverArt: endpoint.MustParseURL(covers.Address.URL(""))})
	lib := storagetest.Library(t, node.LibrarySchemas(contenttype.NewSet(music.Def))...)
	m := metadatatest.Artwork(t, lib, map[works.TypeID]metadata.Source{music.Def.ID(): src}, client, metadata.DefaultTimings())
	w, err := works.Open(t.Context(), works.Config{
		Library: lib, Catalogs: map[works.TypeID]works.Catalog{music.Def.ID(): library()}, Timings: works.DefaultTimings(),
		Source: m, Artwork: m, Changed: discard[works.WorkChanged]{}, TypesChanged: discard[works.ContentTypesChanged]{}, Log: slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}

	tree, _, err := w.Add(t.Context(), identity.Admin{}, music.Def.ID(), lanterns, works.Adding{})
	if err != nil {
		t.Fatal(err)
	}
	cover := tree.Artwork["cover"]
	medium, small := "/release-group/"+cover.Path+"/front-500", "/release-group/"+cover.Path+"/front-250"
	deadline := time.Now().Add(10 * time.Second)
	for len(fetched.sent()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got := fetched.sent(); !slices.Equal(got, []string{medium}) {
		t.Fatalf("after adding The Lanterns the cover-art host was asked for %v, want its cover at medium, %s", got, medium)
	}
	for range 2 {
		for _, size := range []artwork.Size{artwork.SizeMedium, artwork.SizeSmall} {
			img, err := m.Artwork(t.Context(), metadatatest.Minter.Token(cover), size)
			if err != nil || img.Type != "image/jpeg" || len(img.Body) == 0 {
				t.Fatalf("the %s cover answered %s of %d bytes, %v; want a JPEG", size, img.Type, len(img.Body), err)
			}
		}
	}
	if got := fetched.sent(); !slices.Equal(got, []string{medium, small}) {
		t.Fatalf("serving the cover twice at two sizes asked the host for %v, want the prefetched medium then small once: %v", got, []string{medium, small})
	}
}
