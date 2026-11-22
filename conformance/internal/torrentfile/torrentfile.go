// Package torrentfile is the torrent a Download Client conformance suite
// adds: one no target holds, so a run never touches an Admin's torrents.
package torrentfile

import (
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
	"testing"
	"time"

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

// Seeded is a torrent of one file with its payload, whose pieces match
// it, so a client that checks the file finds every piece.
type Seeded struct {
	Metainfo []byte
	Hash     bencode.InfoHash
	// Name is the file's name, which is its path in the save folder.
	Name string
	Data []byte
}

// Seed is a private torrent of one file of Size random bytes announcing
// to announce.
func Seed(t testing.TB, announce string) Seeded {
	t.Helper()
	pieceLength := int64(1 << 18)
	data := make([]byte, Size)
	_, _ = rand.Read(data)
	var pieces []byte
	for at := int64(0); at < Size; at += pieceLength {
		sum := sha1.Sum(data[at:min(at+pieceLength, Size)]) //nolint:gosec // the v1 piece hash
		pieces = append(pieces, sum[:]...)
	}
	name := "slipstream-conformance-" + rand.Text()[:8] + ".mkv"
	meta, err := bencode.Encode(map[string]any{
		"announce": announce,
		"info":     map[string]any{"length": int64(Size), "name": name, "piece length": pieceLength, "pieces": pieces, "private": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	return Seeded{Metainfo: meta, Hash: m.InfoHash, Name: name, Data: data}
}

// Online is the seeded torrent an online run announces to the tracker at
// announce, which answers it every interval.
func Online(t testing.TB, announce, interval string) (Seeded, time.Duration) {
	t.Helper()
	every, err := time.ParseDuration(interval)
	if err != nil {
		t.Fatalf("the tracker's interval: %v", err)
	}
	return Seed(t, announce), every
}
