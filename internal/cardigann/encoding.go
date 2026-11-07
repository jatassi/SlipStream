package cardigann

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
)

// charset is a definition's text encoding. It encodes form values and
// query strings and decodes every response, whatever charset the response
// names.
type charset struct {
	enc encoding.Encoding
}

// lookupCharset resolves an encoding name as .NET's Encoding.GetEncoding
// does with the code page provider: by IANA name, where iso-8859-1 is
// Latin-1 itself, then by the WHATWG labels, which add aliases such as
// tis-620.
func lookupCharset(name string) (charset, error) {
	enc, err := ianaindex.IANA.Encoding(name)
	if err != nil || enc == nil {
		enc, err = htmlindex.Get(name)
	}
	if err != nil || enc == nil {
		return charset{}, fmt.Errorf("the encoding %q isn't known", name)
	}
	if enc == unicode.UTF8 {
		return charset{}, nil
	}
	return charset{enc: enc}, nil
}

// Encode is s in the charset. A character the charset lacks becomes '?'.
func (c charset) Encode(s string) []byte {
	if c.enc == nil {
		return []byte(s)
	}
	enc := c.enc.NewEncoder()
	out := make([]byte, 0, len(s))
	for _, r := range s {
		b, err := enc.Bytes([]byte(string(r)))
		if err != nil {
			out = append(out, '?')
			continue
		}
		out = append(out, b...)
	}
	return out
}

// Decode is b read in the charset. Bytes it can't read become U+FFFD.
func (c charset) Decode(b []byte) string {
	if c.enc == nil {
		return strings.ToValidUTF8(string(b), string(utf8.RuneError))
	}
	out, err := c.enc.NewDecoder().Bytes(b)
	if err != nil {
		return strings.ToValidUTF8(string(b), string(utf8.RuneError))
	}
	return string(out)
}

// URLEncode is .NET's WebUtility.UrlEncode over s in the charset: a space
// becomes '+', letters, digits and -_.!*() stay, and every other byte
// becomes %XX in uppercase hex.
func (c charset) URLEncode(s string) string {
	return urlEncodeBytes(c.Encode(s))
}

// urlEncodeUTF8 is URLEncode in UTF-8, as WebUtility.UrlEncode is when no
// encoding is given.
func urlEncodeUTF8(s string) string { return urlEncodeBytes([]byte(s)) }

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

// URLDecode is .NET's WebUtility.UrlDecode in the charset: '+' is a space
// and %XX a byte, and a '%' without two hex digits stays as it is.
func (c charset) URLDecode(s string) string {
	in := c.Encode(s)
	out := make([]byte, 0, len(in))
	for i := 0; i < len(in); i++ {
		switch {
		case in[i] == '+':
			out = append(out, ' ')
		case in[i] == '%' && i+2 < len(in) && isHex(in[i+1]) && isHex(in[i+2]):
			out = append(out, unhex(in[i+1])<<4|unhex(in[i+2]))
			i += 2
		default:
			out = append(out, in[i])
		}
	}
	return c.Decode(out)
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

// pair is one key and value of a form or query string, kept in order.
type pair struct {
	Key, Value string
}

// QueryString is Prowlarr's GetQueryString: key=value pairs in order joined
// by sep, each value URL-encoded in the charset and each key as written.
func (c charset) QueryString(pairs []pair, sep string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = p.Key + "=" + c.URLEncode(p.Value)
	}
	return strings.Join(parts, sep)
}
