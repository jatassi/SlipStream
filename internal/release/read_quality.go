package release

import "strings"

// sourceWords are the phrases that name each source, the first source
// named winning, so a cinema source wins over anything else in a title.
// HDTV and SDTV are both the TV origin: the resolution tells them apart.
var sourceWords = []struct {
	source   Source
	origin   Origin
	modifier Modifier
	words    []string
}{
	{Cinema, OriginCAM, 0, []string{"cam", "hdcam"}},
	{Cinema, OriginTelesync, 0, []string{"ts", "hdts", "telesync"}},
	{Remux, OriginBluray, ModRemux, []string{"remux", "bdremux"}},
	{Bluray, OriginBluray, 0, []string{"bluray", "blu ray", "bdrip", "brrip"}},
	{WEBRip, OriginWEBRip, 0, []string{"webrip", "web rip"}},
	{WEBDL, OriginWEBDL, 0, []string{"webdl", "web dl", "web"}},
	{HDTV, OriginTV, 0, []string{"hdtv"}},
	{TV, OriginTV, 0, []string{"sdtv", "tvrip"}},
	{DVD, OriginDVD, 0, []string{"dvd", "dvdrip"}},
}

var resolutionWords = map[string]Resolution{
	"2160p": R2160p, "4k": R2160p, "uhd": R2160p,
	"1080p": R1080p, "720p": R720p, "576p": R576p, "480p": R480p,
}

// readQuality sets f's Quality from the source and the first resolution
// p names, and the Source, Resolution and Modifier they are.
func readQuality(p Prepared, f *Facts) {
	w := words(p.Original)
	source := Fact[Source]{}
	for _, s := range sourceWords {
		if named(w, s.words) {
			source = Known(s.source)
			f.Source = Known(s.origin)
			if s.modifier != 0 {
				f.Modifier = Known(s.modifier)
			}
			break
		}
	}
	f.Resolution = resolutionIn(w)
	f.Quality = qualityFact(source, f.Resolution)
}

// qualityAt is the quality of origin with modifier at res.
func qualityAt(origin Fact[Origin], modifier Fact[Modifier], res Resolution) Fact[Quality] {
	source := Fact[Source]{}
	if o, ok := origin.Get(); ok {
		source = Known(coarse(o, modifier))
	}
	return qualityFact(source, Known(res))
}

// coarse is the quality table's source of an origin with modifier.
func coarse(o Origin, modifier Fact[Modifier]) Source {
	switch o {
	case OriginCAM, OriginTelesync, OriginTelecine, OriginWorkprint:
		return Cinema
	case OriginDVD:
		return DVD
	case OriginTV:
		return HDTV
	case OriginWEBDL:
		return WEBDL
	case OriginWEBRip:
		return WEBRip
	case OriginBluray:
	}
	if m, _ := modifier.Get(); m == ModRemux {
		return Remux
	}
	return Bluray
}

func qualityFact(s Fact[Source], r Fact[Resolution]) Fact[Quality] {
	if q, ok := QualityOf(s, r); ok {
		return Known(q)
	}
	return Fact[Quality]{}
}

func named(w string, phrases []string) bool {
	for _, p := range phrases {
		if strings.Contains(w, phrase(p)) {
			return true
		}
	}
	return false
}

func resolutionIn(w string) Fact[Resolution] {
	for _, word := range strings.Fields(w) {
		if r, ok := resolutionWords[word]; ok {
			return Known(r)
		}
	}
	return Fact[Resolution]{}
}
