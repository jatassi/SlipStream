package downloadclients_test

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/contenttype/contenttypetest"
	dc "github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/health"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1/downloadingv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	tc "github.com/jatassi/slipstream/internal/torrent/clients"
)

// reach is two downloading Nodes, Media server in step and attic out of
// step, with external clients on each and a Node that checks its clients
// as checks says.
type reach struct {
	lib    library
	media  role.NodeID
	attic  role.NodeID
	roster *roster
	c      *dc.Clients
	node   *checkingNode
}

func newReach(t *testing.T) *reach {
	t.Helper()
	lib := open(t)
	w := &reach{lib: lib, media: addNode(t, lib, "Media server"), attic: addNode(t, lib, "attic")}
	w.roster = &roster{serving: []role.NodeID{w.media, w.attic}, away: map[role.NodeID]bool{w.attic: true}}
	w.node = &checkingNode{w: w, polls: map[dc.ClientID]*dlv1.ClientPoll{}}
	w.c = dc.New(dc.Config{
		Library: lib, Touch: func(...role.NodeID) {}, Changed: &heard{}, CardsChanged: &told{}, Uses: uses{}, Roster: w.roster,
		Remote: func(role.NodeID) downloadingv1connect.DownloadingServiceHandler { return w.node },
		Facts:  func(role.NodeID) (*dlv1.Facts, bool) { return &dlv1.Facts{Bound: "0.0.0.0:6881"}, true },
		Log:    slog.New(slog.DiscardHandler),
	})
	return w
}

// add is a client of kind named name on node at address, on.
func (w *reach) add(t *testing.T, node role.NodeID, kind tc.Kind, name, address string) dc.ClientID {
	t.Helper()
	a, err := tc.ParseAddress(address, kind)
	if err != nil {
		t.Fatal(err)
	}
	l, err := tc.ParseLabel("tv-sonarr", kind)
	if err != nil {
		t.Fatal(err)
	}
	return mustImport(t, w.lib, secrettest.Sealer(t), dc.Imported{Node: node, Kind: kind, Name: name, Address: a, Secrets: tc.Secrets{Password: "hunter2"}, Label: l})
}

// poll applies a poll of client failing for cause, or responding for "",
// as node's outbox delivers it.
func (w *reach) poll(t *testing.T, node role.NodeID, client dc.ClientID, cause tc.Cause) {
	t.Helper()
	w.report(t, node, &dlv1.Report{Kind: &dlv1.Report_Client{Client: pollOf(client, cause, nil)}})
}

func (w *reach) report(t *testing.T, node role.NodeID, rep *dlv1.Report) {
	t.Helper()
	write(t, w.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error { return w.c.Apply(ctx, tx, node, rep) })
	w.c.Applied(node, string(role.Downloading))
}

var causeOnWire = map[tc.Cause]dlv1.Cause{
	tc.NoResponse: dlv1.Cause_CAUSE_NO_RESPONSE, tc.WrongCredentials: dlv1.Cause_CAUSE_WRONG_CREDENTIALS,
	tc.AddressRefused: dlv1.Cause_CAUSE_ADDRESS_REFUSED,
}

var stateOnWire = map[tc.PollState]dlv1.PollState{
	tc.Responding: dlv1.PollState_POLL_STATE_RESPONDING, tc.NotResponding: dlv1.PollState_POLL_STATE_NOT_RESPONDING,
	tc.SignInRefused: dlv1.PollState_POLL_STATE_SIGN_IN_REFUSED, tc.NeedsAttention: dlv1.PollState_POLL_STATE_NEEDS_ATTENTION,
}

func pollOf(client dc.ClientID, cause tc.Cause, check []byte) *dlv1.ClientPoll {
	p := &dlv1.ClientPoll{
		ClientId: client.Bytes(), SettingsVersion: 1, State: dlv1.PollState_POLL_STATE_RESPONDING,
		SinceUnixMs: reportedAt.UnixMilli(), Version: "4.1.1", Check: check,
	}
	if cause != "" {
		p.State, p.Problem = stateOnWire[cause.State()], &dlv1.Problem{Cause: causeOnWire[cause]}
	}
	return p
}

