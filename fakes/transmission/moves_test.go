package transmission_test

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

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/clientfake/clientfaketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/transmission"
)

// rpcClient speaks the legacy protocol to 4.0 and JSON-RPC 2.0 to 4.1,
// with the session id from the daemon's 409.
type rpcClient struct {
	t       *testing.T
	f       faketest.Served
	json2   bool
	session string
}

func (c *rpcClient) call(method string, args map[string]any) map[string]any {
	c.t.Helper()
	req := map[string]any{"method": method, "arguments": args}
	if c.json2 {
		req = map[string]any{"jsonrpc": "2.0", "method": method, "params": args, "id": 1}
	}
	body, _ := json.Marshal(req)
	for range 2 {
		r, _ := http.NewRequestWithContext(c.t.Context(), http.MethodPost, c.f.Address.URL("/transmission/rpc"), bytes.NewReader(body))
		r.Header.Set("X-Transmission-Session-Id", c.session)
		res, err := c.f.Client.Do(r)
		if err != nil {
			c.t.Fatal(err)
		}
		out, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if res.StatusCode == http.StatusConflict {
			c.session = res.Header.Get("X-Transmission-Session-Id")
			continue
		}
		var reply struct {
			Result    json.RawMessage `json:"result"`
			Arguments map[string]any  `json:"arguments"`
		}
		if err := json.Unmarshal(out, &reply); err != nil {
			c.t.Fatalf("%s: %v in %s", method, err, out)
		}
		if !c.json2 {
			return reply.Arguments
		}
		var result map[string]any
		_ = json.Unmarshal(reply.Result, &result)
		return result
	}
	c.t.Fatalf("%s: the daemon kept answering 409", method)
	return nil
}

func TestAVerifiedAddAdoptsTheFilesAndTheLastAnnounceRises(t *testing.T) {
	for _, c := range []struct {
		version                   string
		add, verify, start, get   string
		dir, percent, stats, last string
	}{
		{"4.0", "torrent-add", "torrent-verify", "torrent-start", "torrent-get", "download-dir", "percentDone", "trackerStats", "lastAnnounceTime"},
		{"4.1", "torrent_add", "torrent_verify", "torrent_start", "torrent_get", "download_dir", "percent_done", "tracker_stats", "last_announce_time"},
	} {
		t.Run(c.version, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				cat, shared := clientfaketest.Tracked(t)
				d := transmission.Daemon{Node: "den", Version: c.version, Folder: "/downloads", Dir: t.TempDir(), Rate: 1}
				whole, short := cat.Releases[0], cat.Releases[1]
				clientfaketest.Place(t, filepath.Join(d.Dir, "Movies"), whole, clientfaketest.Size)
				clientfaketest.Place(t, filepath.Join(d.Dir, "Movies"), short, clientfaketest.Size/2)
				f := faketest.InMemory(t, transmission.Playing(d), scenario.Scenario{Seed: "transmission"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(d.Node))
				rpc := &rpcClient{t: t, f: f, json2: c.version == "4.1"}
				for _, r := range cat.Releases {
					rpc.call(c.add, map[string]any{"metainfo": base64.StdEncoding.EncodeToString(r.Metainfo), c.dir: "/downloads/Movies", "paused": true})
				}
				hashes := []string{whole.InfoHash.String(), short.InfoHash.String()}
				rpc.call(c.verify, map[string]any{"ids": hashes})
				rpc.call(c.start, map[string]any{"ids": hashes})
				read := func() (percents []float64, last float64) {
					got := rpc.call(c.get, map[string]any{"ids": hashes, "fields": []string{c.percent, c.stats}})
					for _, raw := range got["torrents"].([]any) {
						row := raw.(map[string]any)
						percents = append(percents, row[c.percent].(float64))
					}
					first := got["torrents"].([]any)[0].(map[string]any)
					return percents, first[c.stats].([]any)[0].(map[string]any)[c.last].(float64)
				}
				percents, before := read()
				if percents[0] != 1 || percents[1] != 0.5 {
					t.Fatalf("verified, the torrents read %v done, want 1 over the whole file and 0.5 over the short one", percents)
				}
				time.Sleep(31 * time.Second)
				if _, after := read(); after != before+31 {
					t.Fatalf("31s on the last announce reads %v, want %v", after, before+31)
				}
			})
		})
	}
}
