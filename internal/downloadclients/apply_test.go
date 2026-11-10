package downloadclients_test

import (
	"bytes"
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
)

// mediaServer is Transmission on Media server, in step, with the
// Controller's readers recorded.
type mediaServer struct {
	lib    library
	node   role.NodeID
	client dc.ClientID
	c      *dc.Clients
	heard  *heard
	cards  *told
	uses   uses
}

func newMediaServer(t *testing.T) *mediaServer {
	t.Helper()
	lib := open(t)
	node := addNode(t, lib, "Media server")
	s := &mediaServer{
		lib: lib, node: node, heard: &heard{}, cards: &told{}, uses: uses{},
		client: mustImport(t, lib, secrettest.Sealer(t), transmission(t, node, "Transmission", "http://192.168.1.20:9091")),
	}
	s.c = dc.New(dc.Config{
		Library: lib, Touch: s.heard.touch, Changed: s.heard, CardsChanged: s.cards, Uses: s.uses, Blocklist: s.heard, Seeding: seeder{},
		Roster: &roster{serving: []role.NodeID{node}}, Log: slog.New(slog.DiscardHandler),
		Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return &client{} },
		Live:   dc.LiveConfig{Watched: func() []role.NodeID { return nil }, Event: &admin{}, Sample: liveSample},
	})
	return s
}

// send is a Grab of name routed to Transmission.
func (s *mediaServer) send(t *testing.T, name string) dc.ID {
	t.Helper()
	m, raw := dotTorrent(t, name)
	id, _ := mustAttach(t, s.lib, dc.Want{Target: dc.Target{Node: s.node, Client: &s.client}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	return id
}

// report applies rep from node and lets the readers hear of it.
func (s *mediaServer) report(t *testing.T, node role.NodeID, rep *dlv1.Report) {
	t.Helper()
	write(t, s.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return s.c.Apply(ctx, tx, node, rep) })
	s.c.Applied(node, string(role.Downloading))
}

func (s *mediaServer) torrent(t *testing.T, r *dlv1.TorrentReport) *dlv1.Report {
	t.Helper()
	r.AtUnixMs = reportedAt.UnixMilli()
	return &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{r}}}}
}

func (s *mediaServer) poll(version int64, state dlv1.PollState, problem *dlv1.Problem, check []byte) *dlv1.Report {
	return &dlv1.Report{Kind: &dlv1.Report_Client{Client: &dlv1.ClientPoll{
		ClientId: s.client.Bytes(), SettingsVersion: version, State: state, SinceUnixMs: reportedAt.UnixMilli(),
		Version: "4.1.1", Problem: problem, Check: check,
	}}}
}

func vanished(id dc.ID) *dlv1.Report {
	return &dlv1.Report{Kind: &dlv1.Report_Vanished{Vanished: &dlv1.Vanished{Id: id.Bytes()}}}
}

