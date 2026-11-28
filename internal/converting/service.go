package converting

import (
	"context"

	"connectrpc.com/connect/v2"

	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/encode"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1/convertingv1connect"
)

// Service is ConvertingService on this Node, mounted in both halves.
func Service(p Proofs) convertingv1connect.ConvertingServiceHandler {
	return service{proofs: p}
}

type service struct{ proofs Proofs }

func (s service) Test(ctx context.Context, req *cv1.TestRequest) (*cv1.TestResponse, error) {
	k, err := keyOf(req)
	if err != nil {
		return nil, err
	}
	proved, err := s.proofs.Test(ctx, k)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "the test run didn't finish: "+err.Error())
	}
	res := &cv1.TestResponse{
		Passed: proved.Passed, Device: proved.Device, Speed: proved.Speed, Pauses: proved.Pauses,
		Signature: string(proved.Signature),
	}
	if proved.Passed {
		res.Pipeline = pipelineOf(proved.Pipeline)
	}
	for _, t := range proved.Tries {
		res.Tries = append(res.Tries, &cv1.Try{Pipeline: pipelineOf(t.Pipeline), Error: t.Err})
	}
	return res, nil
}

// keyOf parses a request's shape as Conversion settings would hold it, and
// its class as one the clip maker makes.
func keyOf(req *cv1.TestRequest) (encode.Key, error) {
	sh := req.GetShape()
	s, err := encode.Parse(encode.Raw{
		Codec: sh.GetCodec(), MaxHeight: int(encode.P1080), Kbps: encode.MinKbps, Depth: int(sh.GetDepth()),
		Range: sh.GetRange(), KeepDV: sh.GetKeepDv(), Curve: sh.GetCurve(),
		Audio: string(encode.CopyAudio), Container: string(encode.MKV),
	})
	if err != nil {
		return encode.Key{}, connect.NewError(connect.CodeInvalidArgument, err.Error())
	}
	c := encode.Class(req.GetClass())
	if clip.VideoOf(c, clip.Trial, "").Codec == nil {
		return encode.Key{}, connect.NewError(connect.CodeInvalidArgument, "the class "+string(c)+" has no clip")
	}
	return encode.Key{Shape: s.Shape(), Class: c}, nil
}

func pipelineOf(p encode.Pipeline) *cv1.Pipeline {
	return &cv1.Pipeline{Path: string(p.Path), Api: string(p.API)}
}
