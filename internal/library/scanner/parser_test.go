package scanner

import (
	"testing"
)

func TestParseFilename_TVShow_SEFormat(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantTitle  string
		wantSeason int
		wantEp     int
		wantEndEp  int
		wantIsTV   bool
	}{
		{
			name:       "standard S01E02 format",
			filename:   "Breaking.Bad.S01E02.1080p.BluRay.x264.mkv",
			wantTitle:  "Breaking Bad",
			wantSeason: 1,
			wantEp:     2,
			wantIsTV:   true,
		},
		{
			name:       "lowercase s01e02 format",
			filename:   "the.office.s03e15.720p.hdtv.mkv",
			wantTitle:  "the office",
			wantSeason: 3,
			wantEp:     15,
			wantIsTV:   true,
		},
		{
			name:       "multi-episode S01E01E02",
			filename:   "Game.of.Thrones.S01E01E02.1080p.mkv",
			wantTitle:  "Game of Thrones",
			wantSeason: 1,
			wantEp:     1,
			wantEndEp:  2,
			wantIsTV:   true,
		},
		{
			name:       "double digit season and episode",
			filename:   "Supernatural.S15E20.WEB-DL.x265.mkv",
			wantTitle:  "Supernatural",
			wantSeason: 15,
			wantEp:     20,
			wantIsTV:   true,
		},
		{
			name:       "with spaces in title",
			filename:   "The Walking Dead S10E05 720p.mkv",
			wantTitle:  "The Walking Dead",
			wantSeason: 10,
			wantEp:     5,
			wantIsTV:   true,
		},
		{
			name:       "with underscores",
			filename:   "Stranger_Things_S04E09_1080p.mkv",
			wantTitle:  "Stranger Things",
			wantSeason: 4,
			wantEp:     9,
			wantIsTV:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Season != tt.wantSeason {
				t.Errorf("Season = %d, want %d", result.Season, tt.wantSeason)
			}
			if result.Episode != tt.wantEp {
				t.Errorf("Episode = %d, want %d", result.Episode, tt.wantEp)
			}
			if result.EndEpisode != tt.wantEndEp {
				t.Errorf("EndEpisode = %d, want %d", result.EndEpisode, tt.wantEndEp)
			}
			if result.IsTV != tt.wantIsTV {
				t.Errorf("IsTV = %v, want %v", result.IsTV, tt.wantIsTV)
			}
		})
	}
}

func TestParseFilename_TVShow_XFormat(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantTitle  string
		wantSeason int
		wantEp     int
		wantIsTV   bool
	}{
		{
			name:       "standard 1x02 format",
			filename:   "Breaking.Bad.1x02.1080p.BluRay.mkv",
			wantTitle:  "Breaking Bad",
			wantSeason: 1,
			wantEp:     2,
			wantIsTV:   true,
		},
		{
			name:       "uppercase 3X15 format",
			filename:   "the.office.3X15.720p.hdtv.mkv",
			wantTitle:  "the office",
			wantSeason: 3,
			wantEp:     15,
			wantIsTV:   true,
		},
		{
			name:       "double digit season",
			filename:   "Supernatural.15x20.WEB-DL.mkv",
			wantTitle:  "Supernatural",
			wantSeason: 15,
			wantEp:     20,
			wantIsTV:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Season != tt.wantSeason {
				t.Errorf("Season = %d, want %d", result.Season, tt.wantSeason)
			}
			if result.Episode != tt.wantEp {
				t.Errorf("Episode = %d, want %d", result.Episode, tt.wantEp)
			}
			if result.IsTV != tt.wantIsTV {
				t.Errorf("IsTV = %v, want %v", result.IsTV, tt.wantIsTV)
			}
		})
	}
}

