package external

import (
	"cmp"
	"context"
	"path/filepath"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading/internal/announce"
	"github.com/jatassi/slipstream/internal/downloading/internal/db"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/torrent/clients"
)

// report adds to b each live torrent's report that changed, its counters
// at the totals cadence and at first sight, the rows that changed, and the
// client's poll when it changed or a Check waits on it.
func (s *state) report(now time.Time, b *batch) {
	b.totalsSent = now.Sub(s.totalsAt) >= s.cfg.Timings.TotalsReport
	for _, t := range s.sorted() {
		if t.wanted.Applied != 0 {
			t.row.Applied = new(runWord[t.wanted.Applied])
		}
		if t.row.Origin != nil && !sameRow(t.row, t.written) {
			b.rows = append(b.rows, t)
		}
		if t.vanished && !t.told {
			b.vanished = append(b.vanished, t)
		}
		if t.live() && t.status != nil {
			s.reportTorrent(t, now, b)
		}
	}
	b.poll = s.pollReport()
}

func (s *state) reportTorrent(t *track, now time.Time, b *batch) {
	key := s.reportOf(t, now)
	beat := key.GetHold() == dlv1.Hold_HOLD_CLIENT_ERROR && now.Sub(t.sentAt) >= s.cfg.Timings.ErrorHeartbeat
	if !proto.Equal(key, t.sent) || t.change != nil || t.completion != nil || beat {
		r := proto.CloneOf(key)
		r.AtUnixMs, r.Done, r.Wanted = now.UnixMilli(), t.status.Done, t.status.Size
		if t.change != nil {
			r.WantedChange = &dlv1.WantedChange{From: runOnWire[t.change.From], To: runOnWire[t.change.To]}
		}
		r.Completion = t.completion
		b.reports, b.reported, b.keys = append(b.reports, r), append(b.reported, t), append(b.keys, key)
	}
	if row := countersOf(t); t.sentTotals == nil || b.totalsSent && !proto.Equal(row, t.sentTotals) {
		b.totals, b.totalled = append(b.totals, row), append(b.totalled, t)
	}
}

// reportOf is t's report without what rides along with a change: the time,
// progress, a reported change and the files.
func (s *state) reportOf(t *track, now time.Time) *dlv1.TorrentReport {
	st := t.status
	r := &dlv1.TorrentReport{
		Id: t.id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Complete: st.Finished,
		Error: st.LocalError, Fault: t.fault, Origin: t.origin(), Run: runOnWire[st.Run],
		QueuePosition: int32(st.QueuePosition), StoppedAtLimit: st.StoppedAtLimit, //nolint:gosec // a queue position fits
		Stalled: stalled(t.lastGrowth, now, s.share.StalledAfter, st.Run == clients.Started && st.Activity == clients.Downloading, st.Finished),
	}
	switch {
	case st.Activity == clients.Checking:
		r.Engine = dlv1.EngineState_ENGINE_STATE_CHECKING
	case st.Finished:
		r.Engine = dlv1.EngineState_ENGINE_STATE_SEEDING
	}
	switch {
	case st.LocalError != "":
		r.Hold = dlv1.Hold_HOLD_CLIENT_ERROR
	case st.Activity == clients.Queued:
		r.Hold = dlv1.Hold_HOLD_SLOT
	}
	if !t.row.Completed && !t.importAt.IsZero() {
		r.ImportAtUnixMs = t.importAt.UnixMilli()
	}
	for i, m := range st.Trackers {
		line := &dlv1.TrackerLine{Index: int32(i), Message: m.Message, Status: dlv1.TrackerStatus_TRACKER_STATUS_WORKING}
		switch {
		case m.Error && announce.Unregistered(m.Message):
			line.Status = dlv1.TrackerStatus_TRACKER_STATUS_UNREGISTERED
		case m.Error:
			line.Status = dlv1.TrackerStatus_TRACKER_STATUS_ERROR
		}
		r.Trackers = append(r.Trackers, line)
	}
	return r
}

func countersOf(t *track) *dlv1.ClientCounters {
	c := t.status.Counters
	row := &dlv1.ClientCounters{
		Id: t.id.Bytes(), Uploaded: c.Uploaded, Downloaded: c.Downloaded,
		ActiveSeconds: int64(c.Active / time.Second), SeedingSeconds: int64(c.Seeding / time.Second),
	}
	if !t.status.Finished {
		row.Done = t.status.Done
	}
	return row
}

