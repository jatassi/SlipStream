package grabs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/torrent/engine"
	"github.com/jatassi/slipstream/internal/works"
)

func target(name string, cutoff release.Quality, sizes map[release.Quality]targets.Size, tiers ...[]release.Quality) targets.Target {
	return targetOf(targets.Spec{Scores: targets.Scores{Increment: 1}, Name: name, Tiers: tiers, Cutoff: cutoff, Sizes: sizes})
}

func targetOf(s targets.Spec) targets.Target {
	t, err := targets.NewTarget(s)
	if err != nil {
		panic(err)
	}
	t.ID = targets.TargetID{UUID: storage.NewUUID()}
	return t
}

var (
	hd = target("1080p", release.Bluray1080p, map[release.Quality]targets.Size{release.WEBDL1080p: {Min: 12.5, Max: new(40.0)}},
		[]release.Quality{release.Bluray1080p, release.Remux1080p}, []release.Quality{release.WEBDL1080p, release.WEBRip1080p, release.HDTV1080p})
	small = target("Small", release.WEBDL720p, nil, []release.Quality{release.WEBDL720p}, []release.Quality{release.HDTV720p, release.DVDQuality})
	item  = works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, Monitored: true}
)

func cell(t targets.Target, s files.State) files.Cell {
	return cellOf(targets.AnyEdition, t, s)
}

func cellOf(e targets.Edition, t targets.Target, s files.State) files.Cell {
	return files.Cell{Item: item, Edition: e, Target: t, State: s}
}

var noFormats = formats.Compile(nil, formats.DefaultTimings())

// orchard is how a search judges a release of The Quiet Orchard (2021).
var orchard = judging{
	searched: item.ID,
	expect: indexers.Expect{
		Claim: release.Claim{
			IDs:    map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"},
			Titles: []string{"The Quiet Orchard", "Orchard"}, Year: release.Known(2021), Slack: 1,
		},
		Covers: indexers.CoversOnly(item.ID),
	},
	names: episodic{}, cuts: release.BuiltIn(), scoring: noFormats,
}

// lanternS01E03 is how a search judges a release of S01E03 of The Lantern
// Keepers, TVDB series 1001.
var lanternS01E03 = judging{
	searched: item.ID,
	expect: indexers.Expect{
		Claim: release.Claim{IDs: map[string]string{"tvdbid": "1001"}, Titles: []string{"The Lantern Keepers", "Lantern Keepers"}},
		Covers: func(n release.Name) []works.ItemID {
			switch marks(n) {
			case "S01E02":
				return []works.ItemID{s01e02}
			case "S01":
				return []works.ItemID{s01e02, item.ID}
			case "S01E03":
				return []works.ItemID{item.ID}
			}
			return nil
		},
	},
	names: episodic{}, cuts: release.BuiltIn(), scoring: noFormats,
}

var s01e02 = works.ItemID{UUID: storage.NewUUID()}

// lasting is j where the searched Item runs d.
func lasting(j judging, d time.Duration) judging {
	j.runtime = map[works.ItemID]release.Fact[time.Duration]{item.ID: release.Known(d)}
	return j
}

func offered(title string, seeders int, size int64, ids map[string]string) listed {
	return listed{Release: indexers.Release{Release: searching.Release{Title: title, Seeders: seeders, Size: size, IDs: ids}}}
}

func assessedBy(j judging, ls ...listed) []assessed {
	out := make([]assessed, len(ls))
	for i, l := range ls {
		l.at = i
		out[i] = assess(j, l)
	}
	return out
}

// searched is what a search does with as at the open cells: its picks
// and its refusals.
func searched(open []files.Cell, as []assessed) ([]Pick, []refusal) {
	c := cells{open: open}
	d := decide(c, as)
	return d.picks(c), refusedIn(outcomes(d, c, -1))
}

// refusal is a Release a search refused and why.
type refusal struct {
	Title   string
	Reasons []Reason
}

func refusedIn(rows []row) []refusal {
	var out []refusal
	for _, r := range rows {
		if r.outcome == ReleaseRefused {
			out = append(out, refusal{r.release.Title, r.reasons})
		}
	}
	return out
}

// episodic reads a name's capitalised words up to its first other word as
// its title, a year among its words, and its S01E02, S01E03 or S01 as its
// one Mark. Its clean rule folds case.
type episodic struct{}

func (episodic) ReadName(p release.Prepared) release.Name {
	words := strings.Split(p.Title.String(), ".")
	var n release.Name
	end := slices.IndexFunc(words, func(w string) bool { return !capitalised(w) })
	if end < 0 {
		end = len(words)
	}
	if end > 0 {
		n.Titles = []string{strings.Join(words[:end], " ")}
	}
	for _, w := range words {
		if y, err := strconv.Atoi(w); err == nil && y > 1900 {
			n.Year = release.Known(y)
		}
		if w == "S01E02" || w == "S01E03" || w == "S01" {
			n.Marks = []release.Mark{{Words: w, Key: w}}
		}
	}
	return n
}

