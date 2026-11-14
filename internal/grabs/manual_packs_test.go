package grabs

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// season is Season 1 of The Lantern Keepers: four episodes, of which a
// release named S01 covers every one and S01E02 the second.
type season struct {
	level works.Level
	items []works.Item
}

func newSeason() season {
	s := season{level: works.Level{ID: works.LevelID{UUID: storage.NewUUID()}, Label: "Season 1"}}
	for range 4 {
		s.items = append(s.items, works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, LevelID: s.level.ID, Monitored: true})
	}
	return s
}

func (s season) ids() []works.ItemID {
	out := make([]works.ItemID, len(s.items))
	for i, it := range s.items {
		out[i] = it.ID
	}
	return out
}

// judging is how a search of the season's episodes judges a release.
func (s season) judging() judging {
	return judging{
		expect: indexers.Expect{
			Claim: release.Claim{IDs: map[string]string{"tvdbid": "1001"}, Titles: []string{"The Lantern Keepers", "Lantern Keepers"}},
			Covers: func(n release.Name) []works.ItemID {
				switch marks(n) {
				case "S01":
					return s.ids()
				case "S01E02":
					return []works.ItemID{s.items[1].ID}
				}
				return nil
			},
		},
		names: episodic{}, cuts: release.BuiltIn(), scoring: noFormats,
	}
}

// manual is a Manual search of the season with a run for each episode,
// every run hearing ls, and the episodes' cells cs.
func (s season) manual(cs []files.Cell, ls ...listed) manualIn {
	in := manualOf(s.judging(), cs, ls...)
	in.tree = works.Tree{Summary: works.Summary{Label: "The Lantern Keepers"}, Levels: []works.Level{s.level}, Items: s.items}
	in.nouns = works.Description{Item: works.Noun{One: "Episode", Many: "Episodes"}, Items: works.ItemsMany}
	first := in.runs[0]
	in.runs = nil
	for _, it := range s.items {
		run := first
		run.ID, run.ItemID = SearchID{UUID: storage.NewUUID()}, it.ID
		in.runs = append(in.runs, run)
		in.heard[run.ID] = in.heard[first.ID]
	}
	delete(in.heard, first.ID)
	return in
}

func (s season) cell(i int, t targets.Target, st files.State) files.Cell {
	return files.Cell{Item: s.items[i], Edition: targets.AnyEdition, Target: t, State: st}
}

// TestAPackRowFillsTheEpisodesItBeatsAndNamesTheOnesThatKeepTheirFiles
// plans a season pack heard in every episode's run, where episode 4
// already has its file: one row fills episodes 1 to 3, says episode 4
// keeps its file, and All Releases lists the pack once.
func TestAPackRowFillsTheEpisodesItBeatsAndNamesTheOnesThatKeepTheirFiles(t *testing.T) {
	s := newSeason()
	met := s.cell(3, hd, files.Met)
	met.File = &files.File{Path: "Season 1/E04.mkv", Quality: release.Bluray1080p}
	cs := []files.Cell{s.cell(0, hd, files.Missing), s.cell(1, hd, files.Missing), s.cell(2, hd, files.Missing), met}
	pack := hashed(offered("Lantern.Keepers.S01.1080p.BluRay-GRP", 5, 10, nil), 'p')
	in := s.manual(cs, pack)
	v := viewOf(t, in)
	if len(v.Plan.Rows) != 1 || v.Plan.Rows[0].Pick == nil {
		t.Fatalf("the plan is %+v, want the pack's one row", v.Plan.Rows)
	}
	r := v.Plan.Rows[0]
	if want := []files.CellKey{keyOf(cs[0]), keyOf(cs[1]), keyOf(cs[2])}; !slices.Equal(r.Cells, want) {
		t.Errorf("the pack's row fills %v, want episodes 1 to 3", r.Cells)
	}
	if want := "It gets you 1080p for E01 to E03. E04 keeps its file."; r.Outcome != want {
		t.Errorf("the pack's row says %q, want %q", r.Outcome, want)
	}
	if v.Plan.Headline != "1 Grab gets you everything." {
		t.Errorf("the headline is %q, want 1 Grab gets you everything", v.Plan.Headline)
	}
	if offers := offersOf(t, in); len(offers) != 1 {
		t.Errorf("All Releases lists %d, want the pack once", len(offers))
	}
}

