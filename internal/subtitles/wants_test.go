package subtitles_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/subtitling"
	"github.com/jatassi/slipstream/internal/targets"
)

// world is a subtitlestest World on the default timings, tune's changes
// made, that checks the records invariant after every step.
func world(t *testing.T, tune func(*subtitles.Timings)) *subtitlestest.World {
	t.Helper()
	tm := subtitles.DefaultTimings()
	if tune != nil {
		tune(&tm)
	}
	w := subtitlestest.Open(t, tm, subtitling.DefaultTimings())
	w.Check = func(t testing.TB, w *subtitlestest.World) {
		t.Helper()
		err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
			return subtitles.Audit(ctx, r, w.Subs)
		})
		if err != nil {
			t.Fatalf("the records invariant breaks: %v", err)
		}
	}
	w.Check(t, w)
	return w
}

func outcome(t *testing.T, w *subtitlestest.World, f files.ID, want subtitles.Want) subtitles.State {
	t.Helper()
	var st subtitles.State
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		st, err = w.Subs.Outcome(ctx, r, f, want)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return st
}

// wantRows is each want row as "path want".
func wantRows(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	return query(t, w, `SELECT f.path || ' ' || w.language || CASE w.forced WHEN 1 THEN '-forced' ELSE '' END FROM subtitles_want w
		JOIN files_file f ON f.id = w.file_id ORDER BY f.path, w.language, w.forced`)
}

func query(t *testing.T, w *subtitlestest.World, q string, args ...any) []string {
	t.Helper()
	var out []string
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := r.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				return errors.Join(fmt.Errorf("scan %s: %w", q, err), rows.Close())
			}
			out = append(out, s)
		}
		return errors.Join(rows.Err(), rows.Close())
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func audio(langs ...release.Language) media.Record {
	m := media.Record{Video: media.Video{Width: 1920, Height: 1080}}
	for _, l := range langs {
		m.Audio = append(m.Audio, media.Audio{Codec: "eac3", Channels: 6, Language: l})
	}
	return m
}

func TestAGainedLanguageWantsOnlyWhereNothingMeetsIt(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films")
		english := audio("en")
		withFrench := audio("en")
		withFrench.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "fr"}}
		withImage := audio("en")
		withImage.Subtitles = []media.Subtitle{{Codec: "hdmv_pgs_subtitle", Image: true, Language: "fr"}}
		placed := map[string]files.ID{
			"plain":    w.Place(w.Work("plain", p), "Plain/Plain.mkv", english),
			"embedded": w.Place(w.Work("embedded", p), "Embedded/Embedded.mkv", withFrench),
			"image":    w.Place(w.Work("image", p), "Image/Image.mkv", withImage),
			"french":   w.Place(w.Work("french", p), "French/French.mkv", audio("fr")),
		}
		if got := wantRows(t, w); len(got) != 0 {
			t.Fatalf("a Profile with no Wanted Language has want rows %v", got)
		}
		french := subtitlestest.Wanted(t, "fr", false, targets.Allow, targets.NoAudio)
		w.Languages(p, "Films", french)
		w.Reconcile()
		if got, want := wantRows(t, w), []string{"Image/Image.mkv fr", "Plain/Plain.mkv fr"}; !slices.Equal(got, want) {
			t.Fatalf("after French is wanted, the want rows are %v, want %v", got, want)
		}
		fr := subtitles.Want{Language: french.Language}
		for _, c := range []struct {
			file string
			kind subtitles.StateKind
			why  subtitles.NotNeededWhy
		}{
			{"plain", subtitles.Missing, 0},
			{"image", subtitles.Missing, 0},
			{"embedded", subtitles.Embedded, 0},
			{"french", subtitles.NotNeeded, subtitles.AudioInLanguage},
		} {
			st := outcome(t, w, placed[c.file], fr)
			if st.Kind != c.kind || st.Why != c.why {
				t.Errorf("%s reads %+v, want kind %d why %d", c.file, st, c.kind, c.why)
			}
		}
	})
}
