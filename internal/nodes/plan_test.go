package nodes_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	home    = nodes.Named{ID: nodes.ID{UUID: storage.NewUUID()}, Name: "home"}
	seedbox = nodes.Named{ID: nodes.ID{UUID: storage.NewUUID()}, Name: "seedbox"}
	relay   = nodes.Named{ID: nodes.ID{UUID: storage.NewUUID()}, Name: "relay-fra"}

	torrents = nodes.Held{Units: 3, Noun: nodes.Noun{One: "torrent", Many: "torrents"}, Bytes: 41 << 30, EndlessUnits: 3}
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

func wantEffects(t *testing.T, o nodes.Outcome, want ...nodes.Effect) {
	t.Helper()
	if !reflect.DeepEqual(o.Effects, want) {
		t.Fatalf("effects = %#v, want %#v", o.Effects, want)
	}
}

// wantFacts checks words names each fact, such as a count, a name or an
// error, whatever sentence carries it.
func wantFacts(t *testing.T, words string, facts ...string) {
	t.Helper()
	for _, f := range facts {
		if !strings.Contains(words, f) {
			t.Errorf("%q doesn't name %q", words, f)
		}
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
}

func TestUntickingARoleWithNothingHeldLetsItGo(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Searching, role.Subtitles)))
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching)}})
	wantEffects(t, out, nodes.LetsGo{Node: seedbox, Role: role.Subtitles})
	if _, ok := out.Next[seedbox.ID]; !ok || out.Next[seedbox.ID].Has(role.Subtitles) {
		t.Fatalf("next = %v", out.Next)
	}
}

func TestUntickingTheControllersOwnLastAlwaysRoleKeepsIt(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Downloading)))
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{home.ID: homeRoles.Without(role.Searching)}})
	wantEffects(t, out, nodes.HandedBack{Role: role.Searching, From: []nodes.Named{home}, To: home})
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
	wantEffects(t, add, nodes.Gains{Node: seedbox, Role: role.Relaying})
	wantFacts(t, add.Checks[0].Words, relay.Name)

	move := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{
		seedbox.ID: role.Of(role.Searching, role.Relaying),
		relay.ID:   0,
	}})
	if move.Blocked() {
		t.Fatalf("move blocked: %+v", move.Checks)
	}
	wantEffects(t, move, nodes.Gains{Node: seedbox, Role: role.Relaying}, nodes.LetsGo{Node: relay, Role: role.Relaying})
}

func TestUntickingServingOnTheControllerIsPinned(t *testing.T) {
	s := situation(seat(home, homeRoles), seat(seedbox, role.Of(role.Searching)))
	out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{home.ID: homeRoles.Without(role.Serving)}})
	wantEffects(t, out, nodes.Pinned{Node: home, Role: role.Serving})
	if !out.Same(s) || out.Blocked() {
		t.Fatalf("next = %v", out.Next)
	}
}

func TestProbesGuardAnAddedRole(t *testing.T) {
	tick := nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching, role.Downloading)}}
	for _, tc := range []struct {
		name    string
		probe   role.ProbeResult
		away    bool
		verdict nodes.Verdict
		reason  nodes.Reason
	}{
		{name: "fails", probe: role.ProbeResult{Probe: role.EngineStarts, Fit: role.Fails, Reason: role.DownloadsNotWritable},
			verdict: nodes.Block, reason: nodes.ProbeFails},
		{name: "weak", probe: role.ProbeResult{Probe: role.EngineStarts, Fit: role.Weak, Reason: role.DownloadsOnNetworkShare},
			verdict: nodes.Warn, reason: nodes.ProbeWeak},
		{name: "offline", probe: role.ProbeResult{Probe: role.EngineStarts, Fit: role.Fails, Reason: role.EngineNotListening},
			away: true, verdict: nodes.Warn, reason: nodes.OfflineUnchecked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sb := seat(seedbox, role.Of(role.Searching))
			sb.Away = tc.away
			sb.Probes = map[role.Probe]role.ProbeResult{tc.probe.Probe: tc.probe}
			out := nodes.Plan(situation(seat(home, homeRoles), sb), tick)
			if len(out.Checks) != 1 || out.Checks[0].Verdict != tc.verdict || out.Checks[0].Reason != tc.reason ||
				out.Checks[0].Node != seedbox || out.Checks[0].Role != role.Downloading {
				t.Fatalf("checks = %+v", out.Checks)
			}
			wantEffects(t, out, nodes.Gains{Node: seedbox, Role: role.Downloading})
		})
	}

	sb := seat(seedbox, role.Of(role.Searching))
	sb.Probes = map[role.Probe]role.ProbeResult{role.EngineStarts: {Probe: role.EngineStarts, Fit: role.Fits}}
	if out := nodes.Plan(situation(seat(home, homeRoles), sb), tick); len(out.Checks) != 0 {
		t.Fatalf("a fitting probe checks %+v", out.Checks)
	}
}

