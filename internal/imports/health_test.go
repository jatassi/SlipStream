package imports

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// shelve puts Works w1 to wn in the world's root, each with a file per
// clip, and answers their paths.
func (w *world) shelve(t *testing.T, n int) []string {
	t.Helper()
	var out []string
	for i := 1; i <= n; i++ {
		key := works.Key(fmt.Sprintf("w%d", i))
		tree := w.add(t, key, w.root.ID)
		for _, it := range tree.Items {
			p := "Shelf " + string(key) + "/" + it.Label + ".mkv"
			w.place(t, tree, p)
			out = append(out, p)
		}
	}
	return out
}

// scanned records what a scan of the world's root read of the files at
// paths: each gone, or each unreadable with line.
func (w *world) scanned(t *testing.T, paths []string, line string) {
	t.Helper()
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		on, err := files.OnRoot(ctx, tx, w.root.ID)
		if err != nil {
			return err
		}
		shares := map[works.WorkID]files.Scan{}
		for _, r := range on.Files {
			if slices.Contains(paths, r.Path.String()) {
				shares[r.Work] = read(shares[r.Work], w.root.ID, r, line)
			}
		}
		for _, s := range shares {
			if _, err := w.files.Scan(ctx, tx, s); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// read adds r to s as a scan of root read it: gone, or unreadable with
// line.
func read(s files.Scan, root roots.ID, r scanning.Recorded, line string) files.Scan {
	s.Work, s.Root = r.Work, root
	if line == "" {
		s.Share.Gone = append(s.Share.Gone, r)
		return s
	}
	at := storing.Walked{Path: r.Path, Size: r.Size, Seen: storing.Seen{Modified: time.UnixMilli(1_800_000_000_000)}}
	s.Share.Same = append(s.Share.Same, scanning.Same{Rec: r, At: at, Reprobe: scanning.Touched})
	if s.Probes == nil {
		s.Probes = map[storing.Rel]files.Probe{}
	}
	s.Probes[r.Path] = files.Probe{Error: line}
	return s
}

func (w *world) health(t *testing.T) map[health.Key]health.Item {
	t.Helper()
	a, err := w.im.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	out := map[health.Key]health.Item{}
	for _, it := range a.Items {
		out[it.Key] = it
	}
	if len(a.Unsure) != 0 {
		t.Errorf("the Library answers unsure %+v, want none", a.Unsure)
	}
	return out
}

func texts(fs []health.Fact) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Label + ": " + f.Text
	}
	return out
}

func TestHealthWarnsOfGoneFilesPerRoot(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	shelved := w.shelve(t, 3)
	words := "Shelves library root on Media server"
	if items := w.health(t); len(items) != 0 {
		t.Fatalf("a root with every file in place raises %+v, want nothing", items)
	}

	w.scanned(t, shelved[:1], "")
	key := health.KeyOf("gone", w.root.ID.String())
	it, ok := w.health(t)[key]
	if !ok || it.Level != health.Warn || it.Subject != words || it.Text != "1 file is gone. Restore it, or forget it on its Work's page." {
		t.Fatalf("one gone file raises %+v, want a Warn on the root saying 1 file is gone", it)
	}
	if got := texts(it.Detail); !slices.Equal(got, []string{"File: " + shelved[0]}) {
		t.Errorf("its detail = %v, want the gone file's path", got)
	}
	if a := it.Action; a.Kind() != health.ActionLink || a.Href() != "/library/fixture?root="+w.root.ID.String()+"&gone=1" || a.Label() != "Show the Works" {
		t.Errorf("its action = %s %q %q, want a link to the root's Works with gone files", a.Kind(), a.Href(), a.Label())
	}

	w.scanned(t, shelved[1:7], "")
	it = w.health(t)[key]
	if it.Text != "7 files are gone. Restore them, or forget them on each Work's page." || len(it.Detail) != 5 {
		t.Errorf("seven gone files raise %q with %d facts, want the count and the first five paths", it.Text, len(it.Detail))
	}
}

