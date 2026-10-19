package storing

import (
	"context"
	"errors"
	"io"
	"io/fs"

	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"
)

func newOSDisk(t *testing.T) OSDisk {
	t.Helper()
	d, err := NewOSDisk("/data", "/downloads")
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestOSDiskHost(t *testing.T) {
	h := newOSDisk(t).Host()
	if h.Style != nativeStyle || h.Account == "" || h.Service || h.Data != "/data" || h.Downloads != "/downloads" {
		t.Errorf("Host = %+v", h)
	}
}

func TestOSDiskVolumeOfATempDir(t *testing.T) {
	d, dir := newOSDisk(t), t.TempDir()
	v, err := d.Volume(t.Context(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if v.ID == "" || v.Name == "" || v.Mount == "" || v.Kind != Local || v.Total <= 0 || v.Free < 0 || v.Free > v.Total || v.SampledAt.IsZero() {
		t.Errorf("Volume(%s) = %+v", dir, v)
	}
	if kind, err := KindOf(dir); err != nil || kind != Local {
		t.Errorf("KindOf(%s) = %v, %v; want local", dir, kind, err)
	}
}

func TestOSDiskTwoFoldersShareOneVolume(t *testing.T) {
	d, base := newOSDisk(t), t.TempDir()
	a, b := filepath.Join(base, "Movies"), filepath.Join(base, "TV")
	for _, p := range []string{a, b} {
		if err := d.Mkdir(t.Context(), p); err != nil {
			t.Fatal(err)
		}
	}
	va, err := d.Volume(t.Context(), a)
	if err != nil {
		t.Fatal(err)
	}
	vb, err := d.Volume(t.Context(), b)
	if err != nil {
		t.Fatal(err)
	}
	if va.ID != vb.ID || va.Mount != vb.Mount {
		t.Errorf("two folders in one temp dir: %+v and %+v", va, vb)
	}
	ia, _ := d.Stat(t.Context(), a)
	ib, _ := d.Stat(t.Context(), b)
	again, _ := d.Stat(t.Context(), filepath.Join(base, ".", "Movies"))
	if !ia.Dir() || ia.Same(ib) || !ia.Same(again) {
		t.Errorf("Same: Movies vs TV %v, Movies vs Movies %v", ia.Same(ib), ia.Same(again))
	}
}

func TestOSDiskPlaces(t *testing.T) {
	places, err := newOSDisk(t).Places(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(places) == 0 {
		t.Fatal("no places")
	}
	for _, p := range places {
		if p.Name == "" || p.Path == "" || p.Total <= 0 {
			t.Errorf("place %+v", p)
		}
	}
	if nativeStyle == Unix && places[0].Path != "/" {
		t.Errorf("first place %+v, want /", places[0])
	}
}

func TestOSDiskFilesRoundTrip(t *testing.T) {
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	if ok, err := d.Writable(ctx, dir); err != nil || !ok {
		t.Fatalf("Writable(temp dir) = %v, %v", ok, err)
	}
	if _, err := d.Writable(ctx, filepath.Join(dir, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Writable(missing) = %v, want not exist", err)
	}
	part, name := filepath.Join(dir, "a.mkv.part"), filepath.Join(dir, "a.mkv")
	f, err := d.Create(ctx, part)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("film")); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(f.Sync(), f.Close(), d.Rename(ctx, part, name), d.Link(ctx, name, filepath.Join(dir, "b.mkv"))); err != nil {
		t.Fatal(err)
	}
	r, err := d.Open(ctx, filepath.Join(dir, "b.mkv"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err := errors.Join(err, r.Close()); err != nil || string(b) != "film" {
		t.Fatalf("read the link: %q, %v", b, err)
	}
	if err := d.Mkdir(ctx, filepath.Join(dir, "Season 1")); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove(ctx, name); err != nil {
		t.Fatal(err)
	}
	entries, err := d.ReadDir(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{Name: "Season 1", Dir: true, Readable: true}, {Name: "b.mkv", Readable: true}}
	if !slices.Equal(entries, want) {
		t.Errorf("ReadDir = %+v, want %+v", entries, want)
	}
	if _, err := d.Open(ctx, name); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Open(removed) = %v", err)
	}
}

type stuckDisk struct {
	OSDisk
	release chan struct{}
}

func (d stuckDisk) Stat(ctx context.Context, path string) (Info, error) {
	<-d.release
	return d.OSDisk.Stat(ctx, path)
}

func TestAStuckStatIsUnresponsive(t *testing.T) {
	d, dir := newOSDisk(t), t.TempDir()
	synctest.Test(t, func(t *testing.T) {
		stuck := stuckDisk{d, make(chan struct{})}
		s := NewStore(Config{Disk: stuck, Timings: Timings{Stat: 10 * time.Second}})
		start := time.Now()
		_, err := timed(t.Context(), s, dir, func(ctx context.Context) (Info, error) { return stuck.Stat(ctx, dir) })
		if !errors.Is(err, errUnresponsive) || time.Since(start) != 10*time.Second {
			t.Errorf("stuck stat: %v after %v, want unresponsive after 10s", err, time.Since(start))
		}
		close(stuck.release)
	})
	s := NewStore(Config{Disk: d, Timings: Timings{Stat: 10 * time.Second}})
	info, err := timed(t.Context(), s, dir, func(ctx context.Context) (Info, error) { return d.Stat(ctx, dir) })
	if err != nil || !info.Dir() {
		t.Errorf("a stat that answers: %v, %v", info, err)
	}
}

func TestTimedAnswersTheCallersEnd(t *testing.T) {
	d := newOSDisk(t)
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		s := NewStore(Config{Disk: d, Timings: Timings{Stat: 10 * time.Second}})
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		_, err := timed(ctx, s, "key", func(context.Context) (int, error) { <-release; return 0, nil })
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("caller's deadline first: %v", err)
		}
		close(release)
	})
}
