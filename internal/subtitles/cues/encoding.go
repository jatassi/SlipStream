package cues

import (
	"bytes"
	"errors"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode/utf32"

	xunicode "golang.org/x/text/encoding/unicode"

	"github.com/jatassi/slipstream/internal/language"
)

// codePage is one encoding a language's subtitles come in. chardet lists
// the charsets saintfish/chardet names for it. A page chardet has no
// recogniser for has none, and wins only by its place in the table.
type codePage struct {
	name    string
	enc     encoding.Encoding
	chardet []string
}

var (
	cp1250    = codePage{"windows-1250", charmap.Windows1250, []string{"windows-1250"}}
	cp1251    = codePage{"windows-1251", charmap.Windows1251, []string{"windows-1251"}}
	cp1252    = codePage{"windows-1252", charmap.Windows1252, []string{"windows-1252", "ISO-8859-1"}}
	cp1253    = codePage{"windows-1253", charmap.Windows1253, []string{"windows-1253"}}
	cp1254    = codePage{"windows-1254", charmap.Windows1254, []string{"windows-1254"}}
	cp1255    = codePage{"windows-1255", charmap.Windows1255, []string{"windows-1255"}}
	cp1256    = codePage{"windows-1256", charmap.Windows1256, []string{"windows-1256"}}
	cp1257    = codePage{"windows-1257", charmap.Windows1257, nil}
	cp1258    = codePage{"windows-1258", charmap.Windows1258, nil}
	cp874     = codePage{"windows-874", charmap.Windows874, nil}
	iso1      = codePage{"iso-8859-1", charmap.ISO8859_1, nil}
	iso2      = codePage{"iso-8859-2", charmap.ISO8859_2, []string{"ISO-8859-2"}}
	iso3      = codePage{"iso-8859-3", charmap.ISO8859_3, nil}
	iso4      = codePage{"iso-8859-4", charmap.ISO8859_4, nil}
	iso5      = codePage{"iso-8859-5", charmap.ISO8859_5, []string{"ISO-8859-5"}}
	iso6      = codePage{"iso-8859-6", charmap.ISO8859_6, []string{"ISO-8859-6"}}
	iso7      = codePage{"iso-8859-7", charmap.ISO8859_7, []string{"ISO-8859-7"}}
	iso8      = codePage{"iso-8859-8", charmap.ISO8859_8, []string{"ISO-8859-8", "ISO-8859-8-I"}}
	iso9      = codePage{"iso-8859-9", charmap.ISO8859_9, []string{"ISO-8859-9"}}
	iso13     = codePage{"iso-8859-13", charmap.ISO8859_13, nil}
	iso15     = codePage{"iso-8859-15", charmap.ISO8859_15, nil}
	koi8r     = codePage{"koi8-r", charmap.KOI8R, []string{"KOI8-R"}}
	koi8u     = codePage{"koi8-u", charmap.KOI8U, []string{"KOI8-R"}}
	gbk       = codePage{"gbk", simplifiedchinese.GBK, []string{"GB-18030"}}
	gb18030   = codePage{"gb18030", simplifiedchinese.GB18030, []string{"GB-18030"}}
	big5      = codePage{"big5", traditionalchinese.Big5, []string{"Big5"}}
	shiftJIS  = codePage{"shift-jis", japanese.ShiftJIS, []string{"Shift_JIS"}}
	eucJP     = codePage{"euc-jp", japanese.EUCJP, []string{"EUC-JP"}}
	iso2022JP = codePage{"iso-2022-jp", japanese.ISO2022JP, []string{"ISO-2022-JP"}}
	eucKR     = codePage{"euc-kr", korean.EUCKR, []string{"EUC-KR"}}
	utf16LE   = codePage{"utf-16le", xunicode.UTF16(xunicode.LittleEndian, xunicode.IgnoreBOM), nil}
)

