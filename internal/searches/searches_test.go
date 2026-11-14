package searches_test

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

type recorded struct {
	triggers []searches.Trigger
	asks     [][]searches.Ask
}

func (r *recorded) Queue(_ context.Context, _ *storage.Tx[storage.Library], t searches.Trigger, asks ...searches.Ask) error {
	r.triggers = append(r.triggers, t)
	r.asks = append(r.asks, asks)
	return nil
}

func TestALateQueuePanicsBeforeBindAndPassesAsksOnAfter(t *testing.T) {
	var late searches.Late
	func() {
		defer func() {
			if recover() == nil {
				t.Error("a Late Queue took an ask before Bind, want a panic")
			}
		}()
		_ = late.Queue(t.Context(), nil, searches.Add)
	}()

	var r recorded
	late.Bind(&r)
	ask := searches.Ask{Work: works.WorkID{UUID: storage.NewUUID()}}
	if err := late.Queue(t.Context(), nil, searches.Monitor, ask); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(r.triggers, []searches.Trigger{searches.Monitor}) || !reflect.DeepEqual(r.asks, [][]searches.Ask{{ask}}) {
		t.Errorf("the bound Queue was asked %v %v, want one monitor ask for %v", r.triggers, r.asks, ask.Work)
	}
}
