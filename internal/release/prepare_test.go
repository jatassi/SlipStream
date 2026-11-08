package release_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/release"
)

func TestPrepare(t *testing.T) {
	for _, c := range []struct{ name, title, simple string }{
		{
			"yrucreM-462.H.0.2CAA.LD-BEW.p027.10E10S.esaeleR.dehsaH.emoS.mkv",
			"Some.Hashed.Release.S01E01.720p.WEB-DL.AAC2.0.H.264-Mercury",
			"Some.Hashed.Release.S01E01..WEB-DL.AAC2.0.-Mercury",
		},
		{
			"[ www.Torrenting.com ] - Series.S03E14.720p.HDTV.X264-DIMENSION",
			"Series.S03E14.720p.HDTV.X264-DIMENSION",
			"Series.S03E14..HDTV.-DIMENSION",
		},
		{
			"Series.Title.S01E01.1080p.WEB-DL.DD5.1.H.264-GRP [rartv]",
			"Series.Title.S01E01.1080p.WEB-DL.DD5.1.H.264-GRP ",
			"Series.Title.S01E01..WEB-DL..-GRP ",
		},
		{"Series Title - S01E01 [HDTV 1080p]", "Series Title - S01E01 ", "Series Title - S01E01 "},
		{"Series Title - S01E01 [Director]", "Series Title - S01E01 [Director]", "Series Title - S01E01 [Director]"},
		{"Movie.Title.2010.720p.BluRay.x264.-[YTS.LT]", "Movie.Title.2010.720p.BluRay.x264.-", "Movie.Title.2010..BluRay..-"},
		{"【喵萌奶茶屋】★01月新番★[Series Title][01][1080p][简体]", "[喵萌奶茶屋] Series Title - 01 [1080p][简体]", "[喵萌奶茶屋] Series Title - 01 [][简体]"},
		{"Series.Title.S01E01.mkv", "Series.Title.S01E01", "Series.Title.S01E01"},
		{"Series.Title.S01E01.nfo", "Series.Title.S01E01.nfo", "Series.Title.S01E01.nfo"},
	} {
		p := release.Prepare(c.name)
		if p.Title.String() != c.title || p.Simple.String() != c.simple || p.Obfuscated {
			t.Errorf("%q prepares as %q, simple %q, obfuscated %v; want %q, simple %q", c.name, p.Title, p.Simple, p.Obfuscated, c.title, c.simple)
		}
	}
}

// The core rows of Sonarr's and Radarr's CrapParserFixture and Sonarr's
// HashedReleaseFixture: names ValidateBeforeParsing rejects.
var obfuscated = []string{
	"76El6LcgLzqb426WoVFg1vVVVGx4uCYopQkfjmLe",
	`password - "bdc435cb-93c4-4902-97ea-ca00568c3887.337" yEnc`,
	"185d86a343e39f3341e35c4dad3f9959",
	"ba27283b17c00d01193eacc02a8ba98eeb523a76",
	"Vh1FvU3bJXw6zs8EEUX4bMo5vbbMdHghxHirc.mkv",
	"0e895c37245186812cb08aab1529cf8ee389dd05.mkv",
	"ah63jka93jf0jh26ahjas961.mkv",
	"QZC4HDl7ncmzyUj9amucWe1ddKU1oFMZDd8r0dEDUsTd",
	"abc.xyz.af6021c37f7852",
	"AHFMZXGHEWD660.mkv",
	"Backup_72023S02-12.mkv",
	"123.mkv",
	"b00bs.mkv",
	"cgajsofuejsa501.mkv",
	"170424_26.mkv",
	"cwnOJYks5E2WP7zGuzPkdkRK3JkWw0.mkv",
	"46XuzmawYQeUBwNdH2Hw2996.mkv",
	"-_. []",
}

func TestObfuscatedNamesReadNoName(t *testing.T) {
	for _, name := range obfuscated {
		if !release.Prepare(name).Obfuscated {
			t.Errorf("%q isn't obfuscated", name)
		}
		if n := release.Title(name, release.TitleOnly{}, nil).Name; n.Titles != nil {
			t.Errorf("%q reads the title %q, want none", name, n.Titles)
		}
	}
	name := "Show.Title.S02E12.1080p.BluRay.x264-SONARR"
	if release.Prepare(name).Obfuscated || release.Title(name, release.TitleOnly{}, nil).Name.Titles == nil {
		t.Errorf("%q reads as obfuscated", name)
	}
}