func TestParseFilename_TVShow_SeasonPack(t *testing.T) {
	tests := []struct {
		name           string
		filename       string
		wantTitle      string
		wantSeason     int
		wantEp         int
		wantIsTV       bool
		wantSeasonPack bool
	}{
		{
			name:           "season pack S01 format",
			filename:       "Mr Robot S01 iTALiAN MULTi 1080p BluRay x264-NTROPiC",
			wantTitle:      "Mr Robot",
			wantSeason:     1,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
		{
			name:           "season pack S02 format",
			filename:       "Breaking.Bad.S02.1080p.BluRay.x264",
			wantTitle:      "Breaking Bad",
			wantSeason:     2,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
		{
			name:           "season pack with hyphen",
			filename:       "The-Office-S03-720p-HDTV",
			wantTitle:      "The Office",
			wantSeason:     3,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
		{
			name:           "season pack double digit",
			filename:       "Supernatural.S15.Complete.1080p.WEB-DL",
			wantTitle:      "Supernatural",
			wantSeason:     15,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
		{
			name:           "special E00 is NOT a season pack",
			filename:       "Mr Robot S02E00 Special 1080p BluRay x264",
			wantTitle:      "Mr Robot",
			wantSeason:     2,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: false,
		},
		{
			name:           "regular episode is NOT a season pack",
			filename:       "Mr Robot S02E05 1080p BluRay x264",
			wantTitle:      "Mr Robot",
			wantSeason:     2,
			wantEp:         5,
			wantIsTV:       true,
			wantSeasonPack: false,
		},
		{
			name:           "spelled out Season 2 format",
			filename:       "Mr Robot Season 2 1080p BluRay",
			wantTitle:      "Mr Robot",
			wantSeason:     2,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
		{
			name:           "spelled out Season 02 with dots",
			filename:       "Mr.Robot.Season.02.1080p.BluRay",
			wantTitle:      "Mr Robot",
			wantSeason:     2,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
		{
			name:           "spelled out Season 1 Complete",
			filename:       "Breaking Bad Season 1 Complete 720p",
			wantTitle:      "Breaking Bad",
			wantSeason:     1,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
		{
			name:           "spelled out Season with underscores",
			filename:       "The_Office_Season_3_1080p_WEB-DL",
			wantTitle:      "The Office",
			wantSeason:     3,
			wantEp:         0,
			wantIsTV:       true,
			wantSeasonPack: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Season != tt.wantSeason {
				t.Errorf("Season = %d, want %d", result.Season, tt.wantSeason)
			}
			if result.Episode != tt.wantEp {
				t.Errorf("Episode = %d, want %d", result.Episode, tt.wantEp)
			}
			if result.IsTV != tt.wantIsTV {
				t.Errorf("IsTV = %v, want %v", result.IsTV, tt.wantIsTV)
			}
			if result.IsSeasonPack != tt.wantSeasonPack {
				t.Errorf("IsSeasonPack = %v, want %v", result.IsSeasonPack, tt.wantSeasonPack)
			}
		})
	}
}

func TestParseFilename_TVShow_CompleteSeries(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		wantTitle        string
		wantSeason       int
		wantEndSeason    int
		wantIsTV         bool
		wantSeasonPack   bool
		wantCompleteSeries bool
	}{
		{
			name:             "COMPLETE keyword only",
			filename:         "Mr Robot COMPLETE 1080p BluRay AV1 DDP 5 1-dAV1nci",
			wantTitle:        "Mr Robot",
			wantSeason:       0,
			wantEndSeason:    0,
			wantIsTV:         true,
			wantSeasonPack:   true,
			wantCompleteSeries: true,
		},
		{
			name:             "Complete Series phrase",
			filename:         "Breaking Bad Complete Series 1080p BluRay x264",
			wantTitle:        "Breaking Bad",
			wantSeason:       0,
			wantEndSeason:    0,
			wantIsTV:         true,
			wantSeasonPack:   true,
			wantCompleteSeries: true,
		},
		{
			name:             "The Complete Series phrase",
			filename:         "The Office The Complete Series 720p WEB-DL",
			wantTitle:        "The Office",
			wantSeason:       0,
			wantEndSeason:    0,
			wantIsTV:         true,
			wantSeasonPack:   true,
			wantCompleteSeries: true,
		},
		{
			name:             "season range S01-04 format",
			filename:         "Mr Robot S01-04 1080p BluRay x265-RARBG",
			wantTitle:        "Mr Robot",
			wantSeason:       1,
			wantEndSeason:    4,
			wantIsTV:         true,
			wantSeasonPack:   true,
			wantCompleteSeries: true,
		},
		{
			name:             "season range S01-S04 format",
			filename:         "Breaking Bad S01-S05 2160p UHD BluRay x265",
			wantTitle:        "Breaking Bad",
			wantSeason:       1,
			wantEndSeason:    5,
			wantIsTV:         true,
			wantSeasonPack:   true,
			wantCompleteSeries: true,
		},
		{
			name:             "complete series with year in parens",
			filename:         "Mr Robot (2015) Complete Series S01-S04 1080p BluRay x265 HEVC 10bit AAC 5 1 Vyndros",
			wantTitle:        "Mr Robot (2015) Complete Series",
			wantSeason:       1,
			wantEndSeason:    4,
			wantIsTV:         true,
			wantSeasonPack:   true,
			wantCompleteSeries: true,
		},
		{
			name:             "single season with Complete is NOT complete series",
			filename:         "Mr Robot S02 Complete 1080p BluRay x264",
			wantTitle:        "Mr Robot",
			wantSeason:       2,
			wantEndSeason:    0,
			wantIsTV:         true,
			wantSeasonPack:   true,
			wantCompleteSeries: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Season != tt.wantSeason {
				t.Errorf("Season = %d, want %d", result.Season, tt.wantSeason)
			}
			if result.EndSeason != tt.wantEndSeason {
				t.Errorf("EndSeason = %d, want %d", result.EndSeason, tt.wantEndSeason)
			}
			if result.IsTV != tt.wantIsTV {
				t.Errorf("IsTV = %v, want %v", result.IsTV, tt.wantIsTV)
			}
			if result.IsSeasonPack != tt.wantSeasonPack {
				t.Errorf("IsSeasonPack = %v, want %v", result.IsSeasonPack, tt.wantSeasonPack)
			}
			if result.IsCompleteSeries != tt.wantCompleteSeries {
				t.Errorf("IsCompleteSeries = %v, want %v", result.IsCompleteSeries, tt.wantCompleteSeries)
			}
		})
	}
}

func TestParseFilename_Movie_DotFormat(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantTitle string
		wantYear  int
		wantIsTV  bool
	}{
		{
			name:      "standard movie format",
			filename:  "The.Matrix.1999.1080p.BluRay.x264.mkv",
			wantTitle: "The Matrix",
			wantYear:  1999,
			wantIsTV:  false,
		},
		{
			name:      "movie with year in title",
			filename:  "2001.A.Space.Odyssey.1968.2160p.UHD.BluRay.mkv",
			wantTitle: "2001 A Space Odyssey",
			wantYear:  1968,
			wantIsTV:  false,
		},
		{
			name:      "movie with underscore",
			filename:  "Inception_2010_720p_WEB-DL.mkv",
			wantTitle: "Inception",
			wantYear:  2010,
			wantIsTV:  false,
		},
		{
			name:      "recent movie",
			filename:  "Dune.2021.1080p.WEBDL.x265.mkv",
			wantTitle: "Dune",
			wantYear:  2021,
			wantIsTV:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", result.Year, tt.wantYear)
			}
			if result.IsTV != tt.wantIsTV {
				t.Errorf("IsTV = %v, want %v", result.IsTV, tt.wantIsTV)
			}
		})
	}
}

func TestParseFilename_Movie_ParenFormat(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantTitle string
		wantYear  int
	}{
		{
			name:      "parentheses format",
			filename:  "The Matrix (1999) 1080p BluRay.mkv",
			wantTitle: "The Matrix",
			wantYear:  1999,
		},
		{
			name:      "parentheses with quality after",
			filename:  "Inception (2010) 720p WEB-DL x264.mkv",
			wantTitle: "Inception",
			wantYear:  2010,
		},
		{
			name:      "long title with parentheses",
			filename:  "The Lord of the Rings The Fellowship of the Ring (2001) 1080p.mkv",
			wantTitle: "The Lord of the Rings The Fellowship of the Ring",
			wantYear:  2001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", result.Year, tt.wantYear)
			}
			if result.IsTV {
				t.Error("IsTV = true, want false")
			}
		})
	}
}

