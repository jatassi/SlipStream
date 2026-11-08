package formats

import (
	"strconv"

	"github.com/jatassi/slipstream/internal/release"
)

// Value is one value an enum kind's value field takes in a dialect.
type Value struct {
	// Key is the value as the field holds it: upstream's number in
	// Radarr's and Sonarr's dialects, SlipStream's name in its own.
	Key   string
	Words string
}

// Offered is one kind as a dialect offers it.
type Offered struct {
	KindInfo
	// Values is what its value field takes, nil for a kind whose value
	// is a pattern or a number.
	Values []Value
}

// KindsIn lists the kinds d offers, in upstream's order, each with the
// values its value field takes there. A kind only the other app has is
// left out of Radarr's and Sonarr's.
func KindsIn(d Dialect) []Offered {
	var out []Offered
	for _, k := range kinds {
		if app, only := appOnly[k.Kind]; only && d != SlipStream && app != d {
			continue
		}
		out = append(out, Offered{KindInfo: k.KindInfo, Values: valuesIn(d, k.Kind)})
	}
	return out
}

var upstreamEnums = map[Dialect]map[Kind][]upstreamValue{
	Radarr: {
		Source: radarrQualitySource, Resolution: radarrResolution, QualityModifier: radarrModifier,
		Language: radarrLanguage, IndexerFlag: radarrIndexerFlags,
	},
	Sonarr: {
		Source: sonarrQualitySource, Resolution: sonarrResolution, ReleaseType: sonarrReleaseType,
		Language: sonarrLanguage, IndexerFlag: sonarrIndexerFlags,
	},
}

func valuesIn(d Dialect, k Kind) []Value {
	if d == SlipStream {
		return slipstreamValues[k]
	}
	var out []Value
	for _, v := range upstreamEnums[d][k] {
		out = append(out, Value{Key: strconv.Itoa(v.Value), Words: v.Name})
	}
	return out
}

var slipstreamValues = map[Kind][]Value{
	Source: words(release.Origin.String, []release.Origin{
		release.OriginCAM, release.OriginTelesync, release.OriginTelecine, release.OriginWorkprint, release.OriginDVD,
		release.OriginTV, release.OriginWEBDL, release.OriginWEBRip, release.OriginBluray,
	}, []string{"CAM", "Telesync", "Telecine", "Workprint", "DVD", "TV", "WEB-DL", "WEBRip", "Bluray"}),
	Resolution: words(release.Resolution.String, []release.Resolution{
		release.R360p, release.R480p, release.R540p, release.R576p, release.R720p, release.R1080p, release.R2160p,
	}, []string{"360p", "480p", "540p", "576p", "720p", "1080p", "2160p"}),
	QualityModifier: words(release.Modifier.String, []release.Modifier{
		release.ModRegional, release.ModScreener, release.ModRawHD, release.ModBRDisk, release.ModRemux,
	}, []string{"Regional", "Screener", "Raw-HD", "BR-DISK", "Remux"}),
	IndexerFlag: words(func(f release.Flag) string { return string(f) }, release.Flags(), []string{
		"Freeleech", "Halfleech", "Freeleech 75%", "Freeleech 25%", "Double upload", "Internal", "Scene", "Exclusive", "Nuked", "Golden", "Approved",
	}),
	ReleaseType: words(release.Span.String, []release.Span{release.One, release.Several, release.Level, release.Levels},
		[]string{"One Item", "Several Items", "A whole level", "Several whole levels"}),
	Language: slipstreamLanguages(),
}

func words[T any](key func(T) string, vs []T, ws []string) []Value {
	if len(vs) != len(ws) {
		panic("formats: a SlipStream value has no words")
	}
	out := make([]Value, len(vs))
	for i, v := range vs {
		out[i] = Value{Key: key(v), Words: ws[i]}
	}
	return out
}

// slipstreamLanguages are the languages Radarr and Sonarr name, by
// Radarr's word for each, then Sonarr's.
func slipstreamLanguages() []Value {
	var out []Value
	seen := map[release.Language]bool{}
	for _, v := range append(append([]upstreamValue{}, radarrLanguage...), sonarrLanguage...) {
		l, ok := release.UpstreamLanguage(v.Name)
		if !ok || seen[l] {
			continue
		}
		seen[l] = true
		out = append(out, Value{Key: string(l), Words: v.Name})
	}
	return out
}
