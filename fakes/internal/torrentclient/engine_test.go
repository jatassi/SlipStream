package torrentclient_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/internal/torrentclient"
)

const size = 1000

func engine(t *testing.T, cfg torrentclient.Config, names ...string) (*torrentclient.Engine, *releases.Catalog) {
	t.Helper()
	cfg.Releases = faketest.Catalog(t, size, names...)
	if cfg.Dir == "" {
		cfg.Dir, cfg.Name = t.TempDir(), "/downloads"
	}
	if cfg.Rate == 0 {
		cfg.Rate = 100
	}
	return torrentclient.New(cfg), cfg.Releases
}

func add(t *testing.T, e *torrentclient.Engine, r releases.Release, folder string) torrentclient.Torrent {
	t.Helper()
	got, duplicate, err := e.Add(r.Metainfo, folder, []string{"slipstream"}, false)
	if err != nil || duplicate {
		t.Fatalf("Add(%s) = %v, %v", r.Name, duplicate, err)
	}
	return got
}

func get(t *testing.T, e *torrentclient.Engine, r releases.Release) torrentclient.Torrent {
	t.Helper()
	got, ok := e.Get(r.InfoHash)
	if !ok {
		t.Fatalf("the engine doesn't list %s", r.Name)
	}
	return got
}

func payload(t *testing.T, r releases.Release) []byte {
	t.Helper()
	b, err := os.ReadFile(r.Disk(r.Files[0]))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func onDisk(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestTheEngineWritesThePayloadOnTheClock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		e, c := engine(t, torrentclient.Config{Dir: dir, Name: "/downloads"}, "orchard")
		r := c.Releases[0]
		start := time.Now()
		add(t, e, r, "/downloads/Movies")
		file := filepath.Join(dir, "Movies", "orchard.mkv")

		time.Sleep(4 * time.Second)
		got := get(t, e, r)
		if got.Done != 400 || got.State != torrentclient.Downloading || got.Rates.Down != 100 {
			t.Fatalf("after 4s: done %d, state %d, down %d, want 400 bytes downloading at 100", got.Done, got.State, got.Rates.Down)
		}
		if b := onDisk(t, file); !bytes.Equal(b, payload(t, r)[:400]) {
			t.Fatalf("after 4s the file holds %d bytes, want the payload's first 400", len(b))
		}

		time.Sleep(6 * time.Second)
		got = get(t, e, r)
		if got.Done != size || got.State != torrentclient.Seeding || !got.CompletedAt.Equal(start.Add(10*time.Second)) {
			t.Fatalf("after 10s: done %d, state %d, completed %v, want complete and seeding at 10s", got.Done, got.State, got.CompletedAt.Sub(start))
		}
		if !bytes.Equal(onDisk(t, file), payload(t, r)) {
			t.Fatal("the complete file isn't the payload")
		}
		if got.Folder != "/downloads/Movies" || got.Files[0] != (torrentclient.File{Path: "orchard.mkv", Size: size, Done: size}) {
			t.Fatalf("folder %q, files %+v", got.Folder, got.Files)
		}
	})
}

func TestTheEngineWritesEachFileOfAPackFromItsOwnPayload(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		src := t.TempDir()
		pack := releases.Release{Dir: src}
		pack.Name = "Lantern.S01"
		var payload []byte
		var files []releases.TorrentFile
		for _, f := range []releases.TorrentFile{{Path: "Lantern.S01E01.mkv", Size: 600}, {Path: "Extras/Lantern.nfo", Size: 400}} {
			data := releases.Filler(f.Path, f.Size)
			file := releases.File{Path: pack.Name + "/" + f.Path, Size: f.Size}
			if err := os.MkdirAll(filepath.Dir(pack.Disk(file)), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(pack.Disk(file), data, 0o600); err != nil {
				t.Fatal(err)
			}
			pack.Files, payload, files = append(pack.Files, file), append(payload, data...), append(files, f)
		}
		var err error
		if pack.Metainfo, pack.InfoHash, err = releases.Metainfo(pack.Name, files, payload, "https://127.0.0.1/announce/faketest", false); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		e := torrentclient.New(torrentclient.Config{Dir: dir, Name: "/downloads", Rate: 100, Releases: &releases.Catalog{Releases: []releases.Release{pack}}})
		add(t, e, pack, "/downloads")

		time.Sleep(7 * time.Second)
		if got := get(t, e, pack); got.Done != 700 {
			t.Fatalf("after 7s: done %d, want 700", got.Done)
		}
		episode, nfo := filepath.Join(dir, "Lantern.S01", "Lantern.S01E01.mkv"), filepath.Join(dir, "Lantern.S01", "Extras", "Lantern.nfo")
		if !bytes.Equal(onDisk(t, episode), payload[:600]) || !bytes.Equal(onDisk(t, nfo), payload[600:700]) {
			t.Fatal("after 7s the episode isn't its payload or the nfo doesn't hold its first 100 bytes")
		}
		time.Sleep(3 * time.Second)
		if got := get(t, e, pack); got.State != torrentclient.Seeding || !bytes.Equal(onDisk(t, nfo), payload[600:]) {
			t.Fatalf("after 10s: state %d, want seeding with the nfo whole", got.State)
		}
	})
}

