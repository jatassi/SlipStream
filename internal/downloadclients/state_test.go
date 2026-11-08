package downloadclients_test

import (
	"testing"
	"time"

	dc "github.com/jatassi/slipstream/internal/downloadclients"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
)

func report(engine dc.EngineState, change func(*dc.Report)) *dc.Report {
	r := &dc.Report{Engine: engine}
	if change != nil {
		change(r)
	}
	return r
}

func held(h dc.Hold) func(*dc.Report) { return func(r *dc.Report) { r.Hold = h } }

func lines(statuses ...dc.TrackerStatus) func(*dc.Report) {
	return func(r *dc.Report) {
		for i, s := range statuses {
			r.Trackers = append(r.Trackers, dc.TrackerLine{Index: i, Status: s})
		}
	}
}

func complete(r *dc.Report) { r.Complete = true }

func run(w dc.Wanted) func(*dc.Report) { return func(r *dc.Report) { r.Run = w } }

func stalled(r *dc.Report) { r.Stalled = true }

func TestStateOf(t *testing.T) {
	reached := map[dc.State]bool{}
	for _, c := range []struct {
		name  string
		facts dc.Facts
		want  dc.State
	}{
		{"removed and not yet confirmed", dc.Facts{Removing: true, Wanted: dc.Started, Report: report(dc.EngineSeeding, complete)}, dc.StateRemoving},
		{"failed and removing reads removing", dc.Facts{Removing: true, Wanted: dc.Started, Report: report(dc.EngineFailed, nil)}, dc.StateRemoving},
		{"the engine failed it", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineFailed, nil)}, dc.StateFailed},
		{"failed outranks a tracker error", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineFailed, lines(dc.TrackerError))}, dc.StateFailed},
		{"the only tracker errs", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, lines(dc.TrackerError))}, dc.StateTrackerErr},
		{"paused and unregistered reads tracker-error", dc.Facts{Wanted: dc.Paused, Report: report(dc.EngineSeeding, lines(dc.TrackerUnregistered))}, dc.StateTrackerErr},
		{"a working tier serves the torrent", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, lines(dc.TrackerError, dc.TrackerWorking))}, dc.StateDownloading},
		{"an error beside an untried tracker", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, lines(dc.TrackerError, dc.TrackerNotContacted))}, dc.StateTrackerErr},
		{"wanted paused while the Node still runs it", dc.Facts{Wanted: dc.Paused, Report: report(dc.EngineDownloading, nil)}, dc.StatePaused},
		{"wanted paused before the first report", dc.Facts{Wanted: dc.Paused}, dc.StatePaused},
		{"the resume hasn't reached the Node", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, func(r *dc.Report) { complete(r); r.Hold = dc.HoldWanted })}, dc.StatePaused},
		{"waiting out a window or the start spread", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldWindow))}, dc.StatePaused},
		{"paused for disk", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldDiskFull))}, dc.StatePaused},
		{"an error in Transmission", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldClientError))}, dc.StatePaused},
		{"paused in Transmission before the Controller heard", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, run(dc.Paused))}, dc.StatePaused},
		{"Transmission stopped it at its seeding limit", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, func(r *dc.Report) { complete(r); r.Run, r.StoppedAtLimit = dc.Paused, true })}, dc.StatePaused},
		{"running in Transmission", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, run(dc.Started))}, dc.StateDownloading},
		{"wanted paused outranks checking", dc.Facts{Wanted: dc.Paused, Report: report(dc.EngineChecking, nil)}, dc.StatePaused},
		{"a recheck that then resumes", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineChecking, func(r *dc.Report) { r.Until = time.Date(2026, 10, 19, 14, 32, 0, 0, time.UTC) })}, dc.StateChecking},
		{"checking outranks waiting to check", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineChecking, held(dc.HoldCheckSlot))}, dc.StateChecking},
		{"a new Torrent the Node hasn't reported", dc.Facts{Wanted: dc.Started}, dc.StateQueued},
		{"no free download slot", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldSlot))}, dc.StateQueued},
		{"waiting to check", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldCheckSlot))}, dc.StateQueued},
		{"the disk guard holds new downloads", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldDiskGuard))}, dc.StateQueued},
		{"the seedbox is draining", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldDraining))}, dc.StateQueued},
		{"the Node is out of step", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldController))}, dc.StateQueued},
		{"queued outranks stalled", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, func(r *dc.Report) { stalled(r); r.Hold = dc.HoldSlot })}, dc.StateQueued},
		{"no payload for the stall time", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, stalled)}, dc.StateStalled},
		{"fetching data", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, lines(dc.TrackerWorking))}, dc.StateDownloading},
		{"a tracker still updating", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, lines(dc.TrackerUpdating))}, dc.StateDownloading},
		{"complete and running", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, complete)}, dc.StateSeeding},
	} {
		got := dc.StateOf(c.facts)
		if got != c.want {
			t.Errorf("%s: StateOf = %s, want %s", c.name, got, c.want)
		}
		reached[got] = true
	}
	for _, s := range dc.States {
		if !reached[s] {
			t.Errorf("no row reaches %s", s)
		}
	}
}