func (w *reach) turnOff(t *testing.T, id dc.ClientID) {
	t.Helper()
	write(t, w.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE downloadclients_client SET is_on = 0 WHERE id = ?", id)
		return err
	})
}

// checkingNode is a Node that, asked to check clients, answers once each poll
// is in its outbox and delivers the polls afterwards, each with the
// check's token and the cause polls holds for its client.
type checkingNode struct {
	downloadingv1connect.UnimplementedDownloadingServiceHandler
	w *reach

	mu      sync.Mutex
	polls   map[dc.ClientID]*dlv1.ClientPoll
	pending sync.WaitGroup
}

func (n *checkingNode) fixed(id dc.ClientID, cause tc.Cause) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.polls[id] = pollOf(id, cause, nil)
}

func (n *checkingNode) CheckClients(_ context.Context, req *dlv1.CheckClientsRequest) (*dlv1.CheckClientsResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []*dlv1.ClientPoll
	for _, raw := range req.GetIds() {
		u, err := storage.UUIDFromBytes(raw)
		if err != nil {
			return nil, err
		}
		p := pollOf(dc.ClientID{UUID: u}, "", req.GetCheck())
		if fixed, ok := n.polls[dc.ClientID{UUID: u}]; ok {
			p.State, p.Problem = fixed.GetState(), fixed.GetProblem()
		}
		out = append(out, p)
	}
	n.pending.Go(func() {
		for _, p := range out {
			if err := n.w.lib.Write(context.Background(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				return n.w.c.Apply(ctx, tx, n.w.media, &dlv1.Report{Kind: &dlv1.Report_Client{Client: p}})
			}); err != nil {
				panic(err)
			}
			n.w.c.Applied(n.w.media, string(role.Downloading))
		}
	})
	return &dlv1.CheckClientsResponse{}, nil
}

func items(a health.Answer) map[health.Key]health.Item {
	out := map[health.Key]health.Item{}
	for _, it := range a.Items {
		out[it.Key] = it
	}
	return out
}

func TestEachAwayExternalClientIsAHealthErrorAndOneOnAnOfflineNodeIsUnsure(t *testing.T) {
	w := newReach(t)
	down := w.add(t, w.media, tc.Transmission, "Transmission", "http://192.168.1.20:9091")
	refused := w.add(t, w.media, tc.Transmission, "Transmission 4K", "http://192.168.1.20:9092")
	refusedDeluge := w.add(t, w.media, tc.Deluge, "Deluge", "http://192.168.1.20:8112")
	blocked := w.add(t, w.media, tc.Transmission, "Transmission anime", "http://192.168.1.20:9093")
	fine := w.add(t, w.media, tc.Transmission, "Transmission kids", "http://192.168.1.20:9094")
	off := w.add(t, w.media, tc.Transmission, "Transmission old", "http://192.168.1.20:9095")
	offline := w.add(t, w.attic, tc.Transmission, "Attic Transmission", "http://192.168.1.30:9091")
	w.poll(t, w.media, down, tc.NoResponse)
	w.poll(t, w.media, refused, tc.WrongCredentials)
	w.poll(t, w.media, refusedDeluge, tc.WrongCredentials)
	w.poll(t, w.media, blocked, tc.AddressRefused)
	w.poll(t, w.media, fine, "")
	w.poll(t, w.media, off, tc.NoResponse)
	w.turnOff(t, off)
	w.poll(t, w.attic, offline, tc.NoResponse)

	a := answer(t, w.c)
	got := items(a)
	for _, c := range []struct {
		what    string
		id      dc.ClientID
		subject string
		text    string
	}{
		{"not responding", down, "Transmission on Media server",
			"Transmission isn't responding. Check that it's running and that Media server can reach 192.168.1.20:9091."},
		{"a refused sign-in on a kind with a lockout", refused, "Transmission 4K on Media server",
			"Transmission 4K rejected the saved password. SlipStream stopped checking it to avoid a lockout. Change the password, or select Check now once it's fixed."},
		{"a refused sign-in on a kind with no lockout", refusedDeluge, "Deluge on Media server",
			"Deluge rejected the password. Check it and test again."},
		{"needing attention", blocked, "Transmission anime on Media server",
			"Transmission anime refused Media server's address. Add it to rpc-whitelist in Transmission anime's settings, then select Check now."},
	} {
		it, ok := got[dc.ClientHealthKey(c.id)]
		if !ok {
			t.Errorf("the client %s raises no Health item", c.what)
			continue
		}
		if it.Level != health.Fail || it.Subject != c.subject || it.Text != c.text {
			t.Errorf("the client %s reads %v %q: %q, want an error %q: %q", c.what, it.Level, it.Subject, it.Text, c.subject, c.text)
		}
		if it.Action.Kind() != health.ActionLink || it.Action.Href() != "/settings/download-clients?client="+c.id.String() {
			t.Errorf("the client %s links to %v %q, want its form", c.what, it.Action.Kind(), it.Action.Href())
		}
		if !slices.ContainsFunc(it.Detail, func(f health.Fact) bool { return f.Label == "Since" && f.At.Equal(reportedAt) }) {
			t.Errorf("the client %s's details are %+v, want since when", c.what, it.Detail)
		}
	}
	for _, id := range []dc.ClientID{fine, off, offline} {
		if it, ok := got[dc.ClientHealthKey(id)]; ok {
			t.Errorf("a responding, off or offline Node's client raises %+v, want no item", it)
		}
	}
	if !slices.ContainsFunc(a.Unsure, func(u health.Unsure) bool {
		return u.Key == dc.ClientHealthKey(offline) && u.Subject == "Attic Transmission on attic" && u.Text == "attic is offline."
	}) {
		t.Errorf("unsure = %+v, want the attic's client unsure while attic is offline", a.Unsure)
	}
}

