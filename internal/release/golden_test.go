package release_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/release/golden"
)

func titleFacts(name string) release.Facts {
	return release.Title(name, release.TitleOnly{}, release.BuiltIn()).Facts
}

func version(f release.Facts, want int) error {
	return golden.Same("version", max(f.Revision.Version, 1), want)
}

func proper(r golden.Row) int {
	if r.Bool("proper") {
		return 2
	}
	return 1
}

func sonarrQuality(name string) golden.Check {
	return func(r golden.Row) error {
		f := titleFacts(r.String("title"))
		return errors.Join(golden.SonarrQuality(name).Read(f), version(f, proper(r)))
	}
}

var radarrSources = map[string]release.Fact[release.Origin]{
	"UNKNOWN":   {},
	"CAM":       release.Known(release.OriginCAM),
	"TELESYNC":  release.Known(release.OriginTelesync),
	"TELECINE":  release.Known(release.OriginTelecine),
	"WORKPRINT": release.Known(release.OriginWorkprint),
	"DVD":       release.Known(release.OriginDVD),
	"TV":        release.Known(release.OriginTV),
	"WEBDL":     release.Known(release.OriginWEBDL),
	"WEBRIP":    release.Known(release.OriginWEBRip),
	"BLURAY":    release.Known(release.OriginBluray),
}

var radarrResolutions = map[string]release.Fact[release.Resolution]{
	"Unknown": {},
	"R480p":   release.Known(release.R480p),
	"R576p":   release.Known(release.R576p),
	"R720p":   release.Known(release.R720p),
	"R1080p":  release.Known(release.R1080p),
	"R2160p":  release.Known(release.R2160p),
}

var radarrModifiers = map[string]release.Modifier{
	"REMUX":  release.ModRemux,
	"BRDISK": release.ModBRDisk,
	"RAWHD":  release.ModRawHD,
}

// radarrReads is Radarr's ParseAndVerifyQuality: an Unknown resolution
// and a NONE modifier aren't checked.
func radarrReads(f release.Facts, source, resolution, modifier string, v int) error {
	errs := []error{golden.Same("source", f.Source, radarrSources[source]), version(f, v)}
	if res := radarrResolutions[resolution]; res != (release.Fact[release.Resolution]{}) {
		errs = append(errs, golden.Same("resolution", f.Resolution, res))
	}
	if m, ok := radarrModifiers[modifier]; ok {
		errs = append(errs, golden.Same("modifier", f.Modifier, release.Known(m)))
	}
	return errors.Join(errs...)
}

func radarrQuality(source, resolution, modifier string) golden.Check {
	return func(r golden.Row) error {
		return radarrReads(titleFacts(r.String("title")), source, resolution, modifier, proper(r))
	}
}

// radarrBluray is radarrQuality for Radarr's Bluray methods, which have no
// proper parameter.
func radarrBluray(resolution, modifier string) golden.Check {
	return func(r golden.Row) error {
		return radarrReads(titleFacts(r.String("title")), "BLURAY", resolution, modifier, 1)
	}
}

// resolutionKnown is a detection-source method in SlipStream's terms: the
// resolution is read, and the source is read when source is true.
func resolutionKnown(source bool) golden.Check {
	return func(r golden.Row) error {
		f := titleFacts(r.String("title"))
		_, s := f.Source.Get()
		_, res := f.Resolution.Get()
		return errors.Join(golden.Same("source known", s, source), golden.Same("resolution known", res, true))
	}
}

func sourceKnown(r golden.Row) error {
	_, ok := titleFacts(r.String("title")).Source.Get()
	return golden.Same("source known", ok, true)
}

func repack(r golden.Row) error {
	f := titleFacts(r.String("title"))
	return errors.Join(version(f, r.Int("version")), golden.Same("repack", f.Revision.Repack, r.Bool("isRepack")))
}

func readVersion(r golden.Row) error {
	return version(titleFacts(r.String("title")), r.Int("version"))
}

// separators are the ones Sonarr's and Radarr's other-source tests join
// a quality's words with.
var separators = []string{"-", ".", " ", "_"}

