package link

import (
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// Names maps a server's codec and container names into the release
// vocabulary. Each kind keeps one, beside its own parsing. A name a table
// lacks passes through lowercased, so it never equals a vocabulary word.
type Names struct {
	Containers map[string]release.Container
	Video      map[string]release.VideoCodec
	Audio      map[string]release.AudioCodec
	Subtitles  map[string]release.SubtitleCodec
}

func (n Names) Container(s string) string { return word(n.Containers, s) }

func (n Names) VideoCodec(s string) string { return word(n.Video, s) }

func (n Names) AudioCodec(s string) string { return word(n.Audio, s) }

func (n Names) SubtitleCodec(s string) string { return word(n.Subtitles, s) }

// Received is a transcode's received facts in these words; bitrate in bits
// a second.
func (n Names) Received(container, video, audio string, height, channels int, bitrate int64) Received {
	return Received{
		Container: n.Container(container), VideoCodec: n.VideoCodec(video), AudioCodec: n.AudioCodec(audio),
		Height: height, Channels: channels, Bitrate: bitrate,
	}
}

func word[W ~string](table map[string]W, s string) string {
	s = strings.ToLower(s)
	if w, ok := table[s]; ok {
		return string(w)
	}
	return s
}
