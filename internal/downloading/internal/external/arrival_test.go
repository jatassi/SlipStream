package external_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	fakertorrent "github.com/jatassi/slipstream/fakes/rtorrent"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/downloading/internal/external"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/rtorrent"
)

// lostCheck fails the first d.check_hash request before it reaches
// rTorrent, as a dropped connection does.
type lostCheck struct {
	inner http.RoundTripper
	mu    sync.Mutex
	lost  bool
}

func (l *lostCheck) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("read the request: %w", err)
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	l.mu.Lock()
	lose := !l.lost && bytes.Contains(body, []byte("d.check_hash"))
	l.lost = l.lost || lose
	l.mu.Unlock()
	if lose {
		return nil, errors.New("connection reset")
	}
	return l.inner.RoundTrip(req) //nolint:wrapcheck // the session wraps it
}

// place copies the payload file at from into folder, as a Move places
// it.
func place(t *testing.T, folder, from string) {
	t.Helper()
	payload, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(folder, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(folder, filepath.Base(from)), payload, 0o600); err != nil {
		t.Fatal(err)
	}
}

// arrivals is every Arrival the actor reported, oldest first.
func arrivals(h *holder) []*dlv1.Arrival {
	var out []*dlv1.Arrival
	for _, rep := range h.reported() {
		for _, tr := range rep.GetTorrents().GetReports() {
			if a := tr.GetArrival(); a != nil {
				out = append(out, a)
			}
		}
	}
	return out
}

// TestACompleteAddThatFailsAfterTheLoadIsCheckedOnTheNextPoll adds a
// Move's arriving torrent to rTorrent, whose d.check_hash after the load
// is lost. The next poll lists it stopped and unchecked, and the actor
// sends it once more rather than report the check failed.
func TestACompleteAddThatFailsAfterTheLoadIsCheckedOnTheNextPoll(t *testing.T) {
	for _, version := range fakertorrent.Versions {
		t.Run(version, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c := fakertorrent.Client{Node: node, Version: version, Dir: t.TempDir(), Folder: "/downloads", Rate: 100}
				catalog := faketest.Catalog(t, size, "orchard")
				rel := catalog.Releases[0]
				place(t, filepath.Join(c.Dir, "Movies"), rel.Disk(rel.Files[0]))
				served := faketest.InMemory(t, fakertorrent.Playing(c), scenario.Scenario{Seed: "rtorrent"}, faketest.WithReleases(catalog), faketest.As(node))
				id, clientID := storage.NewUUID(), storage.NewUUID()
				ctl := &control{torrents: map[storage.UUID]*dlv1.TorrentResponse{
					id: {Metainfo: rel.Metainfo, ContentType: "Movies", Origin: dlv1.Origin_ORIGIN_ADDED},
				}}
				h := &holder{}
				share := external.Share{
					Client: &dlv1.Client{
						Id: clientID.Bytes(), Kind: dlv1.Kind_KIND_RTORRENT, Name: "rTorrent", Address: served.Address.URL(""),
						Label: clients.DefaultLabel, CheckEverySeconds: int64(checkEvery / time.Second), On: true, SettingsVersion: 1,
						ClientFolders: []*dlv1.ClientFolder{{Client: "/downloads", Node: c.Dir}},
					},
					Torrents: []*dlv1.Torrent{{
						Id: id.Bytes(), InfoHash: rel.InfoHash[:], Wanted: dlv1.Wanted_WANTED_PAUSED, ClientId: clientID.Bytes(), Arriving: true,
					}},
					StalledAfter: 10 * time.Minute, UnregisteredGap: 30 * time.Minute, MayStart: true,
				}
				actor := external.Start(context.Background(), external.Config{
					ID: clientID, Kind: clients.RTorrent, Open: rtorrent.Open, HTTP: &http.Client{Transport: &lostCheck{inner: served.Client.Transport}},
					Subfolder: func(name string) string { return name },
					Links:     func(string) (uint64, error) { return 1, nil },
					Timings:   timings, Local: storagetest.Local(t, downloading.Schema()), Holder: h, Control: ctl,
					Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
					Log:  slog.New(slog.DiscardHandler),
				}, share)
				t.Cleanup(actor.Halt)
				time.Sleep(3 * checkEvery)
				synctest.Wait()

				want := &dlv1.Arrival{Result: dlv1.ArrivalResult_ARRIVAL_RESULT_COMPLETE}
				if got := arrivals(h); len(got) != 1 || !proto.Equal(got[0], want) {
					t.Errorf("arrivals %v, want only %v", got, want)
				}
			})
		})
	}
}
