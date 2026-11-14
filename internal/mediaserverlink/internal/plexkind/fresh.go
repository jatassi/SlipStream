package plexkind

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// fresh is the fresh folder rule on Plex, which ignores a partial refresh
// into a section never scanned with content (#101). Its step comes from
// the mapping's columns: with no scan started it looks for an item under
// the folder and, finding none, refreshes the section unless one already
// runs. Once a scan started it waits for the scan's end, then looks again.
// The Controller writes Requested before it sends the Job, so a retry
// compares the end with the first request.
func (c *conn) fresh(ctx context.Context, j link.Job, w link.Fresh) (link.Advanced, error) {
	if w.Started.IsZero() {
		held, err := c.holds(ctx, w.Library, w.Folder)
		if err != nil {
			return link.Advanced{}, err
		}
		if held {
			return link.Advanced{Job: j, Done: true, Mapping: &link.MappingFact{Scanned: true}}, nil
		}
		s, err := c.section(ctx, w.Library)
		if err != nil {
			return link.Advanced{}, err
		}
		if !s.Refreshing {
			if err := c.c.RefreshSection(ctx, s.Key); err != nil {
				return link.Advanced{}, err
			}
		}
		return link.Advanced{Job: j, Mapping: &link.MappingFact{Started: time.Now()}, Wait: later(c.t.ScanEndPoll)}, nil
	}
	s, err := c.section(ctx, w.Library)
	if err != nil {
		return link.Advanced{}, err
	}
	since := time.Since(w.Started)
	if !ended(s, w.Requested) && since < c.t.ScanEndBound {
		return link.Advanced{Job: j, Wait: later(min(max(since, c.t.ScanEndPoll), c.t.ScanEndPollMax))}, nil
	}
	held, err := c.holds(ctx, w.Library, w.Folder)
	if err != nil {
		return link.Advanced{}, err
	}
	return link.Advanced{Job: j, Done: true, Mapping: &link.MappingFact{Started: w.Started, Scanned: held}}, nil
}

// ended is graft 3's scan end on Plex: the section isn't refreshing and
// its last scan ended no earlier than the second of the request. Plex's
// scannedAt is whole seconds.
func ended(s plex.Section, requested time.Time) bool {
	return !s.Refreshing && s.ScannedAt >= requested.Unix()
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