func (episodic) CleanTitle(title string) string { return strings.ToLower(title) }

func capitalised(w string) bool {
	r := []rune(w)
	return len(r) > 1 && unicode.IsUpper(r[0]) && !slices.ContainsFunc(r[1:], func(c rune) bool { return !unicode.IsLower(c) })
}

func notChecked(t targets.Target) Reason {
	return Reason{Reason: targets.Reason{Code: targets.SizeNotChecked}, Target: t.ID}
}

func TestDecideRanksByTargetThenTierThenSeedersSizeAndTitle(t *testing.T) {
	open := []files.Cell{cell(hd, files.Missing), cell(small, files.Missing)}
	ls := []listed{
		offered("Orchard.720p.WEB-DL.A", 90, 10, nil),
		offered("Orchard.576p.DVD.B", 99, 10, nil),
		offered("Orchard.1080p.WEB-DL.C", 5, 10, map[string]string{"tmdbid": "9000001"}),
		offered("Orchard.1080p.WEBRip.D", 5, 20, nil),
		offered("Orchard.1080p.HDTV.B", 5, 20, nil),
		offered("Orchard.1080p.BluRay.A", 1, 1, nil),
	}
	pick, refused := best(searched(open, assessedBy(orchard, ls...)))
	if want := (&Pick{Release: ls[5].Release, at: 5, Edition: targets.AnyEdition, Targets: []targets.TargetID{hd.ID}, Notes: []Reason{notChecked(hd)}}); !reflect.DeepEqual(pick, want) || refused != nil {
		t.Fatalf("decide = %+v, %v; want %+v", pick, refused, want)
	}
	pick, _ = best(searched(open, assessedBy(orchard, ls[:5]...)))
	if pick.Release.Title != "Orchard.1080p.HDTV.B" {
		t.Fatalf("decide among tied 1080p web picked %s, want the larger, then first by title, Orchard.1080p.HDTV.B", pick.Release.Title)
	}
	pick, _ = best(searched(open[1:], assessedBy(orchard, ls...)))
	if want := (&Pick{Release: ls[0].Release, Edition: targets.AnyEdition, Targets: []targets.TargetID{small.ID}, Notes: []Reason{notChecked(small)}}); !reflect.DeepEqual(pick, want) {
		t.Fatalf("decide for Small = %+v, want WEB-DL 720p over a DVD in a lower tier: %+v", pick, want)
	}
	both := target("Any", release.WEBDL1080p, nil, []release.Quality{release.WEBDL1080p, release.WEBDL720p})
	pick, _ = best(searched([]files.Cell{cell(small, files.Missing), cell(both, files.Missing)}, assessedBy(orchard, ls[0])))
	if want := []targets.TargetID{small.ID, both.ID}; !reflect.DeepEqual(pick.Targets, want) {
		t.Fatalf("decide for two Targets = %v, want both in Profile order", pick.Targets)
	}
}

func TestDecideRefusesWithEveryReasonPerTarget(t *testing.T) {
	as := assessedBy(orchard,
		offered("Orchard.1080p.WEB-DL.Other", 50, 10, map[string]string{"tmdbid": "9000002"}),
		offered("Orchard.WEB-DL", 50, 10, nil),
		offered("Orchard.2021", 50, 10, nil),
		offered("Orchard.1080p.TS.x264", 50, 10, nil),
	)
	big := assess(lasting(orchard, 100*time.Minute), offered("Orchard.1080p.WEB-DL.Big", 50, 10000*(1<<20), nil))
	big.at = len(as)
	as = append(as, big)
	pick, refused := best(searched([]files.Cell{cell(hd, files.Missing), cell(small, files.Failed)}, as))
	notIn := func(t targets.Target, q string) Reason {
		return Reason{Reason: targets.Reason{Code: targets.QualityNotInTarget, Value: q}, Target: t.ID}
	}
	want := []refusal{
		{"Orchard.1080p.WEB-DL.Other", []Reason{own(targets.WrongWork, "")}},
		{"Orchard.2021", []Reason{notIn(hd, ""), notIn(small, "")}},
		{"Orchard.1080p.TS.x264", []Reason{notIn(hd, "cam"), notIn(small, "cam")}},
		{"Orchard.1080p.WEB-DL.Big", []Reason{{Reason: targets.Reason{Code: targets.SizeAbove}, Target: hd.ID}, notIn(small, "webdl-1080p")}},
	}
	if want := (&Pick{Release: as[1].Release, at: 1, Edition: targets.AnyEdition, Targets: []targets.TargetID{small.ID}, Notes: []Reason{notChecked(small)}}); !reflect.DeepEqual(pick, want) {
		t.Fatalf("decide = %+v, want the source-only WEB-DL as 720p for Small: %+v", pick, want)
	}
	if !reflect.DeepEqual(refused, want) {
		t.Fatalf("refused = %+v\nwant %+v", refused, want)
	}
}