func (s *mediaServer) external(t *testing.T) dc.External {
	t.Helper()
	e, err := externalOf(t, s.lib, s.client)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func (s *mediaServer) row(t *testing.T, id dc.ID) dc.Row {
	t.Helper()
	got := list(t, s.c, dc.Filter{IDs: []dc.ID{id}})
	if len(got.Rows) != 1 {
		t.Fatalf("the Torrents view lists %d rows for %s, want 1", len(got.Rows), id)
	}
	return got.Rows[0]
}

func (s *mediaServer) intent(t *testing.T) *dlv1.Intent {
	t.Helper()
	var in *dlv1.Intent
	err := s.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		in, err = dc.Intent(ctx, r, s.node)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return in
}

func (s *mediaServer) wantedInIntent(t *testing.T, id dc.ID) dlv1.Wanted {
	t.Helper()
	for _, tr := range s.intent(t).GetTorrents() {
		if idFrom(tr.GetId()) == id {
			return tr.GetWanted()
		}
	}
	t.Fatalf("the intent doesn't list %s", id)
	return dlv1.Wanted_WANTED_UNSPECIFIED
}

func TestAClientPollAppliesOnlyUnderSettingsAsNewAsTheStoredPoll(t *testing.T) {
	s := newMediaServer(t)
	if r := dc.ReachOf(s.external(t), true); r != dc.ReachUnchecked {
		t.Fatalf("before any poll the client reads %s, want unchecked", r)
	}

	s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_RESPONDING, nil, nil))
	if p := s.external(t).Poll; p == nil || p.State != tc.Responding || !p.Since.Equal(reportedAt) || p.Problem != nil || p.Version != "4.1.1" {
		t.Fatalf("after a responding poll the client's poll is %+v, want responding since %s on 4.1.1", p, reportedAt)
	}
	if len(s.cards.nodes) != 1 || !slices.Equal(s.cards.nodes[0], []string{s.node.String()}) {
		t.Errorf("the poll told the Admins %v, want Media server's card changed", s.cards.nodes)
	}

	exec(t, s.lib, "UPDATE downloadclients_client SET settings_version = 2, poll_settings_version = 2 WHERE id = ?", s.client)
	s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_NOT_RESPONDING, &dlv1.Problem{Cause: dlv1.Cause_CAUSE_NO_RESPONSE}, nil))
	if p := s.external(t).Poll; p.State != tc.Responding {
		t.Errorf("a poll from before the tested edit left the client %s, want it still responding", p.State)
	}
	if len(s.cards.nodes) != 1 {
		t.Errorf("a dropped poll told the Admins %v, want nothing new", s.cards.nodes[1:])
	}

	tooOld := &dlv1.Problem{Cause: dlv1.Cause_CAUSE_VERSION_TOO_OLD, Version: "3.0"}
	s.report(t, s.node, s.poll(2, dlv1.PollState_POLL_STATE_NEEDS_ATTENTION, tooOld, nil))
	p := s.external(t).Poll
	if p.State != tc.NeedsAttention || p.Problem == nil || p.Problem.Cause != tc.VersionTooOld || p.Problem.Version.String() != "3.0" {
		t.Fatalf("a poll under the new settings left the poll %+v, want needs attention for version 3.0", p)
	}
	if r := dc.ReachOf(s.external(t), true); r != dc.ReachNeedsAttention {
		t.Errorf("the client reads %s, want needs attention", r)
	}

	other := addNode(t, s.lib, "Seedbox")
	s.report(t, other, s.poll(2, dlv1.PollState_POLL_STATE_RESPONDING, nil, nil))
	if p := s.external(t).Poll; p.State != tc.NeedsAttention {
		t.Errorf("another Node's poll of Media server's client left it %s, want needs attention", p.State)
	}
}

func TestAPollKeepsTheLastCheckNowToken(t *testing.T) {
	s := newMediaServer(t)
	token := []byte("check-0001")
	checked := func() []byte {
		t.Helper()
		var got []byte
		if err := s.lib.QueryRowContext(t.Context(), "SELECT poll_check FROM downloadclients_client WHERE id = ?", s.client).Scan(&got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_RESPONDING, nil, token))
	s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_NOT_RESPONDING, &dlv1.Problem{Cause: dlv1.Cause_CAUSE_TIMED_OUT}, nil))
	if got := checked(); !bytes.Equal(got, token) {
		t.Errorf("after a poll that answers no Check now the client carries token %q, want %q", got, token)
	}
}

func TestAVanishedTorrentPartsOnceAsRemovedByItsClient(t *testing.T) {
	s := newMediaServer(t)
	orchard, harbor := s.send(t, "Orchard"), s.send(t, "Harbor")
	removeTorrent(t, s.lib, harbor, dc.ByAdmin)

	for range 2 {
		s.report(t, s.node, vanished(orchard))
	}
	if got := outcomeOf(t, s.lib, orchard).Outcome; got != (dc.Gone{Reason: dc.ByClient, Client: "Transmission"}) {
		t.Errorf("a torrent Transmission stopped listing reads %#v, want gone, removed by Transmission", got)
	}
	if got := s.heard.touchedNodes(); !slices.Equal(got, []role.NodeID{s.node}) {
		t.Errorf("the vanished torrent and its replay touched %v, want Media server's intent once", got)
	}
	if !slices.Contains(s.heard.changedIDs(), orchard.String()) {
		t.Errorf("the Admins heard of %v, want the vanished torrent", s.heard.changedIDs())
	}

	s.report(t, s.node, vanished(harbor))
	if got := outcomeOf(t, s.lib, harbor).Outcome; got != (dc.Gone{Reason: dc.ByAdmin}) {
		t.Errorf("a torrent the Admin was removing reads %#v once Transmission stopped listing it, want gone as the Admin removed it", got)
	}
	if in := s.intent(t); len(in.GetTorrents()) != 0 || len(in.GetDiscards()) != 0 {
		t.Errorf("the intent lists %v and discards %v, want neither vanished torrent", in.GetTorrents(), in.GetDiscards())
	}

	attic := s.send(t, "Attic")
	s.report(t, addNode(t, s.lib, "Seedbox"), vanished(attic))
	if got := outcomeOf(t, s.lib, attic).Outcome; got != (dc.Pending{}) {
		t.Errorf("another Node's vanished report left Media server's torrent %#v, want pending", got)
	}
}

