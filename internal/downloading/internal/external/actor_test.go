package external_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/scenario"
	faketransmission "github.com/jatassi/slipstream/fakes/transmission"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloading"
	"github.com/jatassi/slipstream/internal/downloading/internal/external"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/torrent/clients"
	"github.com/jatassi/slipstream/internal/torrent/clients/transmission"
	"github.com/jatassi/slipstream/internal/transfers/source"
)

const (
	node = "Media server"
	// size is each release's payload: ten seconds at the fake's rate.
	size       = 1000
	checkEvery = 10 * time.Second
)

var timings = external.Timings{
	RetryMin: 5 * time.Second, RetryMax: 5 * time.Minute, RefusedRetry: 30 * time.Minute, Grace: time.Minute,
	ResendWait: 2 * time.Minute, MissingShare: 50, MissingFloor: 3, Trackers: time.Minute, Call: 30 * time.Second,
	LiveFloor: 2 * time.Second, LiveSample: time.Second, TotalsReport: 5 * time.Minute,
}

type release struct {
	name     string
	hash     bencode.InfoHash
	metainfo []byte
	id       storage.UUID
}

// holder records every report the actor writes, and calls onReport as it
// does, inside the Write.
type holder struct {
	external.Holder

	mu       sync.Mutex
	reports  []*dlv1.Report
	onReport func(*dlv1.Report)
}

func (h *holder) Report(_ context.Context, _ *storage.Tx[storage.Local], r *dlv1.Report) error {
	h.mu.Lock()
	h.reports = append(h.reports, r)
	on := h.onReport
	h.mu.Unlock()
	if on != nil {
		on(r)
	}
	return nil
}

func (h *holder) reported() []*dlv1.Report {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.reports)
}

// control is the Controller as the actor fetches from it.
type control struct {
	downloadingv1connect.UnimplementedDownloadingControlServiceHandler

	mu       sync.Mutex
	torrents map[storage.UUID]*dlv1.TorrentResponse
	password string
}

func (c *control) Torrent(_ context.Context, req *dlv1.TorrentRequest) (*dlv1.TorrentResponse, error) {
	id, err := storage.UUIDFromBytes(req.GetId())
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	res, ok := c.torrents[id]
	if !ok {
		return nil, rpc.Errorf(rpc.CodeNotFound, "no torrent %s", id)
	}
	return res, nil
}

func (c *control) ClientSecrets(context.Context, *dlv1.ClientSecretsRequest) (*dlv1.ClientSecretsResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return &dlv1.ClientSecretsResponse{Secrets: &dlv1.Secrets{Password: c.password}}, nil
}

func (c *control) setPassword(p string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.password = p
}

// signIns counts the requests the client refused its sign-in to.
type signIns struct {
	inner http.RoundTripper
	mu    sync.Mutex
	n     int
}

func (s *signIns) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := s.inner.RoundTrip(req)
	if err == nil && res.StatusCode == http.StatusUnauthorized {
		s.mu.Lock()
		s.n++
		s.mu.Unlock()
	}
	return res, err //nolint:wrapcheck // the transport's own error
}

func (s *signIns) refused() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.n
}

// rig is one Node's fake Transmission, the actor for it, and the
// Controller the actor reports to.
type rig struct {
	t       *testing.T
	daemon  faketransmission.Daemon
	served  faketest.Served
	signIns *signIns
	rels    map[string]release
	local   *storage.DB[storage.Local]
	ctl     *control
	h       *holder
	id      storage.UUID
	share   external.Share
	actor   *external.Actor
}

