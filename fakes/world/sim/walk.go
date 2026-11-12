package sim

import (
	"encoding/json"
	"log/slog"
	"maps"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/clientfake"
	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/omdb"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/tmdb"
	"github.com/jatassi/slipstream/fakes/tvdb"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/internal/setup"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/contenttypes/movie"
	"github.com/jatassi/slipstream/internal/contenttypes/series"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
	"github.com/jatassi/slipstream/internal/transfers"
)

// PayloadSize is the payload's size of a release of one video in the
// walk: five Transfer chunks, the last one short.
const PayloadSize = 4*transfers.ChunkSize + transfers.ChunkSize/2

// WithWalk runs the skeleton's walk on a Scenario with releases and a
// library: every Node registers the movie and series types, fake-tmdb,
// fake-tvdb, fake-omdb and fixture-site serve on the network at
// 127.0.0.1, each Node's engine finishes a seeded release at once with its
// Payload, its swarm sends a public seeded release's metadata to a magnet
// link that reaches it through the World's tracker or DHT, each Node
// probes a release's file at its video's size, and the library is seeded
// once the Nodes are set up.
func WithWalk() Option { return func(o *options) { o.walk = true } }

// Listed edits each release fixture-site lists.
func Listed(edit func(*site.Release)) Option { return func(o *options) { o.listed = edit } }

// Edit changes the Env of the Node named by its first argument at each of
// its starts.
func Edit(edit func(name string, e *node.Env)) Option {
	return func(o *options) { o.edits = append(o.edits, edit) }
}

// Payload is the payload of a release of one video in the walk:
// PayloadSize bytes that depend only on its name.
func Payload(release string) []byte { return releases.Filler(release, PayloadSize) }

type walk struct {
	faults   *fault.Plan
	mediaDir string
	site     fake.Address
	// announce is each site's tracker URL, by label.
	announce map[string]string
	payloads map[bencode.InfoHash][]byte
	// served are the public seeded releases, whose seeders send their
	// metadata to a magnet link.
	served []bencode.Metainfo
	hashes map[string]bencode.InfoHash
	// catalog is the releases the fake Download Clients serve.
	catalog *releases.Catalog
	probes  map[string]media.Record
	// fails are the probes that answer unreadable, by the same keys.
	fails map[string]string
	// clients is where each Node reaches the fake Download Client of each
	// kind, and the folder that client downloads into, by Node then kind.
	clients map[string]map[string]clientAt
	// envs are the client fakes' Envs, by kind.
	envs map[string]fake.Env
}

type clientAt struct{ url, folder string }

// startWalk serves the walk's fakes on local and points endpoints at them.
func startWalk(t *testing.T, s scenario.Scenario, local *fault.Host, fakes *servers, listed func(*site.Release), endpoints *node.EndpointsBlock) *walk {
	t.Helper()
	base := shortTempDir(t)
	w := &walk{
		mediaDir: filepath.Join(base, "media"),
		payloads: map[bencode.InfoHash][]byte{},
		hashes:   map[string]bencode.InfoHash{},
		probes:   map[string]media.Record{},
		fails:    map[string]string{},
		clients:  map[string]map[string]clientAt{},
		envs:     map[string]fake.Env{},
	}
	imdb := imdbIDs(s)
	w.announce = siteAnnounces(s)
	var listing []site.Release
	catalog := &releases.Catalog{}
	w.catalog = catalog
	for _, r := range s.Releases {
		payload, meta, hash, err := torrent(s, r)
		if err != nil {
			t.Fatal(err)
		}
		w.hashes[r.Name] = hash
		rel := releases.Release{Release: r, Dir: base, Size: int64(len(payload)), BLAKE3: blake3.Sum256(payload), Metainfo: meta, InfoHash: hash}
		writePayload(t, &rel)
		catalog.Releases = append(catalog.Releases, rel)
		if r.Seeded {
			w.payloads[hash] = payload
		}
		if r.Seeded && r.Public {
			m, err := bencode.DecodeMetainfo(meta)
			if err != nil {
				t.Fatal(err)
			}
			w.served = append(w.served, m)
		}
		probes := releaseProbes(r)
		maps.Copy(w.probes, probes.Of)
		maps.Copy(w.fails, probes.Fail)
		size := int64(len(payload))
		if r.AdvertisedSize > 0 {
			size = int64(r.AdvertisedSize)
		}
		item := site.Release{
			Name: r.Name, Size: size, InfoHash: hash.String(), TMDBID: r.TMDBID, IMDbID: imdb[r.TMDBID], TVDB: r.TVDB, Seeders: int64(r.Seeders), Torrent: meta,
			Link: r.Link, Trackerless: r.Trackerless, Sites: r.Sites, DownloadFactor: r.DownloadFactor, Untagged: r.Untagged, Published: r.Published,
		}
		if listed != nil {
			listed(&item)
		}
		listing = append(listing, item)
	}
	w.writeFolders(t, s)
	log := slog.New(slog.NewTextHandler(t.Output(), nil))
	faults := fault.NewPlan(append([]fault.Front{{Target: fault.Target(site.Fake.Name), Layers: fault.HTTP}}, clientFronts(s)...), nil)
	w.faults = faults
	for _, spec := range []fake.Spec{tmdb.Fake, omdb.Fake, tvdb.Fake} {
		server, err := spec.New(s, fake.NewEnv(s, spec, log, faults, nil))
		if err != nil {
			t.Fatal(err)
		}
		spec.Endpoints(endpoints, fakes.serve(t, local, server))
	}
	w.site = fakes.serve(t, local, fake.HTTP(fake.NewEnv(s, site.Fake, log, faults, nil), site.Routes(s, listing, func(label, _ string, _ bool) (string, error) { return w.announce[label], nil })))
	site.Fake.Endpoints(endpoints, w.site)
	w.serveClients(t, s, local, fakes, catalog, filepath.Join(base, "fakes"), log)
	return w
}

