package mstest

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Apply applies one of node's reports, as the Role's receive Write does,
// and follows its commit as the Role does.
func (w *World) Apply(ctx context.Context, node role.NodeID, r *msv1.Report) error {
	err := w.Lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.ms.Load().Apply(ctx, tx, node, r)
	})
	if err != nil {
		return err
	}
	w.ms.Load().Applied(node, string(role.MediaServerLink))
	return nil
}

// Plays is every play of server, by start, each as "<path> <decision>
// (<video> <audio> <subtitles>) <reasons> by <user> on <player>, open" or
// "..., ended <n>s in" for one that ended that long after it started. A
// play that names its player key, platform, item, received facts or
// tracks shows each after the player.
func (w *World) Plays(t *testing.T, server link.ServerID) []string {
	t.Helper()
	var out []string
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		rows, err := db.New(r).ListPlays(ctx, server)
		for _, p := range rows {
			var b strings.Builder
			fmt.Fprintf(&b, "%s %s (%s %s %s) %s by %s on %s", p.ServerPath, p.Decision, p.Video, p.Audio, p.Subtitles, p.Reasons, p.UserName, p.Player)
			for _, f := range []struct{ name, value string }{
				{"key", p.PlayerKey}, {"platform", p.Platform}, {"item", p.ItemKey},
				{"audio", trackWords(p.AudioTrack)}, {"subtitle", trackWords(p.SubtitleTrack)},
			} {
				if f.value != "" {
					fmt.Fprintf(&b, " %s %s", f.name, f.value)
				}
			}
			if p.Received != (link.Received{}) {
				fmt.Fprintf(&b, " received %+v", p.Received)
			}
			if p.EndedAt.IsZero() {
				b.WriteString(", open")
			} else {
				fmt.Fprintf(&b, ", ended %ds in", int(p.EndedAt.Time().Sub(p.StartedAt.Time()).Seconds()))
			}
			out = append(out, b.String())
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// PlaysSince is when the Controller started recording server's plays.
func (w *World) PlaysSince(t *testing.T, server link.ServerID) time.Time {
	t.Helper()
	var since storage.Time
	err := w.Lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		c, err := db.New(r).GetContact(ctx, server)
		since = c.PlaysSince
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return since.Time()
}

func trackWords(t *link.Track) string {
	switch {
	case t == nil:
		return ""
	case t.External != nil:
		return "external " + t.External.Codec + " " + t.External.Language
	}
	return strconv.Itoa(t.Position)
}

// Play is one play as its server's reaching Node reports it.
type Play struct {
	Server                                                    link.ServerID
	Session, Path, Item, PlayerKey, Player, Platform, Product string
	User                                                      string
	Streams                                                   link.Streams
	Hardware                                                  bool
	Reasons                                                   []string
	Received                                                  link.Received
	Audio, Subtitle                                           *link.Track
	Started, Seen                                             time.Time
}

// Play reports p started from node, and answers its id.
func (w *World) Play(t *testing.T, node role.NodeID, p Play) link.PlayID {
	t.Helper()
	id := link.PlayID{UUID: storage.NewUUID()}
	w.report(t, node, &msv1.Report{ServerId: p.Server.Bytes(), Report: &msv1.Report_PlayStarted{PlayStarted: &msv1.PlayStarted{Session: &msv1.Session{
		PlayId: id.Bytes(), SessionKey: p.Session, ServerPath: p.Path, User: p.User, Player: p.Player, Product: p.Product,
		Hardware: p.Hardware, Reasons: p.Reasons, StartedUnixMs: p.Started.UnixMilli(), SeenUnixMs: p.Seen.UnixMilli(),
		PlayerKey: p.PlayerKey, Platform: p.Platform, ItemKey: p.Item, Streams: streamsOf(p.Streams),
		Received: receivedOf(p.Received), AudioTrack: trackOf(p.Audio), SubtitleTrack: trackOf(p.Subtitle),
	}}}})
	return id
}

// ChangePlay reports that play id now plays as p says, seen at p.Seen.
func (w *World) ChangePlay(t *testing.T, node role.NodeID, id link.PlayID, p Play) {
	t.Helper()
	w.report(t, node, &msv1.Report{ServerId: p.Server.Bytes(), Report: &msv1.Report_PlayChanged{PlayChanged: &msv1.PlayChanged{
		PlayId: id.Bytes(), Hardware: p.Hardware, Reasons: p.Reasons, SeenUnixMs: p.Seen.UnixMilli(),
		Streams: streamsOf(p.Streams), Received: receivedOf(p.Received), AudioTrack: trackOf(p.Audio), SubtitleTrack: trackOf(p.Subtitle),
	}}})
}

// EndPlay reports that play id of server ended, last seen at seen.
func (w *World) EndPlay(t *testing.T, node role.NodeID, server link.ServerID, id link.PlayID, seen time.Time) {
	t.Helper()
	w.report(t, node, &msv1.Report{ServerId: server.Bytes(), Report: &msv1.Report_PlayEnded{PlayEnded: &msv1.PlayEnded{
		PlayId: id.Bytes(), LastSeenUnixMs: seen.UnixMilli(),
	}}})
}

func (w *World) report(t *testing.T, node role.NodeID, r *msv1.Report) {
	t.Helper()
	if err := w.Apply(t.Context(), node, r); err != nil {
		t.Fatal(err)
	}
}

func streamsOf(s link.Streams) *msv1.Streams {
	return &msv1.Streams{Video: string(s.Video), Audio: string(s.Audio), Subtitles: string(s.Subtitles), ContainerChanged: s.ContainerChanged}
}

func receivedOf(r link.Received) *msv1.Received {
	return &msv1.Received{
		Container: r.Container, VideoCodec: r.VideoCodec, Height: int32(r.Height), Bitrate: r.Bitrate, //nolint:gosec // a test's height
		AudioCodec: r.AudioCodec, Channels: int32(r.Channels), //nolint:gosec // a test's channels
	}
}

func trackOf(t *link.Track) *msv1.Track {
	switch {
	case t == nil:
		return nil
	case t.External != nil:
		return &msv1.Track{Track: &msv1.Track_External{External: &msv1.ExternalTrack{Codec: t.External.Codec, Language: t.External.Language}}}
	}
	return &msv1.Track{Track: &msv1.Track_Position{Position: int32(t.Position)}} //nolint:gosec // a test's track position
}

// SessionsChanged is every media-servers.sessions-changed published so
// far.
func (w *World) SessionsChanged() []mediaservers.SessionsChanged {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]mediaservers.SessionsChanged(nil), w.sessions...)
}

type sessionsEvents struct{ w *World }

func (e sessionsEvents) Publish(_ context.Context, v mediaservers.SessionsChanged) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	e.w.sessions = append(e.w.sessions, v)
}

// PlaysChanged is every media-servers.plays-changed published so far.
func (w *World) PlaysChanged() []mediaservers.PlaysChanged {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]mediaservers.PlaysChanged(nil), w.plays...)
}

type playsEvents struct{ w *World }

func (e playsEvents) Publish(_ context.Context, v mediaservers.PlaysChanged) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	e.w.plays = append(e.w.plays, v)
}
