package enginetest_test

import (
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

const single = "d4:infod6:lengthi5e4:name5:a.mkv12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaaee"

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
		if s, err := e.Status(h); err != nil || s.State != want {
			t.Fatalf("Status(%d) = %+v, %v; want %s", h, s, err, want)
		}
	}
	if err := e.Remove(seeding); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Status(seeding); !errors.Is(err, engine.ErrUnknownTorrent) {
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
	for range 64 {
		<-e.Events()
	}
	if _, err := e.Add(add(false)); err != nil {
		t.Fatal(err)
	}
	if ev := <-e.Events(); ev != (engine.Dropped{}) {
		t.Fatalf("the first event after falling behind = %+v, want Dropped", ev)
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

func TestADownloadWithAPayloadFinishesIntoItsSavePath(t *testing.T) {
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
	if s, err := e.Status(h); err != nil || s != (engine.Status{State: engine.StateSeeding, Done: 5, Wanted: 5}) {
		t.Fatalf("Status = %+v, %v; want seeding 5 of 5", s, err)
	}
	if got, err := os.ReadFile(filepath.Join(dir, "a.mkv")); err != nil || string(got) != "hello" {
		t.Fatalf("a.mkv holds %q, %v; want hello", got, err)
	}
	_ = e.Close()
	var got []engine.Event
	for ev := range e.Events() {
		got = append(got, ev)
	}
	want := []engine.Event{engine.Added{Torrent: h}, engine.StateChanged{Torrent: h, State: engine.StateSeeding}, engine.Finished{Torrent: h}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}
