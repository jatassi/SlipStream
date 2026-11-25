package sim_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/fakes/tracker"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/movestest"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/downloadclients"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/moves"
	"github.com/jatassi/slipstream/internal/node"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
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
	s := movestest.Scenario(t)
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

// noted is whether v is amber with one Hit and run note of code.
func noted(v moves.Verdict, code string) bool {
	return v.Color == moves.Amber && len(v.Reasons) == 1 && v.Reasons[0].Check == moves.CheckHitAndRun && v.Reasons[0].Color == moves.Amber && v.Reasons[0].Code == code
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
		if plan.Movable() != 1 || !noted(plan.Units[0].Verdict, "window_starts") {
			t.Errorf("the preview is %+v, want it movable with an amber window_starts note", plan.Units)
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

// TestADestinationCheckThatNeverEndsRollsBack sets the check bound to 30
// minutes and cuts den's reports once den placed the payload in
// Transmission's folder: home never hears the check's result, so the Move
// fails destination_check_failed at the bound, and once den is back the
// undo removes its copy while seedbox seeds on.
func TestADestinationCheckThatNeverEndsRollsBack(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var w *sim.World
		var cut atomic.Bool
		w = launchMoves(t, sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Test.BeforeMoveStep = func(_ context.Context, s moves.Step) {
					if s.Action == moves.RecordPlace && cut.CompareAndSwap(false, true) {
						simtest.Flip(t, w.Path("den", "home"), fault.Stall{})
					}
				}
			}
		}))
		simtest.Decode[json.RawMessage](t, "patchMoveSettings",
			w.API().Admin("home", http.MethodPatch, "/api/v1/moves/settings", `{"values":{"moves.checkBound":1800}}`))
		s := seedOrchard(t, w, kinds[0])
		unit := ended(t, w, moveTorrent(t, w, s, client{"den", "transmission"}.side(t, w)))
		if !cut.Load() || unit.Phase != moves.PhaseFailed || unit.Failure.Code != moves.DestinationCheckFailed {
			t.Fatalf("with den's reports cut after its place the unit ended %s %+v, want failed destination_check_failed", unit.Phase, unit.Failure)
		}
		w.Path("den", "home").Clear("")
		within(t, time.Hour, "the undo removes den's copy", func() bool {
			return slices.Equal(wantedOf(t, w, s), []string{"seedbox:started"})
		})
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

// TestAPrivateTorrentIntoAnExternalClientNotesItsWindowStops previews The
// Quiet Orchard's Move out of seedbox's built-in client into Transmission
// on den, its Tracker needing 3 days of seeding with the margin: it can move, with the
// amber note that SlipStream stops counting its Hit and run window.
func TestAPrivateTorrentIntoAnExternalClientNotesItsWindowStops(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		torznab := addTracker(t, w, "torznab", indexers.Private)
		put(t, w.API(), fmt.Sprintf("/api/v1/trackers/%s/rule", torznab), `{"clears":"time","ratio":null,"seedSeconds":252000}`)
		s := seedOrchard(t, w, kinds[0])
		within(t, time.Hour, "its Hit and run window opens", func() bool {
			d, err := w.Node("home").Half().Clients.Get(t.Context(), s.id)
			return err == nil && d.Seeding.Owes != nil
		})
		plan, err := w.Node("home").Half().Moves.Preview(t.Context(), moves.Ask{Origin: moves.OriginTorrent, Torrents: []moves.TorrentTo{{Torrent: s.id, To: client{"den", "transmission"}.side(t, w)}}})
		if err != nil {
			t.Fatal(err)
		}
		v := plan.Units[0].Verdict
		if plan.Movable() != 1 || !noted(v, "window_stops") || !strings.Contains(v.Reasons[0].Words, "torznab") || !strings.Contains(v.Reasons[0].Words, "3 more days") {
			t.Errorf("the preview is %+v, want it movable with an amber window_stops note naming torznab's 3 more days", v)
		}
	})
}

