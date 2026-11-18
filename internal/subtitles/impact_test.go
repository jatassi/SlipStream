package subtitles_test

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// TestImpactCountsTheSearchesTheNextPassMakes pins each gained want's
// Impact to the unmet want rows the bulk pass then writes, across audio
// in and out of the language, embedded tracks, a Kept file, a found
// Sidecar, and subtitles beside a file surveyed before their language was
// wanted.
func TestImpactCountsTheSearchesTheNextPassMakes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		w.Maps.Hold()
		en := func(forced bool, a targets.AudioCondition) targets.WantedLanguage {
			return subtitlestest.Wanted(t, "en", forced, targets.Allow, a)
		}
		p := w.Profile("Films", en(false, targets.Always))
		placeEveryCase(t, w, p)
		err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE files_file SET extra = 1 WHERE path = 'Kept/Kept.mkv'`)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		w.Reconcile()
		w.Sweep()
		if got := sidecars(t, w); !slices.Equal(got, []string{"en.srt found 1"}) {
			t.Fatalf("the found Sidecar wasn't adopted: %v", got)
		}
		w.Languages(p, "Films")
		for _, langs := range [][]targets.WantedLanguage{
			{en(false, targets.Always)},
			{en(false, targets.NoAudio)},
			{en(true, targets.Always)},
			{en(true, targets.NoAudio)},
			{en(false, targets.NoAudio), en(true, targets.Always)},
			{en(false, targets.NoAudio), en(true, targets.NoAudio)},
			{en(false, targets.Always), en(true, targets.Always)},
			{subtitlestest.Wanted(t, "fr", false, targets.Allow, targets.NoAudio)},
		} {
			preview := w.Preview(p, "Films", langs...)
			w.Languages(p, "Films", langs...)
			w.Reconcile()
			if len(preview.Subtitles.Gained) != len(langs) {
				t.Fatalf("%v: Impact gained %+v, want every language", langs, preview.Subtitles.Gained)
			}
			for _, g := range preview.Subtitles.Gained {
				searchesMade(t, w, langs, g)
			}
			w.Languages(p, "Films")
		}
		if got := sidecars(t, w); !slices.Equal(got, []string{"en.forced.srt found 1", "en.srt found 1", "fr.srt found 1"}) {
			t.Fatalf("the gains adopted %v, want the subtitles beside the file that met them", got)
		}
	})
}

// searchesMade checks g's Impact against the unmet want rows the pass
// made for it.
func searchesMade(t *testing.T, w *subtitlestest.World, langs []targets.WantedLanguage, g files.GainedLanguage) {
	t.Helper()
	unmet := query(t, w, `SELECT count(*) FROM subtitles_want WHERE language = ? AND forced = ? AND unmet = 1`, g.Language.String(), g.Forced)
	if want := strconv.Itoa(g.Files); unmet[0] != want {
		t.Errorf("%v: Impact counts %d searches for %s forced=%t, the pass made %s", langs, g.Files, g.Language, g.Forced, unmet[0])
	}
	if g.NeedScan != g.Files {
		t.Errorf("%v: %d of %d searches need a scan, want all: no map is built", langs, g.NeedScan, g.Files)
	}
}

func TestDroppingALanguageRemovesOnlyWhatSlipStreamFetched(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		english := subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always)
		p := w.Profile("Films", english)
		w.Put("Found/Found.en.srt", "found")
		w.Place(w.Work("found", p), "Found/Found.mkv", audio("fr"))
		fetched := w.Place(w.Work("fetched", p), "Fetched/Fetched.mkv", audio("fr"))
		w.Sweep()
		sha := w.Put("Fetched/Fetched.en.srt", "fetched")
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO subtitles_sidecar (id, file_id, language, forced, hi, tail, source, provider, final, timed, sha256, size, cues, added_at)
				VALUES (?, ?, 'en', 0, 0, 'en.srt', 'provider', 'subdl', 1, 0, ?, 7, 1, 1)`, storage.NewUUID(), fetched, sha[:])
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE subtitles_want SET unmet = 0, due_at = NULL WHERE file_id = ?`, fetched)
			return err
		})
		if got := sidecars(t, w); !slices.Equal(got, []string{"en.srt found 1", "en.srt provider 1"}) {
			t.Fatalf("the Sidecars are %v", got)
		}
		preview := w.Preview(p, "Films")
		if len(preview.Subtitles.Dropped) != 1 || preview.Subtitles.Dropped[0].Deleted != 1 {
			t.Fatalf("dropping English previews %+v, want one fetched subtitle deleted", preview.Subtitles.Dropped)
		}
		w.Languages(p, "Films")
		if got := sidecars(t, w); !slices.Equal(got, []string{"en.srt found 1"}) {
			t.Fatalf("after the drop the Sidecars are %v, want the found one kept", got)
		}
		if got, want := w.On("Fetched/Fetched.en.srt")+"|"+w.On("Found/Found.en.srt"), "|found"; got != want {
			t.Fatalf("after the drop the disk holds %q, want %q", got, want)
		}
		if got := wantRows(t, w); len(got) != 0 {
			t.Fatalf("after the drop want rows %v remain", got)
		}
	})
}

// placeEveryCase places a file with English audio, French, untagged,
// both, an English text track, a forced one, a found English Sidecar, one
// with forced English and French subtitles beside it, and one that will
// be Kept.
func placeEveryCase(t *testing.T, w *subtitlestest.World, p targets.ProfileID) {
	t.Helper()
	embedded := audio("fr")
	embedded.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "en"}}
	forcedTrack := audio("fr")
	forcedTrack.Subtitles = []media.Subtitle{{Codec: "subrip", Language: "en", Forced: true}}
	w.Put("Found/Found.en.srt", "found")
	w.Put("Extra/Extra.en.forced.srt", "forced extra")
	w.Put("Extra/Extra.fr.srt", "french extra")
	for _, f := range []struct {
		key string
		m   media.Record
	}{
		{"english", audio("en")}, {"french", audio("fr")}, {"untagged", audio("")}, {"embedded", embedded},
		{"forced", forcedTrack}, {"found", audio("fr")}, {"kept", audio("fr")}, {"both", audio("en", "fr")}, {"extra", audio("en")},
	} {
		name := string([]rune{rune(f.key[0]) - 32}) + f.key[1:]
		w.Place(w.Work(works.Key(f.key), p), name+"/"+name+".mkv", f.m)
	}
}