func newRig(t *testing.T, version string, d faketransmission.Daemon) *rig {
	t.Helper()
	d.Node, d.Version, d.Dir, d.Folder = node, version, t.TempDir(), "/downloads"
	if d.Rate == 0 {
		d.Rate = 100
	}
	catalog := faketest.Catalog(t, size, "orchard", "lantern", "harbor")
	r := &rig{
		t: t, daemon: d, rels: map[string]release{}, local: storagetest.Local(t, downloading.Schema()),
		ctl: &control{torrents: map[storage.UUID]*dlv1.TorrentResponse{}, password: d.Password}, h: &holder{}, id: storage.NewUUID(),
	}
	for _, rel := range catalog.Releases {
		x := release{name: rel.Name, hash: rel.InfoHash, metainfo: rel.Metainfo, id: storage.NewUUID()}
		r.rels[rel.Name] = x
		r.ctl.torrents[x.id] = &dlv1.TorrentResponse{Metainfo: x.metainfo, ContentType: "Movies"}
	}
	r.served = faketest.InMemory(t, faketransmission.Playing(d), scenario.Scenario{Seed: "transmission"},
		faketest.WithReleases(catalog), faketest.As(node))
	r.signIns = &signIns{inner: r.served.Client.Transport}
	r.share = external.Share{
		Client: &dlv1.Client{
			Id: r.id.Bytes(), Kind: dlv1.Kind_KIND_TRANSMISSION, Name: "Transmission", Address: r.served.Address.URL("/transmission/rpc"),
			Username: d.Username, Label: clients.DefaultLabel, CheckEverySeconds: int64(checkEvery / time.Second), On: true, SettingsVersion: 1,
			ClientFolders: []*dlv1.ClientFolder{{Client: "/downloads", Node: d.Dir}},
		},
		StalledAfter: 10 * time.Minute, UnregisteredGap: 30 * time.Minute, MayStart: true,
	}
	return r
}

func inEachVersion(t *testing.T, test func(t *testing.T, version string)) {
	for _, v := range faketransmission.Versions {
		t.Run(v, func(t *testing.T) { synctest.Test(t, func(t *testing.T) { test(t, v) }) })
	}
}

func (r *rig) want(names ...string) {
	r.share.Torrents = nil
	for i, name := range names {
		rel := r.rels[name]
		r.share.Torrents = append(r.share.Torrents, &dlv1.Torrent{Id: rel.id.Bytes(), InfoHash: rel.hash[:], Wanted: dlv1.Wanted_WANTED_STARTED, Order: int64(i), ClientId: r.id.Bytes()})
	}
}

func (r *rig) start() {
	r.t.Helper()
	r.actor = external.Start(context.Background(), external.Config{
		ID: r.id, Kind: clients.Transmission, Open: transmission.Open, HTTP: &http.Client{Transport: r.signIns},
		Subfolder: func(name string) string { return name },
		Links:     func(string) (uint64, error) { return 1, nil },
		Timings:   timings, Local: r.local, Holder: r.h, Control: r.ctl,
		Rand: rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Log:  slog.New(slog.NewTextHandler(r.t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug})),
	}, r.snapshot())
	r.t.Cleanup(r.actor.Halt)
	synctest.Wait()
}

func (r *rig) update() {
	r.actor.Update(r.snapshot())
	synctest.Wait()
}

// snapshot is the share as a new intent carries it, apart from the rig's.
func (r *rig) snapshot() external.Share {
	s := r.share
	s.Client = proto.CloneOf(s.Client)
	s.Torrents, s.Discards = nil, nil
	for _, t := range r.share.Torrents {
		s.Torrents = append(s.Torrents, proto.CloneOf(t))
	}
	for _, d := range r.share.Discards {
		s.Discards = append(s.Discards, proto.CloneOf(d))
	}
	return s
}

// session is a session of the test's own with the fake, as a person using
// Transmission's web UI has.
func (r *rig) session() clients.Client {
	a, err := clients.ParseAddress(r.served.Address.URL("/transmission/rpc"), clients.Transmission)
	if err != nil {
		r.t.Fatal(err)
	}
	return transmission.Open(clients.Session{
		HTTP: r.served.Client, Address: a, Username: r.daemon.Username, Secrets: clients.Secrets{Password: r.daemon.Password},
		Call: 30 * time.Second, Log: slog.New(slog.DiscardHandler),
	})
}

// inClient is orchard's torrent as the fake lists it, false when it
// doesn't.
func (r *rig) inClient() (clients.Status, bool) {
	r.t.Helper()
	h := r.rels["orchard"].hash
	got, err := r.session().Poll(r.t.Context(), []bencode.InfoHash{h})
	if err != nil {
		r.t.Fatal(err)
	}
	s, ok := got[h]
	return s, ok
}

