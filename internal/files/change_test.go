package files_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func (l *library) change(t *testing.T, svc *files.Files, c targets.Change, m files.Mode) files.Impact {
	t.Helper()
	i, err := svc.Change(t.Context(), admin, c, m)
	if err != nil {
		t.Fatal(err)
	}
	l.settle(t, svc)
	return i
}

func entries(ts ...targets.Target) []targets.Entry {
	var out []targets.Entry
	for _, t := range ts {
		out = append(out, targets.Entry{Target: t.ID, Sourcing: targets.Download})
	}
	return out
}

// holding is Profile Both, the shelf's, holding ts.
func holding(id targets.ProfileID, ts ...targets.Target) targets.UpdateProfile {
	return targets.UpdateProfile{ID: id, Spec: targets.ProfileSpec{Name: "Both", ContentType: "fixture", Entries: entries(ts...)}}
}

var (
	deleting = files.Mode{Unwanted: files.Delete}
	keeping  = files.Mode{Unwanted: files.Keep}
)

type shelf struct {
	*library
	svc     *files.Files
	tree    works.Tree
	both    targets.ProfileID
	hd, web targets.Target
}

// shelved is a Work wanting from Profile Both, of HD and Web, whose first
// Item holds one WEB-DL filling both, named after HD.
func shelved(t *testing.T) shelf {
	t.Helper()
	s := shelf{library: open(t)}
	s.hd = s.target(t, "HD", []release.Quality{release.Bluray1080p}, []release.Quality{release.WEBDL1080p})
	s.web = s.target(t, "Web", []release.Quality{release.WEBDL1080p})
	s.both = s.profile(t, "Both", s.hd, s.web)
	s.tree = s.work(t, "w1", true, &s.both)
	s.svc = s.files(files.NoGrabs{})
	s.download(t, s.svc, s.tree, 0, release.WEBDL1080p, "webdl")
	s.settle(t, s.svc)
	return s
}

func TestTakingEveryTargetAwayKeepsTheFileAsKeptAndPuttingThemBackRestoresIt(t *testing.T) {
	s := shelved(t)
	got := s.change(t, s.svc, holding(s.both), keeping)
	if want := (files.Impact{Works: 1, Unwanted: 1, Bytes: 5}); got != want {
		t.Fatalf("taking HD and Web away with Keep = %+v, want %+v", got, want)
	}
	kept := map[string]string{"Shelf w1/Shelf w1 - Scarf - HD (extra 1).mkv": "webdl"}
	if disk := s.onDisk(t); !maps.Equal(disk, kept) {
		t.Fatalf("the root holds %v, want the WEB-DL Kept under its label", disk)
	}

	got = s.change(t, s.svc, holding(s.both, s.web, s.hd), deleting)
	if want := (files.Impact{Works: 1, Renamed: 1, NewlyWanted: 3}); got != want {
		t.Fatalf("putting Web and HD back = %+v, want %+v", got, want)
	}
	back := map[string]string{"Shelf w1/Shelf w1 - Scarf - Web.mkv": "webdl"}
	if disk := s.onDisk(t); !maps.Equal(disk, back) {
		t.Fatalf("the root holds %v, want the WEB-DL filling again, named after Web", disk)
	}
	_, cells, err := s.svc.WorkCells(t.Context(), admin, s.tree.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := "Scarf any Web met wanted=false Shelf w1/Shelf w1 - Scarf - Web.mkv webdl-1080p 1920x1080\n" +
		"Scarf any HD upgradable wanted=true Shelf w1/Shelf w1 - Scarf - Web.mkv webdl-1080p 1920x1080\n" +
		"Mittens any Web missing wanted=true\n" +
		"Mittens any HD missing wanted=true\n"
	if lines(cells) != want {
		t.Fatalf("cells =\n%swant\n%s", lines(cells), want)
	}
}

func TestAKeptFileMeetingNoTargetKeepsItsLabelAndARenamedTargetRenamesItsFiles(t *testing.T) {
	s := shelved(t)
	bluray := s.target(t, "Bluray", []release.Quality{release.Bluray1080p})
	s.change(t, s.svc, holding(s.both, bluray, s.hd), keeping)
	s.change(t, s.svc, targets.UpdateTarget{ID: s.hd.ID, Spec: targets.Spec{Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p}}, Cutoff: release.Bluray1080p}}, keeping)
	s.change(t, s.svc, holding(s.both, bluray), keeping)
	if disk := s.onDisk(t); !maps.Equal(disk, map[string]string{"Shelf w1/Shelf w1 - Scarf - HD (extra 1).mkv": "webdl"}) {
		t.Fatalf("the root holds %v, want the WEB-DL Kept under the label it last filled, meeting no Target", disk)
	}
	stream := s.target(t, "Stream", []release.Quality{release.WEBDL1080p})
	renamed := targets.UpdateTarget{ID: stream.ID, Spec: targets.Spec{Name: "Streamed", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p}}
	s.change(t, s.svc, holding(s.both, bluray, stream), keeping)
	if disk := s.onDisk(t); !maps.Equal(disk, map[string]string{"Shelf w1/Shelf w1 - Scarf - Stream.mkv": "webdl"}) {
		t.Fatalf("the root holds %v, want the WEB-DL filling Stream", disk)
	}
	if got := s.change(t, s.svc, renamed, keeping); got != (files.Impact{Works: 1, Renamed: 1}) {
		t.Fatalf("renaming Stream = %+v, want one Work and one file renamed", got)
	}
	if disk := s.onDisk(t); !maps.Equal(disk, map[string]string{"Shelf w1/Shelf w1 - Scarf - Streamed.mkv": "webdl"}) {
		t.Fatalf("the root holds %v, want the WEB-DL renamed with its Target", disk)
	}
}

