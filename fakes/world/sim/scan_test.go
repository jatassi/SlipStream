package sim_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/imports"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/media/mediatest"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
)

// films are the movies fake-tmdb lists beside the Scenario's own, each
// shelved in its own folder tagged with its TMDB id.
var films = []scenario.TMDBMovie{
	{ID: 9100001, IMDbID: "tt9100001", Title: "Paper Lanterns", Released: "2019-03-01", Runtime: 98 * time.Minute},
	{ID: 9100002, IMDbID: "tt9100002", Title: "Northern Glass", Released: "2018-06-15", Runtime: 110 * time.Minute},
	{ID: 9100003, IMDbID: "tt9100003", Title: "Salt Road", Released: "2020-09-04", Runtime: 121 * time.Minute},
	{ID: 9100004, IMDbID: "tt9100004", Title: "Copper Weather", Released: "2017-11-10", Runtime: 93 * time.Minute},
}

var orchard = scenario.TMDBMovie{ID: 9000001, Title: "The Quiet Orchard", Released: "2021-04-16"}

func folderOf(m scenario.TMDBMovie) string {
	return m.Title + " (" + m.Released[:4] + ") {tmdb-" + strconv.FormatInt(m.ID, 10) + "}"
}

func fileOf(m scenario.TMDBMovie) string {
	return folderOf(m) + "/" + m.Title + " (" + m.Released[:4] + ") WEBDL-1080p.mkv"
}

var (
	hd = media.Record{Version: media.Version, Container: "matroska,webm", Duration: 98 * time.Minute, Video: media.Video{Codec: "h264", Width: 1920, Height: 1080}}
	sd = media.Record{Version: media.Version, Container: "matroska,webm", Duration: 98 * time.Minute, Video: media.Video{Codec: "h264", Width: 1280, Height: 720}}
)

// prober reads each file under the media dir as mediatest.Prober keys it,
// by its path in its root, and keeps the reads; it holds a read of a file
// hold names until the read's context ends. Any other path, such as a
// download's, goes to the walk's prober.
type prober struct {
	mu    sync.Mutex
	dir   string
	walk  media.Prober
	of    mediatest.Prober
	reads []string
	hold  string
	held  chan struct{}
}

func (p *prober) Probe(ctx context.Context, path string) (media.Record, error) {
	p.mu.Lock()
	rel, err := filepath.Rel(p.dir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		p.mu.Unlock()
		return p.walk.Probe(ctx, path)
	}
	rel = filepath.ToSlash(rel)
	p.reads = append(p.reads, rel)
	if p.hold == "" || !strings.HasSuffix(rel, "/"+p.hold) {
		defer p.mu.Unlock()
		return p.of.Probe(ctx, path)
	}
	close(p.held)
	p.hold = ""
	p.mu.Unlock()
	<-ctx.Done()
	return media.Record{}, fmt.Errorf("hold the read of %s: %w", rel, ctx.Err())
}

func (p *prober) answer(rel string, r media.Record) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.of.Of[rel] = r
}

// holding holds the next read of rel, and is closed once it starts.
func (p *prober) holding(rel string) <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.hold, p.held = rel, make(chan struct{})
	return p.held
}

// took is the files read since it was last asked, by their paths in the
// media dir.
func (p *prober) took() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.reads
	p.reads = nil
	slices.Sort(out)
	return out
}

// shelf is a movies root, whose files a test puts on disk, and the library
// a scan makes of them.
type shelf struct {
	t      *testing.T
	w      *sim.World
	dir    string
	probes *prober
}

func shelve(t *testing.T, edit func(*scenario.Scenario), opts ...sim.Option) *shelf {
	t.Helper()
	s := home(t)
	s.TMDB.Movies = append(s.TMDB.Movies, films...)
	if edit != nil {
		edit(&s)
	}
	p := &prober{of: mediatest.Prober{Of: map[string]media.Record{}, Fail: map[string]string{}}}
	w := sim.Launch(t, s, append([]sim.Option{sim.WithWalk(), sim.Edit(func(_ string, e *node.Env) {
		p.mu.Lock()
		defer p.mu.Unlock()
		p.walk = e.Media
		e.Media = p
	})}, opts...)...)
	p.mu.Lock()
	p.dir = w.Manifest().MediaDir
	p.mu.Unlock()
	sh := &shelf{t: t, w: w, dir: filepath.Join(w.Manifest().MediaDir, "movies"), probes: p}
	sh.idle()
	return sh
}

