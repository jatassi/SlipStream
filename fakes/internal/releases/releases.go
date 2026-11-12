// Package releases builds the release catalog the fakes share: for each
// Scenario release, a payload video the bundled ffmpeg makes and a torrent
// of it that announces to fake-tracker, private unless the release is
// public. The world builds it once per launch, after the fakes' listeners
// are bound.
package releases

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec // v1 piece hashes are SHA-1 by the protocol
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/node"
)

// PieceLength is the piece length of every catalog torrent.
const PieceLength = 256 << 10

type Catalog struct {
	Releases []Release
}

// Release is a Scenario release with its payload and metainfo.
type Release struct {
	scenario.Release
	// Dir holds the payload's Files, as a seeder's save path holds them.
	Dir string
	// Files are the payload's files in torrent order: <name>.mkv for a
	// release of one video, else each of its files under a <name> folder.
	Files []File
	// Size and BLAKE3 are the payload's, its files back to back.
	Size     int64
	BLAKE3   [32]byte
	Metainfo []byte
	InfoHash bencode.InfoHash
}

// File is one file of a release's payload.
type File struct {
	// Path is the file's in the torrent and under Dir, '/' separated.
	Path   string
	Size   int64
	BLAKE3 [32]byte
}

// Disk is where r's payload holds f.
func (r Release) Disk(f File) string { return filepath.Join(r.Dir, filepath.FromSlash(f.Path)) }

func (c *Catalog) ByInfoHash(h bencode.InfoHash) (Release, bool) {
	for _, r := range c.Releases {
		if r.InfoHash == h {
			return r, true
		}
	}
	return Release{}, false
}

// Seeders is the account fixture-seeders announce as. Every other account
// is a fixture-site site's, named by its label.
const Seeders = "seeders"

// Passkey is account's tracker passkey in a world with Scenario s.
func Passkey(s scenario.Scenario, account string) string {
	r := node.NewStreams(s.Seed + "/releases").Stream("passkey/" + account)
	return fmt.Sprintf("%016x%016x", r.Uint64(), r.Uint64())
}

type Config struct {
	// Tools are the bundled tools that make each payload's videos.
	Tools mediafix.Tools
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

// Build makes the catalog of s's releases, whose metainfo announces as
// Seeders. A Scenario without releases gets an empty catalog, and runs no
// ffmpeg.
func Build(ctx context.Context, s scenario.Scenario, cfg Config) (*Catalog, error) {
	c := &Catalog{}
	if len(s.Releases) == 0 {
		return c, nil
	}
	version, err := exec.CommandContext(ctx, cfg.Tools.FFmpeg, "-version").Output()
	if err != nil {
		return nil, fmt.Errorf("%s -version: %w", cfg.Tools.FFmpeg, err)
	}
	media, err := mediafix.NewBuilder(ctx, cfg.Tools, cfg.Cache)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("create the release folder: %w", err)
	}
	cached, err := payloads(ctx, cfg, version, s.Releases)
	if err != nil {
		return nil, err
	}
	announce := "https://" + cfg.Tracker.String() + "/announce/" + Passkey(s, Seeders)
	for _, r := range s.Releases {
		release, err := place(ctx, cfg.Dir, media, cached, r, announce)
		if err != nil {
			return nil, fmt.Errorf("release %s: %w", r.Name, err)
		}
		c.Releases = append(c.Releases, release)
	}
	return c, nil
}

// payloads returns the cached payload of each video releases name, making
// the distinct ones the cache lacks as many at a time as there are CPUs. An
// encode runs on one thread, so only running several at once spreads a cold
// launch over the machine.
func payloads(ctx context.Context, cfg Config, version []byte, releases []scenario.Release) (map[scenario.Video]string, error) {
	type encode struct {
		video   scenario.Video
		release string
		path    string
		err     error
	}
	var encodes []encode
	for _, r := range releases {
		if r.Video != nil && !slices.ContainsFunc(encodes, func(e encode) bool { return e.video == *r.Video }) {
			encodes = append(encodes, encode{video: *r.Video, release: r.Name})
		}
	}
	slots := make(chan struct{}, runtime.NumCPU())
	var wg sync.WaitGroup
	for i := range encodes {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			encodes[i].path, encodes[i].err = payload(ctx, cfg, version, encodes[i].video)
		})
	}
	wg.Wait()
	paths := make(map[scenario.Video]string, len(encodes))
	for _, e := range encodes {
		if e.err != nil {
			return nil, fmt.Errorf("release %s: %w", e.release, e.err)
		}
		paths[e.video] = e.path
	}
	return paths, nil
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
	// Worlds that launch together on a cold cache would each encode every
	// payload, so one encodes while the others wait for its file.
	lock := flock.New(path + ".lock")
	if _, err := lock.TryLockContext(ctx, 50*time.Millisecond); err != nil {
		return "", fmt.Errorf("lock the payload: %w", err)
	}
	defer func() { _ = lock.Unlock() }()
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	tmp, err := os.CreateTemp(cfg.Cache, ".encoding-*.mkv")
	if err != nil {
		return "", fmt.Errorf("create the payload: %w", err)
	}
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmp.Name()) }()
	cmd := exec.CommandContext(ctx, cfg.Tools.FFmpeg, append(encodeArgs(v, tmp.Name()), "-y")...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	// A world past the unlocked Stat above may open path while it is
	// published. Windows refuses that open during a rename onto path, and
	// a link publishes it with no such window, as mediafix does.
	if err := os.Link(tmp.Name(), path); err != nil {
		return "", fmt.Errorf("cache the payload: %w", err)
	}
	return path, nil
}