func TestAPauseInTheClientReadsPausedUntilAnAdminResumes(t *testing.T) {
	s := newMediaServer(t)
	id := s.send(t, "Orchard")
	pausedInClient := func() *dlv1.TorrentReport {
		return &dlv1.TorrentReport{Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Run: dlv1.Wanted_WANTED_PAUSED}
	}

	r := pausedInClient()
	r.WantedChange = &dlv1.WantedChange{From: dlv1.Wanted_WANTED_STARTED, To: dlv1.Wanted_WANTED_PAUSED}
	s.report(t, s.node, s.torrent(t, r))
	if row := s.row(t, id); row.State != dc.StatePaused || row.Pending != "" {
		t.Errorf("paused in Transmission, the row reads %s with %q pending, want paused with nothing pending", row.State, row.Pending)
	}
	if w := s.wantedInIntent(t, id); w != dlv1.Wanted_WANTED_PAUSED {
		t.Errorf("paused in Transmission, the intent wants %s, want paused", w)
	}
	if got := s.heard.touchedNodes(); !slices.Equal(got, []role.NodeID{s.node}) {
		t.Errorf("the pause in Transmission touched %v, want Media server's intent", got)
	}

	act(t, s.c, dc.Action{Kind: dc.Resume, IDs: []dc.ID{id}})
	if w := s.wantedInIntent(t, id); w != dlv1.Wanted_WANTED_STARTED {
		t.Errorf("resumed in SlipStream, the intent wants %s, want started", w)
	}
	if row := s.row(t, id); row.Pending != dc.Started {
		t.Errorf("before Transmission resumes it, the row has %q pending, want the resume", row.Pending)
	}

	r = pausedInClient()
	r.WantedChange = &dlv1.WantedChange{From: dlv1.Wanted_WANTED_PAUSED, To: dlv1.Wanted_WANTED_STARTED}
	r.Run = dlv1.Wanted_WANTED_STARTED
	touched := len(s.heard.touchedNodes())
	s.report(t, s.node, s.torrent(t, r))
	if got := s.heard.touchedNodes()[touched:]; len(got) != 0 {
		t.Errorf("a change in Transmission from a state SlipStream no longer wants touched %v, want nothing", got)
	}
	if row := s.row(t, id); row.State != dc.StateDownloading || row.Pending != "" {
		t.Errorf("running in Transmission again, the row reads %s with %q pending, want downloading", row.State, row.Pending)
	}
}

func TestAStopAtTheSeedingLimitIsNoPause(t *testing.T) {
	s := newMediaServer(t)
	id := s.send(t, "Orchard")
	s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{
		Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true, Run: dlv1.Wanted_WANTED_PAUSED, StoppedAtLimit: true,
		WantedChange: &dlv1.WantedChange{From: dlv1.Wanted_WANTED_STARTED, To: dlv1.Wanted_WANTED_PAUSED},
	}))
	if w := s.wantedInIntent(t, id); w != dlv1.Wanted_WANTED_STARTED {
		t.Errorf("stopped at Transmission's seeding limit, the intent wants %s, want started", w)
	}
	if row := s.row(t, id); row.State != dc.StatePaused || row.Pending != "" || !row.StoppedAtLimit {
		t.Errorf("stopped at Transmission's seeding limit, the row reads %s with %q pending, stopped at the limit %t, want paused with nothing pending, stopped at the limit", row.State, row.Pending, row.StoppedAtLimit)
	}
}

func TestAFinishedTorrentSaysWhenItImports(t *testing.T) {
	s := newMediaServer(t)
	id := s.send(t, "Orchard")
	at := time.Date(2026, 11, 7, 14, 32, 0, 0, time.UTC)
	s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{
		Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Complete: true, Run: dlv1.Wanted_WANTED_STARTED, ImportAtUnixMs: at.UnixMilli(),
	}))
	if row := s.row(t, id); !row.ImportAt.Equal(at) {
		t.Errorf("waiting out Transmission's Import delay, the row imports at %v, want %v", row.ImportAt, at)
	}
}