func (r *rig) cue(name string, args map[string]any) {
	r.t.Helper()
	args["node"] = node
	r.served.Cue(r.t, "transmission-"+name, args)
}

func (r *rig) sleep(d time.Duration) {
	time.Sleep(d)
	synctest.Wait()
}

// reportsOf is every report of the release's torrent, oldest first.
func (r *rig) reportsOf(name string) []*dlv1.TorrentReport {
	var out []*dlv1.TorrentReport
	for _, rep := range r.h.reported() {
		for _, tr := range rep.GetTorrents().GetReports() {
			if bytes.Equal(tr.GetId(), r.rels[name].id.Bytes()) {
				out = append(out, tr)
			}
		}
	}
	return out
}

func (r *rig) latest(name string) *dlv1.TorrentReport {
	r.t.Helper()
	reps := r.reportsOf(name)
	if len(reps) == 0 {
		r.t.Fatalf("no report of %s", name)
	}
	return reps[len(reps)-1]
}

// completion is orchard's report with its Completion, false before one.
func (r *rig) completion() (*dlv1.TorrentReport, bool) {
	for _, rep := range r.reportsOf("orchard") {
		if rep.GetCompletion() != nil {
			return rep, true
		}
	}
	return nil, false
}

func (r *rig) polls() []*dlv1.ClientPoll {
	var out []*dlv1.ClientPoll
	for _, rep := range r.h.reported() {
		if p := rep.GetClient(); p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (r *rig) lastPoll() *dlv1.ClientPoll {
	r.t.Helper()
	ps := r.polls()
	if len(ps) == 0 {
		r.t.Fatal("no ClientPoll")
	}
	return ps[len(ps)-1]
}

func (r *rig) vanished() []storage.UUID {
	var out []storage.UUID
	for _, rep := range r.h.reported() {
		if v := rep.GetVanished(); v != nil {
			id, _ := storage.UUIDFromBytes(v.GetId())
			out = append(out, id)
		}
	}
	return out
}

func TestTheActorPollsAtCheckEveryAndCompletesOnlyAfterTheImportDelay(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{})
		r.want("orchard")
		r.share.Client.ImportDelaySeconds = 60
		start := time.Now()
		r.start()
		r.sleep(30 * time.Second)

		importAt := r.waitingImport(start)
		r.sleep(time.Until(importAt) + checkEvery)
		done, ok := r.completion()
		if !ok {
			t.Fatalf("no Completion %s after the Import delay ended", time.Until(importAt))
		}
		if at := time.UnixMilli(done.GetAtUnixMs()); at.Before(importAt) {
			t.Errorf("the Completion went at %s, before the import time %s", at.Sub(start), importAt.Sub(start))
		}
		if files := done.GetCompletion().GetFiles(); len(files) != 1 || files[0].GetPath() != "orchard.mkv" || files[0].GetSize() != size {
			t.Errorf("completion files %v, want orchard.mkv of %d bytes", files, size)
		}
		if got := done.GetOrigin(); got != dlv1.Origin_ORIGIN_ADDED {
			t.Errorf("origin %s, want added", got)
		}
		for _, rep := range r.reportsOf("orchard") {
			if since := time.UnixMilli(rep.GetAtUnixMs()).Sub(start); since%checkEvery != 0 {
				t.Errorf("a report %s after the start, want each at a poll, every %s", since, checkEvery)
			}
		}
		r.seamsAnswer()
	})
}

// waitingImport is when orchard, finished, imports: 60s after its 10s
// download, with no Completion and no files seam before.
func (r *rig) waitingImport(start time.Time) time.Time {
	r.t.Helper()
	finished := slices.IndexFunc(r.reportsOf("orchard"), func(rep *dlv1.TorrentReport) bool { return rep.GetComplete() })
	if finished < 0 {
		r.t.Fatalf("no complete report 30s in: %v", r.reportsOf("orchard"))
	}
	rep := r.reportsOf("orchard")[finished]
	importAt := time.UnixMilli(rep.GetImportAtUnixMs())
	if rep.GetEngine() != dlv1.EngineState_ENGINE_STATE_SEEDING || importAt.Before(start.Add(70*time.Second)) || importAt.After(start.Add(80*time.Second)) {
		r.t.Fatalf("finished: engine %s, imports at %s; want seeding, importing 60s after the 10s download", rep.GetEngine(), importAt.Sub(start))
	}
	if _, ok := r.completion(); ok {
		r.t.Fatal("the Completion went before the Import delay ended")
	}
	if _, ok := r.actor.Path(r.rels["orchard"].hash, 0); ok {
		r.t.Error("the files seam answered before the Completion")
	}
	return importAt
}

