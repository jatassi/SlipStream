package converting

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/encode"
	"github.com/jatassi/slipstream/internal/language"
	"github.com/jatassi/slipstream/internal/media"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
)

const partSuffix = ".part"

// watch follows one run to its end. A run whose output time doesn't
// advance for Timings.Stall is killed and reported stalled; a paused run
// never stalls.
func (h *Holder) watch(ctx context.Context, rh holderOf, j *cv1.Job, r *running) {
	stall := time.NewTimer(h.cfg.Timings.Stall)
	defer stall.Stop()
	for {
		select {
		case p, ok := <-r.run.Progress():
			if !ok {
				e := r.run.Wait()
				if !h.end(r) {
					return
				}
				h.report(ctx, rh, j, h.exited(ctx, r, e))
				return
			}
			if h.advanced(r, p) {
				stall.Reset(h.cfg.Timings.Stall)
			}
		case <-r.wake:
			if h.isPaused(r) {
				stall.Stop()
			} else {
				stall.Reset(h.cfg.Timings.Stall)
			}
		case <-stall.C:
			if !h.end(r) {
				return
			}
			r.run.Kill()
			e := r.run.Wait()
			h.report(ctx, rh, j, fault(Stalled, e.Tail))
			return
		}
	}
}

// advanced records p as r's progress, and whether it moved the output time
// past any earlier progress while r isn't paused.
func (h *Holder) advanced(r *running, p Progress) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	moved := p.OutTimeUs > r.progress.OutTimeUs && !r.paused
	r.progress = p
	return moved
}

func (h *Holder) isPaused(r *running) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return r.paused
}

// end marks r ended, false when the holder already killed it, which
// reports nothing.
func (h *Holder) end(r *running) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r.killed {
		return false
	}
	r.ended = true
	return true
}

// exited is the result of a run that exited by itself. An exit 0 renames
// the .part output into place and probes it: a fast fail, which the
// root-side check doesn't trust.
func (h *Holder) exited(ctx context.Context, r *running, e Exit) *cv1.Result {
	if e.Code != 0 {
		if code := Classify(e); code != DiskFull {
			return fault(code, e.Tail)
		}
		return &cv1.Result{Result: &cv1.Result_DiskFull{DiskFull: &cv1.DiskFull{}}}
	}
	if err := h.cfg.Disk.Rename(ctx, r.job.Output, r.output); err != nil {
		h.cfg.Log.Warn("rename a Conversion's output into place", "err", err.Error())
		return fault(Crash, e.Tail)
	}
	out, err := h.cfg.Media.Probe(ctx, r.output)
	if err != nil {
		return fault(Streams, e.Tail)
	}
	if code := mismatch(r.job.Source, r.job.Settings, r.job.Keep, out, h.cfg.Timings.LengthTolerance); code != "" {
		return fault(code, e.Tail)
	}
	info, err := h.cfg.Disk.Stat(ctx, r.output)
	if err != nil {
		h.cfg.Log.Warn("read a Conversion's output size", "err", err.Error())
		return fault(Crash, e.Tail)
	}
	return &cv1.Result{Result: &cv1.Result_Exited{Exited: &cv1.Exited{Bytes: info.Size()}}}
}

// mismatch is short_output when out's length is outside tolerance of
// src's, streams when out lacks the video codec or the audio and subtitle
// tracks s promises, and empty when out matches.
func mismatch(src media.Record, s encode.Settings, keep []language.Code, out media.Record, tol Tolerance) string {
	if d := out.Duration - src.Duration; d < -tol.Of(src.Duration) || d > tol.Of(src.Duration) {
		return ShortOutput
	}
	sel := encode.Tracks(src, s, keep)
	audio := len(sel.Audio)
	if audio > 0 && s.Audio == encode.AC3AndStereoAAC {
		audio = 2
	}
	codec := encode.Codec(src.Codec)
	if !encode.CopiesVideo(src, s) {
		codec = s.Codec
	}
	if encode.Codec(out.Codec) != codec || len(out.Audio) != audio || len(out.Subtitles) != len(sel.Subtitles) {
		return Streams
	}
	return ""
}
