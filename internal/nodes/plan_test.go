package nodes_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	home    = nodes.Named{ID: nodes.ID{UUID: storage.NewUUID()}, Name: "home"}
	seedbox = nodes.Named{ID: nodes.ID{UUID: storage.NewUUID()}, Name: "seedbox"}
	relay   = nodes.Named{ID: nodes.ID{UUID: storage.NewUUID()}, Name: "relay-fra"}

	torrents = nodes.Held{Units: 3, Noun: nodes.Noun{One: "torrent", Many: "torrents"}, Bytes: 41 << 30, Endless: true}
	indexers = nodes.Held{Units: 3, Noun: nodes.Noun{One: "Indexer", Many: "Indexers"}}
)

// homeRoles is what a Controller holds after first run: every Role but Relaying.
var homeRoles = role.Of(role.Names()...).Without(role.Relaying)

func seat(n nodes.Named, roles role.Set) nodes.Seat { return nodes.Seat{Named: n, Assigned: roles} }

func situation(seats ...nodes.Seat) nodes.Situation {
	return nodes.Situation{Controller: home.ID, Seats: seats}
}

func words(o nodes.Outcome) []string {
	var out []string
	for _, e := range o.Effects {
		out = append(out, e.Words())
	}
	for _, c := range o.Checks {
		out = append(out, c.Words)
	}
	return out
}

func reasons(o nodes.Outcome) []nodes.Reason {
	var out []nodes.Reason
	for _, c := range o.Checks {
		out = append(out, c.Reason)
	}
	return out
}

func wantWords(t *testing.T, o nodes.Outcome, want ...string) {
	t.Helper()
	if got := words(o); !slices.Equal(got, want) {
		t.Fatalf("words:\n got %q\nwant %q", got, want)
	}
}

func seedboxDownloading() nodes.Situation {
	sb := seat(seedbox, role.Of(role.Searching, role.Downloading))
	sb.Held = map[role.Name]nodes.Held{role.Downloading: torrents}
	return situation(seat(home, homeRoles.Without(role.Downloading)), sb)
}

func TestUntickingTheLastDownloaderDrainsItAndHandsDownloadingToTheController(t *testing.T) {
	s := seedboxDownloading()
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching)}})

	want := []nodes.Effect{
		nodes.Drains{Node: seedbox, Role: role.Downloading, Held: torrents},
		nodes.HandedBack{Role: role.Downloading, From: []nodes.Named{seedbox}, To: home},
	}
	if !reflect.DeepEqual(out.Effects, want) {
		t.Fatalf("effects = %#v", out.Effects)
	}
	if !out.Next[home.ID].Has(role.Downloading) || out.Next[seedbox.ID].Has(role.Downloading) || out.Blocked() {
		t.Fatalf("next = %v, blocked %v", out.Next, out.Blocked())
	}
	wantWords(t, out,
		"seedbox takes no new Downloading work. It keeps its 3 torrents (41 GB) until you move or write them off.",
		"home takes over Downloading, since SlipStream always needs a Node that downloads.")
}

func TestADrainThatFinishesSaysSo(t *testing.T) {
	s := seedboxDownloading()
	s.Seats[1].Held[role.Downloading] = nodes.Held{Units: 1, Noun: torrents.Noun, Bytes: 2_500_000_000}
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching)}})
	if got := out.Effects[0].Words(); got != "seedbox takes no new Downloading work and finishes the 1 torrent (2.3 GB) it holds." {
		t.Fatalf("words = %q", got)
	}
}

func TestUntickingARoleWithNothingHeldLetsItGo(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Searching, role.Subtitles)))
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching)}})
	wantWords(t, out, "seedbox stops Subtitles.")
	if _, ok := out.Next[seedbox.ID]; !ok || out.Next[seedbox.ID].Has(role.Subtitles) {
		t.Fatalf("next = %v", out.Next)
	}
}

func TestUntickingTheControllersOwnLastAlwaysRoleKeepsIt(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Downloading)))
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{home.ID: homeRoles.Without(role.Searching)}})
	wantWords(t, out, "home keeps Searching, since SlipStream always needs a Node that searches.")
	if !out.Same(s) {
		t.Fatalf("next = %v, want the assignment unchanged", out.Next)
	}
}

func TestASecondRelayIsBlockedNamingTheHolderAndAMovePasses(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Searching)), seat(relay, role.Of(role.Relaying)))

	add := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching, role.Relaying)}})
	if !add.Blocked() || !slices.Equal(reasons(add), []nodes.Reason{nodes.RoleTaken}) || add.Checks[0].Node != seedbox {
		t.Fatalf("checks = %+v", add.Checks)
	}
	wantWords(t, add, "seedbox starts Relaying.", "relay-fra already relays. Drag Relaying to move it.")

	move := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{
		seedbox.ID: role.Of(role.Searching, role.Relaying),
		relay.ID:   0,
	}})
	if move.Blocked() {
		t.Fatalf("move blocked: %+v", move.Checks)
	}
	wantWords(t, move, "seedbox starts Relaying.", "relay-fra stops Relaying.")
}

