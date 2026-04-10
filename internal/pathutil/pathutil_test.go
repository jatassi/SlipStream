package pathutil

import "testing"

func TestPathsEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{
			name: "identical forward slash paths",
			a:    "D:/Plex/Shows/The Boys/S05E01.mkv",
			b:    "D:/Plex/Shows/The Boys/S05E01.mkv",
			want: true,
		},
		{
			name: "mixed separators same file (production bug)",
			a:    "D:/Plex/Shows/The Boys (2019)/Season 5/The Boys (2019) - S05E01.mkv",
			b:    `D:\Plex\Shows\The Boys (2019)\Season 5\The Boys (2019) - S05E01.mkv`,
			want: true,
		},
		{
			name: "different files",
			a:    "D:/Plex/Shows/The Boys/S05E01.mkv",
			b:    "D:/Plex/Shows/The Boys/S05E02.mkv",
			want: false,
		},
		{
			name: "redundant slashes",
			a:    "D:/Plex//Shows/The Boys/S05E01.mkv",
			b:    "D:/Plex/Shows/The Boys/S05E01.mkv",
			want: true,
		},
		{
			name: "both empty",
			a:    "",
			b:    "",
			want: true,
		},
		{
			name: "one empty",
			a:    "",
			b:    "D:/Plex/Shows/The Boys/S05E01.mkv",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PathsEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("PathsEqual(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestTrimPathPrefix(t *testing.T) {
	tests := []struct {
		name          string
		path, prefix  string
		wantRemainder string
		wantOK        bool
	}{
		{
			name:          "exact equal",
			path:          "D:/foo",
			prefix:        "D:/foo",
			wantRemainder: "",
			wantOK:        true,
		},
		{
			name:          "strict child",
			path:          "D:/foo/bar/baz.mkv",
			prefix:        "D:/foo",
			wantRemainder: "/bar/baz.mkv",
			wantOK:        true,
		},
		{
			name:          "mixed separators (cross-origin DB)",
			path:          `D:\Downloads\SlipStream\The.Boys.mkv`,
			prefix:        "D:/Downloads/SlipStream",
			wantRemainder: "/The.Boys.mkv",
			wantOK:        true,
		},
		{
			name:          "reverse mixed separators",
			path:          "D:/Downloads/SlipStream/The.Boys.mkv",
			prefix:        `D:\Downloads\SlipStream`,
			wantRemainder: "/The.Boys.mkv",
			wantOK:        true,
		},
		{
			name:          "boundary guard rejects substring",
			path:          "D:/foobar/baz.mkv",
			prefix:        "D:/foo",
			wantRemainder: "",
			wantOK:        false,
		},
		{
			name:          "prefix with trailing slash equals path",
			path:          "D:/foo",
			prefix:        "D:/foo/",
			wantRemainder: "",
			wantOK:        true,
		},
		{
			name:          "prefix with trailing slash strict child",
			path:          "D:/foo/bar.mkv",
			prefix:        "D:/foo/",
			wantRemainder: "/bar.mkv",
			wantOK:        true,
		},
		{
			name:          "unrelated paths",
			path:          "D:/foo/bar.mkv",
			prefix:        "E:/baz",
			wantRemainder: "",
			wantOK:        false,
		},
		{
			name:          "redundant slashes normalized",
			path:          "D:/foo//bar//baz.mkv",
			prefix:        "D:/foo",
			wantRemainder: "/bar/baz.mkv",
			wantOK:        true,
		},
		{
			name:          "unix root prefix keeps full path",
			path:          "/media/shows/foo.mkv",
			prefix:        "/",
			wantRemainder: "/media/shows/foo.mkv",
			wantOK:        true,
		},
		{
			name:          "empty path",
			path:          "",
			prefix:        "D:/foo",
			wantRemainder: "",
			wantOK:        false,
		},
		{
			name:          "empty prefix",
			path:          "D:/foo",
			prefix:        "",
			wantRemainder: "",
			wantOK:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRemainder, gotOK := TrimPathPrefix(tt.path, tt.prefix)
			if gotOK != tt.wantOK || gotRemainder != tt.wantRemainder {
				t.Errorf("TrimPathPrefix(%q, %q) = (%q, %v), want (%q, %v)",
					tt.path, tt.prefix, gotRemainder, gotOK, tt.wantRemainder, tt.wantOK)
			}
			if gotHas := HasPathPrefix(tt.path, tt.prefix); gotHas != tt.wantOK {
				t.Errorf("HasPathPrefix(%q, %q) = %v, want %v",
					tt.path, tt.prefix, gotHas, tt.wantOK)
			}
		})
	}
}
