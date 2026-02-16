package renamer

import (
	"regexp"
	"strings"
	"unicode"
)

// CaseMode defines how to transform text case.
type CaseMode string

const (
	CaseDefault CaseMode = "default"
	CaseUpper   CaseMode = "upper"
	CaseLower   CaseMode = "lower"
	CaseTitle   CaseMode = "title"
)

// ApplyCase transforms the case of a string based on the mode.
func ApplyCase(s string, mode CaseMode) string {
	switch mode {
	case CaseUpper:
		return strings.ToUpper(s)
	case CaseLower:
		return strings.ToLower(s)
	case CaseTitle:
		return toTitleCase(s)
	default:
		return s
	}
}

// mediaIdentifierPattern matches S##E## patterns that should be preserved in title case.
var mediaIdentifierPattern = regexp.MustCompile(`(?i)(S\d+E\d+)`)

// toTitleCase converts string to title case, capitalizing first letter of each word.
// Preserves media identifier patterns like S01E01.
func toTitleCase(s string) string {
	if s == "" {
		return s
	}

	// Find all media identifier patterns and their positions
	matches := mediaIdentifierPattern.FindAllStringIndex(s, -1)
	if len(matches) == 0 {
		return toTitleCaseSimple(s)
	}

	// Build result, preserving media identifiers in uppercase
	var result strings.Builder
	result.Grow(len(s))
	lastEnd := 0

	for _, match := range matches {
		// Process text before the match with simple title case
		if match[0] > lastEnd {
			result.WriteString(toTitleCaseSimple(s[lastEnd:match[0]]))
		}
		// Preserve the media identifier in uppercase
		result.WriteString(strings.ToUpper(s[match[0]:match[1]]))
		lastEnd = match[1]
	}

	// Process remaining text after last match
	if lastEnd < len(s) {
		result.WriteString(toTitleCaseSimple(s[lastEnd:]))
	}

	return result.String()
}

// toTitleCaseSimple performs basic title case transformation.
func toTitleCaseSimple(s string) string {
	if s == "" {
		return s
	}

	runes := []rune(s)
	result := make([]rune, len(runes))
	capitalizeNext := true

	for i, r := range runes {
		switch {
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.':
			result[i] = r
			capitalizeNext = true
		case capitalizeNext:
			result[i] = unicode.ToUpper(r)
			capitalizeNext = false
		default:
			result[i] = unicode.ToLower(r)
		}
	}

	return string(result)
}

// ApplySeparator replaces spaces with the specified separator.
func ApplySeparator(s, separator string) string {
	if separator == "" || separator == " " {
		return s
	}
	return strings.ReplaceAll(s, " ", separator)
}

// Truncate shortens a string to the specified limit.
// Positive limit truncates from end, negative from beginning.
func Truncate(s string, limit int) string {
	if limit == 0 {
		return s
	}

	runes := []rune(s)

	if limit > 0 {
		if len(runes) > limit {
			return string(runes[:limit-1]) + "…"
		}
	} else {
		limit = -limit
		if len(runes) > limit {
			return "…" + string(runes[len(runes)-limit+1:])
		}
	}

	return s
}

// LanguageConfig holds configuration for language formatting.
type LanguageConfig struct {
	Separator    string // Default: "+"
	BracketStyle string // "square" ([EN+DE]), "round" ((EN+DE)), "none" (EN+DE)
}

// DefaultLanguageConfig returns the default language formatting configuration.
func DefaultLanguageConfig() LanguageConfig {
	return LanguageConfig{
		Separator:    "+",
		BracketStyle: "square",
	}
}

// FormatLanguages formats a list of language codes according to config.
func FormatLanguages(langs []string, config LanguageConfig) string {
	if len(langs) == 0 {
		return ""
	}

	joined := strings.Join(langs, config.Separator)

	switch config.BracketStyle {
	case "square":
		return "[" + joined + "]"
	case "round":
		return "(" + joined + ")"
	case "none":
		return joined
	default:
		return "[" + joined + "]"
	}
}

// FilterLanguages filters a list of language codes based on include/exclude patterns.
func FilterLanguages(langs []string, filter string) []string {
	if filter == "" {
		return langs
	}

	// Exclusion filter: -DE excludes German
	if strings.HasPrefix(filter, "-") {
		exclude := strings.ToUpper(strings.TrimPrefix(filter, "-"))
		result := make([]string, 0, len(langs))
		for _, lang := range langs {
			if !strings.EqualFold(lang, exclude) {
				result = append(result, lang)
			}
		}
		return result
	}

	// Inclusion filter: EN+DE includes only English and German
	includes := strings.FieldsFunc(filter, func(r rune) bool {
		return r == '+' || r == ','
	})

	includeMap := make(map[string]bool)
	for _, inc := range includes {
		includeMap[strings.ToUpper(strings.TrimSpace(inc))] = true
	}

	result := make([]string, 0, len(langs))
	for _, lang := range langs {
		if includeMap[strings.ToUpper(lang)] {
			result = append(result, lang)
		}
	}

	return result
}

// languageCodeMap maps ISO 639-2/3 codes and language names to ISO 639-1 codes.
var languageCodeMap = map[string]string{
	// ISO 639-2/3 to ISO 639-1 mappings
	"eng": "EN", "deu": "DE", "ger": "DE", "fra": "FR", "fre": "FR",
	"spa": "ES", "ita": "IT", "por": "PT", "rus": "RU", "jpn": "JA",
	"zho": "ZH", "chi": "ZH", "kor": "KO", "ara": "AR", "hin": "HI",
	"nld": "NL", "dut": "NL", "pol": "PL", "tur": "TR", "vie": "VI",
	"tha": "TH", "swe": "SV", "nor": "NO", "dan": "DA", "fin": "FI",
	"ces": "CS", "cze": "CS", "hun": "HU", "ron": "RO", "rum": "RO",
	"ell": "EL", "gre": "EL", "heb": "HE", "ukr": "UK", "ind": "ID",
	"msa": "MS", "may": "MS", "fil": "TL", "tgl": "TL",
	// Common language names to ISO 639-1 mappings
	"english": "EN", "german": "DE", "french": "FR", "spanish": "ES",
	"italian": "IT", "portuguese": "PT", "russian": "RU", "japanese": "JA",
	"chinese": "ZH", "korean": "KO", "arabic": "AR", "hindi": "HI",
	"dutch": "NL", "polish": "PL", "turkish": "TR", "vietnamese": "VI",
	"thai": "TH", "swedish": "SV", "norwegian": "NO", "danish": "DA",
	"finnish": "FI", "czech": "CS", "hungarian": "HU", "romanian": "RO",
	"greek": "EL", "hebrew": "HE", "ukrainian": "UK", "indonesian": "ID",
	"malay": "MS", "filipino": "TL", "tagalog": "TL",
}

// NormalizeLanguageCode normalizes a language code to uppercase 2-letter ISO 639-1 code.
// Supports ISO 639-1 (2-letter), ISO 639-2/3 (3-letter), and common language names.
func NormalizeLanguageCode(lang string) string {
	lang = strings.TrimSpace(lang)
	if lang == "" {
		return ""
	}

	lower := strings.ToLower(lang)

	// Check lookup table first (handles 3-letter codes and language names)
	if code, ok := languageCodeMap[lower]; ok {
		return code
	}

	// If already 2 letters, just uppercase it
	if len(lang) == 2 {
		return strings.ToUpper(lang)
	}

	// Fallback: take first 2 characters (for unknown inputs)
	if len(lang) >= 2 {
		return strings.ToUpper(lang[:2])
	}

	return strings.ToUpper(lang)
}