func TestDecideRefusesAnotherEpisodeAndAPack(t *testing.T) {
	ls := []listed{
		offered("Lantern.Keepers.S01E02.1080p.WEB-DL", 90, 10, nil),
		offered("Lantern.Keepers.S01.1080p.WEB-DL", 90, 10, nil),
		offered("Lantern.Keepers.S01E03.1080p.WEB-DL", 5, 10, map[string]string{"tvdbid": "1001"}),
		offered("Lantern.Keepers.S01E03.Extended.1080p.WEB-DL", 5, 10, map[string]string{"tvdbid": "1002"}),
	}
	pick, refused := best(searched([]files.Cell{cell(hd, files.Missing)}, assessedBy(lanternS01E03, ls...)))
	if pick == nil || pick.Release.Title != "Lantern.Keepers.S01E03.1080p.WEB-DL" {
		t.Fatalf("decide picked %+v, want the release of S01E03 alone over better-seeded ones of S01E02 and S01", pick)
	}
	want := []refusal{
		{"Lantern.Keepers.S01E02.1080p.WEB-DL", []Reason{own(targets.WrongItem, "S01E02")}},
		{"Lantern.Keepers.S01.1080p.WEB-DL", []Reason{own(targets.CoversMore, "S01"), notChecked(hd)}},
		{"Lantern.Keepers.S01E03.Extended.1080p.WEB-DL", []Reason{own(targets.WrongWork, "")}},
	}
	if !reflect.DeepEqual(refused, want) {
		t.Fatalf("refused = %+v\nwant another episode as wrong_item and another series as wrong_work, judged no further, and the pack as covers_more, judged at the cell: %+v", refused, want)
	}
}

func TestASoftRefusedReleaseIsStillScoredAndRanked(t *testing.T) {
	x265 := formats.Format{ID: formats.ID{1}, Name: "x265", Conditions: []formats.Condition{
		{Name: "x265", Kind: formats.ReleaseTitle, Fields: json.RawMessage(`{"value":"\\bx265\\b"}`)},
	}}
	scored := targetOf(targets.Spec{
		Name: "Scored", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
		Scores: targets.Scores{Formats: map[formats.ID]int{x265.ID: 50}, Increment: 1},
	})
	j := lanternS01E03
	j.scoring = formats.Compile([]formats.Format{x265}, formats.DefaultTimings())
	c := cells{open: []files.Cell{cell(scored, files.Missing)}}
	d := decide(c, assessedBy(j,
		offered("Lantern.Keepers.S01E03.1080p.WEB-DL", 5, 10, nil),
		offered("Lantern.Keepers.S01.1080p.WEB-DL.x265", 90, 10, nil),
		offered("Lantern.Watchers.S01E03.1080p.WEB-DL.x265", 50, 10, nil),
	))
	var ranked []string
	for _, r := range d.Ranked {
		var codes []targets.ReasonCode
		for _, reason := range r.refusals {
			codes = append(codes, reason.Code)
		}
		ranked = append(ranked, fmt.Sprintf("%s %d %v", r.Release.Title, r.verdicts[0].v.Score(), codes))
	}
	want := []string{
		"Lantern.Keepers.S01.1080p.WEB-DL.x265 50 [covers_more]",
		"Lantern.Watchers.S01E03.1080p.WEB-DL.x265 50 [wrong_title]",
		"Lantern.Keepers.S01E03.1080p.WEB-DL 0 []",
	}
	if !slices.Equal(ranked, want) {
		t.Fatalf("ranked %q\nwant %q: a pack and another title, softly refused, still scored at the Target and ranked", ranked, want)
	}
	if picks := d.picks(c); len(picks) != 1 || picks[0].Release.Title != "Lantern.Keepers.S01E03.1080p.WEB-DL" {
		t.Fatalf("an automatic search may take %+v, want only the release with no refusal of its own", picks)
	}
}

