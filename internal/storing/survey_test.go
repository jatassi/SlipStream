package storing

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"

	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
)

func occupants(s Surveyed) []string {
	var out []string
	for _, o := range s.Occupants {
		out = append(out, o.Path.String())
	}
	return out
}

func TestSurveyAnswersTheFilesAtEachTarget(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Movies/heat.mkv", "film")
		w.file(t, "Movies/Ronin.mkv", "another film")
		planned := []Planned{
			{From: mustRel(t, "Movies/heat.mkv"), To: mustRel(t, "Movies/RONIN.mkv")},
			{From: mustRel(t, "Movies/heat.mkv"), To: mustRel(t, "Movies/Heat (1995).mkv")},
			{From: mustRel(t, "Movies/heat.mkv"), To: mustRel(t, "Gone/Heat.mkv")},
		}
		got, err := w.remote.Survey(t.Context(), w.at, planned, false)
		if err != nil {
			t.Fatal(err)
		}
		for i, want := range [][]string{{"Movies/Ronin.mkv"}, nil, nil} {
			if !slices.Equal(occupants(got[i]), want) {
				t.Errorf("%s holds %v, want %v", planned[i].To, occupants(got[i]), want)
			}
		}
		walked := w.pages(t, 1000)[0].Entries
		ronin := walked[slices.IndexFunc(walked, func(e Walked) bool { return e.Path.String() == "Movies/Ronin.mkv" })]
		if seen := got[0].Occupants[0].Seen; !seen.Same(ronin.Seen) {
			t.Errorf("the file at RONIN.mkv reads %+v, want the walk's %+v", seen, ronin.Seen)
		}
	})
}

func TestSurveyFindsTheSourceUnderACaseAlias(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.disk.foldCase = true
		w.file(t, "Movies/heat.mkv", "film")
		got, err := w.remote.Survey(t.Context(), w.at, []Planned{{From: mustRel(t, "Movies/heat.mkv"), To: mustRel(t, "Movies/Heat.mkv")}}, false)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"Movies/heat.mkv"}; !slices.Equal(occupants(got[0]), want) {
			t.Fatalf("Heat.mkv holds %v, want %v", occupants(got[0]), want)
		}
		source := w.pages(t, 1000)[0].Entries[0]
		if seen := got[0].Occupants[0].Seen; !seen.Same(source.Seen) {
			t.Errorf("the file at Heat.mkv reads %+v, want the source's %+v", seen, source.Seen)
		}
	})
}

func TestSurveyRefusesMoreThanOneBatch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		req := &storagev1.SurveyRequest{RootId: w.at.Root.Bytes()}
		for range surveyBatch + 1 {
			req.Planned = append(req.Planned, &storagev1.Planned{From: "a.mkv", To: "b.mkv"})
		}
		if _, err := w.store.Survey(t.Context(), req); err == nil {
			t.Errorf("a Survey of %d renames was answered", len(req.Planned))
		}
		planned := make([]Planned, surveyBatch+1)
		for i := range planned {
			planned[i] = Planned{From: mustRel(t, "a.mkv"), To: mustRel(t, "b.mkv")}
		}
		if got, err := w.remote.Survey(t.Context(), w.at, planned, false); err != nil || len(got) != len(planned) {
			t.Errorf("Remote.Survey of %d = %d answers, %v; want one each, in batches", len(planned), len(got), err)
		}
	})
}

func TestPruneRemovesEmptyFoldersUpToOneThatHoldsAFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Movies/notes.txt", "notes")
		dir(t, filepath.Join(w.folder, "Movies", "Heat", "Extras"))
		for _, want := range [][]string{{"Movies/Heat/Extras", "Movies/Heat"}, nil} {
			removed, err := w.remote.Prune(t.Context(), w.at, mustRel(t, "Movies/Heat/Extras"), nil)
			if err != nil || !slices.Equal(removed, rels(t, want...)) {
				t.Errorf("Prune = %v, %v; want %v", removed, err, want)
			}
		}
		if _, err := os.Stat(filepath.Join(w.folder, "Movies", "notes.txt")); err != nil {
			t.Errorf("the folder that holds a file went: %v", err)
		}
	})
}

func TestPruneStopsAtTheRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		dir(t, filepath.Join(w.folder, "Heat"))
		removed, err := w.remote.Prune(t.Context(), w.at, mustRel(t, "Heat"), nil)
		if err != nil || !slices.Equal(removed, rels(t, "Heat")) {
			t.Errorf("Prune = %v, %v; want [Heat]", removed, err)
		}
		if _, err := os.Stat(filepath.Join(w.folder, MarkerName)); err != nil {
			t.Errorf("the root went: %v", err)
		}
	})
}

func TestPruneKeepsHeldAndKeptFolders(t *testing.T) {
	for _, c := range []struct {
		name, folder string
		keep         []string
		removed      []string
	}{
		{"holding", HoldingDir, nil, nil},
		{"a staging folder", StagingDir + "/0190", nil, nil},
		{"a kept folder", "Movies/Heat", []string{"movies/heat"}, nil},
		{"a parent of a kept folder", "Movies/Heat", []string{"Movies/Heat (1995)"}, []string{"Movies/Heat"}},
	} {
		synctest.Test(t, func(t *testing.T) {
			w := newWalking(t, nil)
			dir(t, filepath.Join(w.folder, filepath.FromSlash(c.folder)))
			removed, err := w.remote.Prune(t.Context(), w.at, mustRel(t, c.folder), rels(t, c.keep...))
			if err != nil || !slices.Equal(removed, rels(t, c.removed...)) {
				t.Errorf("%s: Prune = %v, %v; want %v", c.name, removed, err, c.removed)
			}
		})
	}
}

func TestSurveyPruneAndMoveWaitForAnUnmountedRoot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := newWalking(t, nil)
		w.file(t, "Heat/Heat.mkv", "film")
		w.file(t, "Heat/Heat.en.srt", "subs")
		dir(t, filepath.Join(w.folder, "Empty"))
		if err := os.Remove(filepath.Join(w.folder, MarkerName)); err != nil {
			t.Fatal(err)
		}
		from, to := mustRel(t, "Heat/Heat.mkv"), mustRel(t, "Heat (1995)/Heat (1995).mkv")
		_, err := w.remote.Survey(t.Context(), w.at, []Planned{{From: from, To: to}}, true)
		if nr := notReady(t, err); nr.State != Unmounted {
			t.Errorf("Survey = %+v, want unmounted", nr)
		}
		_, err = w.remote.Prune(t.Context(), w.at, mustRel(t, "Empty"), nil)
		if nr := notReady(t, err); nr.State != Unmounted {
			t.Errorf("Prune = %+v, want unmounted", nr)
		}
		_, err = w.remote.Move(t.Context(), w.at, from, to, true)
		if nr := notReady(t, err); nr.State != Unmounted {
			t.Errorf("Move = %+v, want unmounted", nr)
		}
		if got, want := sortedTree(t, w.folder), []string{"Heat/Heat.en.srt", "Heat/Heat.mkv"}; !slices.Equal(got, want) {
			t.Errorf("the folder holds %v, want %v untouched", got, want)
		}
		if _, err := os.Stat(filepath.Join(w.folder, "Empty")); err != nil {
			t.Errorf("an empty folder went from an unmounted root: %v", err)
		}
	})
}
