// Ported from Prowlarr's src/NzbDrone.Core/Parser/ParseUtil.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package parseutil is Prowlarr's ParseUtil and CookieUtil: the number,
// size, IMDb id, query string and cookie readers that site code calls.
// A member that throws in C# returns an error; a Try member reports
// false. A C# null string is the empty string here.
package parseutil

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// normalizeNumber is ParseUtil.NormalizeNumber: keep digits, '.' and ','.
// An integer with both separators is read as written, else '.' becomes a
// group separator. A decimal reads ',' as '.' and keeps only the last '.'.
func normalizeNumber(s string, isInt bool) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) || r == '.' || r == ',' {
			b.WriteRune(r)
		}
	}
	v := b.String()
	if v == "" {
		return "0"
	}
	if isInt {
		if strings.Contains(v, ",") && strings.Contains(v, ".") {
			return v
		}
		return strings.ReplaceAll(v, ".", ",")
	}
	v = strings.ReplaceAll(v, ",", ".")
	if last := strings.LastIndexByte(v, '.'); strings.Count(v, ".") > 1 {
		v = strings.ReplaceAll(v[:last], ".", "") + v[last:]
	}
	return v
}

// parseDotNetInteger reads s as .NET's long.Parse with NumberStyles.Any and
// the invariant culture: ',' groups digits anywhere after the first digit,
// and a '.' may follow if every digit after it is zero.
func parseDotNetInteger(s string) (int64, error) {
	var digits strings.Builder
	seenDigit, seenPoint := false, false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9' && !seenPoint:
			digits.WriteRune(r)
			seenDigit = true
		case r >= '0' && r <= '9':
			if r != '0' {
				return 0, fmt.Errorf("%q isn't a whole number", s)
			}
		case r == ',' && seenDigit && !seenPoint:
		case r == '.' && !seenPoint:
			seenPoint = true
		default:
			return 0, fmt.Errorf("%q isn't a number", s)
		}
	}
	if digits.Len() == 0 {
		if seenPoint && strings.ContainsAny(s, "0123456789") {
			return 0, nil
		}
		return 0, fmt.Errorf("%q isn't a number", s)
	}
	n, err := strconv.ParseInt(digits.String(), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%q is out of range", s)
	}
	return n, nil
}

func CoerceDouble(s string) (float64, error) {
	f, err := strconv.ParseFloat(normalizeNumber(s, false), 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, fmt.Errorf("%q isn't a number", s)
	}
	return f, nil
}

func CoerceFloat(s string) (float32, error) {
	f, err := CoerceDouble(s)
	return float32(f), err
}

func CoerceInt(s string) (int, error) {
	n, err := CoerceLong(s)
	if err != nil {
		return 0, err
	}
	if n > math.MaxInt32 {
		return 0, fmt.Errorf("%q is out of range", s)
	}
	return int(n), nil
}

func CoerceLong(s string) (int64, error) {
	return parseDotNetInteger(normalizeNumber(s, true))
}

func TryCoerceDouble(s string) (float64, bool) {
	f, err := CoerceDouble(s)
	return f, err == nil
}

func TryCoerceFloat(s string) (float32, bool) {
	f, err := CoerceFloat(s)
	return f, err == nil
}

func TryCoerceInt(s string) (int, bool) {
	n, err := CoerceInt(s)
	return n, err == nil
}

func TryCoerceLong(s string) (int64, bool) {
	n, err := CoerceLong(s)
	return n, err == nil
}

// GetLongFromString is the first run of ASCII digits in s, read by
// CoerceLong. It reports false where C# returns null, when s is blank.
func GetLongFromString(s string) (n int64, ok bool, err error) {
	if strings.TrimSpace(s) == "" {
		return 0, false, nil
	}
	start := strings.IndexFunc(s, isASCIIDigit)
	run := ""
	if start >= 0 {
		end := strings.IndexFunc(s[start:], func(r rune) bool { return !isASCIIDigit(r) })
		if end < 0 {
			end = len(s) - start
		}
		run = s[start : start+end]
	}
	n, err = CoerceLong(run)
	return n, true, err
}

func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }

// \n? is .NET's $, which also matches before a final newline.
var imdbID = regexp.MustCompile(`^(?:tt)?(\d{1,8})\n?$`)

// GetImdbID is the number of an IMDb id written as tt0123456 or 123456.
// It reports false where C# returns null.
func GetImdbID(s string) (int, bool) {
	m := imdbID.FindStringSubmatch(s)
	if m == nil {
		return 0, false
	}
	n, err := strconv.Atoi(m[1])
	return n, err == nil
}

// GetArgumentFromQueryString is the first value of arg, matched ignoring
// case, in the query after url's first '?', read as
// QueryHelpers.ParseQuery reads it: a key without '=' has an empty value.
// A url without a query or without arg is an error, as both throw in C#.
func GetArgumentFromQueryString(url, arg string) (string, error) {
	_, qs, ok := strings.Cut(url, "?")
	if !ok {
		return "", fmt.Errorf("%q has no query string", url)
	}
	qs, _, _ = strings.Cut(qs, "#")
	for part := range strings.SplitSeq(qs, "&") {
		key, value, _ := strings.Cut(part, "=")
		if strings.EqualFold(unescapeQuery(strings.TrimLeftFunc(key, unicode.IsSpace)), arg) {
			return unescapeQuery(value), nil
		}
	}
	return "", fmt.Errorf("%q has no %s parameter", url, arg)
}

// unescapeQuery is Uri.UnescapeDataString after '+' becomes a space. A
// sequence that isn't UTF-8 stays escaped.
func unescapeQuery(s string) string {
	s = strings.ReplaceAll(s, "+", " ")
	if !strings.Contains(s, "%") {
		return s
	}
	var out []byte
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			out = append(out, unhex(s[i+1])<<4|unhex(s[i+2]))
			i += 2
			continue
		}
		out = append(out, s[i])
	}
	if !utf8.Valid(out) {
		return s
	}
	return string(out)
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func unhex(c byte) byte {
	switch {
	case c <= '9':
		return c - '0'
	case c <= 'F':
		return c - 'A' + 10
	}
	return c - 'a' + 10
}

// GetBytes is the number in s times the unit its letters name, read as
// kb, mb, gb or tb in base 1024 once every lowercase 'i' is dropped. No
// unit means bytes.
func GetBytes(s string) (int64, error) {
	var unit strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			unit.WriteRune(r)
		}
	}
	v, err := CoerceDouble(s)
	if err != nil {
		return 0, err
	}
	u := strings.ToLower(strings.ReplaceAll(unit.String(), "i", ""))
	for _, m := range sizeUnits {
		if strings.Contains(u, m.unit) {
			return int64(v * m.factor), nil
		}
	}
	return int64(v), nil
}

var sizeUnits = []struct {
	unit   string
	factor float64
}{
	{"kb", 1 << 10},
	{"mb", 1 << 20},
	{"gb", 1 << 30},
	{"tb", 1 << 40},
}