// on is the root at folder under the media dir.
func (s *shelf) on(folder string) *shelf {
	o := *s
	o.dir = filepath.Join(s.w.Manifest().MediaDir, folder)
	return &o
}

func (s *shelf) path(rel string) string { return filepath.Join(s.dir, filepath.FromSlash(rel)) }

// put writes size bytes at rel under the root, a file that reads as r.
func (s *shelf) put(rel string, size int, r media.Record) {
	s.t.Helper()
	s.probes.answer(rel, r)
	if err := os.MkdirAll(filepath.Dir(s.path(rel)), 0o750); err != nil {
		s.t.Fatal(err)
	}
	b := make([]byte, size)
	_, _ = rand.NewChaCha8(sha256.Sum256([]byte(rel))).Read(b)
	if err := os.WriteFile(s.path(rel), b, 0o600); err != nil {
		s.t.Fatal(err)
	}
}

func (s *shelf) stock(ms ...scenario.TMDBMovie) {
	s.t.Helper()
	for i, m := range ms {
		s.put(fileOf(m), 4096+i, hd)
	}
}

func (s *shelf) rename(from, to string) {
	s.t.Helper()
	if err := os.MkdirAll(filepath.Dir(to), 0o750); err != nil {
		s.t.Fatal(err)
	}
	if err := os.Rename(from, to); err != nil {
		s.t.Fatal(err)
	}
}

// aside is where away puts rel, out of every root.
func (s *shelf) aside(rel string) string {
	return filepath.Join(s.w.Manifest().MediaDir, "aside", filepath.FromSlash(rel))
}

// away moves rel out of the root, keeping its bytes and identity.
func (s *shelf) away(rel string) { s.rename(s.path(rel), s.aside(rel)) }

// back moves rel back to where away took it from.
func (s *shelf) back(rel string) { s.rename(s.aside(rel), s.path(rel)) }

func (s *shelf) rootID() roots.ID {
	s.t.Helper()
	id, err := storage.ParseUUID(rootAt(s.t, s.w.API(), s.dir).ID)
	if err != nil {
		s.t.Fatal(err)
	}
	return roots.ID{UUID: id}
}

// idle waits until Library scan has run at least once and runs no more.
func (s *shelf) idle() {
	s.t.Helper()
	eventually(s.t, "Library scan is idle after a run", func() bool {
		d := getTask(s.t, s.w.API(), "library-scan")
		return d.Task.State.Kind != "running" && len(d.Runs) > 0
	})
}

// scan is Run now on Library scan, once it has ended.
func (s *shelf) scan() scanned {
	s.t.Helper()
	runNow(s.t, s.w.API(), "library-scan")
	s.idle()
	return s.last()
}

type scanned struct {
	trigger, result string
	// units are label|node|outcome|cause.
	units []string
}

func (s *shelf) last() scanned {
	s.t.Helper()
	d := getTask(s.t, s.w.API(), "library-scan")
	return scanned{trigger: d.Runs[0].Trigger, result: d.Runs[0].Result, units: d.units()}
}

func expectScanned(t *testing.T, got scanned, result string, units ...string) {
	t.Helper()
	if got.result != result || !slices.Equal(got.units, units) {
		t.Fatalf("the Library scan run is %s with units %q, want %s with %q", got.result, got.units, result, units)
	}
}

// works is the library's movies, by label.
func (s *shelf) works() map[string]string {
	s.t.Helper()
	type row struct{ ID, Label string }
	rows := decode[struct{ Rows []row }](s.t, "listLibrary", s.w.API().Admin("home", http.MethodGet, "/api/v1/library?contentType=movie", "")).Rows
	out := map[string]string{}
	for _, r := range rows {
		out[r.Label] = r.ID
	}
	return out
}