// pollReport is the client's poll as the Controller stores it, nil while
// the client has no state yet.
func (s *state) pollReport() *dlv1.ClientPoll {
	if s.reach.state == "" {
		return nil
	}
	p := &dlv1.ClientPoll{
		ClientId: s.cfg.ID.Bytes(), SettingsVersion: s.version, State: pollStateOnWire[s.reach.state],
		SinceUnixMs: s.reach.since.UnixMilli(), Version: s.product, Problem: ProblemOnWire(s.reach.problem),
	}
	return p
}

// commit writes b in one local Write, then marks what it sent and
// publishes, so the seams see a torrent gone only once its Vanished is in
// the outbox.
func (s *state) commit(ctx context.Context, b *batch) {
	poll := b.poll
	if poll != nil && len(b.checks) == 0 && proto.Equal(poll, s.sentPoll) {
		poll = nil
	}
	reports := b.outbox(s.polls(poll, b.checks))
	err := s.cfg.Local.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		for _, r := range reports {
			if err := s.cfg.Holder.Report(ctx, tx, r); err != nil {
				return err
			}
		}
		return s.writeRows(ctx, db.New(tx), b)
	})
	if err != nil {
		if ctx.Err() == nil {
			s.cfg.Log.Error("report the client's torrents", "err", err.Error())
		}
		answer(b.checks, err)
		return
	}
	b.sent()
	if b.totalsSent {
		s.totalsAt = s.last
	}
	if poll != nil {
		s.sentPoll = poll
	}
	s.publish()
	answer(b.checks, nil)
}

// outbox is b's reports in the order they apply, then polls.
func (b *batch) outbox(polls []*dlv1.ClientPoll) []*dlv1.Report {
	var out []*dlv1.Report
	if len(b.reports) > 0 {
		out = append(out, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: b.reports}}})
	}
	for _, t := range b.vanished {
		out = append(out, &dlv1.Report{Kind: &dlv1.Report_Vanished{Vanished: &dlv1.Vanished{Id: t.id.Bytes()}}})
	}
	for i, t := range b.parted {
		out = append(out, &dlv1.Report{Kind: &dlv1.Report_Parted{Parted: &dlv1.Parted{Id: t.id.Bytes(), Freed: b.freed[i]}}})
	}
	if len(b.totals) > 0 {
		out = append(out, &dlv1.Report{Kind: &dlv1.Report_ClientTotals{ClientTotals: &dlv1.ClientTotals{Rows: b.totals}}})
	}
	for _, p := range polls {
		out = append(out, &dlv1.Report{Kind: &dlv1.Report_Client{Client: p}})
	}
	return out
}

// writeRows forgets each parted torrent's row and writes every other row
// that changed.
func (s *state) writeRows(ctx context.Context, q *db.Queries, b *batch) error {
	for _, t := range b.parted {
		if err := q.DeleteExternal(ctx, db.DeleteExternalParams{ClientID: s.cfg.ID, InfoHash: t.hash[:]}); err != nil {
			return err
		}
	}
	for _, t := range b.rows {
		if t.parted || slices.Contains(b.parted, t) {
			continue
		}
		if err := q.PutExternal(ctx, db.PutExternalParams(t.row)); err != nil {
			return err
		}
	}
	return nil
}

// sent marks what b's committed Write sent.
func (b *batch) sent() {
	for i, t := range b.reported {
		t.sent, t.sentAt, t.change, t.completion = b.keys[i], time.UnixMilli(b.reports[i].GetAtUnixMs()), nil, nil
	}
	for _, t := range b.vanished {
		t.told = true
	}
	for _, t := range b.parted {
		t.parted = true
	}
	for i, t := range b.totalled {
		t.sentTotals = b.totals[i]
	}
	for _, t := range b.rows {
		row := t.row
		t.written = &row
	}
}

// polls is the ClientPolls to write: one for each Check's token, or p
// alone when it changed.
func (s *state) polls(p *dlv1.ClientPoll, checks []check) []*dlv1.ClientPoll {
	if len(checks) == 0 {
		if p == nil {
			return nil
		}
		return []*dlv1.ClientPoll{p}
	}
	base := s.pollReport()
	if base == nil {
		base = &dlv1.ClientPoll{ClientId: s.cfg.ID.Bytes(), SettingsVersion: s.version}
	}
	out := make([]*dlv1.ClientPoll, len(checks))
	for i, c := range checks {
		out[i] = proto.CloneOf(base)
		out[i].Check = c.token
	}
	return out
}