func TestDeletingSavesRemoveTheFilesTheyLeaveFillingNothing(t *testing.T) {
	s := shelved(t)
	if got := s.change(t, s.svc, holding(s.both, s.hd), deleting); got != (files.Impact{Works: 1}) {
		t.Fatalf("taking Web away = %+v, want one Work changed and the WEB-DL kept for HD", got)
	}
	if got := s.change(t, s.svc, holding(s.both), deleting); got != (files.Impact{Works: 1, Unwanted: 1, Bytes: 5}) {
		t.Fatalf("taking HD away = %+v, want the WEB-DL unwanted", got)
	}
	if disk := s.onDisk(t); len(disk) != 0 || fileCount(t, s.library) != 0 {
		t.Fatalf("the root holds %v and the library %d files, want neither", disk, fileCount(t, s.library))
	}
}

// snapshot is everything a user of l can see: what the root holds, each
// file's row and every cell.
func (l *library) snapshot(t *testing.T, svc *files.Files, ws ...works.WorkID) string {
	t.Helper()
	var out strings.Builder
	disk := l.onDisk(t)
	for _, p := range slices.Sorted(maps.Keys(disk)) {
		out.WriteString(p + "=" + disk[p] + "\n")
	}
	for _, w := range ws {
		g, err := l.targets.Goals(t.Context(), admin, w)
		if err != nil {
			t.Fatal(err)
		}
		_, cells, err := svc.WorkCells(t.Context(), admin, w)
		if err != nil {
			t.Fatal(err)
		}
		out.WriteString(g.Profile.Name + "\n" + lines(cells))
	}
	return out.String()
}

