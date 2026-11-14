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
	for _, x := range sessions[:min(len(sessions), c.t.SessionCap)] {
		s.Sessions = append(s.Sessions, sessionOf(x))
	}
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

// sessionOf is one session. videoDecision is transcode for a re-encode
// and copy for a remux, and direct play has no transcode session. Plex
// gives no transcode reasons.
func sessionOf(x plex.Session) link.Session {
	s := link.Session{Key: string(x.SessionKey), Decision: link.DecisionDirectPlay}
	if x.User != nil {
		s.User = x.User.Title
	}
	if x.Player != nil {
		s.Player, s.Product = x.Player.Title, x.Player.Product
	}
	if fs := files(x.Metadata); len(fs) > 0 {
		s.Path = fs[0]
	}
	if t := x.Transcode; t != nil {
		s.Decision, s.Hardware = link.DecisionCopy, t.TranscodeHwFullPipeline || t.TranscodeHwEncoding != ""
		if t.VideoDecision == "transcode" {
			s.Decision = link.DecisionTranscode
		}
	}
	return s
}
