package files_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/roots/rootstest"
	"github.com/jatassi/slipstream/internal/scanning"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	read1080 = media.Record{Version: media.Version, Container: "matroska,webm", Video: media.Video{Codec: "h264", Width: 1920, Height: 1080}}
	read2160 = media.Record{Version: media.Version, Container: "matroska,webm", Video: media.Video{Codec: "hevc", Width: 3840, Height: 2160}}
)

// walked is a file a walk listed at p, of size bytes, with file id n,
// modified n minutes into today.
func walked(t *testing.T, p string, size int64, n int) storing.Walked {
	t.Helper()
	rel, err := storing.ParseRel(p)
	if err != nil {
		t.Fatal(err)
	}
	return storing.Walked{Path: rel, Size: size, Seen: storing.Seen{
		Modified: today.Add(time.Duration(n) * time.Minute), ID: storing.FileID{Volume: "v1", File: strconv.Itoa(n)},
	}}
}

// found is a found file of item read at 1080p, whose name reads q.
func found(item works.ItemID, q release.Quality, at storing.Walked) files.Found {
	return files.Found{Item: item, Cut: release.Standard, At: at, Media: read1080, Quality: q}
}

// onRoot is l's root's files and held paths as a scan reads them now.
func (l *library) onRoot(t *testing.T) files.OnRootFiles {
	t.Helper()
	var out files.OnRootFiles
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = files.OnRoot(ctx, r, l.root.ID)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// compare is each Work's share of a scan of l's root whose walk lists
// entries, with every gone candidate confirmed.
func (l *library) compare(t *testing.T, entries ...storing.Walked) map[works.WorkID]scanning.Share {
	t.Helper()
	listing, err := scanning.Finish([]storing.Page{{Entries: entries, Done: true}})
	if err != nil {
		t.Fatal(err)
	}
	on := l.onRoot(t)
	plan := scanning.Compare(listing, on.Files, scanning.Known{Held: on.Held}, scanning.Guard{Share: 100})
	marked, ok := plan.Gone.(scanning.Marked)
	if !ok {
		t.Fatalf("the comparison stopped: %+v", plan.Gone)
	}
	return plan.ByWork(marked.Files)
}

func (l *library) scan(t *testing.T, svc *files.Files, s files.Scan) files.Scanned {
	t.Helper()
	var out files.Scanned
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		var err error
		out, err = svc.Scan(ctx, tx, s)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// markGone marks the files of work at paths gone, as a scan that found
// them absent does.
func (l *library) markGone(t *testing.T, svc *files.Files, work works.WorkID, paths ...string) {
	t.Helper()
	var gone []scanning.Recorded
	for _, r := range l.onRoot(t).Files {
		if slices.Contains(paths, r.Path.String()) {
			gone = append(gone, r)
		}
	}
	if len(gone) != len(paths) {
		t.Fatalf("%d of %v are recorded", len(gone), paths)
	}
	l.scan(t, svc, files.Scan{Work: work, Root: l.root.ID, Share: scanning.Share{Gone: gone}})
}

// listed is each file of work as its Files section lists it, by path: its
// label and N, whether it is Kept or gone, and how many cells it fills.
func (l *library) listed(t *testing.T, svc *files.Files, work works.WorkID) string {
	t.Helper()
	listing, err := svc.WorkFiles(t.Context(), admin, work)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range listing.Files {
		line := fmt.Sprintf("%s: label=%s extra=%d fills=%d", f.Path, f.Label, f.Extra, len(f.Fills))
		if f.Kept() {
			line += " kept"
		}
		if f.Gone() {
			line += " gone"
		}
		out = append(out, line)
	}
	slices.Sort(out)
	return strings.Join(out, "\n")
}

func TestAScanRecordsEveryFoundFileWhereItIsWithItsLabelAndN(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	svc := l.files(files.NoGrabs{})
	entries := []storing.Walked{
		walked(t, "Shelf w1/Scarf.mkv", 10, 1),
		walked(t, "Shelf w1/Scarf (copy).mkv", 11, 2),
		walked(t, "Shelf w1/Season/Mittens.mkv", 12, 3),
	}
	got := l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Found: []files.Found{
		found(scarf, release.WEBDL1080p, entries[0]), found(scarf, release.WEBDL1080p, entries[1]), found(mittens, release.WEBDL1080p, entries[2]),
	}})

	want := "Shelf w1/Scarf (copy).mkv: label=HD extra=1 fills=0 kept\n" +
		"Shelf w1/Scarf.mkv: label=HD extra=0 fills=1\n" +
		"Shelf w1/Season/Mittens.mkv: label=HD extra=0 fills=1"
	if list := l.listed(t, svc, tree.ID); !got.Changed || list != want {
		t.Fatalf("after the scan (%+v) the Work lists\n%s\nwant every file at the path it was found at, the second Scarf Kept\n%s", got, list, want)
	}
	if n := l.queued(t); n != 0 {
		t.Fatalf("the scan queued %d moves or removals, want none", n)
	}
	if again := l.compare(t, entries...); len(again) != 0 {
		t.Fatalf("a second walk of the same files gives %+v to write, want nothing: the scan recorded their stamps and records", again)
	}
}

