package renamer

import (
	"strings"
	"unicode"
)

// IllegalCharacters are characters not allowed in filenames on most filesystems.
var IllegalCharacters = []rune{'\\', '/', ':', '*', '?', '"', '<', '>', '|'}

// ColonReplacement defines how to replace colons in filenames.
type ColonReplacement string

const (
	ColonDelete         ColonReplacement = "delete"
	ColonDash           ColonReplacement = "dash"
	ColonSpaceDash      ColonReplacement = "space_dash"
	ColonSpaceDashSpace ColonReplacement = "space_dash_space"
	ColonSmart          ColonReplacement = "smart"
	ColonCustom         ColonReplacement = "custom"
)

// ReplaceIllegalCharacters replaces or removes illegal filesystem characters.
func ReplaceIllegalCharacters(s string, replace bool, colonMode ColonReplacement, customColon string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))

	runes := []rune(s)
	for i, r := range runes {
		if r == ':' {
			replacement := handleColon(s, i, colonMode, customColon)
			result.WriteString(replacement)
			continue
		}

		if isIllegalChar(r) {
			if replace {
				// Replace with appropriate alternative
				result.WriteRune(getReplacement(r))
			}
			// If not replace, character is simply omitted
			continue
		}

		result.WriteRune(r)
	}

	return cleanupSpaces(result.String())
}

// handleColon returns the replacement for a colon based on the mode.
func handleColon(s string, pos int, mode ColonReplacement, custom string) string {
	switch mode {
	case ColonDelete:
		return ""
	case ColonDash:
		return "-"
	case ColonSpaceDash:
		return " -"
	case ColonSpaceDashSpace:
		return " - "
	case ColonSmart:
		return smartColonReplace(s, pos)
	case ColonCustom:
		if custom != "" {
			return custom
		}
		return "-"
	default:
		return "-"
	}
}

// smartColonReplace performs context-aware colon replacement.
// Uses " - " when between words, "-" when adjacent to other punctuation.
func smartColonReplace(s string, pos int) string {
	runes := []rune(s)

	// Check character before colon
	var prevIsSpace, prevIsWord bool
	if pos > 0 {
		prev := runes[pos-1]
		prevIsSpace = unicode.IsSpace(prev)
		prevIsWord = unicode.IsLetter(prev) || unicode.IsDigit(prev)
	}

	// Check character after colon
	var nextIsSpace, nextIsWord bool
	if pos < len(runes)-1 {
		next := runes[pos+1]
		nextIsSpace = unicode.IsSpace(next)
		nextIsWord = unicode.IsLetter(next) || unicode.IsDigit(next)
	}

	// If colon is between word characters (possibly with spaces), use " - "
	if prevIsWord && (nextIsWord || nextIsSpace) {
		if prevIsSpace {
			return "- "
		}
		if nextIsSpace {
			return " -"
		}
		return " - "
	}

	// Otherwise just use dash
	return "-"
}

// isIllegalChar checks if a character is illegal for filenames.
func isIllegalChar(r rune) bool {
	for _, illegal := range IllegalCharacters {
		if r == illegal {
			return true
		}
	}
	return false
}

// getReplacement returns a safe replacement for an illegal character.
func getReplacement(r rune) rune {
	switch r {
	case '\\':
		return '-'
	case '/':
		return '-'
	case '*':
		return '-'
	case '?':
		return ' '
	case '"':
		return '\''
	case '<':
		return '('
	case '>':
		return ')'
	case '|':
		return '-'
	default:
		return ' '
	}
}

// cleanupSpaces removes double spaces and trims the string.
func cleanupSpaces(s string) string {
	// Replace multiple spaces with single space
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// CleanTitle removes only filesystem-illegal characters, preserving all other characters.
// This is less aggressive than full sanitization - keeps Unicode, accents, etc.
func CleanTitle(s string) string {
	if s == "" {
		return s
	}

	var result strings.Builder
	result.Grow(len(s))

	for _, r := range s {
		if isIllegalChar(r) {
			continue
		}
		result.WriteRune(r)
	}

	return cleanupSpaces(result.String())
}

// SanitizeFilename applies all filename safety rules.
func SanitizeFilename(s string, replaceIllegal bool, colonMode ColonReplacement, customColon string) string {
	if s == "" {
		return s
	}

	// First replace illegal characters
	s = ReplaceIllegalCharacters(s, replaceIllegal, colonMode, customColon)

	// Trim leading/trailing spaces and dots (problematic on Windows)
	s = strings.Trim(s, " .")

	// Prevent reserved Windows filenames
	s = avoidReservedNames(s)

	return s
}

// avoidReservedNames handles Windows reserved device names.
func avoidReservedNames(s string) string {
	reserved := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	upper := strings.ToUpper(s)
	for _, r := range reserved {
		if upper == r {
			return s + "_"
		}
		// Also check with extension (e.g., "CON.txt")
		if strings.HasPrefix(upper, r+".") {
			return s[:len(r)] + "_" + s[len(r):]
		}
	}

	return s
}

// SanitizeFolderName applies folder name safety rules.
func SanitizeFolderName(s string, replaceIllegal bool, colonMode ColonReplacement, customColon string) string {
	// Same as filename but folder-specific rules could be added here
	return SanitizeFilename(s, replaceIllegal, colonMode, customColon)
}
