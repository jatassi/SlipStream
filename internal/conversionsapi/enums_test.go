package conversionsapi

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/conversions"
	"github.com/jatassi/slipstream/internal/encode"
)

func enumOf[T any](t *testing.T, field string) string {
	t.Helper()
	f, ok := reflect.TypeFor[T]().FieldByName(field)
	if !ok {
		t.Fatalf("%s has no field %s", reflect.TypeFor[T](), field)
	}
	return f.Tag.Get("enum")
}

func joined[S ~string](values ...S) string {
	words := make([]string, len(values))
	for i, v := range values {
		words[i] = string(v)
	}
	return strings.Join(words, ",")
}

func TestTheWireEnumsNameEveryDomainValueInOrder(t *testing.T) {
	stages := joined(conversions.StageQueued, conversions.StageCopying, conversions.StageReady, conversions.StageRunning, conversions.StagePaused,
		conversions.StageChecking, conversions.StageReturning, conversions.StageLanding, conversions.StageDone, conversions.StageFailed, conversions.StageCanceled)
	lanes := joined(conversions.LaneMovedUp, conversions.LaneNew, conversions.LaneUpgrade, conversions.LaneBackfill)
	classes := joined(encode.ClassH264x8, encode.ClassHEVCx8, encode.ClassHEVCx10, encode.ClassAV1x8, encode.ClassAV1x10, encode.ClassHDR10,
		encode.ClassHLG, encode.ClassDV5, encode.ClassDV81, encode.ClassDecodeSDR, encode.ClassDecodeHDR10)
	for _, c := range []struct{ name, got, want string }{
		{"ConversionRow.Stage", enumOf[ConversionRow](t, "Stage"), stages},
		{"ConversionRow.Lane", enumOf[ConversionRow](t, "Lane"), lanes},
		{"laneRows.Lane", enumOf[laneRows](t, "Lane"), lanes},
		{"failure.Class", enumOf[failure](t, "Class"), joined(conversions.RunClass, conversions.SourceClass)},
		{"attempt.Outcome", enumOf[attempt](t, "Outcome"), joined(conversions.AttemptFailed, conversions.AttemptStalled, conversions.AttemptDiskFull, conversions.AttemptPutBack)},
		{"conversionGroup.GroupKind", enumOf[conversionGroup](t, "GroupKind"), joined(conversions.GroupBackfill, conversions.GroupSeason)},
		{"listInput.Band", enumOf[listInput](t, "Band"), joined(conversions.BandAttention, conversions.BandConverting, conversions.BandNext)},
		{"proof.Class", enumOf[proof](t, "Class"), classes},
		{"proof.Path", enumOf[proof](t, "Path"), joined(encode.Hardware, encode.HardwareOpenCL, encode.Processor)},
		{"proof.API", enumOf[proof](t, "API"), joined(encode.QSV, encode.VAAPI, encode.CUDA, encode.VideoToolbox)},
		{"sentence.API", enumOf[sentence](t, "API"), joined(encode.QSV, encode.VAAPI, encode.CUDA, encode.VideoToolbox)},
		{"shape.Codec", enumOf[shape](t, "Codec"), joined(encode.H264, encode.HEVC, encode.AV1)},
		{"shape.Depth", enumOf[shape](t, "Depth"), strconv.Itoa(int(encode.Bit8)) + "," + strconv.Itoa(int(encode.Bit10))},
		{"shape.Range", enumOf[shape](t, "Range"), joined(encode.SDR, encode.HDR10, encode.SourceRange)},
		{"shape.Curve", enumOf[shape](t, "Curve"), joined(encode.BT2390, encode.Hable, encode.Mobius, encode.Reinhard)},
	} {
		if c.got != c.want {
			t.Errorf("%s enum = %q, want %q", c.name, c.got, c.want)
		}
	}
}
