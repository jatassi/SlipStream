package backups

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/firstrun"
	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/tasks"
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

// HealthAbout declares the Backups category, between Scheduled tasks and
// Notifications. It owns the Backup task's item.
var HealthAbout = health.About{
	ID:     health.BackupsID,
	Label:  "Backups",
	Phrase: "Backups",
	Events: []live.Event{ChangedEvent, nodes.ChangedEvent, tasks.ChangedEvent},
}

var (
	controllerKey = health.KeyOf("controller")
	passphraseKey = health.KeyOf("passphrase")
)

// HealthSource answers the Controller's Backups, the passphrase not saved
// and each destination from the index and the stored settings.
func (b *Backups) HealthSource() health.Source {
	return health.Source{Answer: func(ctx context.Context) (health.Answer, error) {
		within, err := b.warnings(ctx)
		if err != nil {
			return health.Answer{}, err
		}
		l := b.Ledger()
		now := b.cfg.Engine.cfg.Clock()
		names := map[role.NodeID]string{}
		for _, d := range l.Active() {
			names[d.Node], _ = b.cfg.Roster(d.Node)
		}
		online := func(id role.NodeID) bool { _, ok := b.cfg.Roster(id); return ok }
		a := b.DestinationHealth(now, within, names, online)
		item, next := controllerHealth(l, now, within, b.cfg.Zone)
		if item != nil {
			a.Items = append([]health.Item{*item}, a.Items...)
		}
		if !next.IsZero() && (a.Next.IsZero() || next.Before(a.Next)) {
			a.Next = next
		}
		if l.PassphraseConfirmedAt.IsZero() {
			a.Items = append(a.Items, health.Item{Key: passphraseKey, Level: health.Warn, Says: health.Says{
				Subject: "Backup passphrase not saved",
				Text:    "You need the passphrase to restore a Backup on another machine. Save it outside SlipStream, then confirm in Settings > Backups.",
				Action:  health.LinkAction("/settings/backups", "Download recovery sheet"),
			}})
		}
		return a, nil
	}}
}

// countable is whether e counts toward Health: it passed its check, and
// its file is on this machine or a destination has a copy.
func countable(e Entry) bool {
	if e.State != Verified {
		return false
	}
	return e.Local || slices.ContainsFunc(e.Copies, func(c Copy) bool { return c.State == Copied })
}

// controllerHealth is the Controller's one item: a Warning while the last
// Backup failed and a good one is younger than within, then an Error. A
// Controller with no good Backup counts from its first failure. It
// answers when the Warning turns into the Error.
func controllerHealth(l Ledger, now time.Time, within time.Duration, zone *time.Location) (*health.Item, time.Time) {
	var good time.Time
	for _, e := range l.Backups {
		if countable(e) && e.TakenAt.After(good) {
			good = e.TakenAt
		}
	}
	failed := l.Failure != nil && l.Failure.At.After(good)
	base := good
	if base.IsZero() {
		if !failed {
			return nil, time.Time{}
		}
		base = l.Failure.Since
	}
	due := base.Add(within)
	action := health.LinkAction("/settings/backups", "Open Backups settings")
	text := "No Backup taken since then passed its check and is kept on this machine or a destination. Open the Backup task to run one."
	if failed {
		text = failureWords(*l.Failure, zone)
	}
	if !now.Before(due) {
		subject := "No good Backup yet"
		if !good.IsZero() {
			subject = "No good Backup since " + good.In(zone).Format("2 Jan")
		}
		return &health.Item{Key: controllerKey, Level: health.Fail, Says: health.Says{Subject: subject, Text: text, Action: action}}, time.Time{}
	}
	if !failed {
		return nil, due
	}
	return &health.Item{Key: controllerKey, Level: health.Warn, Says: health.Says{Subject: "Last Backup failed", Text: text, Action: action}}, due
}

func failureWords(f Failure, zone *time.Location) string {
	at := f.At.In(zone).Format("2 Jan, 15:04")
	if f.Space == nil {
		return "The last Backup failed at " + at + ". Open the Backup task for the cause."
	}
	return fmt.Sprintf("The last Backup failed at %s because %s has %s free and the Backup needs about %s. Free some space there, or pick another Backups folder on the Controller's Node page.",
		at, f.Space.Folder, Size(f.Space.Free), Size(f.Space.Need))
}

// Step is the first setup step backup: done once the Admin saved the
// passphrase.
func (b *Backups) Step() firstrun.Step {
	return firstrun.Step{
		ID: firstrun.BackupID,
		Done: func(context.Context, storage.Reader[storage.Library]) (bool, error) {
			return !b.Ledger().PassphraseConfirmedAt.IsZero(), nil
		},
		Events: []live.Event{ChangedEvent},
	}
}
