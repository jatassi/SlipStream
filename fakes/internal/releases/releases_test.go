package releases_test

import (
	"bytes"
	"encoding/hex"
	"net/netip"
	"os"
	"path"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/fakes/internal/mediafix"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/bencode"
)

// tools are the bundled tools that go run ./cmd/release test names.
func tools(t *testing.T) mediafix.Tools {
	t.Helper()
	tools, err := mediafix.NativeTools()
	if err != nil {
		t.Fatal(err)
	}
	return tools
}

func TestBuildMakesTheSameBytesFromOneInput(t *testing.T) {
	s := scenario.Scenario{Seed: "catalog", Releases: []scenario.Release{
		{Name: "The.Quiet.Orchard.2021.360p", TMDBID: 9000001, Video: &scenario.Video{Width: 640, Height: 360, Seconds: 2}, Seeded: true},
	}}
	tracker := netip.MustParseAddrPort("127.0.0.1:8443")
	build := func(cache string) releases.Release {
		t.Helper()
		dir := t.TempDir()
		c, err := releases.Build(t.Context(), s, releases.Config{Tools: tools(t), Cache: cache, Dir: dir, Tracker: tracker})
		if err != nil {
			t.Fatal(err)
		}
		if len(c.Releases) != 1 {
			t.Fatalf("catalog = %+v", c)
		}
		r := c.Releases[0]
		if len(r.Files) != 1 || r.Files[0].Path != "The.Quiet.Orchard.2021.360p.mkv" || r.Dir != dir || r.Files[0].Size != r.Size || r.Files[0].BLAKE3 != r.BLAKE3 {
			t.Fatalf("payload is %+v in %s, want The.Quiet.Orchard.2021.360p.mkv in %s", r.Files, r.Dir, dir)
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
	wantAnnounce := "https://127.0.0.1:8443/announce/" + releases.Passkey(s, releases.Seeders)
	if m.Announce != wantAnnounce || m.Info.Name != "The.Quiet.Orchard.2021.360p.mkv" || !m.Info.Private ||
		m.Info.PieceLength != 256<<10 || len(m.Info.Files) != 1 || m.Info.Files[0].Length != first.Size || m.InfoHash != first.InfoHash {
		t.Fatalf("metainfo = %+v, want announce %s", m, wantAnnounce)
	}
	if want := int((first.Size + 256<<10 - 1) / (256 << 10) * 20); len(m.Info.Pieces) != want {
		t.Fatalf("metainfo has %d piece hash bytes, want %d", len(m.Info.Pieces), want)
	}
	data, err := os.ReadFile(first.Disk(first.Files[0]))
	if err != nil || int64(len(data)) != first.Size || string(data[:4]) != "\x1a\x45\xdf\xa3" {
		t.Fatalf("payload is %d bytes starting %x, %v; want %d bytes of Matroska", len(data), data[:min(4, len(data))], err, first.Size)
	}
}

// TestWorldsSharingACacheAllPlaceTheirPayload runs worlds at once on one
// empty cache, as parallel browser workers do: each encodes the same
// payload, and none may fail to read the one another published.
func TestWorldsSharingACacheAllPlaceTheirPayload(t *testing.T) {
	s := scenario.Scenario{Seed: "shared", Releases: []scenario.Release{
		{Name: "The.Quiet.Orchard.2021.360p", TMDBID: 9000001, Video: &scenario.Video{Width: 320, Height: 180, Seconds: 1}, Seeded: true},
	}}
	tracker := netip.MustParseAddrPort("127.0.0.1:8443")
	for range 4 {
		cache := t.TempDir()
		var worlds sync.WaitGroup
		for range 12 {
			worlds.Go(func() {
				if _, err := releases.Build(t.Context(), s, releases.Config{Tools: tools(t), Cache: cache, Dir: t.TempDir(), Tracker: tracker}); err != nil {
					t.Error(err)
				}
			})
		}
		worlds.Wait()
	}
}

func TestEachReleaseHoldsItsOwnVideo(t *testing.T) {
	short := scenario.Video{Width: 320, Height: 180, Seconds: 1}
	long := scenario.Video{Width: 320, Height: 180, Seconds: 2}
	s := scenario.Scenario{Seed: "videos", Releases: []scenario.Release{
		{Name: "The.Quiet.Orchard.2021.WEB-DL", TMDBID: 9000001, Video: &short, Seeded: true},
		{Name: "The.Quiet.Orchard.2021.BluRay", TMDBID: 9000001, Video: &long, Seeded: true},
		{Name: "The.Quiet.Orchard.2021.REPACK.WEB-DL", TMDBID: 9000001, Video: &short, Seeded: true},
	}}
	tracker := netip.MustParseAddrPort("127.0.0.1:8443")
	alone := func(v scenario.Video) [32]byte {
		t.Helper()
		one := scenario.Scenario{Seed: "videos", Releases: []scenario.Release{{Name: "alone", TMDBID: 9000001, Video: &v, Seeded: true}}}
		c, err := releases.Build(t.Context(), one, releases.Config{Tools: tools(t), Cache: t.TempDir(), Dir: t.TempDir(), Tracker: tracker})
		if err != nil {
			t.Fatal(err)
		}
		return c.Releases[0].BLAKE3
	}
	c, err := releases.Build(t.Context(), s, releases.Config{Tools: tools(t), Cache: t.TempDir(), Dir: t.TempDir(), Tracker: tracker})
	if err != nil {
		t.Fatal(err)
	}
	want := map[scenario.Video][32]byte{short: alone(short), long: alone(long)}
	if want[short] == want[long] {
		t.Fatal("a 1 second and a 2 second video have one payload")
	}
	for i, r := range c.Releases {
		if r.Name != s.Releases[i].Name || r.BLAKE3 != want[*r.Video] {
			t.Errorf("release %d is %s with BLAKE3 %x, want %s with its %v video's %x", i, r.Name, r.BLAKE3, s.Releases[i].Name, r.Video, want[*r.Video])
		}
	}
}

func TestPasskeyComesFromTheSeedAndTheAccount(t *testing.T) {
	home, office := scenario.Scenario{Seed: "home"}, scenario.Scenario{Seed: "office"}
	a := releases.Passkey(home, "heron")
	others := []string{releases.Passkey(office, "heron"), releases.Passkey(home, "tamarack"), releases.Passkey(home, releases.Seeders)}
	if _, err := hex.DecodeString(a); err != nil || len(a) != 32 || slices.Contains(others, a) || a != releases.Passkey(home, "heron") {
		t.Fatalf("passkey %q, others %q", a, others)
	}
}

func TestAPublicReleaseIsAnotherTorrentWithNoPrivateKey(t *testing.T) {
	payload := []byte("the orchard payload")
	private, privateHash, err := releases.Metainfo("orchard.mkv", nil, payload, "https://127.0.0.1:1/announce/key", false)
	if err != nil {
		t.Fatal(err)
	}
	public, publicHash, err := releases.Metainfo("orchard.mkv", nil, payload, "https://127.0.0.1:1/announce/key", true)
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(public)
	if err != nil {
		t.Fatal(err)
	}
	if m.Info.Private || bytes.Contains(m.RawInfo, []byte("7:private")) || !bytes.Contains(private, []byte("7:privatei1e")) {
		t.Fatalf("the public info dictionary is %q, want no private key", m.RawInfo)
	}
	if publicHash == privateHash || m.InfoHash != publicHash {
		t.Fatalf("the public torrent's info hash is %s and the private one's %s, want two hashes", publicHash, privateHash)
	}
}

func TestAMetainfoOfFilesListsThemUnderItsFolder(t *testing.T) {
	files := []releases.TorrentFile{{Path: "Season 01/Lantern.S01E01.mkv", Size: 300 << 10}, {Path: "Lantern.nfo", Size: 120}}
	payload := slices.Concat(releases.Filler("a", 300<<10), releases.Filler("b", 120))
	meta, hash, err := releases.Metainfo("Lantern.S01", files, payload, "https://127.0.0.1:1/announce/key", false)
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(meta)
	if err != nil {
		t.Fatal(err)
	}
	want := []bencode.File{{Path: []string{"Season 01", "Lantern.S01E01.mkv"}, Length: 300 << 10}, {Path: []string{"Lantern.nfo"}, Length: 120}}
	if m.Info.Name != "Lantern.S01" || !reflect.DeepEqual(m.Info.Files, want) || m.InfoHash != hash || len(m.Info.Pieces) != 2*20 {
		t.Fatalf("metainfo is %s with files %+v, want Lantern.S01 with %+v and two pieces", m.Info.Name, m.Info.Files, want)
	}
}

func TestBuildLaysAReleaseOfFilesOutUnderItsFolder(t *testing.T) {
	episode := &scenario.MediaVideo{Width: 320, Height: 180, Seconds: 1, HDR: "sdr"}
	s := scenario.Scenario{Seed: "pack", Releases: []scenario.Release{{
		Name: "Lantern.S01", TVDB: &scenario.ReleaseEpisode{Series: 1001, Season: 1}, Seeded: true, Files: []scenario.ReleaseFile{
			{Path: "Lantern.S01E01.mkv", Items: []scenario.TVDBNumber{{Season: 1, Episode: 1}}, Video: episode, Size: 1 << 20},
			{Path: "Sample/lantern.sample.mkv", Unreadable: true, Size: 2048},
			{Path: "Lantern.nfo", Size: 120},
		},
	}}}
	dir := t.TempDir()
	c, err := releases.Build(t.Context(), s, releases.Config{Tools: tools(t), Cache: t.TempDir(), Dir: dir, Tracker: netip.MustParseAddrPort("127.0.0.1:8443")})
	if err != nil {
		t.Fatal(err)
	}
	r := c.Releases[0]
	m, err := bencode.DecodeMetainfo(r.Metainfo)
	if err != nil {
		t.Fatal(err)
	}
	if m.Info.Name != "Lantern.S01" || len(m.Info.Files) != 3 || len(r.Files) != 3 {
		t.Fatalf("metainfo %s lists %+v, and the release %+v", m.Info.Name, m.Info.Files, r.Files)
	}
	var payload []byte
	for i, f := range r.Files {
		data, err := os.ReadFile(r.Disk(f))
		if err != nil {
			t.Fatal(err)
		}
		at := "Lantern.S01/" + s.Releases[0].Files[i].Path
		if f.Path != at || f.Size != int64(len(data)) || f.BLAKE3 != blake3.Sum256(data) || m.Info.Files[i].Length != f.Size || path.Join(m.Info.Files[i].Path...) != s.Releases[0].Files[i].Path {
			t.Errorf("file %d is %s of %d bytes, want %s holding the %d bytes on disk, as the metainfo lists it", i, f.Path, f.Size, at, len(data))
		}
		payload = append(payload, data...)
	}
	if string(payload[:4]) != "\x1a\x45\xdf\xa3" {
		t.Errorf("the episode starts %x, want Matroska", payload[:4])
	}
	if want := slices.Concat(releases.Filler("Lantern.S01/Sample/lantern.sample.mkv", 2048), releases.Filler("Lantern.S01/Lantern.nfo", 120)); !bytes.Equal(payload[r.Files[0].Size:], want) {
		t.Error("the unreadable sample and the nfo aren't Filler's bytes")
	}
	if r.Size != int64(len(payload)) || r.BLAKE3 != blake3.Sum256(payload) {
		t.Errorf("the payload is %d bytes, want the files' %d back to back", r.Size, len(payload))
	}
}
