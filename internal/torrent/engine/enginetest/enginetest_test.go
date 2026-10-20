package enginetest_test

import (
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

const single = "d4:infod6:lengthi5e4:name5:a.mkv12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee"

func statusOf(t *testing.T, e engine.Engine, h engine.Handle) (engine.Status, error) {
	t.Helper()
	fake, ok := e.(*enginetest.Engine)
	if !ok {
		t.Fatalf("engine %T is not the in-memory one", e)
	}
	return fake.Status(h)
}

func add(seed bool) engine.AddParams {
	return engine.AddParams{Metainfo: []byte(single), SeedMode: seed}
}

func TestAnAddedTorrentIsTrackedUntilRemoved(t *testing.T) {
	e, err := enginetest.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.3:0")})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := e.ListenAddr(); got != netip.MustParseAddrPort("127.0.0.3:6881") {
		t.Fatalf("ListenAddr = %s, want 127.0.0.3:6881", got)
	}
	seeding, _ := e.Add(add(true))
	downloading, _ := e.Add(add(false))
	for h, want := range map[engine.Handle]engine.State{seeding: engine.StateSeeding, downloading: engine.StateDownloading} {
		if s, err := statusOf(t, e, h); err != nil || s.State != want {
			t.Fatalf("Status(%d) = %+v, %v; want %s", h, s, err, want)
		}
	}
	if err := e.Remove(seeding, false); err != nil {
		t.Fatal(err)
	}
	if _, err := statusOf(t, e, seeding); !errors.Is(err, engine.ErrUnknownTorrent) {
		t.Fatalf("Status of a removed torrent = %v, want unknown torrent", err)
	}
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	var got []engine.Event
	for ev := range e.Events() {
		got = append(got, ev)
	}
	want := []engine.Event{
		engine.Added{Torrent: 1}, engine.StateChanged{Torrent: 1, State: engine.StateSeeding},
		engine.Added{Torrent: 2}, engine.StateChanged{Torrent: 2, State: engine.StateDownloading},
		engine.Removed{Torrent: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %+v, want %+v", got, want)
	}
	if _, err := e.Add(add(false)); !errors.Is(err, engine.ErrClosed) {
		t.Fatalf("Add after Close = %v, want engine closed", err)
	}
}

func TestAReaderThatFallsBehindGetsDropped(t *testing.T) {
	e, _ := enginetest.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.3:0")})
	for range 40 {
		if _, err := e.Add(add(false)); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 63 {
		if ev := <-e.Events(); ev == (engine.Dropped{}) {
			t.Fatalf("event %d = Dropped, want the events queued before the drop first", i)
		}
	}
	if ev := <-e.Events(); ev != (engine.Dropped{}) {
		t.Fatalf("the event after those queued before falling behind = %+v, want Dropped", ev)
	}
}

func TestFilesListTheMetainfosFiles(t *testing.T) {
	e, _ := enginetest.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.3:0")})
	multi := "d4:infod5:filesld6:lengthi6e4:pathl6:Extras11:trailer.mkveed6:lengthi84e4:pathl11:feature.mkveee" +
		"4:name7:Orchard12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee"
	for meta, want := range map[string][]engine.File{
		single: {{Index: 0, Path: "a.mkv", Size: 5}},
		multi:  {{Index: 0, Path: "Orchard/Extras/trailer.mkv", Size: 6}, {Index: 1, Path: "Orchard/feature.mkv", Size: 84}},
	} {
		h, err := e.Add(engine.AddParams{Metainfo: []byte(meta)})
		if err != nil {
			t.Fatal(err)
		}
		if got, err := e.Files(h); err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("Files = %+v, %v; want %+v", got, err, want)
		}
	}
	if _, err := e.Add(engine.AddParams{Metainfo: []byte("not bencode")}); !errors.Is(err, engine.CategoryBdecode) {
		t.Errorf("Add of bad metainfo err = %v, want a bdecode error", err)
	}
	if _, err := e.Files(99); !errors.Is(err, engine.ErrUnknownTorrent) {
		t.Errorf("Files of an unknown torrent err = %v", err)
	}
}

func TestADownloadWithAPayloadFinishesIntoItsSavePathOnceResumed(t *testing.T) {
	m, err := bencode.DecodeMetainfo([]byte(single))
	if err != nil {
		t.Fatal(err)
	}
	e, err := enginetest.Payloads(map[bencode.InfoHash][]byte{m.InfoHash: []byte("hello")})(engine.Config{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	h, err := e.Add(engine.AddParams{Metainfo: []byte(single), SavePath: dir})
	if err != nil {
		t.Fatal(err)
	}
	if s, err := statusOf(t, e, h); err != nil || s.State != engine.StateDownloading || !s.Paused || s.Done != 0 {
		t.Fatalf("added, Status = %+v, %v; want paused downloading with nothing done", s, err)
	}
	if err := e.Resume(h); err != nil {
		t.Fatal(err)
	}
	if s, err := statusOf(t, e, h); err != nil || s.State != engine.StateSeeding || s.Done != 5 || s.Wanted != 5 || s.Totals.Downloaded != 5 {
		t.Fatalf("resumed, Status = %+v, %v; want seeding 5 of 5, all downloaded", s, err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "a.mkv")); err != nil || string(got) != "hello" {
		t.Fatalf("a.mkv holds %q, %v; want hello", got, err)
	}
	_ = e.Close()
	var got []engine.Event
	for ev := range e.Events() {
		got = append(got, ev)
	}
	want := []engine.Event{
		engine.Added{Torrent: h}, engine.StateChanged{Torrent: h, State: engine.StateDownloading},
		engine.Resumed{Torrent: h}, engine.StateChanged{Torrent: h, State: engine.StateSeeding}, engine.Finished{Torrent: h},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestAPacedDownloadThatFindsTheDiskFullPausesUntilResumed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m, err := bencode.DecodeMetainfo([]byte(single))
		if err != nil {
			t.Fatal(err)
		}
		w := enginetest.NewWorld()
		w.Payload(m.InfoHash, []byte("hello"))
		w.Transfer(m.InfoHash, 1, 0)
		e, err := w.Open(engine.Config{})
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		h, err := e.Add(engine.AddParams{Metainfo: []byte(single), SavePath: dir})
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Resume(h); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		w.DiskFull(true)
		if s, err := statusOf(t, e, h); err != nil || !s.Paused || s.Done != 2 {
			t.Fatalf("on a full disk two seconds into a 1 B/s download, Status = %+v, %v; want paused with 2 bytes done", s, err)
		}
		w.DiskFull(false)
		if err := e.Resume(h); err != nil {
			t.Fatal(err)
		}
		time.Sleep(3 * time.Second)
		if s, err := statusOf(t, e, h); err != nil || s.State != engine.StateSeeding || s.Done != 5 {
			t.Fatalf("resumed with room again, Status = %+v, %v; want seeding 5 of 5", s, err)
		}
		want := []engine.Event{
			engine.Resumed{Torrent: h}, engine.FileError{Torrent: h, Err: engine.ErrDiskFull}, engine.Paused{Torrent: h},
			engine.Resumed{Torrent: h}, engine.Finished{Torrent: h},
		}
		if got := runEvents(e); !reflect.DeepEqual(got, want) {
			t.Fatalf("events = %#v, want %#v", got, want)
		}
	})
}

// runEvents closes e and lists its events that start, pause, fail or
// finish a torrent.
func runEvents(e engine.Engine) []engine.Event {
	_ = e.Close()
	var got []engine.Event
	for ev := range e.Events() {
		switch ev.(type) {
		case engine.FileError, engine.Paused, engine.Resumed, engine.Finished:
			got = append(got, ev)
		}
	}
	return got
}