func TestParseFilename_Quality(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantQuality string
	}{
		{
			name:        "2160p quality",
			filename:    "Movie.2020.2160p.BluRay.mkv",
			wantQuality: "2160p",
		},
		{
			name:        "4K quality alias",
			filename:    "Movie.2020.4K.BluRay.mkv",
			wantQuality: "2160p",
		},
		{
			name:        "UHD quality alias",
			filename:    "Movie.2020.UHD.BluRay.mkv",
			wantQuality: "2160p",
		},
		{
			name:        "1080p quality",
			filename:    "Movie.2020.1080p.WEB-DL.mkv",
			wantQuality: "1080p",
		},
		{
			name:        "720p quality",
			filename:    "Movie.2020.720p.HDTV.mkv",
			wantQuality: "720p",
		},
		{
			name:        "480p quality",
			filename:    "Movie.2020.480p.DVDRip.mkv",
			wantQuality: "480p",
		},
		{
			name:        "SD alias",
			filename:    "Movie.2020.SD.DVDRip.mkv",
			wantQuality: "480p",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Quality != tt.wantQuality {
				t.Errorf("Quality = %q, want %q", result.Quality, tt.wantQuality)
			}
		})
	}
}

func TestParseFilename_Source(t *testing.T) {
	tests := []struct {
		name       string
		filename   string
		wantSource string
	}{
		{
			name:       "BluRay source",
			filename:   "Movie.2020.1080p.BluRay.x264.mkv",
			wantSource: "BluRay",
		},
		{
			name:       "Blu-Ray hyphenated",
			filename:   "Movie.2020.1080p.Blu-Ray.x264.mkv",
			wantSource: "BluRay",
		},
		{
			name:       "BDRip source",
			filename:   "Movie.2020.1080p.BDRip.mkv",
			wantSource: "BluRay",
		},
		{
			name:       "BRRip source",
			filename:   "Movie.2020.1080p.BRRip.mkv",
			wantSource: "BluRay",
		},
		{
			name:       "WEB-DL source",
			filename:   "Movie.2020.1080p.WEB-DL.mkv",
			wantSource: "WEB-DL",
		},
		{
			name:       "WEBDL source",
			filename:   "Movie.2020.1080p.WEBDL.mkv",
			wantSource: "WEB-DL",
		},
		{
			name:       "WEBRip source",
			filename:   "Movie.2020.1080p.WEBRip.mkv",
			wantSource: "WEBRip",
		},
		{
			name:       "HDTV source",
			filename:   "Show.S01E01.720p.HDTV.mkv",
			wantSource: "HDTV",
		},
		{
			name:       "DVDRip source",
			filename:   "Movie.2005.DVDRip.mkv",
			wantSource: "DVDRip",
		},
		{
			name:       "Remux source",
			filename:   "Movie.2020.1080p.Remux.mkv",
			wantSource: "Remux",
		},
		{
			name:       "CAM source",
			filename:   "Movie.2020.CAM.mkv",
			wantSource: "CAM",
		},
		{
			name:       "HDCAM source",
			filename:   "Movie.2020.HDCAM.mkv",
			wantSource: "CAM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", result.Source, tt.wantSource)
			}
		})
	}
}

