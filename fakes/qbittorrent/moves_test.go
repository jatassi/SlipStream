package qbittorrent_test

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/clientfake/clientfaketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/qbittorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
)

func call(t *testing.T, f faketest.Served, req *http.Request, out any) {
	t.Helper()
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%s: %s %s", req.URL.Path, res.Status, body)
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			t.Fatalf("%s: %v in %s", req.URL.Path, err, body)
		}
	}
}

func addSkippingCheck(t *testing.T, f faketest.Served, r releases.Release) {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("torrents", r.Name+".torrent")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(r.Metainfo)
	_ = w.WriteField("savepath", "/downloads/Movies")
	_ = w.WriteField("skip_checking", "true")
	_ = w.Close()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, f.Address.URL("/api/v2/torrents/add"), &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	call(t, f, req, nil)
}

func get(t *testing.T, f faketest.Served, path string, out any) {
	t.Helper()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, f.Address.URL(path), http.NoBody)
	call(t, f, req, out)
}

func TestACompleteAddAdoptsTheFilesAndTheReannounceCountdownRises(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cat, shared := clientfaketest.Tracked(t)
		c := qbittorrent.Client{Node: "den", Version: "4.6", Folder: "/downloads", Dir: t.TempDir(), Rate: 1}
		whole, short := cat.Releases[0], cat.Releases[1]
		clientfaketest.Place(t, filepath.Join(c.Dir, "Movies"), whole, clientfaketest.Size)
		clientfaketest.Place(t, filepath.Join(c.Dir, "Movies"), short, clientfaketest.Size/2)
		f := faketest.InMemory(t, qbittorrent.Playing(c), scenario.Scenario{Seed: "qbittorrent"}, faketest.WithReleases(cat), faketest.WithShared(shared), faketest.As(c.Node))
		addSkippingCheck(t, f, whole)
		addSkippingCheck(t, f, short)

		var info []struct {
			Hash     string  `json:"hash"`
			Progress float64 `json:"progress"`
		}
		get(t, f, "/api/v2/torrents/info", &info)
		progress := map[string]float64{}
		for _, i := range info {
			progress[i.Hash] = i.Progress
		}
		if progress[whole.InfoHash.String()] != 1 || progress[short.InfoHash.String()] != 0.5 {
			t.Fatalf("after the adds the progress is %v, want 1 over the whole file and 0.5 over the short one", progress)
		}

		reannounce := func() int64 {
			var p struct {
				Reannounce int64 `json:"reannounce"`
			}
			get(t, f, "/api/v2/torrents/properties?hash="+whole.InfoHash.String(), &p)
			return p.Reannounce
		}
		var got []int64
		for _, wait := range []time.Duration{0, 10 * time.Second, 21 * time.Second} {
			time.Sleep(wait)
			got = append(got, reannounce())
		}
		if got[0] != 30 || got[1] != 20 || got[2] != 30 {
			t.Fatalf("reannounce read %v at 0s, 10s and 31s, want 30, 20, then 30 again after the announce at 30s", got)
		}
	})
}
