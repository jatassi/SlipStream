package nodes

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

var (
	start = time.Date(2000, 1, 1, 21, 0, 0, 0, time.UTC)
	home  = nodeID("0190a0e0-0000-7000-8000-000000000001")
	box   = nodeID("0190a0e0-0000-7000-8000-000000000002")
	crate = nodeID("0190a0e0-0000-7000-8000-000000000003")
)

func nodeID(s string) ID {
	u, err := storage.ParseUUID(s)
	if err != nil {
		panic(err)
	}
	return ID{UUID: u}
}

// cluster is home, the Controller holding every Role but Relaying, and
// seedbox holding Searching and Downloading, in step since start, with
// the shipped grace, blip and settle.
func cluster() *inputs {
	all := role.Of(role.Names()...).Without(role.Relaying)
	return &inputs{
		controller: home,
		start:      start,
		grace:      15 * time.Minute,
		blip:       time.Minute,
		settle:     time.Minute,
		nodes: []nodeIn{
			{id: home, name: "home", assigned: all.Without(role.Searching).Without(role.Downloading), attached: true, since: start},
			{id: box, name: "seedbox", assigned: role.Of(role.Searching, role.Downloading), attached: true, since: start, seen: start},
		},
	}
}

func (in *inputs) at(t time.Time) *Board { return derive(*in, t) }

func standing(t *testing.T, b *Board, id ID) Standing {
	t.Helper()
	v, ok := b.Node(id)
	if !ok {
		t.Fatalf("the board has no Node %s", id)
	}
	return v.Standing
}

func TestAwayTurnsLapsedExactlyAtTheGraceFromWhenItWasLastSeen(t *testing.T) {
	in := cluster()
	in.node(box).seen = start.Add(time.Hour)
	in.detach(box, start.Add(time.Hour+30*time.Second))
	lapse := start.Add(time.Hour + 15*time.Minute)
	for _, step := range []struct {
		at       time.Time
		standing Standing
		state    RoleState
	}{
		{start.Add(time.Hour + time.Minute), Reconnecting, Away},
		{start.Add(time.Hour + 2*time.Minute), Offline, Away},
		{lapse.Add(-time.Nanosecond), Offline, Away},
		{lapse, LapsedNode, Lapsed},
	} {
		b := in.at(step.at)
		if got, st := standing(t, b, box), b.Who(role.Downloading).State(box); got != step.standing || st != step.state {
			t.Errorf("at %s seedbox stands %d with Downloading %d, want %d and %d", step.at.Format(time.TimeOnly), got, st, step.standing, step.state)
		}
	}
	if v, _ := in.at(lapse).Node(box); !v.Since.Equal(start.Add(time.Hour)) {
		t.Errorf("seedbox is away since %s, want its last frame at %s", v.Since, start.Add(time.Hour))
	}
	if got := nextEdge(*in, start.Add(time.Hour+2*time.Minute)); !got.Equal(lapse) {
		t.Errorf("the next edge after the blip is %s, want the lapse at %s", got, lapse)
	}
}

func TestAControllerDownForHoursLapsesNobodyWhenItStarts(t *testing.T) {
	in := cluster()
	b := in.node(box)
	b.attached, b.seen = false, start.Add(-3*time.Hour)
	if got := standing(t, in.at(start), box); got != Reconnecting {
		t.Errorf("at start seedbox stands %d, want reconnecting", got)
	}
	if got := in.at(start.Add(15*time.Minute - time.Nanosecond)).Who(role.Searching).State(box); got != Away {
		t.Errorf("just before the grace from start seedbox's Searching is %d, want away", got)
	}
	if got := in.at(start.Add(15 * time.Minute)).Who(role.Searching).State(box); got != Lapsed {
		t.Errorf("at the grace from start seedbox's Searching is %d, want lapsed", got)
	}
}

func TestMismatchedCountsAsAway(t *testing.T) {
	in := cluster()
	in.detach(box, start)
	in.node(box).mismatched = true
	b := in.at(start.Add(time.Minute))
	if got, st := standing(t, b, box), b.Who(role.Searching).State(box); got != MismatchedNode || st != Away {
		t.Fatalf("a mismatched seedbox stands %d with Searching %d, want mismatched and away", got, st)
	}
	if got := in.at(start.Add(15 * time.Minute)).Who(role.Searching).State(box); got != Lapsed {
		t.Fatalf("a mismatched seedbox past the grace holds Searching %d, want lapsed", got)
	}
}

func TestTheControllerCoversOnlyRolesWhoseWorkGoesToIt(t *testing.T) {
	in := cluster()
	in.detach(box, start)
	b := in.at(start.Add(15 * time.Minute))
	if who := b.Who(role.Searching); !who.Covering() || !who.May(home) || who.May(box) {
		t.Errorf("Searching: covering %v, home may %v, seedbox may %v; want the Controller covering for lapsed seedbox", who.Covering(), who.May(home), who.May(box))
	}
	if got, ok := b.Who(role.Searching).Pick(box); !ok || got != home {
		t.Errorf("Pick(seedbox) for Searching = %s, %v; want home", got, ok)
	}
	if who := b.Who(role.Downloading); who.Covering() || who.May(home) || !who.Lapsed(box) {
		t.Errorf("Downloading: covering %v, home may %v, seedbox lapsed %v; want it waiting on seedbox", who.Covering(), who.May(home), who.Lapsed(box))
	}
	if got := b.covering(); got != role.Of(role.Searching) {
		t.Errorf("the Controller covers %v, want Searching", got.Names())
	}
	if in.at(start.Add(15*time.Minute - time.Nanosecond)).Who(role.Searching).Covering() {
		t.Errorf("the Controller covers Searching before seedbox lapsed")
	}
}