func sameRow(r db.DownloadingExternal, written *db.DownloadingExternal) bool {
	if written == nil {
		return false
	}
	w := *written
	return sameString(r.Origin, w.Origin) && r.SentAt == w.SentAt && r.Resends == w.Resends && r.Seen == w.Seen &&
		sameString(r.Applied, w.Applied) && r.FinishedAt == w.FinishedAt && r.Completed == w.Completed
}

func sameString(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

// published is what other goroutines read of the actor, swapped whole
// after each pass and each Update.
type published struct {
	// polled is a successful poll since the actor started.
	polled bool
	// answers is the client's polls succeeding.
	answers bool
	byHash  map[bencode.InfoHash]heldFiles
	folder  string
	live    []*dlv1.Live
	down    int64
	up      int64
}

type heldFiles struct {
	completed bool
	vanished  bool
	// paths are each file as this Node sees it, by index.
	paths []string
}

func (s *state) publish() {
	p := &published{polled: s.polled, answers: s.answers(), byHash: make(map[bencode.InfoHash]heldFiles, len(s.tracks))}
	base := cmp.Or(s.share.Client.GetFolder(), s.defaultFolder)
	if f, _, ok := s.folders.OnNode(base, filepath.Separator); base != "" && ok {
		p.folder = f
	}
	for _, t := range s.sorted() {
		p.byHash[t.hash] = s.heldOf(t)
		if t.live() && t.status != nil {
			st := t.status
			p.live = append(p.live, &dlv1.Live{
				Id: t.id.Bytes(), DownBps: st.Rates.Down, UpBps: st.Rates.Up, Done: st.Done, Wanted: st.Size,
				Peers: int32(st.Rates.Peers), Seeds: int32(st.Rates.Seeds), //nolint:gosec // a torrent's peer count fits
			})
			p.down += st.Rates.Down
			p.up += st.Rates.Up
		}
	}
	s.a.pub.Store(p)
}

func (s *state) heldOf(t *track) heldFiles {
	h := heldFiles{completed: t.row.Completed && t.files != nil, vanished: t.vanished && t.told}
	if t.status == nil || t.files == nil {
		return h
	}
	h.paths = make([]string, len(t.files))
	for _, f := range t.files {
		if path, _, ok := s.folders.OnNode(joinClient(t.status.Folder, f.Path), filepath.Separator); ok && f.Index < len(h.paths) {
			h.paths[f.Index] = path
		}
	}
	return h
}

// answers is the client's polls succeeding now.
func (s *state) answers() bool {
	return s.polled && s.reach.failures == 0 && s.reach.state != clients.SignInRefused
}

func (s *state) inspect(ctx context.Context, id storage.UUID) *dlv1.InspectResponse {
	t := s.byID(id)
	if t == nil {
		return nil
	}
	res := &dlv1.InspectResponse{}
	if t.status != nil && s.client != nil && s.answers() {
		if files, err := s.files(ctx, t); err == nil {
			t.files = files
		}
	}
	for _, f := range t.files {
		res.Files = append(res.Files, &dlv1.FileProgress{Index: int32(f.Index), Done: f.Done}) //nolint:gosec // a torrent's file index fits
	}
	if t.status != nil {
		for _, line := range s.reportOf(t, time.Now()).GetTrackers() {
			res.Trackers = append(res.Trackers, &dlv1.TrackerDetail{Index: line.GetIndex(), Status: line.GetStatus(), Message: line.GetMessage()})
		}
	}
	return res
}

func (s *state) act(ctx context.Context, act clients.Act, ids []storage.UUID) []*dlv1.ActResult {
	out := make([]*dlv1.ActResult, len(ids))
	var hashes []bencode.InfoHash
	var asked []int
	for i, id := range ids {
		out[i] = &dlv1.ActResult{Id: id.Bytes()}
		switch t := s.byID(id); {
		case t == nil || !t.live():
			out[i].Refused = dlv1.Refusal_REFUSAL_NOT_HELD
		case !s.answers() || s.client == nil:
			out[i].Refused = dlv1.Refusal_REFUSAL_CLIENT_AWAY
		default:
			hashes, asked = append(hashes, t.hash), append(asked, i)
		}
	}
	if len(hashes) == 0 {
		return out
	}
	if err := s.client.Act(ctx, act, hashes); err != nil {
		s.cfg.Log.Warn("announce or recheck torrents in the client", "err", err.Error())
		for _, i := range asked {
			out[i].Refused = dlv1.Refusal_REFUSAL_CLIENT_AWAY
		}
	}
	return out
}