func TestAFoundFileThatOutranksTheIncumbentLeavesItKeptWhereItIs(t *testing.T) {
	l := open(t)
	best := l.target(t, "Best", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "Best", best)
	tree := l.work(t, "w1", true, &p)
	scarf := tree.Items[0].ID
	svc := l.files(files.NoGrabs{})
	l.place(t, scarf, tree.ID, "Shelf w1/Shelf w1 - Scarf - Best.mkv", release.WEBDL1080p, best.ID)
	incumbent, bluray := walked(t, "Shelf w1/Shelf w1 - Scarf - Best.mkv", 1024, 1), walked(t, "Shelf w1/Scarf.BluRay.mkv", 2048, 2)

	shares := l.compare(t, incumbent, bluray)
	l.scan(t, svc, files.Scan{
		Work: tree.ID, Root: l.root.ID, Share: shares[tree.ID], Probes: map[storing.Rel]files.Probe{incumbent.Path: {Media: read1080}},
		Found: []files.Found{found(scarf, release.Bluray1080p, bluray)},
	})

	want := "Shelf w1/Scarf.BluRay.mkv: label=Best extra=0 fills=1\n" +
		"Shelf w1/Shelf w1 - Scarf - Best.mkv: label=Best extra=1 fills=0 kept"
	if got := l.listed(t, svc, tree.ID); got != want {
		t.Fatalf("the Work lists\n%s\nwant the Bluray filling Best and the WEB-DL Kept at its own path\n%s", got, want)
	}
	if n := l.queued(t); n != 0 {
		t.Fatalf("the scan queued %d moves or removals, want none: a scan never dooms or renames", n)
	}
}

func TestAFoundFileFillingNothingIsKept(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Found: []files.Found{found(tree.Items[0].ID, release.SDTV, walked(t, "Shelf w1/Scarf.avi", 10, 1))}})

	if got, want := l.listed(t, svc, tree.ID), "Shelf w1/Scarf.avi: label=SDTV extra=1 fills=0 kept"; got != want {
		t.Fatalf("the Work lists %q, want the SDTV file Kept under its quality's name: %q", got, want)
	}
}