func TestDecideRefusesForEachReason(t *testing.T) {
	notX265 := targets.Spec{Name: "No x265", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p, Scores: targets.Scores{Increment: 1}}
	if err := notX265.Filters.Set(targets.VideoCodec, "not", []string{"x265"}); err != nil {
		t.Fatal(err)
	}
	minimum := targets.Spec{Name: "Minimum", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p, Scores: targets.Scores{Minimum: 10, Increment: 1}}
	extended := []files.Cell{cellOf(targets.Edition("extended"), hd, files.Missing)}
	open := []files.Cell{cell(hd, files.Missing)}
	long := lasting(orchard, 100*time.Minute)
	for _, c := range []struct {
		j     judging
		open  []files.Cell
		title string
		size  int64
		ids   map[string]string
		want  []targets.ReasonCode
	}{
		{orchard, open, "Orchard.2021.1080p.WEB-DL", 10, map[string]string{"tmdbid": "9000002"}, []targets.ReasonCode{targets.WrongWork}},
		{orchard, open, "Another.Film.2021.1080p.WEB-DL", 10, nil, []targets.ReasonCode{targets.WrongTitle, targets.SizeNotChecked}},
		{orchard, open, "Orchard.2019.1080p.WEB-DL", 10, nil, []targets.ReasonCode{targets.WrongTitle, targets.SizeNotChecked}},
		{lanternS01E03, open, "Lantern.Keepers.S01E02.1080p.WEB-DL", 10, nil, []targets.ReasonCode{targets.WrongItem}},
		{lanternS01E03, open, "Lantern.Keepers.S01.1080p.WEB-DL", 10, nil, []targets.ReasonCode{targets.CoversMore, targets.SizeNotChecked}},
		{orchard, extended, "Orchard.2021.1080p.WEB-DL", 10, nil, []targets.ReasonCode{targets.CutNotWanted, targets.SizeNotChecked}},
		{orchard, open, "Orchard.2021.1080p.TS.x264", 10, nil, []targets.ReasonCode{targets.QualityNotInTarget}},
		{orchard, []files.Cell{cell(targetOf(notX265), files.Missing)}, "Orchard.2021.1080p.WEB-DL.x265", 10, nil, []targets.ReasonCode{targets.FilterFailed, targets.SizeNotChecked}},
		{long, open, "Orchard.2021.1080p.WEB-DL", 10, nil, []targets.ReasonCode{targets.SizeBelow}},
		{long, open, "Orchard.2021.1080p.WEB-DL", 10000 << 20, nil, []targets.ReasonCode{targets.SizeAbove}},
		{orchard, []files.Cell{cell(targetOf(minimum), files.Missing)}, "Orchard.2021.1080p.WEB-DL", 10, nil, []targets.ReasonCode{targets.SizeNotChecked, targets.ScoreBelowMinimum}},
	} {
		_, refused := searched(c.open, assessedBy(c.j, offered(c.title, 5, c.size, c.ids)))
		var got []targets.ReasonCode
		for _, r := range refused {
			for _, reason := range r.Reasons {
				got = append(got, reason.Code)
			}
		}
		if len(refused) != 1 || !slices.Equal(got, c.want) {
			t.Errorf("%s is refused as %v, want %v", c.title, got, c.want)
		}
	}
}

func TestDecideTakesAProperOnAFilledCell(t *testing.T) {
	web := target("Web", release.WEBDL1080p, nil, []release.Quality{release.WEBDL1080p}, []release.Quality{release.WEBDL720p})
	filled := cell(web, files.Met)
	filled.File = &files.File{Quality: release.WEBDL1080p, Title: release.Title("Orchard.2021.1080p.WEB-DL-GRP", episodic{}, release.BuiltIn())}
	c := cells{filled: []files.Cell{filled}}
	ls := []listed{
		offered("Orchard.2021.1080p.WEB-DL-GRP2", 90, 10, nil),
		offered("Orchard.2021.720p.WEB-DL.PROPER-GRP", 90, 10, nil),
		offered("Orchard.2021.1080p.WEB-DL.REPACK-OTHER", 90, 10, nil),
		offered("Orchard.2021.1080p.WEB-DL.PROPER-GRP", 5, 10, nil),
	}
	d := decide(c, assessedBy(orchard, ls...))
	if want := []Pick{{Release: ls[3].Release, at: 3, Edition: targets.AnyEdition, Targets: []targets.TargetID{web.ID}, Notes: []Reason{notChecked(web)}}}; !reflect.DeepEqual(d.picks(c), want) {
		t.Fatalf("decide over a filled cell picks %+v, want only the proper of its file: %+v", d.picks(c), want)
	}
	want := []refusal{
		{"Orchard.2021.1080p.WEB-DL-GRP2", nil},
		{"Orchard.2021.720p.WEB-DL.PROPER-GRP", nil},
		{"Orchard.2021.1080p.WEB-DL.REPACK-OTHER", []Reason{notChecked(web), {Reason: targets.Reason{Code: targets.RepackOtherGroup, Value: "GRP"}, Target: web.ID}}},
	}
	if got := refusedIn(outcomes(d, c, -1)); !reflect.DeepEqual(got, want) {
		t.Fatalf("refused = %+v\nwant the same revision and another Quality's proper judged at no cell, and a repack from another group refused: %+v", got, want)
	}
}

func TestDecideAsksARankAboveTheFileFillingTheCell(t *testing.T) {
	held := cell(hd, files.Upgradable)
	held.File = &files.File{Quality: release.WEBDL1080p}
	pick, refused := best(searched([]files.Cell{held}, assessedBy(orchard, offered("Orchard.1080p.WEBRip", 90, 10, nil), offered("Orchard.1080p.BluRay", 1, 10, nil))))
	want := []refusal{{"Orchard.1080p.WEBRip", []Reason{notChecked(hd), {Reason: targets.Reason{Code: targets.NotAnUpgrade}, Target: hd.ID}}}}
	if pick == nil || pick.Release.Title != "Orchard.1080p.BluRay" || !reflect.DeepEqual(refused, want) {
		t.Fatalf("decide over a held WEB-DL = %+v, %+v; want the Bluray and the WEBRip refused as %+v", pick, refused, want)
	}
}

