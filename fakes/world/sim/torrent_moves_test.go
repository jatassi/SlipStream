package sim_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// orchard is the moves Scenario's The Quiet Orchard release.
const orchard = "The.Quiet.Orchard.2021.1080p.WEB-DL.H264-FKW"

// launchMoves starts the moves Scenario walking, so every client
// announces to fake-tracker.
func launchMoves(t *testing.T, opts ...sim.Option) *sim.World {
	t.Helper()
	s, err := scenario.Load("moves")
	if err != nil {
		t.Fatal(err)
	}
	return sim.Launch(t, s, append([]sim.Option{sim.WithWalk()}, opts...)...)
}

// client is a Download Client of the moves Scenario: its Node and kind,
// "" for the built-in one.
type client struct{ node, kind string }

func (c client) String() string { return c.node + "/" + cmp.Or(c.kind, "built-in") }

// side is where c is, as a Move names it.
func (c client) side(t *testing.T, w *sim.World) moves.Side {
	t.Helper()
	s := moves.Side{Node: nodeID(t, w, c.node)}
	if c.kind != "" {
		s.Client = clientOn(t, w, c.node, c.kind)
	}
	return s
}

// grab has home hand c The Quiet Orchard from Fixture Site's torznab
// feed, as a Grab does, answering the Torrent it attached to.
func grab(t *testing.T, w *sim.World, c client) seeded {
	t.Helper()
	raw, hash, err := releases.Metainfo(orchard+".mkv", nil, sim.Payload(orchard), w.Announce("torznab"), false)
	if err != nil {
		t.Fatal(err)
	}
	m, err := bencode.DecodeMetainfo(raw)
	if err != nil {
		t.Fatal(err)
	}
	var torznab indexers.TrackerID
	controller := w.Node("home").Half()
	if controller.Library.QueryRowContext(t.Context(), `SELECT id FROM indexers_tracker WHERE name = 'torznab'`).Scan(&torznab) != nil {
		torznab = addTracker(t, w, "torznab", indexers.Private)
	}
	target := downloadclients.Target{Node: nodeID(t, w, c.node)}
	if c.kind != "" {
		target.Client = clientOn(t, w, c.node, c.kind)
	}
	var id downloadclients.ID
	err = controller.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		want := downloadclients.Want{
			Target: target, Metainfo: m, Raw: raw, ContentType: works.TypeID("movie"), Tracker: torznab,
			WrittenOff: func(role.NodeID) bool { return false },
		}
		id, _, err = downloadclients.Attach(ctx, tx, want, storage.Now())
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	controller.Clients.Touch(target.Node)
	return seeded{id: id, hash: hash}
}

// seedOrchard grabs The Quiet Orchard into c and waits until it seeds.
func seedOrchard(t *testing.T, w *sim.World, c client) seeded {
	t.Helper()
	s := grab(t, w, c)
	within(t, time.Hour, "The Quiet Orchard seeds in "+c.String(), func() bool { return seeding(t, w.Node("home").Half()) == 1 })
	return s
}

