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
