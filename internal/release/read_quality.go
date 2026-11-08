package release

import (
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release/internal/upstream"
)

// The quality patterns are Radarr's, which read every source Sonarr's do
// and the cinema, screener and regional ones besides. Sonarr's source and
// resolution patterns read a name Radarr's find nothing in, for the words
// only Sonarr knows: NTSC, PAL, ATVP, H266.
var (
	sourceRegex       = compile(upstream.RadarrQualityParserSourceRegex)
	sonarrSource      = compile(upstream.SonarrQualityParserSourceRegex)
	rawHDRegex        = compile(upstream.RadarrQualityParserRawHDRegex)
	mpeg2Regex        = compile(upstream.RadarrQualityParserMPEG2Regex)
	brDiskRegex       = compile(upstream.RadarrQualityParserBRDISKRegex)
	resolutionRegex   = compile(upstream.RadarrQualityParserResolutionRegex)
	sonarrResolution  = compile(upstream.SonarrQualityParserResolutionRegex)
	alternativeRegex  = compile(upstream.RadarrQualityParserAlternativeResolutionRegex)
	codecRegex        = compile(upstream.RadarrQualityParserCodecRegex)
	otherSourceRegex  = compile(upstream.RadarrQualityParserOtherSourceRegex)
	animeBlurayRegex  = compile(upstream.RadarrQualityParserAnimeBlurayRegex)
	animeWebDlRegex   = compile(upstream.RadarrQualityParserAnimeWebDlRegex)
	highDefPdtvRegex  = compile(upstream.RadarrQualityParserHighDefPdtvRegex)
	remuxRegex        = compile(upstream.RadarrQualityParserRemuxRegex)
	germanRemuxRegex  = compile(upstream.RadarrQualityParserGermanRemuxRegex)
	sourceGroups      = []string{"bluray", "webdl", "webrip", "hdtv", "bdrip", "brrip", "dvdr", "dvd", "dsr", "regional", "scr", "ts", "tc", "cam", "wp", "pdtv", "sdtv", "tvrip"}
	cinemaGroups      = []string{"ts", "tc", "cam", "wp"}
	codecGroups       = []string{"x264", "h264", "xvidhd", "xvid", "divx"}
	otherSourceGroups = []string{"hdtv", "sdtv"}
)

// resolutionGroups are ParseResolution's groups in the order it tries
// them.
var resolutionGroups = []struct {
	group string
	res   Resolution
}{
	{"R360p", R360p}, {"R480p", R480p}, {"R540p", R540p}, {"R576p", R576p},
	{"R720p", R720p}, {"R1080p", R1080p}, {"R2160p", R2160p},
}

// grade is one of Sonarr's or Radarr's qualities as fine facts. A zero
// res is a quality of no resolution, such as CAM. DVD is Sonarr's, at
// 480p, where Radarr's has none, so a DVD's named resolution is known.
type grade struct {
	origin   Origin
	modifier Modifier
	res      Resolution
}

func (g grade) at(r Resolution) grade { g.res = r; return g }

var (
	sdtv      = grade{OriginTV, 0, R480p}
	dvd       = grade{OriginDVD, 0, R480p}
	dvdr      = grade{OriginDVD, ModRemux, R480p}
	dvdscr    = grade{OriginDVD, ModScreener, R480p}
	regional  = grade{OriginDVD, ModRegional, R480p}
	workprint = grade{OriginWorkprint, 0, 0}
	cam       = grade{OriginCAM, 0, 0}
	telesync  = grade{OriginTelesync, 0, 0}
	telecine  = grade{OriginTelecine, 0, 0}
	hdtv      = grade{OriginTV, 0, 0}
	rawHD     = grade{OriginTV, ModRawHD, R1080p}
	webdl     = grade{OriginWEBDL, 0, 0}
	webrip    = grade{OriginWEBRip, 0, 0}
	bluray    = grade{OriginBluray, 0, 0}
	remux     = grade{OriginBluray, ModRemux, 0}
	brDisk    = grade{OriginBluray, ModBRDisk, R1080p}
)

// mediaExtensions is Sonarr's MediaFileExtensions: the quality a file's
// extension says. Radarr reads .mkv as WEBDL-720p and .ts and .wtv as
// SDTV, and SlipStream reads them as Sonarr does.
var mediaExtensions = func() map[string]grade {
	m := map[string]grade{}
	for _, e := range []string{
		".m4v", ".3gp", ".nsv", ".ty", ".strm", ".rm", ".rmvb", ".m3u", ".ifo", ".mov", ".qt", ".divx", ".xvid",
		".bivx", ".nrg", ".pva", ".wmv", ".asf", ".asx", ".ogm", ".ogv", ".m2v", ".avi", ".bin", ".dat", ".dvr-ms",
		".mpg", ".mpeg", ".mp4", ".avc", ".vp3", ".svq3", ".nuv", ".viv", ".dv", ".fli", ".flv", ".wpl",
	} {
		m[e] = sdtv
	}
	for _, e := range []string{".img", ".iso", ".vob"} {
		m[e] = dvd
	}
	for _, e := range []string{".mkv", ".ts", ".wtv"} {
		m[e] = hdtv.at(R720p)
	}
	m[".m2ts"] = bluray.at(R720p)
	return m
}()

