package storing

import (
	"context"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

type holder struct {
	snaps chan role.Snapshot[*storagev1.Intent]
	facts chan *storagev1.Facts
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*storagev1.Intent]] {
	return func(yield func(role.Snapshot[*storagev1.Intent]) bool) {
		for {
			select {
			case s := <-h.snaps:
				if !yield(s) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}
}

func (h *holder) SetFacts(f *storagev1.Facts) { h.facts <- f }

func (h *holder) Report(context.Context, *storage.Tx[storage.Local], *storagev1.Result) error {
	return errors.New("storage reports by facts")
}

func running(t *testing.T, s *Store, intent *storagev1.Intent) *holder {
	t.Helper()
	h := &holder{snaps: make(chan role.Snapshot[*storagev1.Intent], 1), facts: make(chan *storagev1.Facts, 16)}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx, h) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("Run = %v", err)
		}
	})
	h.send(role.Snapshot[*storagev1.Intent]{Intent: intent, Standing: role.Live})
	return h
}

func (h *holder) send(s role.Snapshot[*storagev1.Intent]) { h.snaps <- s }

// next is the next facts set, failing when none comes within a minute.
func (h *holder) next(t *testing.T) *storagev1.Facts {
	t.Helper()
	select {
	case f := <-h.facts:
		return f
	case <-time.After(time.Minute):
		t.Fatal("no facts came")
		return nil
	}
}

func intentOf(roots ...*storagev1.Root) *storagev1.Intent {
	return &storagev1.Intent{Roots: roots, SampleEverySeconds: 3600}
}

func rootAt(id storage.UUID, path string, adopt bool, proof ...string) *storagev1.Root {
	return &storagev1.Root{Id: id.Bytes(), Path: path, Adopt: adopt, Place: 1, Proof: proof}
}

func newStore(d *testDisk) *Store {
	return NewStore(Config{Disk: d, Payloads: payloads{}, Timings: Timings{Stat: 10 * time.Second}})
}

func dir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0o750); err != nil {
		t.Fatal(err)
	}
	return p
}

func write(t *testing.T, path, body string) string {
	t.Helper()
	dir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func markerOf(t *testing.T, folder string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(folder, MarkerName))
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func marked(t *testing.T, id storage.UUID) string {
	t.Helper()
	folder := t.TempDir()
	write(t, filepath.Join(folder, MarkerName), id.String())
	return folder
}

func states(t *testing.T, f *storagev1.Facts) map[storage.UUID]RootState {
	t.Helper()
	out := map[storage.UUID]RootState{}
	for _, r := range f.GetRoots() {
		id, err := storage.UUIDFromBytes(r.GetId())
		if err != nil {
			t.Fatal(err)
		}
		st, err := ParseRootState(r.GetState())
		if err != nil {
			t.Fatal(err)
		}
		out[id] = st
	}
	return out
}

func checkStates(t *testing.T, f *storagev1.Facts, want map[storage.UUID]RootState) {
	t.Helper()
	got := states(t, f)
	for id, st := range want {
		if got[id] != st {
			t.Errorf("root %s reads %v, want %v", id, got[id], st)
		}
	}
}

type payloads map[bencode.InfoHash]string

func (p payloads) Path(h bencode.InfoHash, index int) (string, error) {
	path, ok := p[h]
	if !ok || index != 0 {
		return "", errors.New("the torrent is not in this Node's client")
	}
	return path, nil
}

func TestAFolderWithoutItsMarkerReadsUnmounted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		empty, gone, ready, foreign, readOnly, file := storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
		readOnlyAt := marked(t, readOnly)
		d.flag(d.readOnly, readOnlyAt)
		h := running(t, newStore(d), intentOf(
			rootAt(empty, t.TempDir(), false),
			rootAt(gone, filepath.Join(t.TempDir(), "unplugged"), false),
			rootAt(ready, marked(t, ready), false),
			rootAt(foreign, marked(t, storage.NewUUID()), false),
			rootAt(readOnly, readOnlyAt, false),
			rootAt(file, write(t, filepath.Join(t.TempDir(), "a file"), "x"), false),
		))
		checkStates(t, h.next(t), map[storage.UUID]RootState{
			empty: Unmounted, gone: Missing, ready: Ready, foreign: Foreign, readOnly: Unwritable, file: Missing,
		})
	})
}

func TestAdoptWritesTheMarkerOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		id, folder := storage.NewUUID(), t.TempDir()
		intent := intentOf(rootAt(id, folder, true))
		h := running(t, newStore(d), intent)
		checkStates(t, h.next(t), map[storage.UUID]RootState{id: Ready})
		h.send(role.Snapshot[*storagev1.Intent]{Intent: intent, Standing: role.Live})
		checkStates(t, h.next(t), map[storage.UUID]RootState{id: Ready})
		if got := markerOf(t, folder); got != id.String() {
			t.Errorf("the marker names %q, want %s", got, id)
		}
		if n := d.count("create", filepath.Join(folder, MarkerName+".part")); n != 1 {
			t.Errorf("the marker was written %d times, want once", n)
		}
	})
}