// TestAPickOfAPackFromTwoRunsIsOneGrabOfEveryItemItCovers ticks the pack
// as two runs offer it: the picks are one Release, and its Grab holds
// every episode the pack covers and the cells it fills across them.
func TestAPickOfAPackFromTwoRunsIsOneGrabOfEveryItemItCovers(t *testing.T) {
	s := newSeason()
	cs := []files.Cell{s.cell(0, hd, files.Missing), s.cell(1, hd, files.Missing), s.cell(2, hd, files.Missing), s.cell(3, hd, files.Missing)}
	pack := hashed(offered("Lantern.Keepers.S01.1080p.BluRay-GRP", 5, 10, nil), 'p')
	sh, err := worked(s.manual(cs, pack), nil)
	if err != nil {
		t.Fatal(err)
	}
	pickIn := func(run int, k ...int) chosen {
		p := ManualPick{Release: ReleaseRef{Run: sh.runs[run].run.ID, Position: 0}}
		for _, i := range k {
			p.Cells = append(p.Cells, keyOf(cs[i]))
		}
		c, e := sh.choose(p)
		if e != nil {
			t.Fatalf("the pick in run %d was refused: %+v", run, e)
		}
		return c
	}
	first, second := pickIn(0, 0, 1, 2, 3), pickIn(2, 2, 3)
	if !sh.same(first, second) {
		t.Fatal("the pack picked in two runs reads as two Releases, want one")
	}
	sh.join(&first, second, 1)
	items, cells := sh.runs[first.run].grabbed(first)
	if !slices.Equal(items, s.ids()) {
		t.Errorf("the Grab holds Items %v, want every episode", items)
	}
	var keys []files.CellKey
	for _, c := range cells {
		keys = append(keys, c.Key)
		if c.Forced {
			t.Errorf("cell %v is forced, want none", c.Key)
		}
	}
	if want := []files.CellKey{keyOf(cs[0]), keyOf(cs[1]), keyOf(cs[2]), keyOf(cs[3])}; !slices.Equal(keys, want) {
		t.Errorf("the Grab's cells are %v, want each episode's", keys)
	}
}

// TestAPickPastNotAnUpgradeForcesItsCell picks a Release no better than
// the file the Item has: its lift names that file, and the Grab's cell
// is forced, so its Import puts the file there.
func TestAPickPastNotAnUpgradeForcesItsCell(t *testing.T) {
	web := target("Web", release.Bluray1080p, nil, []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	upgradable := cell(web, files.Upgradable)
	upgradable.File = &files.File{Path: "Orchard (2021)/Orchard (2021) WEBDL-1080p.mkv", Quality: release.WEBDL1080p, Title: release.Title("Orchard.2021.1080p.WEB-DL-GRP", episodic{}, release.BuiltIn())}
	in := manualOf(orchard, []files.Cell{upgradable}, offered("Orchard.2021.1080p.WEB-DL-GRP2", 5, 10, nil))
	in.now = time.Now()
	v := viewOf(t, in)
	choices := v.Plan.Rows[0].Choices
	if len(choices) != 1 {
		t.Fatalf("Other choices are %+v, want the Release that isn't better", choices)
	}
	c := choices[0]
	if want := "It isn't better than Orchard (2021) WEBDL-1080p.mkv, the file at Web. Grabbing it replaces that file."; len(c.Lifts) != 1 || c.Lifts[0].Words != want {
		t.Fatalf("the choice gets past %+v, want %q", c.Lifts, want)
	}
	sh, err := worked(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	ch, e := sh.choose(ManualPick{Release: c.Release.Ref, Cells: c.Cells})
	if e != nil {
		t.Fatalf("the pick was refused: %+v", e)
	}
	if _, cells := sh.runs[ch.run].grabbed(ch); len(cells) != 1 || !cells[0].Forced {
		t.Fatalf("the Grab's cells are %+v, want its one cell forced", cells)
	}
}
