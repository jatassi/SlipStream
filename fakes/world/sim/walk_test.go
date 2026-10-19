package sim_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/site"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/transfers"
)

const (
	release = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"
	placed  = "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv"
	// settled is longer than any wait the walk takes in fake time.
	settled = 10 * time.Minute
)

var walked = []string{"fetching", "downloading", "probing", "transferring", "importing", "imported"}

type grabView struct {
	ID          string `json:"id"`
	NodeID      string `json:"nodeId"`
	FetchNodeID string `json:"fetchNodeId"`
	InfoHash    string `json:"infoHash"`
	Stage       string `json:"stage"`
	Failure     *struct {
		Stage   string `json:"stage"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"failure"`
	History []struct {
		Stage string `json:"stage"`
	} `json:"history"`
}

func (g grabView) stages() []string {
	var out []string
	for _, e := range g.History {
		out = append(out, e.Stage)
	}
	return out
}

func decode[T any](t *testing.T, what string, r worldtest.Reply) T {
	t.Helper()
	var v T
	if r.Status != http.StatusOK {
		t.Fatalf("%s = %d %s, want 200", what, r.Status, r.Body)
	}
	if err := json.Unmarshal([]byte(r.Body), &v); err != nil {
		t.Fatalf("%s = %s: %v", what, r.Body, err)
	}
	return v
}

// addOrchard adds the Scenario's movie monitored and returns its Work.
func addOrchard(t *testing.T, w *sim.World) string {
	t.Helper()
	added := decode[struct {
		Work struct {
			ID string `json:"id"`
		} `json:"work"`
	}](t, "addWork", w.API().Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"movie","key":"9000001","monitor":true}`))
	return added.Work.ID
}

// search presses Search on work and returns its one run.
func search(t *testing.T, w *sim.World, work string) string {
	t.Helper()
	reply := w.API().Admin("home", http.MethodPost, "/api/v1/works/"+work+"/search", "")
	var started struct {
		SearchIDs []string `json:"searchIds"`
	}
	if err := json.Unmarshal([]byte(reply.Body), &started); err != nil || reply.Status != http.StatusAccepted || len(started.SearchIDs) != 1 {
		t.Fatalf("searchWork = %d %s, want 202 with one search", reply.Status, reply.Body)
	}
	return started.SearchIDs[0]
}

func grabsListed(t *testing.T, w *sim.World) []grabView {
	t.Helper()
	return decode[struct {
		Grabs []grabView `json:"grabs"`
	}](t, "listGrabs", w.API().Admin("home", http.MethodGet, "/api/v1/grabs", "")).Grabs
}

// await waits in fake time until the world's one Grab is done, and
// returns it.
func await(t *testing.T, w *sim.World) grabView {
	t.Helper()
	for range settled / time.Second {
		if gs := grabsListed(t, w); len(gs) == 1 && (gs[0].Stage == "imported" || gs[0].Stage == "failed") {
			return gs[0]
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("no Grab finished within %s: %+v", settled, grabsListed(t, w))
	return grabView{}
}

func expectImported(t *testing.T, g grabView) {
	t.Helper()
	if g.Stage != "imported" || !slices.Equal(g.stages(), walked) {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported after %v", g.Stage, g.Failure, g.stages(), walked)
	}
}

func expectFailed(t *testing.T, g grabView, stage, code, message string) {
	t.Helper()
	if g.Stage != "failed" || g.Failure == nil || g.Failure.Stage != stage || g.Failure.Code != code || g.Failure.Message != message {
		t.Fatalf("the Grab ended %s with %+v after %v, want failed at %s with %s: %s", g.Stage, g.Failure, g.stages(), stage, code, message)
	}
}

func fileRows(t *testing.T, w *sim.World) int64 {
	t.Helper()
	var n int64
	if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT count(*) FROM files_file").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func expectPlaced(t *testing.T, w *sim.World) {
	t.Helper()
	synctest.Wait()
	dir := w.Manifest().MediaDir
	var got []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		got = append(got, filepath.ToSlash(rel))
		return err //nolint:wrapcheck // the test reports it as is
	})
	want := []string{"movies/" + storing.MarkerName, placed}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("the media dir holds %v, %v; want exactly %v", got, err, want)
	}
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(placed)))
	if err != nil {
		t.Fatal(err)
	}
	if blake3.Sum256(data) != blake3.Sum256(sim.Payload(release)) {
		t.Fatal("the placed file's BLAKE3 is not the release payload's")
	}
	if entries, err := os.ReadDir(filepath.Join(dir, "movies", ".slipstream-staging")); err != nil || len(entries) != 0 {
		t.Fatalf("the staging folder holds %v, %v; want nothing", entries, err)
	}
	if n := fileRows(t, w); n != 1 {
		t.Fatalf("the library holds %d file rows, want 1", n)
	}
}

// watermarks is the result seq home applied from each of seedbox's epochs.
func watermarks(t *testing.T, w *sim.World) []int64 {
	t.Helper()
	rows, err := w.Node("home").Half().Library.QueryContext(t.Context(), "SELECT seq FROM role_watermark WHERE node_id = ? ORDER BY seq", nodeID(t, w, "seedbox"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []int64
	for rows.Next() {
		var seq int64
		if err := rows.Scan(&seq); err != nil {
			t.Fatal(err)
		}
		out = append(out, seq)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

func onSeedbox(edit func(e *node.Env)) sim.Option {
	return sim.Edit(func(name string, e *node.Env) {
		if name == "seedbox" {
			edit(e)
		}
	})
}

// TestAControllerRestartedAtEachStageImportsOnce stops home just before
// each step of the walk and starts it again on its link port, as a
// Controller on the default port restarts: the Grab resumes and ends
// imported with one file.
func TestAControllerRestartedAtEachStageImportsOnce(t *testing.T) {
	for _, stage := range []grabs.Stage{grabs.Fetching, grabs.Downloading, grabs.Probing, grabs.Transferring, grabs.Importing} {
		t.Run(string(stage), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				reached := make(chan struct{})
				var held atomic.Bool
				w := sim.Launch(t, home(t), sim.WithWalk(), sim.Edit(func(_ string, e *node.Env) {
					e.Test.BeforeStep = func(ctx context.Context, s grabs.Stage) {
						if s == stage && held.CompareAndSwap(false, true) {
							close(reached)
							<-ctx.Done()
						}
					}
				}))
				search(t, w, addOrchard(t, w))
				<-reached
				h := w.Node("home")
				link := h.Ready().Listeners.Link
				h.Stop()
				h.Start(func(e *node.Env) { e.Environ = append(e.Environ, "SLIPSTREAM_LINK_LISTEN="+link.String()) })
				expectImported(t, await(t, w))
				expectPlaced(t, w)
			})
		})
	}
}

// heldFinish is an engine whose Finished events wait for release.
type heldFinish struct {
	engine.Engine
	events chan engine.Event
	closed chan struct{}
}

func holdFinished(open func(engine.Config) (engine.Engine, error), finished chan<- struct{}, release <-chan struct{}) func(engine.Config) (engine.Engine, error) {
	return func(cfg engine.Config) (engine.Engine, error) {
		e, err := open(cfg)
		if err != nil {
			return nil, err
		}
		h := &heldFinish{Engine: e, events: make(chan engine.Event), closed: make(chan struct{})}
		go func() {
			defer close(h.events)
			for ev := range e.Events() {
				if _, ok := ev.(engine.Finished); ok {
					finished <- struct{}{}
					<-release
				}
				select {
				case h.events <- ev:
				case <-h.closed:
					return
				}
			}
		}()
		return h, nil
	}
}

func (h *heldFinish) Events() <-chan engine.Event { return h.events }

func (h *heldFinish) Close() error {
	close(h.closed)
	return h.Engine.Close()
}

// TestAResultMadeWhileTheWorkersLinkIsDownLandsOnce finishes seedbox's
// download while its link to home is stalled: after the heal the result
// lands once, home applies it once, and the Grab walks on.
func TestAResultMadeWhileTheWorkersLinkIsDownLandsOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		finished, release := make(chan struct{}), make(chan struct{})
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Engine = holdFinished(e.Engine, finished, release) }))
		search(t, w, addOrchard(t, w))
		<-finished
		path := w.Path("home", "seedbox")
		flip(t, path, fault.Stall{})
		time.Sleep(partitioned)
		close(release)
		synctest.Wait()
		if gs := grabsListed(t, w); len(gs) != 1 || gs[0].Stage != "downloading" {
			t.Fatalf("while partitioned the Grabs are %+v, want one downloading", gs)
		}
		path.Clear("")
		expectImported(t, await(t, w))
		if got := watermarks(t, w); !slices.Equal(got, []int64{1}) {
			t.Fatalf("home applied seedbox's results through %v, want [1]: one result, applied once", got)
		}
		expectPlaced(t, w)
	})
}

// adds records the info hash of each torrent an engine adds.
type adds struct {
	engine.Engine
	mu     *sync.Mutex
	hashes *[]bencode.InfoHash
}

func (a adds) Add(p engine.AddParams) (engine.Handle, error) {
	if m, err := bencode.DecodeMetainfo(p.Metainfo); err == nil {
		a.mu.Lock()
		*a.hashes = append(*a.hashes, m.InfoHash)
		a.mu.Unlock()
	}
	return a.Engine.Add(p)
}

// TestAWorkerRestartedAfterCompletedReAddsAndChangesNothing restarts
// seedbox with its link to home stalled: once home had time to answer, it
// adds the torrent again from its cached intent, and its second Completed,
// once applied, changes nothing.
func TestAWorkerRestartedAfterCompletedReAddsAndChangesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var added []bencode.InfoHash
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) {
			open := e.Engine
			e.Engine = func(cfg engine.Config) (engine.Engine, error) {
				eng, err := open(cfg)
				return adds{eng, &mu, &added}, err
			}
		}))
		search(t, w, addOrchard(t, w))
		before := await(t, w)
		expectImported(t, before)
		if got := watermarks(t, w); !slices.Equal(got, []int64{1}) {
			t.Fatalf("home applied seedbox's results through %v, want [1]", got)
		}

		path := w.Path("home", "seedbox")
		flip(t, path, fault.Stall{})
		time.Sleep(partitioned)
		box := w.Node("seedbox")
		box.Stop()
		box.Start()
		timings := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
		time.Sleep(timings.RedialMax + timings.Handshake)
		synctest.Wait()
		hash, err := bencode.ParseInfoHash(before.InfoHash)
		if err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		if !slices.Equal(added, []bencode.InfoHash{hash, hash}) {
			t.Fatalf("seedbox's engines added %v, want %s twice: once from the intent, once from its cache while partitioned", added, hash)
		}
		mu.Unlock()

		path.Clear("")
		time.Sleep(redialed)
		synctest.Wait()
		if got := watermarks(t, w); !slices.Equal(got, []int64{2}) {
			t.Fatalf("home applied seedbox's results through %v, want [2]: the second Completed applied", got)
		}
		after := await(t, w)
		if !slices.Equal(after.History, before.History) || after.Stage != "imported" {
			t.Fatalf("after the second Completed the Grab is %s after %+v, want unchanged %+v", after.Stage, after.History, before.History)
		}
		expectPlaced(t, w)
	})
}

// faultyChunks fails the first read of chunk cut and flips a byte in the
// first read of chunk flip, by index from 0; -1 is none.
type faultyChunks struct {
	io.ReaderAt
	cut, flip int
	cutDone   *atomic.Bool
	flipDone  *atomic.Bool
}

func (f faultyChunks) ReadAt(b []byte, off int64) (int, error) {
	chunk := int(off / transfers.ChunkSize)
	if chunk == f.cut && f.cutDone.CompareAndSwap(false, true) {
		return 0, errors.New("the disk went away")
	}
	n, err := f.ReaderAt.ReadAt(b, off)
	if chunk == f.flip && f.flipDone.CompareAndSwap(false, true) {
		b[0] ^= 1
	}
	return n, err //nolint:wrapcheck // the source wraps the read error
}

// TestATransferResumes breaks a Transfer's stream and corrupts a chunk on
// the source: the receiver asks again only for what it has not verified,
// and the placed file's BLAKE3 is the payload's.
func TestATransferResumes(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cut, flip int
		wants     [][]int
	}{
		{name: "after a break at chunk 3 of 5", cut: 2, flip: -1, wants: [][]int{{0, 1, 2, 3, 4}, {2, 3, 4}}},
		{name: "after a flipped byte in chunk 2", cut: -1, flip: 1, wants: [][]int{{0, 1, 2, 3, 4}, {1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				var mu sync.Mutex
				var wants [][]int
				var cut, flip atomic.Bool
				w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) {
					e.Test.Transfers = transfers.Hooks{
						Read: func(r io.ReaderAt) io.ReaderAt {
							return faultyChunks{ReaderAt: r, cut: tc.cut, flip: tc.flip, cutDone: &cut, flipDone: &flip}
						},
						Wanted: func(_ bencode.InfoHash, _ int, chunks []int) {
							mu.Lock()
							defer mu.Unlock()
							wants = append(wants, chunks)
						},
					}
				}))
				search(t, w, addOrchard(t, w))
				expectImported(t, await(t, w))
				mu.Lock()
				if !slices.EqualFunc(wants, tc.wants, slices.Equal[[]int]) {
					t.Fatalf("seedbox was asked for chunks %v, want %v", wants, tc.wants)
				}
				mu.Unlock()
				expectPlaced(t, w)
			})
		})
	}
}

// TestASearchWhileTheIndexersHomeNodeIsDisconnectedGrabsNothing stalls
// seedbox, the Indexer's home Node, before the search: the run ends
// indexer_unavailable and makes no Grab.
func TestASearchWhileTheIndexersHomeNodeIsDisconnectedGrabsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithWalk())
		work := addOrchard(t, w)
		flip(t, w.Path("home", "seedbox"), fault.Stall{})
		time.Sleep(partitioned)
		run := search(t, w, work)
		time.Sleep(time.Minute)
		var searches struct {
			Searches []struct {
				ID, IndexerID, StartedAt string
				FinishedAt               *string
			} `json:"searches"`
		}
		reply := w.API().Admin("home", http.MethodGet, "/api/v1/works/"+work+"/searches", "")
		if err := json.Unmarshal([]byte(reply.Body), &searches); err != nil || len(searches.Searches) != 1 || searches.Searches[0].FinishedAt == nil {
			t.Fatalf("listSearches = %s, want one finished run", reply.Body)
		}
		s := searches.Searches[0]
		worldtest.Expect(t, "listSearches", reply, worldtest.Reply{Status: 200, Body: `{"searches":[{"id":"` + run + `","itemId":"` + itemOf(t, w, work) +
			`","indexerId":"` + s.IndexerID + `","startedAt":"` + s.StartedAt + `","finishedAt":"` + *s.FinishedAt +
			`","outcome":"indexer_unavailable","candidates":0,"refused":[],"grabId":null,"notes":[]}]}` + "\n"})
		if gs := grabsListed(t, w); len(gs) != 0 {
			t.Fatalf("listGrabs = %+v, want none", gs)
		}
	})
}