func TestAdoptRefusesAFolderWithoutProof(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		id, folder := storage.NewUUID(), t.TempDir()
		proof := "Heat (1995)/Heat (1995).mkv"
		intent := intentOf(rootAt(id, folder, true, "Gone (2001)/Gone (2001).mkv", proof))
		h := running(t, newStore(d), intent)
		checkStates(t, h.next(t), map[storage.UUID]RootState{id: Unmounted})
		if got := markerOf(t, folder); got != "" {
			t.Fatalf("an empty mount point got a marker naming %q", got)
		}
		write(t, filepath.Join(folder, "Heat (1995)", "Heat (1995).mkv"), "film")
		h.send(role.Snapshot[*storagev1.Intent]{Intent: intent, Standing: role.Live})
		checkStates(t, h.next(t), map[storage.UUID]RootState{id: Ready})
		if got := markerOf(t, folder); got != id.String() {
			t.Errorf("the proven folder's marker names %q, want %s", got, id)
		}
	})
}

func TestAStrayMarkerIsOverwrittenOnAdoptButAKnownOneIsNot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		stray, other, known := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
		strayAt, knownAt := marked(t, storage.NewUUID()), marked(t, other)
		garbled := t.TempDir()
		write(t, filepath.Join(garbled, MarkerName), "not a root id")
		garbledID := storage.NewUUID()
		h := running(t, newStore(d), intentOf(
			rootAt(stray, strayAt, true),
			rootAt(other, marked(t, other), false),
			rootAt(known, knownAt, true),
			rootAt(garbledID, garbled, true),
		))
		checkStates(t, h.next(t), map[storage.UUID]RootState{stray: Ready, other: Ready, known: Foreign, garbledID: Ready})
		if got := markerOf(t, strayAt); got != stray.String() {
			t.Errorf("the stray marker names %q, want %s", got, stray)
		}
		if got := markerOf(t, knownAt); got != other.String() {
			t.Errorf("adopt overwrote the marker of a root the intent knows: it names %q", got)
		}
	})
}

func TestAHungStatLeavesTheLoopRunning(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		hung, fine := storage.NewUUID(), storage.NewUUID()
		hungAt := marked(t, hung)
		release := d.hang(hungAt)
		intent := intentOf(rootAt(hung, hungAt, false), rootAt(fine, marked(t, fine), false))
		h := running(t, newStore(d), intent)
		checkStates(t, h.next(t), map[storage.UUID]RootState{hung: Unresponsive, fine: Ready})

		start := time.Now()
		h.send(role.Snapshot[*storagev1.Intent]{Intent: intent, Standing: role.Live})
		checkStates(t, h.next(t), map[storage.UUID]RootState{hung: Unresponsive, fine: Ready})
		if waited := time.Since(start); waited != 0 {
			t.Errorf("the second pass waited %v on the stuck folder", waited)
		}
		if n := d.count("stat", hungAt); n != 1 {
			t.Errorf("the hung folder was stat %d times, want once: calls stacked behind it", n)
		}

		release()
		synctest.Wait()
		h.send(role.Snapshot[*storagev1.Intent]{Intent: intent, Standing: role.Live})
		checkStates(t, h.next(t), map[storage.UUID]RootState{hung: Ready, fine: Ready})
	})
}

func TestTwoRootsOneVolume(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		media, other := t.TempDir(), t.TempDir()
		d.volumes[media] = Volume{ID: "fs-1", Name: "/srv/media", Mount: "/srv/media", Kind: Local, Total: 100, Free: 40}
		d.volumes[other] = Volume{ID: "fs-2", Name: "/mnt/usb", Mount: "/mnt/usb", Kind: Removable, Total: 10, Free: 5}
		movies, tv, usb := storage.NewUUID(), storage.NewUUID(), storage.NewUUID()
		write(t, filepath.Join(media, "movies", MarkerName), movies.String())
		write(t, filepath.Join(media, "tv", MarkerName), tv.String())
		write(t, filepath.Join(other, MarkerName), usb.String())
		h := running(t, newStore(d), intentOf(
			rootAt(movies, filepath.Join(media, "movies"), false),
			rootAt(tv, filepath.Join(media, "tv"), false),
			rootAt(usb, other, false),
		))
		f := h.next(t)
		var vols []string
		for _, v := range f.GetVolumes() {
			vols = append(vols, v.GetId())
		}
		if len(vols) != 2 || vols[0] != "fs-1" || vols[1] != "fs-2" {
			t.Errorf("volumes = %v, want [fs-1 fs-2]", vols)
		}
		want := []string{"fs-1", "fs-1", "fs-2"}
		for i, r := range f.GetRoots() {
			if r.GetVolumeId() != want[i] {
				t.Errorf("root %d is on volume %q, want %q", i, r.GetVolumeId(), want[i])
			}
		}
	})
}

func TestAPlacementResamplesAtMostOnceInTenSeconds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := newTestDisk(t)
		id := storage.NewUUID()
		folder := marked(t, id)
		d.volumes[folder] = Volume{ID: "fs-1", Kind: Local, Total: 100, Free: 40}
		hash := bencode.InfoHash{1}
		s := NewStore(Config{Disk: d, Payloads: payloads{hash: write(t, filepath.Join(t.TempDir(), "heat.mkv"), "film")}, Timings: Timings{Stat: 10 * time.Second}})
		h := running(t, s, intentOf(rootAt(id, folder, false)))
		h.next(t)
		place := func(name string) {
			t.Helper()
			if _, _, err := s.place(t.Context(), id, Payload{InfoHash: hash}, mustRel(t, HoldingDir+"/"+name)); err != nil {
				t.Fatal(err)
			}
		}

		place("a.mkv")
		h.next(t)
		place("b.mkv")
		synctest.Wait()
		if len(h.facts) != 0 {
			t.Error("a second placement within ten seconds sampled the volume again")
		}
		time.Sleep(resampleEvery)
		place("c.mkv")
		h.next(t)
	})
}

func mustRel(t *testing.T, s string) Rel {
	t.Helper()
	r, err := ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
