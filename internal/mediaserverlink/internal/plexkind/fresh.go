package plexkind

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// fresh is the fresh folder rule on Plex, which ignores a partial refresh
// into a section never scanned with content (#101). Its step comes from
// the mapping's columns, and a section already refreshing is waited on.
// Undecided, an item under the folder ends it. Else the Job comes back
// with Requested and Started, the section's scannedAt, and nothing sent,
// so the Controller writes them before the refresh goes. Decided, a later
// scannedAt is the scan's end (graft 3), and none yet refreshes the
// section. scannedAt is whole seconds on Plex's own clock: a refresh that
// ends in Started's second reads as not ended and is sent again, which
// ends later.
func (c *conn) fresh(ctx context.Context, j link.Job, w link.Fresh) (link.Advanced, error) {
	s, err := c.section(ctx, w.Library)
	if err != nil {
		return link.Advanced{}, err
	}
	now := time.Now()
	wait := later(c.t.ScanWait(w.Requested, now))
	switch {
	case s.Refreshing:
		return link.Advanced{Job: j, Wait: wait}, nil
	case w.Requested.IsZero():
		held, err := c.holds(ctx, w.Library, w.Folder)
		if err != nil || held {
			return link.Advanced{Job: j, Done: true}, err
		}
		w.Requested, w.Started = now, scannedAt(s)
		j.Work = w
		return link.Advanced{Job: j}, nil
	case scannedAt(s).After(w.Started) || now.After(w.Requested.Add(c.t.ScanEndBound)):
		return link.Advanced{Job: j, Done: true}, nil
	}
	if err := c.c.RefreshSection(ctx, s.Key); err != nil {
		return link.Advanced{}, err
	}
	return link.Advanced{Job: j, Wait: wait}, nil
}

// scannedAt is when the section's last scan ended, zero for never.
func scannedAt(s plex.Section) time.Time {
	if s.ScannedAt == 0 {
		return time.Time{}
	}
	return time.Unix(s.ScannedAt, 0)
}

func (c *conn) section(ctx context.Context, key string) (plex.Section, error) {
	s, ok, err := c.c.Section(ctx, plex.ID(key))
	if err != nil {
		return plex.Section{}, err
	}
	if !ok {
		return plex.Section{}, &link.ServerError{Problem: link.Problem{Reason: link.ReasonFailed, Detail: "the server has no section " + key}}
	}
	return s, nil
}
