// Package releases builds the release catalog the fakes share: for each
// Scenario release, a payload video the bundled ffmpeg makes and a private
// torrent of it that announces to fake-tracker. The world builds it once
// per launch, after the fakes' listeners are bound.
package releases

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// PieceLength is the piece length of every catalog torrent.
const PieceLength = 256 << 10

type Catalog struct {
	// Passkey is the one passkey fake-tracker accepts, derived from the
	// Scenario seed.
	Passkey  string
	Releases []Release
}

// Release is a Scenario release with its payload and metainfo.
type Release struct {
	scenario.Release
	// Dir holds the payload, File, as a seeder's save path holds it.
	Dir      string
	File     string
	Size     int64
	BLAKE3   [32]byte
	Metainfo []byte
	InfoHash bencode.InfoHash
}

func (c *Catalog) ByInfoHash(h bencode.InfoHash) (Release, bool) {
	for _, r := range c.Releases {
		if r.InfoHash == h {
			return r, true
		}
	}
	return Release{}, false
}

// Passkey is the tracker passkey of a world with Scenario s.
func Passkey(s scenario.Scenario) string {
	r := node.NewStreams(s.Seed + "/releases").Stream("passkey")
	return fmt.Sprintf("%016x%016x", r.Uint64(), r.Uint64())
}

type Config struct {
	// FFmpeg is the bundled ffmpeg that makes each payload.
	FFmpeg string
	// Cache holds each payload ever made on this machine, named by its
	// inputs, so a launch encodes only a release it has not seen.
	Cache string
	// Dir is the run's folder the payloads are copied into.
	Dir string
	// Tracker is where fake-tracker listens.
	Tracker netip.AddrPort
}

// CacheDir is the per-machine payload cache.
func CacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find the fixture cache: %w", err)
	}
	return filepath.Join(dir, "slipstream-fixtures"), nil
}

// Build makes the catalog of s's releases. A Scenario without releases
// gets a catalog of only its passkey, and runs no ffmpeg.
func Build(ctx context.Context, s scenario.Scenario, cfg Config) (*Catalog, error) {
	c := &Catalog{Passkey: Passkey(s)}
	if len(s.Releases) == 0 {
		return c, nil
	}
	version, err := exec.CommandContext(ctx, cfg.FFmpeg, "-version").Output()
	if err != nil {
		return nil, fmt.Errorf("%s -version: %w", cfg.FFmpeg, err)
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create the release folder: %w", err)
	}
	announce := "https://" + cfg.Tracker.String() + "/announce/" + c.Passkey
	for _, r := range s.Releases {
		cached, err := payload(ctx, cfg, version, r.Video)
		if err != nil {
			return nil, fmt.Errorf("release %s: %w", r.Name, err)
		}
		release, err := place(cached, cfg.Dir, r, announce)
		if err != nil {
			return nil, fmt.Errorf("release %s: %w", r.Name, err)
		}
		c.Releases = append(c.Releases, release)
	}
	return c, nil
}

// encodeArgs make a video whose bytes depend only on v and the ffmpeg
// build: one encoder thread, and no metadata or muxer field that varies.
func encodeArgs(v scenario.Video, out string) []string {
	size := strconv.Itoa(v.Width) + "x" + strconv.Itoa(v.Height)
	return []string{
		"-nostdin", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "testsrc2=size=" + size + ":rate=24:duration=" + strconv.Itoa(v.Seconds),
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "23", "-pix_fmt", "yuv420p", "-g", "48",
		"-threads", "1", "-x264-params", "threads=1:lookahead-threads=1",
		"-fflags", "+bitexact", "-flags:v", "+bitexact", "-map_metadata", "-1",
		"-f", "matroska", out,
	}
}

