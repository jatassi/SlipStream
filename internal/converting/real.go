package converting

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/role"
)

type RealConfig struct {
	Scratch func() string
	// Override is the configured ffmpeg path, empty for the bundled one.
	Override func() string
	Media    media.Prober
	Timings  Timings
}

// Real is the production Converter and Proofs: the bundled ffmpeg, ffprobe
// and dovi_tool (or the override ffmpeg path), under proctree, with
// Env.Environ minus every SLIPSTREAM_ variable.
func Real(RealConfig) (Converter, Proofs) { return realConverter{}, realProofs{} }

var errNotBuilt = errors.New("converting isn't built yet")

type realConverter struct{}

func (realConverter) Start(context.Context, []string) (Run, error) { return nil, errNotBuilt }

type realProofs struct{}

func (realProofs) Signature(context.Context) (Signature, []Device, error) {
	return "", nil, errNotBuilt
}

func (realProofs) Test(context.Context, encode.Key) (Proved, error) { return Proved{}, errNotBuilt }

func (realProofs) Runs() (role.Fit, role.ProbeReason) { return role.Fails, "" }

func (realProofs) Warm(context.Context, func()) error { return nil }
