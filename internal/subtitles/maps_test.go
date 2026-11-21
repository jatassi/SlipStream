package subtitles_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media/speech"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// maps is each file's map state as "path state fault".
func maps(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	return query(t, w, `SELECT f.path || ' ' || s.map_state || coalesce(' ' || s.map_fault, '') FROM subtitles_file s
		JOIN files_file f ON f.id = s.file_id ORDER BY f.path`)
}

func english(t *testing.T) targets.WantedLanguage {
	t.Helper()
	return subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always)
}

func TestMapsBuildInRankOrderNewestFirstOneAtATime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		w.Maps.Hold()
		w.Place(w.Work("a", p), "A/A.mkv", audio("fr"))
		w.Sweep()
		var fresh files.ID
		for _, name := range []string{"D", "B", "C"} {
			f := w.Place(w.Work(works.Key(name), p), name+"/"+name+".mkv", audio("fr"))
			if name == "B" {
				fresh = f
			}
		}
		w.Sweep()
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return w.Subs.Landed(ctx, tx, files.File{ID: fresh})
		})
		time.Sleep(2 * storing.DefaultSidecarTimings().BuildWait)
		w.Settle()
		if got := w.Maps.Started(); !slices.Equal(got, []string{"A.mkv"}) {
			t.Fatalf("while A's build outlived its call bound, builds %v started, want A's alone in the one slot", got)
		}
		for range 3 {
			w.Maps.Release()
			w.Settle()
		}
		w.Maps.Unhold()
		w.Settle()
		if got, want := w.Maps.Started(), []string{"A.mkv", "B.mkv", "C.mkv", "D.mkv"}; !slices.Equal(got, want) {
			t.Fatalf("builds started %v, want %v: the fresh import, then the newest", got, want)
		}
		if got, want := maps(t, w), []string{"A/A.mkv built", "B/B.mkv built", "C/C.mkv built", "D/D.mkv built"}; !slices.Equal(got, want) {
			t.Fatalf("maps read %v, want %v", got, want)
		}
	})
}

func TestAMapIsDueOnlyWhileAWantIsUnmet(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		w.Put("Met/Met.en.srt", "english")
		w.Place(w.Work("met", p), "Met/Met.mkv", audio("fr"))
		w.Sweep()
		w.Place(w.Work("spoken", w.Profile("Spoken", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.NoAudio))), "Spoken/Spoken.mkv", audio("en"))
		w.Sweep()
		if got, want := maps(t, w), []string{"Met/Met.mkv idle", "Spoken/Spoken.mkv idle"}; !slices.Equal(got, want) {
			t.Fatalf("maps read %v, want %v: a found subtitle and English audio leave nothing to time", got, want)
		}
	})
}

func TestMapFaults(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, func(tm *subtitles.Timings) {
			tm.MapRetryMin, tm.MapRetryMax, tm.NodeRetry = time.Hour, time.Hour, time.Hour
		})
		p := w.Profile("Films", english(t))
		w.Maps.Fail("Silent.mkv", speech.ErrNoAudio)
		w.Maps.Fail("Broken.mkv", &speech.DecodeError{First: "Invalid data found when processing input"})
		w.Place(w.Work("silent", p), "Silent/Silent.mkv", audio("fr"))
		w.Place(w.Work("broken", p), "Broken/Broken.mkv", audio("fr"))
		w.Sweep()
		if got, want := maps(t, w), []string{"Broken/Broken.mkv failed decode_failed", "Silent/Silent.mkv failed no_audio"}; !slices.Equal(got, want) {
			t.Fatalf("maps read %v, want %v", got, want)
		}
		w.Maps.Fail("Broken.mkv", nil)
		w.Maps.Fail("Silent.mkv", nil)
		time.Sleep(time.Hour)
		w.Sweep()
		if got, want := maps(t, w), []string{"Broken/Broken.mkv built", "Silent/Silent.mkv failed no_audio"}; !slices.Equal(got, want) {
			t.Fatalf("an hour on maps read %v, want %v: a decode failure retries, no audio holds", got, want)
		}

		w.Maps.Fail("Faulty.mkv", speech.ErrNodeFault)
		w.Place(w.Work("faulty", p), "Faulty/Faulty.mkv", audio("fr"))
		w.Sweep()
		if got := maps(t, w); !slices.Contains(got, "Faulty/Faulty.mkv due") {
			t.Fatalf("maps read %v, want the map of a Node that can't scan still due", got)
		}
		if got := query(t, w, `SELECT fault_words FROM subtitles_node WHERE fault_at IS NOT NULL`); len(got) != 1 {
			t.Fatalf("the Node's fault rows are %v, want one for Health", got)
		}
		w.Maps.Fail("Faulty.mkv", nil)
		time.Sleep(time.Hour)
		w.Sweep()
		if got := maps(t, w); !slices.Contains(got, "Faulty/Faulty.mkv built") {
			t.Fatalf("maps read %v, want Faulty built once the Node scans again", got)
		}
		if got := query(t, w, `SELECT node_id FROM subtitles_node WHERE fault_at IS NOT NULL`); len(got) != 0 {
			t.Fatalf("the Node's fault row stayed after a build passed")
		}
	})
}
