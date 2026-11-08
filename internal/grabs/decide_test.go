package grabs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
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
	t, err := targets.NewTarget(targets.Spec{Name: name, Tiers: tiers, Cutoff: cutoff, Sizes: sizes})
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

var orchard = indexers.Terms{Mode: "search", IDs: map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"}, Query: "The Quiet Orchard 2021", Covers: indexers.CoversOnly}

func listed(title string, seeders int, size int64, ids map[string]string) candidate {
	return candidateOf(indexers.Release{Release: searching.Release{Title: title, Seeders: seeders, Size: size, IDs: ids}}, release.BuiltIn(), item.ID, release.Fact[time.Duration]{})
}

func notChecked(t targets.Target) Reason {
	return Reason{Reason: targets.Reason{Code: targets.SizeNotChecked}, Target: t.ID}
}

func TestDecideRanksByTargetThenTierThenSeedersSizeAndTitle(t *testing.T) {
	open := []files.Cell{cell(hd, files.Missing), cell(small, files.Missing)}
	candidates := []candidate{
		listed("Orchard.720p.WEB-DL.A", 90, 10, nil),
		listed("Orchard.576p.DVD.B", 99, 10, nil),
		listed("Orchard.1080p.WEB-DL.C", 5, 10, map[string]string{"tmdbid": "9000001"}),
		listed("Orchard.1080p.WEBRip.D", 5, 20, nil),
		listed("Orchard.1080p.HDTV.B", 5, 20, nil),
		listed("Orchard.1080p.BluRay.A", 1, 1, nil),
	}
	pick, refused := best(decide(open, orchard, candidates))
	if want := (&Pick{Release: candidates[5].Release, Edition: targets.AnyEdition, Targets: []targets.TargetID{hd.ID}, Notes: []Reason{notChecked(hd)}}); !reflect.DeepEqual(pick, want) || refused != nil {
		t.Fatalf("decide = %+v, %v; want %+v", pick, refused, want)
	}
	pick, _ = best(decide(open, orchard, candidates[:5]))
	if pick.Release.Title != "Orchard.1080p.HDTV.B" {
		t.Fatalf("decide among tied 1080p web picked %s, want the larger, then first by title, Orchard.1080p.HDTV.B", pick.Release.Title)
	}
	pick, _ = best(decide(open[1:], orchard, candidates))
	if want := (&Pick{Release: candidates[0].Release, Edition: targets.AnyEdition, Targets: []targets.TargetID{small.ID}, Notes: []Reason{notChecked(small)}}); !reflect.DeepEqual(pick, want) {
		t.Fatalf("decide for Small = %+v, want WEB-DL 720p over a DVD in a lower tier: %+v", pick, want)
	}
	both := target("Any", release.WEBDL1080p, nil, []release.Quality{release.WEBDL1080p, release.WEBDL720p})
	pick, _ = best(decide([]files.Cell{cell(small, files.Missing), cell(both, files.Missing)}, orchard, candidates[:1]))
	if want := []targets.TargetID{small.ID, both.ID}; !reflect.DeepEqual(pick.Targets, want) {
		t.Fatalf("decide for two Targets = %v, want both in Profile order", pick.Targets)
	}
}

func TestDecideRefusesWithEveryReasonPerTarget(t *testing.T) {
	big := listed("Orchard.1080p.WEB-DL.Big", 50, 10000*(1<<20), nil)
	big.Facts.Coverage.Runtime = release.Known(100 * time.Minute)
	candidates := []candidate{
		listed("Orchard.1080p.WEB-DL.Other", 50, 10, map[string]string{"tmdbid": "9000002"}),
		listed("Orchard.WEB-DL", 50, 10, nil),
		listed("Orchard.2021", 50, 10, nil),
		listed("Orchard.1080p.TS", 50, 10, nil),
		big,
	}
	pick, refused := best(decide([]files.Cell{cell(hd, files.Missing), cell(small, files.Failed)}, orchard, candidates))
	notIn := func(t targets.Target, q string) Reason {
		return Reason{Reason: targets.Reason{Code: targets.QualityNotInTarget, Value: q}, Target: t.ID}
	}
	want := []Refusal{
		{"Orchard.1080p.WEB-DL.Other", []Reason{{Reason: targets.Reason{Code: WrongWork}}}},
		{"Orchard.2021", []Reason{notIn(hd, ""), notIn(small, "")}},
		{"Orchard.1080p.TS", []Reason{notIn(hd, "cam"), notIn(small, "cam")}},
		{"Orchard.1080p.WEB-DL.Big", []Reason{{Reason: targets.Reason{Code: targets.SizeAbove}, Target: hd.ID}, notIn(small, "webdl-1080p")}},
	}
	if want := (&Pick{Release: candidates[1].Release, Edition: targets.AnyEdition, Targets: []targets.TargetID{small.ID}, Notes: []Reason{notChecked(small)}}); !reflect.DeepEqual(pick, want) {
		t.Fatalf("decide = %+v, want the source-only WEB-DL as 720p for Small: %+v", pick, want)
	}
	if !reflect.DeepEqual(refused, want) {
		t.Fatalf("refused = %+v\nwant %+v", refused, want)
	}
}

// lanternS01E03 is S01E03 of TVDB series 1001, covered by the titles
// that name S01E03 alone.
var lanternS01E03 = indexers.Terms{
	Mode: "tvsearch", IDs: map[string]string{"tvdbid": "1001"}, Query: "The Lantern Keepers", Season: new(1), Episode: "3",
	Covers: func(title string) indexers.Covering {
		switch {
		case strings.Contains(title, ".S01E02."):
			return indexers.Covering{Kind: indexers.CoversOther, Names: "S01E02"}
		case strings.Contains(title, ".S01."):
			return indexers.Covering{Kind: indexers.CoversMore, Names: "S01"}
		}
		return indexers.Covering{Kind: indexers.CoversItem}
	},
}

func TestDecideRefusesAnotherEpisodeAndAPack(t *testing.T) {
	candidates := []candidate{
		listed("Lantern.Keepers.S01E02.1080p.WEB-DL", 90, 10, nil),
		listed("Lantern.Keepers.S01.1080p.WEB-DL", 90, 10, nil),
		listed("Lantern.Keepers.S01E03.1080p.WEB-DL", 5, 10, map[string]string{"tvdbid": "1001"}),
		listed("Lantern.Keepers.S01E03.Extended.1080p.WEB-DL", 5, 10, map[string]string{"tvdbid": "1002"}),
	}
	pick, refused := best(decide([]files.Cell{cell(hd, files.Missing)}, lanternS01E03, candidates))
	if pick == nil || pick.Release.Title != "Lantern.Keepers.S01E03.1080p.WEB-DL" {
		t.Fatalf("decide picked %+v, want the release of S01E03 alone over better-seeded ones of S01E02 and S01", pick)
	}
	want := []Refusal{
		{"Lantern.Keepers.S01E02.1080p.WEB-DL", []Reason{{Reason: targets.Reason{Code: WrongItem, Value: "S01E02"}}}},
		{"Lantern.Keepers.S01.1080p.WEB-DL", []Reason{{Reason: targets.Reason{Code: CoversMore, Value: "S01"}}}},
		{"Lantern.Keepers.S01E03.Extended.1080p.WEB-DL", []Reason{{Reason: targets.Reason{Code: WrongWork}}}},
	}
	if !reflect.DeepEqual(refused, want) {
		t.Fatalf("refused = %+v\nwant another episode as wrong_item, the pack as covers_more and another series as wrong_work, each before its cut: %+v", refused, want)
	}
}

func TestDecideAsksARankAboveTheFileFillingTheCell(t *testing.T) {
	held := cell(hd, files.Upgradable)
	held.File = &files.File{Quality: release.WEBDL1080p}
	candidates := []candidate{listed("Orchard.1080p.WEBRip", 90, 10, nil), listed("Orchard.1080p.BluRay", 1, 10, nil)}
	pick, refused := best(decide([]files.Cell{held}, orchard, candidates))
	want := []Refusal{{"Orchard.1080p.WEBRip", []Reason{notChecked(hd), {Reason: targets.Reason{Code: targets.NotAnUpgrade}, Target: hd.ID}}}}
	if pick == nil || pick.Release.Title != "Orchard.1080p.BluRay" || !reflect.DeepEqual(refused, want) {
		t.Fatalf("decide over a held WEB-DL = %+v, %+v; want the Bluray and the WEBRip refused as %+v", pick, refused, want)
	}
}

func TestDecideJudgesACandidateOnlyForTheCellsOfItsCut(t *testing.T) {
	standard, extended := targets.EditionOf(release.Standard), targets.Edition("extended")
	candidates := []candidate{listed("Orchard.1080p.BluRay", 90, 10, nil), listed("Orchard.EXTENDED.1080p.WEB-DL", 5, 10, nil)}
	pick, refused := best(decide([]files.Cell{cellOf(standard, hd, files.Upgradable), cellOf(extended, hd, files.Missing)}, orchard, candidates))
	if want := (&Pick{Release: candidates[0].Release, Edition: standard, Targets: []targets.TargetID{hd.ID}, Notes: []Reason{notChecked(hd)}}); !reflect.DeepEqual(pick, want) || refused != nil {
		t.Fatalf("decide over Standard then Extended = %+v, %+v; want the Standard Bluray for Standard's cell: %+v", pick, refused, want)
	}
	pick, refused = best(decide([]files.Cell{cellOf(extended, hd, files.Missing)}, orchard, candidates))
	want := []Refusal{{"Orchard.1080p.BluRay", []Reason{{Reason: targets.Reason{Code: CutNotWanted, Value: "standard"}}}}}
	if pick == nil || pick.Release.Title != "Orchard.EXTENDED.1080p.WEB-DL" || pick.Edition != extended || !reflect.DeepEqual(refused, want) {
		t.Fatalf("decide over Extended alone = %+v, %+v; want the Extended WEB-DL and the Standard Bluray refused as %+v", pick, refused, want)
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
func best(picks []Pick, refused []Refusal) (*Pick, []Refusal) {
	if len(picks) == 0 {
		return nil, refused
	}
	return &picks[0], refused
}

func from(priority int, title, hash string, size int64, at int) candidate {
	c := listed(title, 5, size, nil)
	c.Release.Priority, c.at = priority, at
	if hash != "" {
		h := bencode.InfoHash([]byte(strings.Repeat(hash, 20)))
		c.Release.InfoHash = &h
	}
	return c
}

func TestDedupeCountsAReleaseListedTwiceOnceFromTheBetterPriority(t *testing.T) {
	kept := dedupe([]candidate{
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

func TestDecideRanksTheBetterIndexerPriorityAfterTierAndBeforeSeeders(t *testing.T) {
	open := []files.Cell{cell(hd, files.Missing)}
	worse := listed("Orchard.1080p.WEB-DL.A", 90, 20, nil)
	worse.Release.Priority = 25
	better := listed("Orchard.1080p.WEBRip.B", 1, 20, nil)
	better.Release.Priority = 10
	bluray := listed("Orchard.1080p.BluRay.C", 1, 20, nil)
	bluray.Release.Priority = 50
	picks, _ := decide(open, orchard, []candidate{worse, better})
	if len(picks) != 2 || picks[0].Release.Title != better.Release.Title || picks[1].Release.Title != worse.Release.Title {
		t.Fatalf("decide ranked %+v, want the priority 10 Release over the better-seeded priority 25 one in one tier", picks)
	}
	if pick, _ := best(decide(open, orchard, []candidate{worse, better, bluray})); pick.Release.Title != bluray.Release.Title {
		t.Fatalf("decide picked %s, want %s from a better tier whatever its Indexer's priority", pick.Release.Title, bluray.Release.Title)
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
