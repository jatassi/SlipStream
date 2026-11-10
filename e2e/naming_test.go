package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/world"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
)

// A file SlipStream doesn't track sits where the templates name The
// Lantern Keeper's Sized file, so its Grab fails at its probe with the
// path, before any bytes move, and the file stays as it was.
func TestAGrabWhoseNameAFileSlipStreamDoesntTrackHoldsFailsNameTaken(t *testing.T) {
	t.Parallel()
	w, _ := sized(t, "9000002", "Films", "Sized")
	const theirs = "movies/The Lantern Keeper (2019)/The Lantern Keeper (2019) - Sized.mkv"
	at := filepath.Join(w.m.MediaDir, filepath.FromSlash(theirs))
	if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(at, []byte("theirs"), 0o600); err != nil {
		t.Fatal(err)
	}

	found := w.searched(t, w.search(t))
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want a release grabbed", found)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "failed" || w.grab.Failure == nil || w.grab.Failure.Code != "name_taken" ||
		w.grab.Failure.Message != `a file SlipStream doesn't track is already at "The Lantern Keeper (2019)/The Lantern Keeper (2019) - Sized.mkv"` {
		t.Fatalf("the Grab ended %s with %+v after %v, want failed with its name taken", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	if got := w.grab.stages(); !slices.Equal(got[len(got)-2:], []string{"probing", "failed"}) {
		t.Fatalf("the Grab went through %v, want it to fail at its probe", got)
	}
	holdsOnly(t, w.m, theirs)
	if data, err := os.ReadFile(at); err != nil || string(data) != "theirs" {
		t.Fatalf("the file at the name holds %q, %v, want what it held before", data, err)
	}
}

const qualityFile = "{Movie Title} {(Year)} - {Edition Tag} {Target} {[Quality Full]}"

type planned struct {
	Files   int `json:"files"`
	Blocked int `json:"blocked"`
	Works   []struct {
		WorkID  string `json:"workId"`
		Title   string `json:"title"`
		Digest  string `json:"digest"`
		Rename  int    `json:"rename"`
		Blocked int    `json:"blocked"`
		Moving  int    `json:"moving"`
	} `json:"works"`
}

// lines is the plan as "<files> to rename, <blocked> blocked" then a line
// per Work: "<title>: rename <n>, blocked <n>, moving <n>".
func (p planned) lines() []string {
	lines := []string{fmt.Sprintf("%d to rename, %d blocked", p.Files, p.Blocked)}
	for _, w := range p.Works {
		lines = append(lines, fmt.Sprintf("%s: rename %d, blocked %d, moving %d", w.Title, w.Rename, w.Blocked, w.Moving))
	}
	return lines
}

// confirmed is the confirm body of every Work p lists, as shown.
func (p planned) confirmed() []confirmedWork {
	out := make([]confirmedWork, len(p.Works))
	for i, w := range p.Works {
		out[i] = confirmedWork{Work: w.WorkID, Digest: w.Digest}
	}
	return out
}

type workPlanned struct {
	Digest string `json:"digest"`
	Rows   []struct {
		FileID     string   `json:"fileId"`
		From       string   `json:"from"`
		To         string   `json:"to"`
		Group      int      `json:"group"`
		Companions []string `json:"companions"`
		Blocked    *struct {
			Reason string `json:"reason"`
			Detail string `json:"detail"`
		} `json:"blocked"`
	} `json:"rows"`
}

// lines is each row as "<from> -> <to>", then " with <companions>",
// " in a group" and " blocked <reason>: <detail>" where they apply.
func (p workPlanned) lines() []string {
	var lines []string
	for _, r := range p.Rows {
		line := r.From + " -> " + r.To
		if len(r.Companions) > 0 {
			line += " with " + strings.Join(r.Companions, ", ")
		}
		if r.Group != 0 {
			line += " in a group"
		}
		if r.Blocked != nil {
			line += " blocked " + r.Blocked.Reason + ": " + r.Blocked.Detail
		}
		lines = append(lines, line)
	}
	return lines
}

func (p workPlanned) confirmed(work string, leave ...string) confirmedWork {
	return confirmedWork{Work: work, Digest: p.Digest, Leave: leave}
}

type confirmedWork struct {
	Work   string   `json:"work"`
	Digest string   `json:"digest"`
	Leave  []string `json:"leave,omitempty"`
}

type renamedFiles struct {
	Queued  int      `json:"queued"`
	Changed []string `json:"changed"`
}

func planRenames(t *testing.T, a *worldtest.API, body string) planned {
	t.Helper()
	return *decode(t, "planRenames", a.Admin("home", http.MethodPost, "/api/v1/renames/plan", body), &planned{})
}

func planWork(t *testing.T, a *worldtest.API, work, body string) workPlanned {
	t.Helper()
	return *decode(t, "planWorkRenames", a.Admin("home", http.MethodPost, "/api/v1/renames/plan/"+work, body), &workPlanned{})
}

func renameFiles(t *testing.T, a *worldtest.API, ws ...confirmedWork) renamedFiles {
	t.Helper()
	body, err := json.Marshal(map[string][]confirmedWork{"works": ws})
	if err != nil {
		t.Fatal(err)
	}
	return *decode(t, "renameFiles", a.Admin("home", http.MethodPost, "/api/v1/renames", string(body)), &renamedFiles{})
}

func saveNaming(t *testing.T, a *worldtest.API, contentType, values string) {
	t.Helper()
	if r := a.Admin("home", http.MethodPatch, "/api/v1/naming/"+contentType, `{"values":`+values+`}`); r.Status != http.StatusOK {
		t.Fatalf("setNaming %s %s = %d %s, want 200", contentType, values, r.Status, r.Body)
	}
}

func expectRows(t *testing.T, when string, p workPlanned, want ...string) {
	t.Helper()
	if got := p.lines(); !slices.Equal(got, want) {
		t.Fatalf("%s, the Work's plan reads\n%s\nwant\n%s", when, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// writeFiles writes a small file at each path under dir, as an Admin
// copying files in would.
func writeFiles(t *testing.T, dir string, paths ...string) {
	t.Helper()
	for _, p := range paths {
		at := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(at), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(at, []byte(p), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// shelfHolds waits for the shelf folder in the media dir to hold exactly
// paths, and for each of gone, a folder in it, to be removed.
func shelfHolds(t *testing.T, m world.Manifest, folder string, gone []string, paths ...string) {
	t.Helper()
	dir := filepath.Join(m.MediaDir, folder)
	want := slices.Sorted(slices.Values(paths))
	var got []string
	worldtest.Until(t, folder+" holds exactly "+strings.Join(want, ", "), func() (bool, error) {
		got = slices.Sorted(slices.Values(filesUnder(t, dir)))
		return slices.Equal(got, want), nil
	})
	for _, g := range gone {
		worldtest.Until(t, "the emptied folder "+g+" is removed", func() (bool, error) {
			_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(g)))
			return errors.Is(err, fs.ErrNotExist), nil
		})
	}
}

// An Admin saves a new movie file template the way the Naming card does:
// the plan of the draft first, then the save, then the confirm with the
// draft plan's digests. A file SlipStream doesn't track lands at Copper
// Kettle's new path between the plan and the confirm, so that Work comes
// back changed and its next plan shows the row blocked; the other two
// rename on disk with their subtitles, the folder Lanterns at Low Tide
// leaves is removed, and Saltmarsh Radio's folder keeps its sample and
// featurette.
func TestSavingAFileTemplateThenConfirmingItsPlanRenamesTheFoundMoviesWithTheirCompanions(t *testing.T) {
	t.Parallel()
	sh := scannedShelves(t, "home-naming")
	_, works, _ := sh.library()
	stream := worldtest.NewMessages(t, sh.m, world.AdminIdentity)
	writeFiles(t, filepath.Join(sh.m.MediaDir, "shelf-movies"), "Saltmarsh Radio (2018)/Saltmarsh Radio (2018).en.srt", "Saltmarsh Radio (2018)/Saltmarsh Radio (2018).nfo")

	draft := `"draft":{"contentType":"movie","values":{"movie.naming.file.standard":"` + qualityFile + `"}}`
	drafted := planRenames(t, sh.a, `{"scope":{"kind":"type","id":"movie"},`+draft+`}`)
	if got, want := drafted.lines(), []string{
		"3 to rename, 0 blocked",
		"Copper Kettle: rename 1, blocked 0, moving 0",
		"Lanterns at Low Tide: rename 1, blocked 0, moving 0",
		"Saltmarsh Radio: rename 1, blocked 0, moving 0",
	}; !slices.Equal(got, want) {
		t.Fatalf("the plan of the draft reads\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	expectRows(t, "under the draft", planWork(t, sh.a, works["Saltmarsh Radio"], `{`+draft+`}`),
		"Saltmarsh Radio (2018)/Saltmarsh Radio (2018).mkv -> Saltmarsh Radio (2018)/Saltmarsh Radio (2018) - 1080p [HDTV-1080p].mkv with Saltmarsh Radio (2018)/Saltmarsh Radio (2018).en.srt, Saltmarsh Radio (2018)/Saltmarsh Radio (2018).nfo")

	saveNaming(t, sh.a, "movie", `{"movie.naming.file.standard":"`+qualityFile+`"}`)
	const theirs = "Copper Kettle (2012)/Copper Kettle (2012) - 1080p [HDTV-1080p].mkv"
	writeFiles(t, filepath.Join(sh.m.MediaDir, "shelf-movies"), theirs)
	done := renameFiles(t, sh.a, drafted.confirmed()...)
	if done.Queued != 2 || !slices.Equal(done.Changed, []string{works["Copper Kettle"]}) {
		t.Fatalf("the confirm answered %+v, want 2 queued and Copper Kettle changed", done)
	}

	shelfHolds(t, sh.m, "shelf-movies", []string{"Lanterns at Low Tide (2016) {tmdb-9000101}"},
		"Copper Kettle (2012).mkv",
		theirs,
		"Grandma's 80th Birthday (2019)/Grandma's 80th Birthday (2019).mkv",
		"Lanterns at Low Tide (2016)/Lanterns at Low Tide (2016) - 1080p [HDTV-1080p].mkv",
		"Nightfall/Nightfall.mkv",
		"Saltmarsh Radio (2018)/Featurettes/Making Saltmarsh Radio.mkv",
		"Saltmarsh Radio (2018)/Saltmarsh Radio (2018) - 1080p [HDTV-1080p].en.srt",
		"Saltmarsh Radio (2018)/Saltmarsh Radio (2018) - 1080p [HDTV-1080p].mkv",
		"Saltmarsh Radio (2018)/Saltmarsh Radio (2018) - 1080p [HDTV-1080p].nfo",
		"Saltmarsh Radio (2018)/Saltmarsh Radio (2018)-sample.mkv",
	)
	moved := map[string]bool{}
	for len(moved) < 2 {
		var m struct {
			WorkIDs []string `json:"workIds"`
		}
		if err := json.Unmarshal([]byte(stream.Next("files.moved")), &m); err != nil {
			t.Fatal(err)
		}
		for _, id := range m.WorkIDs {
			moved[id] = true
		}
	}
	if !maps.Equal(moved, map[string]bool{works["Lanterns at Low Tide"]: true, works["Saltmarsh Radio"]: true}) {
		t.Fatalf("files.moved named %v, want Lanterns at Low Tide and Saltmarsh Radio", moved)
	}
	sh.expectLibrary("after the rename",
		"Brass Meridian, added by admin, in shelf-series, files 1, gone 0",
		"  Brass Meridian (2015)/Brass Meridian - 2x05.mkv",
		"Copper Kettle, added by scan, in shelf-movies, files 1, gone 0",
		"  Copper Kettle (2012).mkv",
		"Lanterns at Low Tide, added by scan, in shelf-movies, files 1, gone 0",
		"  Lanterns at Low Tide (2016)/Lanterns at Low Tide (2016) - 1080p [HDTV-1080p].mkv",
		"Saltmarsh Radio, added by scan, in shelf-movies, files 1, gone 0",
		"  Saltmarsh Radio (2018)/Saltmarsh Radio (2018) - 1080p [HDTV-1080p].mkv",
		"The Tin Signal, added by admin, in shelf-series, files 3, gone 0",
		"  "+tinS01E01,
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E05.mkv",
		"  The Tin Signal [tvdbid-9100201]/Season 02/The Tin Signal - S02E06.mkv",
	)
	expectRows(t, "after the confirm", planWork(t, sh.a, works["Copper Kettle"], ""),
		"Copper Kettle (2012).mkv -> "+theirs+" blocked occupied: A file SlipStream doesn't track is at the new path. Move or delete that file, then plan again.")
	if got := planRenames(t, sh.a, `{"scope":{"kind":"type","id":"movie"}}`).lines(); !slices.Equal(got, []string{"0 to rename, 1 blocked", "Copper Kettle: rename 0, blocked 1, moving 0"}) {
		t.Fatalf("the movies' plan after the confirm reads %q, want Copper Kettle blocked alone", got)
	}
}

const (
	tinStatic  = "The Tin Signal (2019)/Season 01/The Tin Signal - S01E01 - Static 1080p.mkv"
	tinDeadAir = "The Tin Signal (2019)/Season 02/The Tin Signal - S02E05 - Dead Air 1080p.mkv"
	tinSignOff = "The Tin Signal (2019)/Season 02/The Tin Signal - S02E06 - Sign Off 1080p.mkv"
)

// brassMeridian is shelf-series' Brass Meridian folder, which no test
// here renames.
var brassMeridian = []string{"Brass Meridian (2015)/Brass Meridian - 2x05.mkv", "Brass Meridian (2015)/Brass Meridian - S03E09.mkv"}

// tinRenamed is home, as edit changes it, with its shelves scanned, the
// series naming values saved, and The Tin Signal's found episodes renamed
// as its plan says, to names. It answers the shelves and the Work's id.
func tinRenamed(t *testing.T, name string, edit func(*scenario.Scenario), values string, names ...string) (sh *shelves, tin string) {
	t.Helper()
	s, err := scenario.Load("home")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = name
	edit(&s)
	sh = scannedShelvesOf(t, s)
	saveNaming(t, sh.a, "series", values)
	_, works, _ := sh.library()
	tin = works["The Tin Signal"]
	found := planWork(t, sh.a, tin, "")
	if done := renameFiles(t, sh.a, found.confirmed(tin)); done.Queued != 3 {
		t.Fatalf("the confirm of The Tin Signal's found files answered %+v, want 3 queued", done)
	}
	shelfHolds(t, sh.m, "shelf-series", []string{"The Tin Signal [tvdbid-9100201]"}, append(slices.Clone(brassMeridian), names...)...)
	return sh, tin
}

// With episode titles left out of the file template, DVD order swaps The
// Tin Signal's episodes 5 and 6, so their files trade names: the plan
// joins the two rows, a confirm leaving one renames neither, and a
// confirm of both trades them on disk.
func TestAfterAnEpisodeOrderChangeTwoFilesTradeNamesTogether(t *testing.T) {
	t.Parallel()
	const e05, e06 = "The Tin Signal (2019)/Season 02/The Tin Signal - S02E05 - 1080p.mkv", "The Tin Signal (2019)/Season 02/The Tin Signal - S02E06 - 1080p.mkv"
	sh, tin := tinRenamed(t, "home-naming-order", func(s *scenario.Scenario) {
		s.TVDB.Series[0].DVD = map[int64]scenario.TVDBNumber{
			9120101: {Season: 1, Episode: 1}, 9120102: {Season: 1, Episode: 2},
			9120205: {Season: 2, Episode: 6}, 9120206: {Season: 2, Episode: 5},
		}
	}, `{"series.naming.file.standard":"{Series Title} - S{season:00}E{episode:00} - {Edition} {Target}"}`,
		"The Tin Signal (2019)/Season 01/The Tin Signal - S01E01 - 1080p.mkv", e05, e06)
	shelf := filepath.Join(sh.m.MediaDir, "shelf-series")
	stat := func(p string) (os.FileInfo, error) { return os.Stat(filepath.Join(shelf, filepath.FromSlash(p))) }
	identity := func(p string) os.FileInfo {
		info, err := stat(p)
		if err != nil {
			t.Fatal(err)
		}
		// On Windows a FileInfo reads its file's id from its path when it is
		// first compared, which after the trade is the other file.
		os.SameFile(info, info)
		return info
	}
	deadAir, signOff := identity(e05), identity(e06)
	holds := func(p string, was os.FileInfo) bool {
		now, err := stat(p)
		return err == nil && os.SameFile(was, now)
	}
	if r := sh.a.Admin("home", http.MethodPut, "/api/v1/works/"+tin+"/options", `{"type":{"order":"dvd"},"preview":false}`); r.Status != http.StatusOK {
		t.Fatalf("setWorkOptions to DVD order = %d %s, want 200", r.Status, r.Body)
	}

	traded := planWork(t, sh.a, tin, "")
	expectRows(t, "after the order change", traded, e05+" -> "+e06+" in a group", e06+" -> "+e05+" in a group")
	if done := renameFiles(t, sh.a, traded.confirmed(tin, traded.Rows[0].FileID)); done.Queued != 0 || len(done.Changed) != 0 {
		t.Fatalf("the confirm leaving Dead Air's file answered %+v, want nothing queued", done)
	}
	if done := renameFiles(t, sh.a, traded.confirmed(tin)); done.Queued != 2 || len(done.Changed) != 0 {
		t.Fatalf("the confirm of both answered %+v, want 2 queued", done)
	}
	worldtest.Until(t, "the two files traded names", func() (bool, error) {
		return holds(e06, deadAir) && holds(e05, signOff), nil
	})
	shelfHolds(t, sh.m, "shelf-series", nil, append(slices.Clone(brassMeridian), "The Tin Signal (2019)/Season 01/The Tin Signal - S01E01 - 1080p.mkv", e05, e06)...)
	expectRows(t, "after the trade", planWork(t, sh.a, tin, ""))
}

// Turning on Name by air date for The Tin Signal plans each episode's
// file under its air date, and the confirm names them so on disk.
func TestNameByAirDateNamesASeriesEpisodesByTheirAirDates(t *testing.T) {
	t.Parallel()
	sh, tin := tinRenamed(t, "home-naming-daily", func(*scenario.Scenario) {}, `{}`, tinStatic, tinDeadAir, tinSignOff)
	if r := sh.a.Admin("home", http.MethodPut, "/api/v1/works/"+tin+"/options", `{"type":{"nameByAirDate":true},"preview":false}`); r.Status != http.StatusOK {
		t.Fatalf("setWorkOptions naming by air date = %d %s, want 200", r.Status, r.Body)
	}

	daily := planWork(t, sh.a, tin, "")
	expectRows(t, "named by air date", daily,
		tinStatic+" -> The Tin Signal (2019)/Season 01/The Tin Signal - 2019-01-07 - Static 1080p.mkv",
		tinDeadAir+" -> The Tin Signal (2019)/Season 02/The Tin Signal - 2020-02-03 - Dead Air 1080p.mkv",
		tinSignOff+" -> The Tin Signal (2019)/Season 02/The Tin Signal - 2020-02-10 - Sign Off 1080p.mkv")
	if done := renameFiles(t, sh.a, daily.confirmed(tin)); done.Queued != 3 {
		t.Fatalf("the confirm answered %+v, want 3 queued", done)
	}
	shelfHolds(t, sh.m, "shelf-series", nil, append(slices.Clone(brassMeridian),
		"The Tin Signal (2019)/Season 01/The Tin Signal - 2019-01-07 - Static 1080p.mkv",
		"The Tin Signal (2019)/Season 02/The Tin Signal - 2020-02-03 - Dead Air 1080p.mkv",
		"The Tin Signal (2019)/Season 02/The Tin Signal - 2020-02-10 - Sign Off 1080p.mkv")...)
}

type renamesImpact struct {
	Impact struct {
		Renamed int `json:"renamed"`
		Renames []struct {
			Title string `json:"title"`
			Files []struct {
				From string `json:"from"`
				To   string `json:"to"`
			} `json:"files"`
		} `json:"renames"`
	} `json:"impact"`
}

// lines is "<renamed> renamed", then each listed rename as
// "<title>: <from> -> <to>".
func (r renamesImpact) lines() []string {
	lines := []string{fmt.Sprintf("%d renamed", r.Impact.Renamed)}
	for _, w := range r.Impact.Renames {
		for _, f := range w.Files {
			lines = append(lines, w.Title+": "+f.From+" -> "+f.To)
		}
	}
	return lines
}

// renameTarget renames the Target id to name, keeping the rest of it,
// with query on the PUT, and checks the Impact lists want.
func (w *walked) renameTarget(t *testing.T, id, name, query string, want ...string) {
	t.Helper()
	var held map[string]json.RawMessage
	decode(t, "getTarget", w.a.Admin("home", http.MethodGet, "/api/v1/targets/"+id, ""), &held)
	body, err := json.Marshal(map[string]any{
		"name": name, "tiers": held["tiers"], "cutoff": held["cutoff"], "filters": held["filters"], "sizes": held["sizes"],
		"propers": held["propers"], "scores": held["scores"],
	})
	if err != nil {
		t.Fatal(err)
	}
	got := decode(t, "PUT target "+query, w.a.Admin("home", http.MethodPut, "/api/v1/targets/"+id+query, string(body)), &renamesImpact{}).lines()
	if !slices.Equal(got, want) {
		t.Fatalf("renaming the Target to %s%s answered an Impact of\n%s\nwant\n%s", name, query, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// The Impact of renaming the 1080p Target lists the file named after it,
// from its path to its new one, and the save renames it on disk.
func TestATargetRenamesImpactListsTheFileItRenames(t *testing.T) {
	t.Parallel()
	w := grabbedTargets(t)
	w.holds(t, release)
	starter := w.named(t, "/api/v1/targets", "targets", "1080p")
	const renamed = "The Quiet Orchard (2021)/The Quiet Orchard (2021) - Full HD.mkv"
	listed := []string{"1 renamed", "The Quiet Orchard: The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv -> " + renamed}

	w.renameTarget(t, starter, "Full HD", "?preview=true", listed...)
	w.holds(t, release)
	w.renameTarget(t, starter, "Full HD", "", listed...)
	w.holdsRelease(t, "movies/"+renamed, release)
}

// With renaming off for movies the import keeps the downloaded file's
// name in the movie's folder, and renaming the Target it fills renames
// nothing.
func TestWithRenamingOffAnImportKeepsTheDownloadedNameAndATargetRenameRenamesNothing(t *testing.T) {
	t.Parallel()
	w := launchedTargets(t)
	saveNaming(t, w.a, "movie", `{"movie.naming.renameFiles":false}`)
	w.add(t)
	found := w.searched(t, w.search(t))
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want a release grabbed", found)
	}
	w.ended(t, *found.GrabID)
	if w.grab.Stage != "imported" {
		t.Fatalf("the Grab ended %s with %+v after %v, want imported", w.grab.Stage, w.grab.Failure, w.grab.stages())
	}
	kept := "movies/The Quiet Orchard (2021)/" + release + ".mkv"
	w.holdsRelease(t, kept, release)

	w.renameTarget(t, w.named(t, "/api/v1/targets", "targets", "1080p"), "Full HD", "", "0 renamed")
	w.holdsRelease(t, kept, release)
	if got := planRenames(t, w.a, `{"scope":{"kind":"library"}}`).lines(); !slices.Equal(got, []string{"0 to rename, 0 blocked"}) {
		t.Fatalf("the library's plan reads %q, want nothing to rename", got)
	}
}

// nas writes paths of at most 259 bytes, as a Windows Node without long
// paths does. The movie's title is long enough that its folder and file
// name pass that in any root, though each name fits in 255 bytes, so its
// Grab fails at its probe with both lengths and nothing reaches nas.
func TestAGrabWhoseNameIsTooLongForAWindowsNodeFailsNameTooLong(t *testing.T) {
	t.Parallel()
	const title = "The Quiet Orchard and the Long Road Home Through the Valley of Lanterns Where the Rivers Meet the Sea at the End of Summer"
	s, err := scenario.Load("seedbox-nas")
	if err != nil {
		t.Fatal(err)
	}
	s.Name = "seedbox-nas-long-name"
	s.TMDB.Movies[0].Title = title
	s.Releases[0].Name = strings.ReplaceAll(title, " ", ".") + ".2021.1080p.WEB-DL.H264-FKW"
	run := worldtest.LaunchScenario(t, s)
	w := walked{s: s, run: run, m: run.Manifest(), a: api(t, run)}
	w.add(t)

	found := w.searched(t, w.search(t))
	if found.Outcome == nil || *found.Outcome != "grabbed" || found.GrabID == nil {
		t.Fatalf("the search ended %+v, want a release grabbed", found)
	}
	w.ended(t, *found.GrabID)
	path := title + " (2021)/" + title + " (2021) - 1080p.mkv"
	limit := 259 - len(filepath.Join(w.m.MediaDir, "movies")) - 1
	want := fmt.Sprintf("the path in the root is %d bytes long and the limit there is %d bytes", len(path), limit)
	if w.grab.Stage != "failed" || w.grab.Failure == nil || w.grab.Failure.Code != "name_too_long" || w.grab.Failure.Message != want {
		t.Fatalf("the Grab ended %s with %+v after %v, want failed with %q", w.grab.Stage, w.grab.Failure, w.grab.stages(), want)
	}
	if got := w.grab.stages(); !slices.Equal(got[len(got)-2:], []string{"probing", "failed"}) {
		t.Fatalf("the Grab went through %v, want it to fail at its probe", got)
	}
	holdsOnly(t, w.m)
}