func TestUntickingServingOnTheControllerIsPinned(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Searching)))
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{home.ID: homeRoles.Without(role.Serving)}})
	if !reflect.DeepEqual(out.Effects, []nodes.Effect{nodes.Pinned{Node: home, Role: role.Serving}}) {
		t.Fatalf("effects = %#v", out.Effects)
	}
	if !out.Same(s) || out.Blocked() {
		t.Fatalf("next = %v", out.Next)
	}
	wantWords(t, out, "home keeps Serving, because the Controller always serves SlipStream.")
}

func TestProbesGuardAnAddedRole(t *testing.T) {
	tick := nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching, role.Downloading)}}
	for _, tc := range []struct {
		name    string
		fit     role.Fit
		away    bool
		verdict nodes.Verdict
		reason  nodes.Reason
		words   string
	}{
		{name: "fails", fit: role.Fails, verdict: nodes.Block, reason: nodes.ProbeFails,
			words: "Downloading can't run on seedbox. Its torrent engine check failed."},
		{name: "weak", fit: role.Weak, verdict: nodes.Warn, reason: nodes.ProbeWeak,
			words: "Downloading may run poorly on seedbox. Its torrent engine check found problems."},
		{name: "offline", fit: role.Fails, away: true, verdict: nodes.Warn, reason: nodes.OfflineUnchecked,
			words: "Can't check Downloading while seedbox is offline."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sb := seat(seedbox, role.Of(role.Searching))
			sb.Away = tc.away
			sb.Probes = map[role.Probe]role.Fit{role.EngineStarts: tc.fit}
			out := nodes.Plan(situation(seat(home, homeRoles), sb), tick)
			if len(out.Checks) != 1 || out.Checks[0].Verdict != tc.verdict || out.Checks[0].Reason != tc.reason {
				t.Fatalf("checks = %+v", out.Checks)
			}
			wantWords(t, out, "seedbox starts Downloading.", tc.words)
		})
	}

	sb := seat(seedbox, role.Of(role.Searching))
	sb.Probes = map[role.Probe]role.Fit{role.EngineStarts: role.Fits}
	if out := nodes.Plan(situation(seat(home, homeRoles), sb), tick); len(out.Checks) != 0 {
		t.Fatalf("a fitting probe checks %+v", out.Checks)
	}
}

func TestReachTestsGuardAnAddedNotifying(t *testing.T) {
	tick := nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Notifying)}}
	at := nodes.NodeRole{Node: seedbox.ID, Role: role.Notifying}
	s := situation(seat(home, homeRoles), seat(seedbox, 0))

	untested := nodes.Plan(s, tick)
	wantWords(t, untested, "seedbox starts Notifying.", "Nothing to test yet for Notifying on seedbox.")
	if untested.Blocked() {
		t.Fatal("no target blocked")
	}

	s.Targets = role.Of(role.Notifying)
	s.Reach = map[nodes.NodeRole]nodes.ReachResult{at: {Target: "smtp.example.com:587", Err: "connection refused"}}
	failed := nodes.Plan(s, tick)
	if !failed.Blocked() || !slices.Equal(reasons(failed), []nodes.Reason{nodes.ReachFailed}) {
		t.Fatalf("checks = %+v", failed.Checks)
	}
	wantWords(t, failed, "seedbox starts Notifying.", "seedbox can't reach smtp.example.com:587 (connection refused).")

	s.Reach[at] = nodes.ReachResult{Target: "smtp.example.com:587", OK: true, Millis: 42}
	if ok := nodes.Plan(s, tick); ok.Blocked() || !slices.Equal(reasons(ok), []nodes.Reason{nodes.Reached}) {
		t.Fatalf("checks = %+v", ok.Checks)
	}
}

func TestUntickingSearchingMovesItsIndexersToTheHeir(t *testing.T) {
	sb := seat(seedbox, role.Of(role.Searching))
	sb.Moves = map[role.Name]nodes.Held{role.Searching: indexers}
	s := situation(seat(home, homeRoles), sb)
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: 0}})
	want := []nodes.Effect{
		nodes.LetsGo{Node: seedbox, Role: role.Searching},
		nodes.Moves{Role: role.Searching, From: seedbox, To: home, Held: indexers},
	}
	if !reflect.DeepEqual(out.Effects, want) {
		t.Fatalf("effects = %#v", out.Effects)
	}
	wantWords(t, out, "seedbox stops Searching.", "3 Indexers move to home.")
}

