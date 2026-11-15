package grabstest_test

import (
	"context"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/grabs/grabstest"
	"github.com/jatassi/slipstream/internal/grabs/internal/pace"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/indexers/indexerstest"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/searching"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestARestartKeepsWhatTheQueueHolds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		node := &indexerstest.Node{}
		w := grabstest.New(t, grabstest.Options{
			Sites: []string{"Quick Site"}, Remote: node,
			Grabs: grabs.DefaultTimings(), Indexers: indexers.DefaultTimings(), Formats: formats.DefaultTimings(),
		})
		node.Answer(w.Sites["Quick Site"], 5*time.Second, searching.Answer{})
		if _, err := w.Grabs.SearchWork(t.Context(), identity.Admin{}, w.Work, grabs.Scope{}); err != nil {
			t.Fatal(err)
		}
		queued := grabstest.Due{Work: w.Work, Group: indexers.ItemGroup(w.Item), Items: []works.ItemID{w.Item}}
		err := w.Lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			if _, err := tx.ExecContext(ctx, `INSERT INTO grabs_due (work_id, group_key, lane, queued_at, trigger, due_at, gap_seconds, changed_at)
				VALUES (?, ?, 40, 1, 'backlog', 0, 3600, 0)`, w.Work, queued.Group); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO grabs_due_item (work_id, group_key, item_id) VALUES (?, ?, ?)`, w.Work, queued.Group, w.Item)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		queued.Lane, queued.Trigger = pace.Backlog, searches.Backlog
		want := grabstest.Queue{
			Due: []grabstest.Due{queued},
			Runs: []grabstest.Run{{
				Work: w.Work, Group: indexers.LevelGroup(w.Level), Lane: pace.Now, Trigger: searches.Search, Gone: true, Phase: 1,
				Items: []works.ItemID{w.Item}, Lines: []grabstest.Line{{Indexer: "Quick Site", State: "sent"}},
			}},
		}
		if got := w.Queue(t); !reflect.DeepEqual(got, want) {
			t.Fatalf("before the restart the queue holds %+v, want %+v", got, want)
		}
		w.Restart()
		if got := w.Queue(t); !reflect.DeepEqual(got, want) {
			t.Fatalf("after the restart the queue holds %+v, want %+v", got, want)
		}
		w.Run()
		time.Sleep(5 * time.Second)
		if got := w.Queue(t); len(got.Runs) != 0 {
			t.Fatalf("the restarted Grabs left runs %+v open, want the press finished", got.Runs)
		}
	})
}
