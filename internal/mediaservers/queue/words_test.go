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
	missed := tb.change(link.ChangeMoved, "A>B", "f")
	missed.State, missed.Lost = queue.Prepared, []link.Loss{link.LostWatchState, link.LostCollections}
	moving := tb.change(link.ChangeMoved, "A>d:A", "f")
	moving.State, moving.Snapshot = queue.Prepared, &link.Snapshot{}
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
		{sent(tb.change(link.ChangeSidecarRemoved, "A.en.srt", "s"), link.ResultAccepted), "Sent the subtitle file's removal."},
		{refused, "The Media Server refused SlipStream's credential (status 401). SlipStream retries automatically."},
		{tb.doomed(tb.change(link.ChangeRemoved, "A", "old")), "This change is part of an upgrade. SlipStream sends it together with the upgrade's other file change."},
		{prepared, "Saved every user's watch data before the rename. SlipStream sends this change once the rename finishes."},
		{missed, "SlipStream couldn't save the watch data before the rename. It tries again when it sends this change."},
		{sent(tb.change(link.ChangeMoved, "A>d:A", "f"), link.ResultAccepted, link.LostCollections), "Sent the Move to Vault (E:). Lost: collections."},
		{moving, "Saved every user's watch data before the Move. SlipStream sends this change once the Move finishes."},
	} {
		if got := c.change.Words("Vault (E:)"); got != c.want {
			t.Errorf("%s reads %q, want %q", show(tb, c.change), got, c.want)
		}
	}
}
