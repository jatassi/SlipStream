package release

import (
	"slices"
	"strings"
	"unicode"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// Prepared is a name made ready for reading, as Sonarr's ParseTitle
// prepares it before any pattern runs. Prepare is deterministic.
type Prepared struct {
	// Original is the name as given. REAL is counted in it, case-sensitively.
	Original string
	// Title is Original reversed when written backwards, without its
	// extension, site prefix and suffix, torrent suffix and trailing
	// quality bracket, with 【】 read as [].
	Title regex.Text
	// Simple is Title with resolution, codec, audio, bit depth and frame
	// sizes blanked, so no number in them reads as an episode.
	Simple regex.Text
	// Release is Sonarr's releaseTitle: Original reversed when written
	// backwards, without its extension, with 【】 read as [] and every
	// pre-substitution applied. The group reads it when the Name masks
	// nothing.
	Release regex.Text
	// Obfuscated is a hashed or obfuscated name. Its Name is zero.
	Obfuscated bool

	slow []string
}

// Prepare ports Sonarr's ParseTitle up to its pattern loop, with
// ValidateBeforeParsing deciding Obfuscated. Sonarr's rules are a
// superset of Radarr's here: Radarr reads no reversed episode marker,
// rejects fewer hashes and has no pre-substitutions.
func Prepare(name string) Prepared {
	var s scan
	p := Prepared{Original: name, Obfuscated: obfuscated(&s, name)}
	title := name
	if s.match(reversedTitle, regex.TextOf(name)) {
		base := withoutExtension(&s, name)
		title = reverse(base) + name[len(base):]
	}
	title = strings.NewReplacer("【", "[", "】", "]").Replace(withoutExtension(&s, title))
	p.Release = regex.TextOf(title)
	for _, sub := range preSubstitutions {
		p.Release = s.replace(sub, p.Release)
	}
	p.Title = unsited(&s, p.Release)
	p.Simple = unsited(&s, s.replace(simpleTitle, p.Release))
	p.slow = s.slow
	return p
}

var (
	fileExtension    = compile(upstream.SonarrFileExtensionsFileExtensionRegex)
	hashedReleases   = compileAll(upstream.SonarrParserRejectHashedReleasesRegexes)
	reversedTitle    = compile(upstream.SonarrParserReversedTitleRegex)
	preSubstitutions = compileReplaces(upstream.SonarrParserCommonPreSubstitutionRegex)
	simpleTitle      = compileReplace(upstream.SonarrParserSimpleTitleRegex)
	websitePrefix    = compileReplace(upstream.SonarrParserCommonWebsitePrefixRegex)
	websitePostfix   = compileReplace(upstream.SonarrParserCommonWebsitePostfixRegex)
	torrentSuffix    = compileReplace(upstream.SonarrParserCommonCleanTorrentSuffixRegex)
	qualityBrackets  = compile(upstream.SonarrParserCleanQualityBracketsRegex)
)

// obfuscated is Sonarr's ValidateBeforeParsing failing, but for its
// season folder rule, which is the series type's.
func obfuscated(s *scan, name string) bool {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "password") && strings.Contains(lower, "yenc") {
		return true
	}
	if !strings.ContainsFunc(name, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
		return true
	}
	t := regex.TextOf(withoutExtension(s, name))
	return slices.ContainsFunc(hashedReleases, func(f fixed) bool { return s.match(f, t) })
}

// withoutExtension is Sonarr's FileExtensions.RemoveFileExtension: name
// without a media or usenet file extension.
func withoutExtension(s *scan, name string) string {
	m, ok := s.find(fileExtension, regex.TextOf(name))
	if !ok {
		return name
	}
	ext := strings.ToLower(m.Value())
	if !slices.Contains(upstream.SonarrMediaFileExtensionsFileExtensions, ext) &&
		!slices.Contains(upstream.SonarrFileExtensionsUsenetExtensions, ext) {
		return name
	}
	return name[:len(name)-len(m.Value())]
}

func reverse(s string) string {
	r := []rune(s)
	slices.Reverse(r)
	return string(r)
}

// unsited is t without a site prefix or suffix, a torrent site's suffix
// and a trailing bracket that names a quality.
func unsited(s *scan, t regex.Text) regex.Text {
	t = s.replace(websitePrefix, t)
	t = s.replace(websitePostfix, t)
	t = s.replace(torrentSuffix, t)
	m, ok := s.find(qualityBrackets, t)
	if !ok || parseQualityName(s, m.Value()).grade == (grade{}) {
		return t
	}
	return regex.TextOf(string([]rune(t.String())[:m.Index()]))
}