func TestATorrentsOwnTroubleNeverRaisesAClientItem(t *testing.T) {
	w := newReach(t)
	client := w.add(t, w.media, tc.Transmission, "Transmission", "http://192.168.1.20:9091")
	w.poll(t, w.media, client, "")
	m, raw := dotTorrent(t, "The.Orchard.S01E01.1080p.WEB")
	id, _ := mustAttach(t, w.lib, dc.Want{Target: dc.Target{Node: w.media, Client: &client}, Metainfo: m, Raw: raw, ContentType: contenttypetest.ID, WrittenOff: noneWrittenOff})
	w.report(t, w.media, &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{{
		Id: id.Bytes(), AtUnixMs: reportedAt.UnixMilli(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING,
		Hold: dlv1.Hold_HOLD_CLIENT_ERROR, Error: "No data found! Ensure your drives are connected", Stalled: true,
		Trackers: []*dlv1.TrackerLine{{Index: 0, Status: dlv1.TrackerStatus_TRACKER_STATUS_ERROR, Message: "Tracker is down"}},
	}}}}})
	if it, ok := items(answer(t, w.c))[dc.ClientHealthKey(client)]; ok {
		t.Errorf("a torrent's local error, stall and tracker warning raise %+v, want no client item", it)
	}
}

func TestCheckNowPollsEveryOnClientAndAnswersFromWhatItFound(t *testing.T) {
	w := newReach(t)
	refused := w.add(t, w.media, tc.Transmission, "Transmission", "http://192.168.1.20:9091")
	down := w.add(t, w.media, tc.Transmission, "Transmission 4K", "http://192.168.1.20:9092")
	off := w.add(t, w.media, tc.Transmission, "Transmission old", "http://192.168.1.20:9095")
	offline := w.add(t, w.attic, tc.Transmission, "Attic Transmission", "http://192.168.1.30:9091")
	w.poll(t, w.media, refused, tc.WrongCredentials)
	w.poll(t, w.media, down, tc.NoResponse)
	w.turnOff(t, off)
	w.poll(t, w.attic, offline, tc.NoResponse)
	w.node.fixed(down, tc.NoResponse)

	src := w.c.HealthSource()
	if src.Check == nil {
		t.Fatal("the Download Clients category offers no Check now")
	}
	unreached, err := src.Check(t.Context(), "")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(unreached, []health.Unreached{{Words: "attic is offline, so its Download Clients weren't checked."}}) {
		t.Errorf("unreached = %+v, want the attic's clients unchecked", unreached)
	}
	got := items(answer(t, w.c))
	if it, ok := got[dc.ClientHealthKey(refused)]; ok {
		t.Errorf("after Check now the refused client, since fixed, still reads %+v", it)
	}
	if _, ok := got[dc.ClientHealthKey(down)]; !ok {
		t.Errorf("after Check now the client still down raises no item, want its error")
	}
	e, err := externalOf(t, w.lib, off)
	if err != nil || e.Poll != nil {
		t.Errorf("the off client reads poll %+v (%v) after Check now, want it left unchecked", e.Poll, err)
	}
	w.node.pending.Wait()
}

