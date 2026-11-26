package playback

import (
	"slices"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// FactsLike is a sibling's facts with the tracks a play of it would pick:
// its audio in like's audio language, else its default; its subtitle in
// like's subtitle language, a Sidecar included, when a subtitle showed.
func FactsLike(r media.Record, sidecars []Sidecar, like Facts) Facts {
	var audio, subtitle *link.Track
	if like.Audio != nil {
		if i := slices.IndexFunc(r.Audio, func(a media.Audio) bool { return string(a.Language) == like.Audio.Language }); i >= 0 {
			audio = &link.Track{Position: i}
		}
	}
	if like.Subtitle != nil {
		lang := like.Subtitle.Language
		if i := slices.IndexFunc(r.Subtitles, func(s media.Subtitle) bool { return string(s.Language) == lang }); i >= 0 {
			subtitle = &link.Track{Position: i}
		} else if i := slices.IndexFunc(sidecars, func(s Sidecar) bool { return s.Language == lang }); i >= 0 {
			subtitle = &link.Track{External: &link.ExternalTrack{Codec: string(sidecars[i].Codec), Language: lang}}
		}
	}
	return FactsOf(r, audio, subtitle)
}
