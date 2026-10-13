package enginetest_test

import (
	"errors"
	"net/netip"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/torrent/engine/enginetest"
)

func TestAnAddedTorrentIsTrackedUntilRemoved(t *testing.T) {
	e, err := enginetest.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.3:0")})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := e.ListenAddr(); got != netip.MustParseAddrPort("127.0.0.3:6881") {
		t.Fatalf("ListenAddr = %s, want 127.0.0.3:6881", got)
	}
	seeding, _ := e.Add(engine.AddParams{SeedMode: true})
	downloading, _ := e.Add(engine.AddParams{})
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
	if _, err := e.Add(engine.AddParams{}); !errors.Is(err, engine.ErrClosed) {
		t.Fatalf("Add after Close = %v, want engine closed", err)
	}
}

func TestAReaderThatFallsBehindGetsDropped(t *testing.T) {
	e, _ := enginetest.Open(engine.Config{ListenInterface: netip.MustParseAddrPort("127.0.0.3:0")})
	for range 40 {
		if _, err := e.Add(engine.AddParams{}); err != nil {
			t.Fatal(err)
		}
	}
	for range 64 {
		<-e.Events()
	}
	if _, err := e.Add(engine.AddParams{}); err != nil {
		t.Fatal(err)
	}
	if ev := <-e.Events(); ev != (engine.Dropped{}) {
		t.Fatalf("the first event after falling behind = %+v, want Dropped", ev)
	}
}
