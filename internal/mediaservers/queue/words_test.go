package queue_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/queue"
)

func TestWordsSayWhatAChangeDidAndWhatItLost(t *testing.T) {
	tb := newTable(t)
	sent := func(c queue.Change, r link.Result, lost ...link.Loss) queue.Change {
		c.State, c.Sent, c.Result, c.Lost = queue.Sent, at, r, lost
		return c
	}
	refused := tb.change(link.ChangeAdded, "A", "f")
	refused.Attempts, refused.Problem = 1, &link.Problem{Reason: link.ReasonRefused, Status: 401}
	prepared := tb.change(link.ChangeMoved, "A>B", "f")
	prepared.State, prepared.Snapshot = queue.Prepared, &link.Snapshot{}
	for _, c := range []struct {
		change queue.Change
		want   string
	}{
		{sent(tb.change(link.ChangeMoved, "A>B", "f"), link.ResultAccepted), "Sent the rename. Kept every user's watch data."},
		{sent(tb.change(link.ChangeReplaced, "A>B", "f"), link.ResultAccepted, link.LostPlayCount, link.LostLastPlayed),
			"Sent the replaced file. Lost: play count, last played."},
		{sent(tb.change(link.ChangeMoved, "A>B", "f"), link.ResultConverted),
			"The server had no item at the old path. SlipStream sent this change as a new file."},
		{sent(tb.change(link.ChangeAdded, "A", "f"), link.ResultCovered), "A refresh of the whole library covered this change."},
		{sent(tb.change(link.ChangeSidecarRemoved, "A.en.srt", "s"), link.ResultAccepted), "Sent the Sidecar's removal."},
		{refused, "The Media Server refused SlipStream's credential (status 401). SlipStream retries automatically."},
		{tb.doomed(tb.change(link.ChangeRemoved, "A", "old")), "This change is waiting for the other half of its upgrade."},
		{prepared, "Saved every user's watch data before the rename. SlipStream sends this change once the rename finishes."},
	} {
		if got := c.change.Words(); got != c.want {
			t.Errorf("%s reads %q, want %q", show(tb, c.change), got, c.want)
		}
	}
}
