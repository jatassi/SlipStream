// Ported from Prowlarr's src/NzbDrone.Common/Extensions/StringExtensions.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

// Package encoding is .NET's Encoding as Prowlarr's sites use it, with
// Prowlarr's UrlEncode and UrlDecode string extensions over it, and
// .NET's WebUtility encoders. The zero Charset is UTF-8.
package encoding

import (
	"fmt"
	"strings"
	"unicode/utf8"

	xencoding "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/encoding/unicode"
)

// Charset is a text encoding such as Encoding.UTF8 or
// Encoding.GetEncoding("windows-1251").
type Charset struct {
	enc xencoding.Encoding
}

// UTF8 is Encoding.UTF8.
var UTF8 = Charset{}

// GetEncoding resolves an encoding name as .NET's Encoding.GetEncoding
// does with the code page provider: by IANA name, where iso-8859-1 is
// Latin-1 itself, then by the WHATWG labels, which add aliases such as
// tis-620.
func GetEncoding(name string) (Charset, error) {
	enc, err := ianaindex.IANA.Encoding(name)
	if err != nil || enc == nil {
		enc, err = htmlindex.Get(name)
	}
	if err != nil || enc == nil {
		return Charset{}, fmt.Errorf("the encoding %q isn't known", name)
	}
	if enc == unicode.UTF8 {
		return UTF8, nil
	}
	return Charset{enc: enc}, nil
}

// GetBytes is s in the charset. A character the charset lacks becomes '?'.
func (c Charset) GetBytes(s string) []byte {
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

// GetString is b read in the charset. Bytes it can't read become U+FFFD.
func (c Charset) GetString(b []byte) string {
	if c.enc == nil {
		return strings.ToValidUTF8(string(b), string(utf8.RuneError))
	}
	out, err := c.enc.NewDecoder().Bytes(b)
	if err != nil {
		return strings.ToValidUTF8(string(b), string(utf8.RuneError))
	}
	return string(out)
}

// URLEncode is the UrlEncode(Encoding) string extension:
// WebUtility.UrlEncode over s in the charset.
func (c Charset) URLEncode(s string) string {
	return urlEncodeBytes(c.GetBytes(s))
}

// URLDecode is the UrlDecode(Encoding) string extension: '+' is a space
// and %XX a byte, and a '%' without two hex digits stays as it is.
func (c Charset) URLDecode(s string) string {
	in := c.GetBytes(s)
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
	return c.GetString(out)
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
