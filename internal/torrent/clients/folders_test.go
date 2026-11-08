package clients_test

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestOnNodeRewritesTheLongestClientFolder(t *testing.T) {
	f := folders(t,
		clients.Folder{Client: "/downloads", Node: `D:\Torrents`},
		clients.Folder{Client: "/downloads/movies", Node: `E:\Movies`},
		clients.Folder{Client: "/data", Node: "/mnt/data"},
	)
	cases := []struct {
		path string
		sep  byte
		want string
		used string
	}{
		{"/downloads/The.Attic.2024/The.Attic.2024.mkv", '\\', `D:\Torrents\The.Attic.2024\The.Attic.2024.mkv`, "/downloads"},
		{"/downloads/movies/The.Attic.2024", '\\', `E:\Movies\The.Attic.2024`, "/downloads/movies"},
		{"/downloads/movies", '\\', `E:\Movies`, "/downloads/movies"},
		{"/downloads/movies2/x.mkv", '\\', `D:\Torrents\movies2\x.mkv`, "/downloads"},
		{"/downloads//./tv/../x.mkv", '\\', `D:\Torrents\x.mkv`, "/downloads"},
		{"/data/tv/Show", '/', "/mnt/data/tv/Show", "/data"},
	}
	for _, c := range cases {
		got, used, ok := f.OnNode(c.path, c.sep)
		if !ok || got != c.want || used.Client != c.used {
			t.Errorf("OnNode(%q) = %q, %q, %v, want %q through %q", c.path, got, used.Client, ok, c.want, c.used)
		}
	}
}

func TestOnNodeMatchesBySegmentNeverByLetters(t *testing.T) {
	f := folders(t, clients.Folder{Client: "/down", Node: "/mnt/down"})
	if got, _, ok := f.OnNode("/downloads/x.mkv", '/'); ok {
		t.Errorf("OnNode(/downloads/x.mkv) = %q through /down, want no pair", got)
	}
}

func TestOnNodeReadsWindowsClientPathsWithEitherSeparatorAndNoCase(t *testing.T) {
	f := folders(t,
		clients.Folder{Client: `C:\Users\jackson\Downloads`, Node: "/mnt/downloads"},
		clients.Folder{Client: `\\nas\torrents`, Node: "/mnt/nas"},
	)
	cases := map[string]string{
		`C:\Users\jackson\Downloads\The.Attic.2024\a.mkv`: "/mnt/downloads/The.Attic.2024/a.mkv",
		`c:/users/JACKSON/downloads/a.mkv`:                "/mnt/downloads/a.mkv",
		`\\NAS\Torrents\b.mkv`:                            "/mnt/nas/b.mkv",
		"//nas/torrents/c.mkv":                            "/mnt/nas/c.mkv",
	}
	for p, want := range cases {
		if got, _, ok := f.OnNode(p, '/'); !ok || got != want {
			t.Errorf("OnNode(%q) = %q, %v, want %q", p, got, ok, want)
		}
	}
}

func TestOnNodeMatchesPOSIXClientPathsWithCase(t *testing.T) {
	f := folders(t, clients.Folder{Client: "/Downloads", Node: `D:\Torrents`})
	if got, _, ok := f.OnNode("/downloads/a.mkv", '\\'); ok {
		t.Errorf("OnNode(/downloads/a.mkv) = %q, want no pair for another case on POSIX", got)
	}
}

func TestOnNodeKeepsANodeRootFolderWhole(t *testing.T) {
	f := folders(t, clients.Folder{Client: "/downloads", Node: `D:\`}, clients.Folder{Client: "/data", Node: "/"})
	cases := []struct {
		path string
		sep  byte
		want string
	}{
		{"/downloads/a.mkv", '\\', `D:\a.mkv`},
		{"/data/b.mkv", '/', "/b.mkv"},
	}
	for _, c := range cases {
		if got, _, ok := f.OnNode(c.path, c.sep); !ok || got != c.want {
			t.Errorf("OnNode(%q) = %q, %v, want %q", c.path, got, ok, c.want)
		}
	}
}

func TestOnNodeFailsAPathNoPairHolds(t *testing.T) {
	f := folders(t, clients.Folder{Client: "/downloads", Node: `D:\Torrents`})
	for _, p := range []string{"/incomplete/a.mkv", "downloads/a.mkv", `D:\downloads\a.mkv`} {
		if got, _, ok := f.OnNode(p, '\\'); ok {
			t.Errorf("OnNode(%q) = %q, want no pair", p, got)
		}
	}
}

func TestOnNodeWithNoPairsUsesThePathAsItIs(t *testing.T) {
	f := folders(t)
	got, used, ok := f.OnNode("/downloads/a.mkv", '\\')
	if !ok || got != "/downloads/a.mkv" || used != (clients.Folder{}) {
		t.Errorf("OnNode = %q, %+v, %v, want the path as it is", got, used, ok)
	}
}

func TestParseFoldersRefusesAPairItCantUse(t *testing.T) {
	cases := []struct {
		pairs []clients.Folder
		index int
		want  error
	}{
		{[]clients.Folder{{Client: "downloads", Node: "/mnt/d"}}, 0, clients.ErrClientFolderNotAbsolute},
		{[]clients.Folder{{Client: "/downloads", Node: `Torrents`}}, 0, clients.ErrNodeFolderNotAbsolute},
		{[]clients.Folder{{Client: "/downloads", Node: `D:`}}, 0, clients.ErrNodeFolderNotAbsolute},
		{[]clients.Folder{{Client: "/a", Node: "/mnt/a"}, {Client: "", Node: "/mnt/b"}}, 1, clients.ErrClientFolderNotAbsolute},
		{[]clients.Folder{{Client: "/downloads", Node: "/mnt/a"}, {Client: "/downloads/", Node: "/mnt/b"}}, 1, clients.ErrClientFolderTwice},
		{[]clients.Folder{{Client: `D:\Torrents`, Node: "/mnt/a"}, {Client: "d:/torrents", Node: "/mnt/b"}}, 1, clients.ErrClientFolderTwice},
	}
	for _, c := range cases {
		_, err := clients.ParseFolders(c.pairs)
		fe, ok := errors.AsType[*clients.FolderError](err)
		if !ok || fe.Index != c.index || !errors.Is(err, c.want) {
			t.Errorf("ParseFolders(%+v) = %v, want pair %d: %v", c.pairs, err, c.index, c.want)
		}
	}
}

func TestParseFoldersKeepsEachPairAsTyped(t *testing.T) {
	f := folders(t, clients.Folder{Client: " /downloads ", Node: `D:\Torrents`}, clients.Folder{Client: "/data", Node: "/mnt/data"})
	want := []clients.Folder{{Client: "/downloads", Node: `D:\Torrents`}, {Client: "/data", Node: "/mnt/data"}}
	got := f.Pairs()
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("Pairs = %+v, want %+v", got, want)
	}
}

func folders(t *testing.T, pairs ...clients.Folder) clients.Folders {
	t.Helper()
	f, err := clients.ParseFolders(pairs)
	if err != nil {
		t.Fatalf("ParseFolders: %v", err)
	}
	return f
}
