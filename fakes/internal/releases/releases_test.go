package releases_test

import (
	"encoding/hex"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
)

// ffmpeg is the bundled ffmpeg that go run ./cmd/release test names.
func ffmpeg(t *testing.T) string {
	t.Helper()
	for _, p := range filepath.SplitList(os.Getenv("SLIPSTREAM_NATIVE_TOOLS")) {
		if name := filepath.Base(p); name == "ffmpeg" || name == "ffmpeg.exe" {
			return p
		}
	}
	t.Fatal("SLIPSTREAM_NATIVE_TOOLS names no ffmpeg: run the tests with go run ./cmd/release test")
	return ""
}

func TestBuildMakesTheSameBytesFromOneInput(t *testing.T) {
	s := scenario.Scenario{Seed: "catalog", Releases: []scenario.Release{
		{Name: "The.Quiet.Orchard.2021.360p", TMDBID: 9000001, Video: scenario.Video{Width: 640, Height: 360, Seconds: 2}, Seeded: true},
	}}
	tracker := netip.MustParseAddrPort("127.0.0.1:8443")
	build := func(cache string) releases.Release {
		t.Helper()
		dir := t.TempDir()
		c, err := releases.Build(t.Context(), s, releases.Config{FFmpeg: ffmpeg(t), Cache: cache, Dir: dir, Tracker: tracker})
		if err != nil {
			t.Fatal(err)
		}
		if c.Passkey != releases.Passkey(s) || len(c.Releases) != 1 {
			t.Fatalf("catalog = %+v", c)
		}
		r := c.Releases[0]
		if want := filepath.Join(dir, "The.Quiet.Orchard.2021.360p.mkv"); r.File != want || r.Dir != dir {
			t.Fatalf("payload at %s in %s, want %s", r.File, r.Dir, want)
		}
		return r
	}
	first, second := build(t.TempDir()), build(t.TempDir())
	if first.BLAKE3 != second.BLAKE3 || first.InfoHash != second.InfoHash || first.Size != second.Size {
		t.Fatalf("two encodes differ: BLAKE3 %x and %x, info hash %s and %s", first.BLAKE3, second.BLAKE3, first.InfoHash, second.InfoHash)
	}

	m, err := bencode.DecodeMetainfo(first.Metainfo)
	if err != nil {
		t.Fatal(err)
	}
	wantAnnounce := "https://127.0.0.1:8443/announce/" + releases.Passkey(s)
	if m.Announce != wantAnnounce || m.Info.Name != "The.Quiet.Orchard.2021.360p.mkv" || !m.Info.Private ||
		m.Info.PieceLength != 256<<10 || len(m.Info.Files) != 1 || m.Info.Files[0].Length != first.Size || m.InfoHash != first.InfoHash {
		t.Fatalf("metainfo = %+v, want announce %s", m, wantAnnounce)
	}
	if want := int((first.Size + 256<<10 - 1) / (256 << 10) * 20); len(m.Info.Pieces) != want {
		t.Fatalf("metainfo has %d piece hash bytes, want %d", len(m.Info.Pieces), want)
	}
	data, err := os.ReadFile(first.File)
	if err != nil || int64(len(data)) != first.Size || string(data[:4]) != "\x1a\x45\xdf\xa3" {
		t.Fatalf("payload is %d bytes starting %x, %v; want %d bytes of Matroska", len(data), data[:min(4, len(data))], err, first.Size)
	}
}

func TestPasskeyComesFromTheSeed(t *testing.T) {
	a, b := releases.Passkey(scenario.Scenario{Seed: "home"}), releases.Passkey(scenario.Scenario{Seed: "office"})
	if _, err := hex.DecodeString(a); err != nil || len(a) != 32 || a == b || a != releases.Passkey(scenario.Scenario{Seed: "home"}) {
		t.Fatalf("passkeys %q and %q", a, b)
	}
}
