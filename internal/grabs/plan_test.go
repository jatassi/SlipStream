package grabs

import (
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	betterSite = indexers.ID{UUID: storage.NewUUID()}
	hiddenSite = indexers.ID{UUID: storage.NewUUID()}
)

// manualOf is a Manual search of The Quiet Orchard's one Item, judged as
// j, whose run Better Site answered with ls and whose Item has cells.
func manualOf(j judging, cs []files.Cell, ls ...listed) manualIn {
	run := Search{
		ID: SearchID{UUID: storage.NewUUID()}, ItemID: item.ID, Finished: time.Now(),
		Indexers: []SearchIndexer{{Indexer: betterSite, Name: "Better Site", Outcome: &indexers.Outcome{Kind: indexers.Found, Releases: len(ls)}}},
	}
	in := manualIn{
		tree: works.Tree{Summary: works.Summary{Label: "The Quiet Orchard (2021)"}, Items: []works.Item{item}}, runs: []Search{run},
		nouns: works.Description{Item: works.Noun{One: "Item", Many: "Items"}, Items: works.ItemsOne}, heard: map[SearchID][]heardRelease{},
		cells: cs, judging: j, blocks: blocks{}, routed: map[indexers.ID]string{}, now: time.Now(),
		standings: indexers.Standings{betterSite: {Indexer: betterSite, Automatic: true}, hiddenSite: {Indexer: hiddenSite}},
		line:      func(*indexers.Outcome) string { return "Found" },
	}
	for i, l := range ls {
		l.at = i
		if l.Release.Indexer.IsZero() {
			l.Release.Indexer = betterSite
		}
		name := "Better Site"
		if l.Release.Indexer == hiddenSite {
			name = "Hidden Site"
		}
		in.heard[run.ID] = append(in.heard[run.ID], heardRelease{listed: l, read: release.Title(l.Release.Title, j.names, j.cuts), indexer: name})
	}
	return in
}