// TestEveryProbeReasonBlocksTheRoleItGuards fails a probe for each reason
// on the Role it guards: the Review blocks that Role on seedbox, in words
// no other reason shares.
func TestEveryProbeReasonBlocksTheRoleItGuards(t *testing.T) {
	seen := map[string]role.ProbeReason{}
	for _, why := range role.ProbeReasons() {
		i := slices.IndexFunc(role.Names(), func(r role.Name) bool { return r.Rule().Probe == why.Probe() })
		if i < 0 {
			t.Fatalf("no Role is guarded by %s, the probe that finds %s", why.Probe(), why)
		}
		r := role.Names()[i]
		sb := seat(seedbox, 0)
		sb.Probes = map[role.Probe]role.ProbeResult{why.Probe(): {Probe: why.Probe(), Fit: role.Fails, Reason: why}}
		out := nodes.Plan(situation(seat(home, homeRoles), sb), nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(r)}})
		if len(out.Checks) != 1 {
			t.Fatalf("%s: checks = %+v, want one", why, out.Checks)
		}
		c := out.Checks[0]
		if c.Node != seedbox || c.Role != r || c.Verdict != nodes.Block || c.Reason != nodes.ProbeFails || c.Words == "" {
			t.Errorf("%s: check = %+v, want %s blocked on seedbox", why, c, r)
		}
		if other, ok := seen[c.Words]; ok {
			t.Errorf("%s reads as %s: %q", why, other, c.Words)
		}
		seen[c.Words] = why
	}
}

// TestSuggestedTicksAcquireRolesWhoseProbesFit is the Roles step's first
// ticks: Searching always, and Downloading only while its engine-starts
// probe fits.
func TestSuggestedTicksAcquireRolesWhoseProbesFit(t *testing.T) {
	for _, tc := range []struct {
		name   string
		probes map[role.Probe]role.ProbeResult
		want   role.Set
	}{
		{name: "no probes yet", want: role.Of(role.Searching)},
		{name: "engine fits", probes: map[role.Probe]role.ProbeResult{role.EngineStarts: {Fit: role.Fits}, role.DataWritable: {Fit: role.Fits}}, want: role.Of(role.Searching, role.Downloading)},
		{name: "engine weak", probes: map[role.Probe]role.ProbeResult{role.EngineStarts: {Fit: role.Weak}}, want: role.Of(role.Searching)},
		{name: "engine fails", probes: map[role.Probe]role.ProbeResult{role.EngineStarts: {Fit: role.Fails}}, want: role.Of(role.Searching)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sb := seat(seedbox, 0)
			sb.Probes = tc.probes
			if got := sb.Suggested(); got != tc.want {
				t.Fatalf("Suggested() = %v, want %v", got.Names(), tc.want.Names())
			}
		})
	}
}

var (
	smtpTarget   = nodes.Target{Target: nodesystem.Target{Host: "smtp.example.com", Port: 587}, What: "the mail server", Block: true}
	gotifyTarget = nodes.Target{Target: nodesystem.Target{Host: "gotify.lan", Port: 443, TLS: true}, What: "Gotify 'Home'"}
)

