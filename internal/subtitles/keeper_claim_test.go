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
)

// replace is an Import's doom of d at path for by: Leaving parks d's
// Sidecars and intents, then d's row goes.
func replace(t *testing.T, w *subtitlestest.World, d files.ID, path string, by files.ID) {
	t.Helper()
	w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		if err := doom(ctx, tx, w, d, path, nil, by); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `DELETE FROM files_file WHERE id = ?`, d)
		return err
	})
}

func short(f files.ID) string { return strings.ReplaceAll(f.String(), "-", "")[:8] }

func TestAReplacerClaimsTheBestFetchedSubtitleAndKeepsTheAdminsOwn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		w.Put("Four/Four.en.vtt", "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nFound.\n")
		by := w.Place(w.Work("by", p), "By/By.mkv", audio("fr"))
		one := w.Place(w.Work("one", p), "One/One.mkv", audio("fr"))
		two := w.Place(w.Work("two", p), "Two/Two.mkv", audio("fr"))
		three := w.Place(w.Work("three", p), "Three/Three.mkv", audio("fr"))
		four := w.Place(w.Work("four", p), "Four/Four.mkv", audio("fr"))
		w.Sweep()
		intend(t, w, one, subtitles.FromProvider, onSpeech)
		intend(t, w, two, subtitles.FromProvider, againstSpeech)
		intend(t, w, three, subtitles.FromUpload, againstSpeech)
		w.Settle()
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE subtitles_sidecar SET match = 70 WHERE file_id = ?`, two)
			return err
		})
		for _, d := range []struct {
			f    files.ID
			path string
		}{{two, "Two/Two.mkv"}, {one, "One/One.mkv"}, {three, "Three/Three.mkv"}, {four, "Four/Four.mkv"}} {
			replace(t, w, d.f, d.path, by)
		}
		w.Sweep()
		w.Sweep()
		if got, want := rows(t, w), []string{"By/By.mkv en provider"}; !slices.Equal(got, want) {
			t.Fatalf("the replacer's Sidecars read %v, want %v: the better fetched one, timed again and home", got, want)
		}
		if got := w.On("By/By.en.srt"); got == "" || got == onSpeech {
			t.Fatalf("By/By.en.srt holds %q, want One's subtitle retimed to By's speech", got)
		}
		for path, want := range map[string]string{
			"One/One.en.srt": "", "Two/Two.en.srt": "", "Three/Three.en.srt": "",
			"By/By." + short(three) + ".en.srt": againstSpeech,
			"Four/Four.en.vtt":                  "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nFound.\n",
		} {
			if got := w.On(path); got != want {
				t.Errorf("%s holds %q, want %q", path, got, want)
			}
		}
		if got := query(t, w, `SELECT (SELECT count(*) FROM subtitles_parked) || ' ' || (SELECT count(*) FROM subtitles_removal)`); got[0] != "0 0" {
			t.Fatalf("parked and removal rows left read %s, want none", got[0])
		}
	})
}

func TestAWriteInFlightWhenItsFileIsReplacedReachesTheReplacer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		by := w.Place(w.Work("by", p), "By/By.mkv", audio("fr"))
		d := w.Place(w.Work("d", p), "D/D.mkv", audio("fr"))
		w.Sweep()
		w.Writes.Hold()
		intend(t, w, d, subtitles.FromProvider, onSpeech)
		w.Settle()
		replace(t, w, d, "D/D.mkv", by)
		w.Writes.Let(true, true)
		w.Sweep()
		w.Sweep()
		if got, want := rows(t, w), []string{"By/By.mkv en provider"}; !slices.Equal(got, want) {
			t.Fatalf("the replacer's Sidecars read %v, want %v: the write that landed, claimed and timed", got, want)
		}
		if got := w.On("D/D.en.srt"); got != "" {
			t.Fatalf("D/D.en.srt holds %q, want it brought home to By", got)
		}
	})
}
