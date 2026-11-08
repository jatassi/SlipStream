package formats

import (
	"maps"
	"strconv"

	"github.com/jatassi/slipstream/internal/release"
)

// sourceIs is what a Source condition tests: the fine source, zero for
// unknown, and for Sonarr's composite values the modifier its raw value
// carries and its plain value lacks.
type sourceIs struct {
	origin   release.Origin
	modifier release.Modifier
	raw      bool
}

// radarrSources read the fine source alone, so Bluray matches a Remux.
var radarrSources = map[string]sourceIs{
	"UNKNOWN":   {},
	"CAM":       {origin: release.OriginCAM},
	"TELESYNC":  {origin: release.OriginTelesync},
	"TELECINE":  {origin: release.OriginTelecine},
	"WORKPRINT": {origin: release.OriginWorkprint},
	"DVD":       {origin: release.OriginDVD},
	"TV":        {origin: release.OriginTV},
	"WEBDL":     {origin: release.OriginWEBDL},
	"WEBRIP":    {origin: release.OriginWEBRip},
	"BLURAY":    {origin: release.OriginBluray},
}

// sonarrSources split TV and Bluray as Sonarr's qualities do: Raw-HD
// carries TelevisionRaw and the Remux qualities carry BlurayRaw.
var sonarrSources = map[string]sourceIs{
	"Unknown":       {},
	"Television":    {origin: release.OriginTV, modifier: release.ModRawHD},
	"TelevisionRaw": {origin: release.OriginTV, modifier: release.ModRawHD, raw: true},
	"Web":           {origin: release.OriginWEBDL},
	"WebRip":        {origin: release.OriginWEBRip},
	"DVD":           {origin: release.OriginDVD},
	"Bluray":        {origin: release.OriginBluray, modifier: release.ModRemux},
	"BlurayRaw":     {origin: release.OriginBluray, modifier: release.ModRemux, raw: true},
}

var resolutions = map[string]release.Resolution{
	"Unknown": 0,
	"R360p":   release.R360p,
	"R480p":   release.R480p,
	"R540p":   release.R540p,
	"R576p":   release.R576p,
	"R720p":   release.R720p,
	"R1080p":  release.R1080p,
	"R2160p":  release.R2160p,
}

var radarrModifiers = map[string]release.Modifier{
	"NONE":     0,
	"REGIONAL": release.ModRegional,
	"SCREENER": release.ModScreener,
	"RAWHD":    release.ModRawHD,
	"BRDISK":   release.ModBRDisk,
	"REMUX":    release.ModRemux,
}

var sonarrSpans = map[string]release.Span{
	"Unknown":         release.SpanUnknown,
	"SingleEpisode":   release.One,
	"MultiEpisode":    release.Several,
	"SeasonPack":      release.Level,
	"MultiSeasonPack": release.Levels,
}

// The flags a Release never carries map to "": Radarr's AwesomeHD flags
// came from an indexer it dropped, and no Indexer reports Subtitles.
var radarrFlags = map[string]release.Flag{
	"G_Freeleech":     release.Freeleech,
	"G_Halfleech":     release.Halfleech,
	"G_DoubleUpload":  release.DoubleUpload,
	"PTP_Golden":      release.Golden,
	"PTP_Approved":    release.Approved,
	"G_Internal":      release.Internal,
	"AHD_Internal":    "",
	"G_Scene":         release.Scene,
	"G_Freeleech75":   release.Freeleech75,
	"G_Freeleech25":   release.Freeleech25,
	"AHD_UserRelease": "",
	"Nuked":           release.Nuked,
}

var sonarrFlags = map[string]release.Flag{
	"Freeleech":    release.Freeleech,
	"Halfleech":    release.Halfleech,
	"DoubleUpload": release.DoubleUpload,
	"Internal":     release.Internal,
	"Scene":        release.Scene,
	"Freeleech75":  release.Freeleech75,
	"Freeleech25":  release.Freeleech25,
	"Nuked":        release.Nuked,
	"Subtitles":    "",
}

const (
	// unknownLanguage is upstream's Unknown: what a name with no language
	// tag reads as when the Work's original language is unknown too.
	unknownLanguage release.Language = ""
	// anyLanguage is Radarr's Any, which no name reads as.
	anyLanguage release.Language = "*"
)

var languagesWithoutISO = map[string]release.Language{
	"Unknown": unknownLanguage,
	"Any":     anyLanguage,
}

// dialect is one app's numbering of each kind's values, keyed by the
// value as its JSON reads.
type dialect struct {
	name        Dialect
	sources     map[string]sourceIs
	resolutions map[string]release.Resolution
	modifiers   map[string]release.Modifier
	languages   map[string]release.Language
	flags       map[string]release.Flag
	spans       map[string]release.Span
}

var dialects = map[Dialect]*dialect{
	Radarr: {
		name:        Radarr,
		sources:     numbered(radarrQualitySource, radarrSources),
		resolutions: numbered(radarrResolution, resolutions),
		modifiers:   numbered(radarrModifier, radarrModifiers),
		languages:   upstreamLanguages(radarrLanguage),
		flags:       numbered(radarrIndexerFlags, radarrFlags),
	},
	Sonarr: {
		name:        Sonarr,
		sources:     numbered(sonarrQualitySource, sonarrSources),
		resolutions: numbered(sonarrResolution, resolutions),
		languages:   upstreamLanguages(sonarrLanguage),
		flags:       numbered(sonarrIndexerFlags, sonarrFlags),
		spans:       numbered(sonarrReleaseType, sonarrSpans),
	},
	SlipStream: {
		name:        SlipStream,
		sources:     named(release.OriginCAM, release.OriginBluray, func(o release.Origin) sourceIs { return sourceIs{origin: o} }),
		resolutions: named(release.R2160p, release.R360p, same[release.Resolution]),
		modifiers:   named(release.ModRemux, release.ModScreener, same[release.Modifier]),
		flags:       flagsByName(),
		spans:       named(release.One, release.Levels, same[release.Span]),
	},
}

// numbered keys an upstream enum's readings by number. A name the hand
// table lacks stays unreadable, which TestEveryUpstreamValueHasAReading
// catches.
func numbered[T any](enum []upstreamValue, byName map[string]T) map[string]T {
	m := make(map[string]T, len(enum))
	for _, v := range enum {
		if r, ok := byName[v.Name]; ok {
			m[strconv.Itoa(v.Value)] = r
		}
	}
	return m
}

// upstreamLanguages keys an upstream Language enum's languages by
// number, as release.UpstreamLanguage reads each name.
func upstreamLanguages(enum []upstreamValue) map[string]release.Language {
	byName := maps.Clone(languagesWithoutISO)
	for _, v := range enum {
		if l, ok := release.UpstreamLanguage(v.Name); ok {
			byName[v.Name] = l
		}
	}
	return numbered(enum, byName)
}

type key interface {
	~uint8
	String() string
}

func named[K key, T any](first, last K, read func(K) T) map[string]T {
	m := map[string]T{}
	for k := first; k <= last; k++ {
		m[k.String()] = read(k)
	}
	return m
}

func same[K any](k K) K { return k }

func flagsByName() map[string]release.Flag {
	m := map[string]release.Flag{}
	for _, f := range release.Flags() {
		m[string(f)] = f
	}
	return m
}

// language is the language a key names: the dialect's number, or for
// SlipStream an ISO 639 code or "original".
func (d *dialect) language(key string) (release.Language, bool) {
	if d.name == SlipStream {
		l, err := release.ParseLanguage(key)
		return l, err == nil
	}
	l, ok := d.languages[key]
	return l, ok
}