func TestReachTestsGuardAnAddedNotifying(t *testing.T) {
	tick := nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Notifying)}}
	at := nodes.NodeRole{Node: seedbox.ID, Role: role.Notifying}
	smtp, gotify := nodes.ReachKey{NodeRole: at, Target: "smtp.example.com:587"}, nodes.ReachKey{NodeRole: at, Target: "gotify.lan:443"}
	s := situation(seat(home, homeRoles), seat(seedbox, 0))

	untested := nodes.Plan(s, tick)
	wantEffects(t, untested, nodes.Gains{Node: seedbox, Role: role.Notifying})
	if untested.Blocked() || !slices.Equal(reasons(untested), []nodes.Reason{nodes.NothingToTest}) {
		t.Fatalf("checks = %+v, want nothing to test and no block", untested.Checks)
	}

	s.Targets = map[role.Name][]nodes.Target{role.Notifying: {smtpTarget, gotifyTarget}}
	if out := nodes.Plan(s, tick); !slices.Equal(out.Untested, []nodes.ReachKey{smtp, gotify}) {
		t.Fatalf("untested = %+v, want both targets from seedbox", out.Untested)
	}

	s.Reach = map[nodes.ReachKey]nodes.ReachResult{smtp: {Err: "connection refused"}, gotify: {OK: true, Millis: 7}}
	failed := nodes.Plan(s, tick)
	if !failed.Blocked() || !slices.Equal(reasons(failed), []nodes.Reason{nodes.ReachFailed, nodes.Reached}) || failed.Checks[0].Target != "smtp.example.com:587" {
		t.Fatalf("checks = %+v", failed.Checks)
	}
	wantFacts(t, failed.Checks[0].Words, "connection refused")
	wantFacts(t, failed.Checks[1].Words, "7 ms")

	s.Reach = map[nodes.ReachKey]nodes.ReachResult{smtp: {OK: true, Millis: 42}, gotify: {Err: "no such host"}}
	warned := nodes.Plan(s, tick)
	if warned.Blocked() || !slices.Equal(reasons(warned), []nodes.Reason{nodes.Reached, nodes.ReachWarned}) || warned.Checks[1].Verdict != nodes.Warn || warned.Checks[1].Target != "gotify.lan:443" {
		t.Fatalf("checks = %+v", warned.Checks)
	}
	wantFacts(t, warned.Checks[0].Words, "42 ms")
	wantFacts(t, warned.Checks[1].Words, "Gotify 'Home'")
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
}

func TestUntickingARequeuingRolePutsItsWorkBackHeirOrNot(t *testing.T) {
	conversions := nodes.Noun{One: "Conversion", Many: "Conversions"}
	for _, tc := range []struct {
		name  string
		heir  bool
		held  nodes.Held
		facts []string
	}{
		{"the last holder with only waiting work", false, nodes.Held{Units: 3, Noun: conversions}, []string{"3 Conversions waiting on seedbox"}},
		{"a holder with an heir", true, nodes.Held{Units: 1, Noun: conversions}, []string{"1 Conversion waiting on seedbox"}},
		{"running work", false, nodes.Held{Units: 2, Started: 2, Noun: conversions}, []string{"2 Conversions running on seedbox"}},
		{"running and waiting work", true, nodes.Held{Units: 5, Started: 2, Noun: conversions}, []string{"2 Conversions running on seedbox", "3 Conversions waiting on seedbox"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sb := seat(seedbox, role.Of(role.Converting))
			sb.Moves = map[role.Name]nodes.Held{role.Converting: tc.held}
			own := homeRoles.Without(role.Converting)
			if tc.heir {
				own = homeRoles
			}
			s := situation(seat(home, own), sb)
			s.Requeues = role.Of(role.Converting)
			out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: 0}})
			wantEffects(t, out,
				nodes.LetsGo{Node: seedbox, Role: role.Converting},
				nodes.Requeues{Role: role.Converting, From: seedbox, Held: tc.held},
			)
			wantFacts(t, out.Effects[1].Words(), tc.facts...)
		})
	}
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
	wantFacts(t, holding.Checks[0].Words, torrents.String(), role.Downloading.Label())

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
}

