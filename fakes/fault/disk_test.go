package fault_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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
		{fault.FoldCase{}, "foldcase"},
		{fault.NoIDs{}, "noids"},
		{fault.NoLinks{}, "nolinks"},
		{fault.NoClones{}, "noclones"},
		{fault.CloneFull{}, "clonefull"},
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
	t.Run("foldcase", playsFoldCase)
	t.Run("noids", playsNoIDs)
	t.Run("nolinks, noclones and clonefull", playsTheLadderFaults)
	t.Run("hung", func(t *testing.T) { synctest.Test(t, playsHung) })
	t.Run("on an open writer", playsOnAnOpenWriter)
	t.Run("on an open reader", playsOnAnOpenReader)
	t.Run("hung on an open reader", func(t *testing.T) { synctest.Test(t, playsHungOnAnOpenReader) })
}

func playsOnAnOpenWriter(t *testing.T) {
	c := newCued(t)
	part := filepath.Join(c.folder, "0.part")
	w, err := c.d.OpenWrite(c.ctx, part)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = w.Close() }()
	if _, err := w.WriteAt([]byte("film"), 0); err != nil {
		t.Fatalf("WriteAt with room = %v", err)
	}
	lift := c.cue(fault.Full{})
	if _, err := w.WriteAt([]byte("film"), 4); !storing.NoSpace(err) {
		t.Fatalf("WriteAt once the volume fills = %v, want no space", err)
	}
	if again, err := c.d.OpenWrite(c.ctx, part); err != nil {
		t.Fatalf("OpenWrite on a full volume = %v, want it to open", err)
	} else {
		_ = again.Close()
	}
	lift()
	readOnly := c.cue(fault.ReadOnly{})
	if _, err := w.WriteAt([]byte("film"), 4); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("WriteAt once the folder turns read-only = %v, want EROFS", err)
	}
	if err := w.Truncate(0); !errors.Is(err, syscall.EROFS) {
		t.Fatalf("Truncate once the folder turns read-only = %v, want EROFS", err)
	}
	readOnly()
	if _, err := w.WriteAt([]byte("film"), 4); err != nil {
		t.Fatalf("WriteAt once the folder takes writes again = %v", err)
	}
	if err := w.Sync(); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(part); err != nil || string(b) != "filmfilm" {
		t.Fatalf("the file holds %q, %v; want the two writes that had room", b, err)
	}
}

func playsOnAnOpenReader(t *testing.T) {
	c := newCued(t)
	r, err := c.d.OpenAt(c.ctx, c.marker)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	b := make([]byte, 1)
	lift := c.cue(fault.Unmounted{})
	if _, err := r.ReadAt(b, 0); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadAt once the folder unmounts = %v, want not exist", err)
	}
	if _, err := c.d.OpenAt(c.ctx, c.marker); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("OpenAt on an unmounted folder = %v, want not exist", err)
	}
	lift()
	if _, err := r.ReadAt(b, 0); err != nil || string(b) != "x" {
		t.Fatalf("ReadAt once mounted again = %q, %v", b, err)
	}
}

