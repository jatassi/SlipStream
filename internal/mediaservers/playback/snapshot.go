// Package playback is the Playback report: why each player transcoded the
// files it played, and which file or Target would avoid it. It does no IO.
package playback

import (
	"database/sql/driver"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"slices"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Snapshot is the file a play started on, as it stood then.
type Snapshot struct {
	Work    works.WorkID `json:"work"`
	Item    works.ItemID `json:"item"`
	Label   string       `json:"label"`
	Targets []TargetRef  `json:"targets"`
	Facts   Facts        `json:"facts"`
}

func (s *Snapshot) Scan(src any) error {
	var b []byte
	switch x := src.(type) {
	case string:
		b = []byte(x)
	case []byte:
		b = x
	default:
		return fmt.Errorf("read a play's snapshot from %T", src)
	}
	if err := jsonv2.Unmarshal(b, s); err != nil {
		return fmt.Errorf("read a play's snapshot: %w", err)
	}
	return nil
}

func (s Snapshot) Value() (driver.Value, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("write a play's snapshot: %w", err)
	}
	return string(b), nil
}

// Met is whether the file met t when the play started.
func (s Snapshot) Met(t targets.TargetID) bool {
	return slices.ContainsFunc(s.Targets, func(r TargetRef) bool { return r.ID == t.String() })
}

type Facts struct {
	Container release.Container `json:"container"`
	Bitrate   int64             `json:"bitrate"`
	Video     Video             `json:"video"`
	// Audio is the played track, nil when the file has none.
	Audio *Audio `json:"audio"`
	// Subtitle is the shown track, nil when none shows.
	Subtitle *Subtitle `json:"subtitle"`
}

// SubtitleCodec is the shown track's codec, "" for none.
func (f Facts) SubtitleCodec() release.SubtitleCodec {
	if f.Subtitle == nil {
		return ""
	}
	return f.Subtitle.Codec
}

type Video struct {
	Codec    release.VideoCodec `json:"codec"`
	Profile  string             `json:"profile"`
	BitDepth int                `json:"bitDepth"`
	// Resolution is the class, from width and height.
	Resolution release.Resolution `json:"resolution"`
	// Height is the raw height, compared with what the player received.
	Height int `json:"height"`
	// HDR is every HDR format the file carries: a Dolby Vision 8.1 file is
	// DV and HDR10.
	HDR                []release.HDR `json:"hdr"`
	DolbyVisionProfile int           `json:"dolbyVisionProfile"`
}

type Audio struct {
	Codec    release.AudioCodec `json:"codec"`
	Channels int                `json:"channels"`
	Language string             `json:"language"`
}

// Layout is the track's channel layout, "" for a count the vocabulary has
// no word for.
func (a Audio) Layout() release.Channels { return media.LayoutOf(a.Channels) }

type Subtitle struct {
	Codec    release.SubtitleCodec `json:"codec"`
	Image    bool                  `json:"image"`
	External bool                  `json:"external"`
	Language string                `json:"language"`
}

// TargetRef is a Target by id and name, kept in a snapshot and served.
type TargetRef struct {
	ID   string `json:"id" format:"uuid"`
	Name string `json:"name"`
}