func TestAPreviewChangesNothingAndItsImpactIsTheSaves(t *testing.T) {
	s := shelved(t)
	other := s.work(t, "w2", true, &s.both)
	s.download(t, s.svc, other, 1, release.Bluray1080p, "bluray")
	s.settle(t, s.svc)
	bluray := s.target(t, "Bluray", []release.Quality{release.Bluray1080p})
	for _, c := range []struct {
		name   string
		change targets.Change
		want   files.Impact
	}{
		{"a reordered Profile", holding(s.both, s.web, s.hd), files.Impact{Works: 2, Renamed: 1}},
		{"a tightened Target", targets.UpdateTarget{ID: s.hd.ID, Spec: targets.Spec{Name: "HD", Tiers: [][]release.Quality{{release.Bluray1080p}}, Cutoff: release.Bluray1080p}}, files.Impact{Works: 1}},
		{"a Profile of other Targets", holding(s.both, bluray), files.Impact{Works: 2, Unwanted: 1, Bytes: 5, Renamed: 1, NewlyWanted: 3}},
	} {
		before := s.snapshot(t, s.svc, s.tree.ID, other.ID)
		preview, err := s.svc.Change(t.Context(), admin, c.change, files.Mode{Unwanted: files.Delete, Preview: true})
		if err != nil {
			t.Fatal(err)
		}
		s.settle(t, s.svc)
		if after := s.snapshot(t, s.svc, s.tree.ID, other.ID); after != before {
			t.Fatalf("previewing %s changed\n%s\nto\n%s", c.name, before, after)
		}
		if saved := s.change(t, s.svc, c.change, deleting); preview != c.want || saved != preview {
			t.Fatalf("%s: preview %+v and save %+v, want both %+v", c.name, preview, saved, c.want)
		}
	}
}

func TestASaveThatWouldDeleteMoreThanConfirmedIsRefused(t *testing.T) {
	s := shelved(t)
	before := s.snapshot(t, s.svc, s.tree.ID)
	_, err := s.svc.Change(t.Context(), admin, holding(s.both), files.Mode{Unwanted: files.Delete, Confirmed: new(0)})
	changed, ok := errors.AsType[*files.ImpactChangedError](err)
	if !ok || changed.Impact != (files.Impact{Works: 1, Unwanted: 1, Bytes: 5}) || changed.Confirmed != 0 {
		t.Fatalf("a save confirming no deletes that deletes one: err = %v, want *ImpactChangedError with the fresh Impact", err)
	}
	s.settle(t, s.svc)
	if after := s.snapshot(t, s.svc, s.tree.ID); after != before {
		t.Fatalf("the refused save changed\n%s\nto\n%s", before, after)
	}
	if got := s.change(t, s.svc, holding(s.both), files.Mode{Unwanted: files.Keep, Confirmed: new(0)}); got.Unwanted != 1 {
		t.Fatalf("a save keeping the file = %+v, want it saved: keeping deletes nothing", got)
	}
}

func TestMovingAWorkToAnotherProfileRefillsItAndDeletingAProfileMovesItsWorks(t *testing.T) {
	s := shelved(t)
	bluray := s.target(t, "Bluray", []release.Quality{release.Bluray1080p})
	discs := s.profile(t, "Discs", bluray)
	got := s.change(t, s.svc, targets.SetWants{Work: s.tree.ID, Profile: discs}, keeping)
	if want := (files.Impact{Works: 1, Unwanted: 1, Bytes: 5, NewlyWanted: 2}); got != want {
		t.Fatalf("moving the Work to Discs = %+v, want %+v", got, want)
	}
	hds := s.profile(t, "HDs", s.hd)
	got = s.change(t, s.svc, targets.DeleteProfile{ID: discs, To: hds}, deleting)
	if want := (files.Impact{Works: 1, Renamed: 1, NewlyWanted: 2}); got != want {
		t.Fatalf("deleting Discs into HDs = %+v, want %+v: the Kept WEB-DL fills HD again", got, want)
	}
	if snap := s.snapshot(t, s.svc, s.tree.ID); snap != "Shelf w1/Shelf w1 - Scarf - HD.mkv=webdl\nHDs\n"+
		"Scarf any HD upgradable wanted=true Shelf w1/Shelf w1 - Scarf - HD.mkv webdl-1080p 1920x1080\n"+
		"Mittens any HD missing wanted=true\n" {
		t.Fatalf("after the delete:\n%s", snap)
	}
	for _, c := range []struct {
		change targets.Change
		want   error
	}{
		{targets.DeleteProfile{ID: s.both, To: s.both}, targets.ErrMovesToItself},
		{targets.DeleteProfile{ID: discs, To: hds}, targets.ErrNoProfile},
		{targets.DeleteProfile{ID: s.both, To: discs}, targets.ErrNoReplacement},
		{targets.SetWants{Work: s.tree.ID, Profile: discs}, targets.ErrNoProfile},
		{targets.SetWants{Work: works.WorkID{UUID: storage.NewUUID()}, Profile: hds}, works.ErrNoWork},
	} {
		if _, err := s.svc.Change(t.Context(), admin, c.change, deleting); !errors.Is(err, c.want) {
			t.Errorf("%+v: err = %v, want %v", c.change, err, c.want)
		}
	}
}