// seamsAnswer checks that Path and Open answer orchard's file as it lies
// on this Node, through the Client folders.
func (r *rig) seamsAnswer() {
	r.t.Helper()
	path, ok := r.actor.Path(r.rels["orchard"].hash, 0)
	if want := filepath.Join(r.daemon.Dir, "Movies", "orchard.mkv"); !ok || path != want {
		r.t.Fatalf("Path = %q, %v; want %q through the Client folders", path, ok, want)
	}
	if info, err := os.Stat(path); err != nil || info.Size() != size {
		r.t.Errorf("the file at Path: %v, %v", info, err)
	}
	if got, err := r.actor.Open(source.Torrent{Hash: r.rels["orchard"].hash}); err != nil || got != path {
		r.t.Errorf("Open = %q, %v; want %q", got, err, path)
	}
}

func TestAVanishedTorrentIsInTheOutboxBeforeTheSeamsSeeItGone(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{})
		r.want("orchard")
		r.start()
		r.sleep(30 * time.Second)
		if _, ok := r.completion(); !ok {
			t.Fatal("orchard didn't complete")
		}
		ref := source.Torrent{Hash: r.rels["orchard"].hash}
		var openedThen error
		r.h.mu.Lock()
		r.h.onReport = func(rep *dlv1.Report) {
			if rep.GetVanished() != nil {
				_, openedThen = r.actor.Open(ref)
			}
		}
		r.h.mu.Unlock()

		r.cue("remove", map[string]any{"release": "orchard"})
		r.sleep(checkEvery)
		if len(r.vanished()) != 0 {
			t.Fatal("one poll without the torrent reported it vanished")
		}
		r.sleep(checkEvery)
		if got := r.vanished(); !slices.Equal(got, []storage.UUID{r.rels["orchard"].id}) {
			t.Fatalf("vanished %v after two polls without it, want orchard", got)
		}
		if isGone(openedThen) {
			t.Error("the seams saw the torrent gone before its Vanished was in the outbox")
		}
		if _, err := r.actor.Open(ref); !isGone(err) {
			t.Errorf("Open once vanished = %v, want Gone", err)
		}
		if _, ok := r.actor.Path(ref.Hash, 0); ok {
			t.Error("Path answered for a vanished torrent")
		}
	})
}

func isGone(err error) bool {
	a, ok := errors.AsType[*source.Absent](err)
	return ok && a.Reason == source.Gone
}

func TestATorrentThatFinishedWhileTheNodeWasDownCompletesOnTheFirstPoll(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{})
		r.want("orchard")
		r.start()
		r.sleep(time.Second)
		r.actor.Halt()
		r.sleep(time.Minute)

		r.h = &holder{}
		restarted := time.Now()
		r.start()
		done, ok := r.completion()
		if !ok || !time.UnixMilli(done.GetAtUnixMs()).Equal(restarted) {
			t.Fatalf("after the restart: completion %v, want one from the first poll", done)
		}
		if got := done.GetOrigin(); got != dlv1.Origin_ORIGIN_ADDED {
			t.Errorf("origin %s after the restart, want added from the local set", got)
		}
	})
}

func TestARestartKeepsTheResendItAlreadyMade(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{})
		r.want("orchard")
		r.cue("drop-add", map[string]any{})
		r.start()
		r.sleep(timings.ResendWait - time.Second)
		r.cue("drop-add", map[string]any{})
		r.sleep(checkEvery)
		r.actor.Halt()

		r.start()
		if p := r.lastPoll(); p.GetState() != dlv1.PollState_POLL_STATE_RESPONDING {
			t.Fatalf("after the restart: %s, want responding while the resend waits", p.GetState())
		}
		r.sleep(timings.ResendWait)
		p := r.lastPoll()
		if p.GetState() != dlv1.PollState_POLL_STATE_NEEDS_ATTENTION || p.GetProblem().GetCause() != dlv1.Cause_CAUSE_NOT_KEPT {
			t.Fatalf("a resend made before the restart missed again: %s %v, want needs attention, not kept", p.GetState(), p.GetProblem())
		}
		if _, ok := r.inClient(); ok {
			t.Error("the restarted actor sent the torrent again, past its one resend")
		}
	})
}