// torrent is r's payload in the walk, its files back to back, and its
// torrent, which announces as fixture-seeders.
func torrent(s scenario.Scenario, r scenario.Release) (payload, meta []byte, hash bencode.InfoHash, err error) {
	name, files := r.Name, []releases.TorrentFile(nil)
	if r.Video != nil {
		name += ".mkv"
	}
	for _, f := range payloadFiles(r) {
		payload = append(payload, f.data...)
		if r.Video == nil {
			files = append(files, releases.TorrentFile{Path: strings.TrimPrefix(f.path, r.Name+"/"), Size: int64(len(f.data))})
		}
	}
	meta, hash, err = releases.Metainfo(name, files, payload, "https://127.0.0.1/announce/"+releases.Passkey(s, releases.Seeders), r.Public)
	return payload, meta, hash, err
}

// writePayload writes rel's payload files under its Dir, as a seeder's
// save path holds them, and lists them in rel.
func writePayload(t *testing.T, rel *releases.Release) {
	t.Helper()
	for _, f := range payloadFiles(rel.Release) {
		file := releases.File{Path: f.path, Size: int64(len(f.data)), BLAKE3: blake3.Sum256(f.data)}
		if err := os.MkdirAll(filepath.Dir(rel.Disk(file)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(rel.Disk(file), f.data, 0o600); err != nil {
			t.Fatal(err)
		}
		rel.Files = append(rel.Files, file)
	}
}

type payloadFile struct {
	// path is the file's in the torrent, '/' separated.
	path string
	data []byte
}

// payloadFiles are r's payload files in the walk: Payload as <name>.mkv
// for a release of one video, else each file's Size bytes of its own
// under a <name> folder.
func payloadFiles(r scenario.Release) []payloadFile {
	if r.Video != nil {
		return []payloadFile{{r.Name + ".mkv", Payload(r.Name)}}
	}
	out := make([]payloadFile, len(r.Files))
	for i, f := range r.Files {
		at := r.Name + "/" + f.Path
		out[i] = payloadFile{at, releases.Filler(at, int64(f.Size))}
	}
	return out
}

// unreadableLine is what ffprobe says of bytes that are no media file.
const unreadableLine = "Invalid data found when processing input"

// releaseProbes is how a Node probes r's videos in the walk, keyed by
// their paths in its torrent, as the process world's encodes read: a
// release of one video at its size, and each video of a release of files
// as the library folder file it is. An unreadable file fails.
func releaseProbes(r scenario.Release) mediatest.Prober {
	p := mediatest.Prober{Of: map[string]media.Record{}, Fail: map[string]string{}}
	if r.Video != nil {
		p.Of[r.Name+".mkv"] = media.Record{
			Version: media.Version, Container: "matroska,webm", Duration: time.Duration(r.Video.Seconds) * time.Second, Video: media.Video{Codec: "h264", Width: r.Video.Width, Height: r.Video.Height},
		}
	}
	for _, f := range r.Files {
		at := r.Name + "/" + f.Path
		switch {
		case f.Video != nil:
			p.Of[at] = mediafix.Want(f.Media())
		case f.Unreadable:
			p.Fail[at] = unreadableLine
		}
	}
	return p
}

// writeFolders writes each file of the Scenario's library folders under
// the media dir as a few bytes of its own, which the Nodes probe as the
// process world's encode reads, since the simulation tier runs no ffmpeg.
func (w *walk) writeFolders(t *testing.T, s scenario.Scenario) {
	t.Helper()
	if s.Library == nil {
		return
	}
	for _, f := range s.Library.Folders {
		for _, m := range f.Files {
			rel := f.Folder + "/" + m.Path
			file := filepath.Join(w.mediaDir, filepath.FromSlash(rel))
			if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(rel), 0o600); err != nil {
				t.Fatal(err)
			}
			w.probes[rel] = mediafix.Want(m)
		}
	}
}

// siteAnnounces are each fixture-site site's tracker URL, by label.
func siteAnnounces(s scenario.Scenario) map[string]string {
	out := map[string]string{}
	for _, labels := range scenario.FixtureSites {
		for _, label := range labels {
			out[label] = "https://127.0.0.1/announce/" + releases.Passkey(s, label)
		}
	}
	return out
}

// imdbIDs is the IMDb id of each movie the Scenario's TMDB lists, by its
// TMDB id.
func imdbIDs(s scenario.Scenario) map[int64]string {
	ids := map[int64]string{}
	if s.TMDB != nil {
		for _, m := range s.TMDB.Movies {
			ids[m.ID] = m.IMDbID
		}
	}
	return ids
}

// clientFronts are the fronts of each Node's Download Clients, one a
// Node and kind.
func clientFronts(s scenario.Scenario) []fault.Front {
	var out []fault.Front
	for _, n := range s.Nodes {
		for _, c := range n.DownloadClients {
			out = append(out, clientFront(c.Kind, n.Name))
		}
	}
	return out
}

func clientFront(kind, origin string) fault.Front {
	return fault.Front{Target: fault.Target(kind), Origin: origin, Layers: fault.Conn | fault.HTTP}
}

// serveClients serves the fake of each kind of Download Client the
// Scenario's Nodes have, one front a Node, each downloading into its own
// folder under dir.
func (w *walk) serveClients(t *testing.T, s scenario.Scenario, local *fault.Host, fakes *servers, catalog *releases.Catalog, dir string, log *slog.Logger) {
	t.Helper()
	runDir := t.TempDir()
	sink, err := events.Create(runDir, func(events.Record) []events.Record { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sink.Close() })
	served := map[string]fake.Server{}
	for _, n := range s.Nodes {
		for _, c := range n.DownloadClients {
			spec, ok := world.FakeNamed(c.Kind)
			if !ok {
				t.Fatalf("the world has no fake for %s", c.Kind)
			}
			server, ok := served[c.Kind]
			if !ok {
				env := fake.NewEnv(s, spec, log, w.faults, sink)
				env.Releases, env.Files, env.Dir = catalog, runDir, filepath.Join(dir, c.Kind)
				if server, err = spec.New(s, env); err != nil {
					t.Fatal(err)
				}
				served[c.Kind] = server
				w.envs[c.Kind] = env
			}
			l, err := local.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			at := fake.Address{AddrPort: netip.MustParseAddrPort(l.Addr().String())}
			fakes.serveOn(server, w.faults.Listener(l, clientFront(c.Kind, n.Name)))
			if w.clients[n.Name] == nil {
				w.clients[n.Name] = map[string]clientAt{}
			}
			w.clients[n.Name][c.Kind] = clientAt{url: spec.URL(at), folder: clientfake.Folder(filepath.Join(dir, c.Kind), n.Name)}
		}
	}
}

// ClientFolder is the folder the client of kind on the Node named on
// downloads into, as that Node sees it.
func (w *World) ClientFolder(on, kind string) string { return w.walk.clients[on][kind].folder }

// Cue plays the cue name of the client fake of kind with args, as
// fakeworld cue does.
func (w *World) Cue(kind, name string, args any) {
	w.t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		w.t.Fatal(err)
	}
	if _, err := w.walk.envs[kind].Cue(w.t.Context(), name, raw); err != nil {
		w.t.Fatalf("cue %s: %v", name, err)
	}
}

