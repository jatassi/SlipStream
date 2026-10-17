package release

import "fmt"

// Source is where a release's video came from.
type Source uint8

const (
	TV Source = iota + 1 // standard definition: SDTV, TVRip
	HDTV
	DVD
	WEBRip
	WEBDL
	Bluray
	Remux
	Cinema // CAM, HDCAM, TS, HDTS, TELESYNC
)

// Quality is one of the 18 fixed qualities. Its value is its place in the
// table, which is only the presets' default order: a Target's tiers rank.
type Quality uint8

const (
	SDTV Quality = iota + 1
	DVDQuality
	WEBRip480p
	HDTV720p
	WEBRip720p
	WEBDL720p
	Bluray720p
	HDTV1080p
	WEBRip1080p
	WEBDL1080p
	Bluray1080p
	Remux1080p
	HDTV2160p
	WEBRip2160p
	WEBDL2160p
	Bluray2160p
	Remux2160p
	CAM
)

// QualityInfo is one row of the quality table.
type QualityInfo struct {
	Key    string // stable on the wire and in the database
	Name   string
	Source Source
	// Resolution is R480p for the SD qualities and zero for CAM, which a
	// resolution never changes.
	Resolution Resolution
	// Floor is the minimum size a new Target starts with, in MiB per minute.
	Floor float64
}

func (i QualityInfo) Cinema() bool { return i.Source == Cinema }

// qualities is the one table, low to high. Floors are TRaSH Guides' movie
// minimums (docs/json/radarr/quality-size/movie.json in TRaSH-Guides/Guides),
// 0 where TRaSH lists none.
var qualities = [...]QualityInfo{
	SDTV - 1:        {"sdtv", "SDTV", TV, R480p, 0},
	DVDQuality - 1:  {"dvd", "DVD", DVD, R480p, 0},
	WEBRip480p - 1:  {"webrip-480p", "WEBRip-480p", WEBRip, R480p, 0},
	HDTV720p - 1:    {"hdtv-720p", "HDTV-720p", HDTV, R720p, 17.1},
	WEBRip720p - 1:  {"webrip-720p", "WEBRip-720p", WEBRip, R720p, 12.5},
	WEBDL720p - 1:   {"webdl-720p", "WEBDL-720p", WEBDL, R720p, 12.5},
	Bluray720p - 1:  {"bluray-720p", "Bluray-720p", Bluray, R720p, 25.7},
	HDTV1080p - 1:   {"hdtv-1080p", "HDTV-1080p", HDTV, R1080p, 33.8},
	WEBRip1080p - 1: {"webrip-1080p", "WEBRip-1080p", WEBRip, R1080p, 12.5},
	WEBDL1080p - 1:  {"webdl-1080p", "WEBDL-1080p", WEBDL, R1080p, 12.5},
	Bluray1080p - 1: {"bluray-1080p", "Bluray-1080p", Bluray, R1080p, 50.8},
	Remux1080p - 1:  {"remux-1080p", "Remux-1080p", Remux, R1080p, 102},
	HDTV2160p - 1:   {"hdtv-2160p", "HDTV-2160p", HDTV, R2160p, 85},
	WEBRip2160p - 1: {"webrip-2160p", "WEBRip-2160p", WEBRip, R2160p, 34.5},
	WEBDL2160p - 1:  {"webdl-2160p", "WEBDL-2160p", WEBDL, R2160p, 34.5},
	Bluray2160p - 1: {"bluray-2160p", "Bluray-2160p", Bluray, R2160p, 102},
	Remux2160p - 1:  {"remux-2160p", "Remux-2160p", Remux, R2160p, 187.4},
	CAM - 1:         {"cam", "CAM", Cinema, 0, 0},
}

// Qualities is every quality in the table's order.
func Qualities() []Quality {
	all := make([]Quality, len(qualities))
	for i := range qualities {
		all[i] = Quality(i + 1)
	}
	return all
}

func (q Quality) Info() QualityInfo { return qualities[q-1] }

func (q Quality) String() string { return q.Info().Key }

func ParseQuality(key string) (Quality, error) {
	for _, q := range Qualities() {
		if q.Info().Key == key {
			return q, nil
		}
	}
	return 0, fmt.Errorf("%q is not a quality", key)
}

// QualityOf is the quality of a source at a resolution. A cinema source
// wins over any resolution. A missing source is the lowest quality at the
// resolution, and a missing resolution the source's lowest. A pair the
// table lacks is the lowest quality at the resolution, so a fact never
// ranks above what it is. Neither known is false.
func QualityOf(s Fact[Source], r Fact[Resolution]) (Quality, bool) {
	source, sourceKnown := s.Get()
	res, resKnown := r.Get()
	if res == R576p {
		res = R480p
	}
	first := func(match func(QualityInfo) bool) (Quality, bool) {
		for _, q := range Qualities() {
			if match(q.Info()) {
				return q, true
			}
		}
		return 0, false
	}
	switch {
	case sourceKnown && source == Cinema:
		return CAM, true
	case sourceKnown && resKnown:
		if q, ok := first(func(i QualityInfo) bool { return i.Source == source && i.Resolution == res }); ok {
			return q, true
		}
		return first(func(i QualityInfo) bool { return i.Resolution == res })
	case resKnown:
		return first(func(i QualityInfo) bool { return i.Resolution == res })
	case sourceKnown:
		return first(func(i QualityInfo) bool { return i.Source == source })
	}
	return 0, false
}
