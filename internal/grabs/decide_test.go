package grabs

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/naming"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func TestResolutionReadsTheFirstWordBoundedToken(t *testing.T) {
	for title, want := range map[string]targets.Resolution{
		"The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW": targets.R1080p,
		"Orchard 2021 4K HDR":                          targets.R2160p,
		"orchard.UHD.bluray":                           targets.R2160p,
		"Orchard [720P]":                               targets.R720p,
		"Orchard.576p.DVD":                             targets.R576p,
		"Orchard 480p 1080p":                           targets.R480p,
	} {
		if got, ok := resolution(title); !ok || got != want {
			t.Errorf("resolution(%q) = %s, %v; want %s", title, got, ok, want)
		}
	}
	for _, title := range []string{"Orchard.2021.WEB-DL", "Orchard.1080i", "Orchard.x1080p", "4kids"} {
		if got, ok := resolution(title); ok {
			t.Errorf("resolution(%q) = %s, want none", title, got)
		}
	}
}

var (
	hd    = targets.Target{ID: targets.TargetID{UUID: storage.NewUUID()}, Name: "1080p", Resolutions: []targets.Resolution{targets.R1080p}}
	small = targets.Target{ID: targets.TargetID{UUID: storage.NewUUID()}, Name: "Small", Resolutions: []targets.Resolution{targets.R720p, targets.R576p}}
	item  = works.Item{ID: works.ItemID{UUID: storage.NewUUID()}, Monitored: true}
)

func cell(t targets.Target, s files.State) files.Cell {
	return files.Cell{Item: item, Edition: files.Standard, Target: t, State: s}
}

var orchard = indexers.Terms{Mode: "search", IDs: map[string]string{"tmdbid": "9000001", "imdbid": "tt9000001"}, Query: "The Quiet Orchard 2021"}

func release(title string, seeders, size int64, ids map[string]string) indexers.Item {
	return indexers.Item{Title: title, Seeders: seeders, Size: size, IDs: ids}
}

func TestDecideRanksByTargetThenResolutionThenSeedersSizeAndTitle(t *testing.T) {
	open := []files.Cell{cell(hd, files.Missing), cell(small, files.Missing)}
	candidates := []indexers.Item{
		release("Orchard.720p.A", 90, 10, nil),
		release("Orchard.576p.B", 99, 10, nil),
		release("Orchard.1080p.C", 5, 10, map[string]string{"tmdbid": "9000001"}),
		release("Orchard.1080p.D", 5, 20, nil),
		release("Orchard.1080p.B", 5, 20, nil),
		release("Orchard.1080p.A", 6, 1, nil),
	}
	pick, rejected := decide(open, orchard, candidates)
	if want := (&Pick{Item: candidates[5], Targets: []targets.TargetID{hd.ID}}); !reflect.DeepEqual(pick, want) || rejected != nil {
		t.Fatalf("decide = %+v, %v; want %+v", pick, rejected, want)
	}
	pick, _ = decide(open, orchard, candidates[:5])
	if pick.Item.Title != "Orchard.1080p.B" {
		t.Fatalf("decide without the most seeded 1080p picked %s, want the larger, then first by title, Orchard.1080p.B", pick.Item.Title)
	}
	pick, _ = decide(open[1:], orchard, candidates)
	if want := (&Pick{Item: candidates[0], Targets: []targets.TargetID{small.ID}}); !reflect.DeepEqual(pick, want) {
		t.Fatalf("decide for Small = %+v, want 720p ranked over 576p: %+v", pick, want)
	}
}

func TestDecideRejectsTheWrongWorkAnUnknownResolutionAndWhatMeetsNothing(t *testing.T) {
	candidates := []indexers.Item{
		release("Orchard.1080p.Other", 50, 10, map[string]string{"tmdbid": "9000002"}),
		release("Orchard.WEB-DL", 50, 10, nil),
		release("Orchard.720p", 50, 10, nil),
	}
	pick, rejected := decide([]files.Cell{cell(hd, files.Missing)}, orchard, candidates)
	want := []Rejection{{"Orchard.1080p.Other", WrongWork}, {"Orchard.WEB-DL", ResolutionUnknown}, {"Orchard.720p", MeetsNothing}}
	if pick != nil || !reflect.DeepEqual(rejected, want) {
		t.Fatalf("decide = %+v, %v; want nothing and %v", pick, rejected, want)
	}
}

func TestDecideWithNothingWantedPicksNothing(t *testing.T) {
	pick, rejected := decide(nil, orchard, []indexers.Item{release("Orchard.1080p", 1, 1, nil)})
	if want := []Rejection{{"Orchard.1080p", MeetsNothing}}; pick != nil || !reflect.DeepEqual(rejected, want) {
		t.Fatalf("decide with no open cell = %+v, %v; want nothing and %v", pick, rejected, want)
	}
}

func TestPlanNamesTheFileAfterTheFirstTargetItFills(t *testing.T) {
	both := targets.Target{ID: targets.TargetID{UUID: storage.NewUUID()}, Name: "Any HD", Resolutions: []targets.Resolution{targets.R720p, targets.R1080p}}
	root := roots.Root{ID: roots.ID{UUID: storage.NewUUID()}, Path: "/media/movies"}
	templates := naming.Templates{Folder: "{Movie Title} ({Year})", File: "{Movie Title} ({Year}) - {Edition Tag} {Target}"}
	tokens := map[string]string{"Movie Title": "The Quiet Orchard", "Year": "2021"}
	cells := []files.Cell{cell(small, files.Missing), cell(hd, files.Downloading), cell(both, files.Failed)}

	got, err := plan(cells, item.ID, targets.R1080p, root, templates, tokens, "mkv")
	want := Plan{Root: root, Path: "The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv", Fills: []targets.TargetID{hd.ID, both.ID}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("plan = %+v, %v; want %+v", got, err, want)
	}

	tokens["Movie Title"] = `CON: The "Quiet" Orchard.`
	got, err = plan(cells[1:2], item.ID, targets.R1080p, root, templates, tokens, "mkv")
	if want := "CON The Quiet Orchard. (2021)/CON The Quiet Orchard. (2021) - 1080p.mkv"; err != nil || got.Path != want {
		t.Fatalf("plan of a title with reserved characters = %q, %v; want %q", got.Path, err, want)
	}

	_, err = plan([]files.Cell{cell(hd, files.Met), cell(small, files.Missing)}, item.ID, targets.R1080p, root, templates, tokens, "mkv")
	if err == nil || err.Error() != "a 1080p file fills no Target the Work still wants" {
		t.Fatalf("plan of a file whose Targets are met = %v, want fills nothing", err)
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
