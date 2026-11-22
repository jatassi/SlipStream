package deluge_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/deluge"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/clientfake/clientfaketest"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// webClient calls the web UI's JSON-RPC with the session cookie its
// sign-in set.
type webClient struct {
	t       *testing.T
	f       faketest.Served
	cookies []*http.Cookie
}

func (c *webClient) call(method string, params ...any) json.RawMessage {
	c.t.Helper()
	body, err := json.Marshal(map[string]any{"method": method, "params": params, "id": 1})
	if err != nil {
		c.t.Fatal(err)
	}
	r, _ := http.NewRequestWithContext(c.t.Context(), http.MethodPost, c.f.Address.URL("/json"), bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for _, k := range c.cookies {
		r.AddCookie(k)
	}
	res, err := c.f.Client.Do(r)
	if err != nil {
		c.t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if k := res.Cookies(); len(k) > 0 {
		c.cookies = k
	}
	out, _ := io.ReadAll(res.Body)
	var reply struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(out, &reply); err != nil || string(reply.Error) != "null" {
		c.t.Fatalf("%s: %s", method, out)
	}
	return reply.Result
}

func TestASeedModeAddAdoptsTheFilesAndTheNextAnnounceRises(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cat, shared := clientfaketest.Tracked(t)
		d := deluge.Daemon{Host: "127.0.0.1", Port: 58846, Folder: "/downloads", Dir: t.TempDir(), Rate: 1}
		c := deluge.Client{Node: "den", Version: "2.1", Password: "orchard", Daemons: []deluge.Daemon{d}, AutoConnect: 1}
		whole, short := cat.Releases[0], cat.Releases[1]
		clientfaketest.Place(t, filepath.Join(d.Dir, "Movies"), whole, clientfaketest.Size)
		clientfaketest.Place(t, filepath.Join(d.Dir, "Movies"), short, clientfaketest.Size/2)
		f := faketest.InMemory(t, deluge.Playing(c), scenario.Scenario{Seed: "deluge"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
		web := &webClient{t: t, f: f}
		web.call("auth.login", c.Password)
		for _, r := range cat.Releases {
			web.call("core.add_torrent_file", r.Name+".torrent", base64.StdEncoding.EncodeToString(r.Metainfo), map[string]any{"download_location": "/downloads/Movies", "seed_mode": true})
		}
		status := func() map[string]struct {
			Progress     float64 `json:"progress"`
			NextAnnounce int64   `json:"next_announce"`
		} {
			var out map[string]struct {
				Progress     float64 `json:"progress"`
				NextAnnounce int64   `json:"next_announce"`
			}
			if err := json.Unmarshal(web.call("core.get_torrents_status", map[string]any{}, []string{"progress", "next_announce"}), &out); err != nil {
				t.Fatal(err)
			}
			return out
		}
		got := status()
		if got[whole.InfoHash.String()].Progress != 100 || got[short.InfoHash.String()].Progress != 50 {
			t.Fatalf("after the adds the torrents read %+v, want 100 over the whole file and 50 over the short one", got)
		}
		var next []int64
		for _, wait := range []time.Duration{0, 10 * time.Second, 21 * time.Second} {
			time.Sleep(wait)
			next = append(next, status()[whole.InfoHash.String()].NextAnnounce)
		}
		if next[0] != 30 || next[1] != 20 || next[2] != 30 {
			t.Fatalf("next_announce read %v at 0s, 10s and 31s, want 30, 20, then 30 again after the announce at 30s", next)
		}
	})
}