// TestATorrentWithNoRoomAtItsDestinationStays previews The Quiet
// Orchard's Move into den's built-in client with every volume's error
// level at 100%: it is red for space.
func TestATorrentWithNoRoomAtItsDestinationStays(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, kinds[0])
		simtest.Eventually(t, "den samples its downloads volume", func() bool {
			var n int
			return w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT count(*) FROM downloadclients_volume WHERE node_id = ?`, nodeID(t, w, "den")).Scan(&n) == nil && n == 1
		})
		onTorrents(t, w, `UPDATE health_settings SET error_amount = 100, error_unit = 'percent'`)
		plan, err := w.Node("home").Half().Moves.Preview(t.Context(), moves.Ask{Origin: moves.OriginTorrent, Torrents: []moves.TorrentTo{{Torrent: s.id, To: moves.Side{Node: nodeID(t, w, "den")}}}})
		if err != nil {
			t.Fatal(err)
		}
		if v := plan.Units[0].Verdict; v.Color != moves.Red || v.Reasons[0].Code != "space_error" {
			t.Errorf("the verdict is %+v, want red space_error", v)
		}
	})
}

// TestAClientsNodeChangeTakesItsTorrents changes Box's Node from seedbox
// to den, as an Admin does once the client and its disk run there. A
// Client folder where den finds none of its files refuses the change;
// with its disk's folder den finds The Quiet Orchard's files, the Torrent
// goes with the client to den, nothing is copied, and it keeps seeding.
func TestAClientsNodeChangeTakesItsTorrents(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		s := seedOrchard(t, w, client{"seedbox", "qbittorrent"})
		box, den := clientOn(t, w, "seedbox", "qbittorrent"), nodeID(t, w, "den")
		path := "/api/v1/external-download-clients/" + box.String()
		edit := func(folder string) worldtest.Reply {
			body, err := json.Marshal(map[string]any{"nodeId": den.String(), "clientFolders": []map[string]string{{"client": "/downloads", "node": folder}}})
			if err != nil {
				t.Fatal(err)
			}
			return w.API().Admin("home", http.MethodPatch, path, string(body))
		}
		if r := edit(t.TempDir()); r.Status != http.StatusConflict || !strings.Contains(r.Body, `"code":"download_client_files_missing"`) {
			t.Fatalf("changing Box's Node to den with a Client folder that holds nothing = %d %s, want 409 download_client_files_missing", r.Status, r.Body)
		}
		if r := edit(w.ClientFolder("seedbox", "qbittorrent")); r.Status != http.StatusOK {
			t.Fatalf("changing Box's Node to den = %d %s, want 200", r.Status, r.Body)
		}
		home := w.Node("home").Half()
		var on storage.UUID
		if err := home.Library.QueryRowContext(t.Context(), `SELECT node_id FROM downloadclients_torrent WHERE id = ?`, s.id).Scan(&on); err != nil || on != den.UUID {
			t.Fatalf("The Quiet Orchard's Torrent is on %s (%v) after the change, want den", on, err)
		}
		within(t, time.Hour, "The Quiet Orchard seeds from Box on den", func() bool { return seeding(t, home) == 1 })
		var units int
		if err := home.Library.QueryRowContext(t.Context(), `SELECT count(*) FROM moves_unit`).Scan(&units); err != nil || units != 0 {
			t.Errorf("the Node change made %d Move units (%v), want none", units, err)
		}
	})
}

// seedSeconds is the seeding seconds the Torrent of s in a client of the
// Node named on counts.
func seedSeconds(t *testing.T, w *sim.World, s seeded, on string) int64 {
	t.Helper()
	var n int64
	err := w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT t.seeding_seconds FROM downloadclients_torrent t
		JOIN nodes_node n ON n.id = t.node_id WHERE t.info_hash = ? AND n.name = ? ORDER BY t.added_order DESC LIMIT 1`, s.hash[:], on).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// TestAMovedTorrentKeepsCountingItsTotals moves The Quiet Orchard from
// seedbox's built-in client into den's built-in client and into
// Transmission on den: the destination starts from the source's seeding
// time and keeps counting it.
func TestAMovedTorrentKeepsCountingItsTotals(t *testing.T) {
	t.Parallel()
	for _, to := range []client{{"den", ""}, {"den", "transmission"}} {
		t.Run(to.String(), func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				w := launchMoves(t)
				s := seedOrchard(t, w, kinds[0])
				within(t, time.Hour, "seedbox counts 10 minutes of seeding", func() bool { return seedSeconds(t, w, s, "seedbox") >= 600 })
				if unit := ended(t, w, moveTorrent(t, w, s, to.side(t, w))); unit.Phase != moves.PhaseDone {
					t.Fatalf("the unit ended %s %+v (%s), want done", unit.Phase, unit.Failure, unit.Words)
				}
				was := seedSeconds(t, w, s, "seedbox")
				if now := seedSeconds(t, w, s, "den"); now < was {
					t.Fatalf("den counts %d seconds of seeding once it switched, want at least seedbox's %d", now, was)
				}
				within(t, time.Hour, "den counts on from seedbox's seeding time", func() bool { return seedSeconds(t, w, s, "den") > was })
			})
		})
	}
}

