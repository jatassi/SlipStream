package mediaserverlink_test

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/jellyfin"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// started records the file of each play a holder reports started.
type started struct {
	*holder
	mu    sync.Mutex
	files []string
}

func (s *started) Report(ctx context.Context, tx *storage.Tx[storage.Local], r *msv1.Report) error {
	if p := r.GetPlayStarted(); p != nil {
		s.mu.Lock()
		s.files = append(s.files, p.GetSession().GetServerPath())
		s.mu.Unlock()
	}
	return s.holder.Report(ctx, tx, r)
}

// TestAWatchStartedAgainKeepsThePlayAndItsVersion plays Heat's 1080p
// version from its 2160p item on a fake Jellyfin, then has Den answer 500
// to every item read and starts the watch again with a new settings
// version: the play keeps its version without a read, so it never ends
// and no other play starts.
func TestAWatchStartedAgainKeepsThePlayAndItsVersion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const (
			uhd = "/media/Movies/Heat (1995)/Heat (1995) - 2160p.mkv"
			hd  = "/media/Movies/Heat (1995)/Heat (1995) - 1080p.mkv"
		)
		media := t.TempDir()
		for _, p := range []string{uhd, hd} {
			file := filepath.Join(media, filepath.FromSlash(p[len("/media/"):]))
			if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(file, []byte(p), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		srv := jellyfin.Server{
			Name: "Den", Version: "12.4.0", ID: "ddc0b5b7f2824180aae9f2f245dd00ca", Keys: []string{"key-1", "key-2"},
			Users: []jellyfin.User{{Name: "ada", Password: "ada-password", Admin: true}}, Root: "/media",
			Libraries: []jellyfin.Library{{Name: "Movies", Type: "movies", Folders: []string{"/media/Movies"}, Scanned: true}},
		}
		f := faketest.InMemory(t, jellyfin.Playing(srv), scenario.Scenario{Seed: "holder"}, faketest.WithMedia(media), faketest.As("Den"))
		c := &control{version: 1}
		h := &started{holder: &holder{intents: make(chan *msv1.Intent), plays: map[string]int{}}}
		l := mediaserverlink.NewLink(mediaserverlink.Config{
			Kinds: mediaserverlink.Kinds, HTTP: f.Client, Timings: link.DefaultTimings(), Log: slog.New(slog.NewTextHandler(t.Output(), nil)),
		})
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = l.Run(storagetest.Local(t, role.LocalSchema()), c)(ctx, h)
		}()
		t.Cleanup(func() {
			cancel()
			<-done
		})
		send := func(version int64) {
			h.intents <- &msv1.Intent{SessionsEveryMs: 15_000, Servers: []*msv1.Server{{
				Id: den.Bytes(), Kind: "jellyfin", Name: "Den", Address: f.Address.URL(""), Identity: srv.ID, SettingsVersion: version, DeviceId: "holder-device",
			}}}
			synctest.Wait()
		}
		polls := func(n int) {
			for range n {
				time.Sleep(15 * time.Second)
				synctest.Wait()
			}
		}
		send(1)
		f.Cue(t, "jellyfin-play", map[string]any{
			"path": uhd, "version": hd, "user": "ada", "player": "Living room", "playerKey": "living-room",
			"video": link.StreamDirect, "audio": link.StreamDirect, "subtitles": link.SubtitlesNone,
		})
		synctest.Wait()
		polls(1)
		if _, err := f.Faults.Set(fault.Rule{ID: "items", Target: "jellyfin", Route: "item", Fault: fault.Status{Code: http.StatusInternalServerError}}); err != nil {
			t.Fatal(err)
		}
		c.set(2)
		send(2)
		polls(3)
		want := []string{"responding at 1", "watching []", "play 1 started", "responding at 2", "watching [1]"}
		if got := h.reported(); !slices.Equal(got, want) {
			t.Errorf("the holder reported %q, want %q", got, want)
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		if !slices.Equal(h.files, []string{hd}) {
			t.Errorf("the plays started on %q, want %s", h.files, hd)
		}
	})
}