func (s *shelf) expectWorks(want ...scenario.TMDBMovie) {
	s.t.Helper()
	got := s.works()
	if len(got) != len(want) {
		s.t.Fatalf("the library holds %v, want %d movies", got, len(want))
	}
	for _, m := range want {
		if _, ok := got[m.Title]; !ok {
			s.t.Fatalf("the library holds %v, want %s in it", got, m.Title)
		}
	}
}

// filled is the path of the file that fills m's cell, "missing" while
// none does.
func (s *shelf) filled(m scenario.TMDBMovie) string {
	s.t.Helper()
	type cell struct {
		State string
		File  *struct{ Path string }
	}
	work := s.works()[m.Title]
	cells := decode[struct{ Cells []cell }](s.t, "getWorkCells", s.w.API().Admin("home", http.MethodGet, "/api/v1/works/"+work+"/cells", "")).Cells
	if len(cells) != 1 {
		s.t.Fatalf("%s has cells %+v, want one", m.Title, cells)
	}
	if cells[0].File == nil {
		return cells[0].State
	}
	return cells[0].File.Path
}

func (s *shelf) expectFilled(ms ...scenario.TMDBMovie) {
	s.t.Helper()
	for _, m := range ms {
		if got := s.filled(m); got != fileOf(m) {
			s.t.Fatalf("%s's cell holds %s, want %s", m.Title, got, fileOf(m))
		}
	}
}

// health is the Library category's items, as level: subject: text.
func (s *shelf) health() []string {
	s.t.Helper()
	out := []string{}
	for _, it := range healthItems(s.t, s.w.API(), "library") {
		out = append(out, it.Level+": "+it.Subject+": "+it.Text)
	}
	slices.Sort(out)
	return out
}

func (s *shelf) expectHealth(want ...string) {
	s.t.Helper()
	if got := s.health(); !slices.Equal(got, want) {
		s.t.Fatalf("Health's Library items read %q, want %q", got, want)
	}
}

func (s *shelf) count(query string) int {
	s.t.Helper()
	var n int
	if err := s.w.Node("home").Half().Library.QueryRowContext(s.t.Context(), query).Scan(&n); err != nil {
		s.t.Fatal(err)
	}
	return n
}

func (s *shelf) expectReads(what string, want ...string) {
	s.t.Helper()
	if got := s.probes.took(); !slices.Equal(got, want) && (len(got) != 0 || len(want) != 0) {
		s.t.Fatalf("%s read %q, want %q", what, got, want)
	}
}

const (
	onHome         = "Movies library root on home"
	filmsOnSeedbox = "Movies library root on seedbox"
	succeeded      = onHome + "|home|succeeded|"
)

func goneWords(n int) string {
	if n == 1 {
		return "warn: " + onHome + ": 1 file is gone. Restore it, or forget it on its Work's page."
	}
	return "warn: " + onHome + ": " + strconv.Itoa(n) + " files are gone. Restore them, or forget them on each Work's page."
}

// TestAFirstScanAddsTheWorksItIsSureOfAndAnUnchangedRootReadsNothing
// stocks the movies root, which its first Ready report scanned empty,
// with four tagged folders and one no source lists: a scan adds the four
// as Works filled by their files and leaves the fifth in Pending imports,
// and the next scan of the unchanged root reads no file.
func TestAFirstScanAddsTheWorksItIsSureOfAndAnUnchangedRootReadsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		if got := s.last(); got.trigger != "event" || got.result != "succeeded" {
			t.Fatalf("the movies root's first Ready report started a %s run that ended %s, want an event run that succeeded", got.trigger, got.result)
		}
		s.stock(films...)
		s.put("Halloween/Halloween.mkv", 2048, hd)
		expectScanned(t, s.scan(), "succeeded", succeeded)
		if got := s.probes.took(); len(got) != len(films)+1 {
			t.Fatalf("the first scan read %q, want each of the %d files once", got, len(films)+1)
		}
		s.expectWorks(films...)
		s.expectFilled(films...)
		if n := s.count("SELECT count(*) FROM imports_pending WHERE folder = 'Halloween' AND ignored_at IS NULL"); n != 1 {
			t.Fatalf("Pending imports hold %d entries for Halloween, want 1", n)
		}

		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectReads("a scan of the unchanged root")
		s.expectWorks(films...)
		if n := s.count("SELECT count(*) FROM imports_pending"); n != 1 {
			t.Fatalf("Pending imports hold %d entries after a scan of the unchanged root, want 1", n)
		}
	})
}