func TestParseFilename_Codec(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantCodec string
	}{
		{
			name:      "x264 codec",
			filename:  "Movie.2020.1080p.BluRay.x264.mkv",
			wantCodec: "x264",
		},
		{
			name:      "H.264 codec",
			filename:  "Movie.2020.1080p.BluRay.H.264.mkv",
			wantCodec: "x264",
		},
		{
			name:      "H264 codec",
			filename:  "Movie.2020.1080p.BluRay.H264.mkv",
			wantCodec: "x264",
		},
		{
			name:      "AVC codec",
			filename:  "Movie.2020.1080p.BluRay.AVC.mkv",
			wantCodec: "x264",
		},
		{
			name:      "x265 codec",
			filename:  "Movie.2020.1080p.BluRay.x265.mkv",
			wantCodec: "x265",
		},
		{
			name:      "H.265 codec",
			filename:  "Movie.2020.1080p.BluRay.H.265.mkv",
			wantCodec: "x265",
		},
		{
			name:      "HEVC codec",
			filename:  "Movie.2020.1080p.BluRay.HEVC.mkv",
			wantCodec: "x265",
		},
		{
			name:      "AV1 codec",
			filename:  "Movie.2020.1080p.WEB-DL.AV1.mkv",
			wantCodec: "AV1",
		},
		{
			name:      "VP9 codec",
			filename:  "Movie.2020.1080p.WEB-DL.VP9.mkv",
			wantCodec: "VP9",
		},
		{
			name:      "XviD codec",
			filename:  "Movie.2005.DVDRip.XviD.avi",
			wantCodec: "XviD",
		},
		{
			name:      "DivX codec",
			filename:  "Movie.2005.DVDRip.DivX.avi",
			wantCodec: "DivX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Codec != tt.wantCodec {
				t.Errorf("Codec = %q, want %q", result.Codec, tt.wantCodec)
			}
		})
	}
}