func TestTheEngineStopsATorrentAtItsSeedRatio(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e, c := engine(t, torrentclient.Config{SeedRatio: 1.5}, "orchard")
		r := c.Releases[0]
		add(t, e, r, "")
		time.Sleep(24 * time.Second)
		if got := get(t, e, r); got.State != torrentclient.Seeding || got.SeedGoalMet || got.Uploaded != 1400 {
			t.Fatalf("after 14s of seeding: state %d, goal %v, uploaded %d, want seeding with 1400 bytes up", got.State, got.SeedGoalMet, got.Uploaded)
		}
		time.Sleep(time.Hour)
		got := get(t, e, r)
		if got.State != torrentclient.Stopped || !got.SeedGoalMet || got.Uploaded != 1500 || got.Seeding != 15*time.Second {
			t.Fatalf("past the ratio: state %d, goal %v, uploaded %d, seeding %v, want stopped at 1500 bytes after 15s", got.State, got.SeedGoalMet, got.Uploaded, got.Seeding)
		}
	})
}

func TestTheEngineQueuesPastItsSlots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e, c := engine(t, torrentclient.Config{Slots: 1}, "orchard", "lantern")
		first, second := c.Releases[0], c.Releases[1]
		add(t, e, first, "")
		add(t, e, second, "")
		if got := get(t, e, second); got.State != torrentclient.Queued || got.Order != 1 {
			t.Fatalf("second: state %d, order %d, want queued second", got.State, got.Order)
		}
		time.Sleep(15 * time.Second)
		if got := get(t, e, second); got.State != torrentclient.Downloading || got.Done != 500 {
			t.Fatalf("second after 15s: state %d, done %d, want downloading from 10s", got.State, got.Done)
		}
	})
}

func TestTheEngineRemovesOnlyItsTorrentsFiles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		e, c := engine(t, torrentclient.Config{Dir: dir, Name: "/downloads"}, "orchard", "lantern")
		kept, removed := c.Releases[0], c.Releases[1]
		add(t, e, kept, "/downloads/Movies")
		add(t, e, removed, "/downloads/Movies")
		time.Sleep(time.Minute)
		get(t, e, kept)
		if err := e.Remove(removed.InfoHash, true); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "Movies", "lantern.mkv")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the removed torrent's file is still there: %v", err)
		}
		if !bytes.Equal(onDisk(t, filepath.Join(dir, "Movies", "orchard.mkv")), payload(t, kept)) {
			t.Error("the other torrent's file changed")
		}
		if err := e.Remove(kept.InfoHash, false); err != nil {
			t.Fatal(err)
		}
		if len(e.List(nil)) != 0 {
			t.Error("the engine still lists a removed torrent")
		}
		onDisk(t, filepath.Join(dir, "Movies", "orchard.mkv"))
		if err := e.Remove(kept.InfoHash, false); !errors.Is(err, torrentclient.ErrNotFound) {
			t.Errorf("removing it again = %v, want ErrNotFound", err)
		}
	})
}

func TestTheEngineMovesATorrentsFiles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		e, c := engine(t, torrentclient.Config{Dir: dir, Name: "/downloads"}, "orchard")
		r := c.Releases[0]
		add(t, e, r, "/downloads/incoming")
		time.Sleep(time.Minute)
		get(t, e, r)
		if err := e.Move(r.InfoHash, "/downloads/done"); err != nil {
			t.Fatal(err)
		}
		if got := get(t, e, r); got.Folder != "/downloads/done" {
			t.Errorf("moved, its folder is %s", got.Folder)
		}
		if !bytes.Equal(onDisk(t, filepath.Join(dir, "done", "orchard.mkv")), payload(t, r)) {
			t.Error("the moved file isn't the payload")
		}
		if _, err := os.Stat(filepath.Join(dir, "incoming", "orchard.mkv")); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the file is still in the old folder: %v", err)
		}
		if err := e.Move(r.InfoHash, "/elsewhere"); err == nil {
			t.Error("moving outside the client's folder succeeded")
		}
	})
}

