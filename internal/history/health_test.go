package history_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/history/historytest"
)

func TestAHealthFaultAndItsClearAreEntriesAtTheirCaptureWithTheTransitionsTimes(t *testing.T) {
	run(t, func(t *testing.T, w *world) {
		since := time.Now().Add(-10 * time.Minute)
		raised := health.Transition{
			Category: "storage", Key: "d", Kind: health.Raised, To: health.Fail,
			Subject: "Disk d is almost full", Text: "It has 1 GB left.", At: since, Since: since,
		}
		w.Transitions(raised)
		w.Advance(time.Hour)
		clearedAt := time.Now().Add(-time.Minute)
		cleared := raised
		cleared.Kind, cleared.From, cleared.To, cleared.At = health.Cleared, health.Fail, health.None, clearedAt
		w.Transitions(cleared)

		got := historytest.Entries(t, w.Lib)
		if len(got) != 2 || got[0].Kind != "new_health_item" || got[1].Kind != "health_item_cleared" {
			t.Fatalf("History holds %+v, want New Health item then Health item cleared", got)
		}
		if !got[0].At.Equal(since.Add(10*time.Minute)) || !got[1].At.Equal(clearedAt.Add(time.Minute)) {
			t.Errorf("the entries are at %v and %v, want each capture's time", got[0].At, got[1].At)
		}
		raise, clear := got[0].Items[0], got[1].Items[0]
		if raise.Key != "storage.d" || raise.Level != "fail" || !raise.Since.Equal(since) || raise.Subject != "Disk d is almost full" {
			t.Errorf("the raise holds %+v, want storage.d at Error since the transition", raise)
		}
		if !clear.Since.Equal(since) || !clear.Cleared.Equal(clearedAt) || clear.Subject != "Disk d is almost full" {
			t.Errorf("the clear holds %+v, want the raise's words, its since and the clear's time", clear)
		}
	})
}
