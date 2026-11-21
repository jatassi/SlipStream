package subtitles_test

import (
	"context"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles/subtitlestest"
	"github.com/jatassi/slipstream/internal/targets"
)

// items is the Subtitles category's open items as "key level: text".
func items(t *testing.T, w *subtitlestest.World) []string {
	t.Helper()
	a, err := w.Subs.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, it := range a.Items {
		out = append(out, string(it.Key)+" "+it.Level.String()+": "+it.Text)
	}
	slices.Sort(out)
	return out
}

// raises fails unless the open items are exactly want, or hold want when
// some is set.
func raises(t *testing.T, w *subtitlestest.World, when string, some bool, want ...string) {
	t.Helper()
	got := items(t, w)
	ok := slices.Equal(got, want)
	if some {
		ok = !slices.ContainsFunc(want, func(s string) bool { return !slices.Contains(got, s) })
	}
	if !ok {
		t.Fatalf("%s the items are %v, want %v", when, got, want)
	}
}

func exec(t *testing.T, w *subtitlestest.World, q string, args ...any) {
	t.Helper()
	err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, q, args...)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEveryHealthItemIsRaisedAndClearedFromItsRows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, nil)
		raises(t, w, "with no Wanted Language", false)
		w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		uncovered := "uncovered.fixture warn: No Subtitle Provider that is on has subtitles for shelves. Add a SubDL key."
		raises(t, w, "with Shelves wanting English and no SubDL key", false, uncovered)
		w.Holders(false)
		noHolder := "no-holder fail: No Node has the Subtitles Role, so SlipStream can't download or time subtitles. Assign it on the Nodes page."
		raises(t, w, "with no holder", true, noHolder)
		w.Holders(true)
		w.Holder = subtitlestest.KeyHolder{Account: "jsmith", Plan: "free"}
		if _, err := w.Subs.SetKey(t.Context(), identity.Admin{}, "subdl", "k"); err != nil {
			t.Fatal(err)
		}
		raises(t, w, "with SubDL keyed", false)
		if _, err := w.Subs.SetProviderOn(t.Context(), identity.Admin{}, "subdl", false); err != nil {
			t.Fatal(err)
		}
		raises(t, w, "with SubDL keyed and off", false, "uncovered.fixture warn: No Subtitle Provider that is on has subtitles for shelves. Turn on SubDL in Settings > Subtitles.")
		if _, err := w.Subs.SetProviderOn(t.Context(), identity.Admin{}, "subdl", true); err != nil {
			t.Fatal(err)
		}
		now := storage.FromTime(time.Now())
		exec(t, w, `UPDATE subtitles_provider SET paused_until = ?, pause_reason = 'key', pause_words = 'refused the key: Not Authorized' WHERE id = 'subdl'`, now)
		rejected := `key-rejected.subdl fail: SubDL rejected your key and responded "Not Authorized". Add a new key in Settings > Subtitles.`
		raises(t, w, "with SubDL's key rejected", true, rejected)
		if _, err := w.Subs.SetKey(t.Context(), identity.Admin{}, "subdl", "k2"); err != nil {
			t.Fatal(err)
		}
		exec(t, w, `UPDATE subtitles_provider SET failing_since = ?, last_error = 'Gestdown timed out' WHERE id = 'gestdown'`, storage.FromTime(time.Now().Add(-30*time.Minute)))
		a, err := w.Subs.HealthSource().Answer(t.Context())
		if err != nil || len(a.Items) != 0 || !a.Next.Equal(time.Now().Add(30*time.Minute)) {
			t.Fatalf("a provider failing 30 minutes reads %+v, %v, want no item and the edge in 30 minutes", a, err)
		}
		time.Sleep(31 * time.Minute)
		failing := "failing.gestdown warn: Gestdown isn't responding. SlipStream retries automatically."
		raises(t, w, "past the warning", false, failing)
		exec(t, w, `UPDATE subtitles_provider SET failing_since = NULL WHERE id = 'gestdown'`)
		exec(t, w, `INSERT INTO subtitles_node (node_id, fault_at, fault_words) VALUES (?, ?, 'ffmpeg: error while loading shared libraries')`, w.Node, now)
		fault := "map-fault." + w.Node.String() + " fail: Node A can't scan audio because its bundled ffmpeg doesn't start. Restart Node A, and reinstall SlipStream on it if that doesn't fix it."
		raises(t, w, "with a Node fault", false, fault)
		exec(t, w, `UPDATE subtitles_node SET fault_at = NULL, fault_words = NULL`)
		raises(t, w, "after the Node's fault clears", false)
		if health.SubtitlesID != "subtitles" || !slices.Contains(health.IDs(), health.SubtitlesID) {
			t.Fatal("Subtitles is not a Health category")
		}
	})
}
