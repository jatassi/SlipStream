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

// sessionOf is one session. A transcode session puts the streams in a new
// container, and its videoDecision is transcode for a re-encode. Direct
// play has no transcode session. Plex gives no transcode reasons.
func sessionOf(x plex.Session) link.Session {
	s := link.Session{Key: string(x.SessionKey), Streams: link.Streams{Video: link.StreamDirect, Audio: link.StreamDirect, Subtitles: link.SubtitlesNone}}
	if x.User != nil {
		s.User = x.User.Title
	}
	if x.Player != nil {
		s.PlayerKey, s.Player, s.Product = x.Player.MachineIdentifier, x.Player.Title, x.Player.Product
	}
	if len(x.Media) == 1 {
		s.VersionID = string(x.Media[0].ID)
	}
	if fs := files(x.Metadata); len(fs) > 0 {
		s.Version = link.Version{Path: fs[0], Items: []string{string(x.RatingKey)}}
	}
	if t := x.Transcode; t != nil {
		s.Streams.ContainerChanged, s.Hardware = true, t.TranscodeHwFullPipeline || t.TranscodeHwEncoding != ""
		if t.VideoDecision == "transcode" {
			s.Streams.Video = link.StreamTranscode
		}
	}
	return s
}
