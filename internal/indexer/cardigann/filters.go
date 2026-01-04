package cardigann

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// FilterFunc is a function that transforms a string value.
type FilterFunc func(value string, args []string) (string, error)

// filters is the registry of all available filter functions.
var filters = map[string]FilterFunc{
	// String manipulation
	"replace":    filterReplace,
	"re_replace": filterReReplace,
	"split":      filterSplit,
	"trim":       filterTrim,
	"trimleft":   filterTrimLeft,
	"trimright":  filterTrimRight,
	"prepend":    filterPrepend,
	"append":     filterAppend,
	"tolower":    filterToLower,
	"toupper":    filterToUpper,

	// Date parsing
	"dateparse": filterDateParse,
	"timeago":   filterTimeAgo,
	"fuzzytime": filterFuzzyTime,

	// URL processing
	"urldecode":   filterURLDecode,
	"urlencode":   filterURLEncode,
	"querystring": filterQueryString,

	// HTML processing
	"htmldecode": filterHTMLDecode,
	"htmlencode": filterHTMLEncode,
	"striptags":  filterStripTags,

	// Regex extraction
	"regexp": filterRegexp,

	// Validation
	"validate": filterValidate,

	// Size parsing
	"size": filterSize,

	// Numeric
	"multiply": filterMultiply,
	"divide":   filterDivide,

	// Debug
	"strdump": filterStrDump,

	// Text extraction
	"diacritics": filterDiacritics,
	"normalize":  filterNormalize,
}

// ApplyFilters applies a sequence of filters to a value.
func ApplyFilters(value string, filterList []Filter) (string, error) {
	return ApplyFiltersWithContext(value, filterList, nil, nil)
}

// ApplyFiltersWithContext applies filters with template evaluation support.
func ApplyFiltersWithContext(value string, filterList []Filter, engine *TemplateEngine, ctx *TemplateContext) (string, error) {
	result := value
	for _, f := range filterList {
		args := normalizeFilterArgs(f.Args)

		// Evaluate template expressions in filter arguments
		if engine != nil && ctx != nil {
			for i, arg := range args {
				if strings.Contains(arg, "{{") {
					evaluated, err := engine.Evaluate(arg, ctx)
					if err == nil {
						args[i] = evaluated
					}
				}
			}
		}

		fn, ok := filters[f.Name]
		if !ok {
			// Unknown filter, skip with warning
			continue
		}
		var err error
		result, err = fn(result, args)
		if err != nil {
			return "", fmt.Errorf("filter %s failed: %w", f.Name, err)
		}
	}
	return result, nil
}

// normalizeFilterArgs converts filter args to []string.
func normalizeFilterArgs(args interface{}) []string {
	if args == nil {
		return nil
	}
	switch v := args.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	default:
		return []string{fmt.Sprintf("%v", v)}
	}
}

// String manipulation filters

func filterReplace(value string, args []string) (string, error) {
	if len(args) < 2 {
		return value, nil
	}
	return strings.ReplaceAll(value, args[0], args[1]), nil
}

func filterReReplace(value string, args []string) (string, error) {
	if len(args) < 2 {
		return value, nil
	}
	re, err := regexp.Compile(args[0])
	if err != nil {
		return value, nil // Skip invalid regex
	}
	return re.ReplaceAllString(value, args[1]), nil
}

func filterSplit(value string, args []string) (string, error) {
	if len(args) < 2 {
		return value, nil
	}
	sep := args[0]
	idx, err := strconv.Atoi(args[1])
	if err != nil {
		return value, nil
	}
	parts := strings.Split(value, sep)
	if idx < 0 {
		idx = len(parts) + idx
	}
	if idx >= 0 && idx < len(parts) {
		return parts[idx], nil
	}
	return "", nil
}

func filterTrim(value string, args []string) (string, error) {
	if len(args) > 0 {
		return strings.Trim(value, args[0]), nil
	}
	return strings.TrimSpace(value), nil
}

func filterTrimLeft(value string, args []string) (string, error) {
	if len(args) > 0 {
		return strings.TrimLeft(value, args[0]), nil
	}
	return strings.TrimLeft(value, " \t\n\r"), nil
}

func filterTrimRight(value string, args []string) (string, error) {
	if len(args) > 0 {
		return strings.TrimRight(value, args[0]), nil
	}
	return strings.TrimRight(value, " \t\n\r"), nil
}

func filterPrepend(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}
	return args[0] + value, nil
}

func filterAppend(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}
	return value + args[0], nil
}

func filterToLower(value string, args []string) (string, error) {
	return strings.ToLower(value), nil
}

func filterToUpper(value string, args []string) (string, error) {
	return strings.ToUpper(value), nil
}

// Date parsing filters

func filterDateParse(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}

	layout := convertGoDateFormat(args[0])
	t, err := time.Parse(layout, value)
	if err != nil {
		// Try common formats
		formats := []string{
			time.RFC3339,
			time.RFC1123,
			time.RFC1123Z,
			"2006-01-02 15:04:05",
			"2006-01-02",
			"Jan 02 2006",
			"Jan 2 2006",
			"02 Jan 2006",
			"2 Jan 2006",
			"January 2, 2006",
		}
		for _, f := range formats {
			if t, err = time.Parse(f, value); err == nil {
				break
			}
		}
		if err != nil {
			return value, nil
		}
	}
	return t.Format(time.RFC3339), nil
}