// TestABuiltInTorrentIntoQBittorrentReleasesItsWindow moves a private
// torrent whose Hit and run window owes from seedbox's built-in client
// into Box on seedbox: Box follows its own rules, so the window is
// released.
func TestABuiltInTorrentIntoQBittorrentReleasesItsWindow(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		torznab := addTracker(t, w, "torznab", indexers.Private)
		put(t, w.API(), fmt.Sprintf("/api/v1/trackers/%s/rule", torznab), `{"clears":"time","ratio":null,"seedSeconds":252000}`)
		s := seedOrchard(t, w, kinds[0])
		state := func() string {
			var st string
			if err := w.Node("home").Half().Library.QueryRowContext(t.Context(), `SELECT state FROM trackers_window WHERE info_hash = ?`, s.hash[:]).Scan(&st); err != nil {
				return err.Error()
			}
			return st
		}
		within(t, time.Hour, "its Hit and run window opens", func() bool { return state() == "owing" })
		if unit := ended(t, w, moveTorrent(t, w, s, kinds[1].side(t, w))); unit.Phase != moves.PhaseDone {
			t.Fatalf("the unit ended %s %+v (%s), want done", unit.Phase, unit.Failure, unit.Words)
		}
		within(t, time.Hour, "its window is released", func() bool { return state() == "released" })
	})
}

// TestAGoalMetMidMoveCancelsTheMove starts The Quiet Orchard's Move from
// seedbox to den while den is down, then gives its public Tracker a goal
// it already met: the goal cancels the Move, removes the source as goal_met, and den
// never gets the torrent.
func TestAGoalMetMidMoveCancelsTheMove(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := launchMoves(t)
		torznab := addTracker(t, w, "torznab", indexers.Public)
		s := seedOrchard(t, w, kinds[0])
		within(t, time.Hour, "seedbox counts 2 minutes of seeding", func() bool { return seedSeconds(t, w, s, "seedbox") >= 120 })
		to, a := client{"den", ""}.side(t, w), w.API()
		w.Node("den").Stop()
		batch := moveTorrent(t, w, s, to)
		put(t, a, fmt.Sprintf("/api/v1/trackers/%s/goal", torznab), `{"combine":"first","ratio":null,"seedSeconds":60,"action":"remove"}`)
		var unit moves.View
		within(t, time.Hour, "the goal cancels the Move", func() bool {
			_, units, err := w.Node("home").Half().Moves.Get(t.Context(), batch, -1, 10)
			if err != nil || len(units) != 1 {
				return false
			}
			unit = units[0]
			return unit.Phase == moves.PhaseCanceled
		})
		if got := goalRunOf(t, w, s).removed; got != string(downloadclients.GoalMet) {
			t.Errorf("the source was removed %q, want goal_met", got)
		}
		if got := wantedOf(t, w, s); len(got) != 0 {
			t.Errorf("the hash's Torrents are %v after the goal, want none", got)
		}
	})
}

