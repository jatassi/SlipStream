package encode

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
)

// Selection is the source tracks a made file keeps, by index into the
// record's Audio and Subtitles. When the audio is re-encoded, Audio is the
// one track it is made from.
type Selection struct {
	Audio     []int
	Subtitles []int
}

// Tracks is what a made file keeps of m. Audio and subtitle tracks are kept
// in keep's languages, or all with a nil keep, and an untagged track is
// always kept. Commentary is dropped when s says so. A re-encode takes the
// first kept default track, else the first kept. MP4 drops image
// subtitles.
func Tracks(m media.Record, s Settings, keep []language.Code) Selection {
	kept := func(l release.Language) bool {
		return keep == nil || l == "" || slices.ContainsFunc(keep, func(c language.Code) bool { return c.Spoken([]release.Language{l}) })
	}
	var sel Selection
	for i, a := range m.Audio {
		if kept(a.Language) && (!s.DropCommentary || !commentary(a)) {
			sel.Audio = append(sel.Audio, i)
		}
	}
	if s.Audio != CopyAudio && len(sel.Audio) > 0 {
		from := sel.Audio[0]
		if i := slices.IndexFunc(sel.Audio, func(n int) bool { return m.Audio[n].Default }); i >= 0 {
			from = sel.Audio[i]
		}
		sel.Audio = []int{from}
	}
	for i, t := range m.Subtitles {
		if kept(t.Language) && (s.Container != MP4 || !t.Image) {
			sel.Subtitles = append(sel.Subtitles, i)
		}
	}
	return sel
}

// commentary reports whether a's title names it a commentary track.
func commentary(a media.Audio) bool { return strings.Contains(strings.ToLower(a.Title), "commentary") }

// Unfit is a source stream the settings' container can't hold, by its
// index into the record's Audio.
type Unfit struct {
	Code   string
	Stream int
}

// ContainerAudio is copied audio MP4 can't hold: TrueHD or DTS.
const ContainerAudio = "container_audio"

var notInMP4 = []string{"truehd", "dts"}

// Fits is nil when every stream s keeps of m fits s's container.
func Fits(s Settings, m media.Record, keep []language.Code) *Unfit {
	if s.Container != MP4 || s.Audio != CopyAudio {
		return nil
	}
	for _, i := range Tracks(m, s, keep).Audio {
		if slices.Contains(notInMP4, m.Audio[i].Codec) {
			return &Unfit{Code: ContainerAudio, Stream: i}
		}
	}
	return nil
}
