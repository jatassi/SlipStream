package grabstest_test

import (
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
		w.Run()
		group := indexers.LevelGroup(w.Level)
		want := grabstest.Queue{
			Due: []grabstest.Due{{Work: w.Work, Group: group, Searching: true}},
			Runs: []grabstest.Run{{
				Work: w.Work, Group: group, Lane: pace.Now, Trigger: searches.Search, Gone: true, Phase: 1,
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
		time.Sleep(5 * time.Second)
		if got := w.Queue(t); len(got.Runs) != 0 || len(node.Jobs()) != 1 {
			t.Fatalf("the restarted Grabs left runs %+v open after %d jobs, want the press finished by its one job", got.Runs, len(node.Jobs()))
		}
	})
}