func otherSource(r golden.Row, read func(release.Facts) error) error {
	var errs []error
	for _, sep := range separators {
		name := "My series S01E01 " + strings.ReplaceAll(r.String("qualityString"), " ", sep)
		if err := read(titleFacts(name)); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

func enumValue(ref string) string {
	_, v, _ := strings.Cut(ref, ".")
	return v
}

func group(r golden.Row) error {
	want := release.Fact[string]{}
	if g := r.String("expected"); g != "" {
		want = release.Known(g)
	}
	return golden.Same("group", titleFacts(r.String("title")).Group, want)
}

func noGroup(r golden.Row) error {
	want := release.Fact[string]{}
	var read string
	if r.Reads(&read) {
		want = release.Known(read)
	}
	return golden.Same("group", titleFacts(r.String("title")).Group, want)
}

func TestGoldenSonarrQuality(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/QualityParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_sdtv_quality":                                   sonarrQuality("SDTV"),
			"should_parse_dvd_quality":                                    sonarrQuality("DVD"),
			"should_parse_webdl480p_quality":                              sonarrQuality("WEBDL480p"),
			"should_parse_bluray480p_quality":                             sonarrQuality("Bluray480p"),
			"should_parse_webrip480p_quality":                             sonarrQuality("WEBRip480p"),
			"should_parse_bluray576p_quality":                             sonarrQuality("Bluray576p"),
			"should_parse_hdtv720p_quality":                               sonarrQuality("HDTV720p"),
			"should_parse_hdtv1080p_quality":                              sonarrQuality("HDTV1080p"),
			"should_parse_hdtv2160p_quality":                              sonarrQuality("HDTV2160p"),
			"should_parse_webdl720p_quality":                              sonarrQuality("WEBDL720p"),
			"should_parse_webrip720p_quality":                             sonarrQuality("WEBRip720p"),
			"should_parse_webdl1080p_quality":                             sonarrQuality("WEBDL1080p"),
			"should_parse_webrip1080p_quality":                            sonarrQuality("WEBRip1080p"),
			"should_parse_webdl2160p_quality":                             sonarrQuality("WEBDL2160p"),
			"should_parse_webrip2160p_quality":                            sonarrQuality("WEBRip2160p"),
			"should_parse_bluray720p_quality":                             sonarrQuality("Bluray720p"),
			"should_parse_bluray1080p_quality":                            sonarrQuality("Bluray1080p"),
			"should_parse_bluray1080p_remux_quality":                      sonarrQuality("Bluray1080pRemux"),
			"should_parse_bluray2160p_quality":                            sonarrQuality("Bluray2160p"),
			"should_parse_bluray2160p_remux_quality":                      sonarrQuality("Bluray2160pRemux"),
			"should_parse_raw_quality":                                    sonarrQuality("RAWHD"),
			"quality_parse":                                               sonarrQuality("Unknown"),
			"should_parse_full_quality_from_name":                         resolutionKnown(true),
			"should_parse_resolution_from_name":                           resolutionKnown(false),
			"should_parse_source_from_name":                               sourceKnown,
			"should_parse_quality_from_extension":                         resolutionKnown(true),
			"should_be_able_to_parse_repack":                              repack,
			"should_be_able_to_parse_anime_version":                       readVersion,
			"should_parse_resolution_from_name_and_source_from_extension": resolutionKnown(true),
			"parsing_our_own_quality_enum_name": func(r golden.Row) error {
				name := enumValue(r.Ref("quality"))
				f := titleFacts(fmt.Sprintf("My series S01E01 [%s]", golden.SonarrNames[name]))
				return golden.SonarrQuality(name).Read(f)
			},
			"should_parse_quality_from_other_source": func(r golden.Row) error {
				q := golden.SonarrQuality(enumValue(r.Ref("quality")))
				return otherSource(r, func(f release.Facts) error { return errors.Join(q.Read(f), version(f, 1)) })
			},
		},
	})
}

