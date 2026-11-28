package converting

import (
	"context"
	"testing"

	"github.com/jatassi/slipstream/internal/encode"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/rpc"
)

type keyProofs struct {
	Proofs
	asked []encode.Key
}

func (p *keyProofs) Test(_ context.Context, k encode.Key) (Proved, error) {
	p.asked = append(p.asked, k)
	return Proved{Passed: true, Pipeline: encode.Pipeline{Path: encode.Hardware, API: encode.QSV}, Speed: 4, Signature: "sig"}, nil
}

func TestTheServiceTestsOnlyAShapeSettingsHoldAndAClassWithAClip(t *testing.T) {
	sdr := &cv1.Shape{Codec: "h264", Depth: 8, Range: "sdr", Curve: "bt2390"}
	for _, c := range []struct {
		name  string
		req   *cv1.TestRequest
		asked bool
	}{
		{"the base test", &cv1.TestRequest{Shape: sdr, Class: "h264_8"}, true},
		{"Keep Dolby Vision on HEVC 10-bit", &cv1.TestRequest{Shape: &cv1.Shape{Codec: "hevc", Depth: 10, Range: "source", KeepDv: true, Curve: "bt2390"}, Class: "dv81"}, true},
		{"10-bit H.264", &cv1.TestRequest{Shape: &cv1.Shape{Codec: "h264", Depth: 10, Range: "sdr", Curve: "bt2390"}, Class: "h264_8"}, false},
		{"an unknown codec", &cv1.TestRequest{Shape: &cv1.Shape{Codec: "vp9", Depth: 8, Range: "sdr", Curve: "bt2390"}, Class: "h264_8"}, false},
		{"a class with no clip", &cv1.TestRequest{Shape: sdr, Class: "vc1"}, false},
	} {
		p := &keyProofs{}
		res, err := Service(p).Test(t.Context(), c.req)
		if !c.asked {
			if rpc.CodeOf(err) != rpc.CodeInvalidArgument || len(p.asked) > 0 {
				t.Errorf("%s: %v after %d tests, want invalid_argument before any", c.name, err, len(p.asked))
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		k := p.asked[0]
		sh := c.req.GetShape()
		if string(k.Shape.Codec) != sh.GetCodec() || int(k.Shape.Depth) != int(sh.GetDepth()) || string(k.Shape.Range) != sh.GetRange() ||
			k.Shape.KeepDV != sh.GetKeepDv() || string(k.Class) != c.req.GetClass() {
			t.Errorf("%s: tested %+v", c.name, k)
		}
		if !res.GetPassed() || res.GetPipeline().GetApi() != "qsv" || res.GetSpeed() != 4 || res.GetSignature() != "sig" {
			t.Errorf("%s: answered %v", c.name, res)
		}
	}
}