func TestTabsOf(t *testing.T) {
	const (
		all         = dc.TabAll
		downloading = dc.TabAll | dc.TabDownloading
		seeding     = dc.TabAll | dc.TabSeeding
	)
	for _, c := range []struct {
		name  string
		facts dc.Facts
		want  dc.Tab
	}{
		{"downloading", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, nil)}, downloading},
		{"stalled is a problem too", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, stalled)}, downloading | dc.TabProblems},
		{"queued", dc.Facts{Wanted: dc.Started}, downloading},
		{"checking an incomplete torrent", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineChecking, nil)}, downloading},
		{"paused before completing", dc.Facts{Wanted: dc.Paused, Report: report(dc.EngineDownloading, nil)}, downloading},
		{"seeding", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, complete)}, seeding},
		{"checking a complete torrent", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineChecking, complete)}, seeding},
		{"paused after completing", dc.Facts{Wanted: dc.Paused, Report: report(dc.EngineSeeding, complete)}, seeding},
		{"failed", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineFailed, nil)}, all | dc.TabProblems},
		{"tracker error", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, func(r *dc.Report) { complete(r); lines(dc.TrackerError)(r) })}, all | dc.TabProblems},
		{"removing", dc.Facts{Removing: true, Wanted: dc.Started, Report: report(dc.EngineFailed, nil)}, all},
	} {
		if got := dc.TabsOf(c.facts, dc.StateOf(c.facts)); got != c.want {
			t.Errorf("%s: TabsOf = %04b, want %04b", c.name, got, c.want)
		}
	}
}

func TestSeverityOf(t *testing.T) {
	for _, c := range []struct {
		name  string
		facts dc.Facts
		want  dc.Severity
	}{
		{"failed", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineFailed, nil)}, dc.SeverityFailure},
		{"a tracker says unregistered", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, lines(dc.TrackerError, dc.TrackerUnregistered))}, dc.SeverityFailure},
		{"any other tracker error", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, lines(dc.TrackerError))}, dc.SeverityWarning},
		{"stalled", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, stalled)}, dc.SeverityWarning},
		{"paused for disk", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldDiskFull))}, dc.SeverityWarning},
		{"held by the disk guard", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldDiskGuard))}, dc.SeverityWarning},
		{"an error in Transmission", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldClientError))}, dc.SeverityWarning},
		{"paused in Transmission", dc.Facts{Wanted: dc.Paused, Report: report(dc.EngineDownloading, run(dc.Paused))}, dc.SeverityNone},
		{"paused by the Admin", dc.Facts{Wanted: dc.Paused, Report: report(dc.EngineDownloading, held(dc.HoldWanted))}, dc.SeverityNone},
		{"waiting for a slot", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, held(dc.HoldSlot))}, dc.SeverityNone},
		{"downloading", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineDownloading, nil)}, dc.SeverityNone},
		{"seeding", dc.Facts{Wanted: dc.Started, Report: report(dc.EngineSeeding, complete)}, dc.SeverityNone},
		{"a failed torrent being removed", dc.Facts{Removing: true, Wanted: dc.Started, Report: report(dc.EngineFailed, nil)}, dc.SeverityNone},
	} {
		if got := dc.SeverityOf(c.facts, dc.StateOf(c.facts)); got != c.want {
			t.Errorf("%s: SeverityOf = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestReachOf(t *testing.T) {
	polled := func(state tc.PollState) *dc.Poll { return &dc.Poll{State: state, Since: reportedAt} }
	reached := map[dc.ClientReach]bool{}
	for _, c := range []struct {
		name   string
		client dc.External
		inStep bool
		want   dc.ClientReach
	}{
		{"answering polls", dc.External{On: true, Poll: polled(tc.Responding)}, true, dc.ReachResponding},
		{"not answering", dc.External{On: true, Poll: polled(tc.NotResponding)}, true, dc.ReachNotResponding},
		{"refusing the password", dc.External{On: true, Poll: polled(tc.SignInRefused)}, true, dc.ReachSignInRefused},
		{"most torrents missing", dc.External{On: true, Poll: polled(tc.NeedsAttention)}, true, dc.ReachNeedsAttention},
		{"no poll yet", dc.External{On: true}, true, dc.ReachUnchecked},
		{"its Node offline, whatever it last said", dc.External{On: true, Poll: polled(tc.NotResponding)}, false, dc.ReachNodeOffline},
		{"turned off on an offline Node", dc.External{Poll: polled(tc.Responding)}, false, dc.ReachOff},
		{"turned off before a poll", dc.External{}, true, dc.ReachOff},
	} {
		got := dc.ReachOf(c.client, c.inStep)
		if got != c.want {
			t.Errorf("%s: ReachOf = %s, want %s", c.name, got, c.want)
		}
		reached[got] = true
	}
	if len(reached) != 7 {
		t.Errorf("the cases reach %d of the 7 reaches", len(reached))
	}
}
