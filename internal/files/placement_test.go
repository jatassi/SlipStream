package files_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
)

type refusing struct{}

func (refusing) Link(string, string) error { return errors.New("operation not permitted") }

func payload(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func sameFile(t *testing.T, a, b string) bool {
	t.Helper()
	x, err := os.Stat(a)
	if err != nil {
		t.Fatal(err)
	}
	y, err := os.Stat(b)
	if err != nil {
		t.Fatal(err)
	}
	return os.SameFile(x, y)
}

func TestPlaceHardLinksIntoANewFolder(t *testing.T) {
	from := payload(t, t.TempDir(), "orchard.mkv", "payload")
	to := filepath.Join(t.TempDir(), "The Quiet Orchard (2021)", "The Quiet Orchard (2021) - 1080p.mkv")
	if err := files.Place(files.OSLinker{}, from, to); err != nil {
		t.Fatal(err)
	}
	if !sameFile(t, from, to) {
		t.Fatal("the placed file is not a link to the download")
	}
}

func TestPlaceCopiesWhenTheLinkerRefusesAndConvergesWhenRepeated(t *testing.T) {
	from := payload(t, t.TempDir(), "orchard.mkv", "payload")
	dir := t.TempDir()
	to := filepath.Join(dir, "Orchard", "Orchard - 1080p.mkv")
	for range 2 {
		if err := files.Place(refusing{}, from, to); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(to)
	if err != nil || string(got) != "payload" {
		t.Fatalf("the placed file holds %q, %v; want payload", got, err)
	}
	if sameFile(t, from, to) {
		t.Fatal("the copy is the download itself")
	}
	entries, err := os.ReadDir(filepath.Join(dir, "Orchard"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "Orchard - 1080p.mkv" {
		t.Fatalf("the folder holds %v, %v; want only the placed file", entries, err)
	}
}

func TestPlaceRefusesAnotherFileAtTheTarget(t *testing.T) {
	from := payload(t, t.TempDir(), "orchard.mkv", "payload")
	dir := t.TempDir()
	to := payload(t, dir, "Orchard - 1080p.mkv", "another")
	if err := files.Place(files.OSLinker{}, from, to); !errors.Is(err, files.ErrOccupied) {
		t.Fatalf("Place over another file = %v, want ErrOccupied", err)
	}
	same := payload(t, dir, "same.mkv", "payload")
	if err := files.Place(refusing{}, from, same); err != nil {
		t.Fatalf("Place over the same bytes = %v, want nil", err)
	}
}

func TestMoveRenamesTheStagedFileAndConvergesWhenRepeated(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".slipstream-staging"), 0o700); err != nil {
		t.Fatal(err)
	}
	from := payload(t, filepath.Join(root, ".slipstream-staging"), "grab.part", "payload")
	to := filepath.Join(root, "Orchard", "Orchard - 1080p.mkv")
	for range 2 {
		if err := files.Move(from, to); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(to)
	if err != nil || string(got) != "payload" {
		t.Fatalf("the moved file holds %q, %v; want payload", got, err)
	}
	if entries, err := os.ReadDir(filepath.Join(root, ".slipstream-staging")); err != nil || len(entries) != 0 {
		t.Fatalf("the staging folder holds %v, %v; want nothing", entries, err)
	}
}

func TestMoveRefusesAnotherFileAndDropsAnIdenticalStagedCopy(t *testing.T) {
	dir := t.TempDir()
	to := payload(t, dir, "Orchard - 1080p.mkv", "payload")
	other := payload(t, dir, "other.part", "another")
	if err := files.Move(other, to); !errors.Is(err, files.ErrOccupied) {
		t.Fatalf("Move over another file = %v, want ErrOccupied", err)
	}
	same := payload(t, dir, "same.part", "payload")
	if err := files.Move(same, to); err != nil {
		t.Fatalf("Move over the same bytes = %v, want nil", err)
	}
	if _, err := os.Stat(same); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the identical staged copy is still there: %v", err)
	}
	if err := files.Move(filepath.Join(dir, "gone.part"), filepath.Join(dir, "missing.mkv")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Move with neither file = %v, want ErrNotExist", err)
	}
}
