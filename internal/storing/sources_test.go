package storing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/pb/slipstream/media/v1/mediav1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// fenced is a Node with one root and one download folder, and a folder
// elsewhere that a link in the root points at.
type fenced struct {
	store     *Store
	at        At
	root      string
	downloads string
	elsewhere string
	seeding   []string
	probes    Probes
}

func newFenced(t *testing.T, prober media.Prober) *fenced {
	t.Helper()
	d := newTestDisk(t)
	id := storage.NewUUID()
	f := &fenced{at: At{Node: node, Root: id}, root: marked(t, id), downloads: t.TempDir(), elsewhere: t.TempDir()}
	d.downloads = []string{f.downloads}
	f.store = NewStore(Config{
		Disk: d, Payloads: payloads{}, Downloads: func() []string { return d.downloads }, Torrents: func() []string { return f.seeding },
		Timings: Timings{Stat: 10 * time.Second},
	})
	running(t, f.store, intentOf(rootAt(id, f.root, false))).next(t)
	svc := media.NewService(prober, payloads{}, f.store.Holds, media.Timings{Probe: time.Second})
	f.probes = func(role.NodeID) mediav1connect.MediaServiceHandler { return svc }
	return f
}

// escape is a link in the root to the folder elsewhere, or a skip where
// the OS won't make one for this account.
func (f *fenced) escape(t *testing.T) string {
	t.Helper()
	link := filepath.Join(f.root, "Linked")
	if err := os.Symlink(f.elsewhere, link); err != nil {
		t.Skipf("this OS won't make a link here: %v", err)
	}
	return link
}

// requests are every Node request on the folder and file, by name.
func (f *fenced) requests(t *testing.T, folder, file string) map[string]func() error {
	folders := foldersOf(f.store)
	return map[string]func() error{
		"Place": func() error {
			_, err := remoteOf(f.store).Place(t.Context(), f.at, NodePath{Abs: file}, mustRel(t, HoldingDir+"/0190.mkv"), true)
			return err
		},
		"Probe": func() error {
			_, err := f.probes.Probe(t.Context(), node, NodePath{Abs: file})
			return err
		},
		"ListVideos": func() error {
			_, err := folders.ListVideos(t.Context(), node, folder, media.DefaultVideos)
			return err
		},
		"RemoveSource": func() error { return folders.RemoveSource(t.Context(), node, file) },
		"Browse": func() error {
			_, err := folders.Browse(t.Context(), node, folder, false, PurposeImport)
			return err
		},
	}
}

func TestEveryRequestRefusesAPathOutsideTheFence(t *testing.T) {
	for _, place := range []string{"elsewhere", "a link in the root to elsewhere"} {
		t.Run(place, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { refusesOutside(t, place) })
		})
	}
}

func refusesOutside(t *testing.T, place string) {
	f := newFenced(t, mediatest.Prober{Of: map[string]media.Record{"Heat.mkv": {Container: "matroska"}}})
	write(t, filepath.Join(f.elsewhere, "Heat (1995)", "Heat.mkv"), "film")
	folder := f.elsewhere
	if place != "elsewhere" {
		folder = f.escape(t)
	}
	file := filepath.Join(folder, "Heat (1995)", "Heat.mkv")
	folders := foldersOf(f.store)
	for name, call := range f.requests(t, folder, file) {
		if _, ok := errors.AsType[*OutsideError](call()); !ok {
			t.Errorf("%s of a file %s = %v, want *OutsideError", name, place, call())
		}
	}
	c, err := folders.Check(t.Context(), node, folder, PurposeImport, nil)
	if err != nil || c.Refusal == nil || c.Refusal.Reason != ReasonOutside {
		t.Errorf("Check for import of a folder %s = %+v, %v; want refused outside", place, c, err)
	}
	if _, err := os.Stat(filepath.Join(f.elsewhere, "Heat (1995)", "Heat.mkv")); err != nil {
		t.Errorf("the file outside is gone: %v", err)
	}
}

func TestAFolderInADownloadFolderListsItsVideosAndWhichSeed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFenced(t, nil)
		folder := filepath.Join(f.downloads, "Show.S01")
		e1 := write(t, filepath.Join(folder, "Show.S01E01.mkv"), "one")
		e2 := write(t, filepath.Join(folder, "Season 1", "Show.S01E02.mp4"), "two!")
		write(t, filepath.Join(folder, "Show.S01E01.nfo"), "info")
		write(t, filepath.Join(folder, "Sample", "Show.S01E01.mkv"), "x")
		f.seeding = []string{e1}
		// A Windows path may come with either separator.
		l, err := foldersOf(f.store).ListVideos(t.Context(), node, filepath.ToSlash(folder), media.DefaultVideos)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]string, len(l.Files))
		for i, file := range l.Files {
			got[i] = fmt.Sprintf("%s %d seeded=%v", file.Path, file.Size, file.Seeded)
			if file.Seen.ID.IsZero() {
				t.Errorf("%s has no file identity", file.Path)
			}
		}
		want := []string{e2 + " 4 seeded=false", e1 + " 3 seeded=true"}
		if l.Path != folder || !l.Root.IsZero() || !slices.Equal(got, want) {
			t.Errorf("ListVideos = %s in root %v: %v; want %s in no root: %v", l.Path, l.Root, got, folder, want)
		}
	})
}

func TestAFolderInARootListsAsThatRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFenced(t, nil)
		write(t, filepath.Join(f.root, "Heat (1995)", "Heat.mkv"), "film")
		l, err := foldersOf(f.store).ListVideos(t.Context(), node, filepath.Join(f.root, "Heat (1995)"), media.DefaultVideos)
		if err != nil || l.Root != f.at.Root || l.Rel.String() != "Heat (1995)" || len(l.Files) != 1 {
			t.Errorf("ListVideos in the root = %+v, %v; want root %v, Heat (1995) and one file", l, err, f.at.Root)
		}
	})
}

func TestAFileInADownloadFolderPlacesAndItsSourceGoes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFenced(t, nil)
		src := write(t, filepath.Join(f.downloads, "Heat.1995", "Heat.mkv"), "film")
		placed, err := remoteOf(f.store).Place(t.Context(), f.at, NodePath{Abs: src}, mustRel(t, HoldingDir+"/0190.mkv"), true)
		if err != nil || placed.Size != 4 {
			t.Fatalf("Place = %+v, %v; want 4 bytes placed", placed, err)
		}
		for range 2 {
			if err := foldersOf(f.store).RemoveSource(t.Context(), node, src); err != nil {
				t.Fatalf("RemoveSource = %v, want removed", err)
			}
		}
		if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("the source is still there: %v", err)
		}
		holds(t, filepath.Join(f.root, filepath.FromSlash(HoldingDir), "0190.mkv"), "film")
	})
}

func TestBrowsingForAnImportStartsAtTheRootsAndDownloadFolders(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFenced(t, nil)
		l, err := foldersOf(f.store).Browse(t.Context(), node, "", false, PurposeImport)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, p := range l.Places {
			got = append(got, p.Path)
		}
		if want := []string{f.root, f.downloads}; !slices.Equal(got, want) {
			t.Errorf("the places to import from are %v, want %v", got, want)
		}
		inside, err := foldersOf(f.store).Browse(t.Context(), node, f.downloads, false, PurposeImport)
		if err != nil || inside.Parent != "" {
			t.Errorf("Browse of the download folder = parent %q, %v; want no parent outside it", inside.Parent, err)
		}
	})
}

type stalled struct{}

func (stalled) Probe(ctx context.Context, _ string) (media.Record, error) {
	<-ctx.Done()
	return media.Record{}, fmt.Errorf("run ffprobe: %w", ctx.Err())
}

func TestAProbeTimeoutAndAnFfprobeErrorArriveTyped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newFenced(t, stalled{})
		file := write(t, filepath.Join(f.downloads, "Heat.mkv"), "film")
		if _, err := f.probes.Probe(t.Context(), node, NodePath{Abs: file}); !errors.Is(err, media.ErrProbeTimeout) || rpc.CodeOf(err) != rpc.CodeDeadlineExceeded {
			t.Errorf("a probe that runs out of time = %v, want media.ErrProbeTimeout", err)
		}
		f = newFenced(t, mediatest.Prober{Fail: map[string]string{"Heat.mkv": "Invalid data found when processing input"}})
		file = write(t, filepath.Join(f.downloads, "Heat.mkv"), "film")
		if _, err := f.probes.Probe(t.Context(), node, NodePath{Abs: file}); err == nil || err.Error() != "ffprobe can't read the file: Invalid data found when processing input" {
			t.Errorf("a probe ffprobe can't read = %v, want *media.UnreadableError", err)
		} else if _, ok := errors.AsType[*media.UnreadableError](err); !ok {
			t.Errorf("a probe ffprobe can't read = %T, want *media.UnreadableError", err)
		}
	})
}

func TestAProbeOfAPayloadReadsItWhereItsClientKeepsIt(t *testing.T) {
	h := bencode.InfoHash{1}
	want := media.Record{Container: "matroska"}
	svc := media.NewService(mediatest.Prober{Of: map[string]media.Record{"0100/orchard.mkv": want}}, payloads{h: "/downloads/0100/orchard.mkv"}, nil, media.DefaultTimings())
	probes := Probes(func(role.NodeID) mediav1connect.MediaServiceHandler { return svc })
	if got, err := probes.Probe(t.Context(), node, Payload{InfoHash: h}); err != nil || got.Container != want.Container {
		t.Errorf("Probe of a payload = %+v, %v; want %+v", got, err, want)
	}
	if _, err := probes.Probe(t.Context(), node, Payload{InfoHash: bencode.InfoHash{2}}); rpc.CodeOf(err) != rpc.CodeUnavailable {
		t.Errorf("Probe of a torrent not here = %v, want unavailable", err)
	}
}
