package fault_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/storing"
)

func TestDiskFaultGrammarRoundTrips(t *testing.T) {
	for _, c := range []struct {
		f    fault.Fault
		text string
	}{
		{fault.Full{}, "full"},
		{fault.Unmounted{}, "unmounted"},
		{fault.Hung{}, "hung"},
		{fault.ReadOnly{}, "readonly"},
		{fault.Volume{Total: 10 << 30, Free: 1536 << 20}, "volume=10GiB,free=1536MiB"},
		{fault.Volume{Total: 1 << 40, Free: 0}, "volume=1024GiB,free=0B"},
	} {
		if got := c.f.String(); got != c.text {
			t.Errorf("%#v.String() = %q, want %q", c.f, got, c.text)
		}
		if got, err := fault.Parse(c.text); err != nil || got != c.f || got.Layer() != fault.Disk {
			t.Errorf("Parse(%q) = %#v, %v; want %#v at the disk layer", c.text, got, err, c.f)
		}
	}
	for text, want := range map[string]string{
		"full=1":                   "fault full takes no value",
		"volume":                   "fault volume needs a value: volume=…",
		"volume=10GiB":             "fault volume=10GiB: want volume=<total>,free=<free>",
		"volume=10GB,free=1GiB":    `fault volume=10GB,free=1GiB: size "10GB" is not a count of B`,
		"volume=1GiB,free=2GiB":    "fault volume=1GiB,free=2GiB: free 2GiB of 1GiB: want a positive total and free no more than it",
		"volume=0B,free=0B":        "fault volume=0B,free=0B: free 0B of 0B: want a positive total and free no more than it",
		"volume=1GiB,free=512KiB,": `fault volume=1GiB,free=512KiB,: size "512KiB," needs a unit: B, KiB, MiB or GiB`,
	} {
		if _, err := fault.Parse(text); err == nil || err.Error() != want {
			t.Errorf("Parse(%q) = %v, want %q", text, err, want)
		}
	}
}

func TestADiskRulePlaysOnlyOnADiskPlan(t *testing.T) {
	for _, c := range []struct {
		plan *fault.Plan
		rule fault.Rule
		want string
	}{
		{fault.NewPlan(fronts, nil), fault.Rule{ID: "x", Target: "/media", Fault: fault.Hung{}}, "hung is a disk fault, which plays only in the simulation tier"},
		{fault.NewDiskPlan(), fault.Rule{ID: "x", Fault: fault.Full{}}, `disk rule "x" names no folder`},
		{fault.NewDiskPlan(), fault.Rule{ID: "x", Target: "/media", Fault: fault.ReadOnly{}, Times: 1}, "readonly holds a folder until cleared and takes no origin, skip, times or after"},
		{fault.NewDiskPlan(), fault.Rule{ID: "x", Target: "/media", Origin: "home", Fault: fault.Unmounted{}}, "unmounted holds a folder until cleared and takes no origin, skip, times or after"},
	} {
		if _, err := c.plan.Set(c.rule); err == nil || err.Error() != c.want {
			t.Errorf("Set(%+v) = %v, want %q", c.rule, err, c.want)
		}
	}
}

// cued is a disk Plan over a real temp folder holding a movies folder
// with its marker, and a sibling whose name movies prefixes.
type cued struct {
	t                       *testing.T
	ctx                     context.Context
	base                    storing.Disk
	plan                    *fault.Plan
	d                       storing.Disk
	folder, sibling, marker string
}

