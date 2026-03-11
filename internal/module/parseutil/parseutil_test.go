package parseutil

import (
	"sort"
	"testing"
)

func TestParseVideoQuality(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantQuality string
		wantSource  string
		wantCodec   string
	}{
		{
			name:        "standard 1080p BluRay x264",
			filename:    "Movie.2020.1080p.BluRay.x264",
			wantQuality: "1080p",
			wantSource:  "BluRay",
			wantCodec:   "x264",
		},
		{
			name:        "2160p WEB-DL x265",
			filename:    "Movie.2020.2160p.WEB-DL.x265",
			wantQuality: "2160p",
			wantSource:  "WEB-DL",
			wantCodec:   "x265",
		},
		{
			name:        "4K alias maps to 2160p",
			filename:    "Movie.2020.4K.BluRay.x265",
			wantQuality: "2160p",
			wantSource:  "BluRay",
			wantCodec:   "x265",
		},
		{
			name:        "720p HDTV",
			filename:    "Show.S01E01.720p.HDTV.x264",
			wantQuality: "720p",
			wantSource:  "HDTV",
			wantCodec:   "x264",
		},
		{
			name:        "Remux takes priority over BluRay",
			filename:    "Movie.2020.UHD.BluRay.2160p.REMUX.HEVC",
			wantQuality: "2160p",
			wantSource:  "Remux",
			wantCodec:   "x265",
		},
		{
			name:        "AV1 codec",
			filename:    "Movie.2020.1080p.WEB-DL.AV1",
			wantQuality: "1080p",
			wantSource:  "WEB-DL",
			wantCodec:   "AV1",
		},
		{
			name:        "no quality info",
			filename:    "some-random-file",
			wantQuality: "",
			wantSource:  "",
			wantCodec:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quality, source, codec := ParseVideoQuality(tt.filename)
			if quality != tt.wantQuality {
				t.Errorf("quality = %q, want %q", quality, tt.wantQuality)
			}
			if source != tt.wantSource {
				t.Errorf("source = %q, want %q", source, tt.wantSource)
			}
			if codec != tt.wantCodec {
				t.Errorf("codec = %q, want %q", codec, tt.wantCodec)
			}
		})
	}
}

func TestParseHDRFormats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantLen int
	}{
		{"Dolby Vision", "Movie.2024.2160p.BluRay.DoVi.x265.mkv", []string{"DV"}, 1},
		{"HDR10+", "Movie.2024.2160p.BluRay.HDR10+.x265.mkv", []string{"HDR10+"}, 1},
		{"HDR10", "Movie.2024.2160p.BluRay.HDR10.x265.mkv", []string{"HDR10"}, 1},
		{"generic HDR", "Movie.2024.2160p.BluRay.HDR.x265.mkv", []string{"HDR"}, 1},
		{"HLG", "Movie.2024.2160p.BluRay.HLG.x265.mkv", []string{"HLG"}, 1},
		{"DV + HDR10 combo", "Movie.2024.2160p.BluRay.DV.HDR10.x265.mkv", []string{"DV", "HDR10"}, 2},
		{"SDR (no HDR)", "Movie.2024.1080p.BluRay.x264.mkv", nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseHDRFormats(tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("ParseHDRFormats() returned %d formats %v, want %d %v", len(got), got, tt.wantLen, tt.want)
				return
			}
			for i, format := range got {
				if format != tt.want[i] {
					t.Errorf("ParseHDRFormats()[%d] = %q, want %q", i, format, tt.want[i])
				}
			}
		})
	}
}

func TestParseAudioInfo(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		wantCodecs       []string
		wantChannels     []string
		wantEnhancements []string
	}{
		{
			name:             "TrueHD Atmos 7.1",
			input:            "Movie.2024.2160p.BluRay.TrueHD.Atmos.7.1.x265",
			wantCodecs:       []string{"TrueHD"},
			wantChannels:     []string{"7.1"},
			wantEnhancements: []string{"Atmos"},
		},
		{
			name:             "DDP 5.1",
			input:            "Movie.2024.1080p.WEB-DL.DDP.5.1.x264",
			wantCodecs:       []string{"DDP"},
			wantChannels:     []string{"5.1"},
			wantEnhancements: nil,
		},
		{
			name:             "no audio info",
			input:            "Movie.2024.1080p.BluRay.x264",
			wantCodecs:       nil,
			wantChannels:     nil,
			wantEnhancements: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codecs, channels, enhancements := ParseAudioInfo(tt.input)
			assertSliceEqual(t, "codecs", codecs, tt.wantCodecs)
			assertSliceEqual(t, "channels", channels, tt.wantChannels)
			assertSliceEqual(t, "enhancements", enhancements, tt.wantEnhancements)
		})
	}
}