// TestADeletedFileReadsGoneAndWholeOnceRestored takes one Work's file off
// the root: a scan marks it gone, its cell reads missing and Health counts
// it; once the file is back the next scan fills the cell with it again.
func TestADeletedFileReadsGoneAndWholeOnceRestored(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films...)
		s.scan()
		s.probes.took()

		s.away(fileOf(films[0]))
		expectScanned(t, s.scan(), "succeeded", succeeded)
		if got := s.filled(films[0]); got != "missing" {
			t.Fatalf("%s's cell holds %s with its file gone, want missing", films[0].Title, got)
		}
		s.expectHealth(goneWords(1))

		s.back(fileOf(films[0]))
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectFilled(films...)
		s.expectHealth()
		s.expectReads("scans of a file gone and back")
	})
}

// TestAMassVanishStopsUntilMarkThemGone moves two of four Work folders
// out of the root: the scan marks none gone and stops in Health's Library
// category alone. A third vanishing before Mark them gone stops it again
// with the new count; Mark them gone then applies it, and the folders
// moved back read whole.
func TestAMassVanishStopsUntilMarkThemGone(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films...)
		s.scan()
		stopped := func(gone int) string {
			return "Library scan stopped because " + strconv.Itoa(gone) + " of 4 files are gone. Check the disk or share it is on, then scan again."
		}

		s.away(folderOf(films[0]))
		s.away(folderOf(films[1]))
		expectScanned(t, s.scan(), "skipped", onHome+"|home|skipped|"+stopped(2))
		s.expectFilled(films...)
		s.expectHealth("fail: " + onHome + ": " + stopped(2))
		if items := healthItems(t, s.w.API(), "scheduled-tasks"); len(items) != 0 {
			t.Fatalf("a stopped scan raised Scheduled tasks items %+v, want none", items)
		}

		s.away(folderOf(films[2]))
		im := s.w.Node("home").Half().Imports
		if _, err := im.MarkGone(t.Context(), identity.Admin{}, s.rootID()); err != nil {
			t.Fatal(err)
		}
		s.idle()
		expectScanned(t, s.last(), "skipped", onHome+"|home|skipped|"+stopped(3))

		if _, err := im.MarkGone(t.Context(), identity.Admin{}, s.rootID()); err != nil {
			t.Fatal(err)
		}
		s.idle()
		expectScanned(t, s.last(), "succeeded", succeeded)
		for _, m := range films[:3] {
			if got := s.filled(m); got != "missing" {
				t.Fatalf("%s's cell holds %s once its vanish is marked gone, want missing", m.Title, got)
			}
		}
		s.expectHealth(goneWords(3))
		if _, err := im.MarkGone(t.Context(), identity.Admin{}, s.rootID()); !errors.Is(err, imports.ErrNotStopped) {
			t.Fatalf("Mark them gone after a scan that didn't stop = %v, want ErrNotStopped", err)
		}

		for _, m := range films[:3] {
			s.back(folderOf(m))
		}
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectFilled(films...)
		s.expectHealth()
	})
}

// TestAnUnmountedRootIsSkippedAndItsFilesStay unmounts the movies root:
// its unit is Skipped with Storage's words and no file reads gone.
func TestAnUnmountedRootIsSkippedAndItsFilesStay(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films...)
		s.scan()

		lift := s.w.Disk("home", s.dir, fault.Unmounted{})
		checkStorage(t, s.w.API(), "")
		expectScanned(t, s.scan(), "skipped", onHome+"|home|skipped|The folder is there but looks unmounted. Plug in or mount the disk.")
		s.expectHealth()
		lift()
		checkStorage(t, s.w.API(), "")
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectFilled(films...)
	})
}

