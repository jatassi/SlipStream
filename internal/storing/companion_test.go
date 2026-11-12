package storing

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/media"
)

func rels(t *testing.T, ss ...string) []Rel {
	t.Helper()
	out := make([]Rel, len(ss))
	for i, s := range ss {
		out[i] = mustRel(t, s)
	}
	return out
}

func sortedTree(t *testing.T, folder string) []string {
	t.Helper()
	got := tree(t, folder)
	slices.Sort(got)
	return got
}

func holds(t *testing.T, path, body string) {
	t.Helper()
	if b, err := os.ReadFile(path); err != nil || string(b) != body {
		t.Errorf("%s holds %q, %v; want %q", path, b, err, body)
	}
}

func TestAMoveCarriesTheCompanionsOfItsLongestStem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		for _, f := range []string{
			"Heat/Heat.mkv", "Heat/Heat.en.srt", "Heat/Heat.nfo", "Heat/Heat.1995.1080p.REMUX.mkv",
			"Heat/Heat.1995.1080p.REMUX.en.srt", "Heat/Heat-trailer.mkv", "Heat/poster.jpg", "Heat/Heat.en.srt.part",
		} {
			w.file(t, f, f)
		}
		dir(t, filepath.Join(w.folder, "Heat", "Heat.extras"))
		from, to := mustRel(t, "Heat/Heat.mkv"), mustRel(t, "Heat (1995)/Heat (1995).mkv")
		surveyed, err := w.remote.Survey(t.Context(), w.at, []Planned{{From: from, To: to}}, true, media.DefaultVideos)
		if err != nil {
			t.Fatal(err)
		}
		if want := rels(t, "Heat/Heat.en.srt", "Heat/Heat.nfo"); !slices.Equal(surveyed[0].Companions, want) {
			t.Errorf("Survey found companions %v, want %v", surveyed[0].Companions, want)
		}
		for range 2 {
			if moved, err := w.remote.Move(t.Context(), w.at, from, to, true, media.DefaultVideos); err != nil || len(moved.Left) != 0 {
				t.Fatalf("Move = %+v, %v; want every companion carried", moved, err)
			}
		}
		want := []string{
			"Heat (1995)/Heat (1995).en.srt", "Heat (1995)/Heat (1995).mkv", "Heat (1995)/Heat (1995).nfo",
			"Heat/Heat-trailer.mkv", "Heat/Heat.1995.1080p.REMUX.en.srt", "Heat/Heat.1995.1080p.REMUX.mkv", "Heat/Heat.en.srt.part", "Heat/poster.jpg",
		}
		if got := sortedTree(t, w.folder); !slices.Equal(got, want) {
			t.Errorf("the root holds %v, want %v", got, want)
		}
		holds(t, filepath.Join(w.folder, "Heat (1995)", "Heat (1995).en.srt"), "Heat/Heat.en.srt")
	})
}

func TestAMoveWithoutCarryLeavesTheCompanions(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat/Heat.mkv", "film")
		w.file(t, "Heat/Heat.en.srt", "subs")
		if _, err := w.remote.Move(t.Context(), w.at, mustRel(t, "Heat/Heat.mkv"), mustRel(t, "Heat (1995)/Heat (1995).mkv"), false, media.DefaultVideos); err != nil {
			t.Fatal(err)
		}
		if got, want := sortedTree(t, w.folder), []string{"Heat (1995)/Heat (1995).mkv", "Heat/Heat.en.srt"}; !slices.Equal(got, want) {
			t.Errorf("the root holds %v, want %v", got, want)
		}
	})
}

func TestACompanionWhoseTargetIsTakenStaysAndIsReported(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat/Heat.mkv", "film")
		w.file(t, "Heat/Heat.en.srt", "subs")
		w.file(t, "Heat/Heat.nfo", "info")
		w.file(t, "Heat (1995)/Heat (1995).en.srt", "other subs")
		for range 2 {
			moved, err := w.remote.Move(t.Context(), w.at, mustRel(t, "Heat/Heat.mkv"), mustRel(t, "Heat (1995)/Heat (1995).mkv"), true, media.DefaultVideos)
			if err != nil {
				t.Fatal(err)
			}
			if want := rels(t, "Heat/Heat.en.srt"); !slices.Equal(moved.Left, want) {
				t.Errorf("Move left %v, want %v", moved.Left, want)
			}
		}
		want := []string{"Heat (1995)/Heat (1995).en.srt", "Heat (1995)/Heat (1995).mkv", "Heat (1995)/Heat (1995).nfo", "Heat/Heat.en.srt"}
		if got := sortedTree(t, w.folder); !slices.Equal(got, want) {
			t.Errorf("the root holds %v, want %v", got, want)
		}
		holds(t, filepath.Join(w.folder, "Heat (1995)", "Heat (1995).en.srt"), "other subs")
		holds(t, filepath.Join(w.folder, "Heat", "Heat.en.srt"), "subs")
	})
}

