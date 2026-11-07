package cardigann

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
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

// coerceLong is ParseUtil.CoerceLong.
func coerceLong(s string) (int64, error) {
	return parseDotNetInteger(normalizeNumber(s, true))
}

// coerceInt is ParseUtil.CoerceInt.
func coerceInt(s string) (int, error) {
	n, err := coerceLong(s)
	if err != nil {
		return 0, err
	}
	if n > math.MaxInt32 {
		return 0, fmt.Errorf("%q is out of range", s)
	}
	return int(n), nil
}

// coerceDouble is ParseUtil.CoerceDouble.
func coerceDouble(s string) (float64, error) {
	v := normalizeNumber(s, false)
	f, err := strconv.ParseFloat(v, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, fmt.Errorf("%q isn't a number", s)
	}
	return f, nil
}

func coerceFloat(s string) (float64, error) {
	f, err := coerceDouble(s)
	return float64(float32(f)), err
}

// parseSize is ParseUtil.GetBytes: the number in s times the unit its
// letters name, read as kb, mb, gb or tb in base 1024 once every lowercase
// 'i' is dropped. No unit means bytes.
func parseSize(s string) (int64, error) {
	var unit strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			unit.WriteRune(r)
		}
	}
	v, err := coerceDouble(s)
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

// longFromString is ParseUtil.GetLongFromString: the first run of ASCII
// digits in s. It reports false when s is blank.
func longFromString(s string) (n int64, ok bool, err error) {
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
	n, err = coerceLong(run)
	return n, true, err
}

func isASCIIDigit(r rune) bool { return r >= '0' && r <= '9' }