func newCued(t *testing.T) *cued {
	t.Helper()
	base, err := storing.NewOSDisk(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	c := &cued{t: t, ctx: t.Context(), base: base, plan: fault.NewDiskPlan(), folder: filepath.Join(dir, "movies"), sibling: filepath.Join(dir, "movies2")}
	c.d = c.plan.Disk(base)
	c.marker = filepath.Join(c.folder, storing.MarkerName)
	for _, f := range []string{c.folder, c.sibling} {
		if err := os.Mkdir(f, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(c.marker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	return c
}

func (c *cued) cue(f fault.Fault) (lift func()) {
	c.t.Helper()
	id := f.String()
	arm(c.t, c.plan, fault.Rule{ID: id, Target: fault.Target(c.folder), Fault: f})
	return func() { c.plan.Clear(id) }
}

func TestDiskFaultsPlayOnCue(t *testing.T) {
	t.Run("unmounted", playsUnmounted)
	t.Run("readonly", playsReadOnly)
	t.Run("volume and full", playsVolumeAndFull)
	t.Run("hung", func(t *testing.T) { synctest.Test(t, playsHung) })
}

func playsUnmounted(t *testing.T) {
	c := newCued(t)
	lift := c.cue(fault.Unmounted{})
	if info, err := c.d.Stat(c.ctx, c.folder); err != nil || !info.Dir() {
		t.Fatalf("Stat of an unmounted folder = %v, %v; want the mount point", info, err)
	}
	if entries, err := c.d.ReadDir(c.ctx, c.folder); err != nil || len(entries) != 0 {
		t.Fatalf("an unmounted folder lists %v, %v; want nothing", entries, err)
	}
	if _, err := c.d.Open(c.ctx, c.marker); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open of the marker on an unmounted folder = %v, want not exist", err)
	}
	if _, err := c.d.Create(c.ctx, filepath.Join(c.folder, "a.mkv")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Create under an unmounted folder = %v, want not exist", err)
	}
	if _, err := c.d.Stat(c.ctx, c.sibling); err != nil {
		t.Fatalf("Stat of a sibling sharing the name's prefix = %v", err)
	}
	lift()
	f, err := c.d.Open(c.ctx, c.marker)
	if err != nil {
		t.Fatalf("Open of the marker once mounted again = %v", err)
	}
	_ = f.Close()
}

func playsReadOnly(t *testing.T) {
	c := newCued(t)
	c.cue(fault.ReadOnly{})
	if ok, err := c.d.Writable(c.ctx, c.folder); ok || err != nil {
		t.Fatalf("Writable of a read-only folder = %v, %v; want false", ok, err)
	}
	if err := c.d.Mkdir(c.ctx, filepath.Join(c.folder, "season")); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("Mkdir in a read-only folder = %v, want EROFS", err)
	}
	if err := c.d.Rename(c.ctx, filepath.Join(c.sibling, "a"), filepath.Join(c.folder, "a")); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("Rename into a read-only folder = %v, want EROFS", err)
	}
	if ok, err := c.d.Writable(c.ctx, c.sibling); !ok || err != nil {
		t.Fatalf("Writable of a sibling = %v, %v; want true", ok, err)
	}
}

func playsVolumeAndFull(t *testing.T) {
	c := newCued(t)
	own, err := c.base.Volume(c.ctx, c.sibling)
	if err != nil {
		t.Fatal(err)
	}
	c.cue(fault.Volume{Total: 10 << 30, Free: 2 << 30})
	v, err := c.d.Volume(c.ctx, c.folder)
	if err != nil || v.ID != "sim:"+c.folder || v.Mount != c.folder || v.Total != 10<<30 || v.Free != 2<<30 {
		t.Fatalf("Volume of the folder = %+v, %v; want a volume of its own, 2 GiB free of 10", v, err)
	}
	if v, err := c.d.Volume(c.ctx, c.sibling); err != nil || v.ID != own.ID {
		t.Fatalf("Volume of a sibling = %+v, %v; want the real %s", v, err, own.ID)
	}
	lift := c.cue(fault.Full{})
	if v, err := c.d.Volume(c.ctx, c.folder); err != nil || v.Free != 0 || v.Total != 10<<30 {
		t.Fatalf("Volume of a full folder = %+v, %v; want none of 10 GiB free", v, err)
	}
	f, err := c.d.Create(c.ctx, filepath.Join(c.folder, "a.mkv"))
	if err != nil {
		t.Fatalf("Create on a full volume = %v, want it to open", err)
	}
	if _, err := f.Write([]byte("x")); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Write on a full volume = %v, want ENOSPC", err)
	}
	_ = f.Close()
	if err := c.d.Mkdir(c.ctx, filepath.Join(c.folder, "season")); !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("Mkdir on a full volume = %v, want ENOSPC", err)
	}
	lift()
	if err := c.d.Mkdir(c.ctx, filepath.Join(c.folder, "season")); err != nil {
		t.Fatalf("Mkdir once the volume has room = %v", err)
	}
}

func playsHung(t *testing.T) {
	c := newCued(t)
	lift := c.cue(fault.Hung{})
	answered := make(chan error, 1)
	go func() {
		_, err := c.d.Stat(c.ctx, c.folder)
		answered <- err
	}()
	synctest.Wait()
	select {
	case err := <-answered:
		t.Fatalf("Stat of a hung folder answered %v", err)
	default:
	}
	if _, err := c.d.Stat(c.ctx, c.sibling); err != nil {
		t.Fatalf("Stat of a sibling of a hung folder = %v", err)
	}
	gone, cancel := context.WithCancel(c.ctx)
	cancel()
	if _, err := c.d.ReadDir(gone, c.folder); !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadDir of a hung folder under an ended context = %v, want canceled", err)
	}
	lift()
	if err := <-answered; err != nil {
		t.Fatalf("Stat of a folder no longer hung = %v", err)
	}
}
