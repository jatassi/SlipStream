// Package torrentfile is the torrent a Download Client conformance suite
// adds: one no target holds, so a run never touches an Admin's torrents.
package torrentfile

import (
	"crypto/rand"
	"testing"

	"github.com/jatassi/slipstream/internal/bencode"
)

// Size is the length of the torrent's one file.
const Size = 3 << 18

// New is a private torrent of one file no target holds, its info hash and
// its file's name. Its data never needs to arrive.
func New(t testing.TB) (metainfo []byte, hash bencode.InfoHash, name string) {
	t.Helper()
	pieceLength := int64(1 << 18)
	pieces := make([]byte, 20*((Size+pieceLength-1)/pieceLength))
	_, _ = rand.Read(pieces)
	name = "slipstream-conformance-" + rand.Text()[:8] + ".mkv"
	meta, err := bencode.Encode(map[string]any{
		"announce": "http://tracker.invalid/announce",
		"info":     map[string]any{"length": int64(Size), "name": name, "piece length": pieceLength, "pieces": pieces, "private": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	return meta, m.InfoHash, name
}