func TestDecideReadsTheHeldFilesScoreAgainstUpgradeUntil(t *testing.T) {
	as := assessedBy(orchard, offered("Orchard.1080p.WEBRip", 90, 10, nil))
	for _, c := range []struct {
		score int
		grabs bool
	}{{-10, true}, {0, false}} {
		held := cell(hd, files.Upgradable)
		held.File, held.Score = &files.File{Quality: release.WEBDL1080p}, c.score
		pick, _ := best(searched([]files.Cell{held}, as))
		if (pick != nil) != c.grabs {
			t.Errorf("decide over a WEB-DL scoring %d at upgrade-until 0 picked %+v, want a pick %t: a same-tier Release scoring 0 upgrades only a file below upgrade-until", c.score, pick, c.grabs)
		}
	}
}

func TestDecideJudgesACandidateAtEachCellAndRefusesTheCutsItDoesntTake(t *testing.T) {
	standard, extended := targets.EditionOf(release.Standard), targets.Edition("extended")
	ls := []listed{offered("Orchard.1080p.BluRay", 90, 10, nil), offered("Orchard.EXTENDED.1080p.WEB-DL", 5, 10, nil)}
	pick, refused := best(searched([]files.Cell{cellOf(standard, hd, files.Upgradable), cellOf(extended, hd, files.Missing)}, assessedBy(orchard, ls...)))
	if want := (&Pick{Release: ls[0].Release, at: 0, Edition: standard, Targets: []targets.TargetID{hd.ID}, Notes: []Reason{notChecked(hd)}}); !reflect.DeepEqual(pick, want) || refused != nil {
		t.Fatalf("decide over Standard then Extended = %+v, %+v; want the Standard Bluray for Standard's cell: %+v", pick, refused, want)
	}
	pick, refused = best(searched([]files.Cell{cellOf(extended, hd, files.Missing)}, assessedBy(orchard, ls...)))
	want := []refusal{{"Orchard.1080p.BluRay", []Reason{{Reason: targets.Reason{Code: targets.CutNotWanted, Value: "standard"}, Target: hd.ID}, notChecked(hd)}}}
	if pick == nil || pick.Release.Title != "Orchard.EXTENDED.1080p.WEB-DL" || pick.Edition != extended || !reflect.DeepEqual(refused, want) {
		t.Fatalf("decide over Extended alone = %+v, %+v; want the Extended WEB-DL and the Standard Bluray refused at the cell as %+v", pick, refused, want)
	}
}

func TestDecideRanksTheBetterIndexerPriorityAfterTierAndBeforeSeeders(t *testing.T) {
	open := []files.Cell{cell(hd, files.Missing)}
	worse := offered("Orchard.1080p.WEB-DL.A", 90, 20, nil)
	worse.Release.Priority = 25
	better := offered("Orchard.1080p.WEBRip.B", 1, 20, nil)
	better.Release.Priority = 10
	bluray := offered("Orchard.1080p.BluRay.C", 1, 20, nil)
	bluray.Release.Priority = 50
	picks, _ := searched(open, assessedBy(orchard, worse, better))
	if len(picks) != 2 || picks[0].Release.Title != better.Release.Title || picks[1].Release.Title != worse.Release.Title {
		t.Fatalf("decide ranked %+v, want the priority 10 Release over the better-seeded priority 25 one in one tier", picks)
	}
	if pick, _ := best(searched(open, assessedBy(orchard, worse, better, bluray))); pick.Release.Title != bluray.Release.Title {
		t.Fatalf("decide picked %s, want %s from a better tier whatever its Indexer's priority", pick.Release.Title, bluray.Release.Title)
	}
}