// TestACutReportPathDuringTheStartSeedsFromOneSideAtATime cuts den's
// reports to home as home asks den to start The Quiet Orchard: home never
// hears the start, rolls the Move back once its bound passes and resumes
// seedbox. A Retry hands off again while den's late report of the first
// start arrives, which the Retry ignores, and fake-tracker never sees
// seedbox and den started at once.
func TestACutReportPathDuringTheStartSeedsFromOneSideAtATime(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var w *sim.World
		var cut atomic.Bool
		w = launchMoves(t, sim.Edit(func(name string, e *node.Env) {
			if name != "home" {
				return
			}
			e.Test.BeforeMoveStep = func(_ context.Context, s moves.Step) {
				if s.Action == "ask_start" && cut.CompareAndSwap(false, true) {
					simtest.Flip(t, w.Path("den", "home"), fault.Stall{})
				}
			}
		}))
		s := seedOrchard(t, w, kinds[0])
		unit := ended(t, w, moveTorrent(t, w, s, client{"den", ""}.side(t, w)))
		if !cut.Load() || unit.Phase != moves.PhaseFailed || unit.Failure == nil || unit.Failure.Code != moves.DestinationUnreachable {
			t.Fatalf("with den's reports cut at its start the unit ended %s %+v, want failed destination_unreachable", unit.Phase, unit.Failure)
		}
		within(t, time.Hour, "seedbox seeds again", func() bool { return slices.Contains(wantedOf(t, w, s), "seedbox:started") })
		within(t, time.Hour, "the rollback is undone", func() bool {
			_, units, err := w.Node("home").Half().Moves.Get(t.Context(), unit.Batch, -1, 10)
			return err == nil && len(units) == 1 && units[0].Undone
		})

		if err := w.Node("home").Half().Moves.Retry(t.Context(), unit.Unit); err != nil {
			t.Fatal(err)
		}
		w.Path("den", "home").Clear("")
		if again := ended(t, w, unit.Batch); again.Phase != moves.PhaseDone {
			t.Fatalf("the Retry ended %s %+v, want done", again.Phase, again.Failure)
		}
		if got := wantedOf(t, w, s); !slices.Equal(got, []string{"den:started"}) {
			t.Errorf("the hash's Torrents are %v after the Retry, want den's alone, started", got)
		}
		if wrong := oneLive(w.Announces(s.hash)); wrong != "" {
			t.Error(wrong)
		}
	})
}

// TestAForgedArrivalFromTheSourceNodeAppliesAsNothing cuts den's reports
// as home asks den to start The Quiet Orchard, then has seedbox report
// that den's copy started, echoing the ask: the report counts only for
// seedbox's own Torrents, so the Move never switches on it and rolls
// back once its bound passes.
func TestAForgedArrivalFromTheSourceNodeAppliesAsNothing(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var w *sim.World
		var cut atomic.Bool
		w = launchMoves(t, sim.Edit(func(name string, e *node.Env) {
			if name == "home" {
				e.Test.BeforeMoveStep = func(_ context.Context, s moves.Step) {
					if s.Action == "ask_start" && cut.CompareAndSwap(false, true) {
						simtest.Flip(t, w.Path("den", "home"), fault.Stall{})
					}
				}
			}
		}))
		s := seedOrchard(t, w, kinds[0])
		batch := moveTorrent(t, w, s, client{"den", ""}.side(t, w))
		home := w.Node("home").Half()
		var arriving storage.UUID
		var asked storage.Time
		within(t, time.Hour, "home asks den to start", func() bool {
			return home.Library.QueryRowContext(t.Context(), `SELECT id, start_asked_at FROM downloadclients_torrent
				WHERE info_hash = ? AND arriving_unit IS NOT NULL AND start_asked_at IS NOT NULL`, s.hash[:]).Scan(&arriving, &asked) == nil
		})
		forged := &dlv1.Report{Kind: &dlv1.Report_Torrents{Torrents: &dlv1.Torrents{Reports: []*dlv1.TorrentReport{{
			Id: arriving.Bytes(), AtUnixMs: time.Now().UnixMilli(),
			Arrival: &dlv1.Arrival{AskedUnixMs: asked.Time().UnixMilli(), Result: dlv1.ArrivalResult_ARRIVAL_RESULT_STARTED},
		}}}}}
		err := home.Library.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
			return home.Clients.Apply(ctx, tx, nodeID(t, w, "seedbox"), forged)
		})
		if err != nil {
			t.Fatal(err)
		}
		if unit := ended(t, w, batch); unit.Phase != moves.PhaseFailed || unit.Failure == nil || unit.Failure.Code != moves.DestinationUnreachable {
			t.Fatalf("after seedbox's forged report the unit ended %s %+v, want failed destination_unreachable", unit.Phase, unit.Failure)
		}
		within(t, time.Hour, "seedbox seeds again", func() bool { return slices.Contains(wantedOf(t, w, s), "seedbox:started") })
	})
}
