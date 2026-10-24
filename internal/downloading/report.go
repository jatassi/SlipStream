package downloading

import (
	"context"
	"maps"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
	"github.com/jatassi/slipstream/internal/downloading/internal/plan"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// report sends, in one local Write, the report of each torrent whose
// state changed since its last and every confirmed discard. A report
// carries no clock-driven field in its comparison: progress and totals
// ride along when something else moved, and in the totals batch.
func (l *loop) report(now time.Time) {
	if l.run == nil {
		return
	}
	reports, sent := l.changed(now)
	if len(reports) == 0 && len(l.parted) == 0 {
		return
	}
	if err := l.sendReports(reports); err != nil {
		for _, t := range sent {
			t.sent = nil
		}
		if l.run.ctx.Err() == nil {
			l.c.cfg.Log.Error("report the torrents' state", "err", err.Error())
		}
		return
	}
	l.parted = nil
	for _, t := range sent {
		t.sentFolder = true
		t.sentCompletion = t.sentCompletion || t.Complete
	}
}

// changed is the filled report of each torrent whose report changed, and
// those torrents, which count it sent.
func (l *loop) changed(now time.Time) ([]*dlv1.TorrentReport, []*torrent) {
	var reports []*dlv1.TorrentReport
	var sent []*torrent
	for _, h := range slices.SortedFunc(maps.Keys(l.torrents), compareHashes) {
		t := l.torrents[h]
		r := t.report()
		if r == nil || proto.Equal(r, t.sent) {
			continue
		}
		t.sent = r
		reports = append(reports, l.fill(t, proto.CloneOf(r), now))
		sent = append(sent, t)
	}
	return reports, sent
}

func (l *loop) sendReports(reports []*dlv1.TorrentReport) error {
	run, parted := l.run, l.parted
	return l.local.Write(run.ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		if len(reports) > 0 {
			if err := run.h.Report(ctx, tx, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: reports}}}); err != nil {
				return err
			}
		}
		for _, p := range parted {
			if err := run.h.Report(ctx, tx, &dlv1.Report{Kind: &dlv1.Report_Parted{Parted: p}}); err != nil {
				return err
			}
		}
		return nil
	})
}

func compareHashes(a, b bencode.InfoHash) int { return slices.Compare(a[:], b[:]) }

// report is t's state as the Controller reads it, nil while there is
// nothing to say: a torrent the engine doesn't hold and nothing holds
// back.
func (t *torrent) report() *dlv1.TorrentReport {
	if t.handle == 0 && t.failed == "" && t.state.Hold == plan.HoldNone {
		return nil
	}
	r := &dlv1.TorrentReport{
		Id:       t.id.Bytes(),
		Engine:   t.engineState(),
		Hold:     holdOnWire[t.state.Hold],
		Complete: t.Complete,
		Stalled:  t.state.Stalled,
		Error:    t.failed,
		Fault:    t.fault,
	}
	if !t.state.Until.IsZero() {
		r.UntilUnixMs = t.state.Until.UnixMilli()
	}
	for _, tr := range t.trackers {
		r.Trackers = append(r.Trackers, trackerLine(tr))
	}
	return r
}

// fill adds to r what rides along with a change: the time, progress,
// totals, and once each, the save folder and the files.
func (l *loop) fill(t *torrent, r *dlv1.TorrentReport, now time.Time) *dlv1.TorrentReport {
	r.AtUnixMs, r.Done, r.Wanted = now.UnixMilli(), t.status.Done, t.status.Wanted
	if t.handle == 0 {
		return r
	}
	if !t.statusAt.IsZero() {
		r.Totals = t.totals(now)
	}
	if !t.sentFolder {
		r.SaveFolder = t.folder
	}
	if t.Complete && !t.sentCompletion {
		files, err := l.eng.Files(t.handle)
		if err != nil {
			l.call(t, "list a torrent's files", err)
			return r
		}
		r.Completion = &dlv1.Completion{}
		for _, f := range files {
			r.Completion.Files = append(r.Completion.Files, &dlv1.File{Index: int32(f.Index), Path: f.Path, Size: f.Size}) //nolint:gosec // a torrent's file index fits
		}
	}
	return r
}

