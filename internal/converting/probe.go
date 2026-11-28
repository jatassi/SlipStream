package converting

import (
	"context"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/role"
)

// Base is the base test: SDR H.264 from an H.264 clip.
var Base = encode.Key{
	Shape: encode.Shape{Codec: encode.H264, Depth: encode.Bit8, Range: encode.SDR, Curve: encode.BT2390},
	Class: encode.ClassH264x8,
}

// probe is the base test's last answer.
type probe struct {
	mu       sync.Mutex
	answered bool
	fit      role.Fit
}

func (c *probe) set(fit role.Fit) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.answered, c.fit = true, fit
}

func (p *realProofs) Runs() (role.Fit, role.ProbeReason) {
	p.probe.mu.Lock()
	defer p.probe.mu.Unlock()
	if !p.probe.answered {
		return role.Weak, role.FFmpegTesting
	}
	return p.probe.fit, map[role.Fit]role.ProbeReason{role.Weak: role.NoGPU, role.Fails: role.FFmpegFails}[p.probe.fit]
}

func (p *realProofs) Warm(ctx context.Context, changed func()) error {
	tick := time.NewTicker(p.cfg.Timings.Signature)
	defer tick.Stop()
	var last Signature
	for first := true; ; first = false {
		if sig, _, _ := p.Signature(ctx); first || sig != last {
			last = sig
			fit := runsOf(p.test(ctx, Base))
			if ctx.Err() != nil {
				return nil
			}
			p.probe.set(fit)
			changed()
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}

// runsOf is the ffmpeg-runs probe's answer from the base test: it fits
// when the test passed and a hardware row opened its encoder, is weak when
// only the processor did, and fails when no row passed.
func runsOf(proved Proved, opened bool, err error) role.Fit {
	switch {
	case err != nil || !proved.Passed:
		return role.Fails
	case opened || proved.Pipeline.Path != encode.Processor:
		return role.Fits
	}
	return role.Weak
}