func TestDetectQualityAttributes(t *testing.T) {
	attrs := DetectQualityAttributes("Movie.2020.1080p.BluRay.x264.DTS.5.1.HDR.mkv")
	if attrs.Quality != "1080p" {
		t.Errorf("Quality = %q, want %q", attrs.Quality, "1080p")
	}
	if attrs.Source != "BluRay" {
		t.Errorf("Source = %q, want %q", attrs.Source, "BluRay")
	}
	if attrs.Codec != "x264" {
		t.Errorf("Codec = %q, want %q", attrs.Codec, "x264")
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Movie.Name.2020", "Movie Name 2020"},
		{"Movie_Name_2020", "Movie Name 2020"},
		{"Movie-Name-2020", "Movie Name 2020"},
		{"  Spaced  Out  ", "Spaced Out"},
		{"Already Clean", "Already Clean"},
		{"Breaking.Bad", "Breaking Bad"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CleanTitle(tt.input)
			if got != tt.want {
				t.Errorf("CleanTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractYear(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantYear      int
		wantRemainder string
		wantFound     bool
	}{
		{
			name:          "year in parentheses",
			input:         "Movie Name (2020)",
			wantYear:      2020,
			wantRemainder: "Movie Name",
			wantFound:     true,
		},
		{
			name:          "year in brackets",
			input:         "Movie Name [2019]",
			wantYear:      2019,
			wantRemainder: "Movie Name",
			wantFound:     true,
		},
		{
			name:          "bare year",
			input:         "Movie Name 2021",
			wantYear:      2021,
			wantRemainder: "Movie Name",
			wantFound:     true,
		},
		{
			name:          "no year",
			input:         "Movie Name",
			wantYear:      0,
			wantRemainder: "Movie Name",
			wantFound:     false,
		},
		{
			name:          "year out of range too old",
			input:         "Movie Name 1899",
			wantYear:      0,
			wantRemainder: "Movie Name 1899",
			wantFound:     false,
		},
		{
			name:          "year out of range too new",
			input:         "Movie Name 2101",
			wantYear:      0,
			wantRemainder: "Movie Name 2101",
			wantFound:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			year, remainder, found := ExtractYear(tt.input)
			if year != tt.wantYear {
				t.Errorf("year = %d, want %d", year, tt.wantYear)
			}
			if remainder != tt.wantRemainder {
				t.Errorf("remainder = %q, want %q", remainder, tt.wantRemainder)
			}
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
		})
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"The Movie Name", "movie name"},
		{"A Quiet Place", "quiet place"},
		{"An American Werewolf", "american werewolf"},
		{"Tron: Ares", "tron ares"},
		{"Spider-Man: No Way Home", "spider man no way home"},
		{"It's a Wonderful Life", "its a wonderful life"},
		{"Title (2020)", "title"},
		{"Movie.Name", "movie name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeTitle(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsVideoFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"movie.mkv", true},
		{"movie.MKV", true},
		{"movie.mp4", true},
		{"movie.avi", true},
		{"movie.txt", false},
		{"movie.srt", false},
		{"movie", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := IsVideoFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsVideoFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsSampleFile(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{"sample-file.mkv", true},
		{"Sample.mkv", true},
		{"movie-trailer.mkv", true},
		{"proof.mkv", true},
		{"movie.mkv", false},
		{"movie.1080p.mkv", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := IsSampleFile(tt.filename)
			if got != tt.want {
				t.Errorf("IsSampleFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestVideoExtensions(t *testing.T) {
	exts := VideoExtensions()
	if len(exts) == 0 {
		t.Fatal("VideoExtensions() returned empty slice")
	}

	sort.Strings(exts)
	for _, ext := range []string{".mkv", ".mp4", ".avi"} {
		found := false
		for _, e := range exts {
			if e == ext {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("VideoExtensions() missing %q", ext)
		}
	}
}

func TestParseReleaseGroup(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"standard group", "Movie.2020.1080p.BluRay.x264-SPARKS.mkv", "SPARKS"},
		{"group without extension", "Movie.2020.1080p.BluRay.x264-NTb", "NTb"},
		{"no group", "Movie.2020.1080p.BluRay.mkv", ""},
		{"false positive codec", "Movie.2020.1080p.BluRay-x264.mkv", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseReleaseGroup(tt.input)
			if got != tt.want {
				t.Errorf("ParseReleaseGroup(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRevision(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Movie.2020.PROPER.1080p", "Proper"},
		{"Movie.2020.REPACK.1080p", "REPACK"},
		{"Movie.2020.1080p", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseRevision(tt.input)
			if got != tt.want {
				t.Errorf("ParseRevision(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseEdition(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Movie.Directors.Cut.1080p", "Director's Cut"},
		{"Movie.Extended.1080p", "Extended"},
		{"Movie.IMAX.1080p", "IMAX"},
		{"Movie.1080p", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ParseEdition(tt.input)
			if got != tt.want {
				t.Errorf("ParseEdition(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseLanguages(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"German release", "Movie.2024.German.1080p.BluRay.mkv", []string{"German"}},
		{"French release", "Movie.2024.FRENCH.1080p.BluRay.mkv", []string{"French"}},
		{"no language", "Movie.2024.1080p.BluRay.mkv", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLanguages(tt.input)
			assertSliceEqual(t, "languages", got, tt.want)
		})
	}
}

func assertSliceEqual(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %v (len %d), want %v (len %d)", label, got, len(got), want, len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", label, i, got[i], want[i])
		}
	}
}
