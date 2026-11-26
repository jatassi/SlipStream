package mediaservers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// startPlay records a play the reaching Node started. Plays upsert by the
// id that Node minted, so applying a report twice changes nothing, and a
// report for a play the Controller already ended, or a start for a play
// it already has, changes nothing either.
func startPlay(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.PlayStarted) error {
	s := m.GetSession()
	id, err := playIDOf(s.GetPlayId())
	if err != nil {
		return err
	}
	path, err := link.ParsePath(s.GetServerPath())
	if err != nil {
		return fmt.Errorf("a started play's path: %w", err)
	}
	streams, err := streamsOf(s.GetStreams())
	if err != nil {
		return err
	}
	started, seen := timeOf(s.GetStartedUnixMs()), timeOf(s.GetSeenUnixMs())
	if started.IsZero() || seen.Before(started) {
		return errors.New("a started play's times are out of order")
	}
	reasons, err := reasonsOf(s.GetReasons())
	if err != nil {
		return err
	}
	file, err := q.FileAtPath(ctx, db.FileAtPathParams{ServerID: server, PathKey: path.Key()})
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return err
	}
	return q.StartPlay(ctx, db.StartPlayParams{
		ServerID: server, ID: id, SessionKey: s.GetSessionKey(), FileID: file, ServerPath: path.String(),
		PathKey: path.Key(), ItemKey: s.GetItemKey(), PlayerKey: s.GetPlayerKey(), Player: s.GetPlayer(),
		Platform: s.GetPlatform(), Product: s.GetProduct(), UserName: s.GetUser(),
		Decision: streams.Decision(), Video: streams.Video, Audio: streams.Audio, Subtitles: streams.Subtitles,
		Hardware: s.GetHardware(), Reasons: reasons, Received: receivedOf(s.GetReceived()),
		AudioTrack: trackOf(s.GetAudioTrack()), SubtitleTrack: trackOf(s.GetSubtitleTrack()),
		StartedAt: storage.FromTime(started), LastSeenAt: storage.FromTime(seen),
	})
}

func changePlay(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.PlayChanged) error {
	id, err := playIDOf(m.GetPlayId())
	if err != nil {
		return err
	}
	streams, err := streamsOf(m.GetStreams())
	if err != nil {
		return err
	}
	reasons, err := reasonsOf(m.GetReasons())
	if err != nil {
		return err
	}
	seen, open, err := laterSeen(ctx, q, server, id, timeOf(m.GetSeenUnixMs()))
	if err != nil || !open {
		return err
	}
	return q.ChangePlay(ctx, db.ChangePlayParams{
		ServerID: server, ID: id, Decision: streams.Decision(), Video: streams.Video, Audio: streams.Audio,
		Subtitles: streams.Subtitles, Hardware: m.GetHardware(), Reasons: reasons, Received: receivedOf(m.GetReceived()),
		AudioTrack: trackOf(m.GetAudioTrack()), SubtitleTrack: trackOf(m.GetSubtitleTrack()), LastSeenAt: seen,
	})
}

func endPlay(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.PlayEnded) error {
	id, err := playIDOf(m.GetPlayId())
	if err != nil {
		return err
	}
	seen, open, err := laterSeen(ctx, q, server, id, timeOf(m.GetLastSeenUnixMs()))
	if err != nil || !open {
		return err
	}
	return q.EndPlay(ctx, db.EndPlayParams{ServerID: server, ID: id, EndedAt: seen})
}

// watching ends every open play of server that m doesn't list, at its
// last look: the plays a holder that stopped left open. The plays it
// lists were seen at m's time.
func watching(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.Watching) error {
	listed := map[link.PlayID]bool{}
	for _, b := range m.GetPlayIds() {
		id, err := playIDOf(b)
		if err != nil {
			return err
		}
		listed[id] = true
	}
	at := timeOf(m.GetAtUnixMs())
	if at.IsZero() {
		return errors.New("a Watching report has no time")
	}
	open, err := q.ListOpenPlays(ctx, server)
	if err != nil {
		return err
	}
	for _, id := range open {
		if !listed[id] {
			if err := q.EndPlayAsSeen(ctx, db.EndPlayAsSeenParams{ServerID: server, ID: id}); err != nil {
				return err
			}
			continue
		}
		seen, _, err := laterSeen(ctx, q, server, id, at)
		if err != nil {
			return err
		}
		if err := q.SeePlay(ctx, db.SeePlayParams{ServerID: server, ID: id, LastSeenAt: seen}); err != nil {
			return err
		}
	}
	return nil
}

// laterSeen is the later of the open play's last look and at, false when
// the play isn't open.
func laterSeen(ctx context.Context, q *db.Queries, server link.ServerID, id link.PlayID, at time.Time) (storage.Time, bool, error) {
	p, err := q.GetOpenPlay(ctx, db.GetOpenPlayParams{ServerID: server, ID: id})
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return storage.Time{}, false, nil
	case err != nil:
		return storage.Time{}, false, err
	}
	seen := p.LastSeenAt
	if t := storage.FromTime(at); t.Compare(seen) > 0 {
		seen = t
	}
	return seen, true, nil
}

// hint marks stale the path index entries of the items a push said
// changed.
func hint(ctx context.Context, q *db.Queries, server link.ServerID, m *msv1.IndexHint) error {
	if len(m.GetItemKeys()) == 0 {
		return nil
	}
	return q.MarkItemsStale(ctx, db.MarkItemsStaleParams{ServerID: server, ItemKeys: m.GetItemKeys()})
}

func playIDOf(b []byte) (link.PlayID, error) {
	u, err := storage.UUIDFromBytes(b)
	if err != nil {
		return link.PlayID{}, fmt.Errorf("read a play id: %w", err)
	}
	return link.PlayID{UUID: u}, nil
}

func timeOf(ms int64) time.Time {
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

func reasonsOf(rs []string) (string, error) {
	if rs == nil {
		rs = []string{}
	}
	b, err := json.Marshal(rs)
	return string(b), err
}

func streamsOf(s *msv1.Streams) (link.Streams, error) {
	video, err := link.ParseStreamDecision(s.GetVideo())
	if err != nil {
		return link.Streams{}, fmt.Errorf("a play's video: %w", err)
	}
	audio, err := link.ParseStreamDecision(s.GetAudio())
	if err != nil {
		return link.Streams{}, fmt.Errorf("a play's audio: %w", err)
	}
	subtitles, err := link.ParseSubtitleDecision(s.GetSubtitles())
	if err != nil {
		return link.Streams{}, fmt.Errorf("a play's subtitles: %w", err)
	}
	return link.Streams{Video: video, Audio: audio, Subtitles: subtitles, ContainerChanged: s.GetContainerChanged()}, nil
}

func receivedOf(r *msv1.Received) link.Received {
	return link.Received{
		Container: r.GetContainer(), VideoCodec: r.GetVideoCodec(), Height: int(r.GetHeight()), Bitrate: r.GetBitrate(),
		AudioCodec: r.GetAudioCodec(), Channels: int(r.GetChannels()),
	}
}

func trackOf(t *msv1.Track) *link.Track {
	switch x := t.GetTrack().(type) {
	case *msv1.Track_Position:
		return &link.Track{Position: int(x.Position)}
	case *msv1.Track_External:
		return &link.Track{External: &link.ExternalTrack{Codec: x.External.GetCodec(), Language: x.External.GetLanguage()}}
	}
	return nil
}
