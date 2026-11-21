package subtitles

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/language"
)

// Tail is a Sidecar's name after its video's stem:
// <lang>[.forced][.sdh].<ext>. The path is the stem plus the tail, so a
// rename changes no row.
type Tail struct {
	Language   language.Code
	Forced, HI bool
	// Ext is "srt", "ass" or "ssa" for a Sidecar SlipStream writes, and
	// also "vtt" for a found one.
	Ext string
}

// foundExts are the companions a survey reads as subtitles.
var foundExts = []string{"srt", "ass", "ssa", "vtt"}

// String is t as SlipStream writes it: never ".hi", never ".default".
func (t Tail) String() string {
	var b strings.Builder
	b.WriteString(t.Language.Tag())
	if t.Forced {
		b.WriteString(".forced")
	}
	if t.HI {
		b.WriteString(".sdh")
	}
	b.WriteString(".")
	b.WriteString(t.Ext)
	return b.String()
}

// Want is the language and kind t's subtitle meets.
func (t Tail) Want() Want { return Want{Language: t.Language, Forced: t.Forced} }

// ParseTail reads a companion's tail right to left as Jellyfin does:
// "en.hi.srt" is English hearing impaired, "hi.srt" is Hindi, "en.sdh.srt"
// and "en.cc.srt" are hearing impaired, and "forced" and "foreign" set
// Forced. False when no token names a language or the extension is none
// of srt, ass, ssa and vtt.
func ParseTail(tail string) (Tail, bool) {
	tokens := strings.Split(strings.ToLower(tail), ".")
	t := Tail{Ext: tokens[len(tokens)-1]}
	if len(tokens) < 2 || !slices.Contains(foundExts, t.Ext) {
		return Tail{}, false
	}
	words := tokens[:len(tokens)-1]
	for i := len(words) - 1; i >= 0 && t.Language.IsZero(); i-- {
		switch w := words[i]; w {
		case "forced", "foreign":
			t.Forced = true
		case "sdh", "cc":
			t.HI = true
		case "hi":
			if slices.ContainsFunc(words[:i], named) {
				t.HI = true
				continue
			}
			t.Language, _ = language.ParseTag(w)
		default:
			t.Language, _ = language.ParseTag(w)
		}
	}
	return t, !t.Language.IsZero()
}

func named(token string) bool {
	_, ok := language.ParseTag(token)
	return ok
}