func itemOf(t *testing.T, w *sim.World, work string) string {
	t.Helper()
	return decode[struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}](t, "getWork", w.API().Admin("home", http.MethodGet, "/api/v1/works/"+work, "")).Items[0].ID
}

// TestATorrentWhoseInfoHashDiffersFromTheIndexersFailsFetching lists the
// release with another infohash attribute than its .torrent's.
func TestATorrentWhoseInfoHashDiffersFromTheIndexersFailsFetching(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wrong := strings.Repeat("ab", 20)
		w := sim.Launch(t, home(t), sim.WithWalk(), sim.Listed(func(r *site.Release) { r.InfoHash = wrong }))
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectFailed(t, g, "fetching", "infohash_mismatch", "the .torrent's info hash is "+w.InfoHash(release).String()+" and the Indexer's is "+wrong)
		if got := g.stages(); !slices.Equal(got, []string{"fetching", "failed"}) {
			t.Fatalf("the Grab's history is %v, want [fetching failed]", got)
		}
	})
}

// TestAPayloadProbedAt480pFillsNothing probes the 1080p release's file at
// 480p on seedbox, which the starter Target doesn't accept: probing fails
// fills_nothing, no Transfer stream opens, and seedbox keeps the torrent
// in its intent.
func TestAPayloadProbedAt480pFillsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var asked atomic.Bool
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) {
			e.Media = mediatest.Prober{Of: map[string]media.Probe{release + ".mkv": {Container: "matroska", Codec: "h264", Width: 640, Height: 480}}}
			e.Test.Transfers.Wanted = func(bencode.InfoHash, int, []int) { asked.Store(true) }
		}))
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectFailed(t, g, "probing", "fills_nothing", "a SDTV file fills no Target the Work still wants")
		if got := g.stages(); !slices.Equal(got, []string{"fetching", "downloading", "probing", "failed"}) {
			t.Fatalf("the Grab's history is %v, want [fetching downloading probing failed]", got)
		}
		if asked.Load() {
			t.Fatal("seedbox's source was asked for chunks")
		}
		var opened int64
		if err := w.Node("home").Half().Local.QueryRowContext(t.Context(), "SELECT count(*) FROM transfers_transfer").Scan(&opened); err != nil || opened != 0 {
			t.Fatalf("home recorded %d Transfers (%v), want none: no stream of kind 2 opened", opened, err)
		}
		var intent []string
		err := w.Node("home").Half().Library.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			in, err := grabs.DownloadIntent(ctx, r, nodeID(t, w, "seedbox"))
			for _, tor := range in.GetTorrents() {
				intent = append(intent, bencode.InfoHash(tor.GetInfoHash()).String())
			}
			return err
		})
		if err != nil || !slices.Equal(intent, []string{g.InfoHash}) {
			t.Fatalf("seedbox's Downloading intent is %v (%v), want the torrent %s still in it", intent, err, g.InfoHash)
		}
	})
}

// TestCreatingAnIndexerWithNoSearchingNodeInStepConflicts cuts seedbox,
// the one Node that holds Searching, off: before the offline grace no Node
// may search, so createIndexer answers 409.
func TestCreatingAnIndexerWithNoSearchingNodeInStepConflicts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := home(t)
		s.Library.Indexers = nil
		w := sim.Launch(t, s, sim.WithWalk())
		synctest.Wait()
		flip(t, w.Path("home", "seedbox"), fault.Stall{})
		link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
		time.Sleep(link.Keepalive + link.WriteTimeout)
		synctest.Wait()
		worldtest.Expect(t, "createIndexer", w.API().Admin("home", http.MethodPost, "/api/v1/indexers",
			`{"name":"Fixture Site","url":"http://127.0.0.1:1/torznab","apiKey":"key"}`), worldtest.Reply{
			Status: http.StatusConflict,
			Body:   `{"status":409,"code":"no_searching_node","title":"No Node holds Searching. Give one the Searching Role first."}` + "\n",
		})
	})
}