// moveTorrent previews and starts the Move of s to to, failing unless it
// is movable.
func moveTorrent(t *testing.T, w *sim.World, s seeded, to moves.Side) moves.BatchID {
	t.Helper()
	m := w.Node("home").Half().Moves
	plan, err := m.Preview(t.Context(), moves.Ask{Origin: moves.OriginTorrent, Torrents: []moves.TorrentTo{{Torrent: s.id, To: to}}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Movable() != 1 {
		t.Fatalf("the preview is %+v, want the torrent movable", plan.Units)
	}
	started, err := m.Start(t.Context(), identity.Admin{}, plan.Token, 1)
	if err != nil {
		t.Fatal(err)
	}
	return started.Batch
}

// ended is batch's one unit once it is done or failed.
func ended(t *testing.T, w *sim.World, batch moves.BatchID) moves.View {
	t.Helper()
	var unit moves.View
	for range 120 {
		_, units, err := w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
		if err == nil && len(units) == 1 {
			unit = units[0]
			if unit.Phase == moves.PhaseDone || unit.Phase == moves.PhaseFailed {
				return unit
			}
		}
		time.Sleep(time.Minute)
	}
	t.Fatalf("the torrent's Move is still %s %s after 2 hours: %s", unit.Phase, unit.Waiting, unit.Words)
	return unit
}

// clientOn is the id of on's external Download Client of kind.
func clientOn(t *testing.T, w *sim.World, on, kind string) *downloadclients.ClientID {
	t.Helper()
	var id downloadclients.ClientID
	simtest.Eventually(t, on+"'s "+kind+" client is paired", func() bool {
		return w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT c.id FROM downloadclients_client c
			JOIN nodes_node n ON n.id = c.node_id WHERE n.name = ? AND c.kind = ?`, on, kind).Scan(&id) == nil
	})
	return &id
}

// oneLive is what is wrong with the announces fake-tracker heard: an
// announce it refused, or a second peer started while another was.
func oneLive(heard []sim.Announced) string {
	live := map[string]bool{}
	for _, a := range heard {
		if !a.Accepted {
			return fmt.Sprintf("%s's %q was refused: %s", a.Node, a.Event, a.Failure)
		}
		switch a.Event {
		case "stopped":
			delete(live, a.PeerID)
		default:
			live[a.PeerID] = true
		}
		if len(live) > 1 {
			return fmt.Sprintf("%s announced %q at %s while another peer was started", a.Node, a.Event, a.At.Format(time.TimeOnly))
		}
	}
	return ""
}

// kinds are the moves Scenario's clients of each kind. Each external
// kind has one; the built-in one runs on both Nodes, and a Move into it
// goes to the source's Node, or den's from seedbox's.
var kinds = []client{{"seedbox", ""}, {"seedbox", "qbittorrent"}, {"seedbox", "deluge"}, {"den", "transmission"}, {"den", "rtorrent"}}

// pairs are the client pairs a Move takes: every ordered pair of kinds,
// two kinds on one Node handing off in place, and seedbox's built-in
// client to den's.
func pairs() [][2]client {
	out := [][2]client{{kinds[0], {"den", ""}}}
	for _, from := range kinds {
		for _, to := range kinds {
			switch {
			case from.kind == to.kind:
				continue
			case to.kind == "":
				to.node = from.node
			}
			out = append(out, [2]client{from, to})
		}
	}
	return out
}

// TestATorrentMovesBetweenEveryPairOfClients moves The Quiet Orchard
// between each pair of clients: the guard waits out the source's last
// announce, the source stops before the destination starts, fake-tracker
// accepts every announce and never sees two peers started at once, and
// the hash ends started at the destination alone with the source removed
// as moved.
func TestATorrentMovesBetweenEveryPairOfClients(t *testing.T) {
	t.Parallel()
	for _, p := range pairs() {
		from, to := p[0], p[1]
		t.Run(from.String()+" to "+to.String(), func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				w := launchMoves(t)
				s := seedOrchard(t, w, from)
				if unit := ended(t, w, moveTorrent(t, w, s, to.side(t, w))); unit.Phase != moves.PhaseDone {
					t.Fatalf("the unit ended %s %+v (%s), want done", unit.Phase, unit.Failure, unit.Words)
				}
				if got := wantedOf(t, w, s); !slices.Equal(got, []string{to.node + ":started"}) {
					t.Errorf("the hash's Torrents are %v, want %s's alone, started", got, to)
				}
				if got := goalRunOf(t, w, s).removed; got != string(downloadclients.Moved) {
					t.Errorf("the source was removed %q, want moved", got)
				}
				if wrong := oneLive(w.Announces(s.hash)); wrong != "" {
					t.Error(wrong)
				}
			})
		})
	}
}

// TestAGrabAfterAChangeOfClientAttachesToTheMovedTorrent moves The Quiet
// Orchard from qBittorrent to Deluge on seedbox, then grabs it again: the
// Grab attaches to Deluge's Torrent and adds none.
func TestAGrabAfterAChangeOfClientAttachesToTheMovedTorrent(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, client{"seedbox", "qbittorrent"})
		if unit := ended(t, w, moveTorrent(t, w, s, client{"seedbox", "deluge"}.side(t, w))); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		var moved downloadclients.ID
		if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT id FROM downloadclients_torrent
			WHERE info_hash = ? AND removed_at IS NULL`, s.hash[:]).Scan(&moved); err != nil {
			t.Fatal(err)
		}
		if again := grab(t, w, client{"seedbox", "qbittorrent"}); again.id != moved {
			t.Errorf("the Grab attached to %s, want Deluge's %s", again.id, moved)
		}
		if got := wantedOf(t, w, s); !slices.Equal(got, []string{"seedbox:started"}) {
			t.Errorf("the hash's Torrents are %v, want Deluge's alone", got)
		}
	})
}

// wantedOf is each Torrent of s's hash as Node:wanted, removed ones left
// out.
func wantedOf(t *testing.T, w *sim.World, s seeded) []string {
	t.Helper()
	rows, err := w.Node("home").Half().Library.QueryContext(t.Context(), `SELECT n.name || ':' || t.wanted
		FROM downloadclients_torrent t JOIN nodes_node n ON n.id = t.node_id
		WHERE t.info_hash = ? AND t.removed_at IS NULL ORDER BY t.arriving_unit IS NOT NULL`, s.hash[:])
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// TestARefusedStartSeedsFromTheSourceAgainKeepingTheCopy moves The Quiet
// Orchard from seedbox's built-in client to Transmission on den, whose
// first started fake-tracker refuses: the Move fails switch_refused with
// seedbox seeding again and den's copy kept paused, and a cancel then
// removes the copy.
func TestARefusedStartSeedsFromTheSourceAgainKeepingTheCopy(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, kinds[0])
		w.Cue("tracker", tracker.RejectStart.Name, tracker.RejectStartArgs{Release: orchard, OriginNode: "den", Reason: "You already seed this torrent"})
		unit := ended(t, w, moveTorrent(t, w, s, moves.Side{Node: nodeID(t, w, "den"), Client: clientOn(t, w, "den", "transmission")}))
		if unit.Phase != moves.PhaseFailed || unit.Failure.Code != moves.SwitchRefused {
			t.Fatalf("the unit ended %s %+v, want failed switch_refused", unit.Phase, unit.Failure)
		}
		if got, want := wantedOf(t, w, s), []string{"seedbox:started", "den:paused"}; !slices.Equal(got, want) {
			t.Errorf("the hash's Torrents are %v, want %v", got, want)
		}
		if err := w.Node("home").Half().Moves.Cancel(t.Context(), unit.Unit); err != nil {
			t.Fatal(err)
		}
		within(t, time.Hour, "the cancel removes the kept copy", func() bool {
			return slices.Equal(wantedOf(t, w, s), []string{"seedbox:started"})
		})
	})
}

