// Ported from Prowlarr's src/NzbDrone.Common/Extensions/StringExtensions.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package encoding

import (
	"html"
	"strconv"
	"strings"
	"unicode"
)

// URLEncode is WebUtility.UrlEncode: s in UTF-8 with a space as '+',
// letters, digits and -_.!*() as they are, and every other byte as %XX in
// uppercase hex.
func URLEncode(s string) string { return urlEncodeBytes([]byte(s)) }

func urlEncodeBytes(b []byte) string {
	const hex = "0123456789ABCDEF"
	var out strings.Builder
	for _, c := range b {
		switch {
		case urlSafe(c):
			out.WriteByte(c)
		case c == ' ':
			out.WriteByte('+')
		default:
			out.WriteByte('%')
			out.WriteByte(hex[c>>4])
			out.WriteByte(hex[c&15])
		}
	}
	return out.String()
}

func urlSafe(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || strings.IndexByte("-_.!*()", c) >= 0
}

// HTMLDecode is WebUtility.HtmlDecode: named entities and &#N; and &#xN;
// references that end in ';'. Anything else stays as written.
func HTMLDecode(s string) string {
	if !strings.Contains(s, "&") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '&' {
			b.WriteByte(s[i])
			continue
		}
		end := strings.IndexAny(s[i+1:], ";&")
		if end <= 0 || s[i+1+end] != ';' {
			b.WriteByte('&')
			continue
		}
		entity := s[i+1 : i+1+end]
		if r, ok := decodeEntity(entity); ok {
			b.WriteString(r)
			i += end + 1
			continue
		}
		if len(entity) > 1 && entity[0] == '#' {
			b.WriteByte('&')
			continue
		}
		b.WriteString("&" + entity + ";")
		i += end + 1
	}
	return b.String()
}

func decodeEntity(entity string) (string, bool) {
	if len(entity) > 1 && entity[0] == '#' {
		var n uint64
		var err error
		if entity[1] == 'x' || entity[1] == 'X' {
			n, err = strconv.ParseUint(entity[2:], 16, 32)
		} else {
			n, err = strconv.ParseUint(strings.TrimSpace(entity[1:]), 10, 32)
		}
		if err != nil || (n >= 0xD800 && n <= 0xDFFF) || n > unicode.MaxRune {
			return "", false
		}
		return string(rune(n)), true
	}
	out := html.UnescapeString("&" + entity + ";")
	if strings.Contains(out, ";") {
		return "", false
	}
	return out, true
}

// HTMLEncode is WebUtility.HtmlEncode.
func HTMLEncode(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\'':
			b.WriteString("&#39;")
		case r == '&':
			b.WriteString("&amp;")
		case (r >= 160 && r < 256) || r > 0xFFFF:
			b.WriteString("&#" + strconv.Itoa(int(r)) + ";")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
