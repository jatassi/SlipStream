package converting

import (
	"bytes"
	"context"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"

	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
)

// facts is what the holder reports of its ffmpeg, its scratch volume and
// its runs, at most every Timings.Progress.
type facts struct {
	h         *Holder
	signature Signature
	devices   []*cv1.Device
	last      *cv1.Facts
}

// newFacts reads the Node's signature once per Run.
func (h *Holder) newFacts(ctx context.Context) *facts {
	f := &facts{h: h}
	sig, devices, err := h.cfg.Proofs.Signature(ctx)
	if err != nil {
		h.cfg.Log.Warn("read ffmpeg's signature", "err", err.Error())
		return f
	}
	f.signature = sig
	for _, d := range devices {
		f.devices = append(f.devices, &cv1.Device{Name: d.Name, Vendor: d.Vendor, Driver: d.Driver})
	}
	return f
}

func (f *facts) loop(ctx context.Context, rh holderOf) {
	tick := time.NewTicker(f.h.cfg.Timings.Progress)
	defer tick.Stop()
	for {
		f.publish(ctx, rh)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

func (f *facts) publish(ctx context.Context, rh holderOf) {
	next := &cv1.Facts{
		Signature: string(f.signature), Devices: f.devices, Scratch: f.volume(ctx),
		Progress: f.h.progress(), Override: f.h.cfg.Override() != "",
	}
	if proto.Equal(next, f.last) {
		return
	}
	rh.SetFacts(next)
	f.last = next
}

// volume is the scratch folder's volume, nil while it can't be read.
func (f *facts) volume(ctx context.Context) *cv1.Volume {
	v, err := f.h.cfg.Disk.Volume(ctx, f.h.cfg.Scratch())
	if err != nil {
		return nil
	}
	return &cv1.Volume{Free: v.Free, Total: v.Total, Id: v.ID, Name: v.Name, Mount: v.Mount, SampledAtUnixMs: v.SampledAt.UnixMilli()}
}

// progress is one entry per live run, in conversion id order.
func (h *Holder) progress() []*cv1.Progress {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []*cv1.Progress
	for id, r := range h.runs {
		if r.ended {
			continue
		}
		out = append(out, &cv1.Progress{
			ConversionId: []byte(id), Attempt: r.attempt, OutTimeUs: r.progress.OutTimeUs,
			Speed: float32(r.progress.Speed), Paused: r.paused,
		})
	}
	slices.SortFunc(out, func(a, b *cv1.Progress) int { return bytes.Compare(a.GetConversionId(), b.GetConversionId()) })
	return out
}
