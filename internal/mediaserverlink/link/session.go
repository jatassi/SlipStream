package link

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Session is one playback a server lists, as the kind read it.
type Session struct {
	// Key is the server's own session key.
	Key     string
	Path    ServerPath
	User    string
	Player  string
	Product string
	Streams Streams
	// Hardware is whether the server reports hardware transcoding.
	Hardware bool
	// Reasons are the transcode reasons as the server gave them. Plex gives
	// none.
	Reasons []string
}

// Streams is a play's decision per stream, and whether the server put the
// streams in a new container.
type Streams struct {
	Video, Audio     StreamDecision
	Subtitles        SubtitleDecision
	ContainerChanged bool
}

// Decision is transcode when the video or audio transcodes or subtitles
// burn in, else direct_stream when the container changed, else
// direct_play.
func (s Streams) Decision() Decision {
	switch {
	case s.Video == StreamTranscode || s.Audio == StreamTranscode || s.Subtitles == SubtitlesBurned:
		return DecisionTranscode
	case s.ContainerChanged:
		return DecisionDirectStream
	}
	return DecisionDirectPlay
}

// Received is what the player received while the server transcoded: codec
// and container names in the release vocabulary, or the server's own name
// when the vocabulary has none. Zero is unreported.
type Received struct {
	Container  string `json:"container,omitempty"`
	VideoCodec string `json:"videoCodec,omitempty"`
	Height     int    `json:"height,omitempty"`
	// Bitrate is in bits a second.
	Bitrate    int64  `json:"bitrate,omitempty"`
	AudioCodec string `json:"audioCodec,omitempty"`
	Channels   int    `json:"channels,omitempty"`
}

func (r *Received) Scan(src any) error { return scanJSON(src, r) }

func (r Received) Value() (driver.Value, error) { return valueJSON(r) }

// Track is a chosen stream: Position among the version's embedded streams
// of its type, or External for one in a separate file.
type Track struct {
	Position int
	External *ExternalTrack
}

type ExternalTrack struct {
	Codec    string `json:"codec"`
	Language string `json:"language"`
}

type trackJSON struct {
	Position *int           `json:"position,omitempty"`
	External *ExternalTrack `json:"external,omitempty"`
}

func (t *Track) Scan(src any) error {
	var j trackJSON
	if err := scanJSON(src, &j); err != nil {
		return err
	}
	switch {
	case j.External != nil:
		*t = Track{External: j.External}
	case j.Position != nil:
		*t = Track{Position: *j.Position}
	default:
		return fmt.Errorf("a track names neither a position nor an external stream: %v", src)
	}
	return nil
}

func (t Track) Value() (driver.Value, error) {
	j := trackJSON{External: t.External}
	if t.External == nil {
		j.Position = &t.Position
	}
	return valueJSON(j)
}

func scanJSON(src, v any) error {
	var b []byte
	switch x := src.(type) {
	case string:
		b = []byte(x)
	case []byte:
		b = x
	default:
		return fmt.Errorf("read %T from a JSON column holding %T", v, src)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("read %T from a JSON column: %w", v, err)
	}
	return nil
}

func valueJSON(v any) (driver.Value, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("write %T to a JSON column: %w", v, err)
	}
	return string(b), nil
}

// Seen is one look at a server's sessions: a poll, or a push from its
// socket.
type Seen struct {
	At time.Time
	// Polled is whether this is a poll, which counts toward a session
	// missing from two polls. A push lists only what it carried.
	Polled   bool
	Sessions []Session
	// Transcoding is whether the server re-encodes any video now, counted
	// before the session cap and including a transcode no session lists.
	// A poll sets it.
	Transcoding bool
	// Items are item keys a push said changed, which make their index
	// entries stale.
	Items []string
	// Problem is set when the poll failed. Sessions is then empty and
	// means nothing.
	Problem *Problem
}
