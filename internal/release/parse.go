package release

import (
	"strings"
	"unicode/utf8"
)

// Titles are read as words: apostrophes are dropped, ASCII letters fold
// to lower case, and every other ASCII mark or space splits words. A
// phrase matches whole words only, so a bare "ts" never matches inside
// "Pirates". QualitySQL mirrors this rule exactly.
const (
	separators = " !\"#$%&()*+,-./:;<=>?@[\\]^_`{|}~"
	apostrophe = '\''
	rightQuote = '’'
)

// sourceWords are the phrases that name each source, the first source
// named winning, so a cinema source wins over anything else in a title.
var sourceWords = []struct {
	source Source
	words  []string
}{
	{Cinema, []string{"cam", "hdcam", "ts", "hdts", "telesync"}},
	{Remux, []string{"remux", "bdremux"}},
	{Bluray, []string{"bluray", "blu ray", "bdrip", "brrip"}},
	{WEBRip, []string{"webrip", "web rip"}},
	{WEBDL, []string{"webdl", "web dl", "web"}},
	{HDTV, []string{"hdtv"}},
	{TV, []string{"sdtv", "tvrip"}},
	{DVD, []string{"dvd", "dvdrip"}},
}

var resolutionWords = map[string]Resolution{
	"2160p": R2160p, "4k": R2160p, "uhd": R2160p,
	"1080p": R1080p, "720p": R720p, "576p": R576p, "480p": R480p,
}

// Title is what this package reads from a release title: its quality from
// the source and the first resolution it names, and its cut. cuts is every
// cut SlipStream knows. Size and Coverage are the caller's.
func Title(title string, cuts []Cut) Facts {
	w := words(title)
	return Facts{Quality: qualityFact(sourceIn(w), resolutionIn(w)), Cut: cutIn(w, cuts)}
}

// File is a file's Facts: the source and cut its name names, which for a
// downloaded file is its Grab's release title, and the resolution its
// probe measured. Size, Coverage and the probe's other facts are the
// caller's.
func File(name string, cuts []Cut, probed Resolution) Facts {
	w := words(name)
	return Facts{Quality: qualityFact(sourceIn(w), Known(probed)), Cut: cutIn(w, cuts)}
}

func qualityFact(s Fact[Source], r Fact[Resolution]) Fact[Quality] {
	if q, ok := QualityOf(s, r); ok {
		return Known(q)
	}
	return Fact[Quality]{}
}

// words is s as space-separated words with a space at each end, so
// " "+phrase+" " finds a phrase on word boundaries.
func words(s string) string {
	var b strings.Builder
	b.WriteByte(' ')
	split := true
	for _, r := range s {
		switch {
		case r == apostrophe || r == rightQuote:
			continue
		case r < utf8.RuneSelf && strings.ContainsRune(separators, r):
			if !split {
				b.WriteByte(' ')
				split = true
			}
		default:
			if 'A' <= r && r <= 'Z' {
				r += 'a' - 'A'
			}
			b.WriteRune(r)
			split = false
		}
	}
	if !split {
		b.WriteByte(' ')
	}
	return b.String()
}

func phrase(w string) string { return " " + w + " " }

func sourceIn(w string) Fact[Source] {
	for _, s := range sourceWords {
		for _, p := range s.words {
			if strings.Contains(w, phrase(p)) {
				return Known(s.source)
			}
		}
	}
	return Fact[Source]{}
}

func resolutionIn(w string) Fact[Resolution] {
	for _, word := range strings.Fields(w) {
		if r, ok := resolutionWords[word]; ok {
			return Known(r)
		}
	}
	return Fact[Resolution]{}
}

// cutIn is the cut whose alias w names, the longest alias winning, then
// the leftmost, then the first cut in cuts. No alias named is Standard.
func cutIn(w string, cuts []Cut) CutKey {
	best, bestLen, bestAt := Standard, 0, 0
	for _, c := range cuts {
		for _, alias := range c.Aliases {
			p := words(alias)
			if strings.TrimSpace(p) == "" {
				continue
			}
			at := strings.Index(w, p)
			if at < 0 {
				continue
			}
			if len(p) > bestLen || len(p) == bestLen && at < bestAt {
				best, bestLen, bestAt = c.Key, len(p), at
			}
		}
	}
	return best
}