func TestTheFirstOriginReportedStands(t *testing.T) {
	s := newMediaServer(t)
	id := s.send(t, "Orchard")
	if row := s.row(t, id); row.Client == nil || *row.Client != (dc.ClientRef{ID: s.client, Kind: tc.Transmission, Name: "Transmission"}) || row.Attached {
		t.Fatalf("before the Node's first report the row names client %+v, attached %t, want Transmission, not attached", row.Client, row.Attached)
	}
	for _, o := range []dlv1.Origin{dlv1.Origin_ORIGIN_ATTACHED, dlv1.Origin_ORIGIN_ADDED} {
		s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_SEEDING, Origin: o}))
	}
	if row := s.row(t, id); !row.Attached {
		t.Error("a later report of added overwrote the first report of attached")
	}
	svc := dc.ControlService{Library: s.lib, Nodes: caller{s.node}, Types: types{}}
	res, err := svc.Torrent(t.Context(), &dlv1.TorrentRequest{Id: id.Bytes()})
	if err != nil || res.GetOrigin() != dlv1.Origin_ORIGIN_ATTACHED {
		t.Errorf("a Node that lost its local row fetches origin %s (%v), want attached", res.GetOrigin(), err)
	}
}

// caller is the Controller's side of the link with node calling.
type caller struct{ node role.NodeID }

func (c caller) Caller(context.Context) (role.NodeID, bool) { return c.node, true }

func TestATorrentsGrabsReadItsClient(t *testing.T) {
	s := newMediaServer(t)
	id := s.send(t, "Orchard")
	got := outcomeOf(t, s.lib, id)
	if got.Client == nil || got.Client.Name != "Transmission" || got.Client.ID != s.client {
		t.Errorf("a Grab reads its Torrent's client as %+v, want Transmission", got.Client)
	}
	s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{
		Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_CLIENT_ERROR, Error: "No data found! Ensure your drives are connected",
	}))
	if row := s.row(t, id); row.State != dc.StatePaused || row.Severity != dc.SeverityWarning || row.Error != "No data found! Ensure your drives are connected" {
		t.Errorf("with an error in Transmission the row reads %s, severity %d, error %q, want paused with a warning and Transmission's words", row.State, row.Severity, row.Error)
	}
	if got := outcomeOf(t, s.lib, id).Outcome; got != (dc.Pending{}) {
		t.Errorf("with an error in Transmission the Grab reads %#v, want pending", got)
	}
}

func TestAnExternalTorrentQueuesInItsClientsOwnLine(t *testing.T) {
	s := newMediaServer(t)
	external := s.send(t, "Orchard")
	m, raw := dotTorrent(t, "Harbor")
	local, _ := mustAttach(t, s.lib, dc.Want{Target: dc.Target{Node: s.node}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	apply(t, s.lib, s.node, local, queued())
	s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{
		Id: external.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING, Hold: dlv1.Hold_HOLD_SLOT, QueuePosition: 4, Run: dlv1.Wanted_WANTED_STARTED,
	}))
	for id, want := range map[dc.ID]int{external: 4, local: 1} {
		if got := list(t, s.c, dc.Filter{IDs: []dc.ID{id}}).Rows[0].QueuePosition; got != want {
			t.Errorf("torrent %s is %d in line, want %d", id, got, want)
		}
	}
}

func TestEveryCauseANodeSendsReadsBackAsItself(t *testing.T) {
	s := newMediaServer(t)
	causes := dlv1.Cause(0).Descriptor().Values()
	for i := 1; i < causes.Len(); i++ {
		v := causes.Get(i)
		want := tc.Cause(strings.ToLower(strings.TrimPrefix(string(v.Name()), "CAUSE_")))
		state := dlv1.PollState(dlv1.PollState_value["POLL_STATE_"+strings.ToUpper(string(want.State()))])
		s.report(t, s.node, s.poll(1, state, &dlv1.Problem{Cause: dlv1.Cause(v.Number())}, nil))
		if p := s.external(t).Poll; p.Problem == nil || p.Problem.Cause != want || p.State != want.State() {
			t.Errorf("%s reads back as %+v, want %s in state %s", v.Name(), p, want, want.State())
		}
	}
}
