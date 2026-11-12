package media_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/media"
)

func TestCounts(t *testing.T) {
	for rel, want := range map[string]bool{
		"Heat (1995)/Heat.mkv":              true,
		"Orchard.MP4":                       true,
		"Show/Season 01/Show - S01E01.m2ts": true,
		"Trailer Park Boys/Season 01/Trailer Park Boys - S01E01.mkv": true,
		"Proof (2005)/Proof (2005).mkv":                              true,
		"Other People (2016)/Other People.mkv":                       true,
		"Heat (1995)/Heat-sample.mkv":                                false,
		"Heat (1995)/heat-Trailer.mp4":                               false,
		"Heat (1995)/Heat-behindthescenes.mkv":                       false,
		"Heat (1995)/sample.mkv":                                     false,
		"Heat (1995)/Sample/heat.mkv":                                false,
		"Heat (1995)/Featurettes/Making Of.mkv":                      false,
		"Heat (1995)/Behind The Scenes/Set.mkv":                      false,
		"Heat (1995)/@eaDir/Heat.mkv":                                false,
		"Heat (1995)/BDMV/STREAM/00001.m2ts":                         false,
		"$RECYCLE.BIN/Heat.mkv":                                      false,
		".slipstream-incoming/0001.mkv":                              false,
		"Heat (1995)/.Heat.mkv":                                      false,
		"Heat (1995)/Heat.nfo":                                       false,
		"Heat (1995)/Heat.mkv.part":                                  false,
		"Heat.mkv/readme.txt":                                        false,
	} {
		if got := media.Counts(rel, media.DefaultVideos); got != want {
			t.Errorf("Counts(%q) = %v, want %v", rel, got, want)
		}
	}
}

func TestSkipsFolder(t *testing.T) {
	for name, want := range map[string]bool{
		"Extras": true, "trailers": true, "System Volume Information": true, "#recycle": true, ".git": true, "VIDEO_TS": true,
		"Trailer Park Boys": false, "Season 01": false, "Heat (1995)": false,
	} {
		if got := media.SkipsFolder(name); got != want {
			t.Errorf("SkipsFolder(%q) = %v, want %v", name, got, want)
		}
	}
}
