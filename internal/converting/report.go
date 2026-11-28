package converting

import (
	"context"

	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// The fault codes this holder reports, keys of conversions' faults table.
const (
	Encoder     = "encoder"
	Filter      = "filter"
	DeviceFault = "device"
	Unreadable  = "unreadable"
	Crash       = "crash"
	Stalled     = "stalled"
	// ShortOutput is an output whose length isn't its source's.
	ShortOutput = "short_output"
	// Streams is an output without the streams its settings promise.
	Streams = "streams"
	// DiskFull is reported as Result.disk_full, never as a Fault.
	DiskFull = "disk_full"
)

func fault(code, tail string) *cv1.Result {
	return &cv1.Result{Result: &cv1.Result_Fault{Fault: &cv1.Fault{Code: code, Tail: tail}}}
}

// report queues r for j's attempt in a Local Write, so it reaches the
// Controller once.
func (h *Holder) report(ctx context.Context, rh holderOf, j *cv1.Job, r *cv1.Result) {
	r.ConversionId, r.Attempt = j.GetConversionId(), j.GetAttempt()
	err := h.cfg.DB.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return rh.Report(ctx, tx, r)
	})
	if err != nil && ctx.Err() == nil {
		h.cfg.Log.Error("queue a Conversion's result", "err", err.Error())
	}
}