func TestAGoneFileHoldsNoCellUntilItReturns(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	svc := l.files(files.NoGrabs{})
	l.place(t, scarf, tree.ID, "Shelf w1/Scarf.mkv", release.WEBDL1080p, hd.ID)
	l.place(t, mittens, tree.ID, "Shelf w1/Mittens.mkv", release.WEBDL1080p, hd.ID)
	scarfAt, mittensAt := walked(t, "Shelf w1/Scarf.mkv", 1024, 1), walked(t, "Shelf w1/Mittens.mkv", 1024, 2)

	shares := l.compare(t, mittensAt)
	l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Share: shares[tree.ID], Probes: map[storing.Rel]files.Probe{mittensAt.Path: {Media: read1080}}})

	_, cells, rollup, err := svc.WorkCells(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf any HD missing wanted=true\n" +
		"Mittens any HD met wanted=false Shelf w1/Mittens.mkv webdl-1080p 1920x1080\n"
	if got := lines(cells); got != want || rollup.Files != 1 {
		t.Fatalf("with Scarf's file gone cells =\n%s(%d live files)\nwant Scarf missing and one live file\n%s", got, rollup.Files, want)
	}
	listing := "Shelf w1/Mittens.mkv: label=HD extra=0 fills=1\nShelf w1/Scarf.mkv: label=HD extra=0 fills=0 gone"
	if got := l.listed(t, svc, tree.ID); got != listing {
		t.Fatalf("the Work lists\n%s\nwant the gone file kept with its label\n%s", got, listing)
	}
	held := must[map[works.WorkID]files.Held](t)(heldOf(t, l, tree.ID))
	if h := held[tree.ID]; h.Files != 1 || h.Bytes != 1024 {
		t.Fatalf("a delete would count %+v, want only the live file", h)
	}

	shares = l.compare(t, scarfAt, mittensAt)
	l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Share: shares[tree.ID], Probes: map[storing.Rel]files.Probe{scarfAt.Path: {Media: read1080}}})
	listing = "Shelf w1/Mittens.mkv: label=HD extra=0 fills=1\nShelf w1/Scarf.mkv: label=HD extra=0 fills=1"
	if got := l.listed(t, svc, tree.ID); got != listing {
		t.Fatalf("once Scarf's file is back the Work lists\n%s\nwant it filling its cell again\n%s", got, listing)
	}
}

// exec runs stmt in a Write of its own, as another writer would between a
// walk and its scan's Write.
func (l *library) exec(t *testing.T, stmt string, args ...any) {
	t.Helper()
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, stmt, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func heldOf(t *testing.T, l *library, ws ...works.WorkID) (map[works.WorkID]files.Held, error) {
	t.Helper()
	var out map[works.WorkID]files.Held
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = files.Holding(ctx, r, ws)
		return err
	})
	return out, err
}

func TestAScanDropsWhatChangedSinceTheWalk(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	scarf, mittens := tree.Items[0].ID, tree.Items[1].ID
	svc := l.files(files.NoGrabs{})
	l.place(t, scarf, tree.ID, "Shelf w1/Resized.mkv", release.WEBDL1080p, hd.ID)
	l.place(t, mittens, tree.ID, "Shelf w1/Moving.mkv", release.WEBDL1080p, hd.ID)
	l.place(t, scarf, tree.ID, "Shelf w1/Gone.mkv", release.WEBDL1080p)
	other := walked(t, "Shelf w1/Other.mkv", 5, 9)

	shares := l.compare(t, other)
	if n := len(shares[tree.ID].Gone); n != 3 {
		t.Fatalf("the walk found %d gone candidates, want 3", n)
	}
	l.exec(t, `UPDATE files_file SET size = 2048 WHERE path = 'Shelf w1/Resized.mkv'`)
	l.exec(t, `INSERT INTO files_move (file_id, sits) SELECT id, 'Shelf w1/Elsewhere.mkv' FROM files_file WHERE path = 'Shelf w1/Moving.mkv'`)
	got := l.scan(t, svc, files.Scan{
		Work: tree.ID, Root: l.root.ID, Share: shares[tree.ID],
		Found: []files.Found{found(mittens, release.WEBDL1080p, walked(t, "Shelf w1/Moving.mkv", 7, 8))},
	})

	want := "Shelf w1/Gone.mkv: label= extra=0 fills=0 gone\n" +
		"Shelf w1/Moving.mkv: label=HD extra=0 fills=1\n" +
		"Shelf w1/Resized.mkv: label=HD extra=0 fills=1"
	if list := l.listed(t, svc, tree.ID); got.Dropped != 3 || list != want {
		t.Fatalf("the scan dropped %d and the Work lists\n%s\nwant the resized and moving files and the found file at a recorded path dropped, the third gone\n%s", got.Dropped, list, want)
	}
}