func TestCompanionsGoThroughHoldingUnderTheFileID(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		const id = "0190a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"
		w.file(t, "Heat/Heat.mkv", "film")
		w.file(t, "Heat/Heat.en.srt", "subs")
		w.file(t, "Heat/Heat.nfo", "info")
		w.file(t, "Heat (1995)/Heat (1995).en.srt", "other subs")
		hold := mustRel(t, HoldingDir+"/"+id+".mkv")
		if moved, err := w.remote.Move(t.Context(), w.at, mustRel(t, "Heat/Heat.mkv"), hold, true, media.DefaultVideos); err != nil || len(moved.Left) != 0 {
			t.Fatalf("Move into holding = %+v, %v; want every companion carried", moved, err)
		}
		want := []string{HoldingDir + "/" + id + ".en.srt", HoldingDir + "/" + id + ".mkv", HoldingDir + "/" + id + ".nfo", "Heat (1995)/Heat (1995).en.srt"}
		if got := sortedTree(t, w.folder); !slices.Equal(got, want) {
			t.Fatalf("the root holds %v, want %v", got, want)
		}
		beside := "Heat (1995)/Heat (1995).0190a1b2.en.srt"
		moved, err := w.remote.Move(t.Context(), w.at, hold, mustRel(t, "Heat (1995)/Heat (1995).mkv"), true, media.DefaultVideos)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(moved.Left, rels(t, beside)) {
			t.Errorf("Move out of holding left %v, want [%s]", moved.Left, beside)
		}
		want = []string{beside, "Heat (1995)/Heat (1995).en.srt", "Heat (1995)/Heat (1995).mkv", "Heat (1995)/Heat (1995).nfo"}
		if got := sortedTree(t, w.folder); !slices.Equal(got, want) {
			t.Errorf("the root holds %v, want %v and nothing in holding", got, want)
		}
		holds(t, filepath.Join(w.folder, filepath.FromSlash(beside)), "subs")
	})
}

func TestAMoveRetriedAfterItsFileMovedCarriesWhatIsLeft(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat (1995)/Heat (1995).mkv", "film")
		w.file(t, "Heat (1995)/Heat (1995).en.srt", "subs")
		w.file(t, "Heat/Heat.nfo", "info")
		moved, err := w.remote.Move(t.Context(), w.at, mustRel(t, "Heat/Heat.mkv"), mustRel(t, "Heat (1995)/Heat (1995).mkv"), true, media.DefaultVideos)
		if err != nil || moved.Size != 4 || len(moved.Left) != 0 {
			t.Fatalf("Move = %+v, %v; want the moved file's 4 bytes and the rest carried", moved, err)
		}
		want := []string{"Heat (1995)/Heat (1995).en.srt", "Heat (1995)/Heat (1995).mkv", "Heat (1995)/Heat (1995).nfo"}
		if got := sortedTree(t, w.folder); !slices.Equal(got, want) {
			t.Errorf("the root holds %v, want %v", got, want)
		}
	})
}

func TestAMoveRetriedAfterItsFolderWentConverges(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat (1995)/Heat (1995).mkv", "film")
		if moved, err := w.remote.Move(t.Context(), w.at, mustRel(t, "Heat/Heat.mkv"), mustRel(t, "Heat (1995)/Heat (1995).mkv"), true, media.DefaultVideos); err != nil || moved.Size != 4 {
			t.Fatalf("Move = %+v, %v; want the moved file's 4 bytes", moved, err)
		}
	})
}

func TestACaseOnlyMoveRespellsItsCompanions(t *testing.T) {
	for _, fold := range []bool{false, true} {
		synctest.Test(t, func(t *testing.T) {
			w := newWalking(t, nil)
			w.disk.foldCase = fold
			w.file(t, "heat/heat.mkv", "film")
			w.file(t, "heat/heat.en.srt", "subs")
			for range 2 {
				if moved, err := w.remote.Move(t.Context(), w.at, mustRel(t, "heat/heat.mkv"), mustRel(t, "Heat/Heat.mkv"), true, media.DefaultVideos); err != nil || len(moved.Left) != 0 {
					t.Fatalf("folding %v: Move = %+v, %v; want the companion carried", fold, moved, err)
				}
			}
			want := []string{"Heat/Heat.en.srt", "Heat/Heat.mkv"}
			if got := sortedTree(t, w.folder); !slices.Equal(got, want) {
				t.Errorf("folding %v: the root holds %v, want %v", fold, got, want)
			}
			holds(t, filepath.Join(w.folder, "Heat", "Heat.en.srt"), "subs")
		})
	}
}

func TestACaseOnlyCompanionRenameCutShortFinishesOnRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.disk.foldCase = true
		w.file(t, "Heat.mkv", "film")
		w.file(t, "Heat.en.srt"+recaseSuffix, "subs")
		if _, err := w.remote.Move(t.Context(), w.at, mustRel(t, "heat.mkv"), mustRel(t, "Heat.mkv"), true, media.DefaultVideos); err != nil {
			t.Fatal(err)
		}
		if got, want := sortedTree(t, w.folder), []string{"Heat.en.srt", "Heat.mkv"}; !slices.Equal(got, want) {
			t.Errorf("the root holds %v, want %v", got, want)
		}
	})
}