func TestCoveringEndsOnceTheHolderSettles(t *testing.T) {
	in := cluster()
	in.detach(box, start)
	back := start.Add(20 * time.Minute)
	in.attach(box, back)
	b := in.at(back)
	if who := b.Who(role.Searching); !who.Covering() || who.May(box) || !who.Holds(box) {
		t.Errorf("seedbox just back: covering %v, seedbox may %v, holds %v; want covering while it settles", who.Covering(), who.May(box), who.Holds(box))
	}
	if got := nextEdge(*in, back); !got.Equal(back.Add(time.Minute)) {
		t.Errorf("the next edge is %s, want the settle's end at %s", got, back.Add(time.Minute))
	}
	b = in.at(back.Add(time.Minute))
	if who := b.Who(role.Searching); who.Covering() || !who.May(box) {
		t.Errorf("seedbox settled: covering %v, seedbox may %v; want seedbox searching again", who.Covering(), who.May(box))
	}
}

func TestABlipNeedsNoSettle(t *testing.T) {
	in := cluster()
	in.detach(box, start)
	in.attach(box, start.Add(30*time.Second))
	if who := in.at(start.Add(30 * time.Second)).Who(role.Searching); !who.May(box) || who.Covering() {
		t.Fatalf("seedbox back within the grace: may %v, covering %v; want it searching at once", who.May(box), who.Covering())
	}
}

// A Node away for 10 minutes of wall time with a 20 minute pause in the
// middle has spent 10 minutes of its grace, not 30.
func TestAPauseStopsTheGraceClock(t *testing.T) {
	in := cluster()
	in.detach(box, start)
	in.pause(start.Add(5 * time.Minute))
	if got := nextEdge(*in, start.Add(5*time.Minute)); !got.IsZero() {
		t.Errorf("while paused the next edge is %s, want none", got)
	}
	if got := in.at(start.Add(15 * time.Minute)).Who(role.Downloading).State(box); got != Away {
		t.Errorf("15 minutes away, 10 of them paused: Downloading is %d, want away", got)
	}
	in.resume(start.Add(25 * time.Minute))
	lapse := start.Add(35 * time.Minute)
	if got := nextEdge(*in, start.Add(25*time.Minute)); !got.Equal(lapse) {
		t.Errorf("after the pause the next edge is %s, want the lapse at %s", got, lapse)
	}
	if got := in.at(lapse.Add(-time.Nanosecond)).Who(role.Downloading).State(box); got != Away {
		t.Errorf("just before 15 minutes of grace clock seedbox's Downloading is %d, want away", got)
	}
	if got := in.at(lapse).Who(role.Downloading).State(box); got != Lapsed {
		t.Errorf("at 15 minutes of grace clock seedbox's Downloading is %d, want lapsed", got)
	}
}

func TestAPauseBeforeANodeLeftCountsNothing(t *testing.T) {
	in := cluster()
	in.pause(start)
	in.resume(start.Add(20 * time.Minute))
	in.node(box).seen = start.Add(30 * time.Minute)
	in.detach(box, start.Add(30*time.Minute))
	if got := in.at(start.Add(45 * time.Minute)).Who(role.Downloading).State(box); got != Lapsed {
		t.Fatalf("15 minutes after it left, with the pause before it, Downloading is %d, want lapsed", got)
	}
}

func TestPicksAndHeldWork(t *testing.T) {
	in := cluster()
	in.nodes = append(in.nodes, nodeIn{id: crate, name: "crate", assigned: role.Of(role.Searching), attached: true, since: start})
	b := in.node(box)
	b.assigned = role.Of(role.Searching)
	b.work = role.Of(role.Downloading)
	board := in.at(start)
	if got, ok := board.Who(role.Searching).Pick(crate); !ok || got != crate {
		t.Errorf("Pick(crate) = %s, %v; want crate, which may", got, ok)
	}
	if got, ok := board.Who(role.Searching).Pick(home); !ok || got != box {
		t.Errorf("Pick(home) = %s, %v; want seedbox, the first by id that may", got, ok)
	}
	if who := board.Who(role.Downloading); who.State(box) != Draining || !who.Serves(box) || who.May(box) || who.Holds(box) {
		t.Errorf("seedbox's Downloading is %d, serves %v, may %v, holds %v; want draining, served, nothing new", who.State(box), who.Serves(box), who.May(box), who.Holds(box))
	}
	b.forfeit = role.Of(role.Downloading)
	if who := in.at(start).Who(role.Downloading); who.State(box) != WritingOff || !who.WrittenOff(box) {
		t.Errorf("seedbox's written-off Downloading is %d, written off %v; want writing off", who.State(box), who.WrittenOff(box))
	}
	if _, ok := in.at(start).Who(role.Relaying).Pick(home); ok {
		t.Errorf("Pick found a Relaying Node, want none")
	}
}
