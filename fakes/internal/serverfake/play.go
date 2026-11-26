package serverfake

import (
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// Play is a <kind>-play cue's arguments: a session on the file at Path,
// as the server stores it, playing the file at Version, Path's when
// empty. PlayerKey is required, and every stream decision too.
type Play struct {
	Path             string                `json:"path"`
	Version          string                `json:"version,omitempty"`
	User             string                `json:"user"`
	Player           string                `json:"player"`
	PlayerKey        string                `json:"playerKey"`
	Platform         string                `json:"platform,omitempty"`
	Product          string                `json:"product,omitempty"`
	Video            link.StreamDecision   `json:"video"`
	Audio            link.StreamDecision   `json:"audio"`
	Subtitles        link.SubtitleDecision `json:"subtitles"`
	ContainerChanged bool                  `json:"containerChanged,omitempty"`
	Received         mediaserver.Received  `json:"received"`
	AudioTrack       *int                  `json:"audioTrack,omitempty"`
	SubtitleTrack    *int                  `json:"subtitleTrack,omitempty"`
	Reasons          []string              `json:"reasons,omitempty"`
	Hardware         bool                  `json:"hardware,omitempty"`
	// Unreported is nil for the product's default.
	Unreported *bool `json:"unreported,omitempty"`
}

// Core is a's play for user, unreported by default when the cue doesn't
// say.
func (a Play) Core(user string, unreported bool) (mediaserver.Play, error) {
	p := mediaserver.Play{
		User: user, Player: mediaserver.Player{Key: a.PlayerKey, Name: a.Player, Product: a.Product, Platform: a.Platform},
		Received: a.Received, AudioTrack: a.AudioTrack, SubtitleTrack: a.SubtitleTrack, Reasons: a.Reasons,
		Hardware: a.Hardware, Unreported: unreported,
	}
	if a.Unreported != nil {
		p.Unreported = *a.Unreported
	}
	var err error
	if p.Path, err = link.ParsePath(a.Path); err != nil {
		return mediaserver.Play{}, err
	}
	if a.Version != "" {
		if p.Version, err = link.ParsePath(a.Version); err != nil {
			return mediaserver.Play{}, err
		}
	}
	if p.Streams.Video, err = link.ParseStreamDecision(string(a.Video)); err != nil {
		return mediaserver.Play{}, err
	}
	if p.Streams.Audio, err = link.ParseStreamDecision(string(a.Audio)); err != nil {
		return mediaserver.Play{}, err
	}
	if p.Streams.Subtitles, err = link.ParseSubtitleDecision(string(a.Subtitles)); err != nil {
		return mediaserver.Play{}, err
	}
	p.Streams.ContainerChanged = a.ContainerChanged
	return p, nil
}

// Track is the cue's track for a conformance play's chosen track: nil for
// none, -1 for an external one, else its position.
func Track(t *link.Track) *int {
	switch {
	case t == nil:
		return nil
	case t.External != nil:
		external := -1
		return &external
	}
	return &t.Position
}