func TestTheSameSetIsANoOpAndAnAppliedChangeMovesTheBasis(t *testing.T) {
	s := seedboxDownloading()
	same := nodes.Plan(s, nodes.SetRoles{Basis: s.Basis(), Want: nodes.Roles{seedbox.ID: s.Seats[1].Assigned}})
	if !same.Same(s) || len(same.Effects) != 0 || same.Basis != s.Basis() {
		t.Fatalf("same set = %+v", same)
	}

	reordered := situation(s.Seats[1], s.Seats[0])
	if reordered.Basis() != s.Basis() {
		t.Fatal("the basis depends on seat order")
	}

	out := nodes.Plan(s, nodes.SetRoles{Basis: s.Basis(), Want: nodes.Roles{seedbox.ID: role.Of(role.Searching)}})
	after := applied(s, out)
	if after.Basis() == out.Basis {
		t.Fatal("an applied change left the basis as it was, so a plan made before it can't be told stale")
	}
	if again := nodes.Plan(after, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching)}}); !again.Same(after) {
		t.Fatalf("planning the applied set again = %+v", again)
	}
}

// applied is s after Apply stores o's rows.
func applied(s nodes.Situation, o nodes.Outcome) nodes.Situation {
	var seats []nodes.Seat
	for _, st := range s.Seats {
		if _, ok := o.Next[st.ID]; !ok && slices.ContainsFunc(o.Effects, func(e nodes.Effect) bool { return e == nodes.Leaves{Node: st.Named} }) {
			continue
		}
		st.Assigned, st.Forfeit = o.Next[st.ID], o.Forfeit[st.ID]
		seats = append(seats, st)
	}
	s.Seats = seats
	return s
}

func TestWritingOffAStillTickedRoleUnticksIt(t *testing.T) {
	s := seedboxDownloading()
	out := nodes.Plan(s, nodes.WriteOff{Node: seedbox.ID, Roles: role.Of(role.Downloading)})

	want := []nodes.Effect{
		nodes.WritesOff{Node: seedbox, Role: role.Downloading, Held: torrents},
		nodes.HandedBack{Role: role.Downloading, From: []nodes.Named{seedbox}, To: home},
	}
	if !reflect.DeepEqual(out.Effects, want) || out.Blocked() {
		t.Fatalf("effects = %#v, checks %+v", out.Effects, out.Checks)
	}
	if out.Next[seedbox.ID] != role.Of(role.Searching) || out.Forfeit[seedbox.ID] != role.Of(role.Downloading) {
		t.Fatalf("next = %v, forfeit = %v", out.Next, out.Forfeit)
	}
	wantWords(t, out,
		"The 3 torrents (41 GB) seedbox holds for Downloading are written off. No files are deleted.",
		"home takes over Downloading, since SlipStream always needs a Node that downloads.")

	after := applied(s, out)
	if again := nodes.Plan(after, nodes.WriteOff{Node: seedbox.ID, Roles: role.Of(role.Downloading)}); !again.Same(after) || again.Blocked() {
		t.Fatalf("writing off twice = %+v", again)
	}

	retick := nodes.Plan(after, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching, role.Downloading)}})
	if _, ok := retick.Forfeit[seedbox.ID]; ok {
		t.Fatalf("ticking a written-off Role again kept its forfeit: %v", retick.Forfeit)
	}
}

func TestWritingOffADrainingRoleLeavesTheAssignment(t *testing.T) {
	s := seedboxDownloading()
	s.Seats[0].Assigned = homeRoles
	s.Seats[1].Assigned = role.Of(role.Searching)
	out := nodes.Plan(s, nodes.WriteOff{Node: seedbox.ID, Roles: role.Of(role.Downloading)})
	if !reflect.DeepEqual(out.Effects, []nodes.Effect{nodes.WritesOff{Node: seedbox, Role: role.Downloading, Held: torrents}}) {
		t.Fatalf("effects = %#v", out.Effects)
	}
	if !reflect.DeepEqual(out.Next, nodes.Roles{home.ID: homeRoles, seedbox.ID: role.Of(role.Searching)}) {
		t.Fatalf("next = %v", out.Next)
	}
}

func TestWritingOffNothingIsRefused(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Downloading)))
	out := nodes.Plan(s, nodes.WriteOff{Node: seedbox.ID, Roles: role.Of(role.Downloading)})
	if !out.Blocked() || !slices.Equal(reasons(out), []nodes.Reason{nodes.NothingHeld}) {
		t.Fatalf("checks = %+v", out.Checks)
	}
	wantWords(t, out, "seedbox holds nothing to write off for Downloading.")
}