func (t *torrent) engineState() dlv1.EngineState {
	switch {
	case t.failed != "":
		return dlv1.EngineState_ENGINE_STATE_FAILED
	case t.Phase == plan.Checking || checking(t.status.State):
		return dlv1.EngineState_ENGINE_STATE_CHECKING
	case t.Complete:
		return dlv1.EngineState_ENGINE_STATE_SEEDING
	}
	return dlv1.EngineState_ENGINE_STATE_DOWNLOADING
}

// totals is t's counters at now: the engine's latest, with the clocks
// run on since, as the engine reports a torrent only when something
// besides its clocks moved.
func (t *torrent) totals(now time.Time) *dlv1.TotalsRow {
	tot := t.status.Totals
	if t.Phase == plan.Running {
		ran := now.Sub(t.statusAt)
		tot.Active += ran
		if t.Complete {
			tot.Seeding += ran
		}
	}
	row := &dlv1.TotalsRow{
		Id: t.id.Bytes(), Uploaded: tot.Uploaded, Downloaded: tot.Downloaded,
		ActiveSeconds: int64(tot.Active / time.Second), SeedingSeconds: int64(tot.Seeding / time.Second),
	}
	if !t.Complete {
		row.Done = t.status.Done
	}
	return row
}

func (l *loop) sendTotals(now time.Time) {
	if l.run == nil {
		return
	}
	var rows []*dlv1.TotalsRow
	var moved []*torrent
	for _, t := range l.torrents {
		if t.statusAt.IsZero() {
			continue
		}
		row := t.totals(now)
		if proto.Equal(row, t.sentTotals) {
			continue
		}
		rows = append(rows, row)
		moved = append(moved, t)
	}
	if len(rows) == 0 {
		return
	}
	slices.SortFunc(rows, func(a, b *dlv1.TotalsRow) int { return slices.Compare(a.GetId(), b.GetId()) })
	run := l.run
	err := l.local.Write(run.ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return run.h.Report(ctx, tx, &dlv1.Report{Kind: &dlv1.Report_Totals{Totals: &dlv1.Totals{Rows: rows}}})
	})
	if err != nil {
		if run.ctx.Err() == nil {
			l.c.cfg.Log.Error("report the torrents' totals", "err", err.Error())
		}
		return
	}
	for i, t := range moved {
		t.sentTotals = rows[i]
	}
}

// trackerLine is a tracker as the Controller shows it. A tracker that
// answered once reads working through its later announces, so a regular
// announce changes no report.
func trackerLine(tr engine.Tracker) *dlv1.TrackerLine {
	line := &dlv1.TrackerLine{Index: int32(tr.Index), Message: tr.Message} //nolint:gosec // a tracker index fits
	switch {
	case tr.LastError != nil || tr.Fails > 0:
		line.Status = dlv1.TrackerStatus_TRACKER_STATUS_ERROR
		if announce.Unregistered(tr.Message) {
			line.Status = dlv1.TrackerStatus_TRACKER_STATUS_UNREGISTERED
		}
		if !tr.NextAnnounce.IsZero() {
			line.RetryAtUnixMs = tr.NextAnnounce.UnixMilli()
		}
	case tr.NextAnnounce.IsZero() && !tr.Updating:
		line.Status = dlv1.TrackerStatus_TRACKER_STATUS_NOT_CONTACTED
	case tr.NextAnnounce.IsZero():
		line.Status = dlv1.TrackerStatus_TRACKER_STATUS_UPDATING
	default:
		line.Status = dlv1.TrackerStatus_TRACKER_STATUS_WORKING
	}
	return line
}

var holdOnWire = map[plan.Hold]dlv1.Hold{
	plan.HoldNone:       dlv1.Hold_HOLD_UNSPECIFIED,
	plan.HoldWanted:     dlv1.Hold_HOLD_WANTED,
	plan.HoldWindow:     dlv1.Hold_HOLD_WINDOW,
	plan.HoldDiskFull:   dlv1.Hold_HOLD_DISK_FULL,
	plan.HoldSlot:       dlv1.Hold_HOLD_SLOT,
	plan.HoldCheckSlot:  dlv1.Hold_HOLD_CHECK_SLOT,
	plan.HoldDiskGuard:  dlv1.Hold_HOLD_DISK_GUARD,
	plan.HoldDraining:   dlv1.Hold_HOLD_DRAINING,
	plan.HoldController: dlv1.Hold_HOLD_CONTROLLER,
}