// convertGoDateFormat converts common date format strings to Go layout.
func convertGoDateFormat(format string) string {
	// Go uses reference time: Mon Jan 2 15:04:05 MST 2006
	replacements := map[string]string{
		"yyyy": "2006",
		"YYYY": "2006",
		"yy":   "06",
		"YY":   "06",
		"MM":   "01",
		"M":    "1",
		"dd":   "02",
		"DD":   "02",
		"d":    "2",
		"D":    "2",
		"HH":   "15",
		"hh":   "03",
		"mm":   "04",
		"ss":   "05",
		"SS":   "05",
	}
	result := format
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}

func filterTimeAgo(value string, args []string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	now := time.Now()

	// Handle "today" and "yesterday"
	if value == "today" {
		return now.Format(time.RFC3339), nil
	}
	if value == "yesterday" {
		return now.AddDate(0, 0, -1).Format(time.RFC3339), nil
	}

	// Parse relative time like "2 hours ago", "3 days ago"
	re := regexp.MustCompile(`(\d+)\s*(second|minute|hour|day|week|month|year)s?\s*ago`)
	matches := re.FindStringSubmatch(value)
	if len(matches) < 3 {
		return value, nil
	}

	num, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	var d time.Duration
	switch unit {
	case "second":
		d = time.Duration(num) * time.Second
	case "minute":
		d = time.Duration(num) * time.Minute
	case "hour":
		d = time.Duration(num) * time.Hour
	case "day":
		d = time.Duration(num) * 24 * time.Hour
	case "week":
		d = time.Duration(num) * 7 * 24 * time.Hour
	case "month":
		return now.AddDate(0, -num, 0).Format(time.RFC3339), nil
	case "year":
		return now.AddDate(-num, 0, 0).Format(time.RFC3339), nil
	}

	return now.Add(-d).Format(time.RFC3339), nil
}

func filterFuzzyTime(value string, args []string) (string, error) {
	// First try timeago
	result, _ := filterTimeAgo(value, args)
	if result != value {
		return result, nil
	}
	// Then try dateparse with common formats
	return filterDateParse(value, []string{"2006-01-02 15:04:05"})
}

// URL processing filters

func filterURLDecode(value string, args []string) (string, error) {
	decoded, err := url.QueryUnescape(value)
	if err != nil {
		return value, nil
	}
	return decoded, nil
}

func filterURLEncode(value string, args []string) (string, error) {
	return url.QueryEscape(value), nil
}

func filterQueryString(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}
	paramName := args[0]

	// Try parsing as full URL first
	u, err := url.Parse(value)
	if err == nil && u.RawQuery != "" {
		return u.Query().Get(paramName), nil
	}

	// Try parsing as query string
	values, err := url.ParseQuery(value)
	if err == nil {
		return values.Get(paramName), nil
	}

	return "", nil
}

// HTML processing filters

func filterHTMLDecode(value string, args []string) (string, error) {
	return html.UnescapeString(value), nil
}

func filterHTMLEncode(value string, args []string) (string, error) {
	return html.EscapeString(value), nil
}

func filterStripTags(value string, args []string) (string, error) {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(value, ""), nil
}

// Regex extraction filter

func filterRegexp(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}
	re, err := regexp.Compile(args[0])
	if err != nil {
		return "", nil
	}
	matches := re.FindStringSubmatch(value)
	if len(matches) < 2 {
		return "", nil
	}
	return matches[1], nil // Return first capture group
}

// Validation filter

func filterValidate(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}
	allowedValues := strings.Split(args[0], "|")
	for _, allowed := range allowedValues {
		if value == allowed {
			return value, nil
		}
	}
	return "", nil // Value not in allowed list
}

// Size parsing filter - converts human-readable sizes to bytes

func filterSize(value string, args []string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ",", "")

	// Parse number and unit
	re := regexp.MustCompile(`([\d.]+)\s*([KMGTPE]?i?B?)`)
	matches := re.FindStringSubmatch(strings.ToUpper(value))
	if len(matches) < 2 {
		return "0", nil
	}

	num, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return "0", nil
	}

	var multiplier float64 = 1
	if len(matches) >= 3 {
		unit := strings.ToUpper(matches[2])
		switch {
		case strings.HasPrefix(unit, "K"):
			multiplier = 1024
		case strings.HasPrefix(unit, "M"):
			multiplier = 1024 * 1024
		case strings.HasPrefix(unit, "G"):
			multiplier = 1024 * 1024 * 1024
		case strings.HasPrefix(unit, "T"):
			multiplier = 1024 * 1024 * 1024 * 1024
		case strings.HasPrefix(unit, "P"):
			multiplier = 1024 * 1024 * 1024 * 1024 * 1024
		}
	}

	bytes := int64(num * multiplier)
	return strconv.FormatInt(bytes, 10), nil
}

// Numeric filters

func filterMultiply(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value, nil
	}
	factor, err := strconv.ParseFloat(args[0], 64)
	if err != nil {
		return value, nil
	}
	return fmt.Sprintf("%f", num*factor), nil
}

func filterDivide(value string, args []string) (string, error) {
	if len(args) < 1 {
		return value, nil
	}
	num, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value, nil
	}
	divisor, err := strconv.ParseFloat(args[0], 64)
	if err != nil || divisor == 0 {
		return value, nil
	}
	return fmt.Sprintf("%f", num/divisor), nil
}

// Debug filter

func filterStrDump(value string, args []string) (string, error) {
	// In production, just return the value
	// This is used for debugging definitions
	return value, nil
}

// Text processing filters

func filterDiacritics(value string, args []string) (string, error) {
	// Remove diacritics/accents from characters
	// This is a simplified version
	var result strings.Builder
	for _, r := range value {
		if unicode.Is(unicode.Mn, r) {
			continue // Skip combining marks
		}
		result.WriteRune(r)
	}
	return result.String(), nil
}

func filterNormalize(value string, args []string) (string, error) {
	// Normalize whitespace and trim
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(value, " ")), nil
}
