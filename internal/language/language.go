// Package language is the subtitle language vocabulary: a canonical ISO
// 639 base code, or pt-BR, the one region every launch provider files
// apart. It is not release.Language, which accepts "original" and refuses
// regions.
package language

import (
	"fmt"
	"slices"
	"strings"

	xlanguage "golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"github.com/jatassi/slipstream/internal/release"
)

// Code is a canonical base code ("en", "fr", "zh") or "pt-BR". The zero
// Code is no language. Parse and ParseTag are the only ways to make one.
type Code struct{ s string }

const brazilian = "pt-BR"

// sidecarFlags are tokens Plex, Jellyfin and Emby read as a Sidecar flag,
// so a Code whose tag is one would name the wrong kind of subtitle. "sdh"
// is also ISO 639-3 Southern Kurdish.
var sidecarFlags = []string{"sdh", "cc"}

// Parse reads an ISO 639 code of any part, or a BCP 47 tag whose region is
// BR on Portuguese. Any other region or script, "und", "mis", "mul",
// "zxx", "original", a code with no English name and a Sidecar flag word
// refuse.
func Parse(s string) (Code, error) {
	if tag, err := xlanguage.All.Parse(s); err == nil && tag.String() == brazilian {
		return Code{brazilian}, nil
	}
	l, err := release.ParseLanguage(s)
	if err != nil || l == release.Original || slices.Contains(sidecarFlags, string(l)) {
		return Code{}, fmt.Errorf("%q is not a subtitle language", s)
	}
	c := Code{string(l)}
	if c.Name() == "" {
		return Code{}, fmt.Errorf("%q is not a subtitle language", s)
	}
	return c, nil
}

func (c Code) String() string { return c.s }

func (c Code) IsZero() bool { return c.s == "" }

// Base is the release.Language an audio or embedded track carries for c.
// pt-BR's base is pt, because the media record keeps no region.
func (c Code) Base() release.Language {
	if c.s == brazilian {
		return "pt"
	}
	return release.Language(c.s)
}

// Tag is c in a Sidecar name: ISO 639-1 where one exists, else 639-2/T,
// and "pt-BR". It is the canonical code, so it reads back through
// ParseTag.
func (c Code) Tag() string { return c.s }

// Name is c's English name, "English" or "Brazilian Portuguese".
func (c Code) Name() string { return display.English.Tags().Name(c.tag()) }

// Native is c's name in itself, "Deutsch" or "日本語".
func (c Code) Native() string { return display.Self.Name(c.tag()) }

func (c Code) tag() xlanguage.Tag { return xlanguage.Make(c.s) }

// Spoken reports whether an audio track is in c's language. Portuguese
// audio speaks both pt and pt-BR, and an untagged track (the empty
// release.Language) speaks none.
func (c Code) Spoken(audio []release.Language) bool {
	return !c.IsZero() && slices.Contains(audio, c.Base())
}

// ParseTag reads a Sidecar tail's language token as Jellyfin does: a two-
// or three-letter T or B code, an English name, or a culture name such as
// "pt-br" or "zh-tw", whose region counts only for pt-BR. "hi" alone is
// Hindi. A caller that sees "hi" after another language reads it as
// hearing impaired. The Sidecar flags "sdh" and "cc" are never languages.
func ParseTag(token string) (Code, bool) {
	token = strings.ToLower(token)
	if slices.Contains(sidecarFlags, token) {
		return Code{}, false
	}
	if c, ok := englishNames[token]; ok {
		return c, true
	}
	tag, err := xlanguage.All.Parse(token)
	if err != nil {
		return Code{}, false
	}
	if tag.String() == brazilian {
		return Code{brazilian}, true
	}
	base, _ := tag.Base()
	c, err := Parse(base.String())
	return c, err == nil
}

// iso6391 is every ISO 639-1 code, whose English names ParseTag reads.
const iso6391 = "aa ab ae af ak am an ar as av ay az ba be bg bi bm bn bo br bs ca ce ch co cr cs cu cv cy " +
	"da de dv dz ee el en eo es et eu fa ff fi fj fo fr fy ga gd gl gn gu gv ha he hi ho hr ht hu hy hz " +
	"ia id ie ig ii ik io is it iu ja jv ka kg ki kj kk kl km kn ko kr ks ku kv kw ky la lb lg li ln lo lt " +
	"lu lv mg mh mi mk ml mn mr ms mt my na nb nd ne ng nl nn no nr nv ny oc oj om or os pa pi pl ps pt qu " +
	"rm rn ro ru rw sa sc sd se sg si sk sl sm sn so sq sr ss st su sv sw ta te tg th ti tk tl tn to tr ts " +
	"tt tw ty ug uk ur uz ve vi vo wa wo xh yi yo za zh zu"

var englishNames = func() map[string]Code {
	m := map[string]Code{}
	for _, s := range strings.Fields(iso6391) {
		if c, err := Parse(s); err == nil {
			m[strings.ToLower(c.Name())] = c
		}
	}
	return m
}()