func TestAClientsTorrentRowsReadAsLastKnownWhileItIsAway(t *testing.T) {
	s := newMediaServer(t)
	s.report(t, s.node, s.poll(1, dlv1.PollState_POLL_STATE_RESPONDING, nil, nil))
	id := s.send(t, "The.Orchard.S01E01.1080p.WEB")
	s.report(t, s.node, s.torrent(t, &dlv1.TorrentReport{Id: id.Bytes(), Engine: dlv1.EngineState_ENGINE_STATE_DOWNLOADING}))
	if r := s.row(t, id); !r.LastKnown.IsZero() || r.LastKnownCause != "" {
		t.Fatalf("a responding client's row reads last known %v (%s), want current", r.LastKnown, r.LastKnownCause)
	}
	for _, c := range []struct {
		state dlv1.PollState
		cause dlv1.Cause
		want  dc.StaleCause
	}{
		{dlv1.PollState_POLL_STATE_NOT_RESPONDING, dlv1.Cause_CAUSE_NO_RESPONSE, dc.StaleClient},
		{dlv1.PollState_POLL_STATE_SIGN_IN_REFUSED, dlv1.Cause_CAUSE_WRONG_CREDENTIALS, dc.StaleClient},
		{dlv1.PollState_POLL_STATE_NEEDS_ATTENTION, dlv1.Cause_CAUSE_MOST_MISSING, ""},
	} {
		s.report(t, s.node, s.poll(1, c.state, &dlv1.Problem{Cause: c.cause, Missing: 14}, nil))
		r := s.row(t, id)
		if r.LastKnownCause != c.want || (c.want != "") != r.LastKnown.Equal(reportedAt) {
			t.Errorf("while the client polls %s its row reads last known %v (%q), want %q", c.state, r.LastKnown, r.LastKnownCause, c.want)
		}
	}
	if !slices.Contains(s.heard.changedIDs(), id.String()) {
		t.Errorf("torrents.changed = %v after the polls, want the client's Torrent so its row refreshes", s.heard.changedIDs())
	}
	write(t, s.lib, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, err := tx.ExecContext(ctx, "UPDATE downloadclients_client SET is_on = 0 WHERE id = ?", s.client)
		return err
	})
	if r := s.row(t, id); r.LastKnownCause != dc.StaleOff || !r.LastKnown.Equal(reportedAt) {
		t.Errorf("an off client's row reads last known %v (%q), want off at its last report", r.LastKnown, r.LastKnownCause)
	}
}

func TestDownloadingKeepsTheNodesExternalClientsByName(t *testing.T) {
	w := newReach(t)
	w.add(t, w.media, tc.Transmission, "Transmission", "http://192.168.1.20:9091")
	w.add(t, w.media, tc.Deluge, "Deluge", "http://192.168.1.20:8112")
	var kept []string
	err := w.lib.Read(t.Context(), func(ctx context.Context, r storage.Reader[storage.Library]) error {
		var err error
		kept, err = dc.DownloadingOwner.Keeps(ctx, r, w.media)
		return err
	})
	if err != nil || !slices.Equal(kept, []string{"Deluge", "Transmission"}) {
		t.Errorf("Downloading on Media server keeps %v (%v), want its clients by name", kept, err)
	}
}
