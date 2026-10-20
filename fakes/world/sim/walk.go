package sim

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"math/rand/v2"
	"net/netip"
	"path/filepath"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/transfers"
)

// PayloadSize is each release payload's size in the walk: five Transfer
// chunks, the last one short.
const PayloadSize = 4*transfers.ChunkSize + transfers.ChunkSize/2

// WithWalk runs the skeleton's walk on a Scenario with releases and a
// library: every Node registers the movie type, fake-tmdb, fake-omdb and
// fixture-site serve on the network at 127.0.0.1, each Node's engine
// finishes a seeded release at once with its Payload, each Node probes a
// release's file at its video's size, and the library is seeded once the
// Nodes are set up.
func WithWalk() Option { return func(o *options) { o.walk = true } }

// Listed edits each release fixture-site lists.
func Listed(edit func(*site.Release)) Option { return func(o *options) { o.listed = edit } }

// Edit changes the Env of the Node named by its first argument at each of
// its starts.
func Edit(edit func(name string, e *node.Env)) Option {
	return func(o *options) { o.edits = append(o.edits, edit) }
}

// Payload is the release's payload in the walk: PayloadSize bytes that
// depend only on its name.
func Payload(release string) []byte {
	b := make([]byte, PayloadSize)
	_, _ = rand.NewChaCha8(sha256.Sum256([]byte(release))).Read(b)
	return b
}

type walk struct {
	mediaDir string
	site     fake.Address
	payloads map[bencode.InfoHash][]byte
	hashes   map[string]bencode.InfoHash
	probes   map[string]media.Probe
	cancel   context.CancelFunc
	done     []chan error
}

// startWalk serves the walk's fakes on local and points endpoints at them.
func startWalk(t *testing.T, s scenario.Scenario, local *fault.Host, listed func(*site.Release), endpoints *node.EndpointsBlock) *walk {
	t.Helper()
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	w := &walk{
		mediaDir: filepath.Join(shortTempDir(t), "media"),
		payloads: map[bencode.InfoHash][]byte{},
		hashes:   map[string]bencode.InfoHash{},
		probes:   map[string]media.Probe{},
		cancel:   cancel,
	}
	imdb := map[int64]string{}
	for _, m := range s.TMDB.Movies {
		imdb[m.ID] = m.IMDbID
	}
	announce := "https://127.0.0.1/announce/" + releases.Passkey(s)
	var listing []site.Release
	for _, r := range s.Releases {
		payload := Payload(r.Name)
		meta, hash, err := releases.Metainfo(r.Name+".mkv", payload, announce)
		if err != nil {
			t.Fatal(err)
		}
		w.hashes[r.Name] = hash
		if r.Seeded {
			w.payloads[hash] = payload
		}
		w.probes[r.Name+".mkv"] = media.Probe{
			Duration: time.Duration(r.Video.Seconds) * time.Second, Container: "matroska", Codec: "h264", Width: r.Video.Width, Height: r.Video.Height,
		}
		size := int64(PayloadSize)
		if r.AdvertisedSize > 0 {
			size = int64(r.AdvertisedSize)
		}
		item := site.Release{
			Name: r.Name, Size: size, InfoHash: hash.String(), TMDBID: r.TMDBID, IMDbID: imdb[r.TMDBID], Seeders: int64(r.Seeders), Torrent: meta,
		}
		if listed != nil {
			listed(&item)
		}
		listing = append(listing, item)
	}
	log, faults := slog.New(slog.NewTextHandler(t.Output(), nil)), fault.NewPlan(nil, nil)
	movies, err := tmdb.Fake.New(s, fake.NewEnv(s, tmdb.Fake, log, faults, nil))
	if err != nil {
		t.Fatal(err)
	}
	tmdb.Fake.Endpoints(endpoints, w.serve(ctx, t, local, movies))
	ratings, err := omdb.Fake.New(s, fake.NewEnv(s, omdb.Fake, log, faults, nil))
	if err != nil {
		t.Fatal(err)
	}
	omdb.Fake.Endpoints(endpoints, w.serve(ctx, t, local, ratings))
	w.site = w.serve(ctx, t, local, fake.HTTP(fake.NewEnv(s, site.Fake, log, faults, nil), site.Routes(site.APIKey(s), listing, func(string) (string, error) { return announce, nil })))
	return w
}

func (w *walk) serve(ctx context.Context, t *testing.T, local *fault.Host, server fake.Server) fake.Address {
	t.Helper()
	l, err := local.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, l) }()
	w.done = append(w.done, done)
	return fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String())}
}

// InfoHash is the walk's torrent of release.
func (w *World) InfoHash(release string) bencode.InfoHash { return w.walk.hashes[release] }

// seed creates the Scenario's library on the Controller, as the process
// world's seedLibrary does.
func (w *walk) seed(world *World, ready []node.Ready) {
	t := world.t
	t.Helper()
	err := setup.Library(t.Context(), world.client, world.scenario, ready, world.claims, w.mediaDir, map[string]fake.Address{site.Fake.Name: w.site})
	if err != nil {
		t.Fatal(err)
	}
}

func (w *walk) env(e *node.Env) {
	e.Types = contenttype.NewSet(movie.Def)
	e.Engine = enginetest.Payloads(w.payloads)
	e.Media = mediatest.Prober{Of: w.probes}
}

func (w *walk) stop(t *testing.T) {
	w.cancel()
	for _, done := range w.done {
		if err := <-done; err != nil {
			t.Errorf("a walk fake: %v", err)
		}
	}
}