// TestAHungFolderFailsTheUnitFast hangs one Work folder: the unit fails
// within the stat deadline with Storage's words, and nothing reads gone.
func TestAHungFolderFailsTheUnitFast(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films...)
		s.scan()

		lift := s.w.Disk("home", s.path(folderOf(films[1])), fault.Hung{})
		began := time.Now()
		expectScanned(t, s.scan(), "failed", onHome+"|home|failed|The folder isn't responding. Check the disk or share it is on.")
		if took := time.Since(began); took > storing.DefaultTimings().Stat+5*time.Second {
			t.Fatalf("the scan of a root with a hung folder took %s, want it to fail within the %s stat deadline", took, storing.DefaultTimings().Stat)
		}
		s.expectHealth()
		lift()
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectFilled(films...)
	})
}

// stalled is a Disk that holds a listing of folder until the call's context
// ends, and keeps why it ended.
type stalled struct {
	storing.Disk
	mu      sync.Mutex
	folder  string
	reached chan struct{}
	ended   chan error
}

func (d *stalled) hold(folder string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.folder = folder
}

func (d *stalled) ReadDir(ctx context.Context, dir string) ([]storing.Entry, error) {
	d.mu.Lock()
	folder := d.folder
	d.mu.Unlock()
	if dir != folder {
		return d.Disk.ReadDir(ctx, dir)
	}
	close(d.reached)
	<-ctx.Done()
	d.ended <- ctx.Err()
	return nil, fmt.Errorf("hold the listing of %s: %w", dir, ctx.Err())
}

// TestCancelMidWalkStopsTheWorkersWalk puts a root on seedbox whose
// second Walk page reaches a folder that hangs: Cancel ends the walk on
// seedbox and the unit, and records nothing.
func TestCancelMidWalkStopsTheWorkersWalk(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		d := &stalled{reached: make(chan struct{}), ended: make(chan error, 1)}
		s := shelve(t, func(sc *scenario.Scenario) {
			sc.Nodes[1].Roles = append(sc.Nodes[1].Roles, "storage")
			sc.Library.Roots = append(sc.Library.Roots, scenario.LibraryRoot{ContentType: "movie", Folder: "films", Node: "seedbox"})
		}, onSeedbox(func(e *node.Env) {
			disk := e.Disk
			e.Disk = func(data string) (storing.Disk, error) {
				base, err := disk(data)
				d.Disk = base
				return d, err
			}
		}))
		films := s.on("films")
		for i := range 1001 {
			films.put(filepath.Join("Extras pile", strconv.Itoa(i)+".mkv"), 1, hd)
		}
		films.put("Paper Lanterns (2019)/Paper Lanterns (2019).mkv", 4096, hd)
		d.hold(films.path("Paper Lanterns (2019)"))

		run := runNow(t, s.w.API(), "library-scan")
		<-d.reached
		cancel(t, s, run)
		s.idle()
		if err := <-d.ended; !errors.Is(err, context.Canceled) {
			t.Fatalf("seedbox's walk ended with %v, want Canceled", err)
		}
		got := s.last()
		if got.result != "canceled" || !slices.Contains(got.units, filmsOnSeedbox+"|seedbox|canceled|") {
			t.Fatalf("the canceled run is %s with units %q, want canceled with seedbox's unit canceled", got.result, got.units)
		}
		if n := s.count("SELECT count(*) FROM files_file") + s.count("SELECT count(*) FROM imports_pending"); n != 0 {
			t.Fatalf("a scan canceled mid-walk recorded %d rows, want none", n)
		}
	})
}

// TestCancelWhileSettlingKeepsTheFoldersWritten cancels a scan while it
// reads the second of three new folders: the first folder's Work stays,
// and the next scan adds the other two once each.
func TestCancelWhileSettlingKeepsTheFoldersWritten(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films[:3]...)
		reached := s.probes.holding(fileOf(films[0]))
		run := runNow(t, s.w.API(), "library-scan")
		<-reached
		cancel(t, s, run)
		s.idle()
		expectScanned(t, s.last(), "canceled", onHome+"|home|canceled|")
		s.expectWorks(films[1])

		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectWorks(films[:3]...)
		s.expectFilled(films[:3]...)
	})
}

