package converting

import (
	"cmp"
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/converting/clip"
	"github.com/jatassi/slipstream/internal/converting/internal/drivers"
	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/role"
)

type RealConfig struct {
	Tools   clip.Tools
	Scratch func() string
	// Override is the configured ffmpeg path, empty for the bundled one.
	Override func() string
	Media    media.Prober
	Timings  Timings
}

// Real is the production Converter and Proofs: the bundled ffmpeg, ffprobe
// and dovi_tool (or the override ffmpeg path), under proctree, with
// Env.Environ minus every SLIPSTREAM_ variable.
func Real(cfg RealConfig) (Converter, Proofs) { return realConverter{}, realProofs{cfg: cfg} }

var errNotBuilt = errors.New("converting isn't built yet")

type realConverter struct{}

func (realConverter) Start(context.Context, []string) (Run, error) { return nil, errNotBuilt }

type realProofs struct{ cfg RealConfig }

func (p realProofs) Signature(ctx context.Context) (Signature, []Device, error) {
	ffmpeg := cmp.Or(p.cfg.Override(), p.cfg.Tools.FFmpeg)
	return readSignature(ctx, ffmpeg, p.cfg.Tools.Environ, drivers.Local(ffmpeg, p.cfg.Tools.Environ))
}

func (realProofs) Test(context.Context, encode.Key) (Proved, error) { return Proved{}, errNotBuilt }

func (realProofs) Runs() (role.Fit, role.ProbeReason) { return role.Fails, "" }

func (realProofs) Warm(context.Context, func()) error { return nil }