func TestGesturesAreThePlansVerdicts(t *testing.T) {
	sb := seat(seedbox, role.Of(role.Searching, role.Downloading))
	sb.Held = map[role.Name]nodes.Held{role.Downloading: torrents}
	s := situation(seat(home, homeRoles.Without(role.Downloading)), sb, seat(relay, role.Of(role.Relaying)))
	s.Targets = map[role.Name][]nodes.Target{role.Notifying: {smtpTarget}}
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
		note    bool
	}{
		{find(role.Downloading, seedbox, home), nodes.Allow, "Moves Downloading", false},
		{find(role.Relaying, none, seedbox), nodes.Block, "relay-fra already relays", false},
		{find(role.Relaying, relay, seedbox), nodes.Allow, "Moves Relaying", false},
		{find(role.Notifying, home, seedbox), nodes.Allow, "Tested when you drop", false},
		{find(role.Serving, home, seedbox), nodes.Allow, "Moves Serving", true},
	} {
		if tc.g.Verdict != tc.verdict || tc.g.Word != tc.word || (tc.g.Note != "") != tc.note {
			t.Errorf("%s from %v to %v = %+v", tc.g.Role, tc.g.From, tc.g.To, tc.g)
		}
	}

	for _, g := range gestures {
		if g.Role == role.Downloading && g.From == home.ID {
			t.Fatalf("a gesture moves Downloading from home, which doesn't hold it: %+v", g)
		}
	}
}

func seedboxWithClients(kept ...string) nodes.Situation {
	sb := seat(seedbox, role.Of(role.Searching, role.Downloading))
	sb.Held = map[role.Name]nodes.Held{role.Downloading: torrents}
	sb.Keeps = map[role.Name][]string{role.Downloading: kept}
	return situation(seat(home, homeRoles), sb)
}

func TestUntickingDownloadingOnANodeWithExternalClientsIsBlocked(t *testing.T) {
	for _, kept := range [][]string{{"Transmission"}, {"Transmission", "qBittorrent"}} {
		s := seedboxWithClients(kept...)
		out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Searching)}})
		if !out.Blocked() || !slices.Equal(reasons(out), []nodes.Reason{nodes.KeepsConfig}) {
			t.Fatalf("checks = %+v, want Downloading kept by its clients", out.Checks)
		}
		wantFacts(t, out.Checks[0].Words, kept...)
		if removed := nodes.Plan(s, nodes.Remove{Node: seedbox.ID}); !slices.Contains(reasons(removed), nodes.KeepsConfig) {
			t.Errorf("removing seedbox without a write-off checks %+v, want it kept by its clients too", removed.Checks)
		}
	}
	s := seedboxWithClients("Transmission")
	for _, g := range nodes.Gestures(s) {
		if g.Role == role.Downloading && g.From == seedbox.ID && (g.Verdict != nodes.Block || g.Word != "Has Transmission") {
			t.Errorf("dragging Downloading off seedbox reads %v %q, want blocked by Transmission", g.Verdict, g.Word)
		}
	}
	if out := nodes.Plan(s, nodes.SetRoles{Want: nodes.Roles{seedbox.ID: role.Of(role.Downloading)}}); out.Blocked() {
		t.Errorf("unticking only Searching checks %+v, want it allowed", out.Checks)
	}
}

func TestAWriteOffStillTakesDownloadingOffANodeWithExternalClients(t *testing.T) {
	s := seedboxWithClients("Transmission")
	out := nodes.Plan(s, nodes.WriteOff{Node: seedbox.ID, Roles: role.Of(role.Downloading)})
	if out.Blocked() || out.Next[seedbox.ID].Has(role.Downloading) {
		t.Fatalf("writing off seedbox's Downloading checks %+v, next %v; want it to go through", out.Checks, out.Next)
	}
	if removed := nodes.Plan(s, nodes.Remove{Node: seedbox.ID, WriteOff: true}); removed.Blocked() {
		t.Errorf("removing seedbox with a write-off checks %+v, want it to go through", removed.Checks)
	}
}