// named is what ParseQuality reads: a quality, and whether the name or
// its extension named its source and its resolution, upstream's detection
// sources.
type named struct {
	grade
	source, resolution bool
}

// readQuality sets f's Source, Modifier and Resolution to what the name
// names, and its Quality by S17 (Targets, Profiles and Editions)'s rules.
func readQuality(p Prepared, s *scan, f *Facts) {
	n := parseQuality(s, p.Original)
	if n.source {
		f.Source = Known(n.origin)
		if n.modifier != 0 {
			f.Modifier = Known(n.modifier)
		}
	}
	if n.resolution && n.res != 0 {
		f.Resolution = Known(n.res)
	}
	f.Quality = qualityOfFacts(f.Source, f.Modifier, f.Resolution)
}

// parseQuality is upstream's ParseQuality: the name's quality, or its
// extension's when the name names none.
func parseQuality(s *scan, name string) named {
	name = strings.TrimSpace(name)
	if name == "" {
		return named{}
	}
	n := parseQualityName(s, name)
	if n.grade == (grade{}) {
		if g, ok := mediaExtensions[strings.ToLower(extension(name))]; ok {
			return named{g, true, true}
		}
	}
	return n
}

// parseQualityName is Radarr's ParseQualityName, with Sonarr's source and
// resolution patterns where Radarr's read nothing. Where it departs from
// both, the comment says why.
//
//nolint:gocyclo,gocognit,cyclop,funlen // a line-for-line port of upstream's one function
func parseQualityName(s *scan, name string) named {
	norm := strings.TrimSpace(strings.ReplaceAll(name, "_", " "))
	t := regex.TextOf(norm)
	source := sourceOf(s, t)
	res := parseResolution(s, t)
	codecs := s.group(codecRegex, t, codecGroups)
	remuxed := s.match(remuxRegex, t) || s.match(germanRemuxRegex, t)
	disk := s.match(brDiskRegex, t)
	resNamed := res != 0
	// byName is a quality whose resolution is named only when the name
	// names one: upstream's default for a name without one is left
	// unknown, so S17 (Targets, Profiles and Editions) reads the source
	// alone as its lowest quality. implied is a quality whose words name
	// its resolution.
	byName := func(g grade) named { return named{g, true, resNamed} }
	implied := func(g grade) named { return named{g, true, true} }

	if s.match(rawHDRegex, t) && !disk {
		return implied(rawHD)
	}

	switch source {
	case "bluray":
		switch {
		case disk:
			return implied(brDisk)
		case codecs == "xvid" || codecs == "divx":
			return implied(bluray.at(R480p))
		case res == R2160p || res == R1080p:
			if remuxed {
				return byName(remux.at(res))
			}
			return byName(bluray.at(res))
		case res == R720p || res == R576p:
			return byName(bluray.at(res))
		case res == R360p || res == R480p || res == R540p:
			return byName(bluray.at(R480p))
		case remuxed:
			return byName(remux.at(R1080p))
		}
		return byName(bluray.at(R720p))
	case "webdl":
		switch {
		case res == R2160p || res == R1080p || res == R720p:
			return byName(webdl.at(res))
		case strings.Contains(name, "[WEBDL]"):
			return implied(webdl.at(R720p))
		}
		return byName(webdl.at(R480p))
	case "webrip":
		if res == R2160p || res == R1080p || res == R720p {
			return byName(webrip.at(res))
		}
		return byName(webrip.at(R480p))
	case "scr":
		return implied(dvdscr)
	case "cam":
		return byName(cam)
	case "ts":
		return byName(telesync.at(res))
	case "tc":
		return byName(telecine)
	case "wp":
		return byName(workprint)
	case "regional":
		return implied(regional)
	case "hdtv":
		switch {
		case s.match(mpeg2Regex, t):
			return implied(rawHD)
		case res == R2160p || res == R1080p || res == R720p:
			return byName(hdtv.at(res))
		case strings.Contains(name, "[HDTV]"):
			return implied(hdtv.at(R720p))
		}
		return byName(sdtv)
	case "bdrip", "brrip":
		if res == R720p || res == R1080p || res == R2160p || res == R576p {
			return byName(bluray.at(res))
		}
		return byName(bluray.at(R480p))
	case "dvdr":
		return implied(dvdr)
	case "dvd":
		return byName(dvd)
	case "pdtv", "sdtv", "dsr", "tvrip":
		switch {
		case res == R1080p || containsFold(norm, "1080p"):
			return byName(hdtv.at(R1080p))
		case res == R720p || containsFold(norm, "720p"):
			return byName(hdtv.at(R720p))
		case s.match(highDefPdtvRegex, t):
			return implied(hdtv.at(R720p))
		}
		// SDTV, PDTV, DSR and TVRip name standard definition, as HDTV
		// doesn't.
		return implied(sdtv)
	}

	// Upstream reads a remux without a source as Bluray-720p or Remux by
	// resolution, and as nothing without one. A remux is from a Blu-ray
	// whatever its resolution, so SlipStream reads the source and the
	// modifier, and S17 (Targets, Profiles and Editions) finds its quality.
	if source == "" && remuxed {
		return byName(remux.at(res))
	}

	if s.match(animeBlurayRegex, t) {
		switch {
		case res == R360p || res == R480p || res == R540p || res == R576p || containsFold(norm, "480p"):
			return implied(dvd)
		case res == R1080p || containsFold(norm, "1080p"):
			return implied(bluray.at(R1080p))
		case res == R2160p || containsFold(norm, "2160p"):
			return implied(bluray.at(R2160p))
		case containsFold(norm, "720p"):
			return implied(bluray.at(R720p))
		}
		return byName(bluray.at(R720p))
	}

	if s.match(animeWebDlRegex, t) {
		switch {
		case res == R360p || res == R480p || res == R540p || res == R576p || containsFold(norm, "480p"):
			return implied(webdl.at(R480p))
		case res == R1080p || containsFold(norm, "1080p"):
			return implied(webdl.at(R1080p))
		case res == R2160p || containsFold(norm, "2160p"):
			return implied(webdl.at(R2160p))
		}
		return byName(webdl.at(R720p))
	}

	if resNamed {
		g, ok := mediaExtensions[strings.ToLower(pathExtension(name))]
		if !ok {
			return named{hdtv.at(res480(res)), false, true}
		}
		return named{grade{g.origin, 0, res480(res)}, true, true}
	}

	// Upstream's SDTV for x264 alone names neither its source nor its
	// resolution, and keeps the extension from naming them.
	if codecs == "x264" {
		return named{grade: sdtv}
	}

	for _, size := range []struct {
		frame  string
		bluray grade
		other  grade
	}{
		{"848x480", bluray.at(R480p), sdtv},
		{"1280x720", bluray.at(R720p), hdtv.at(R720p)},
		{"1920x1080", bluray.at(R1080p), hdtv.at(R1080p)},
	} {
		if !containsFold(norm, size.frame) {
			continue
		}
		switch {
		case size.frame == "848x480" && strings.Contains(norm, "dvd"):
			return implied(dvd)
		case containsFold(norm, "bluray"):
			return implied(size.bluray)
		}
		return named{size.other, false, true}
	}

	for _, word := range []struct {
		word string
		res  Resolution
	}{{"bluray720p", R720p}, {"bluray1080p", R1080p}, {"bluray2160p", R2160p}} {
		if containsFold(norm, word.word) {
			return implied(bluray.at(word.res))
		}
	}

	switch s.group(otherSourceRegex, t, otherSourceGroups) {
	case "sdtv":
		return implied(sdtv)
	case "hdtv":
		return implied(hdtv.at(R720p))
	}
	return named{}
}

