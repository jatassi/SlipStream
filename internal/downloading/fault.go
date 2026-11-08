package downloading

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/downloading/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// recordFault records a release fault of kind on t at at. The first one
// stands.
func (t *torrent) recordFault(kind dlv1.FaultKind, at time.Time) {
	if t.fault == nil {
		t.fault = &dlv1.Fault{Kind: kind, AtUnixMs: at.UnixMilli()}
	}
}

// fail records that the engine failed t with err, in memory and in its
// local row, and an engine error fault when err is the torrent's own.
func (l *loop) fail(t *torrent, err error, at time.Time) {
	t.failed = err.Error()
	if e, ok := errors.AsType[*engine.Error](err); ok {
		t.failed = e.Text
	}
	if blamesTheTorrent(err) {
		t.recordFault(dlv1.FaultKind_FAULT_KIND_ENGINE_ERROR, at)
	}
	l.storeFailed(t)
}

// retry acts on the intent's Retry at: it clears t's failure and checks
// its files, whatever it acted on before a restart, so a Retry the Node
// restarted in the middle of runs its check again. The intent pauses t
// in the same snapshot, so the check starts no download.
func (l *loop) retry(t *torrent, at int64) {
	t.retried, t.NeedsCheck = at, true
	if t.failed != "" {
		t.failed = ""
		l.storeFailed(t)
	}
}

// found reads what t's check found, for the report that names its Retry.
func (l *loop) found(t *torrent) {
	done, err := l.eng.FileProgress(t.handle)
	if err != nil {
		l.call(t, "read what a torrent's check found", err)
		return
	}
	t.status.Done = 0
	for _, d := range done {
		t.status.Done += d
	}
}

func (l *loop) storeFailed(t *torrent) {
	var failed *string
	if t.failed != "" {
		failed = &t.failed
	}
	err := l.local.Write(context.Background(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return db.New(tx).SetFailed(ctx, db.SetFailedParams{Failed: failed, InfoHash: t.Hash[:]})
	})
	if err != nil {
		l.c.cfg.Log.Error("record a torrent's failure", "info_hash", t.Hash.String(), "err", err.Error())
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
