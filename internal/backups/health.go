package backups

import (
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
)

// DestinationKey is destination d's Health item.
func DestinationKey(d DestinationID) health.Key { return health.KeyOf("destination", d.String()) }

// DestinationHealth is one Warning per destination whose newest copy was
// copied longer than stale ago, or whose newest Backup's copy or last
// check failed, and Unsure for each whose Node is offline. names names
// each Node and online is whether a Node is connected now.
func (b *Backups) DestinationHealth(now time.Time, stale time.Duration, names map[role.NodeID]string, online func(role.NodeID) bool) health.Answer {
	l := b.cfg.Engine.Index().Read()
	var a health.Answer
	for _, d := range l.Active() {
		subject := fmt.Sprintf("Backup copies in %s on %s", d.Folder, names[d.Node])
		key := DestinationKey(d.ID)
		if !online(d.Node) {
			a.Unsure = append(a.Unsure, health.Unsure{Key: key, Says: health.Says{Subject: subject, Text: names[d.Node] + " is offline."}})
			continue
		}
		text, next := destinationWords(l, d, now, stale)
		if !next.IsZero() && (a.Next.IsZero() || next.Before(a.Next)) {
			a.Next = next
		}
		if text == "" {
			continue
		}
		a.Items = append(a.Items, health.Item{Key: key, Level: health.Warn, Says: health.Says{
			Subject: subject,
			Text:    text,
			Action:  health.LinkAction("/settings/backups", "Open Backups settings"),
		}})
	}
	return a
}

// destinationWords is what is wrong with d's copies, "" for nothing, and
// when its newest copy goes stale.
func destinationWords(l Ledger, d Destination, now time.Time, stale time.Duration) (string, time.Time) {
	var newest time.Time
	var failed string
	first := true
	for _, e := range l.Backups {
		c, ok := e.Copy(d.ID)
		if e.State != Verified || !ok {
			continue
		}
		if first {
			first = false
			switch {
			case c.State == CopyFailed:
				failed = "The last copy failed. " + c.Cause
			case c.Check.Result == CheckUnreadable:
				failed = "The last check failed. " + c.Check.Cause
			case c.Check.Result == CheckMismatch:
				failed = "The last check found that the copy doesn't match the Backup. SlipStream is copying it again."
			}
		}
		if c.State == Copied && c.CopiedAt.After(newest) {
			newest = c.CopiedAt
		}
	}
	since := newest
	if since.IsZero() {
		since = d.AddedAt
	}
	due := since.Add(stale)
	switch {
	case failed != "":
		return failed, due
	case now.Before(due):
		return "", due
	case newest.IsZero():
		return "No Backup has been copied here yet.", time.Time{}
	}
	return fmt.Sprintf("The newest copy is %s old.", age(now.Sub(newest))), time.Time{}
}

// age is d in whole days, or hours under two days.
func age(d time.Duration) string {
	if hours := int64(d / time.Hour); hours < 48 {
		return fmt.Sprintf("%d %s", hours, hourNoun.Of(hours))
	}
	return fmt.Sprintf("%d days", int64(d/(24*time.Hour)))
}

var hourNoun = setting.Noun{One: "hour", Many: "hours"}