// ClientURL is where the Node named on reaches its client of kind.
func (w *World) ClientURL(on, kind string) string { return w.walk.clients[on][kind].url }

// Faults is the Plan the walk's fakes read. A rule's Target names the fake,
// such as fixture-site.
func (w *World) Faults() *fault.Plan { return w.walk.faults }

// Announce is the tracker URL the torrents and magnet links of the site
// labeled label name, one of the trackers each Node's swarm runs.
func (w *World) Announce(label string) string { return w.walk.announce[label] }

// InfoHash is the walk's torrent of release.
func (w *World) InfoHash(release string) bencode.InfoHash { return w.walk.hashes[release] }

// seed creates the Scenario's library on the Controller, as the process
// world's seedLibrary does.
func (w *walk) seed(run *World, ready []node.Ready) {
	t := run.t
	t.Helper()
	err := setup.Library(t.Context(), run.client, run.scenario, ready, run.claims, w.mediaDir, map[string]fake.Address{site.Fake.Name: w.site}, nil, func(on, kind string) (string, string) {
		at := w.clients[on][kind]
		return at.url, at.folder
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (w *walk) env(n *Node, e *node.Env) {
	e.Types = contenttype.NewSet(movie.Def, series.Def)
	e.Engine = func(cfg engine.Config) (engine.Engine, error) {
		swarm := enginetest.NewWorld()
		for _, announce := range w.announce {
			swarm.Tracker(announce, enginetest.TrackerRules{Interval: 30 * time.Minute})
		}
		for h, payload := range w.payloads {
			swarm.Payload(h, payload)
		}
		for _, m := range w.served {
			swarm.Serve(m)
		}
		n.mu.Lock()
		n.swarm = swarm
		n.mu.Unlock()
		return swarm.Open(cfg)
	}
	e.Media = mediatest.Prober{Of: w.probes, Fail: w.fails}
}
