package storing

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestOSDiskHidesByAttribute(t *testing.T) {
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	for _, name := range []string{"ProgramData", ".git", "$RECYCLE.BIN", "Movies"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	p, err := windows.UTF16PtrFromString(filepath.Join(dir, "ProgramData"))
	if err != nil {
		t.Fatal(err)
	}
	if err := windows.SetFileAttributes(p, windows.FILE_ATTRIBUTE_HIDDEN); err != nil {
		t.Fatal(err)
	}
	entries, err := d.ReadDir(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{
		{Name: "$RECYCLE.BIN", Dir: true, Hidden: true, Readable: true},
		{Name: ".git", Dir: true, Readable: true},
		{Name: "Movies", Dir: true, Readable: true},
		{Name: "ProgramData", Dir: true, Hidden: true, Readable: true},
	}
	if !slices.Equal(entries, want) {
		t.Errorf("ReadDir = %+v, want %+v", entries, want)
	}
}

func TestOSDiskListsJunctionsAsFolders(t *testing.T) {
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	movies, films := filepath.Join(dir, "Movies"), filepath.Join(dir, "films")
	if err := os.Mkdir(movies, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.CommandContext(ctx, "cmd", "/c", "mklink", "/J", films, movies).CombinedOutput(); err != nil {
		t.Fatalf("mklink /J: %v: %s", err, out)
	}
	entries, err := d.ReadDir(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{Name: "Movies", Dir: true, Readable: true}, {Name: "films", Dir: true, Readable: true}}
	if !slices.Equal(entries, want) {
		t.Errorf("ReadDir = %+v, want %+v", entries, want)
	}
	viaJunction, err := d.Stat(ctx, films)
	if err != nil {
		t.Fatal(err)
	}
	direct, _ := d.Stat(ctx, movies)
	if !viaJunction.Dir() || !viaJunction.Same(direct) {
		t.Error("Stat through a junction is not the folder it names")
	}
}

func TestOSDiskPlacesHoldTheTempDrive(t *testing.T) {
	places, err := newOSDisk(t).Places(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	drive := strings.ToUpper(filepath.VolumeName(t.TempDir())) + `\`
	if !slices.ContainsFunc(places, func(p Place) bool { return p.Path == drive && strings.HasPrefix(p.Name, drive[:2]) }) {
		t.Errorf("places %+v hold no %s", places, drive)
	}
}

func TestLongPath(t *testing.T) {
	ctx, d := t.Context(), newOSDisk(t)
	deep := t.TempDir()
	for len(deep) < 300 {
		deep = filepath.Join(deep, strings.Repeat("long folder ", 4))
	}
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{deep, `\\?\` + deep} {
		v, err := d.Volume(ctx, p)
		if err != nil || v.Total <= 0 || v.Kind != Local {
			t.Errorf("Volume(%d chars) = %+v, %v", len(p), v, err)
		}
		if ok, err := d.Writable(ctx, p); err != nil || !ok {
			t.Errorf("Writable(%d chars) = %v, %v", len(p), ok, err)
		}
		if err := d.Mkdir(ctx, filepath.Join(p, "Season 1")); err != nil && !os.IsExist(err) {
			t.Errorf("Mkdir under %d chars: %v", len(p), err)
		}
		entries, err := d.ReadDir(ctx, p)
		if err != nil || !slices.Equal(entries, []Entry{{Name: "Season 1", Dir: true, Readable: true}}) {
			t.Errorf("ReadDir(%d chars) = %+v, %v", len(p), entries, err)
		}
	}
}

func TestUNCPath(t *testing.T) {
	ctx, d, dir := t.Context(), newOSDisk(t), t.TempDir()
	vol := filepath.VolumeName(dir)
	share := `\\localhost\` + strings.TrimSuffix(vol, ":") + `$`
	unc := share + dir[len(vol):]
	if _, err := d.Stat(ctx, unc); err != nil {
		t.Skipf("no administrative share to reach %s: %v", unc, err)
	}
	v, err := d.Volume(ctx, unc)
	if err != nil {
		t.Fatal(err)
	}
	if v.Kind != Network || !strings.EqualFold(v.Name, share) || !strings.EqualFold(v.Mount, share+`\`) || v.Total <= 0 {
		t.Errorf("Volume(%s) = %+v", unc, v)
	}
	if kind, err := KindOf(unc); err != nil || kind != Network {
		t.Errorf("KindOf(%s) = %v, %v", unc, kind, err)
	}
	if ok, err := d.Writable(ctx, unc); err != nil || !ok {
		t.Errorf("Writable(%s) = %v, %v", unc, ok, err)
	}
}