func cancel(t *testing.T, s *shelf, run int64) {
	t.Helper()
	r := s.w.API().Admin("home", http.MethodDelete, "/api/v1/tasks/runs/"+strconv.FormatInt(run, 10), "")
	if r.Status != http.StatusNoContent {
		t.Fatalf("cancelTaskRun = %d %s, want 204", r.Status, r.Body)
	}
}

// TestACrashMidScanConverges stops the Controller while a scan reads the
// second of three new folders: after the restart, the next scan ends with
// each folder's Work and file once, as one uninterrupted scan would.
func TestACrashMidScanConverges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films[:3]...)
		reached := s.probes.holding(fileOf(films[0]))
		runNow(t, s.w.API(), "library-scan")
		<-reached
		h := s.w.Node("home")
		h.Stop()
		time.Sleep(time.Minute)
		h.Start()
		synctest.Wait()

		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectWorks(films[:3]...)
		s.expectFilled(films[:3]...)
		if n := s.count("SELECT count(*) FROM files_file"); n != 3 {
			t.Fatalf("the library records %d files, want the 3 once each", n)
		}
	})
}

// TestARenamedFileKeepsItsRecord renames a file in its folder: the scan
// matches it by its file id, reads nothing and fills the cell from the
// new path.
func TestARenamedFileKeepsItsRecord(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films...)
		s.scan()
		s.probes.took()

		renamed := folderOf(films[0]) + "/Paper Lanterns.mkv"
		s.rename(s.path(fileOf(films[0])), s.path(renamed))
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectReads("a scan after a rename")
		if got := s.filled(films[0]); got != renamed {
			t.Fatalf("%s's cell holds %s after the rename, want %s", films[0].Title, got, renamed)
		}
		s.expectWorks(films...)
		if n := s.count("SELECT count(*) FROM imports_pending"); n != 0 {
			t.Fatalf("a rename left %d pending imports, want none", n)
		}
	})
}

// TestAFileMovedOnARootWithoutIDsKeepsItsRecord moves a file to another
// folder of a root whose volume keeps no file ids: the scan matches it by
// its size and name.
func TestAFileMovedOnARootWithoutIDsKeepsItsRecord(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.w.Disk("home", s.dir, fault.NoIDs{})
		s.stock(films...)
		s.scan()
		s.probes.took()

		moved := "Sorted/" + filepath.Base(fileOf(films[0]))
		s.rename(s.path(fileOf(films[0])), s.path(moved))
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectReads("a scan after a move")
		if got := s.filled(films[0]); got != moved {
			t.Fatalf("%s's cell holds %s after the move, want %s", films[0].Title, got, moved)
		}
		if n := s.count("SELECT count(*) FROM imports_pending"); n != 0 {
			t.Fatalf("a move left %d pending imports, want none", n)
		}
	})
}

// TestACaseOnlyRenameIsAMoveOnACaseInsensitiveRoot respells a file's
// name in lower case on a root that resolves names ignoring case.
func TestACaseOnlyRenameIsAMoveOnACaseInsensitiveRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.w.Disk("home", s.dir, fault.FoldCase{})
		s.stock(films...)
		s.scan()
		s.probes.took()

		lower := folderOf(films[0]) + "/" + strings.ToLower(filepath.Base(fileOf(films[0])))
		s.rename(s.path(fileOf(films[0])), s.path(lower))
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectReads("a scan after a case-only rename")
		if got := s.filled(films[0]); got != lower {
			t.Fatalf("%s's cell holds %s after the case-only rename, want %s", films[0].Title, got, lower)
		}
		if n := s.count("SELECT count(*) FROM imports_pending"); n != 0 {
			t.Fatalf("a case-only rename left %d pending imports, want none", n)
		}
	})
}

// gate holds whoever reaches it until it opens.
type gate struct {
	mu      sync.Mutex
	armed   bool
	reached chan struct{}
	open    chan struct{}
}