func TestAnImportWhoseCellsWentFollowsTheSavedChoice(t *testing.T) {
	for _, c := range []struct {
		name   string
		choice files.Unwanted
		meant  bool
		disk   map[string]string
		landed bool
	}{
		{"delete", files.Delete, false, map[string]string{}, false},
		{"keep named by its quality", files.Keep, false, map[string]string{"Shelf w1/Shelf w1 - Scarf - WEBDL-1080p (extra 1).mkv": "webdl"}, true},
		{"keep named by the Target its Grab meant", files.Keep, true, map[string]string{"Shelf w1/Shelf w1 - Scarf - Disc (extra 1).mkv": "webdl"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := open(t)
			empty := l.profile(t, "Empty")
			tree := l.work(t, "w1", true, &empty)
			var meant []targets.TargetID
			if c.meant {
				meant = []targets.TargetID{{UUID: storage.NewUUID()}, l.target(t, "Disc", []release.Quality{release.Bluray1080p}).ID}
			}
			svc := l.files(files.NoGrabs{})
			holding := files.HoldingOf(storage.NewUUID(), ".mkv")
			l.put(t, holding, "webdl")
			in := files.Incoming{
				WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: l.root.ID, Holding: holding,
				Size: 5, Probe: files.Probe{Width: 1920, Height: 1080}, Quality: release.WEBDL1080p, Unwanted: &c.choice, Meant: meant,
			}
			var landed bool
			err := l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				var err error
				_, landed, err = svc.Import(ctx, tx, in)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			l.settle(t, svc)
			if disk := l.onDisk(t); landed != c.landed || !maps.Equal(disk, c.disk) {
				t.Fatalf("an import filling nothing under %s landed %t and the root holds %v, want %t and %v", c.choice, landed, disk, c.landed, c.disk)
			}
		})
	}
}

var (
	h264 = files.Probe{Width: 1920, Height: 1080, Codec: "h264"}
	hevc = files.Probe{Width: 1920, Height: 1080, Codec: "hevc"}
)

// noX264 is Web refusing x264: WEB-DL 1080p of any video codec but x264.
func noX264(t *testing.T, id targets.TargetID) targets.UpdateTarget {
	t.Helper()
	var f targets.Filters
	if err := f.Set(targets.VideoCodec, targets.Not, []string{"x264"}); err != nil {
		t.Fatal(err)
	}
	return targets.UpdateTarget{ID: id, Spec: targets.Spec{Name: "Web", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p, Filters: f}}
}