func TestGoldenRadarrQuality(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/radarr/QualityParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_ts":                                             radarrQuality("TELESYNC", "R720p", "NONE"),
			"should_parse_cam":                                            radarrQuality("CAM", "Unknown", "NONE"),
			"should_parse_sdtv_quality":                                   radarrQuality("TV", "R480p", "NONE"),
			"should_parse_dvd_quality":                                    radarrQuality("DVD", "Unknown", "NONE"),
			"should_parse_dvdr_quality":                                   radarrQuality("DVD", "R480p", "REMUX"),
			"should_parse_webdl480p_quality":                              radarrQuality("WEBDL", "R480p", "NONE"),
			"should_parse_webrip480p_quality":                             radarrQuality("WEBRIP", "R480p", "NONE"),
			"should_parse_bluray480p_quality":                             radarrQuality("BLURAY", "R480p", "NONE"),
			"should_parse_hdtv720p_quality":                               radarrQuality("TV", "R720p", "NONE"),
			"should_parse_hdtv1080p_quality":                              radarrQuality("TV", "R1080p", "NONE"),
			"should_parse_hdtv2160p_quality":                              radarrQuality("TV", "R2160p", "NONE"),
			"should_parse_webdl720p_quality":                              radarrQuality("WEBDL", "R720p", "NONE"),
			"should_parse_webrip720p_quality":                             radarrQuality("WEBRIP", "R720p", "NONE"),
			"should_parse_webdl1080p_quality":                             radarrQuality("WEBDL", "R1080p", "NONE"),
			"should_parse_webrip1080p_quality":                            radarrQuality("WEBRIP", "R1080p", "NONE"),
			"should_parse_webdl2160p_quality":                             radarrQuality("WEBDL", "R2160p", "NONE"),
			"should_parse_webrip2160p_quality":                            radarrQuality("WEBRIP", "R2160p", "NONE"),
			"should_parse_bluray720p_quality":                             radarrQuality("BLURAY", "R720p", "NONE"),
			"should_parse_bluray1080p_quality":                            radarrQuality("BLURAY", "R1080p", "NONE"),
			"should_parse_bluray2160p_quality":                            radarrQuality("BLURAY", "R2160p", "NONE"),
			"should_parse_bluray576p_quality":                             radarrBluray("R576p", "NONE"),
			"should_parse_remux720p_as_bluray720p_quality":                radarrBluray("R720p", "NONE"),
			"should_parse_remux1080p_quality":                             radarrBluray("R1080p", "REMUX"),
			"should_parse_remux2160p_quality":                             radarrBluray("R2160p", "REMUX"),
			"should_parse_brdisk_1080p_quality":                           radarrBluray("R1080p", "BRDISK"),
			"should_parse_raw_quality":                                    radarrQuality("TV", "R1080p", "RAWHD"),
			"quality_parse":                                               radarrQuality("UNKNOWN", "Unknown", "NONE"),
			"should_parse_full_quality_from_name":                         resolutionKnown(true),
			"should_parse_resolution_from_name":                           resolutionKnown(false),
			"should_parse_source_from_name":                               sourceKnown,
			"should_parse_quality_from_extension":                         resolutionKnown(true),
			"should_parse_resolution_from_name_and_source_from_extension": resolutionKnown(true),
			"should_be_able_to_parse_repack":                              repack,
			"should_parse_quality_from_other_source": func(r golden.Row) error {
				source, res, mod := enumValue(r.Ref("source")), enumValue(r.Ref("resolution")), enumValue(r.Ref("modifier"))
				return otherSource(r, func(f release.Facts) error { return radarrReads(f, source, res, mod, 1) })
			},
		},
	})
}

func TestGoldenRadarrExtendedQuality(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/radarr/ExtendedQualityParserRegex.json",
		Checks: map[string]golden.Check{
			"should_parse_reality_from_title": func(r golden.Row) error {
				return golden.Same("real", titleFacts(r.String("title")).Revision.Real, r.Int("reality"))
			},
			"should_parse_version_from_title": readVersion,
			"should_parse_ultrahd_from_title": func(r golden.Row) error {
				return golden.Same("resolution", titleFacts(r.String("title")).Resolution, release.Known(release.R2160p))
			},
		},
	})
}

func TestGoldenSonarrReleaseGroup(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/sonarr/ReleaseGroupParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_release_group":                      group,
			"should_parse_exception_release_group":            group,
			"should_not_include_language_in_release_group":    group,
			"should_not_include_repost_in_release_group":      group,
			"should_parse_anime_release_groups":               group,
			"should_not_parse_anime_hash_as_release_group":    noGroup,
			"should_not_parse_episode_title_as_release_group": noGroup,
		},
	})
}

func TestGoldenRadarrReleaseGroup(t *testing.T) {
	golden.Run(t, golden.Fixture{
		Path: "testdata/upstream/radarr/ReleaseGroupParserFixture.json",
		Checks: map[string]golden.Check{
			"should_parse_release_group":                     group,
			"should_parse_exception_release_group":           group,
			"should_not_include_extension_in_release_group":  group,
			"should_not_include_language_in_release_group":   group,
			"should_not_include_bad_suffix_in_release_group": group,
			"should_parse_anime_release_groups":              group,
			"should_not_parse_anime_hash_as_release_group":   noGroup,
		},
	})
}