// TestAPrivateTorrentIntoTheBuiltInClientNotesItsWindowStarts previews
// The Quiet Orchard's Move out of qBittorrent into seedbox's built-in
// client: it can move, with the amber note that its Hit and run window
// counts from now.
func TestAPrivateTorrentIntoTheBuiltInClientNotesItsWindowStarts(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, client{"seedbox", "qbittorrent"})
		plan, err := w.Node("home").Half().Moves.Preview(t.Context(), moves.Ask{Origin: moves.OriginTorrent, Torrents: []moves.TorrentTo{{Torrent: s.id, To: kinds[0].side(t, w)}}})
		if err != nil {
			t.Fatal(err)
		}
		want := moves.Verdict{Color: moves.Amber, Reasons: []moves.Reason{{
			Check: moves.CheckHitAndRun, Color: moves.Amber, Code: "window_starts", Words: "SlipStream starts counting this Hit and run window from now.",
		}}}
		if plan.Movable() != 1 || !reflect.DeepEqual(plan.Units[0].Verdict, want) {
			t.Errorf("the preview is %+v, want it movable with %+v", plan.Units, want)
		}
	})
}

// startedBy is the Nodes fake-tracker heard a started of s from.
func startedBy(w *sim.World, s seeded) []string {
	var out []string
	for _, a := range w.Announces(s.hash) {
		if a.Event == "started" && !slices.Contains(out, a.Node) {
			out = append(out, a.Node)
		}
	}
	return out
}