func newGate() *gate { return &gate{reached: make(chan struct{}), open: make(chan struct{})} }

func (g *gate) arm() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.armed = true
}

func (g *gate) pass() {
	g.mu.Lock()
	armed := g.armed
	g.armed = false
	g.mu.Unlock()
	if armed {
		close(g.reached)
		<-g.open
	}
}

// TestAScanDuringAnImportChangesNothing walks the movies root while a
// Grab waits to import, then lets the import land before the scan
// compares, while bytes the disk queue can't remove yet still lie there:
// nothing reads gone, found or pending.
func TestAScanDuringAnImportChangesNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		grab, scan := newGate(), newGate()
		s := shelve(t, nil, sim.Edit(func(_ string, e *node.Env) {
			e.Test.BeforeStep = func(_ context.Context, st grabs.Stage) {
				if st == grabs.Importing {
					grab.pass()
				}
			}
			e.Test.BeforeScanStep = func(_ context.Context, _ roots.ID, p imports.Phase) {
				if p == imports.Comparing {
					scan.pass()
				}
			}
		}))
		s.stock(films...)
		s.scan()

		lift := s.w.Disk("home", s.path(fileOf(films[3])), fault.ReadOnly{})
		deleted := s.w.API().Admin("home", http.MethodPost, "/api/v1/works/delete", `{"workIds":["`+s.works()[films[3].Title]+`"],"files":"delete"}`)
		if deleted.Status != http.StatusOK {
			t.Fatalf("deleteWorks = %d %s", deleted.Status, deleted.Body)
		}
		eventually(t, "the deleted Work's file waits in the disk queue", func() bool {
			return s.count("SELECT count(*) FROM files_doomed") == 1
		})

		grab.arm()
		search(t, s.w, addOrchard(t, s.w))
		<-grab.reached
		scan.arm()
		runNow(t, s.w.API(), "library-scan")
		<-scan.reached
		close(grab.open)
		expectImported(t, await(t, s.w))
		close(scan.open)
		s.idle()

		expectScanned(t, s.last(), "succeeded", succeeded)
		var found, gone int
		if err := s.w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT found, gone FROM imports_root").Scan(&found, &gone); err != nil {
			t.Fatal(err)
		}
		if found != 0 || gone != 0 {
			t.Fatalf("the scan during the import found %d files and %d gone, want none", found, gone)
		}
		if n := s.count("SELECT count(*) FROM imports_pending"); n != 0 {
			t.Fatalf("the scan during the import left %d pending imports, want none", n)
		}
		s.expectHealth()
		if _, err := os.Stat(s.path(fileOf(films[3]))); err != nil {
			t.Fatalf("the doomed file is gone before its fault lifted: %v", err)
		}
		lift()
	})
}

// importOrchard grabs and imports The Quiet Orchard into the movies root,
// and answers its file's path in the root, which reads as hd.
func importOrchard(t *testing.T, s *shelf) string {
	t.Helper()
	search(t, s.w, addOrchard(t, s.w))
	expectImported(t, await(t, s.w))
	var path string
	if err := s.w.Node("home").Half().Library.QueryRowContext(t.Context(), "SELECT path FROM files_file WHERE gone_at IS NULL AND path LIKE 'The Quiet Orchard%'").Scan(&path); err != nil {
		t.Fatal(err)
	}
	s.probes.answer(path, hd)
	return path
}