func TestATargetRefusingTheVideoCodecAFileWasProbedAtLetsGoOfIt(t *testing.T) {
	for _, c := range []struct {
		choice files.Unwanted
		disk   map[string]string
	}{
		{files.Keep, map[string]string{"Shelf w1/Shelf w1 - Scarf - Web (extra 1).mkv": "h264", "Shelf w1/Shelf w1 - Mittens - Web.mkv": "hevc"}},
		{files.Delete, map[string]string{"Shelf w1/Shelf w1 - Mittens - Web.mkv": "hevc"}},
	} {
		t.Run(string(c.choice), func(t *testing.T) {
			l := open(t)
			web := l.target(t, "Web", []release.Quality{release.WEBDL1080p})
			tree := l.work(t, "w1", true, new(l.profile(t, "Films", web)))
			svc := l.files(files.NoGrabs{})
			l.downloadProbed(t, svc, tree, 0, release.WEBDL1080p, "h264", h264)
			l.downloadProbed(t, svc, tree, 1, release.WEBDL1080p, "hevc", hevc)
			l.settle(t, svc)
			if got := l.change(t, svc, noX264(t, web.ID), files.Mode{Unwanted: c.choice}); got != (files.Impact{Works: 1, Unwanted: 1, Bytes: 4, NewlyWanted: 1}) {
				t.Fatalf("refusing x264 under %s = %+v, want the H.264 file unwanted and its cell wanted", c.choice, got)
			}
			if disk := l.onDisk(t); !maps.Equal(disk, c.disk) {
				t.Fatalf("under %s the root holds %v, want %v", c.choice, disk, c.disk)
			}
			_, cells, err := svc.WorkCells(t.Context(), admin, tree.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := "Scarf any Web missing wanted=true\n" +
				"Mittens any Web met wanted=false Shelf w1/Shelf w1 - Mittens - Web.mkv webdl-1080p 1920x1080\n"
			if lines(cells) != want {
				t.Fatalf("cells =\n%swant\n%s", lines(cells), want)
			}
		})
	}
}

func TestAnImportProbedAtAVideoCodecItsTargetRefusesFillsNothing(t *testing.T) {
	keep, del := files.Keep, files.Delete
	for _, c := range []struct {
		name     string
		probe    files.Probe
		unwanted *files.Unwanted
		fills    bool
		disk     map[string]string
	}{
		{"hevc fills Web", hevc, nil, true, map[string]string{"Shelf w1/Shelf w1 - Scarf - Web.mkv": "payload"}},
		{"h264 with no choice", h264, nil, false, nil},
		{"h264 kept", h264, &keep, false, map[string]string{"Shelf w1/Shelf w1 - Scarf - Web (extra 1).mkv": "payload"}},
		{"h264 deleted", h264, &del, false, map[string]string{}},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := open(t)
			web := must[targets.Target](t)(l.targets.CreateTarget(t.Context(), admin, noX264(t, targets.TargetID{}).Spec))
			tree := l.work(t, "w1", true, new(l.profile(t, "Films", web)))
			svc := l.files(files.NoGrabs{})
			holding := files.HoldingOf(storage.NewUUID(), ".mkv")
			l.put(t, holding, "payload")
			in := files.Incoming{
				WorkID: tree.ID, ItemID: tree.Items[0].ID, Cut: release.Standard, RootID: l.root.ID, Holding: holding,
				Size: 7, Probe: c.probe, Quality: release.WEBDL1080p, Meant: []targets.TargetID{web.ID}, Unwanted: c.unwanted,
			}
			var placed files.Placement
			err := l.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
				var err error
				placed, err = svc.Place(ctx, r, in)
				return err
			})
			_, nothing := errors.AsType[*files.FillsNothingError](err)
			kept := c.unwanted != nil && *c.unwanted == files.Keep
			switch {
			case c.fills && (err != nil || !slices.Equal(placed.Fills, []targets.TargetID{web.ID})):
				t.Fatalf("Place = %+v, %v, want it to fill Web", placed, err)
			case !c.fills && !kept && !nothing:
				t.Fatalf("Place = %+v, %v, want a FillsNothingError", placed, err)
			case kept && (err != nil || !placed.Kept):
				t.Fatalf("Place = %+v, %v, want it Kept", placed, err)
			}
			if c.disk == nil {
				return
			}
			err = l.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				_, _, err := svc.Import(ctx, tx, in)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			l.settle(t, svc)
			if disk := l.onDisk(t); !maps.Equal(disk, c.disk) {
				t.Fatalf("the root holds %v, want %v", disk, c.disk)
			}
		})
	}
}
