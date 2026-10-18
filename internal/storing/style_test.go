package storing

import (
	"errors"
	"slices"
	"testing"
)

func TestCleanBothStyles(t *testing.T) {
	for _, c := range []struct {
		style    PathStyle
		in, want string
	}{
		{Unix, "/", "/"},
		{Unix, "/srv//media/./movies/", "/srv/media/movies"},
		{Unix, "/srv/media/../tv", "/srv/tv"},
		{Unix, "/../..", "/"},
		{Unix, `/odd\name`, `/odd\name`},
		{Windows, `d:\Media\Movies`, `D:\Media\Movies`},
		{Windows, "d:/Media/Movies/", `D:\Media\Movies`},
		{Windows, `C:\`, `C:\`},
		{Windows, `C:/a\b/..\c`, `C:\a\c`},
		{Windows, `C:\..`, `C:\`},
		{Windows, `\\nas\media`, `\\nas\media\`},
		{Windows, `//nas/media/movies/`, `\\nas\media\movies`},
		{Windows, `\\nas\media\..`, `\\nas\media\`},
		{Windows, `\\?\c:\Long\Path`, `\\?\C:\Long\Path`},
		{Windows, `\\?\UNC\nas\media\tv`, `\\?\UNC\nas\media\tv`},
		{Windows, `\\.\D:\x`, `\\.\D:\x`},
	} {
		got, err := c.style.Clean(c.in)
		if err != nil || got != c.want {
			t.Errorf("style %d Clean(%q) = %q, %v; want %q", c.style, c.in, got, err, c.want)
		}
	}
}

func TestCleanRefusesRelativePaths(t *testing.T) {
	for _, c := range []struct {
		style PathStyle
		in    string
	}{
		{Unix, ""},
		{Unix, "srv/media"},
		{Unix, `C:\Media`},
		{Windows, ""},
		{Windows, "Media"},
		{Windows, `\Media`},
		{Windows, "C:"},
		{Windows, "C:Media"},
		{Windows, `\\nas`},
		{Windows, `\\nas\`},
		{Windows, `\\?\`},
		{Windows, `\\?\Media`},
		{Windows, "/srv/media"},
	} {
		if got, err := c.style.Clean(c.in); !errors.Is(err, ErrNotAbsolute) {
			t.Errorf("style %d Clean(%q) = %q, %v; want ErrNotAbsolute", c.style, c.in, got, err)
		}
	}
}

func TestSegmentsOfDrivesSharesAndRoot(t *testing.T) {
	for _, c := range []struct {
		style PathStyle
		in    string
		want  []Segment
	}{
		{Unix, "/", []Segment{{"/", "/"}}},
		{Unix, "/srv/media", []Segment{{"/", "/"}, {"srv", "/srv"}, {"media", "/srv/media"}}},
		{Windows, `C:\`, []Segment{{`C:\`, `C:\`}}},
		{Windows, `C:\Media\Movies`, []Segment{{`C:\`, `C:\`}, {"Media", `C:\Media`}, {"Movies", `C:\Media\Movies`}}},
		{Windows, `\\nas\media\tv`, []Segment{{`\\nas\media`, `\\nas\media\`}, {"tv", `\\nas\media\tv`}}},
		{Windows, `\\?\C:\x`, []Segment{{`\\?\C:\`, `\\?\C:\`}, {"x", `\\?\C:\x`}}},
		{Windows, `\\?\UNC\nas\media`, []Segment{{`\\?\UNC\nas\media`, `\\?\UNC\nas\media\`}}},
	} {
		if got := c.style.Segments(c.in); !slices.Equal(got, c.want) {
			t.Errorf("style %d Segments(%q) = %q, want %q", c.style, c.in, got, c.want)
		}
	}
}

func TestParentStopsAtTheRoot(t *testing.T) {
	for _, c := range []struct {
		style    PathStyle
		in, want string
		ok       bool
	}{
		{Unix, "/srv/media", "/srv", true},
		{Unix, "/srv", "/", true},
		{Unix, "/", "", false},
		{Windows, `C:\Media`, `C:\`, true},
		{Windows, `C:\`, "", false},
		{Windows, `\\nas\media\tv`, `\\nas\media\`, true},
		{Windows, `\\nas\media\`, "", false},
	} {
		if got, ok := c.style.Parent(c.in); got != c.want || ok != c.ok {
			t.Errorf("style %d Parent(%q) = %q, %v; want %q, %v", c.style, c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestJoinAndUnder(t *testing.T) {
	rel, err := ParseRel("Heat (1995)/Heat (1995).mkv")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		style      PathStyle
		root, want string
	}{
		{Unix, "/", "/Heat (1995)/Heat (1995).mkv"},
		{Unix, "/srv/movies", "/srv/movies/Heat (1995)/Heat (1995).mkv"},
		{Windows, `C:\`, `C:\Heat (1995)\Heat (1995).mkv`},
		{Windows, `D:\Movies`, `D:\Movies\Heat (1995)\Heat (1995).mkv`},
		{Windows, `\\nas\media\`, `\\nas\media\Heat (1995)\Heat (1995).mkv`},
	} {
		if got := c.style.Under(c.root, rel); got != c.want {
			t.Errorf("style %d Under(%q) = %q, want %q", c.style, c.root, got, c.want)
		}
	}
	if got := Windows.Join(`D:\Movies`, "New"); got != `D:\Movies\New` {
		t.Errorf("Windows.Join = %q", got)
	}
}

func TestSplitTypedKeepsTheSeparator(t *testing.T) {
	for _, c := range []struct {
		style             PathStyle
		typed, dir, start string
		sep               byte
		ok                bool
	}{
		{Unix, "/home/us", "/home", "us", '/', true},
		{Unix, "/home/", "/home", "", '/', true},
		{Unix, "/", "/", "", '/', true},
		{Unix, "home", "", "", 0, false},
		{Unix, "", "", "", 0, false},
		{Windows, "D:/Me", `D:\`, "Me", '/', true},
		{Windows, `d:\Media\Mo`, `D:\Media`, "Mo", '\\', true},
		{Windows, `D:\Media/`, `D:\Media`, "", '/', true},
		{Windows, `\\nas\media\T`, `\\nas\media\`, "T", '\\', true},
		{Windows, "D:", "", "", 0, false},
		{Windows, `\\nas\med`, "", "", 0, false},
		{Windows, "Media", "", "", 0, false},
	} {
		dir, start, sep, ok := c.style.Split(c.typed)
		if dir != c.dir || start != c.start || sep != c.sep || ok != c.ok {
			t.Errorf("style %d Split(%q) = %q, %q, %q, %v; want %q, %q, %q, %v",
				c.style, c.typed, dir, start, sep, ok, c.dir, c.start, c.sep, c.ok)
		}
	}
}