func viewOf(t *testing.T, in manualIn, pins ...Pin) ManualView {
	t.Helper()
	v, err := manualView(in, pins)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func offersOf(t *testing.T, in manualIn) []Offer {
	t.Helper()
	s, err := worked(in, nil)
	if err != nil {
		t.Fatal(err)
	}
	return s.offers()
}

func hashed(l listed, c byte) listed {
	h := bencode.InfoHash([]byte(strings.Repeat(string(c), 20)))
	l.Release.InfoHash = &h
	return l
}

func TestEachRefusalIsGrabbableOrNotAsTheSpecsTableSays(t *testing.T) {
	notX265 := targets.Spec{Name: "No x265", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p, Scores: targets.Scores{Increment: 1}}
	if err := notX265.Filters.Set(targets.VideoCodec, "not", []string{"x265"}); err != nil {
		t.Fatal(err)
	}
	minimum := targets.Spec{Name: "Minimum", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p, Scores: targets.Scores{Minimum: 10, Increment: 1}}
	web := target("Web", release.Bluray1080p, nil, []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	upgradable := cell(web, files.Upgradable)
	upgradable.File = &files.File{Quality: release.WEBDL1080p, Title: release.Title("Orchard.2021.1080p.WEB-DL-GRP", episodic{}, release.BuiltIn())}
	open := []files.Cell{cell(hd, files.Missing)}
	long := lasting(orchard, 100*time.Minute)
	blocked := hashed(offered("Orchard.2021.1080p.BluRay-GRP", 5, 10, nil), 'b')
	for _, c := range []struct {
		code      targets.ReasonCode
		j         judging
		cells     []files.Cell
		l         listed
		grabbable bool
		// says is in the offer's lifts, its Refusal or its Blocked words.
		says string
	}{
		{targets.WrongWork, orchard, open, offered("Orchard.2021.1080p.WEB-DL", 5, 10, map[string]string{"tmdbid": "9000002"}), false, "An id in the name belongs to another Work."},
		{targets.WrongItem, lanternS01E03, open, offered("Lantern.Keepers.S01E02.1080p.WEB-DL", 5, 10, nil), false, "The Work has no Item S01E02."},
		{targets.QualityNotInTarget, orchard, open, offered("Orchard.2021.1080p.TS.x264", 5, 10, nil), false, "1080p doesn't accept"},
		{targets.FilterFailed, orchard, []files.Cell{cell(targetOf(notX265), files.Missing)}, offered("Orchard.2021.1080p.WEB-DL.x265", 5, 10, nil), false, "It fails the video codec filter of No x265."},
		{targets.CoversMore, lanternS01E03, open, offered("Lantern.Keepers.S01.1080p.WEB-DL", 5, 10, nil), false, "It's a pack of more than one item, and SlipStream can't import packs yet."},
		{targets.WrongTitle, orchard, open, offered("Orchard.2019.1080p.WEB-DL", 5, 10, nil), true, "Its name gives"},
		{targets.SizeBelow, long, open, offered("Orchard.2021.1080p.WEB-DL", 5, 10, nil), true, "It's smaller than 1080p allows."},
		{targets.SizeAbove, long, open, offered("Orchard.2021.1080p.WEB-DL", 5, 10000<<20, nil), true, "It's larger than 1080p allows."},
		{targets.ScoreBelowMinimum, orchard, []files.Cell{cell(targetOf(minimum), files.Missing)}, offered("Orchard.2021.1080p.WEB-DL", 5, 10, nil), true, "is below the minimum of 10 at Minimum."},
		{targets.RepackOtherGroup, orchard, []files.Cell{upgradable}, offered("Orchard.2021.1080p.WEB-DL.REPACK-OTHER", 5, 10, nil), true, "It's a repack from OTHER"},
		{targets.CutNotWanted, orchard, []files.Cell{cellOf(targets.Edition("extended"), hd, files.Missing)}, offered("Orchard.2021.1080p.WEB-DL", 5, 10, nil), true, "extended"},
		{targets.NotAnUpgrade, orchard, []files.Cell{upgradable}, offered("Orchard.2021.1080p.WEB-DL-GRP2", 5, 10, nil), false, "It isn't better than the file you have."},
		{targets.Blocklisted, orchard, open, blocked, true, "It's been on the Blocklist since"},
	} {
		in := manualOf(c.j, c.cells, c.l)
		if c.code == targets.Blocklisted {
			in.blocks = blocks{*blocked.Release.InfoHash: {Cause: CauseNoPayload, Added: time.Now()}}
		}
		offers := offersOf(t, in)
		if len(offers) != 1 {
			t.Fatalf("%s: All Releases lists %d, want 1", c.code, len(offers))
		}
		o := offers[0]
		said := slices.Concat([]string{o.Grab.Refusal, o.Release.Blocked}, wordsOfAll(o.Grab.Lifts))
		if c.code == targets.CutNotWanted {
			said = append(said, string(o.Grab.Edition))
		}
		if o.Grab.Grabbable != c.grabbable || !slices.ContainsFunc(said, func(s string) bool { return strings.Contains(s, c.says) }) {
			t.Errorf("%s: grabbable %v saying %q, want %v saying %q", c.code, o.Grab.Grabbable, said, c.grabbable, c.says)
		}
		v := viewOf(t, in)
		if len(v.Plan.Rows) != 1 || v.Plan.Rows[0].Pick != nil {
			t.Errorf("%s: the plan is %+v, want one cell nothing fills", c.code, v.Plan.Rows)
			continue
		}
		if offered := len(v.Plan.Rows[0].Choices) == 1; offered != c.grabbable {
			t.Errorf("%s: Other choices offers it %v, want %v", c.code, offered, c.grabbable)
		}
	}
}

func wordsOfAll(ws []Worded) []string {
	out := make([]string, len(ws))
	for i, w := range ws {
		out[i] = w.Words
	}
	return out
}

func TestACellNothingFillsSaysItsCause(t *testing.T) {
	bluray := offered("Orchard.2021.1080p.BluRay-GRP", 5, 10, nil)
	failed := cell(hd, files.Failed)
	for _, c := range []struct {
		name     string
		cells    []files.Cell
		ls       []listed
		in       func(*manualIn)
		missing  string
		cause    string
		headline string
	}{
		{
			name: "nothing listed for a Target", cells: []files.Cell{cell(hd, files.Missing), cell(small, files.Missing)}, ls: []listed{bluray},
			missing: "Missing: Small.", cause: "No Indexer listed a Small Release.",
			headline: "1 Grab gets you 1 of the 2 files you're missing. SlipStream hasn't found a Release for Small yet.",
		},
		{
			name: "every Release on the Blocklist", cells: []files.Cell{cell(hd, files.Missing)}, ls: []listed{hashed(bluray, 'b')},
			in: func(in *manualIn) {
				in.blocks = blocks{bencode.InfoHash([]byte(strings.Repeat("b", 20))): {Cause: CauseNoPayload, Added: time.Now()}}
			},
			missing: "Missing: 1080p.", cause: "Every 1080p Release is on the Blocklist.",
			headline: "SlipStream hasn't found a Release for 1080p yet.",
		},
		{
			name: "every Release needing an override", cells: []files.Cell{failed}, ls: []listed{offered("Orchard.2019.1080p.BluRay-GRP", 5, 10, nil)},
			missing: "Failed: 1080p.", cause: "Every 1080p Release needs an override. Other choices lists them.",
			headline: "SlipStream hasn't found a Release for 1080p yet.",
		},
		{
			name: "an Indexer at its grab limit", cells: []files.Cell{cell(hd, files.Missing)}, ls: []listed{bluray},
			in: func(in *manualIn) {
				in.standings[betterSite] = indexers.Standing{Indexer: betterSite, Automatic: true, AtGrabLimit: &indexers.NotTakenError{
					Outcome: indexers.Outcome{Kind: indexers.LimitReached}, Sentence: "Better Site is at its grab limit until 14:05.",
				}}
			},
			missing: "Missing: 1080p.", cause: "Better Site is at its grab limit until 14:05.",
			headline: "SlipStream hasn't found a Release for 1080p yet.",
		},
	} {
		in := manualOf(orchard, c.cells, c.ls...)
		if c.in != nil {
			c.in(&in)
		}
		v := viewOf(t, in)
		gap := v.Plan.Rows[len(v.Plan.Rows)-1]
		if gap.Pick != nil || gap.Missing != c.missing || gap.Cause != c.cause || v.Plan.Headline != c.headline {
			t.Errorf("%s: the plan says %q, its last row %q %q, want %q, %q %q", c.name, v.Plan.Headline, gap.Missing, gap.Cause, c.headline, c.missing, c.cause)
		}
	}
}

func TestTheReleasesSlipStreamWouldntGrabFoldIntoGroups(t *testing.T) {
	ls := []listed{
		offered("Orchard.2021.1080p.BluRay-GRP", 5, 10, nil),
		offered("Orchard.2021.1080p.TS.x264", 5, 10, nil),
		offered("Orchard.2021.1080p.TS.x264-OTHER", 5, 10, nil),
		offered("Orchard.2021.1080p.WEB-DL", 5, 10, map[string]string{"tmdbid": "9000002"}),
		hashed(offered("Orchard.2021.1080p.Remux-GRP", 5, 10, nil), 'b'),
		offered("Orchard.2019.1080p.BluRay-GRP", 5, 10, nil),
	}
	in := manualOf(orchard, []files.Cell{cell(hd, files.Missing)}, ls...)
	in.blocks = blocks{bencode.InfoHash([]byte(strings.Repeat("b", 20))): {Cause: CauseNoPayload, Added: time.Now()}}
	f := viewOf(t, in).Folded
	want := Folded{Line: "SlipStream wouldn't grab 4 more Releases.", Groups: []FoldGroup{
		{Words: "2 are CAM, which no Target here takes."},
		{Words: "1 names another Work."},
		{Words: "1 is on the Blocklist.", Fault: true},
	}}
	if !slices.Equal(f.Groups, want.Groups) || f.Line != want.Line {
		t.Fatalf("folded %+v\nwant %+v: the Release the plan takes and the one Other choices offers past its title aren't folded", f, want)
	}
}

func TestAPinSwapsItsRowAndADeadPinIsDropped(t *testing.T) {
	ls := []listed{
		offered("Orchard.2021.1080p.BluRay-GRP", 50, 10, nil),
		offered("Orchard.2021.1080p.WEB-DL-GRP", 5, 10, nil),
		offered("Orchard.2021.1080p.TS.x264", 5, 10, nil),
	}
	in := manualOf(orchard, []files.Cell{cell(hd, files.Missing), cell(small, files.Missing)}, ls...)
	run, hdCell := in.runs[0].ID, keyOf(cell(hd, files.Missing))
	v := viewOf(t, in)
	if r := v.Plan.Rows[0]; r.Pick == nil || r.Pick.Release.Title != ls[0].Release.Title || r.Pinned {
		t.Fatalf("before a pin the first row is %+v, want the BluRay", r)
	}
	choices := v.Plan.Rows[0].Choices
	if len(choices) != 1 || choices[0].Release.Title != ls[1].Release.Title {
		t.Fatalf("Other choices is %+v, want the WEB-DL alone", choices)
	}
	dead := Pin{Cell: hdCell, Release: ReleaseRef{Run: run, Position: 2}}
	v = viewOf(t, in, dead)
	if r := v.Plan.Rows[0]; r.Pick == nil || r.Pick.Release.Title != ls[0].Release.Title || len(v.Pins) != 0 {
		t.Fatalf("a pin of a Release 1080p doesn't take makes the first row %+v and keeps pins %+v, want it dropped", r, v.Pins)
	}
	v = viewOf(t, in, choices[0].Pin)
	if r := v.Plan.Rows[0]; r.Pick == nil || r.Pick.Release.Title != ls[1].Release.Title || !r.Pinned {
		t.Fatalf("after the pin the first row is %+v, want the WEB-DL pinned", r)
	}
	if !slices.Equal(v.Pins, []Pin{choices[0].Pin}) {
		t.Fatalf("the pins in effect are %+v, want the one sent", v.Pins)
	}
}

// TestThePlansFirstRowIsAutomaticSearchsTakeOverGeneratedSearches holds
// the plan to the decision. Over generated searches with no pin, the
// plan's first row is the Release automatic search's choose takes when
// that Release's first cell is the first cell the plan walks, and a cell
// nothing fills otherwise.
func TestThePlansFirstRowIsAutomaticSearchsTakeOverGeneratedSearches(t *testing.T) {
	reads := make([]release.Read, len(pool))
	for i, p := range pool {
		reads[i] = release.Title(p.title, orchard.names, orchard.cuts)
	}
	ids := []indexers.ID{{UUID: storage.NewUUID()}, {UUID: storage.NewUUID()}, {UUID: storage.NewUUID()}}
	rng := rand.New(rand.NewPCG(157, 32)) //nolint:gosec // seeded on purpose so a failure replays
	rows, gaps := 0, 0
	for n := range 2000 {
		g := generate(rng, reads, ids)
		if len(g.c.open) == 0 {
			continue
		}
		in := walkingAll(g)
		planned, _, err := plan(in)
		if err != nil {
			t.Fatal(err)
		}
		took, _, ok, err := choose(g.picks, nil, reading(g.sts))
		if err != nil {
			t.Fatal(err)
		}
		first := planned[0]
		if ok && took.Edition == in.item[0].Edition && took.Targets[0] == in.item[0].Target.ID {
			rows++
			if first.group < 0 || first.pick.at != took.at {
				t.Fatalf("case %d: the plan's first row is %+v, want %q at %d", n, first, took.Release.Title, took.at)
			}
			continue
		}
		gaps++
		if first.group >= 0 {
			t.Fatalf("case %d: the plan's first row takes %q, want a cell nothing fills: automatic search takes %q (%t)", n, first.pick.Release.Title, took.Release.Title, ok)
		}
	}
	if rows < 200 || gaps < 200 {
		t.Fatalf("the generator led with a Grab in %d cases and a gap in %d, want both often", rows, gaps)
	}
}

// walkingAll is the plan input of g walking every cell, with no pin.
func walkingAll(g generated) planIn {
	in := planIn{item: g.c.open, fill: make([]bool, len(g.c.open)), as: g.as, take: reading(g.sts)}
	for k := range in.item {
		in.walk, in.fill[k] = append(in.walk, k), true
	}
	for _, a := range g.as {
		j := judge(cells{open: in.item}, a)
		in.js, in.st = append(in.js, j), append(in.st, stances(in.item, j))
	}
	return in
}