func TestAResizedFileIsReadAgainAndLosesItsGrab(t *testing.T) {
	l := open(t)
	uhd := l.target(t, "UHD", []release.Quality{release.Bluray2160p})
	hd := l.target(t, "HD", []release.Quality{release.Bluray1080p})
	p := l.profile(t, "Both", uhd, hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	l.place(t, tree.Items[0].ID, tree.ID, "Shelf w1/Scarf.BluRay.mkv", release.Bluray1080p, hd.ID)
	grab := storage.NewUUID()
	l.exec(t, `INSERT INTO nodes_node (id, public_key, name, paired_at) VALUES (?, randomblob(32), 'home', 0)`, grab)
	l.exec(t, `INSERT INTO grabs_grab (id, work_id, item_id, edition, source, node_id, release_title, release_size, stage, failed_stage,
		failure_code, failure_message, created_at, updated_at) VALUES (?, ?, ?, 'any', 'search', ?, 'Scarf.BluRay', 1, 'failed', 'fetching',
		'indexer_removed', 'The Indexer was removed', 0, 0)`, grab, tree.ID, tree.Items[0].ID, grab)
	l.exec(t, `UPDATE files_file SET grab_id = ?`, grab)
	at := walked(t, "Shelf w1/Scarf.BluRay.mkv", 4096, 1)

	shares := l.compare(t, at)
	l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Share: shares[tree.ID], Probes: map[storing.Rel]files.Probe{at.Path: {Media: read2160}}})

	listing, err := svc.WorkFiles(t.Context(), admin, tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	f := listing.Files[0].File
	if f.Size != 4096 || f.Quality != release.Bluray2160p || f.Media.Width != 3840 || !f.GrabID.IsZero() || f.Label != "UHD" {
		t.Fatalf("the resized file reads %+v, want 4096 bytes read again at 2160p, Bluray-2160p filling UHD and no Grab", f)
	}
}

func TestAFileFfprobeCantReadKeepsItsRecordAndCountsAsUnreadable(t *testing.T) {
	l := open(t)
	hd := l.target(t, "HD", []release.Quality{release.WEBDL1080p})
	p := l.profile(t, "HD", hd)
	tree := l.work(t, "w1", true, &p)
	svc := l.files(files.NoGrabs{})
	l.place(t, tree.Items[0].ID, tree.ID, "Shelf w1/Scarf.mkv", release.WEBDL1080p, hd.ID)
	at := walked(t, "Shelf w1/Scarf.mkv", 1024, 1)
	line := "Invalid data found when processing input"

	shares := l.compare(t, at)
	l.scan(t, svc, files.Scan{Work: tree.ID, Root: l.root.ID, Share: shares[tree.ID], Probes: map[storing.Rel]files.Probe{at.Path: {Error: line}}})

	var unreadable map[roots.ID]files.Tally
	err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		unreadable, err = files.UnreadableOf(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	listing := must[files.Listing](t)(svc.WorkFiles(t.Context(), admin, tree.ID))
	f := listing.Files[0].File
	tally := unreadable[l.root.ID]
	if f.ProbeError != line || f.Media.Width != 1920 || f.Seen == nil || tally.Count != 1 || tally.First[0] != (files.Path{Path: at.Path.String(), Line: line}) {
		t.Fatalf("the unreadable file reads %+v and Health counts %+v, want ffprobe's line beside the record before it", f, tally)
	}
}

func TestAScanOfAWorkThatLeftTheRootIsRefused(t *testing.T) {
	l := open(t)
	tree := l.work(t, "w1", true, nil)
	other := rootstest.Root(t, l.lib, l.root.Node, contenttypetest.ID, t.TempDir())
	svc := l.files(files.NoGrabs{})
	err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := svc.Scan(ctx, tx, files.Scan{Work: tree.ID, Root: other.ID, Found: []files.Found{found(tree.Items[0].ID, release.WEBDL1080p, walked(t, "Shelf w1/Scarf.mkv", 1, 1))}})
		return err
	})
	if !errors.Is(err, files.ErrRootChanged) {
		t.Fatalf("a scan of another root's Work = %v, want ErrRootChanged", err)
	}
}
