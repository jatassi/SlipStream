package downloading

import (
	"errors"
	"time"

	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// recordFault records a release fault of kind on t at at. The first one
// stands.
func (t *torrent) recordFault(kind dlv1.FaultKind, at time.Time) {
	if t.fault == nil {
		t.fault = &dlv1.Fault{Kind: kind, AtUnixMs: at.UnixMilli()}
	}
}

// fail records that the engine failed t with err, and an engine error
// fault when err is the torrent's own.
func (t *torrent) fail(err error, at time.Time) {
	t.failed = err.Error()
	if e, ok := errors.AsType[*engine.Error](err); ok {
		t.failed = e.Text
	}
	if blamesTheTorrent(err) {
		t.recordFault(dlv1.FaultKind_FAULT_KIND_ENGINE_ERROR, at)
	}
}

// blamesTheTorrent is whether the engine blames err on the torrent:
// libtorrent refusing what the metainfo says. A system error is the
// Node's disk or files whatever the release, and the shim's are the
// engine's own.
func blamesTheTorrent(err error) bool {
	return errors.Is(err, engine.CategoryLibtorrent) || errors.Is(err, engine.CategoryBdecode)
}

// noPayload records the fault of each download that ran the fault stall
// time and received no payload. Time paused or queued doesn't count, so
// a download waiting for a slot is never blamed for it.
func (l *loop) noPayload(now time.Time) {
	stall := time.Duration(l.intent.Intent.GetThresholds().GetFaultStallSeconds()) * time.Second
	for _, t := range l.torrents {
		if t.fault != nil || t.Complete || t.statusAt.IsZero() {
			continue
		}
		if tot := t.totals(now); tot.GetDownloaded() == 0 && time.Duration(tot.GetActiveSeconds())*time.Second >= stall {
			t.recordFault(dlv1.FaultKind_FAULT_KIND_NO_PAYLOAD, now)
		}
	}
}
