// Package bencode decodes and encodes BitTorrent's bencoding, and reads a
// torrent's metainfo with the exact bytes of its info dictionary, from
// which the v1 info hash comes.
package bencode

import (
	"bytes"
	"crypto/sha1" //nolint:gosec // the v1 info hash is SHA-1 by the protocol
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ErrV2Only refuses a torrent that has only the v2 form, so no v1 info hash.
var ErrV2Only = errors.New("bencode: the torrent is v2-only")

// Decode decodes one value that fills data: an int64, a string, a []any
// or a map[string]any.
func Decode(data []byte) (any, error) {
	d := decoder{data: data}
	v, err := d.value()
	if err != nil {
		return nil, err
	}
	if d.pos != len(data) {
		return nil, d.errorf("data after the value")
	}
	return v, nil
}

type decoder struct {
	data []byte
	pos  int
}

func (d *decoder) errorf(format string, args ...any) error {
	return fmt.Errorf("bencode: at byte %d: %s", d.pos, fmt.Sprintf(format, args...))
}

func (d *decoder) value() (any, error) {
	if d.pos >= len(d.data) {
		return nil, d.errorf("unexpected end")
	}
	switch c := d.data[d.pos]; {
	case c == 'i':
		d.pos++
		return d.integer('e')
	case c == 'l':
		d.pos++
		var list []any
		for !d.end() {
			v, err := d.value()
			if err != nil {
				return nil, err
			}
			list = append(list, v)
		}
		return list, nil
	case c == 'd':
		d.pos++
		dict := map[string]any{}
		for !d.end() {
			k, v, err := d.entry()
			if err != nil {
				return nil, err
			}
			dict[k] = v
		}
		return dict, nil
	case c >= '0' && c <= '9':
		return d.str()
	default:
		return nil, d.errorf("unexpected %q", c)
	}
}

// end consumes the e that closes a list or dictionary.
func (d *decoder) end() bool {
	if d.pos < len(d.data) && d.data[d.pos] == 'e' {
		d.pos++
		return true
	}
	return false
}

func (d *decoder) entry() (string, any, error) {
	if d.pos >= len(d.data) || d.data[d.pos] < '0' || d.data[d.pos] > '9' {
		return "", nil, d.errorf("a dictionary key is not a string")
	}
	k, err := d.str()
	if err != nil {
		return "", nil, err
	}
	v, err := d.value()
	return k, v, err
}

func (d *decoder) integer(stop byte) (int64, error) {
	end := bytes.IndexByte(d.data[d.pos:], stop)
	if end < 0 {
		return 0, d.errorf("unterminated integer")
	}
	digits := string(d.data[d.pos : d.pos+end])
	n, err := strconv.ParseInt(digits, 10, 64)
	if abs := strings.TrimPrefix(digits, "-"); err != nil || abs[0] == '+' || (abs[0] == '0' && digits != "0") {
		return 0, d.errorf("invalid integer %q", digits)
	}
	d.pos += end + 1
	return n, nil
}

func (d *decoder) str() (string, error) {
	n, err := d.integer(':')
	if err != nil {
		return "", err
	}
	if n < 0 || n > int64(len(d.data)-d.pos) {
		return "", d.errorf("string length %d runs past the end", n)
	}
	s := string(d.data[d.pos : d.pos+int(n)])
	d.pos += int(n)
	return s, nil
}

type InfoHash [20]byte

func (h InfoHash) String() string { return hex.EncodeToString(h[:]) }

// ParseInfoHash reads a hex info hash.
func ParseInfoHash(s string) (InfoHash, error) {
	var h InfoHash
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != len(h) {
		return InfoHash{}, fmt.Errorf("bencode: info hash %q is not 40 hex digits", s)
	}
	copy(h[:], b)
	return h, nil
}

// Metainfo is a v1 or hybrid torrent's metainfo. RawInfo holds the info
// dictionary exactly as data held it, and InfoHash is its SHA-1.
type Metainfo struct {
	Announce string
	Info     Info
	RawInfo  []byte
	InfoHash InfoHash
}

type Info struct {
	Name        string
	PieceLength int64
	Pieces      []byte
	Private     bool
	// Files lists a multi-file torrent's files in metainfo order, or the
	// single file of one, whose Path is then nil.
	Files []File
}

type File struct {
	Path   []string
	Length int64
}

// DecodeMetainfo decodes a .torrent file.
func DecodeMetainfo(data []byte) (Metainfo, error) {
	d := decoder{data: data}
	if d.pos >= len(data) || data[0] != 'd' {
		return Metainfo{}, errors.New("bencode: the metainfo is not a dictionary")
	}
	d.pos++
	top := map[string]any{}
	var raw []byte
	for !d.end() {
		start := d.pos
		k, v, err := d.entry()
		if err != nil {
			return Metainfo{}, err
		}
		if k == "info" {
			raw = data[start+len("4:info") : d.pos]
		}
		top[k] = v
	}
	if d.pos != len(data) {
		return Metainfo{}, d.errorf("data after the metainfo")
	}
	if raw == nil {
		return Metainfo{}, errors.New("bencode: the metainfo has no info dictionary")
	}
	info, err := decodeInfo(top["info"])
	if err != nil {
		return Metainfo{}, err
	}
	announce, _ := top["announce"].(string)
	return Metainfo{Announce: announce, Info: info, RawInfo: raw, InfoHash: sha1.Sum(raw)}, nil //nolint:gosec // the v1 info hash
}

func decodeInfo(v any) (Info, error) {
	dict, ok := v.(map[string]any)
	if !ok {
		return Info{}, errors.New("bencode: info is not a dictionary")
	}
	pieces, ok := dict["pieces"].(string)
	if !ok {
		if _, tree := dict["file tree"]; tree || dict["meta version"] == int64(2) {
			return Info{}, ErrV2Only
		}
		return Info{}, errors.New("bencode: info has no pieces")
	}
	if len(pieces)%sha1.Size != 0 {
		return Info{}, fmt.Errorf("bencode: info pieces are %d bytes, not a multiple of %d", len(pieces), sha1.Size)
	}
	name, ok := dict["name"].(string)
	if !ok {
		return Info{}, errors.New("bencode: info has no name")
	}
	pieceLength, ok := dict["piece length"].(int64)
	if !ok || pieceLength <= 0 {
		return Info{}, errors.New("bencode: info has no positive piece length")
	}
	info := Info{Name: name, PieceLength: pieceLength, Pieces: []byte(pieces), Private: dict["private"] == int64(1)}
	if length, ok := dict["length"].(int64); ok {
		info.Files = []File{{Length: length}}
		return info, nil
	}
	files, ok := dict["files"].([]any)
	if !ok {
		return Info{}, errors.New("bencode: info has neither a length nor files")
	}
	for i, f := range files {
		file, err := decodeFile(f)
		if err != nil {
			return Info{}, fmt.Errorf("bencode: info file %d: %w", i, err)
		}
		info.Files = append(info.Files, file)
	}
	return info, nil
}

func decodeFile(v any) (File, error) {
	dict, ok := v.(map[string]any)
	if !ok {
		return File{}, errors.New("not a dictionary")
	}
	length, ok := dict["length"].(int64)
	if !ok || length < 0 {
		return File{}, errors.New("no length")
	}
	parts, ok := dict["path"].([]any)
	if !ok || len(parts) == 0 {
		return File{}, errors.New("no path")
	}
	var path []string
	for _, p := range parts {
		s, ok := p.(string)
		if !ok {
			return File{}, errors.New("a path element is not a string")
		}
		path = append(path, s)
	}
	return File{Path: path, Length: length}, nil
}

// Encode encodes v, which holds ints, int64s, strings, []bytes, []anys and
// map[string]anys. Dictionary keys go out sorted, as bencoding requires.
func Encode(v any) ([]byte, error) {
	var b bytes.Buffer
	if err := encode(&b, v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func encode(b *bytes.Buffer, v any) error {
	switch v := v.(type) {
	case int:
		fmt.Fprintf(b, "i%de", v)
	case int64:
		fmt.Fprintf(b, "i%de", v)
	case string:
		fmt.Fprintf(b, "%d:%s", len(v), v)
	case []byte:
		fmt.Fprintf(b, "%d:%s", len(v), v)
	case []any:
		b.WriteByte('l')
		for _, e := range v {
			if err := encode(b, e); err != nil {
				return err
			}
		}
		b.WriteByte('e')
	case map[string]any:
		b.WriteByte('d')
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			fmt.Fprintf(b, "%d:%s", len(k), k)
			if err := encode(b, v[k]); err != nil {
				return err
			}
		}
		b.WriteByte('e')
	default:
		return fmt.Errorf("bencode: cannot encode %T", v)
	}
	return nil
}