func TestHealthFailsWhileTheLastComparisonStopped(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	w.record(t, w.root.ID, Comparison{Place: w.root.Place, Stopped: &scanning.Stopped{Gone: 1912, Of: 2000}})
	key := health.KeyOf("stopped", w.root.ID.String())
	it, ok := w.health(t)[key]
	if !ok || it.Level != health.Fail || it.Subject != "Shelves library root on Media server" ||
		it.Text != "Library scan stopped because 1,912 of 2,000 files are gone. Check the disk or share it is on, then scan again." {
		t.Fatalf("a stopped comparison raises %+v, want a Fail on the root with its counts", it)
	}
	if a := it.Action; a.Kind() != health.ActionLink || a.Href() != "/settings/library-roots#root-"+w.root.ID.String() || a.Label() != "Open the root" {
		t.Errorf("its action = %s %q %q, want a link to the root's row", a.Kind(), a.Href(), a.Label())
	}
	w.record(t, w.root.ID, Comparison{Place: w.root.Place, Gone: 1912})
	if _, ok := w.health(t)[key]; ok {
		t.Errorf("a comparison that passed left the stopped item, want it cleared")
	}
	w.record(t, w.root.ID, Comparison{Place: w.root.Place + 1, Stopped: &scanning.Stopped{Gone: 3, Of: 4}})
	if _, ok := w.health(t)[key]; ok {
		t.Errorf("a comparison stopped at another place raises the item, want it about this place only")
	}
}

func TestHealthWarnsOfFilesFfprobeCantRead(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	shelved := w.shelve(t, 1)
	key := health.KeyOf("unreadable", w.root.ID.String())

	w.scanned(t, shelved[:1], "Invalid data found when processing input")
	it, ok := w.health(t)[key]
	if !ok || it.Level != health.Warn || it.Text != "ffprobe can't read 1 file. Replace it, then scan again." {
		t.Fatalf("one unreadable file raises %+v, want a Warn saying ffprobe can't read it", it)
	}
	if got := texts(it.Detail); !slices.Equal(got, []string{"File: " + shelved[0], "Error: Invalid data found when processing input"}) {
		t.Errorf("its detail = %v, want the path and ffprobe's line", got)
	}
	if it.Action.Kind() != health.NoAction {
		t.Errorf("its action = %s, want none", it.Action.Kind())
	}
	w.scanned(t, shelved[1:], "moov atom not found")
	if it := w.health(t)[key]; it.Text != "ffprobe can't read 3 files. Replace them, then scan again." || len(it.Detail) != 6 {
		t.Errorf("three unreadable files raise %q with %d facts, want the count and each path with its line", it.Text, len(it.Detail))
	}
}

// TestHealthWarnsOfAFreshImportPutBackInItsHoldingFolder holds a file in
// its root's holding folder with no move queued, as the disk queue leaves
// a fresh import whose name another file has: Health warns of it and
// links its Work, and the warning clears once the file has its name.
func TestHealthWarnsOfAFreshImportPutBackInItsHoldingFolder(t *testing.T) {
	t.Parallel()
	w := newWorld(t)
	tree := w.add(t, "w1", w.root.ID)
	var id files.ID
	held := storing.HoldingDir + "/" + storage.NewUUID().String() + ".mkv"
	err := w.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		f, err := files.Add(ctx, tx, files.File{
			WorkID: tree.ID, Items: []works.ItemID{tree.Items[0].ID}, Cut: release.Standard, RootID: w.root.ID,
			Path: held, Size: 1 << 30, Quality: release.Bluray1080p,
			Media: media.Record{Version: media.Version, Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080}},
		}, []targets.TargetID{w.hd})
		id = f.ID
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	key := health.KeyOf("put_back", id.String())
	it, ok := w.health(t)[key]
	if !ok || it.Level != health.Warn || it.Subject != "Shelves library root on Media server" {
		t.Fatalf("a put-back file raises %+v, want a Warn on its root", it)
	}
	if want := "A file imported into Shelves went back to its holding folder because another file already has its name, Shelf w1 - Clip 1 - HD.mkv. Rename or remove that file."; it.Text != want {
		t.Errorf("it says %q, want %q", it.Text, want)
	}
	if a := it.Action; a.Kind() != health.ActionLink || a.Href() != "/works/"+tree.ID.String() || a.Label() != "Open the Work" {
		t.Errorf("its action = %s %q %q, want a link to the Work", a.Kind(), a.Href(), a.Label())
	}
	w.scanned(t, []string{held}, "")
	if _, ok := w.health(t)[key]; ok {
		t.Error("the file removed from its holding folder still raises the warning, want it cleared")
	}
}