// sourceOf is the source group SourceRegex names last, Sonarr's when
// Radarr's names none. S17 (Targets, Profiles and Editions) has a cinema
// source win over any other, so the last cinema source wins over a later
// source of another kind.
func sourceOf(s *scan, t regex.Text) string {
	all := s.findAll(sourceRegex, t)
	if len(all) == 0 {
		all = s.findAll(sonarrSource, t)
	}
	last := ""
	for _, m := range slices.Backward(all) {
		g := firstGroup(m, sourceGroups)
		if slices.Contains(cinemaGroups, g) {
			return g
		}
		if last == "" {
			last = g
		}
	}
	return last
}

// parseResolution is upstream's ParseResolution, 0 for none.
func parseResolution(s *scan, t regex.Text) Resolution {
	m, ok := s.find(resolutionRegex, t)
	if !ok {
		m, ok = s.find(sonarrResolution, t)
	}
	if ok {
		for _, g := range resolutionGroups {
			if _, ok := m.Group(g.group); ok {
				return g.res
			}
		}
	}
	if s.match(alternativeRegex, t) {
		return R2160p
	}
	return 0
}

// res480 is QualityFinder's reading of a resolution below 720p as 480p.
func res480(r Resolution) Resolution {
	if r == R360p || r == R540p || r == R576p {
		return R480p
	}
	return r
}

// extension is .NET's Path.GetExtension, reading both separators: the
// last dot and what follows it in the last path element, or "".
func extension(name string) string {
	base := name[strings.LastIndexAny(name, `/\`)+1:]
	i := strings.LastIndexByte(base, '.')
	if i < 0 || i == len(base)-1 {
		return ""
	}
	return base[i:]
}

// pathExtension is NzbDrone's GetPathExtension: from the last dot
// anywhere in the name.
func pathExtension(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i < 0 || i == len(name)-1 {
		return ""
	}
	return name[i:]
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

// qualityOfFacts is the quality of origin with modifier at res.
func qualityOfFacts(origin Fact[Origin], modifier Fact[Modifier], res Fact[Resolution]) Fact[Quality] {
	source := Fact[Source]{}
	if o, ok := origin.Get(); ok {
		source = Known(coarse(o, modifier))
	}
	if q, ok := QualityOf(source, res); ok {
		return Known(q)
	}
	return Fact[Quality]{}
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
