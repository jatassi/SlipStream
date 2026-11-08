package golden

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/release"
)

// Quality is an upstream quality as SlipStream reads it. An unknown
// Resolution isn't checked: Sonarr's DVD is DVD at any resolution.
type Quality struct {
	Source     release.Fact[release.Origin]
	Modifier   release.Fact[release.Modifier]
	Resolution release.Fact[release.Resolution]
}

func sonarr(o release.Origin, m release.Fact[release.Modifier], r release.Resolution) Quality {
	return Quality{release.Known(o), m, release.Known(r)}
}

var (
	none  = release.Fact[release.Modifier]{}
	remux = release.Known(release.ModRemux)
	rawHD = release.Known(release.ModRawHD)
)

// sonarrQualities are Sonarr's Quality values by name. TelevisionRaw reads
// as tv with rawhd, and BlurayRaw as bluray with remux.
var sonarrQualities = map[string]Quality{
	"Unknown":          {},
	"SDTV":             sonarr(release.OriginTV, none, release.R480p),
	"DVD":              {Source: release.Known(release.OriginDVD)},
	"WEBDL480p":        sonarr(release.OriginWEBDL, none, release.R480p),
	"WEBRip480p":       sonarr(release.OriginWEBRip, none, release.R480p),
	"Bluray480p":       sonarr(release.OriginBluray, none, release.R480p),
	"Bluray576p":       sonarr(release.OriginBluray, none, release.R576p),
	"HDTV720p":         sonarr(release.OriginTV, none, release.R720p),
	"HDTV1080p":        sonarr(release.OriginTV, none, release.R1080p),
	"HDTV2160p":        sonarr(release.OriginTV, none, release.R2160p),
	"RAWHD":            sonarr(release.OriginTV, rawHD, release.R1080p),
	"WEBDL720p":        sonarr(release.OriginWEBDL, none, release.R720p),
	"WEBRip720p":       sonarr(release.OriginWEBRip, none, release.R720p),
	"WEBDL1080p":       sonarr(release.OriginWEBDL, none, release.R1080p),
	"WEBRip1080p":      sonarr(release.OriginWEBRip, none, release.R1080p),
	"WEBDL2160p":       sonarr(release.OriginWEBDL, none, release.R2160p),
	"WEBRip2160p":      sonarr(release.OriginWEBRip, none, release.R2160p),
	"Bluray720p":       sonarr(release.OriginBluray, none, release.R720p),
	"Bluray1080p":      sonarr(release.OriginBluray, none, release.R1080p),
	"Bluray2160p":      sonarr(release.OriginBluray, none, release.R2160p),
	"Bluray1080pRemux": sonarr(release.OriginBluray, remux, release.R1080p),
	"Bluray2160pRemux": sonarr(release.OriginBluray, remux, release.R2160p),
}

// SonarrQuality is the Quality of Sonarr's Quality.<name>.
func SonarrQuality(name string) Quality {
	q, ok := sonarrQualities[name]
	if !ok {
		panic(fmt.Sprintf("Sonarr's Quality.%s has no reading here", name))
	}
	return q
}

// SonarrNames are Sonarr's display names of its qualities, which a name
// may carry in brackets.
var SonarrNames = map[string]string{
	"Unknown": "Unknown", "SDTV": "SDTV", "DVD": "DVD", "WEBDL480p": "WEBDL-480p", "WEBRip480p": "WEBRip-480p",
	"Bluray480p": "Bluray-480p", "Bluray576p": "Bluray-576p", "HDTV720p": "HDTV-720p", "HDTV1080p": "HDTV-1080p",
	"HDTV2160p": "HDTV-2160p", "RAWHD": "Raw-HD", "WEBDL720p": "WEBDL-720p", "WEBRip720p": "WEBRip-720p",
	"WEBDL1080p": "WEBDL-1080p", "WEBRip1080p": "WEBRip-1080p", "WEBDL2160p": "WEBDL-2160p", "WEBRip2160p": "WEBRip-2160p",
	"Bluray720p": "Bluray-720p", "Bluray1080p": "Bluray-1080p", "Bluray2160p": "Bluray-2160p",
	"Bluray1080pRemux": "Bluray-1080p Remux", "Bluray2160pRemux": "Bluray-2160p Remux",
}

// Read is nil when f reads as q: the same source and modifier, and the
// same resolution when q names one.
func (q Quality) Read(f release.Facts) error {
	errs := []error{Same("source", f.Source, q.Source), Same("modifier", f.Modifier, q.Modifier)}
	if _, ok := q.Resolution.Get(); ok {
		errs = append(errs, Same("resolution", f.Resolution, q.Resolution))
	}
	return errors.Join(errs...)
}

