package parseutil

import (
	"regexp"
	"strings"
)

var (
	// Release group pattern - typically at the end after a dash: -GroupName
	releaseGroupPattern = regexp.MustCompile(`-([A-Za-z0-9]+)(?:\.[a-z0-9]{2,4})?$`)

	releaseGroupFalsePositives = map[string]bool{
		"x264": true, "x265": true, "hevc": true, "avc": true,
		"h264": true, "h265": true, "xvid": true, "divx": true,
		"av1": true, "vp9": true, "mkv": true, "mp4": true, "avi": true,
	}

	// Revision patterns
	revisionPatterns = map[string]*regexp.Regexp{
		"Proper": regexp.MustCompile(`(?i)(^|[.\s\-_])proper([.\s\-_]|$)`),
		"REPACK": regexp.MustCompile(`(?i)(^|[.\s\-_])repack([.\s\-_]|$)`),
		"REAL":   regexp.MustCompile(`(?i)(^|[.\s\-_])real([.\s\-_]|$)`),
		"RERIP":  regexp.MustCompile(`(?i)(^|[.\s\-_])rerip([.\s\-_]|$)`),
	}
	revisionOrder = []string{"Proper", "REPACK", "REAL", "RERIP"}

	// Edition patterns
	editionPatterns = map[string]*regexp.Regexp{
		"Director's Cut":      regexp.MustCompile(`(?i)(^|[.\s\-_])directors?[.\s\-_]?cut([.\s\-_]|$)`),
		"Extended":            regexp.MustCompile(`(?i)(^|[.\s\-_])extended([.\s\-_]|$)`),
		"Extended Cut":        regexp.MustCompile(`(?i)(^|[.\s\-_])extended[.\s\-_]?cut([.\s\-_]|$)`),
		"Theatrical":          regexp.MustCompile(`(?i)(^|[.\s\-_])theatrical[.\s\-_]?(?:cut|edition)?([.\s\-_]|$)`),
		"Unrated":             regexp.MustCompile(`(?i)(^|[.\s\-_])unrated([.\s\-_]|$)`),
		"Uncut":               regexp.MustCompile(`(?i)(^|[.\s\-_])uncut([.\s\-_]|$)`),
		"Ultimate Cut":        regexp.MustCompile(`(?i)(^|[.\s\-_])ultimate[.\s\-_]?cut([.\s\-_]|$)`),
		"Final Cut":           regexp.MustCompile(`(?i)(^|[.\s\-_])final[.\s\-_]?cut([.\s\-_]|$)`),
		"Special Edition":     regexp.MustCompile(`(?i)(^|[.\s\-_])special[.\s\-_]?edition([.\s\-_]|$)`),
		"Collector's Edition": regexp.MustCompile(`(?i)(^|[.\s\-_])collectors?[.\s\-_]?edition([.\s\-_]|$)`),
		"Anniversary Edition": regexp.MustCompile(`(?i)(^|[.\s\-_])anniversary[.\s\-_]?edition([.\s\-_]|$)`),
		"Criterion":           regexp.MustCompile(`(?i)(^|[.\s\-_])criterion([.\s\-_]|$)`),
		"IMAX":                regexp.MustCompile(`(?i)(^|[.\s\-_])imax([.\s\-_]|$)`),
		"3D":                  regexp.MustCompile(`(?i)(^|[.\s\-_])3d([.\s\-_]|$)`),
		"Remastered":          regexp.MustCompile(`(?i)(^|[.\s\-_])remastered([.\s\-_]|$)`),
		"Restored":            regexp.MustCompile(`(?i)(^|[.\s\-_])restored([.\s\-_]|$)`),
	}
	editionOrder = []string{
		"Director's Cut", "Extended Cut", "Extended", "Theatrical",
		"Unrated", "Uncut", "Ultimate Cut", "Final Cut",
		"Special Edition", "Collector's Edition", "Anniversary Edition",
		"Criterion", "IMAX", "3D", "Remastered", "Restored",
	}

	// Language patterns - detect non-English releases
	languagePatterns = map[string]*regexp.Regexp{
		"German":     regexp.MustCompile(`(?i)(^|[.\s\-_])(german|deutsch|ger|deu)([.\s\-_]|$)`),
		"French":     regexp.MustCompile(`(?i)(^|[.\s\-_])(french|français|fra|fre)([.\s\-_]|$)`),
		"Spanish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(spanish|español|spa|esp)([.\s\-_]|$)`),
		"Italian":    regexp.MustCompile(`(?i)(^|[.\s\-_])(italian|italiano|ita)([.\s\-_]|$)`),
		"Portuguese": regexp.MustCompile(`(?i)(^|[.\s\-_])(portuguese|português|por|pt-br)([.\s\-_]|$)`),
		"Russian":    regexp.MustCompile(`(?i)(^|[.\s\-_])(russian|русский|rus)([.\s\-_]|$)`),
		"Japanese":   regexp.MustCompile(`(?i)(^|[.\s\-_])(japanese|日本語|jpn|jap)([.\s\-_]|$)`),
		"Korean":     regexp.MustCompile(`(?i)(^|[.\s\-_])(korean|한국어|kor)([.\s\-_]|$)`),
		"Chinese":    regexp.MustCompile(`(?i)(^|[.\s\-_])(chinese|中文|chi|chs|cht|mandarin|cantonese)([.\s\-_]|$)`),
		"Dutch":      regexp.MustCompile(`(?i)(^|[.\s\-_])(dutch|nederlands|nld|dut)([.\s\-_]|$)`),
		"Polish":     regexp.MustCompile(`(?i)(^|[.\s\-_])(polish|polski|pol)([.\s\-_]|$)`),
		"Swedish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(swedish|svenska|swe)([.\s\-_]|$)`),
		"Norwegian":  regexp.MustCompile(`(?i)(^|[.\s\-_])(norwegian|norsk|nor)([.\s\-_]|$)`),
		"Danish":     regexp.MustCompile(`(?i)(^|[.\s\-_])(danish|dansk|dan)([.\s\-_]|$)`),
		"Finnish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(finnish|suomi|fin)([.\s\-_]|$)`),
		"Turkish":    regexp.MustCompile(`(?i)(^|[.\s\-_])(turkish|türkçe|tur)([.\s\-_]|$)`),
		"Hindi":      regexp.MustCompile(`(?i)(^|[.\s\-_])(hindi|hin)([.\s\-_]|$)`),
		"Arabic":     regexp.MustCompile(`(?i)(^|[.\s\-_])(arabic|العربية|ara)([.\s\-_]|$)`),
		"Hebrew":     regexp.MustCompile(`(?i)(^|[.\s\-_])(hebrew|עברית|heb)([.\s\-_]|$)`),
		"Czech":      regexp.MustCompile(`(?i)(^|[.\s\-_])(czech|čeština|cze|ces)([.\s\-_]|$)`),
		"Hungarian":  regexp.MustCompile(`(?i)(^|[.\s\-_])(hungarian|magyar|hun)([.\s\-_]|$)`),
		"Greek":      regexp.MustCompile(`(?i)(^|[.\s\-_])(greek|ελληνικά|gre|ell)([.\s\-_]|$)`),
		"Thai":       regexp.MustCompile(`(?i)(^|[.\s\-_])(thai|ไทย|tha)([.\s\-_]|$)`),
		"Vietnamese": regexp.MustCompile(`(?i)(^|[.\s\-_])(vietnamese|tiếng việt|vie)([.\s\-_]|$)`),
		"Indonesian": regexp.MustCompile(`(?i)(^|[.\s\-_])(indonesian|bahasa indonesia|ind)([.\s\-_]|$)`),
		"Romanian":   regexp.MustCompile(`(?i)(^|[.\s\-_])(romanian|română|ron|rum)([.\s\-_]|$)`),
		"Ukrainian":  regexp.MustCompile(`(?i)(^|[.\s\-_])(ukrainian|українська|ukr)([.\s\-_]|$)`),
	}
)

