package subtitles_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// sidecars is each Sidecar row as "tail source final".
func sidecars(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	return query(t, w, `SELECT tail || ' ' || source || ' ' || final FROM subtitles_sidecar ORDER BY tail`)
}

func removals(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	return query(t, w, `SELECT path || ' ' || why FROM subtitles_removal ORDER BY path`)
}

func TestASurveyAdoptsTheFoundCompanionsTheWantsPrefer(t *testing.T) {
	for _, c := range []struct {
		name string
		hi   targets.HI
		want []string
	}{
		{"hearing-impaired preferred", targets.Prefer, []string{"en.hi.srt found 1", "hi.srt found 1"}},
		{"hearing-impaired avoided", targets.Avoid, []string{"en.srt found 1", "hi.srt found 1"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := world(t, nil)
				p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, c.hi, targets.Always), subtitlestest.Wanted(t, "hi", false, targets.Allow, targets.Always))
				for _, tail := range []string{"en.srt", "en.hi.srt", "hi.srt", "nl.srt"} {
					w.Put("Heat/Heat."+tail, "subtitle "+tail)
				}
				f := w.Place(w.Work("heat", p), "Heat/Heat.mkv", audio("en"))
				w.Sweep()
				if got := sidecars(t, w); !slices.Equal(got, c.want) {
					t.Fatalf("the survey adopted %v, want %v: .hi.srt alone is Hindi", got, c.want)
				}
				if got := wantRows(t, w); !slices.Equal(got, []string{"Heat/Heat.mkv en", "Heat/Heat.mkv hi"}) {
					t.Fatalf("want rows %v, want English and Hindi, both met", got)
				}
				extras := query(t, w, `SELECT companions FROM subtitles_file WHERE file_id = ?`, f)
				if len(extras) != 1 || extras[0] == "" {
					t.Fatalf("the survey kept no extras for the companions nothing wants")
				}
			})
		})
	}
}

func TestAGainedLanguageAdoptsTheCompanionTheLastSurveyListed(t *testing.T) {
	for _, c := range []struct {
		name string
		hi   targets.HI
		want string
	}{
		{"hearing-impaired preferred", targets.Prefer, "en.hi.srt found 1"},
		{"hearing-impaired avoided", targets.Avoid, "en.srt found 1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) { gainAfterSurvey(t, c.hi, c.want) })
		})
	}
}

// gainAfterSurvey surveys Copper Kettle with .en.srt and .en.hi.srt beside
// it while nothing is wanted, then gains English with hi, and checks the
// Sidecar is want.
func gainAfterSurvey(t *testing.T, hi targets.HI, want string) {
	w := world(t, nil)
	p := w.Profile("Films")
	for _, tail := range []string{"en.srt", "en.hi.srt"} {
		w.Put("Copper Kettle (2012)/Copper Kettle (2012)."+tail, "subtitle "+tail)
	}
	f := w.Place(w.Work("copper", p), "Copper Kettle (2012)/Copper Kettle (2012).mkv", audio("fr"))
	w.Sweep()
	english := subtitlestest.Wanted(t, "en", false, hi, targets.Always)
	preview := w.Preview(p, "Films", english)
	if g := preview.Subtitles.Gained; len(g) != 1 || g[0].Files != 0 {
		t.Fatalf("gaining English previews %+v, want no search: a subtitle beside the file meets it", g)
	}
	w.Languages(p, "Films", english)
	if got := sidecars(t, w); !slices.Equal(got, []string{want}) {
		t.Fatalf("after gaining English the Sidecars are %v, want %s", got, want)
	}
	st := outcome(t, w, f, subtitles.Want{Language: english.Language})
	if st.Kind != subtitles.Met || st.Sidecar.Source != subtitles.Found || !st.Sidecar.Final {
		t.Fatalf("English reads %+v, want met by the found subtitle", st)
	}
	if got := query(t, w, `SELECT count(*) FROM subtitles_want WHERE unmet = 1`); got[0] != "0" {
		t.Fatalf("%s want rows are unmet, want none", got[0])
	}
}

// adopted is a World whose one file at Heat/Heat.mkv has a found English
// Sidecar.
func adopted(t *testing.T) (w *subtitlestest.World, f files.ID) {
	t.Helper()
	w = world(t, nil)
	p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
	w.Put("Heat/Heat.en.srt", "english")
	f = w.Place(w.Work("heat", p), "Heat/Heat.mkv", audio("en"))
	w.Sweep()
	if got := sidecars(t, w); !slices.Equal(got, []string{"en.srt found 1"}) {
		t.Fatalf("the survey adopted %v", got)
	}
	return w, f
}

func TestADeleteRemovesTheSidecarsBytesWhereverTheyMaySit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, f := adopted(t)
		err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return w.Files.DoomIn(ctx, tx, []works.WorkID{workOf(t, w, f)})
		})
		if err != nil {
			t.Fatal(err)
		}
		holding := strings.TrimSuffix(files.HoldingOf(f.UUID, ".mkv"), ".mkv") + ".en.srt"
		if got, want := removals(t, w), []string{holding + " deleted", "Heat/Heat.en.srt deleted"}; !slices.Equal(got, want) {
			t.Fatalf("the delete wrote removals %v, want %v", got, want)
		}
		w.Settle()
		if got := w.On("Heat/Heat.en.srt"); got != "" {
			t.Fatalf("the Sidecar still holds %q after its file's delete", got)
		}
		if got := removals(t, w); len(got) != 0 {
			t.Fatalf("removals %v remain after the Keeper ran", got)
		}
	})
}

