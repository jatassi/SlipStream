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
		if got := items(t, w); len(got) != 0 {
			t.Fatalf("a library with no Wanted Language raises %v", got)
		}
		w.Profile("Films", subtitlestest.Wanted(t, "en", false, targets.Allow, targets.Always))
		uncovered := "uncovered.fixture warn: No Subtitle Provider that is on searches shelves. Add a SubDL key."
		if got := items(t, w); !slices.Equal(got, []string{uncovered}) {
			t.Fatalf("Shelves with no SubDL key raise %v", got)
		}
		w.Holders(false)
		noHolder := "no-holder fail: No Node has the Subtitles Role, so SlipStream can't download or time subtitles. Assign it on the Nodes page."
		if got := items(t, w); !slices.Contains(got, noHolder) {
			t.Fatalf("with no holder the items are %v", got)
		}
		w.Holders(true)
		w.Holder = subtitlestest.KeyHolder{Account: "jsmith", Plan: "free"}
		if _, err := w.Subs.SetKey(t.Context(), identity.Admin{}, "subdl", "k"); err != nil {
			t.Fatal(err)
		}
		if got := items(t, w); len(got) != 0 {
			t.Fatalf("with SubDL keyed the items are %v", got)
		}
		now := storage.FromTime(time.Now())
		exec(t, w, `UPDATE subtitles_provider SET paused_until = ?, pause_reason = 'key', pause_words = 'Not Authorized' WHERE id = 'subdl'`, now)
		rejected := `key-rejected.subdl fail: SubDL rejected your key and responded "Not Authorized". Add a new key in Settings > Subtitles.`
		if got := items(t, w); !slices.Contains(got, rejected) {
			t.Fatalf("a rejected key raises %v", got)
		}
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
		if got := items(t, w); !slices.Equal(got, []string{failing}) {
			t.Fatalf("past the warning the items are %v", got)
		}
		exec(t, w, `UPDATE subtitles_provider SET failing_since = NULL WHERE id = 'gestdown'`)
		exec(t, w, `INSERT INTO subtitles_node (node_id, fault_at, fault_words) VALUES (?, ?, 'ffmpeg: error while loading shared libraries')`, w.Node, now)
		fault := "map-fault." + w.Node.String() + " fail: Node A can't scan audio because its bundled ffmpeg doesn't start. Restart Node A, and reinstall SlipStream on it if that doesn't fix it."
		if got := items(t, w); !slices.Equal(got, []string{fault}) {
			t.Fatalf("a Node fault raises %v", got)
		}
		exec(t, w, `UPDATE subtitles_node SET fault_at = NULL, fault_words = NULL`)
		if got := items(t, w); len(got) != 0 {
			t.Fatalf("after the Node's fault clears the items are %v", got)
		}
		if health.SubtitlesID != "subtitles" || !slices.Contains(health.IDs(), health.SubtitlesID) {
			t.Fatal("Subtitles is not a Health category")
		}
	})
}