// ParseReleaseGroup extracts the release group from a filename (e.g., "-SPARKS" -> "SPARKS").
func ParseReleaseGroup(filename string) string {
	match := releaseGroupPattern.FindStringSubmatch(filename)
	if match == nil {
		return ""
	}
	if releaseGroupFalsePositives[strings.ToLower(match[1])] {
		return ""
	}
	return match[1]
}

// ParseRevision detects PROPER, REPACK, REAL, RERIP in a filename.
func ParseRevision(filename string) string {
	for _, rev := range revisionOrder {
		if pattern, ok := revisionPatterns[rev]; ok && pattern.MatchString(filename) {
			return rev
		}
	}
	return ""
}

// ParseEdition detects edition tags (Director's Cut, Extended, etc.) in a filename.
func ParseEdition(filename string) string {
	var editions []string
	for _, ed := range editionOrder {
		if pattern, ok := editionPatterns[ed]; ok && pattern.MatchString(filename) {
			editions = append(editions, ed)
		}
	}
	if len(editions) > 0 {
		return strings.Join(editions, " ")
	}
	return ""
}

// ParseLanguages detects non-English language tags in a filename.
func ParseLanguages(filename string) []string {
	var languages []string
	for lang, pattern := range languagePatterns {
		if pattern.MatchString(filename) {
			languages = append(languages, lang)
		}
	}
	return languages
}