// TestAPausedTorrentMovesPaused moves The Quiet Orchard, paused in
// seedbox's built-in client, to Transmission on den: it switches with no
// guard, den's copy stays paused, and den never announces a start.
func TestAPausedTorrentMovesPaused(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, kinds[0])
		act(t, w, downloadclients.Action{Kind: downloadclients.Pause, IDs: []downloadclients.ID{s.id}})
		if unit := ended(t, w, moveTorrent(t, w, s, client{"den", "transmission"}.side(t, w))); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v (%s), want done", unit.Phase, unit.Failure, unit.Words)
		}
		if got := wantedOf(t, w, s); !slices.Equal(got, []string{"den:paused"}) {
			t.Errorf("the hash's Torrents are %v, want den's alone, paused", got)
		}
		if got := startedBy(w, s); slices.Contains(got, "den") {
			t.Errorf("fake-tracker heard den start the paused torrent: %v", got)
		}
	})
}

// TestAShortPayloadFailsTheDestinationsCheck cuts the payload den placed
// in Transmission's folder before its check: the Move fails
// destination_check_failed, seedbox seeds on, and den never announces.
func TestAShortPayloadFailsTheDestinationsCheck(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var w *sim.World
		w = launchMoves(t, sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Test.BeforeMoveStep = func(_ context.Context, s moves.Step) {
					if s.Action == moves.RecordPlace {
						if err := os.Truncate(placedAtDen(w), sim.PayloadSize/2); err != nil {
							t.Error(err)
						}
					}
				}
			}
		}))
		s := seedOrchard(t, w, kinds[0])
		unit := ended(t, w, moveTorrent(t, w, s, client{"den", "transmission"}.side(t, w)))
		if unit.Phase != moves.PhaseFailed || unit.Failure.Code != moves.DestinationCheckFailed {
			t.Fatalf("the unit ended %s %+v, want failed destination_check_failed", unit.Phase, unit.Failure)
		}
		within(t, time.Hour, "the undo removes den's copy", func() bool {
			return slices.Equal(wantedOf(t, w, s), []string{"seedbox:started"})
		})
		if got := startedBy(w, s); slices.Contains(got, "den") {
			t.Errorf("fake-tracker heard den start a copy that failed its check: %v", got)
		}
	})
}

// TestAStartedMoveTakesItsLaneAtOnce starts a Move long after the runner
// last passed: its unit leaves queued within a pass, not at the next
// unrelated event.
func TestAStartedMoveTakesItsLaneAtOnce(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, kinds[0])
		time.Sleep(time.Minute)
		batch := moveTorrent(t, w, s, client{"den", "transmission"}.side(t, w))
		time.Sleep(5 * time.Second)
		_, units, err := w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
		if err != nil {
			t.Fatal(err)
		}
		if units[0].Phase == moves.PhaseQueued {
			t.Errorf("the unit is still queued 5 seconds after its Start: %s", units[0].Words)
		}
	})
}

