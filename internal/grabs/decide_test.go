package grabs

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
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
	return files.Cell{Item: item, Edition: files.Standard, Target: t, State: s}
}

var orchard = indexers.Terms{Mode: "search", IDs: map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"}, Query: "The Quiet Orchard 2021"}

func listed(title string, seeders, size int64, ids map[string]string) candidate {
	return candidateOf(indexers.Item{Title: title, Seeders: seeders, Size: size, IDs: ids}, item.ID)
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
	pick, refused := decide(open, orchard, candidates)
	if want := (&Pick{Item: candidates[5].Item, Targets: []targets.TargetID{hd.ID}, Notes: []Reason{notChecked(hd)}}); !reflect.DeepEqual(pick, want) || refused != nil {
		t.Fatalf("decide = %+v, %v; want %+v", pick, refused, want)
	}
	pick, _ = decide(open, orchard, candidates[:5])
	if pick.Item.Title != "Orchard.1080p.HDTV.B" {
		t.Fatalf("decide among tied 1080p web picked %s, want the larger, then first by title, Orchard.1080p.HDTV.B", pick.Item.Title)
	}
	pick, _ = decide(open[1:], orchard, candidates)
	if want := (&Pick{Item: candidates[0].Item, Targets: []targets.TargetID{small.ID}, Notes: []Reason{notChecked(small)}}); !reflect.DeepEqual(pick, want) {
		t.Fatalf("decide for Small = %+v, want WEB-DL 720p over a DVD in a lower tier: %+v", pick, want)
	}
	both := target("Any", release.WEBDL1080p, nil, []release.Quality{release.WEBDL1080p, release.WEBDL720p})
	pick, _ = decide([]files.Cell{cell(small, files.Missing), cell(both, files.Missing)}, orchard, candidates[:1])
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
	pick, refused := decide([]files.Cell{cell(hd, files.Missing), cell(small, files.Failed)}, orchard, candidates)
	notIn := func(t targets.Target, q string) Reason {
		return Reason{Reason: targets.Reason{Code: targets.QualityNotInTarget, Value: q}, Target: t.ID}
	}
	want := []Refusal{
		{"Orchard.1080p.WEB-DL.Other", []Reason{{Reason: targets.Reason{Code: WrongWork}}}},
		{"Orchard.2021", []Reason{notIn(hd, ""), notIn(small, "")}},
		{"Orchard.1080p.TS", []Reason{notIn(hd, "cam"), notIn(small, "cam")}},
		{"Orchard.1080p.WEB-DL.Big", []Reason{{Reason: targets.Reason{Code: targets.SizeAbove}, Target: hd.ID}, notIn(small, "webdl-1080p")}},
	}
	if want := (&Pick{Item: candidates[1].Item, Targets: []targets.TargetID{small.ID}, Notes: []Reason{notChecked(small)}}); !reflect.DeepEqual(pick, want) {
		t.Fatalf("decide = %+v, want the source-only WEB-DL as 720p for Small: %+v", pick, want)
	}
	if !reflect.DeepEqual(refused, want) {
		t.Fatalf("refused = %+v\nwant %+v", refused, want)
	}
}

func TestDecideAsksARankAboveTheFileFillingTheCell(t *testing.T) {
	held := cell(hd, files.Upgradable)
	held.File = &files.File{Quality: release.WEBDL1080p}
	candidates := []candidate{listed("Orchard.1080p.WEBRip", 90, 10, nil), listed("Orchard.1080p.BluRay", 1, 10, nil)}
	pick, refused := decide([]files.Cell{held}, orchard, candidates)
	want := []Refusal{{"Orchard.1080p.WEBRip", []Reason{notChecked(hd), {Reason: targets.Reason{Code: targets.NotAnUpgrade}, Target: hd.ID}}}}
	if pick == nil || pick.Item.Title != "Orchard.1080p.BluRay" || !reflect.DeepEqual(refused, want) {
		t.Fatalf("decide over a held WEB-DL = %+v, %+v; want the Bluray and the WEBRip refused as %+v", pick, refused, want)
	}
}

func TestPlanNamesTheFileAfterTheFirstTargetItFills(t *testing.T) {
	both := target("Any HD", release.WEBDL1080p, nil, []release.Quality{release.WEBDL1080p}, []release.Quality{release.WEBDL720p})
	root := roots.Root{ID: roots.ID{UUID: storage.NewUUID()}, Path: "/media/movies"}
	templates := naming.Templates{Folder: "{Movie Title} ({Year})", File: "{Movie Title} ({Year}) - {Edition Tag} {Target}"}
	tokens := map[string]string{"Movie Title": "The Quiet Orchard", "Year": "2021"}
	cells := []files.Cell{cell(small, files.Missing), cell(hd, files.Downloading), cell(both, files.Failed)}
	webdl := release.File("The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW", nil, release.R1080p)

	got, err := plan(cells, item.ID, webdl, root, templates, tokens, "mkv")
	want := Plan{Root: root, Path: "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv", Fills: []targets.TargetID{hd.ID, both.ID}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("plan = %+v, %v; want %+v", got, err, want)
	}

	tokens["Movie Title"] = `CON: The "Quiet" Orchard.`
	got, err = plan(cells[1:2], item.ID, webdl, root, templates, tokens, "mkv")
	if want := "CON The Quiet Orchard. (2021)/CON The Quiet Orchard. (2021) - 1080p.mkv"; err != nil || got.Path != want {
		t.Fatalf("plan of a title with reserved characters = %q, %v; want %q", got.Path, err, want)
	}

	filled := cell(hd, files.Upgradable)
	filled.File = &files.File{Quality: release.WEBRip1080p}
	_, err = plan([]files.Cell{filled, cell(small, files.Missing)}, item.ID, webdl, root, templates, tokens, "mkv")
	if err == nil || err.Error() != "a WEBDL-1080p file fills no Target the Work still wants" {
		t.Fatalf("plan of a file whose Targets are filled = %v, want fills nothing", err)
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
		{"link down", live, link.ErrDisconnected, wait{link.ErrDisconnected}},
		{"versions differ", live, link.ErrMismatched, wait{link.ErrMismatched}},
		{"call lost", live, rpc.Errorf(rpc.CodeUnavailable, "the link died"), wait{rpc.Errorf(rpc.CodeUnavailable, "the link died")}},
		{"torrent not re-added", live, downloading.ErrNotPresent, wait{downloading.ErrNotPresent}},
		{"refused by ffprobe", live, rpc.Errorf(rpc.CodeFailedPrecondition, "ffprobe: exit status 1"), fail{ProbeFailed, "failed_precondition: ffprobe: exit status 1"}},
		{"stopping", canceled, link.ErrDisconnected, stopped{}},
	} {
		got := classify(c.ctx, c.err, ProbeFailed)
		if w, ok := got.(wait); ok {
			got = wait{errors.New(w.cause.Error())}
			if want, ok := c.want.(wait); ok {
				c.want = wait{errors.New(want.cause.Error())}
			}
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: classify = %#v, want %#v", c.name, got, c.want)
		}
	}
}