// TestRowsFromBeforeLibraryScanTakeTheirStamps rewinds an imported file's
// row to how rows read before this spec: the next scan reads it once,
// records its modification time and id, and keeps its quality and Grab.
func TestRowsFromBeforeLibraryScanTakeTheirStamps(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		path := importOrchard(t, s)
		lib := s.w.Node("home").Half().Library
		err := lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE files_file SET probe = '{"container":"matroska,webm","codec":"h264","width":1920,"height":1080}',
				modified_at = NULL, node_file_id = NULL, node_volume_id = NULL, probed_at = NULL`)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		type row struct {
			quality  string
			grab     []byte
			version  int
			modified sql.NullInt64
		}
		read := func() row {
			var r row
			if err := lib.QueryRowContext(t.Context(), "SELECT quality, grab_id, probe_version, modified_at FROM files_file").Scan(&r.quality, &r.grab, &r.version, &r.modified); err != nil {
				t.Fatal(err)
			}
			return r
		}
		before := read()
		s.probes.took()

		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectReads("the first scan of a row from before Library scan", "movies/"+path)
		after := read()
		if after.quality != before.quality || !slices.Equal(after.grab, before.grab) || after.version != media.Version || !after.modified.Valid {
			t.Fatalf("the row reads %+v after its first scan, want %+v with its stamps and media version %d", after, before, media.Version)
		}
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectReads("the next scan")
	})
}

// TestAGoneFilesCellIsFilledByTheNextImport takes The Quiet Orchard's
// file off the root: once a Grab imports another, it fills the cell and
// the gone file is forgotten.
func TestAGoneFilesCellIsFilledByTheNextImport(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films...)
		s.put(fileOf(orchard), 8192, hd)
		s.scan()
		s.away(fileOf(orchard))
		s.scan()
		if got := s.filled(orchard); got != "missing" {
			t.Fatalf("The Quiet Orchard's cell holds %s with its file gone, want missing", got)
		}

		search(t, s.w, s.works()[orchard.Title])
		expectImported(t, await(t, s.w))
		if got := s.filled(orchard); got == "missing" || got == fileOf(orchard) {
			t.Fatalf("The Quiet Orchard's cell holds %s after the import, want the imported file", got)
		}
		s.expectHealth()
		if n := s.count("SELECT count(*) FROM files_file WHERE gone_at IS NOT NULL"); n != 0 {
			t.Fatalf("the library keeps %d gone files once the import filled their cells, want none", n)
		}
	})
}

// TestAGoneFileThatReturnsTakesBackItsCell puts a 720p file beside The
// Quiet Orchard's gone 1080p file: when the 1080p file returns, it fills
// the cell again and the 720p file stays a Kept file.
func TestAGoneFileThatReturnsTakesBackItsCell(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, nil)
		s.stock(films...)
		s.put(fileOf(orchard), 8192, hd)
		s.scan()
		s.away(fileOf(orchard))
		s.scan()
		s.put(folderOf(orchard)+"/The Quiet Orchard (2021) HDTV-720p.mkv", 2048, sd)
		s.scan()

		s.back(fileOf(orchard))
		expectScanned(t, s.scan(), "succeeded", succeeded)
		s.expectFilled(orchard)
		if n := s.count("SELECT count(*) FROM files_file WHERE path LIKE 'The Quiet Orchard%' AND gone_at IS NULL"); n != 2 {
			t.Fatalf("The Quiet Orchard keeps %d live files, want the returned one and the 720p one", n)
		}
		s.expectHealth()
	})
}

// TestARootOnAnOfflineWorkerIsSkippedThenScannedOnReturn cuts seedbox off
// with a film in its root: its unit is Skipped, and once seedbox returns
// it runs and adds the film.
func TestARootOnAnOfflineWorkerIsSkippedThenScannedOnReturn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := shelve(t, func(sc *scenario.Scenario) {
			sc.Nodes[1].Roles = append(sc.Nodes[1].Roles, "storage")
			sc.Library.Roots = append(sc.Library.Roots, scenario.LibraryRoot{ContentType: "movie", Folder: "films", Node: "seedbox"})
		})
		s.on("films").stock(films[0])
		down := partition(t, s.w)
		expectScanned(t, s.scan(), "partial", succeeded, filmsOnSeedbox+"|seedbox|skipped|seedbox is offline")
		heal(s.w, down)
		eventually(t, "seedbox's unit runs on its return", func() bool {
			got := s.last()
			return got.trigger == "returned" && got.result != ""
		})
		got := s.last()
		if got.result != "succeeded" || !slices.Equal(got.units, []string{filmsOnSeedbox + "|seedbox|succeeded|"}) {
			t.Fatalf("the run on seedbox's return is %s with %q, want seedbox's unit succeeded", got.result, got.units)
		}
		s.expectWorks(films[0])
	})
}