func TestAssessSumsTheRuntimeOfEveryCoveredItem(t *testing.T) {
	j := lanternS01E03
	j.runtime = map[works.ItemID]release.Fact[time.Duration]{item.ID: release.Known(40 * time.Minute), s01e02: release.Known(45 * time.Minute)}
	for title, want := range map[string]release.Fact[time.Duration]{
		"Lantern.Keepers.S01.1080p.WEB-DL":    release.Known(85 * time.Minute),
		"Lantern.Keepers.S01E03.1080p.WEB-DL": release.Known(40 * time.Minute),
		"Lantern.Keepers.1080p.WEB-DL":        {},
	} {
		if got := assess(j, offered(title, 5, 10, nil)).read.Facts.Coverage.Runtime; got != want {
			t.Errorf("%s runs %v, want %v", title, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	live, canceled := context.Background(), func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}()
	for _, c := range []struct {
		name string
		ctx  context.Context
		err  error
		want outcome
	}{
		{"link down", live, link.ErrDisconnected, wait{cause: link.ErrDisconnected}},
		{"versions differ", live, link.ErrMismatched, wait{cause: link.ErrMismatched}},
		{"call lost", live, rpc.Errorf(rpc.CodeUnavailable, "the link died"), wait{cause: rpc.Errorf(rpc.CodeUnavailable, "the link died")}},
		{"torrent not re-added", live, downloading.ErrNotPresent, wait{cause: downloading.ErrNotPresent}},
		{"refused by ffprobe", live, rpc.Errorf(rpc.CodeFailedPrecondition, "ffprobe: exit status 1"), fail{ProbeFailed, "ffprobe: exit status 1"}},
		{"refused by the Indexer", live, &indexers.UnavailableError{Err: rpc.Errorf(rpc.CodeAborted, "the Indexer responded with status 500")}, fail{ProbeFailed, "The Indexer responded with status 500."}},
		{"stopping", canceled, link.ErrDisconnected, stopped{}},
	} {
		got := classify(c.ctx, c.err, ProbeFailed)
		if w, ok := got.(wait); ok {
			got = wait{cause: errors.New(w.cause.Error())}
			if want, ok := c.want.(wait); ok {
				c.want = wait{cause: errors.New(want.cause.Error())}
			}
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: classify = %#v, want %#v", c.name, got, c.want)
		}
	}
}

func TestAJobWaitingOnTheRosterDoesNotWakeTheReconciler(t *testing.T) {
	now := time.Now()
	waiting, due := storage.NewUUID(), storage.NewUUID()
	s := schedule{inflight: map[storage.UUID]bool{waiting: true}, retryAt: map[storage.UUID]time.Time{waiting: now.Add(-time.Second)}}
	if got := s.next(now, nil, false, DefaultTimings()); got != time.Hour {
		t.Fatalf("with only a running job's past retry, the reconciler sleeps %s, want an hour", got)
	}
	s.retryAt[due] = now.Add(time.Minute)
	if got := s.next(now, nil, false, DefaultTimings()); got != time.Minute {
		t.Fatalf("with a job not running due in a minute, the reconciler sleeps %s, want a minute", got)
	}
}

// best is decide's first pick, nil when it picks none.
func best(picks []Pick, refused []refusal) (*Pick, []refusal) {
	if len(picks) == 0 {
		return nil, refused
	}
	return &picks[0], refused
}

func from(priority int, title, hash string, size int64, at int) listed {
	c := offered(title, 5, size, nil)
	c.Release.Priority, c.at = priority, at
	if hash != "" {
		h := bencode.InfoHash([]byte(strings.Repeat(hash, 20)))
		c.Release.InfoHash = &h
	}
	return c
}

func TestDedupeCountsAReleaseListedTwiceOnceFromTheBetterPriority(t *testing.T) {
	kept := dedupe([]listed{
		from(25, "Orchard.1080p.WEB-DL", "a", 10, 0),
		from(25, "Orchard.720p.WEB-DL", "", 5, 1),
		from(10, "Orchard.1080p.WEB-DL.Mirror", "a", 10, 2),
		from(10, "Orchard.720p.WEB-DL", "b", 5, 3),
		from(10, "Orchard.2160p.WEB-DL", "c", 30, 4),
		from(25, "Orchard.2160p.WEB-DL", "d", 30, 5),
	})
	var got []string
	for _, c := range kept {
		got = append(got, fmt.Sprintf("%d %d %s", c.at, c.Release.Priority, c.Release.Title))
	}
	want := []string{"2 10 Orchard.1080p.WEB-DL.Mirror", "3 10 Orchard.720p.WEB-DL", "4 10 Orchard.2160p.WEB-DL", "5 25 Orchard.2160p.WEB-DL"}
	if !slices.Equal(got, want) {
		t.Fatalf("dedupe kept %q, want %q: one hash, or one title and size where a copy lacks a hash, from the better priority", got, want)
	}
}

func TestAJobWhoseGrabVanishedLeavesTheReconcilerIdle(t *testing.T) {
	now := time.Now()
	vanished, open := storage.NewUUID(), storage.NewUUID()
	s := schedule{
		inflight: map[storage.UUID]bool{}, attempts: map[storage.UUID]int{vanished: 3, open: 1},
		retryAt: map[storage.UUID]time.Time{vanished: now.Add(-time.Second), open: now.Add(time.Minute)},
		shown:   map[GrabID]Progress{{UUID: vanished}: {Done: 1, Total: 2}},
	}
	s.keep(nil, []Grab{{ID: GrabID{UUID: open}}})
	if got := s.next(now, nil, false, DefaultTimings()); got != time.Minute {
		t.Fatalf("after a waiting Grab vanished with its Work, the reconciler sleeps %s, want a minute until the open one's retry", got)
	}
	if _, ok := s.attempts[vanished]; ok {
		t.Errorf("the vanished Grab keeps %d attempts, want none", s.attempts[vanished])
	}
	if len(s.shown) != 0 {
		t.Errorf("the vanished Grab's progress stays shown: %v", s.shown)
	}
}

func TestProbingADownloadOfOnlyExtrasFailsAsNoVideoFile(t *testing.T) {
	g := Grab{Torrent: &Torrent{Attached: downloadclients.Attached{Files: []engine.File{
		{Index: 0, Path: "Heat.1995/Sample/heat.mkv", Size: 50},
		{Index: 1, Path: "Heat.1995/Heat.1995-trailer.mp4", Size: 80},
		{Index: 2, Path: "Heat.1995/Heat.1995.nfo", Size: 1},
	}}}}
	if got, want := (&Grabs{}).probe(t.Context(), g), (fail{NoVideoFile, "The download holds no video file."}); got != want {
		t.Fatalf("probe = %+v, want %+v", got, want)
	}
}

func TestLargestVideoPassesOverALargerSample(t *testing.T) {
	got, ok := largestVideo([]engine.File{
		{Index: 0, Path: "Heat.1995/Heat.1995-sample.mkv", Size: 900},
		{Index: 1, Path: "Heat.1995/Heat.1995.m2ts", Size: 700},
		{Index: 2, Path: "Heat.1995/Heat.1995.Part.2.mkv", Size: 300},
	})
	if !ok || got.Index != 1 {
		t.Fatalf("largestVideo = %+v, %t; want Heat.1995.m2ts", got, ok)
	}
}

func TestDecideRanksByEachStepOfTheChainAndSaysWhereTheOtherLost(t *testing.T) {
	x265 := formats.Format{ID: formats.ID{1}, Name: "x265", Conditions: []formats.Condition{
		{Name: "x265", Kind: formats.ReleaseTitle, Fields: json.RawMessage(`{"value":"\\bx265\\b"}`)},
	}}
	scoring := orchard
	scoring.scoring = formats.Compile([]formats.Format{x265}, formats.DefaultTimings())
	scored := targetOf(targets.Spec{
		Name: "Scored", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
		Scores: targets.Scores{Formats: map[formats.ID]int{x265.ID: 50}, Increment: 1},
	})
	unpreferred := targetOf(targets.Spec{
		Name: "Unpreferred", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p,
		Scores: targets.Scores{Increment: 1}, Propers: targets.DontPrefer,
	})
	with := func(l listed, edit func(*indexers.Release)) listed {
		edit(&l.Release)
		return l
	}
	web := offered("Orchard.1080p.WEB-DL", 5, 10, nil)
	for _, c := range []struct {
		name          string
		j             judging
		open          []files.Cell
		better, worse listed
		want          Loss
	}{
		{"want", orchard, []files.Cell{cell(hd, files.Missing), cell(small, files.Missing)},
			web, offered("Orchard.720p.WEB-DL", 99, 10, nil), Loss{Step: StepWant, Own: "Small", Ref: "1080p"}},
		{"tier", orchard, []files.Cell{cell(hd, files.Missing)},
			offered("Orchard.1080p.BluRay", 5, 10, nil), offered("Orchard.1080p.WEB-DL", 99, 10, nil), Loss{Step: StepTier, Target: hd.ID, Own: "WEBDL-1080p", Ref: "Bluray-1080p"}},
		{"revision", orchard, []files.Cell{cell(hd, files.Missing)},
			offered("Orchard.1080p.WEB-DL.PROPER", 5, 10, nil), offered("Orchard.1080p.WEB-DL", 99, 10, nil), Loss{Step: StepRevision, Target: hd.ID, Own: "v1", Ref: "v2"}},
		{"revision under Don't prefer", orchard, []files.Cell{cell(unpreferred, files.Missing)},
			offered("Orchard.1080p.WEB-DL", 99, 10, nil), offered("Orchard.1080p.WEB-DL.PROPER", 5, 10, nil), Loss{Step: StepSeeders, Own: "5", Ref: "99"}},
		{"score", scoring, []files.Cell{cell(scored, files.Missing)},
			offered("Orchard.1080p.WEB-DL.x265", 5, 10, nil), offered("Orchard.1080p.WEB-DL.x264", 99, 10, nil), Loss{Step: StepScore, Target: scored.ID, Own: "0", Ref: "50"}},
		{"priority", orchard, []files.Cell{cell(hd, files.Missing)},
			with(web, func(r *indexers.Release) { r.Priority = 10 }), with(web, func(r *indexers.Release) { r.Priority, r.Seeders = 25, 99 }), Loss{Step: StepPriority, Own: "25", Ref: "10"}},
		{"freeleech", orchard, []files.Cell{cell(hd, files.Missing)},
			with(web, func(r *indexers.Release) { r.Factors.Download = 0 }), with(web, func(r *indexers.Release) { r.Factors.Download, r.Seeders = 0.5, 99 }), Loss{Step: StepFreeleech, Own: "0.5", Ref: "0"}},
		{"seeders", orchard, []files.Cell{cell(hd, files.Missing)},
			with(web, func(r *indexers.Release) { r.Seeders = 90 }), web, Loss{Step: StepSeeders, Own: "5", Ref: "90"}},
		{"size", orchard, []files.Cell{cell(hd, files.Missing)},
			with(web, func(r *indexers.Release) { r.Size = 20 }), web, Loss{Step: StepSize, Own: "10", Ref: "20"}},
		{"title", orchard, []files.Cell{cell(hd, files.Missing)},
			offered("Orchard.1080p.WEB-DL.A", 5, 10, nil), offered("Orchard.1080p.WEB-DL.B", 5, 10, nil), Loss{Step: StepTitle, Own: "Orchard.1080p.WEB-DL.B", Ref: "Orchard.1080p.WEB-DL.A"}},
	} {
		cs := cells{open: c.open}
		d := decide(cs, assessedBy(c.j, c.worse, c.better))
		if len(d.Ranked) != 2 || d.Ranked[0].at != 1 {
			t.Errorf("%s: decide ranked %d Releases, want 2 with the better one, listed second, first", c.name, len(d.Ranked))
			continue
		}
		if got := lostAt(cs, d.Ranked[0], d.Ranked[1]); got == nil || *got != c.want {
			t.Errorf("%s: the worse Release lost at %+v, want %+v", c.name, got, c.want)
		}
	}
}

// listedRuns are Releases of The Quiet Orchard at the 1080p cell: a
// Bluray, a WEB-DL listed again by a worse Indexer at 5, a WEBRip, another
// film's Bluray and a cam.
func listedRuns() (cells, Decision) {
	ls := []listed{
		offered("Orchard.1080p.BluRay", 5, 10, nil),
		offered("Orchard.1080p.WEB-DL", 90, 10, nil),
		offered("Orchard.1080p.WEBRip", 10, 10, nil),
		offered("Another.Film.1080p.BluRay", 1, 10, nil),
		offered("Orchard.1080p.TS.x264", 99, 10, nil),
	}
	as := assessedBy(orchard, ls...)
	mirror := ls[1].Release
	mirror.Priority = 30
	as[1].copies = []copied{{Release: mirror, at: 5}}
	c := cells{open: []files.Cell{cell(hd, files.Missing)}}
	return c, decide(c, as)
}

func TestEachReleaseARunListedHasItsOutcomeAndWhereItLostToTheGrabbedOne(t *testing.T) {
	c, d := listedRuns()
	describe := func(rows []row) []string {
		var out []string
		for _, r := range rows {
			line := fmt.Sprintf("%d %s rank %d copy of %d", r.at, r.outcome, r.rank, r.copyOf)
			if r.loss != nil {
				line += fmt.Sprintf(" lost at %s %s against %s", r.loss.Step, r.loss.Own, r.loss.Ref)
			}
			out = append(out, line)
		}
		return out
	}
	for _, tc := range []struct {
		name  string
		taken int
		want  []string
	}{
		{"the WEB-DL grabbed once the Bluray's Indexer refused it", 1, []string{
			"0 not_taken rank 1 copy of -1",
			"1 grabbed rank 3 copy of -1",
			"2 outranked rank 4 copy of -1 lost at seeders 10 against 90",
			"3 refused rank 2 copy of -1",
			"4 refused rank 0 copy of -1",
			"5 outranked rank 3 copy of 1 lost at priority 30 against 0",
		}},
		{"the WEB-DL's copy grabbed once both better Indexers refused", 5, []string{
			"0 not_taken rank 1 copy of -1",
			"1 not_taken rank 3 copy of -1",
			"2 outranked rank 4 copy of -1 lost at seeders 10 against 90",
			"3 refused rank 2 copy of -1",
			"4 refused rank 0 copy of -1",
			"5 grabbed rank 3 copy of 1",
		}},
		{"no Grab taken", -1, []string{
			"0 kept rank 1 copy of -1",
			"1 kept rank 3 copy of -1 lost at tier WEBDL-1080p against Bluray-1080p",
			"2 kept rank 4 copy of -1 lost at tier WEBRip-1080p against Bluray-1080p",
			"3 refused rank 2 copy of -1",
			"4 refused rank 0 copy of -1",
			"5 kept rank 3 copy of 1 lost at tier WEBDL-1080p against Bluray-1080p",
		}},
	} {
		if got := describe(outcomes(d, c, tc.taken)); !slices.Equal(got, tc.want) {
			t.Errorf("%s: the rows are\n%s\nwant\n%s", tc.name, strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
		}
	}
}

func TestARefusedRowKeepsEveryReasonAndEachTargetsScore(t *testing.T) {
	c, d := listedRuns()
	rows := outcomes(d, c, 1)
	if got, want := rows[3].reasons, []Reason{own(targets.WrongTitle, "Another Film"), notChecked(hd)}; !reflect.DeepEqual(got, want) {
		t.Errorf("another film's Bluray is refused for %+v, want %+v", got, want)
	}
	if got := rows[3].scores; !reflect.DeepEqual(got, []TargetScore{{Target: hd.ID, Score: 0}}) {
		t.Errorf("another film's Bluray scores %+v, want 0 at 1080p, since a soft refusal keeps judging", got)
	}
	if got := rows[1].covers; !slices.Equal(got, []works.ItemID{item.ID}) {
		t.Errorf("the WEB-DL covers %v, want the searched Item", got)
	}
}