// payload returns the cached payload for v, making it first if the cache
// lacks it. The key hashes the ffmpeg build's version text with the
// arguments, so another build or another encode is another file.
func payload(ctx context.Context, cfg Config, version []byte, v scenario.Video) (string, error) {
	key, err := json.Marshal(struct {
		FFmpeg string   `json:"ffmpeg"`
		Args   []string `json:"args"`
	}{string(version), encodeArgs(v, "-")})
	if err != nil {
		return "", fmt.Errorf("payload key: %w", err)
	}
	sum := sha256.Sum256(key)
	path := filepath.Join(cfg.Cache, hex.EncodeToString(sum[:])+".mkv")
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.MkdirAll(cfg.Cache, 0o755); err != nil {
		return "", fmt.Errorf("create the fixture cache: %w", err)
	}
	tmp, err := os.CreateTemp(cfg.Cache, ".encoding-*.mkv")
	if err != nil {
		return "", fmt.Errorf("create the payload: %w", err)
	}
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmp.Name()) }()
	cmd := exec.CommandContext(ctx, cfg.FFmpeg, append(encodeArgs(v, tmp.Name()), "-y")...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	// A link never replaces a payload another world published, so no open
	// of the cached path races a rename onto it. That payload is this one.
	if err := os.Link(tmp.Name(), path); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", fmt.Errorf("cache the payload: %w", err)
	}
	return path, nil
}

// place copies the cached payload into dir as <release name>.mkv, hashing
// it as it goes, and builds its metainfo.
func place(cached, dir string, r scenario.Release, announce string) (Release, error) {
	name := r.Name + ".mkv"
	file := filepath.Join(dir, name)
	in, err := os.Open(cached)
	if err != nil {
		return Release{}, fmt.Errorf("open the payload: %w", err)
	}
	defer func() { _ = in.Close() }()
	out, err := os.Create(file)
	if err != nil {
		return Release{}, fmt.Errorf("copy the payload: %w", err)
	}
	b3 := blake3.New()
	pieces := &pieceHasher{}
	size, err := io.Copy(io.MultiWriter(out, b3, pieces), in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return Release{}, fmt.Errorf("copy the payload: %w", err)
	}
	meta, hash, err := metainfo(name, size, pieces.sum(), announce)
	if err != nil {
		return Release{}, err
	}
	release := Release{Release: r, Dir: dir, File: file, Size: size, Metainfo: meta, InfoHash: hash}
	copy(release.BLAKE3[:], b3.Sum(nil))
	return release, nil
}

// Metainfo is a catalog torrent of payload as the one file name, for a
// world that makes its payloads without ffmpeg.
func Metainfo(name string, payload []byte, announce string) ([]byte, bencode.InfoHash, error) {
	pieces := &pieceHasher{}
	_, _ = pieces.Write(payload)
	return metainfo(name, int64(len(payload)), pieces.sum(), announce)
}

func metainfo(name string, size int64, pieces []byte, announce string) ([]byte, bencode.InfoHash, error) {
	meta, err := bencode.Encode(map[string]any{
		"announce": announce,
		"info": map[string]any{
			"length":       size,
			"name":         name,
			"piece length": PieceLength,
			"pieces":       pieces,
			"private":      1,
		},
	})
	if err != nil {
		return nil, bencode.InfoHash{}, err
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		return nil, bencode.InfoHash{}, err
	}
	return meta, m.InfoHash, nil
}

// pieceHasher takes the SHA-1 of each PieceLength piece written to it.
type pieceHasher struct {
	pieces []byte
	piece  []byte
}

func (p *pieceHasher) Write(b []byte) (int, error) {
	n := len(b)
	for len(b) > 0 {
		take := min(PieceLength-len(p.piece), len(b))
		p.piece = append(p.piece, b[:take]...)
		b = b[take:]
		if len(p.piece) == PieceLength {
			p.flush()
		}
	}
	return n, nil
}

func (p *pieceHasher) flush() {
	sum := sha1.Sum(p.piece) //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	p.pieces = append(p.pieces, sum[:]...)
	p.piece = p.piece[:0]
}

func (p *pieceHasher) sum() []byte {
	if len(p.piece) > 0 {
		p.flush()
	}
	return p.pieces
}