func TestADroppedAddIsSentOnceMore(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{})
		r.want("orchard")
		r.cue("drop-add", map[string]any{})
		r.start()
		r.sleep(timings.ResendWait + checkEvery)
		if _, ok := r.inClient(); !ok {
			t.Fatal("the dropped add wasn't sent once more")
		}
		if p := r.lastPoll(); p.GetState() != dlv1.PollState_POLL_STATE_RESPONDING {
			t.Errorf("after the resend: %s, want responding", p.GetState())
		}
	})
}

func TestAPauseInTheClientIsReportedAndAResumeInSlipStreamReachesIt(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{Rate: 10})
		r.want("orchard")
		r.start()
		r.sleep(checkEvery)
		if err := r.session().Act(t.Context(), clients.Pause, []bencode.InfoHash{r.rels["orchard"].hash}); err != nil {
			t.Fatal(err)
		}
		r.sleep(checkEvery)
		rep := r.latest("orchard")
		if wc := rep.GetWantedChange(); rep.GetRun() != dlv1.Wanted_WANTED_PAUSED || wc.GetFrom() != dlv1.Wanted_WANTED_STARTED || wc.GetTo() != dlv1.Wanted_WANTED_PAUSED {
			t.Fatalf("after a pause in Transmission: run %s, change %v; want paused, from started to paused", rep.GetRun(), wc)
		}
		r.sleep(checkEvery)
		if s, _ := r.inClient(); s.Run != clients.Paused {
			t.Fatal("the stale intent resumed the torrent before it showed the pause")
		}

		r.share.Torrents[0].Wanted = dlv1.Wanted_WANTED_PAUSED
		r.update()
		r.share.Torrents[0].Wanted = dlv1.Wanted_WANTED_STARTED
		r.update()
		r.sleep(checkEvery)
		if s, _ := r.inClient(); s.Run != clients.Started {
			t.Errorf("after a resume in SlipStream the client has it %d, want started", s.Run)
		}
		if rep := r.latest("orchard"); rep.GetRun() != dlv1.Wanted_WANTED_STARTED {
			t.Errorf("after the resume: run %s, want started", rep.GetRun())
		}
	})
}

func TestARefusedPasswordStopsPollingUntilTheSettingsChangeOrCheckNow(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{Username: "jackson", Password: "attic"})
		r.ctl.setPassword("orchard")
		r.start()
		if p := r.lastPoll(); p.GetState() != dlv1.PollState_POLL_STATE_SIGN_IN_REFUSED {
			t.Fatalf("with a wrong password: %s, want sign-in refused", p.GetState())
		}
		r.sleep(time.Hour)
		if n := r.signIns.refused(); n != 1 {
			t.Errorf("an hour after a refused sign-in: %d refused, want the one attempt", n)
		}

		if err := r.actor.Check(t.Context(), []byte("check-1")); err != nil {
			t.Fatal(err)
		}
		if p := r.lastPoll(); p.GetState() != dlv1.PollState_POLL_STATE_SIGN_IN_REFUSED || string(p.GetCheck()) != "check-1" {
			t.Errorf("Check now: %s with token %q, want refused again with check-1", p.GetState(), p.GetCheck())
		}

		r.ctl.setPassword("attic")
		r.share.Client.SettingsVersion = 2
		r.update()
		p := r.lastPoll()
		if p.GetState() != dlv1.PollState_POLL_STATE_RESPONDING || p.GetSettingsVersion() != 2 {
			t.Errorf("after the password's fix: %s at settings %d, want responding at 2", p.GetState(), p.GetSettingsVersion())
		}
		if got := p.GetVersion(); got == "" {
			t.Error("the poll names no version")
		}
	})
}