func TestParseFilename_Complex(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		wantTitle   string
		wantYear    int
		wantQuality string
		wantSource  string
		wantCodec   string
	}{
		{
			name:        "fully tagged movie",
			filename:    "The.Dark.Knight.2008.2160p.UHD.BluRay.x265.HDR.mkv",
			wantTitle:   "The Dark Knight",
			wantYear:    2008,
			wantQuality: "2160p",
			wantSource:  "BluRay",
			wantCodec:   "x265",
		},
		{
			name:        "web release movie",
			filename:    "Avengers.Endgame.2019.1080p.WEB-DL.H264.mkv",
			wantTitle:   "Avengers Endgame",
			wantYear:    2019,
			wantQuality: "1080p",
			wantSource:  "WEB-DL",
			wantCodec:   "x264",
		},
		{
			name:        "HDTV TV show",
			filename:    "Game.of.Thrones.S08E06.720p.HDTV.x264.mkv",
			wantTitle:   "Game of Thrones",
			wantQuality: "720p",
			wantSource:  "HDTV",
			wantCodec:   "x264",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if tt.wantYear > 0 && result.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", result.Year, tt.wantYear)
			}
			if result.Quality != tt.wantQuality {
				t.Errorf("Quality = %q, want %q", result.Quality, tt.wantQuality)
			}
			if result.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", result.Source, tt.wantSource)
			}
			if result.Codec != tt.wantCodec {
				t.Errorf("Codec = %q, want %q", result.Codec, tt.wantCodec)
			}
		})
	}
}

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Breaking.Bad", "Breaking Bad"},
		{"The.Walking.Dead", "The Walking Dead"},
		{"Game_of_Thrones", "Game of Thrones"},
		{"Movie-Title", "Movie Title"},
		{"  Spaced  Out  ", "Spaced Out"},
		{"Already Clean", "Already Clean"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := cleanTitle(tt.input)
			if result != tt.want {
				t.Errorf("cleanTitle(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantTitle string
		wantYear  int
	}{
		{
			name:      "movie with info in folder",
			path:      "/movies/The Matrix (1999)/The.Matrix.1080p.BluRay.mkv",
			wantTitle: "The Matrix",
			wantYear:  1999,
		},
		{
			name:      "movie with year only in folder",
			path:      "/movies/Inception (2010)/Inception.1080p.mkv",
			wantTitle: "Inception",
			wantYear:  2010,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePath(tt.path)

			if result.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", result.Title, tt.wantTitle)
			}
			if result.Year != tt.wantYear {
				t.Errorf("Year = %d, want %d", result.Year, tt.wantYear)
			}
		})
	}
}

func TestParseFilename_Fallback(t *testing.T) {
	// Test filenames that don't match standard patterns
	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "no year or episode",
			filename: "RandomVideo.mkv",
		},
		{
			name:     "just quality info",
			filename: "1080p.BluRay.x264.mkv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseFilename(tt.filename)

			// Should still return a valid parsed result
			if result == nil {
				t.Error("ParseFilename returned nil for valid file")
			}
			if result.FilePath != tt.filename {
				t.Errorf("FilePath = %q, want %q", result.FilePath, tt.filename)
			}
		})
	}
}
