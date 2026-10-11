package libtorrent_test

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"sort"
)

// singleFileTorrent builds v1 metainfo for one file holding data.
func singleFileTorrent(name string, data []byte, pieceLen int) []byte {
	var pieces []byte
	for off := 0; off < len(data); off += pieceLen {
		sum := sha1.Sum(data[off:min(off+pieceLen, len(data))])
		pieces = append(pieces, sum[:]...)
	}
	var b bytes.Buffer
	bencode(&b, map[string]any{
		"info": map[string]any{
			"length":       len(data),
			"name":         name,
			"piece length": pieceLen,
			"pieces":       pieces,
		},
	})
	return b.Bytes()
}

func bencode(b *bytes.Buffer, v any) {
	switch v := v.(type) {
	case int:
		fmt.Fprintf(b, "i%de", v)
	case string:
		fmt.Fprintf(b, "%d:%s", len(v), v)
	case []byte:
		fmt.Fprintf(b, "%d:%s", len(v), v)
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteByte('d')
		for _, k := range keys {
			bencode(b, k)
			bencode(b, v[k])
		}
		b.WriteByte('e')
	default:
		panic(fmt.Sprintf("bencode: unsupported %T", v))
	}
}