// Bazarr's code pages per language (subliminal_patch/subtitle.py,
// guess_encoding), keyed by Code. Baltic, Thai, Vietnamese, Korean and
// KOI8 come from this table alone, because chardet can't tell them apart.
// Python's cp936 and gb2312 are gbk here, cp932 is shift-jis, cp950 and
// big5hkscs are big5, and tis-620 is windows-874. HZ, cp737, cp875,
// cp869 and mac_greek have no x/text decoder.
var codePages = func() map[string][]codePage {
	latin2 := []codePage{cp1250, iso2}
	cyrillic := []codePage{cp1251, iso5}
	m := map[string][]codePage{
		"zh": {gbk, big5, gb18030},
		"ja": {shiftJIS, eucJP, iso2022JP},
		"th": {cp874},
		"ar": {cp1256, utf16LE, iso6},
		"fa": {cp1256, utf16LE, iso6},
		"he": {cp1255, iso8},
		"tr": {cp1254, iso9, iso3},
		"el": {cp1253, iso7},
		"sl": {cp1250, iso2, iso4},
		"sq": {cp1250, iso2, cp1252, iso15, iso1, iso9},
		"ru": {cp1251, iso5, koi8r},
		"uk": {cp1251, koi8u, iso5},
		"sr": {cp1250, cp1251, iso2, iso5},
		"lv": {cp1257, iso13, iso4},
		"lt": {cp1257, iso13, iso4},
		"et": {cp1257, iso13, iso4},
		"vi": {cp1258},
		"ko": {eucKR},
	}
	for _, l := range []string{"pl", "cs", "sk", "hu", "bs", "hr", "ro"} {
		m[l] = latin2
	}
	for _, l := range []string{"bg", "mk"} {
		m[l] = cyrillic
	}
	return m
}()

var western = []codePage{cp1252, iso15, iso9, iso4, iso1}

var errUndecodable = errors.New("its text is in no encoding SlipStream reads for this language")

type bom struct {
	mark []byte
	name string
	enc  encoding.Encoding
}

var boms = []bom{
	{[]byte{0xEF, 0xBB, 0xBF}, "utf-8", xunicode.UTF8},
	{[]byte{0xFF, 0xFE, 0, 0}, "utf-32le", utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)},
	{[]byte{0, 0, 0xFE, 0xFF}, "utf-32be", utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)},
	{[]byte{0xFF, 0xFE}, "utf-16le", xunicode.UTF16(xunicode.LittleEndian, xunicode.IgnoreBOM)},
	{[]byte{0xFE, 0xFF}, "utf-16be", xunicode.UTF16(xunicode.BigEndian, xunicode.IgnoreBOM)},
}

// Decode answers b as UTF-8 without a BOM and the encoding it was in: a
// BOM's, else UTF-16 by its zero bytes, else UTF-8 when valid, else the
// language's code pages that decode b cleanly, scored by chardet.
func Decode(b []byte, lang language.Code) (text []byte, page string, err error) {
	for _, m := range boms {
		if bytes.HasPrefix(b, m.mark) {
			return decodeAs(b[len(m.mark):], m.name, m.enc)
		}
	}
	if name, enc, ok := utf16Unmarked(b); ok {
		return decodeAs(b, name, enc)
	}
	if utf8.Valid(b) && bytes.IndexByte(b, 0x1B) < 0 {
		return bytes.Clone(b), "utf-8", nil
	}
	pages, ok := codePages[string(lang.Base())]
	if lang.IsZero() || !ok {
		pages = western
	}
	return guess(b, pages)
}

func decodeAs(b []byte, name string, enc encoding.Encoding) (text []byte, page string, err error) {
	out, err := enc.NewDecoder().Bytes(b)
	if err != nil || !clean(out) {
		return nil, "", errUndecodable
	}
	return out, name, nil
}

// utf16Unmarked reads UTF-16 without a BOM from its zero bytes: the high
// byte of every ASCII character, timestamps and line breaks included.
func utf16Unmarked(b []byte) (name string, enc encoding.Encoding, ok bool) {
	n := min(len(b), 4096) &^ 1
	if n < 8 {
		return "", nil, false
	}
	var even, odd int
	for i := 0; i < n; i += 2 {
		if b[i] == 0 {
			even++
		}
		if b[i+1] == 0 {
			odd++
		}
	}
	pairs := n / 2
	switch {
	case odd*10 >= pairs*4 && even*10 < pairs:
		return "utf-16le", xunicode.UTF16(xunicode.LittleEndian, xunicode.IgnoreBOM), true
	case even*10 >= pairs*4 && odd*10 < pairs:
		return "utf-16be", xunicode.UTF16(xunicode.BigEndian, xunicode.IgnoreBOM), true
	}
	return "", nil, false
}

// clean reports whether decoded text has no replacement character, no
// escape and no C1 control, which a wrong code page leaves behind.
func clean(text []byte) bool {
	return !bytes.ContainsFunc(text, func(r rune) bool {
		return r == utf8.RuneError || r == 0x1B || r >= 0x80 && r <= 0x9F
	})
}
