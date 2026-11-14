package plexkind

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/providers/plex"
)

// Watch polls /status/sessions joined with /transcode/sessions every
// every, from the first look on, until ctx ends.
func (c *conn) Watch(ctx context.Context, every time.Duration, seen func(link.Seen)) error {
	if c.base != nil {
		return unreachable(c.base)
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		s := link.Seen{At: time.Now(), Polled: true}
		sessions, err := c.c.Sessions(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if p, ok := link.ServerProblem(fail(ctx, err)); ok {
			s.Problem = &p
		}
		for _, x := range sessions[:min(len(sessions), c.t.SessionCap)] {
			s.Sessions = append(s.Sessions, sessionOf(x))
		}
		seen(s)
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
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
