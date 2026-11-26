package mediaservers

import (
	"errors"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// The tallest picture and the most audio channels a received fact may
// name. A larger one is stored as unreported.
const (
	maxHeight   = 8640
	maxChannels = 32
)

// caps is what the Controller keeps of a reaching Node's values.
type caps struct {
	str, path, reasons, sessions int
	skew                         time.Duration
}

func (m *MediaServers) caps() caps {
	t := m.cfg.Timings
	return caps{str: t.StringCap, path: t.PathCap, reasons: t.ReasonCap, sessions: t.SessionCap, skew: t.FutureSkew}
}

// reported is a play report's values, parsed, cut and checked.
type reported struct {
	id                                         link.PlayID
	sessionKey                                 string
	path                                       link.ServerPath
	playerKey, player, platform, product, user string
	streams                                    link.Streams
	hardware                                   bool
	reasons, itemKeys                          []string
	received                                   link.Received
	audio, subtitle                            *link.Track
	started, seen                              time.Time
}

// started is a started play's session, its start no earlier than since.
func (c caps) started(s *msv1.Session, now, since time.Time) (reported, error) {
	p, err := c.session(s, now)
	if err != nil {
		return reported{}, err
	}
	p.started = later(p.started, since)
	p.seen = later(p.seen, p.started)
	return p, nil
}

func (c caps) changed(m *msv1.PlayChanged, now time.Time) (reported, error) {
	id, err := playIDOf(m.GetPlayId())
	if err != nil {
		return reported{}, err
	}
	p := reported{id: id, hardware: m.GetHardware(), reasons: c.listOf(m.GetReasons()), seen: c.at(timeOf(m.GetSeenUnixMs()), now)}
	if p.streams, err = streamsOf(m.GetStreams()); err != nil {
		return reported{}, err
	}
	if p.received, err = c.receivedOf(m.GetReceived()); err != nil {
		return reported{}, err
	}
	if p.audio, err = c.trackOf(m.GetAudioTrack()); err != nil {
		return reported{}, err
	}
	if p.subtitle, err = c.trackOf(m.GetSubtitleTrack()); err != nil {
		return reported{}, err
	}
	return p, nil
}

// facts is a server's Facts sessions as Now playing and see may use them:
// at most c.sessions, strings cut, seen clamped, a session that won't parse
// dropped. ServerFacts.transcoding stays as the Node sent it.
func (c caps) facts(sf *msv1.ServerFacts, now time.Time) []reported {
	out := []reported{}
	for _, s := range sf.GetSessions() {
		if len(out) == c.sessions {
			break
		}
		if p, err := c.session(s, now); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func (c caps) session(s *msv1.Session, now time.Time) (reported, error) {
	id, err := playIDOf(s.GetPlayId())
	if err != nil {
		return reported{}, err
	}
	if len(s.GetServerPath()) > c.path {
		return reported{}, fmt.Errorf("the play's server path is longer than %d bytes", c.path)
	}
	started, seen := timeOf(s.GetStartedUnixMs()), timeOf(s.GetSeenUnixMs())
	if started.IsZero() || seen.Before(started) {
		return reported{}, errors.New("a play's times are out of order")
	}
	p := reported{
		id: id, sessionKey: c.text(s.GetSessionKey()), itemKeys: c.listOf(s.GetItemKeys()),
		playerKey: c.text(s.GetPlayerKey()), player: c.text(s.GetPlayer()), platform: c.text(s.GetPlatform()),
		product: c.text(s.GetProduct()), user: c.text(s.GetUser()), hardware: s.GetHardware(),
		reasons: c.listOf(s.GetReasons()), started: c.at(started, now),
	}
	p.seen = later(c.at(seen, now), p.started)
	if p.path, err = link.ParsePath(s.GetServerPath()); err != nil {
		return reported{}, fmt.Errorf("a play's path: %w", err)
	}
	if p.streams, err = streamsOf(s.GetStreams()); err != nil {
		return reported{}, err
	}
	if p.received, err = c.receivedOf(s.GetReceived()); err != nil {
		return reported{}, err
	}
	if p.audio, err = c.trackOf(s.GetAudioTrack()); err != nil {
		return reported{}, err
	}
	if p.subtitle, err = c.trackOf(s.GetSubtitleTrack()); err != nil {
		return reported{}, err
	}
	return p, nil
}

// text is s cut to at most c.str characters, at a rune boundary.
func (c caps) text(s string) string {
	n := 0
	for i := range s {
		if n == c.str {
			return s[:i]
		}
		n++
	}
	return s
}

// at is t, or now when t is later than now plus the skew.
func (c caps) at(t, now time.Time) time.Time {
	if t.After(now.Add(c.skew)) {
		return now
	}
	return t
}

// listOf is the first ReasonCap of rs, each cut to StringCap.
func (c caps) listOf(rs []string) []string {
	out := make([]string, 0, min(len(rs), c.reasons))
	for _, r := range rs[:min(len(rs), c.reasons)] {
		out = append(out, c.text(r))
	}
	return out
}

func (c caps) receivedOf(r *msv1.Received) (link.Received, error) {
	if r.GetHeight() < 0 || r.GetChannels() < 0 || r.GetBitrate() < 0 {
		return link.Received{}, errors.New("a play's received height, channels or bitrate is negative")
	}
	out := link.Received{
		Container: c.text(r.GetContainer()), VideoCodec: c.text(r.GetVideoCodec()), Height: int(r.GetHeight()),
		Bitrate: r.GetBitrate(), AudioCodec: c.text(r.GetAudioCodec()), Channels: int(r.GetChannels()),
	}
	if out.Height > maxHeight {
		out.Height = 0
	}
	if out.Channels > maxChannels {
		out.Channels = 0
	}
	return out, nil
}

func (c caps) trackOf(t *msv1.Track) (*link.Track, error) {
	switch x := t.GetTrack().(type) {
	case *msv1.Track_Position:
		if x.Position < 0 {
			return nil, errors.New("a play's track position is negative")
		}
		return &link.Track{Position: int(x.Position)}, nil
	case *msv1.Track_External:
		return &link.Track{External: &link.ExternalTrack{Codec: c.text(x.External.GetCodec()), Language: c.text(x.External.GetLanguage())}}, nil
	}
	return nil, nil
}

// startParams is the row p starts on server.
func (p reported) startParams(server link.ServerID) db.StartPlayParams {
	return db.StartPlayParams{
		ServerID: server, ID: p.id, SessionKey: p.sessionKey, ServerPath: p.path.String(),
		PathKey: p.path.Key(), ItemKeys: listJSON(p.itemKeys), PlayerKey: p.playerKey, Player: p.player,
		Platform: p.platform, Product: p.product, UserName: p.user,
		Decision: p.streams.Decision(), Video: p.streams.Video, Audio: p.streams.Audio, Subtitles: p.streams.Subtitles,
		Hardware: p.hardware, Reasons: listJSON(p.reasons), Received: p.received,
		AudioTrack: p.audio, SubtitleTrack: p.subtitle,
		StartedAt: storage.FromTime(p.started), LastSeenAt: storage.FromTime(p.seen),
	}
}