// place writes r's payload into dir, hashing it as it goes, and builds
// its metainfo: the cached encode of its video as <name>.mkv, or each of
// its files under a <name> folder.
func place(ctx context.Context, dir string, media *mediafix.Builder, cached map[scenario.Video]string, r scenario.Release, announce string) (Release, error) {
	out := Release{Release: r, Dir: dir}
	b3, pieces := blake3.New(), &pieceHasher{}
	name, files := r.Name, []TorrentFile(nil)
	if r.Video != nil {
		name += ".mkv"
		in, err := os.Open(cached[*r.Video])
		if err != nil {
			return Release{}, fmt.Errorf("open the payload: %w", err)
		}
		err = addFile(&out, name, in, b3, pieces)
		_ = in.Close()
		if err != nil {
			return Release{}, err
		}
	}
	for _, f := range r.Files {
		at := r.Name + "/" + f.Path
		in, err := fileBytes(ctx, media, at, f)
		if err != nil {
			return Release{}, err
		}
		err = addFile(&out, at, in, b3, pieces)
		_ = in.Close()
		if err != nil {
			return Release{}, err
		}
		files = append(files, TorrentFile{Path: f.Path, Size: out.Files[len(out.Files)-1].Size})
	}
	meta, hash, err := metainfo(name, files, out.Size, pieces.sum(), announce, r.Public)
	if err != nil {
		return Release{}, err
	}
	out.Metainfo, out.InfoHash = meta, hash
	copy(out.BLAKE3[:], b3.Sum(nil))
	return out, nil
}

// fileBytes are the bytes of the file f, at in its torrent: media's
// encode of a video, else Filler's.
func fileBytes(ctx context.Context, media *mediafix.Builder, at string, f scenario.ReleaseFile) (io.ReadCloser, error) {
	if f.Video == nil {
		return io.NopCloser(bytes.NewReader(Filler(at, int64(f.Size)))), nil
	}
	built, err := media.Build(ctx, f.Media())
	if err != nil {
		return nil, err
	}
	in, err := os.Open(built)
	if err != nil {
		return nil, fmt.Errorf("open the payload: %w", err)
	}
	return in, nil
}

// addFile copies in to r's file at torrentPath under Dir, and to payload,
// which hashes the files back to back.
func addFile(r *Release, torrentPath string, in io.Reader, payload ...io.Writer) error {
	f := File{Path: torrentPath}
	path := r.Disk(f)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("copy the payload: %w", err)
	}
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("copy the payload: %w", err)
	}
	b3 := blake3.New()
	f.Size, err = io.Copy(io.MultiWriter(append([]io.Writer{out, b3}, payload...)...), in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("copy the payload: %w", err)
	}
	copy(f.BLAKE3[:], b3.Sum(nil))
	r.Files = append(r.Files, f)
	r.Size += f.Size
	return nil
}

// Filler is size bytes that depend only on key, the bytes of a payload
// file no encode makes.
func Filler(key string, size int64) []byte {
	b := make([]byte, size)
	_, _ = rand.NewChaCha8(sha256.Sum256([]byte(key))).Read(b)
	return b
}

// TorrentFile is one file of a multi-file torrent: its path under the
// torrent's folder, '/' separated, and its size.
type TorrentFile struct {
	Path string
	Size int64
}

// Metainfo is a catalog torrent of payload, for a world that makes its
// payloads without ffmpeg: the one file name, or, when files are listed,
// the folder name holding them, payload their bytes back to back.
func Metainfo(name string, files []TorrentFile, payload []byte, announce string, public bool) ([]byte, bencode.InfoHash, error) {
	pieces := &pieceHasher{}
	_, _ = pieces.Write(payload)
	return metainfo(name, files, int64(len(payload)), pieces.sum(), announce, public)
}

func metainfo(name string, files []TorrentFile, size int64, pieces []byte, announce string, public bool) ([]byte, bencode.InfoHash, error) {
	info := map[string]any{
		"name":         name,
		"piece length": PieceLength,
		"pieces":       pieces,
	}
	if len(files) == 0 {
		info["length"] = size
	}
	var list []any
	for _, f := range files {
		var path []any
		for part := range strings.SplitSeq(f.Path, "/") {
			path = append(path, part)
		}
		list = append(list, map[string]any{"length": f.Size, "path": path})
	}
	if len(list) > 0 {
		info["files"] = list
	}
	if !public {
		info["private"] = 1
	}
	meta, err := bencode.Encode(map[string]any{"announce": announce, "info": info})
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
