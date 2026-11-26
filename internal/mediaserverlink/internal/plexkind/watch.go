package plexkind

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// Watch polls /status/sessions joined with /transcode/sessions every
// every, and holds /:/websockets/notifications beside: a playing
// notification polls at once, and a timeline names the items it changed.
func (c *conn) Watch(ctx context.Context, every time.Duration, seen func(link.Seen)) error {
	if _, err := c.Reach(ctx); err != nil {
		return err
	}
	return link.Watcher{Poll: c.poll, Socket: c.socket}.Run(ctx, c.t, every, seen)
}

func (c *conn) poll(ctx context.Context) link.Seen {
	s := link.Seen{At: time.Now(), Polled: true}
	sessions, transcodes, err := c.c.Playing(ctx)
	if p, ok := link.ServerProblem(fail(ctx, err)); ok {
		s.Problem = &p
		return s
	}
	s.Sessions, s.Pending = c.sessions(ctx, sessions)
	for _, t := range transcodes {
		s.Transcoding = s.Transcoding || t.VideoDecision == "transcode"
	}
	return s
}

func (c *conn) socket(ctx context.Context, opened func(), push func(link.Seen), poll func()) error {
	return c.c.Notifications(ctx, opened, func(n plex.Notification) {
		if len(n.Playing) > 0 {
			poll()
		}
		var items []string
		for _, e := range n.Timeline {
			items = append(items, string(e.ItemID))
		}
		if len(items) > 0 {
			push(link.Seen{At: time.Now(), Items: items})
		}
	})
}