func TestAClientIsNotRespondingOnlyAfterTheGraceAndKeepsItsTorrents(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{Rate: 10})
		r.want("orchard")
		r.start()
		r.sleep(checkEvery)
		reports := len(r.reportsOf("orchard"))
		if _, err := r.served.Faults.Set(fault.Rule{ID: "down", Target: fault.Target(faketransmission.Fake.Name), Fault: fault.Reset{}}); err != nil {
			t.Fatal(err)
		}
		down := time.Now()
		r.sleep(timings.Grace - time.Second)
		if p := r.lastPoll(); p.GetState() != dlv1.PollState_POLL_STATE_RESPONDING {
			t.Fatalf("inside the grace: %s, want responding", p.GetState())
		}
		r.sleep(5 * time.Minute)
		p := r.lastPoll()
		if p.GetState() != dlv1.PollState_POLL_STATE_NOT_RESPONDING || time.UnixMilli(p.GetSinceUnixMs()).Before(down) || time.UnixMilli(p.GetSinceUnixMs()).After(down.Add(checkEvery)) {
			t.Fatalf("past the grace: %s since %s, want not responding since the first failed poll", p.GetState(), time.UnixMilli(p.GetSinceUnixMs()).Sub(down))
		}
		if n := len(r.reportsOf("orchard")); n != reports || len(r.vanished()) != 0 {
			t.Errorf("failed polls changed the torrent: %d reports, was %d; vanished %v", n, reports, r.vanished())
		}

		r.served.Faults.Clear("down")
		r.sleep(timings.RetryMax)
		if p := r.lastPoll(); p.GetState() != dlv1.PollState_POLL_STATE_RESPONDING {
			t.Errorf("once restored: %s, want responding", p.GetState())
		}
	})
}

func TestMostTorrentsMissingAtOnceMarksNoneRemoved(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{})
		r.want("orchard", "lantern", "harbor")
		r.start()
		r.sleep(checkEvery)
		r.cue("empty", map[string]any{})
		r.sleep(3 * checkEvery)
		p := r.lastPoll()
		if p.GetState() != dlv1.PollState_POLL_STATE_NEEDS_ATTENTION || p.GetProblem().GetCause() != dlv1.Cause_CAUSE_MOST_MISSING || p.GetProblem().GetMissing() != 3 {
			t.Errorf("an emptied client: %s %v, want needs attention with 3 missing", p.GetState(), p.GetProblem())
		}
		if got := r.vanished(); len(got) != 0 {
			t.Errorf("an emptied client vanished %v, want none", got)
		}
	})
}

func TestAnAttachedTorrentIsNeverRemovedFromTheClient(t *testing.T) {
	inEachVersion(t, func(t *testing.T, version string) {
		r := newRig(t, version, faketransmission.Daemon{})
		rel := r.rels["orchard"]
		if err := r.session().Add(t.Context(), clients.Add{Metainfo: rel.metainfo, Hash: rel.hash, Label: mustLabel(t)}); err != nil {
			t.Fatal(err)
		}
		r.want("orchard")
		r.start()
		if got := r.latest("orchard").GetOrigin(); got != dlv1.Origin_ORIGIN_ATTACHED {
			t.Fatalf("a torrent already in the client: origin %s, want attached", got)
		}
		r.share.Torrents = nil
		r.share.Discards = []*dlv1.Discard{{Id: rel.id.Bytes(), InfoHash: rel.hash[:], DeleteFiles: true, ClientId: r.id.Bytes()}}
		r.update()
		r.sleep(checkEvery)
		var parted []*dlv1.Parted
		for _, rep := range r.h.reported() {
			if p := rep.GetParted(); p != nil {
				parted = append(parted, p)
			}
		}
		if len(parted) != 1 || parted[0].GetFreed() != 0 {
			t.Fatalf("parted %v, want the attached torrent parted freeing nothing", parted)
		}
		if _, ok := r.inClient(); !ok {
			t.Error("removing an attached torrent removed it from the client")
		}
	})
}

func mustLabel(t *testing.T) clients.Label {
	t.Helper()
	l, err := clients.ParseLabel(clients.DefaultLabel, clients.Transmission)
	if err != nil {
		t.Fatal(err)
	}
	return l
}
