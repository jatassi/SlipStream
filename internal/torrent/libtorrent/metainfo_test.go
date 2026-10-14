package libtorrent_test

import (
	"crypto/sha1" //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
)

// singleFileTorrent builds v1 metainfo for one file holding data.
func singleFileTorrent(t *testing.T, name string, data []byte, pieceLen int) []byte {
	t.Helper()
	return torrent(t, map[string]any{"length": len(data), "name": name}, data, pieceLen)
}

// torrent builds v1 metainfo whose info dictionary holds layout and the
// piece hashes of data.
func torrent(t *testing.T, layout map[string]any, data []byte, pieceLen int) []byte {
	t.Helper()
	var pieces []byte
	for off := 0; off < len(data); off += pieceLen {
		sum := sha1.Sum(data[off:min(off+pieceLen, len(data))]) //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
		pieces = append(pieces, sum[:]...)
	}
	info := map[string]any{"piece length": pieceLen, "pieces": pieces}
	for k, v := range layout {
		info[k] = v
	}
	meta, err := bencode.Encode(map[string]any{"info": info})
	if err != nil {
		t.Fatal(err)
	}
	return meta
}