func TestTheLastHolderCantWriteOffWhatItKeeps(t *testing.T) {
	h := seat(home, homeRoles)
	h.Held = map[role.Name]nodes.Held{role.Downloading: torrents}
	s := situation(h, seat(seedbox, role.Of(role.Searching)))
	out := nodes.Plan(s, nodes.WriteOff{Node: home.ID, Roles: role.Of(role.Downloading)})
	if !out.Blocked() || !slices.Equal(reasons(out), []nodes.Reason{nodes.KeepsRole}) || len(out.Forfeit) != 0 {
		t.Fatalf("checks = %+v, forfeit %v", out.Checks, out.Forfeit)
	}
}

func TestRemoval(t *testing.T) {
	sb := seat(seedbox, role.Of(role.Searching, role.Downloading))
	sb.Held = map[role.Name]nodes.Held{role.Downloading: torrents}
	sb.Moves = map[role.Name]nodes.Held{role.Searching: indexers}
	s := situation(seat(home, homeRoles.Without(role.Downloading)), sb)

	ctrl := nodes.Plan(s, nodes.Remove{Node: home.ID, WriteOff: true})
	if !ctrl.Blocked() || !slices.Equal(reasons(ctrl), []nodes.Reason{nodes.IsController}) || !ctrl.Same(s) {
		t.Fatalf("removing the Controller = %+v", ctrl)
	}

	holding := nodes.Plan(s, nodes.Remove{Node: seedbox.ID})
	if !holding.Blocked() || !slices.Equal(reasons(holding), []nodes.Reason{nodes.HoldsWork}) {
		t.Fatalf("checks = %+v", holding.Checks)
	}
	if got := holding.Checks[0].Words; got != "seedbox still holds 3 torrents (41 GB) for Downloading. "+
		"Untick its Roles and remove it when they finish, or remove it now and write them off." {
		t.Fatalf("words = %q", got)
	}

	out := nodes.Plan(s, nodes.Remove{Node: seedbox.ID, WriteOff: true})
	if out.Blocked() {
		t.Fatalf("checks = %+v", out.Checks)
	}
	want := []nodes.Effect{
		nodes.WritesOff{Node: seedbox, Role: role.Downloading, Held: torrents},
		nodes.Moves{Role: role.Searching, From: seedbox, To: home, Held: indexers},
		nodes.HandedBack{Role: role.Downloading, From: []nodes.Named{seedbox}, To: home},
		nodes.Leaves{Node: seedbox},
	}
	if !reflect.DeepEqual(out.Effects, want) {
		t.Fatalf("effects = %#v", out.Effects)
	}
	if !reflect.DeepEqual(out.Next, nodes.Roles{home.ID: homeRoles}) || len(out.Forfeit) != 0 {
		t.Fatalf("next = %v, forfeit = %v", out.Next, out.Forfeit)
	}
	if got := out.Effects[3].Words(); got != "seedbox is removed. It can join again with a new Join Code." {
		t.Fatalf("words = %q", got)
	}
}

func TestGesturesAreThePlansVerdicts(t *testing.T) {
	sb := seat(seedbox, role.Of(role.Searching, role.Downloading))
	sb.Held = map[role.Name]nodes.Held{role.Downloading: torrents}
	s := situation(seat(home, homeRoles.Without(role.Downloading)), sb, seat(relay, role.Of(role.Relaying)))
	s.Targets = role.Of(role.Notifying)
	gestures := nodes.Gestures(s)

	find := func(r role.Name, from nodes.Named, to nodes.Named) nodes.Gesture {
		t.Helper()
		for _, g := range gestures {
			if g.Role == r && g.From == from.ID && g.To == to.ID {
				return g
			}
		}
		t.Fatalf("no gesture for %s from %s to %s", r, from.Name, to.Name)
		return nodes.Gesture{}
	}
	none := nodes.Named{}

	for _, tc := range []struct {
		g       nodes.Gesture
		verdict nodes.Verdict
		word    string
		note    string
	}{
		{find(role.Downloading, seedbox, home), nodes.Allow, "Moves Downloading", ""},
		{find(role.Relaying, none, seedbox), nodes.Block, "relay-fra already relays", ""},
		{find(role.Relaying, relay, seedbox), nodes.Allow, "Moves Relaying", ""},
		{find(role.Notifying, home, seedbox), nodes.Allow, "Tested when you drop", ""},
		{find(role.Serving, home, seedbox), nodes.Allow, "Moves Serving",
			"home keeps Serving, because the Controller always serves SlipStream."},
	} {
		if tc.g.Verdict != tc.verdict || tc.g.Word != tc.word || tc.g.Note != tc.note {
			t.Errorf("%s from %v to %v = %+v", tc.g.Role, tc.g.From, tc.g.To, tc.g)
		}
	}

	for _, g := range gestures {
		if g.Role == role.Downloading && g.From == home.ID {
			t.Fatalf("a gesture moves Downloading from home, which doesn't hold it: %+v", g)
		}
	}
}
