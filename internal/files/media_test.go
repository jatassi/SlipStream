package files_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
)

func TestAnHDRTargetJudgesWhatAFilesMediaRecordReads(t *testing.T) {
	dv := media.Record{
		Version: media.Version, Container: "matroska,webm",
		Video: media.Video{Codec: "hevc", Width: 1920, Height: 1080, BitDepth: 10, HDR: []release.HDR{"DV", "HDR10"}, DolbyVision: &media.DolbyVision{Profile: 8, Level: 6, Compatibility: 1}},
		Audio: []media.Audio{{Codec: "truehd", Profile: "Dolby TrueHD + Dolby Atmos", Atmos: true, Channels: 8, Layout: "7.1", Language: "en", Default: true}},
	}
	sdr := media.Record{Version: media.Version, Video: media.Video{Codec: "h264", Width: 1920, Height: 1080, BitDepth: 8, HDR: []release.HDR{"SDR"}}}
	old := media.Record{Video: media.Video{Codec: "h264", Width: 1920, Height: 1080}}
	for _, c := range []struct {
		name   string
		record media.Record
		fills  bool
	}{
		{"Dolby Vision over HDR10 fills HDR", dv, true},
		{"SDR fills nothing", sdr, false},
		{"a record from before HDR was read fills HDR", old, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			l := open(t)
			var f targets.Filters
			if err := f.Set(targets.HDR, targets.Only, []string{"HDR10"}); err != nil {
				t.Fatal(err)
			}
			spec := targets.Spec{Scores: targets.Scores{Increment: 1}, Name: "HDR", Tiers: [][]release.Quality{{release.WEBDL1080p}}, Cutoff: release.WEBDL1080p, Filters: f}
			hdr := must[targets.Target](t)(l.targets.CreateTarget(t.Context(), admin, spec))
			tree := l.work(t, "w1", true, new(l.profile(t, "Films", hdr)))
			svc := l.files(files.NoGrabs{})
			v := l.judge(t, svc, tree.ID, holdingOf(tree, 0, files.HoldingOf(storage.NewUUID(), ".mkv"), 7, c.record, release.WEBDL1080p))
			if c.fills != slices.Equal(targetsOf(v.Fills), []targets.TargetID{hdr.ID}) || c.fills != v.Reasons[hdr.ID].Meets() {
				t.Fatalf("Judge = fills %v, HDR's verdict %+v; want fills %t", v.Fills, v.Reasons[hdr.ID], c.fills)
			}
			if !c.fills {
				return
			}
			l.downloadProbed(t, svc, tree, 0, release.WEBDL1080p, "payload", c.record)
			_, cells, _, err := svc.WorkCells(t.Context(), admin, tree.ID)
			if err != nil {
				t.Fatal(err)
			}
			if cells[0].File == nil || !reflect.DeepEqual(cells[0].File.Media, c.record) {
				t.Fatalf("the first cell's file = %+v, want one keeping its whole media record %+v", cells[0].File, c.record)
			}
		})
	}
}