// TestAMovingTorrentShowsItsMove lists The Quiet Orchard while it moves
// to den: its row carries the Move to den, and the moved row carries none.
func TestAMovingTorrentShowsItsMove(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, kinds[0])
		clients := w.Node("home").Half().Clients
		batch := moveTorrent(t, w, s, client{"den", "transmission"}.side(t, w))
		l, err := clients.List(t.Context(), downloadclients.Filter{IDs: []downloadclients.ID{s.id}})
		if err != nil {
			t.Fatal(err)
		}
		if len(l.Rows) != 1 || l.Rows[0].Move == nil || l.Rows[0].Move.ToNodeName != "den" || l.Rows[0].Move.Batch != batch.UUID {
			t.Fatalf("the moving row is %+v, want its Move to den", l.Rows)
		}
		if unit := ended(t, w, batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		l, err = clients.List(t.Context(), downloadclients.Filter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(l.Rows) != 1 || l.Rows[0].Move != nil {
			t.Errorf("the moved rows are %+v, want one with no Move", l.Rows)
		}
	})
}

// TestAClientsTorrentsMoveThroughTheAPI previews moving every torrent of
// qBittorrent on seedbox into Transmission on den through the API, with
// the client origin, and starts it: The Quiet Orchard ends in Transmission.
func TestAClientsTorrentsMoveThroughTheAPI(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, client{"seedbox", "qbittorrent"})
		den := client{"den", "transmission"}.side(t, w)
		body := fmt.Sprintf(`{"origin":"client","works":[],"clients":[{"nodeId":%q,"clientId":%q,"to":{"nodeId":%q,"clientId":%q}}]}`,
			nodeID(t, w, "seedbox"), clientOn(t, w, "seedbox", "qbittorrent"), den.Node, den.Client)
		p := simtest.Decode[previewReply](t, "previewMoves", w.API().Admin("home", http.MethodPost, "/api/v1/moves/preview", body))
		if p.Movable != 1 {
			t.Fatalf("the preview is %+v, want The Quiet Orchard movable", p)
		}
		var started struct {
			BatchID string `json:"batchId"`
		}
		r := w.API().Admin("home", http.MethodPost, "/api/v1/moves?confirm=1", fmt.Sprintf(`{"token":%q}`, p.Token))
		if r.Status != http.StatusAccepted || json.Unmarshal([]byte(r.Body), &started) != nil {
			t.Fatalf("startMoves = %d %s, want 202", r.Status, r.Body)
		}
		id, err := storage.ParseUUID(started.BatchID)
		if err != nil {
			t.Fatal(err)
		}
		if unit := ended(t, w, moves.BatchID{UUID: id}); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		if got := wantedOf(t, w, s); !slices.Equal(got, []string{"den:started"}) {
			t.Errorf("the hash's Torrents are %v, want Transmission's alone", got)
		}
	})
}

// TestATorrentMovesBack moves The Quiet Orchard from seedbox's built-in
// client to Transmission on den, then back: it seeds from seedbox's
// built-in client alone.
func TestATorrentMovesBack(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, kinds[0])
		m := w.Node("home").Half().Moves
		batch := moveTorrent(t, w, s, client{"den", "transmission"}.side(t, w))
		if unit := ended(t, w, batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		back, err := m.Back(t.Context(), batch)
		if err != nil {
			t.Fatal(err)
		}
		if back.Movable() != 1 || back.Units[0].Torrent.To != kinds[0].side(t, w) {
			t.Fatalf("the way back is %+v, want the torrent movable to seedbox's built-in client", back.Units)
		}
		started, err := m.Start(t.Context(), identity.Admin{}, back.Token, 1)
		if err != nil {
			t.Fatal(err)
		}
		if unit := ended(t, w, started.Batch); unit.Phase != moves.PhaseDone {
			t.Fatalf("the way back ended %s %+v, want done", unit.Phase, unit.Failure)
		}
		if got := wantedOf(t, w, s); !slices.Equal(got, []string{"seedbox:started"}) {
			t.Errorf("the hash's Torrents are %v, want seedbox's alone, started", got)
		}
	})
}
