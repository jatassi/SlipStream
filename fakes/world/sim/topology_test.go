package sim_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// uploadRate is how many bytes a second an uploading engine's torrents
// send.
const uploadRate = 1 << 20

// uploading is an engine whose every torrent has uploaded uploadRate
// bytes for each whole second since it was added.
type uploading struct {
	engine.Engine
	mu    sync.Mutex
	added map[engine.Handle]time.Time
}

func upload(open func(engine.Config) (engine.Engine, error)) func(engine.Config) (engine.Engine, error) {
	return func(cfg engine.Config) (engine.Engine, error) {
		e, err := open(cfg)
		return &uploading{Engine: e, added: map[engine.Handle]time.Time{}}, err
	}
}

func (u *uploading) Add(p engine.AddParams) (engine.Handle, error) {
	h, err := u.Engine.Add(p)
	if err == nil {
		u.mu.Lock()
		u.added[h] = time.Now()
		u.mu.Unlock()
	}
	return h, err
}

func (u *uploading) Status(h engine.Handle) (engine.Status, error) {
	st, err := u.Engine.Status(h)
	u.mu.Lock()
	defer u.mu.Unlock()
	if at, ok := u.added[h]; ok {
		st.Uploaded = int64(time.Since(at)/time.Second) * uploadRate
	}
	return st, err
}

type topologyView struct {
	Sentence string `json:"sentence"`
	Nodes    []struct {
		ID        string            `json:"id"`
		Name      string            `json:"name"`
		State     string            `json:"state"`
		Probes    []worldtest.Probe `json:"probes"`
		Resources struct {
			Downloading *struct {
				Torrents []struct {
					GrabID string `json:"grabId"`
					Title  string `json:"title"`
					Done   int64  `json:"done"`
					Wanted int64  `json:"wanted"`
				} `json:"torrents"`
			} `json:"downloading"`
		} `json:"resources"`
	} `json:"nodes"`
	Links []struct {
		A      string `json:"a"`
		B      string `json:"b"`
		Dialer string `json:"dialer"`
		State  string `json:"state"`
	} `json:"links"`
}

type trafficView struct {
	Nodes []struct {
		ID       string `json:"id"`
		UpBps    int64  `json:"upBps"`
		Torrents []struct {
			GrabID string `json:"grabId"`
			Done   int64  `json:"done"`
		} `json:"torrents"`
	} `json:"nodes"`
}

// nextTraffic reads the live stream until a nodes.traffic sample in which
// Node id uploads.
func nextTraffic(t *testing.T, r *bufio.Reader, id string) trafficView {
	t.Helper()
	event := ""
	for range 1000 {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("reading the live stream: %v", err)
		}
		if name, ok := strings.CutPrefix(line, "event: "); ok {
			event = strings.TrimSpace(name)
		}
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok || event != "nodes.traffic" {
			continue
		}
		var v trafficView
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			t.Fatalf("nodes.traffic carries %s: %v", data, err)
		}
		for _, n := range v.Nodes {
			if n.ID == id && n.UpBps > 0 {
				return v
			}
		}
	}
	t.Fatalf("the live stream carried no nodes.traffic sample with %s uploading", id)
	return trafficView{}
}

// TestTheTopologyDrawsSeedboxsTorrentAndTrafficCarriesItsUpload walks a
// Grab to imported on seedbox, whose engine then uploads 1 MiB a second.
// The topology lists the torrent under seedbox's Downloading with the
// release's title, and the link seedbox dialed; nodes.traffic carries the
// upload rate and the torrent's progress to an Admin's live stream.
func TestTheTopologyDrawsSeedboxsTorrentAndTrafficCarriesItsUpload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, home(t), sim.WithWalk(), onSeedbox(func(e *node.Env) { e.Engine = upload(e.Engine) }))
		search(t, w, addOrchard(t, w))
		g := await(t, w)
		expectImported(t, g)
		a := w.API()
		box := seedbox(t, a)

		topo := decode[topologyView](t, "getTopology", a.Admin("home", http.MethodGet, "/api/v1/nodes/topology", ""))
		if topo.Sentence != "Both Nodes are online." || len(topo.Nodes) != 2 || topo.Nodes[1].ID != box.ID || topo.Nodes[1].State != "online" {
			t.Fatalf("the topology reads %+v, want home and seedbox online", topo)
		}
		dl := topo.Nodes[1].Resources.Downloading
		if dl == nil || len(dl.Torrents) != 1 || dl.Torrents[0].GrabID != g.ID || dl.Torrents[0].Title != release ||
			dl.Torrents[0].Wanted == 0 || dl.Torrents[0].Done != dl.Torrents[0].Wanted {
			t.Fatalf("seedbox's Downloading reads %+v, want the Grab's torrent, titled %s and done", dl, release)
		}
		if l := topo.Links; len(l) != 1 || l[0].A != topo.Nodes[0].ID || l[0].B != box.ID || l[0].Dialer != box.ID || l[0].State != "up" {
			t.Fatalf("the topology's links are %+v, want home's link to seedbox, up and dialed by seedbox", l)
		}

		home := entryOf(t, w, "home")
		signedIn := home.signIn()
		stream := home.send(http.MethodGet, "/api/v1/events", "", signedIn.session, "")
		defer func() { _ = stream.Body.Close() }()
		sample := nextTraffic(t, bufio.NewReader(stream.Body), box.ID)
		for _, n := range sample.Nodes {
			if n.ID != box.ID {
				continue
			}
			if n.UpBps != uploadRate || len(n.Torrents) != 1 || n.Torrents[0].GrabID != g.ID || n.Torrents[0].Done != dl.Torrents[0].Done {
				t.Fatalf("nodes.traffic reads seedbox as %+v, want %d bytes a second up and the Grab's torrent done", n, uploadRate)
			}
		}
	})
}
