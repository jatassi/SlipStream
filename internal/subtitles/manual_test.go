package subtitles_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

func marks(t *testing.T, w *subtitlestest.World) map[works.WorkID]subtitles.LibraryMark {
	t.Helper()
	var out map[works.WorkID]subtitles.LibraryMark
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		out, err = w.Subs.LibraryRows(ctx, r)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTheLibraryMarksOnlyWorksWithAnUnmetWant(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		withEnglish := audio("fr")
		withEnglish.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "en"}}
		missing := w.Work("missing", p)
		w.Place(missing, "Missing/Missing.mkv", audio("fr"))
		failed := w.Work("failed", p)
		failedFile := w.Place(failed, "Failed/Failed.mkv", audio("fr"))
		met := w.Work("met", p)
		w.Place(met, "Met/Met.mkv", withEnglish)
		exec(t, w, `UPDATE subtitles_want SET last_result = 'none_passed' WHERE file_id = ?`, failedFile)
		got := marks(t, w)
		want := map[works.WorkID]subtitles.LibraryMark{missing.ID: subtitles.MarkMissing, failed.ID: subtitles.MarkNonePassed}
		if len(got) != len(want) || got[missing.ID] != want[missing.ID] || got[failed.ID] != want[failed.ID] {
			t.Fatalf("the Library marks are %v, want %v", got, want)
		}
	})
}

func ignore(t *testing.T, w *subtitlestest.World, f files.ID, ordinal int, on bool) {
	t.Helper()
	if err := w.Subs.IgnoreTrack(t.Context(), identity.Admin{}, f, ordinal, on); err != nil {
		t.Fatal(err)
	}
	w.Settle()
}

// ignoredOnly is whether view's one file has its one want missing, with
// its text track 1 ignored and no track meeting it.
func ignoredOnly(view subtitles.WorkView) bool {
	if len(view.Files) != 1 || len(view.Files[0].Wants) != 1 {
		return false
	}
	f := view.Files[0]
	return f.Wants[0].State.Kind == subtitles.Missing && f.Embedded[1].Ignored && f.Embedded[1].Meets == nil && f.Embedded[0].Meets == nil
}

func TestIgnoringATrackStoresItsFingerprintAndMakesTheWantUnmet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		m := audio("fr")
		m.Subtitles = []media.Subtitle{{Codec: "hdmv_pgs_subtitle", Image: true, Language: "en"}, {Codec: "subrip", Language: "en"}}
		tree := w.Work("film", p)
		f := w.Place(tree, "Film/Film.mkv", m)
		en := subtitles.Want{Language: subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always).Language}
		if st := outcome(t, w, f, en); st.Kind != subtitles.Embedded || st.Track != 1 {
			t.Fatalf("before the ignore English reads %+v, want Embedded by track 1", st)
		}
		ignore(t, w, f, 1, true)
		if got := query(t, w, `SELECT stream_index || ' ' || codec || ' ' || language || ' ' || forced FROM subtitles_track_ignored`); !slices.Equal(got, []string{"1 subrip en 0"}) {
			t.Fatalf("the ignore rows are %v", got)
		}
		if got := query(t, w, `SELECT language || ' ' || unmet FROM subtitles_want`); !slices.Equal(got, []string{"en 1"}) {
			t.Fatalf("after the ignore the want rows are %v, want English unmet", got)
		}
		view, err := w.Subs.WorkSubtitles(t.Context(), identity.Admin{}, tree.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !ignoredOnly(view) {
			t.Fatalf("the Work's subtitles read %+v", view)
		}
		if changed := w.Changed.All(); len(changed) != 1 || changed[0].WorkIDs[0] != tree.ID.String() || changed[0].FileIDs[0] != f.String() {
			t.Fatalf("the ignore published %+v", changed)
		}
		ignore(t, w, f, 1, false)
		if got := query(t, w, `SELECT language FROM subtitles_want`); len(got) != 0 {
			t.Fatalf("counting the track again leaves want rows %v", got)
		}
		if err := w.Subs.IgnoreTrack(t.Context(), identity.Admin{}, f, 0, true); !errors.Is(err, subtitles.ErrNoTrack) {
			t.Fatalf("ignoring an image track = %v, want ErrNoTrack", err)
		}
	})
}

func TestTheManualActionsWaitForSubtitleSearch(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		admin := identity.Admin{}
		var key subtitles.WantKey
		_, err := w.Subs.Upload(t.Context(), admin, key, "Film.en.srt", make([]byte, w.Timings.UploadCap+1))
		var large *subtitles.TooLargeError
		if !errors.As(err, &large) || large.Words() != "Use a subtitle file of 5 MB or smaller." {
			t.Fatalf("an upload over the cap = %v, want the 5 MB refusal", err)
		}
		if _, err := w.Subs.Upload(t.Context(), admin, key, "Film.en.srt", []byte("1")); !errors.Is(err, subtitles.ErrNotBuilt) {
			t.Fatalf("an upload = %v, want ErrNotBuilt", err)
		}
		for name, err := range map[string]error{
			"search":   w.Subs.SearchNowFor(t.Context(), admin, key),
			"mark bad": w.Subs.MarkBadNow(t.Context(), admin, key),
			"keep":     w.Subs.KeepUpload(t.Context(), admin, key.File, storage.UUID{}),
		} {
			if !errors.Is(err, subtitles.ErrNotBuilt) {
				t.Errorf("%s = %v, want ErrNotBuilt", name, err)
			}
		}
	})
}
