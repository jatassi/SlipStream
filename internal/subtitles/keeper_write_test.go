package subtitles_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry/entrytest"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storing"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/targets"
)

// onSpeech is a subtitle whose one cue sits half a second after the
// World's speech, so a retime passes.
const onSpeech = "1\n00:00:01,500 --> 00:00:02,500\nHello there.\n"

// againstSpeech is a subtitle no retime fits to the World's speech.
const againstSpeech = "1\n00:00:00,000 --> 00:00:00,400\nOne.\n\n2\n00:00:03,500 --> 00:00:03,900\nTwo.\n"

// rows is each Sidecar row as "video language source", then where it
// strays, whether it waits to be timed and whether it is untimed.
func rows(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	return query(t, w, `SELECT f.path || ' ' || c.language || ' ' || c.source || coalesce(' stray ' || c.stray, '') ||
		CASE WHEN c.timing_due THEN ' due' ELSE '' END || CASE WHEN c.match IS NULL THEN ' untimed' ELSE '' END
		FROM subtitles_sidecar c JOIN files_file f ON f.id = c.file_id ORDER BY f.path, c.language`)
}

func intend(t *testing.T, w *subtitlestest.World, f files.ID, src subtitles.Source, body string) {
	t.Helper()
	if err := subtitles.Intend(t.Context(), w.Subs, f, subtitles.Want{Language: english(t).Language}, src, []byte(body)); err != nil {
		t.Fatal(err)
	}
}

// changes is each Media Server change as "kind path".
func changes(w *subtitlestest.World) []string {
	var out []string
	for _, c := range w.Servers.Changes() {
		out = append(out, kinds[c.Kind]+" "+c.Path.String())
	}
	return out
}

var kinds = map[mediaservers.SidecarKind]string{mediaservers.SidecarAdded: "added", mediaservers.SidecarReplaced: "replaced", mediaservers.SidecarRemoved: "removed"}

func TestAnIntentLandsAsItsWantsSidecarWithOneRowEach(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		f := w.Place(w.Work("a", p), "A/A.mkv", audio("fr"))
		w.Sweep()
		intend(t, w, f, subtitles.FromProvider, onSpeech)
		w.Settle()
		if got := w.On("A/A.en.srt"); got != onSpeech {
			t.Fatalf("A/A.en.srt holds %q, want the intent's bytes", got)
		}
		intend(t, w, f, subtitles.FromUpload, againstSpeech)
		w.Settle()
		if got := w.On("A/A.en.srt"); got != againstSpeech {
			t.Fatalf("A/A.en.srt holds %q, want the replacing intent's bytes", got)
		}
		if got, want := rows(t, w), []string{"A/A.mkv en upload untimed"}; !slices.Equal(got, want) {
			t.Fatalf("Sidecars read %v, want %v", got, want)
		}
		if got, want := changes(w), []string{"added A/A.en.srt", "replaced A/A.en.srt"}; !slices.Equal(got, want) {
			t.Fatalf("Media Servers heard %v, want %v", got, want)
		}
		ids := query(t, w, `SELECT lower(hex(id)) FROM subtitles_sidecar`)
		if got := w.Servers.Changes()[1].Sidecar.String(); len(ids) != 1 || stripDashes(got) != ids[0] {
			t.Fatalf("the replaced change names Sidecar %s, want the row's own id %v", got, ids)
		}
		added, replaced := entrytest.Of(w.History, subtitles.AddedRow), entrytest.Of(w.History, subtitles.ReplacedRow)
		if len(added) != 1 || len(replaced) != 1 || replaced[0].File != "A/A.mkv" || replaced[0].Subtitles[0].Source != subtitles.FromUpload {
			t.Fatalf("History has %v added and %v replaced, want one each, the second for the upload", added, replaced)
		}
	})
}

func stripDashes(s string) string {
	out := make([]byte, 0, len(s))
	for i := range len(s) {
		if s[i] != '-' {
			out = append(out, s[i])
		}
	}
	return string(out)
}

// ended checks the Sidecar rows, what A/A.en.srt holds and what Media
// Servers heard.
func ended(t *testing.T, w *subtitlestest.World, sidecars []string, on string, heard []string) {
	t.Helper()
	if got := rows(t, w); !slices.Equal(got, sidecars) {
		t.Fatalf("Sidecars read %v, want %v", got, sidecars)
	}
	if got := w.On("A/A.en.srt"); got != on {
		t.Fatalf("A/A.en.srt holds %q, want %q", got, on)
	}
	if got := changes(w); !slices.Equal(got, heard) {
		t.Fatalf("Media Servers heard %v, want %v", got, heard)
	}
}

