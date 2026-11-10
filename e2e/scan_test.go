package e2e

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// shelves is home with its shelves added as roots and scanned once. The
// Tin Signal and Brass Meridian were in the library before the series
// root, so their folders join them: the first by its tvdb tag, the second
// by title and year.
type shelves struct {
	t     *testing.T
	a     *worldtest.API
	m     world.Manifest
	roots map[string]string
}

func scannedShelves(t *testing.T, name string) *shelves {
	t.Helper()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = name
	return scannedShelvesOf(t, s)
}

// scannedShelvesOf is scannedShelves of s, a variant of home.
func scannedShelvesOf(t *testing.T, s scenario.Scenario) *shelves {
	t.Helper()
	run := worldtest.LaunchScenario(t, s)
	sh := &shelves{t: t, a: api(t, run), m: run.Manifest()}
	for _, key := range []string{"9100201", "9100202"} {
		decode(t, "addWork "+key, sh.a.Admin("home", http.MethodPost, "/api/v1/works", `{"contentType":"series","key":"`+key+`"}`), &added{})
	}
	sh.roots = addShelves(t, sh.a, sh.m)
	sh.waitScanned(map[string]time.Time{})
	return sh
}

type rootScanned struct {
	ID   string `json:"id"`
	Scan struct {
		LastAt  *time.Time `json:"lastAt"`
		Summary string     `json:"summary"`
		Running *struct{}  `json:"running"`
	} `json:"scan"`
}

// scans is each shelf's library scan line, by folder.
func (sh *shelves) scans() map[string]rootScanned {
	sh.t.Helper()
	var list struct {
		Roots []rootScanned `json:"roots"`
	}
	decode(sh.t, "listRoots", sh.a.Admin("home", http.MethodGet, "/api/v1/roots", ""), &list)
	out := map[string]rootScanned{}
	for _, r := range list.Roots {
		if f := sh.folder(r.ID); f != r.ID {
			out[f] = r
		}
	}
	return out
}

// waitScanned waits until every shelf has a scan that finished after its
// time in before, and none runs.
func (sh *shelves) waitScanned(before map[string]time.Time) {
	sh.t.Helper()
	worldtest.Until(sh.t, "each shelf's library scan finished", func() (bool, error) {
		for folder, r := range sh.scans() {
			if r.Scan.Running != nil || r.Scan.LastAt == nil || !r.Scan.LastAt.After(before[folder]) {
				return false, nil
			}
		}
		return true, nil
	})
}

// scanNow is Scan now on the shelf at folder, once that scan has finished,
// and answers the root's scan summary.
func (sh *shelves) scanNow(folder string) string {
	sh.t.Helper()
	before := map[string]time.Time{folder: *sh.scans()[folder].Scan.LastAt}
	if r := sh.a.Admin("home", http.MethodPost, "/api/v1/roots/"+sh.roots[folder]+"/scan", ""); r.Status != http.StatusAccepted {
		sh.t.Fatalf("scanRoot %s = %d %s, want 202", folder, r.Status, r.Body)
	}
	sh.waitScanned(before)
	return sh.scans()[folder].Scan.Summary
}

func (sh *shelves) folder(rootID string) string {
	for f, id := range sh.roots {
		if id == rootID {
			return f
		}
	}
	return rootID
}

type shelvedFile struct {
	ID     string     `json:"id"`
	Path   string     `json:"path"`
	GoneAt *time.Time `json:"goneAt"`
}

// library is the Library as lines, a Work's row then each of its files:
// "Saltmarsh Radio, added by scan, in shelf-movies, files 1, gone 0" and
// "  Saltmarsh Radio (2018)/Saltmarsh Radio (2018).mkv". It answers the
// Works by label and the files by path too.
func (sh *shelves) library() (lines []string, works map[string]string, files map[string]shelvedFile) {
	sh.t.Helper()
	var library struct {
		Rows []struct {
			ID        string `json:"id"`
			Label     string `json:"label"`
			AddedBy   string `json:"addedBy"`
			RootID    string `json:"rootId"`
			Files     int    `json:"files"`
			GoneFiles int    `json:"goneFiles"`
		} `json:"rows"`
	}
	decode(sh.t, "getLibrary", sh.a.Admin("home", http.MethodGet, "/api/v1/library", ""), &library)
	works, files = map[string]string{}, map[string]shelvedFile{}
	for _, w := range library.Rows {
		works[w.Label] = w.ID
		lines = append(lines, fmt.Sprintf("%s, added by %s, in %s, files %d, gone %d", w.Label, w.AddedBy, sh.folder(w.RootID), w.Files, w.GoneFiles))
		var listed struct {
			Files []shelvedFile `json:"files"`
		}
		decode(sh.t, "listWorkFiles "+w.Label, sh.a.Admin("home", http.MethodGet, "/api/v1/works/"+w.ID+"/files", ""), &listed)
		for _, f := range listed.Files {
			files[f.Path] = f
			line := "  " + f.Path
			if f.GoneAt != nil {
				line += " (gone)"
			}
			lines = append(lines, line)
		}
	}
	return lines, works, files
}

