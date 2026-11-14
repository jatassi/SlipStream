package link_test

import (
	"errors"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/storing"
)

func path(t *testing.T, s string) link.ServerPath {
	t.Helper()
	p, err := link.ParsePath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func rel(t *testing.T, s string) storing.Rel {
	t.Helper()
	r, err := storing.ParseRel(s)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestParsePathKeepsTheServersSeparator(t *testing.T) {
	for _, c := range []struct {
		in, want string
		windows  bool
	}{
		{"/media/movies", "/media/movies", false},
		{"/media//movies/", "/media/movies", false},
		{"/", "/", false},
		{`D:\Movies`, `D:\Movies`, true},
		{`D:\Movies\`, `D:\Movies`, true},
		{"D:/Movies/Heat (1995)", `D:\Movies\Heat (1995)`, true},
		{`D:\`, `D:\`, true},
		{"D:", `D:\`, true},
		{`\\nas\media\movies`, `\\nas\media\movies`, true},
		{`\\nas\media\`, `\\nas\media`, true},
	} {
		p := path(t, c.in)
		if p.String() != c.want || p.Windows() != c.windows {
			t.Errorf("ParsePath(%q) = %q windows %v, want %q windows %v", c.in, p, p.Windows(), c.want, c.windows)
		}
	}
}

func TestParsePathRefusesARelativePath(t *testing.T) {
	for _, s := range []string{"", "media/movies", `Movies\Heat`, "D:Movies", `\\nas`, `\Movies`} {
		if _, err := link.ParsePath(s); !errors.Is(err, link.ErrNotAbsolute) {
			t.Errorf("ParsePath(%q) = %v, want ErrNotAbsolute", s, err)
		}
	}
}

func TestJoinUsesTheServersSeparator(t *testing.T) {
	r := rel(t, "Heat (1995)/Heat (1995).mkv")
	for _, c := range []struct{ folder, want string }{
		{"/media/movies", "/media/movies/Heat (1995)/Heat (1995).mkv"},
		{"/", "/Heat (1995)/Heat (1995).mkv"},
		{`D:\Movies`, `D:\Movies\Heat (1995)\Heat (1995).mkv`},
		{`D:\`, `D:\Heat (1995)\Heat (1995).mkv`},
		{`\\nas\media`, `\\nas\media\Heat (1995)\Heat (1995).mkv`},
	} {
		if got := path(t, c.folder).Join(r).String(); got != c.want {
			t.Errorf("%q joined = %q, want %q", c.folder, got, c.want)
		}
	}
}

func TestDirIsTheFolder(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"/media/movies/Heat (1995)/Heat (1995).mkv", "/media/movies/Heat (1995)"},
		{"/media", "/"},
		{"/", "/"},
		{`D:\Movies\Heat.mkv`, `D:\Movies`},
		{`D:\Movies`, `D:\`},
		{`\\nas\media\Heat.mkv`, `\\nas\media`},
		{`\\nas\media`, `\\nas\media`},
	} {
		if got := path(t, c.in).Dir().String(); got != c.want {
			t.Errorf("Dir(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestWithinAndRelFoldCaseOnlyOnWindows(t *testing.T) {
	for _, c := range []struct {
		path, folder string
		rel          string
		within       bool
	}{
		{"/media/movies/Heat/Heat.mkv", "/media/movies", "Heat/Heat.mkv", true},
		{"/media/Movies/Heat.mkv", "/media/movies", "", false},
		{"/media/movies2/Heat.mkv", "/media/movies", "", false},
		{"/media/movies", "/media/movies", "", true},
		{`D:\Movies\Heat\Heat.mkv`, `d:\movies`, "Heat/Heat.mkv", true},
		{`D:\MOVIES\Heat.mkv`, "D:/Movies", "Heat.mkv", true},
		{`E:\Movies\Heat.mkv`, `D:\Movies`, "", false},
		{`\\NAS\Media\Heat.mkv`, `\\nas\media`, "Heat.mkv", true},
		{`\\nas\tv\Heat.mkv`, `\\nas\media`, "", false},
		{`D:\Movies\Heat.mkv`, "/Movies", "", false},
	} {
		p, f := path(t, c.path), path(t, c.folder)
		if got := p.Within(f); got != c.within {
			t.Errorf("%q within %q = %v, want %v", c.path, c.folder, got, c.within)
		}
		got, ok := p.Rel(f)
		if (c.rel != "") != ok || got.String() != c.rel {
			t.Errorf("%q rel %q = %q %v, want %q", c.path, c.folder, got, ok, c.rel)
		}
	}
}

func TestRelIsJoinsInverse(t *testing.T) {
	for _, folder := range []string{"/media/movies", `D:\Movies`, `\\nas\media`} {
		f := path(t, folder)
		r := rel(t, "Show/Season 01/Show - S01E01.mkv")
		if got, ok := f.Join(r).Rel(f); !ok || got != r {
			t.Errorf("%q: Rel(Join(r)) = %q %v, want %q", folder, got, ok, r)
		}
	}
}

func TestKeyFoldsCaseOnWindows(t *testing.T) {
	if a, b := path(t, `D:\Movies\Heat.mkv`).Key(), path(t, "d:/movies/heat.mkv").Key(); a != b {
		t.Errorf("Windows keys differ: %q and %q", a, b)
	}
	if a, b := path(t, "/media/Heat.mkv").Key(), path(t, "/media/heat.mkv").Key(); a == b {
		t.Errorf("POSIX keys fold case: %q", a)
	}
}