func TestAnIntentSettlesAfterACrashWhereverItsWriteStopped(t *testing.T) {
	for _, c := range []struct {
		name    string
		land    bool
		rows    []string
		on      string
		changes []string
	}{
		{"before the write lands", false, nil, "", nil},
		{"after the write lands, before its finish", true, []string{"A/A.mkv en provider"}, onSpeech, []string{"added A/A.en.srt"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := world(t, nil)
				p := w.Profile("Films", english(t))
				f := w.Place(w.Work("a", p), "A/A.mkv", audio("fr"))
				w.Sweep()
				w.Writes.Hold()
				intend(t, w, f, subtitles.FromProvider, onSpeech)
				w.Settle()
				if c.land {
					w.Writes.Let(true, false)
					w.Settle()
				}
				w.Restart()
				w.Sweep()
				ended(t, w, c.rows, c.on, c.changes)
				if got := query(t, w, `SELECT count(*) FROM subtitles_write`); got[0] != "0" {
					t.Fatalf("%s intents are left, want none", got[0])
				}
				if got, want := len(entrytest.Of(w.History, subtitles.AddedRow)), len(c.changes); got != want {
					t.Fatalf("History has %d Subtitles added, want %d", got, want)
				}
			})
		})
	}
}

func TestARestoredLibraryTakesBackTheSubtitlesTheNodeWroteAsFetchedAndTimesThem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t), subtitlestest.Wanted(t, "fr", false, targets.Allow, targets.Always))
		w.Put("A/A.fr.srt", againstSpeech)
		f := w.Place(w.Work("a", p), "A/A.mkv", audio("de"))
		w.Sweep()
		intend(t, w, f, subtitles.FromProvider, onSpeech)
		w.Settle()
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `DELETE FROM subtitles_sidecar`)
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE subtitles_want SET unmet = 1`)
			}
			if err == nil {
				_, err = tx.ExecContext(ctx, `UPDATE subtitles_file SET survey_due = 1`)
			}
			return err
		})
		w.Restart()
		w.Sweep()
		got := query(t, w, `SELECT tail || ' ' || source || ' ' || final || ' ' || timing_due || ' ' || (match IS NOT NULL) FROM subtitles_sidecar ORDER BY tail`)
		if want := []string{"en.srt provider 0 0 1", "fr.srt found 1 0 0"}; !slices.Equal(got, want) {
			t.Fatalf("after the restore, Sidecars read %v, want %v: the Node's own as fetched, not final and timed again, the rest found", got, want)
		}
	})
}

func TestADoomRacingAWriteLeavesNoBytes(t *testing.T) {
	for _, land := range []bool{false, true} {
		t.Run(map[bool]string{false: "the write never lands", true: "the write lands"}[land], func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := world(t, nil)
				p := w.Profile("Films", english(t))
				tree := w.Work("a", p)
				f := w.Place(tree, "A/A.mkv", audio("fr"))
				w.Sweep()
				w.Writes.Hold()
				intend(t, w, f, subtitles.FromProvider, onSpeech)
				w.Settle()
				w.Doom(tree)
				w.Settle()
				w.Writes.Let(land, land)
				w.Sweep()
				if got := w.On("A/A.en.srt"); got != "" {
					t.Fatalf("A/A.en.srt holds %q after its file's doom, want nothing", got)
				}
				if got := query(t, w, `SELECT (SELECT count(*) FROM subtitles_write) || ' ' || (SELECT count(*) FROM subtitles_removal) || ' ' ||
					(SELECT count(*) FROM subtitles_sidecar)`); got[0] != "0 0 0" {
					t.Fatalf("intents, removals and Sidecars left read %s, want none of each", got[0])
				}
				if got := changes(w); len(got) != 0 {
					t.Fatalf("Media Servers heard %v of a doomed file's subtitle, want nothing", got)
				}
			})
		})
	}
}

// offSpeech is a subtitle that talks the whole time, which no retime
// fits to the World's one second of speech.
const offSpeech = "1\n00:00:00,000 --> 00:00:00,600\nOne.\n\n2\n00:00:00,700 --> 00:00:01,300\nTwo.\n\n3\n00:00:01,400 --> 00:00:02,000\nThree.\n\n" +
	"4\n00:00:02,100 --> 00:00:02,700\nFour.\n\n5\n00:00:02,800 --> 00:00:03,400\nFive.\n\n6\n00:00:03,500 --> 00:00:03,900\nSix.\n"

// replaced is a scan's word that f's bytes changed in place.
func replaced(t *testing.T, w *subtitlestest.World, f files.ID, path string) {
	t.Helper()
	at, err := storing.ParseRel(path)
	if err != nil {
		t.Fatal(err)
	}
	w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.Subs.Changed(ctx, tx, []files.PathChange{{Kind: files.PathReplaced, File: f, Root: w.Root.ID, Path: at}})
	})
}

func TestARetimeThatFailsRemovesAFetchedSubtitleAndKeepsAnUploadUntimed(t *testing.T) {
	for _, c := range []struct {
		name    string
		src     subtitles.Source
		rows    []string
		on      string
		changes []string
		removed int
	}{
		{"fetched", subtitles.FromProvider, nil, "", []string{"added A/A.en.srt", "removed A/A.en.srt"}, 1},
		{"uploaded", subtitles.FromUpload, []string{"A/A.mkv en upload untimed"}, offSpeech, []string{"added A/A.en.srt"}, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := world(t, nil)
				p := w.Profile("Films", english(t))
				f := w.Place(w.Work("a", p), "A/A.mkv", audio("fr"))
				w.Sweep()
				intend(t, w, f, c.src, offSpeech)
				w.Settle()
				replaced(t, w, f, "A/A.mkv")
				w.Sweep()
				w.Sweep()
				ended(t, w, c.rows, c.on, c.changes)
				removed := entrytest.Of(w.History, subtitles.RemovedRow)
				if len(removed) != c.removed || c.removed == 1 && removed[0].Why != "retime_failed" {
					t.Fatalf("History has Subtitles removed %v, want %d for a failed retime", removed, c.removed)
				}
				want := map[int]string{0: "0 0", 1: "1 1"}[c.removed]
				if got := query(t, w, `SELECT unmet || ' ' || coalesce(due_at <= ?, 0) FROM subtitles_want`, storage.FromTime(time.Now())); got[0] != want {
					t.Fatalf("the want reads unmet and due %v, want %s", got, want)
				}
			})
		})
	}
}

func TestAStraySubtitleComesHomeOnlyOnceItsNameIsFree(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		f := w.Place(w.Work("a", p), "A/A.mkv", audio("fr"))
		w.Sweep()
		intend(t, w, f, subtitles.FromUpload, onSpeech)
		w.Settle()
		w.Put("A/A.1a2b3c4d.en.srt", onSpeech)
		w.Put("A/A.en.srt", "found")
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			_, err := tx.ExecContext(ctx, `UPDATE subtitles_file SET survey_due = 1 WHERE file_id = ?`, f)
			return err
		})
		w.Sweep()
		if got, want := rows(t, w), []string{"A/A.mkv en upload stray A/A.1a2b3c4d.en.srt untimed"}; !slices.Equal(got, want) {
			t.Fatalf("Sidecars read %v, want %v", got, want)
		}
		if got := w.On("A/A.en.srt"); got != "found" {
			t.Fatalf("A/A.en.srt holds %q, want the Admin's found subtitle never overwritten", got)
		}
		if err := os.Remove(filepath.Join(w.Root.Path, "A", "A.en.srt")); err != nil {
			t.Fatal(err)
		}
		w.Sweep()
		w.Sweep()
		if got, want := rows(t, w), []string{"A/A.mkv en upload untimed"}; !slices.Equal(got, want) {
			t.Fatalf("once its name was free Sidecars read %v, want %v", got, want)
		}
		if a, b := w.On("A/A.en.srt"), w.On("A/A.1a2b3c4d.en.srt"); a != onSpeech || b != "" {
			t.Fatalf("home holds %q and the stray place %q, want the subtitle home alone", a, b)
		}
		if got, want := changes(w), []string{"added A/A.en.srt", "added A/A.en.srt"}; !slices.Equal(got, want) {
			t.Fatalf("Media Servers heard %v, want %v", got, want)
		}
	})
}

func TestALanguageDropDuringAWriteLeavesNoSidecar(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		f := w.Place(w.Work("a", p), "A/A.mkv", audio("fr"))
		w.Sweep()
		w.Writes.Hold()
		intend(t, w, f, subtitles.FromProvider, onSpeech)
		w.Settle()
		w.Languages(p, "Films")
		w.Writes.Let(true, true)
		w.Sweep()
		if got := rows(t, w); len(got) != 0 {
			t.Fatalf("Sidecars read %v after the drop, want none", got)
		}
		if got := w.On("A/A.en.srt"); got != "" {
			t.Fatalf("A/A.en.srt holds %q after the drop, want the landed bytes removed", got)
		}
		if got := changes(w); len(got) != 0 || w.History.Len() != 0 {
			t.Fatalf("Media Servers heard %v and History kept %d entries, want nothing of a dropped write", got, w.History.Len())
		}
	})
}

func TestARenameDuringASyncTimesTheSubtitleAgainAtTheNewName(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		p := w.Profile("Films", english(t))
		f := w.Place(w.Work("a", p), "A/A.mkv", audio("fr"))
		w.Sweep()
		intend(t, w, f, subtitles.FromProvider, onSpeech)
		w.Settle()
		w.Syncs.Hold()
		replaced(t, w, f, "A/A.mkv")
		w.Sweep()
		if err := os.MkdirAll(filepath.Join(w.Root.Path, "B"), 0o750); err != nil {
			t.Fatal(err)
		}
		for from, to := range map[string]string{"A/A.mkv": "B/B.mkv", "A/A.en.srt": "B/B.en.srt"} {
			if err := os.Rename(filepath.Join(w.Root.Path, filepath.FromSlash(from)), filepath.Join(w.Root.Path, filepath.FromSlash(to))); err != nil {
				t.Fatal(err)
			}
		}
		from, _ := storing.ParseRel("A/A.mkv")
		to, _ := storing.ParseRel("B/B.mkv")
		w.Write(func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			if _, err := tx.ExecContext(ctx, `UPDATE files_file SET path = ? WHERE id = ?`, to.String(), f); err != nil {
				return err
			}
			return w.Subs.Changed(ctx, tx, []files.PathChange{{Kind: files.PathMoved, File: f, Root: w.Root.ID, Path: to, From: from}})
		})
		w.Syncs.Release()
		w.Sweep()
		w.Sweep()
		if got, want := rows(t, w), []string{"B/B.mkv en provider"}; !slices.Equal(got, want) {
			t.Fatalf("Sidecars read %v, want %v: timed again at the new name", got, want)
		}
		if a, b := w.On("A/A.en.srt"), w.On("B/B.en.srt"); a != "" || b == "" || b == onSpeech {
			t.Fatalf("the old name holds %q and the new %q, want the retimed subtitle at the new name alone", a, b)
		}
		if got := entrytest.Of(w.History, subtitles.ReplacedRow); len(got) != 1 || got[0].File != "B/B.mkv" {
			t.Fatalf("History has Subtitle replaced %v, want one, for the file at its new name", got)
		}
	})
}

func TestBytesTheKeeperMovedForTheAdminNeverReadAsFetched(t *testing.T) {
	for _, c := range []struct {
		name  string
		bring func(t *testing.T, w *subtitlestest.World, p targets.ProfileID) string
	}{
		{"found and claimed by an upgrade", func(t *testing.T, w *subtitlestest.World, p targets.ProfileID) string {
			w.Put("Kettle/Kettle.en.srt", againstSpeech)
			f := w.Place(w.Work("kettle", p), "Kettle/Kettle.mkv", audio("fr"))
			w.Sweep()
			up := w.Place(w.Work("upgrade", p), "Kettle/Kettle.2160p.mkv", audio("fr"))
			replace(t, w, f, "Kettle/Kettle.mkv", up)
			if err := os.Remove(filepath.Join(w.Root.Path, "Kettle", "Kettle.mkv")); err != nil {
				t.Fatal(err)
			}
			w.Sweep()
			w.Sweep()
			return "Kettle/Kettle.2160p.en.srt"
		}},
		{"uploaded", func(t *testing.T, w *subtitlestest.World, p targets.ProfileID) string {
			f := w.Place(w.Work("kettle", p), "Kettle/Kettle.mkv", audio("fr"))
			w.Sweep()
			intend(t, w, f, subtitles.FromUpload, againstSpeech)
			w.Settle()
			return "Kettle/Kettle.en.srt"
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				w := world(t, nil)
				p := w.Profile("Films", english(t))
				lanterns := w.Place(w.Work("lanterns", p), "Lanterns/Lanterns.mkv", audio("fr"))
				kettle := w.Profile("Kettle", english(t), subtitlestest.Wanted(t, "ja", false, targets.Allow, targets.Always))
				if at := c.bring(t, w, kettle); w.On(at) != againstSpeech {
					t.Fatalf("%s holds %q, want the Admin's subtitle the Keeper wrote there", at, w.On(at))
				}
				w.Put("Lanterns/Lanterns.de.srt", againstSpeech)
				w.Languages(p, "Films", english(t), subtitlestest.Wanted(t, "de", false, targets.Allow, targets.Always))
				replaced(t, w, lanterns, "Lanterns/Lanterns.mkv")
				w.Sweep()
				w.Sweep()
				got := query(t, w, `SELECT tail || ' ' || source || ' ' || final FROM subtitles_sidecar WHERE file_id = ?`, lanterns)
				if want := []string{"de.srt found 1"}; !slices.Equal(got, want) {
					t.Fatalf("Lanterns' Sidecars read %v, want %v: a copy of the Admin's bytes is the Admin's", got, want)
				}
				if got := w.On("Lanterns/Lanterns.de.srt"); got != againstSpeech {
					t.Fatalf("Lanterns/Lanterns.de.srt holds %q, want the Admin's subtitle kept", got)
				}
				if removed := entrytest.Of(w.History, subtitles.RemovedRow); len(removed) != 0 {
					t.Fatalf("History has Subtitles removed %v, want none", removed)
				}
			})
		})
	}
}