func (sh *shelves) expectLibrary(when string, want ...string) {
	sh.t.Helper()
	if got, _, _ := sh.library(); !slices.Equal(got, want) {
		sh.t.Fatalf("%s, the Library reads\n%s\nwant\n%s", when, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

type pendingList struct {
	Leads []struct {
		Root struct {
			ID string `json:"id"`
		} `json:"root"`
		Added    int `json:"added"`
		NeedsYou int `json:"needsYou"`
	} `json:"leads"`
	Entries []struct {
		ID         string `json:"id"`
		Folder     string `json:"folder"`
		Suggestion *struct {
			Label     string  `json:"label"`
			Year      *int    `json:"year"`
			WorkID    *string `json:"workId"`
			Confident bool    `json:"confident"`
		} `json:"suggestion"`
		IgnoredAt *time.Time `json:"ignoredAt"`
		Files     []struct {
			Path   string `json:"path"`
			Reason struct {
				Code  string `json:"code"`
				Title string `json:"title"`
				Year  int    `json:"year"`
				Count int    `json:"count"`
				Names string `json:"names"`
			} `json:"reason"`
		} `json:"files"`
	} `json:"entries"`
}

// pending is Pending imports, or its Ignored view, as lines: each shelf's
// lead, then a line per file of each entry, "<folder> suggests <Work>:
// <path>: <reason>".
func (sh *shelves) pending(ignored bool) []string {
	sh.t.Helper()
	var l pendingList
	decode(sh.t, "listPendingImports", sh.a.Admin("home", http.MethodGet, "/api/v1/imports/pending?ignored="+strconv.FormatBool(ignored), ""), &l)
	var lines []string
	for _, ld := range l.Leads {
		if f := sh.folder(ld.Root.ID); f != ld.Root.ID {
			lines = append(lines, fmt.Sprintf("%s added %d, %d need you", f, ld.Added, ld.NeedsYou))
		}
	}
	for _, e := range l.Entries {
		suggests := "nothing"
		if s := e.Suggestion; s != nil {
			suggests = s.Label
			if s.Year != nil {
				suggests += " (" + strconv.Itoa(*s.Year) + ")"
			}
			if s.WorkID != nil {
				suggests += " in the library"
			}
			if s.Confident {
				suggests += ", confident"
			}
		}
		if (e.IgnoredAt != nil) != ignored {
			sh.t.Fatalf("Pending imports with ignored=%t list %s with ignoredAt %v", ignored, e.Folder, e.IgnoredAt)
		}
		for _, f := range e.Files {
			r := f.Reason
			reason := strings.Join(slices.DeleteFunc([]string{r.Code, r.Title, yearWord(r.Year), countWord(r.Count), r.Names}, func(s string) bool { return s == "" }), " ")
			lines = append(lines, e.Folder+" suggests "+suggests+": "+f.Path+": "+reason)
		}
	}
	return lines
}

func yearWord(y int) string {
	if y == 0 {
		return ""
	}
	return "(" + strconv.Itoa(y) + ")"
}

func countWord(n int) string {
	if n == 0 {
		return ""
	}
	return "x" + strconv.Itoa(n)
}

func (sh *shelves) expectPending(when string, ignored bool, want ...string) {
	sh.t.Helper()
	if got := sh.pending(ignored); !slices.Equal(got, want) {
		sh.t.Fatalf("%s, Pending imports (ignored=%t) read\n%s\nwant\n%s", when, ignored, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

const (
	lanternsFolder = "Lanterns at Low Tide (2016) {tmdb-9000101}"
	tinS01E01      = "The Tin Signal [tvdbid-9100201]/Season 01/The Tin Signal - S01E01.mkv"
)

var (
	firstScanLibrary = []string{
		"Brass Meridian, added by admin, in shelf-series, files 1, gone 0",
		"  Brass Meridian (2015)/Brass Meridian - 2x05.mkv",
		"Copper Kettle, added by scan, in shelf-movies, files 1, gone 0",
		"  Copper Kettle (2012).mkv",
		"Lanterns at Low Tide, added by scan, in shelf-movies, files 1, gone 0",
		"  " + lanternsFolder + "/Lanterns at Low Tide (2016).mkv",
		"Saltmarsh Radio, added by scan, in shelf-movies, files 1, gone 0",
		"  Saltmarsh Radio (2018)/Saltmarsh Radio (2018).mkv",
		"The Tin Signal, added by admin, in shelf-series, files 3, gone 0",
		"  " + tinS01E01,
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E05.mkv",
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E06.mkv",
	}
	needsYou = []string{
		"Grandma's 80th Birthday (2019) suggests nothing: Grandma's 80th Birthday (2019)/Grandma's 80th Birthday (2019).mkv: no_match Grandma's 80th Birthday (2019)",
		"Nightfall suggests Nightfall (1998): Nightfall/Nightfall.mkv: several Nightfall x2",
		"Brass Meridian (2015) suggests Brass Meridian (2015) in the library: Brass Meridian (2015)/Brass Meridian - S03E09.mkv: no_such_item S03E09",
	}
)

func TestAddingTheShelvesAsRootsJoinsAndAddsWhatTheScanIsSureOfAndListsTheRest(t *testing.T) {
	t.Parallel()
	sh := scannedShelves(t, "home-scan")
	sh.expectLibrary("after the first scans", firstScanLibrary...)
	sh.expectPending("after the first scans", false, append([]string{"shelf-movies added 3, 2 need you", "shelf-series added 0, 1 need you"}, needsYou...)...)
	sh.expectPending("after the first scans", true, "shelf-movies added 3, 2 need you", "shelf-series added 0, 1 need you")
	scans := sh.scans()
	if got := []string{scans["shelf-movies"].Scan.Summary, scans["shelf-series"].Scan.Summary}; !slices.Equal(got, []string{"It found 5 new files.", "It found 5 new files."}) {
		t.Fatalf("the shelves' scan lines read %q, want 5 new files on each", got)
	}
}

func TestALeaveFilesDeleteListsTheFolderUnderIgnoredAndScanNowLeavesItThere(t *testing.T) {
	t.Parallel()
	sh := scannedShelves(t, "home-scan-leave")
	_, works, _ := sh.library()
	r := sh.a.Admin("home", http.MethodPost, "/api/v1/works/delete", `{"workIds":["`+works["Lanterns at Low Tide"]+`"],"files":"leave"}`)
	if r.Status != http.StatusOK {
		t.Fatalf("deleteWorks Lanterns at Low Tide leaving its files = %d %s, want 200", r.Status, r.Body)
	}
	library := slices.DeleteFunc(slices.Clone(firstScanLibrary), func(l string) bool { return strings.Contains(l, "Lanterns at Low Tide") })
	left := "Lanterns at Low Tide (2016) {tmdb-9000101} suggests nothing: " + lanternsFolder + "/Lanterns at Low Tide (2016).mkv: left"
	sh.expectLibrary("after the Leave files delete", library...)
	sh.expectPending("after the Leave files delete", true, "shelf-movies added 3, 2 need you", "shelf-series added 0, 1 need you", left)

	if got := sh.scanNow("shelf-movies"); got != "Nothing changed." {
		t.Fatalf("Scan now of shelf-movies reads %q, want nothing changed", got)
	}
	sh.expectLibrary("after Scan now", library...)
	sh.expectPending("after Scan now", true, "shelf-movies added 0, 2 need you", "shelf-series added 0, 1 need you", left)
	sh.expectPending("after Scan now", false, append([]string{"shelf-movies added 0, 2 need you", "shelf-series added 0, 1 need you"}, needsYou...)...)
}

func TestARemovedFileReadsGoneAfterScanNowUntilForgotten(t *testing.T) {
	t.Parallel()
	sh := scannedShelves(t, "home-scan-gone")
	_, _, files := sh.library()
	if err := os.Remove(filepath.Join(sh.m.MediaDir, "shelf-series", filepath.FromSlash(tinS01E01))); err != nil {
		t.Fatal(err)
	}
	if got := sh.scanNow("shelf-series"); got != "It found 1 gone file." {
		t.Fatalf("Scan now of shelf-series reads %q, want 1 gone file", got)
	}
	tin := slices.Index(firstScanLibrary, "The Tin Signal, added by admin, in shelf-series, files 3, gone 0")
	others := firstScanLibrary[:tin]
	sh.expectLibrary("after Scan now", append(slices.Clone(others),
		"The Tin Signal, added by admin, in shelf-series, files 2, gone 1",
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E05.mkv",
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E06.mkv",
		"  "+tinS01E01+" (gone)",
	)...)

	worldtest.Expect(t, "forgetFiles", sh.a.Admin("home", http.MethodPost, "/api/v1/files/forget", `{"fileIds":["`+files[tinS01E01].ID+`"]}`), worldtest.Reply{Status: http.StatusNoContent})
	sh.expectLibrary("after Forget", append(slices.Clone(others),
		"The Tin Signal, added by admin, in shelf-series, files 2, gone 0",
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E05.mkv",
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E06.mkv",
	)...)
}
