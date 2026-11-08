// Ported from Prowlarr's src/NzbDrone.Core/Parser/TvCategoryFromQualityParser.cs
// at ef9c182466ed04f34e6bcfedebb02c3c4af7eaf9. Copyright the Prowlarr
// contributors, GPL-3.0. Modified for SlipStream: rewritten in Go.

package nebulance

import (
	"regexp"
	"strings"

	"github.com/jatassi/slipstream/internal/newznab"
)

var (
	sourceRegex = regexp.MustCompile(`(?i)\b(?:` +
		`(?P<bluray>BluRay|Blu-Ray|HDDVD|BD)|` +
		`(?P<webdl>WEB[-_. ]DL|WEBDL|WebRip|iTunesHD|WebHD)|` +
		`(?P<hdtv>HDTV)|` +
		`(?P<bdrip>BDRip)|` +
		`(?P<brrip>BRRip)|` +
		`(?P<dvd>DVD|DVDRip|NTSC|PAL|xvidvd)|` +
		`(?P<dsr>WS[-_. ]DSR|DSR)|` +
		`(?P<pdtv>PDTV)|` +
		`(?P<sdtv>SDTV)|` +
		`(?P<tvrip>TVRip)` +
		`)\b`)

	rawHdRegex = regexp.MustCompile(`(?i)\b(?P<rawhd>TrollHD|RawHD|1080i[-_. ]HDTV|Raw[-_. ]HD|MPEG[-_. ]?2)\b`)

	resolutionRegex = regexp.MustCompile(`(?i)\b(?:(?P<q480p>480p|640x480|848x480)|(?P<q576p>576p)|(?P<q720p>720p|1280x720)|(?P<q1080p>1080p|1920x1080)|(?P<q2160p>2160p))\b`)

	codecRegex = regexp.MustCompile(`(?i)\b(?:(?P<x264>x264)|(?P<h264>h264)|(?P<xvidhd>XvidHD)|(?P<xvid>Xvid)|(?P<divx>divx))\b`)

	highDefPdtvRegex = regexp.MustCompile(`(?i)hr[-_. ]ws`)
)

// groups is Regex.Match's Groups: the named groups of the leftmost match
// that took part in it.
func groups(re *regexp.Regexp, s string) map[string]bool {
	out := map[string]bool{}
	m := re.FindStringSubmatchIndex(s)
	for i, name := range re.SubexpNames() {
		if name != "" && m != nil && m[2*i] >= 0 {
			out[name] = true
		}
	}
	return out
}

func parseTvShowQuality(tvShowFileName string) newznab.Category {
	normalizedName := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(tvShowFileName), "_", " ")))

	sourceMatch := groups(sourceRegex, normalizedName)
	resolutionMatch := groups(resolutionRegex, normalizedName)
	codecMatch := groups(codecRegex, normalizedName)

	if sourceMatch["webdl"] {
		if resolutionMatch["q2160p"] {
			return newznab.TVUHD
		}

		if resolutionMatch["q1080p"] || resolutionMatch["q720p"] {
			return newznab.TVHD
		}

		if resolutionMatch["q480p"] {
			return newznab.TVSD
		}
	}

	if sourceMatch["hdtv"] {
		if resolutionMatch["q2160p"] {
			return newznab.TVUHD
		}

		if resolutionMatch["q1080p"] || resolutionMatch["q720p"] {
			return newznab.TVHD
		}
		return newznab.TVSD
	}

	if sourceMatch["bluray"] || sourceMatch["bdrip"] || sourceMatch["brrip"] {
		if codecMatch["xvid"] || codecMatch["divx"] {
			return newznab.TVSD
		}

		if resolutionMatch["q2160p"] {
			return newznab.TVUHD
		}

		if resolutionMatch["q1080p"] || resolutionMatch["q720p"] {
			return newznab.TVHD
		}

		if resolutionMatch["q480p"] || resolutionMatch["q576p"] {
			return newznab.TVSD
		}
	}

	if sourceMatch["dvd"] {
		return newznab.TVSD
	}

	if sourceMatch["pdtv"] || sourceMatch["sdtv"] || sourceMatch["dsr"] || sourceMatch["tvrip"] {
		if highDefPdtvRegex.MatchString(normalizedName) {
			return newznab.TVHD
		}
		return newznab.TVSD
	}

	if rawHdRegex.MatchString(normalizedName) {
		return newznab.TVHD
	}

	return newznab.TV
}