func playsHungOnAnOpenReader(t *testing.T) {
	c := newCued(t)
	r, err := c.d.OpenAt(c.ctx, c.marker)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	lift := c.cue(fault.Hung{})
	answered := make(chan error, 1)
	go func() {
		_, err := r.ReadAt(make([]byte, 1), 0)
		answered <- err
	}()
	synctest.Wait()
	select {
	case err := <-answered:
		t.Fatalf("ReadAt on a hung folder answered %v", err)
	default:
	}
	lift()
	if err := <-answered; err != nil {
		t.Fatalf("ReadAt once the folder answers = %v", err)
	}
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
	if _, err := f.Write([]byte("x")); !storing.NoSpace(err) {
		t.Fatalf("Write on a full volume = %v, want no space", err)
	}
	_ = f.Close()
	if err := c.d.Mkdir(c.ctx, filepath.Join(c.folder, "season")); !storing.NoSpace(err) {
		t.Fatalf("Mkdir on a full volume = %v, want no space", err)
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

func playsFoldCase(t *testing.T) {
	c := newCued(t)
	upper := filepath.Join(c.folder, ".SLIPSTREAM-ROOT")
	// A host that folds case itself, as macOS and Windows do by default,
	// finds names in any case with or without the fault.
	_, err := os.Stat(upper)
	sensitive := errors.Is(err, fs.ErrNotExist)
	if _, err := c.d.Stat(c.ctx, upper); sensitive && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat of the marker in upper case before the fault = %v, want not exist", err)
	}
	c.cue(fault.FoldCase{})
	marker, err := c.d.Stat(c.ctx, c.marker)
	if err != nil {
		t.Fatal(err)
	}
	if found, err := c.d.Stat(c.ctx, upper); err != nil || !found.Same(marker) {
		t.Fatalf("Stat of the marker in upper case = %v, %v; want the marker", found, err)
	}
	if err := os.WriteFile(filepath.Join(c.sibling, "x"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := c.d.Stat(c.ctx, filepath.Join(c.sibling, "X")); sensitive && !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat in upper case in a sibling = %v, want not exist", err)
	}
	if err := os.WriteFile(filepath.Join(c.folder, "heat.mkv"), []byte("film"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := c.d.Rename(c.ctx, filepath.Join(c.folder, "heat.mkv"), filepath.Join(c.folder, "Heat.mkv")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(c.folder, "heat (1995)"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := c.d.Mkdir(c.ctx, filepath.Join(c.folder, "Heat (1995)")); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("Mkdir of a folder there in another case = %v, want exists", err)
	}
	if got := names(t, c.folder); !slices.Equal(got, []string{storing.MarkerName, "Heat.mkv", "heat (1995)"}) {
		t.Fatalf("the folder holds %q, want the file respelled and one folder kept as made", got)
	}
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	des, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, de := range des {
		out = append(out, de.Name())
	}
	slices.Sort(out)
	return out
}

func playsNoIDs(t *testing.T) {
	c := newCued(t)
	own, err := c.d.Stat(c.ctx, c.marker)
	if err != nil || own.FileID() == nil || own.Volume() == "" {
		t.Fatalf("Stat before the fault = %v, %v; want an id on a volume", own, err)
	}
	c.cue(fault.NoIDs{})
	idless, err := c.d.Stat(c.ctx, c.marker)
	if err != nil || idless.FileID() != nil || idless.Volume() != "" || !idless.Same(own) {
		t.Fatalf("Stat under the fault = %v, %v; want the same file with no id and no volume", idless, err)
	}
	if info, err := c.d.Stat(c.ctx, c.sibling); err != nil || info.FileID() == nil {
		t.Fatalf("Stat of a sibling = %v, %v; want its id", info, err)
	}
}

func playsTheLadderFaults(t *testing.T) {
	c := newCued(t)
	from := filepath.Join(c.sibling, "download.mkv")
	if err := os.WriteFile(from, []byte("film"), 0o600); err != nil {
		t.Fatal(err)
	}
	into := func(name string) string { return filepath.Join(c.folder, name) }
	c.cue(fault.NoLinks{})
	if err := c.d.Link(c.ctx, from, into("linked.mkv")); !errors.Is(err, storing.ErrCrossDevice) {
		t.Fatalf("Link into a nolinks folder = %v, want the cross-device errno", err)
	}
	if err := c.d.Link(c.ctx, from, filepath.Join(c.sibling, "linked.mkv")); err != nil {
		t.Fatalf("Link beside it = %v, want a link", err)
	}
	lift := c.cue(fault.NoClones{})
	if err := c.d.Clone(c.ctx, from, into("cloned.mkv")); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Clone into a noclones folder = %v, want ErrUnsupported", err)
	}
	lift()
	c.cue(fault.CloneFull{})
	if err := c.d.Clone(c.ctx, from, into("cloned.mkv")); !storing.NoSpace(err) {
		t.Fatalf("Clone into a clonefull folder = %v, want the disk-full errno", err)
	}
	for _, name := range []string{"linked.mkv", "cloned.mkv", "cloned.mkv.part"} {
		if _, err := os.Stat(into(name)); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("%s is in the folder after its fault: %v", name, err)
		}
	}
}