func TestARemovalThatMeetsOtherBytesKeepsThem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, f := adopted(t)
		w.Put("Heat/Heat.en.srt", "the Admin's own")
		w.Doom(workTree(t, w, f))
		if got := w.On("Heat/Heat.en.srt"); got != "the Admin's own" {
			t.Fatalf("the removal left %q, want the bytes it didn't write kept", got)
		}
		if got := removals(t, w); len(got) != 0 {
			t.Fatalf("removals %v remain after meeting other bytes, want them done", got)
		}
	})
}

func TestALeavingFileParksItsSidecarsForItsReplacer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, d := adopted(t)
		p := w.Profile("Others")
		by := w.Place(w.Work("by", p), "By/By.mkv", audio("en"))
		then := w.Place(w.Work("then", p), "Then/Then.mkv", audio("en"))
		parked := func() []string {
			return query(t, w, `SELECT p.what || ' ' || p.path || ' for ' || f.path FROM subtitles_parked p JOIN files_file f ON f.id = p.to_file ORDER BY p.path`)
		}
		leave := func(d files.ID, path string, by files.ID) {
			err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return doom(ctx, tx, w, d, path, nil, by)
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		leave(d, "Heat/Heat.mkv", by)
		if got, want := parked(), []string{"sidecar Heat/Heat.en.srt for By/By.mkv"}; !slices.Equal(got, want) {
			t.Fatalf("leaving for By parked %v, want %v", got, want)
		}
		holding := strings.TrimSuffix(files.HoldingOf(d.UUID, ".mkv"), ".mkv") + ".en.srt"
		if got, want := removals(t, w), []string{holding + " superseded"}; !slices.Equal(got, want) {
			t.Fatalf("leaving for By wrote removals %v, want only the holding stem's", got)
		}
		w.Settle()
		if got := w.On("Heat/Heat.en.srt"); got != "english" {
			t.Fatalf("the parked Sidecar holds %q, want it kept for By to claim", got)
		}
		leave(by, "By/By.mkv", then)
		if got, want := parked(), []string{"sidecar Heat/Heat.en.srt for Then/Then.mkv"}; !slices.Equal(got, want) {
			t.Fatalf("By leaving for Then left parked %v, want %v", got, want)
		}
	})
}

func TestRemovalRowsAreWrittenOnce(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, d := adopted(t)
		for range 2 {
			err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return doom(ctx, tx, w, d, "Heat/Heat.mkv", new("Moving/Heat.mkv"), files.ID{})
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		holding := strings.TrimSuffix(files.HoldingOf(d.UUID, ".mkv"), ".mkv") + ".en.srt"
		if got, want := removals(t, w), []string{holding + " deleted", "Heat/Heat.en.srt deleted", "Moving/Heat.en.srt deleted"}; !slices.Equal(got, want) {
			t.Fatalf("leaving twice wrote removals %v, want the path's, the holding stem's and the open move's once, %v", got, want)
		}
	})
}

func TestNothingMovesWhileTheControllerReconciles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w, f := adopted(t)
		w.Restoring(true)
		w.Doom(workTree(t, w, f))
		w.Sweep()
		if got := w.On("Heat/Heat.en.srt"); got != "english" {
			t.Fatalf("while Reconciling the Sidecar holds %q, want it kept", got)
		}
		if got := w.Reconcile(); got.Chunks != 0 {
			t.Fatalf("while Reconciling the bulk pass covered %+v, want nothing", got)
		}
		w.Restoring(false)
		w.Sweep()
		if got := w.On("Heat/Heat.en.srt"); got != "" {
			t.Fatalf("after Reconciling the Sidecar holds %q, want it removed", got)
		}
	})
}

// doom is what files' doom Write does around Leaving: d's doomed row,
// sitting at sits when a move is open, then the hook.
func doom(ctx context.Context, tx *storage.Tx[storage.Library], w *subtitlestest.World, d files.ID, path string, sits *string, by files.ID) error {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO files_doomed (file_id, root_id, path, sits) VALUES (?, ?, ?, ?)`, d, w.Root.ID, path, sits)
	if err != nil {
		return err
	}
	return w.Subs.Leaving(ctx, tx, files.File{ID: d, RootID: w.Root.ID, Path: path}, by)
}

func workTree(t *testing.T, w *subtitlestest.World, f files.ID) works.Tree {
	t.Helper()
	return works.Tree{Summary: works.Summary{ID: workOf(t, w, f)}}
}

func workOf(t *testing.T, w *subtitlestest.World, f files.ID) works.WorkID {
	t.Helper()
	var id works.WorkID
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		return r.QueryRowContext(ctx, `SELECT work_id FROM files_file WHERE id = ?`, f).Scan(&id)
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}
