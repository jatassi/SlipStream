package port

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
)

// charsets are the encodings an Entry may name, "" being UTF-8.
var charsets = map[string]encoding.Encoding{"iso-8859-1": charmap.ISO8859_1, "windows-1251": charmap.Windows1251}

// decodeText is b read in the named encoding. Bytes it can't read become
// U+FFFD.
func decodeText(name string, b []byte) string {
	if enc, ok := charsets[name]; ok {
		out, _ := enc.NewDecoder().Bytes(b)
		return string(out)
	}
	return strings.ToValidUTF8(string(b), string(utf8.RuneError))
}

// encodeText is s in the named encoding. A character the encoding lacks
// becomes '?', as .NET's Encoding.GetBytes makes it.
func encodeText(name, s string) []byte {
	enc, ok := charsets[name]
	if !ok {
		return []byte(s)
	}
	e := enc.NewEncoder()
	out := make([]byte, 0, len(s))
	for _, r := range s {
		b, err := e.Bytes([]byte(string(r)))
		if err != nil {
			b = []byte{'?'}
		}
		out = append(out, b...)
	}
	return out
}