func TestTheEngineNamesAWindowsClientsFoldersAsTheClientDoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		e, c := engine(t, torrentclient.Config{Dir: dir, Name: `D:\Torrents`, Windows: true}, "orchard", "lantern")
		add(t, e, c.Releases[0], `d:\torrents\Movies`)
		add(t, e, c.Releases[1], `D:\Elsewhere`)
		time.Sleep(time.Minute)
		get(t, e, c.Releases[0])
		onDisk(t, filepath.Join(dir, "Movies", "orchard.mkv"))
		if got := get(t, e, c.Releases[1]); got.State != torrentclient.Stopped || got.LocalError != `no such file or directory (D:\Elsewhere)` {
			t.Errorf("a folder outside the client's: state %d, error %q, want stopped with a local error", got.State, got.LocalError)
		}
	})
}

func TestTheEnginePlaysItsFaults(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e, c := engine(t, torrentclient.Config{}, "orchard", "lantern", "meadow")
		stalled, late, dropped := c.Releases[0], c.Releases[1], c.Releases[2]
		add(t, e, stalled, "")
		if err := e.Stall(stalled.InfoHash, true); err != nil {
			t.Fatal(err)
		}
		e.ListNextAddAfter(30 * time.Second)
		add(t, e, late, "")
		e.DropNextAdd()
		add(t, e, dropped, "")
		if _, ok := e.Get(late.InfoHash); ok {
			t.Error("a late add is listed at once")
		}
		if _, duplicate, _ := e.Add(late.Metainfo, "", nil, false); !duplicate {
			t.Error("adding a late torrent again isn't a duplicate")
		}
		time.Sleep(30 * time.Second)
		if got := get(t, e, stalled); got.Done != 0 || got.State != torrentclient.Downloading || got.Rates.Down != 0 {
			t.Errorf("stalled: done %d, state %d, down %d, want downloading with nothing arrived", got.Done, got.State, got.Rates.Down)
		}
		get(t, e, late)
		if _, ok := e.Get(dropped.InfoHash); ok {
			t.Error("a dropped add is listed")
		}
		if err := e.Fail(stalled.InfoHash, "No space left on device"); err != nil {
			t.Fatal(err)
		}
		if got := get(t, e, stalled); got.State != torrentclient.Stopped || got.LocalError != "No space left on device" {
			t.Errorf("failed: state %d, error %q", got.State, got.LocalError)
		}
		if err := e.Fail(stalled.InfoHash, ""); err != nil {
			t.Fatal(err)
		}
		if got := get(t, e, stalled); got.State != torrentclient.Downloading || got.LocalError != "" {
			t.Errorf("cleared: state %d, error %q, want downloading again", got.State, got.LocalError)
		}
		e.Empty()
		if got := e.List(nil); len(got) != 0 {
			t.Errorf("an emptied client lists %d torrents", len(got))
		}
	})
}

func TestTheEngineChecksATorrentsDataBeforeItMoves(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		e, c := engine(t, torrentclient.Config{Check: 3 * time.Second}, "orchard", "lantern")
		started, stopped := c.Releases[0], c.Releases[1]
		add(t, e, started, "")
		if _, _, err := e.Add(stopped.Metainfo, "", nil, true); err != nil {
			t.Fatal(err)
		}
		time.Sleep(2 * time.Second)
		for _, r := range c.Releases {
			if got := get(t, e, r); got.State != torrentclient.Checking || got.Done != 0 {
				t.Fatalf("%s while checking: state %d, done %d", r.Name, got.State, got.Done)
			}
		}
		time.Sleep(5 * time.Second)
		if got := get(t, e, started); got.State != torrentclient.Downloading || got.Done != 400 {
			t.Errorf("started after its check: state %d, done %d, want 400 bytes from 3s", got.State, got.Done)
		}
		if got := get(t, e, stopped); got.State != torrentclient.Stopped {
			t.Errorf("stopped after its check: state %d", got.State)
		}
	})
}