func language(name string) (release.Language, bool) {
	if name == "Unknown" {
		return "", false
	}
	l, ok := release.UpstreamLanguage(name)
	if !ok {
		panic(fmt.Sprintf("upstream's Language.%s has no reading here", name))
	}
	return l, true
}

// LanguageCheck is check on got, or, where an override says what
// SlipStream reads, whether got is exactly that.
func LanguageCheck(r Row, got release.Fact[[]release.Language], check func(release.Fact[[]release.Language]) error) error {
	var ours []string
	if r.Reads(&ours) {
		return Languages(got, ours...)
	}
	return check(got)
}

// HasLanguages is nil when got holds each of names. Upstream's Unknown
// means got is unknown.
func HasLanguages(got release.Fact[[]release.Language], names ...string) error {
	have, known := got.Get()
	var errs []error
	for _, name := range names {
		l, ok := language(name)
		switch {
		case !ok && known:
			errs = append(errs, fmt.Errorf("languages read as %s, want none", show(have)))
		case ok && !slices.Contains(have, l):
			errs = append(errs, fmt.Errorf("languages read as %s, want %s among them", show(have), l))
		}
	}
	return errors.Join(errs...)
}

// Languages is nil when got holds exactly names, in any order.
func Languages(got release.Fact[[]release.Language], names ...string) error {
	var want []release.Language
	for _, name := range names {
		if l, ok := language(name); ok && !slices.Contains(want, l) {
			want = append(want, l)
		}
	}
	have, _ := got.Get()
	have = slices.Clone(have)
	slices.Sort(have)
	slices.Sort(want)
	return Same("languages", have, want)
}

// Cleaning are the checks of the title-cleaning methods Sonarr's
// NormalizeSeriesTitleFixture and Radarr's NormalizeTitleFixture share,
// against a Content Type's CleanTitle.
func Cleaning(clean func(string) string) map[string]Check {
	pair := func(dirty, want string) Check {
		return func(r Row) error { return Same("clean title", clean(r.String(dirty)), r.String(want)) }
	}
	formats := func(layouts []string, want func(word string) string) Check {
		return func(r Row) error {
			word := r.String("word")
			var errs []error
			for _, l := range layouts {
				dirty := strings.ReplaceAll(l, "{0}", word)
				errs = append(errs, Same("clean title of "+dirty, clean(dirty), want(word)))
			}
			return errors.Join(errs...)
		}
	}
	return map[string]Check{
		"should_normalize_series_title":                     pair("parsedSeriesName", "seriesName"),
		"should_remove_special_characters_and_casing":       pair("dirty", "clean"),
		"should_not_remove_from_the_beginning_of_the_title": pair("parsedSeriesName", "seriesName"),
		"should_remove_common_words_from_middle_of_title": formats(
			[]string{"word.{0}.word", "word {0} word", "word-{0}-word"},
			func(string) string { return "wordword" }),
		"should_not_remove_common_words_from_end_of_title": formats(
			[]string{"word.word.{0}", "word-word-{0}", "word-word {0}"},
			func(w string) string { return "wordword" + strings.ToLower(w) }),
		"should_not_remove_common_words_in_the_middle_of_word": formats(
			[]string{"word.{0}word", "word {0}word", "word-{0}word", "word{0}.word", "word{0}-word"},
			func(w string) string { return "word" + strings.ToLower(w) + "word" }),
		"should_not_clean_word_from_beginning_of_string": formats(
			[]string{"{0}.word.word", "{0}-word-word", "{0} word word"},
			func(w string) string { return w + "wordword" }),
	}
}

// RandomNames are the 500 names of length random letters and digits that
// upstream's should_not_parse_random reads, drawn from rng.
func RandomNames(rng *rand.Rand, length int) []string {
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	names := make([]string, 500)
	for n := range names {
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rng.IntN(len(charset))]
		}
		names[n] = string(b)
	}
	return names
}

// Unread is nil when nothing in n was read, which is what upstream's
// ParseTitle answering null is in SlipStream.
func Unread(n release.Name) error {
	if len(n.Titles) > 0 || len(n.Marks) > 0 {
		return Same("name", n, release.Name{})
	}
	return nil
}

// FirstTitle is n's first title, upstream's SeriesTitle or
// PrimaryMovieTitle, or "" when n has none.
func FirstTitle(n release.Name) string {
	if len(n.Titles) == 0 {
		return ""
	}
	return n.Titles[0]
}
