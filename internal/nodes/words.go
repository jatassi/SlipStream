package nodes

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jatassi/slipstream/internal/role"
)

// Effect is one thing a Change does. Each renders its own words, so no
// front end writes guardrail text.
type Effect interface {
	Words() string
	effect()
}

type (
	Gains struct {
		Node Named
		Role role.Name
	}
	LetsGo struct {
		Node Named
		Role role.Name
	}
	// Drains is a Node taking no new work for Role while it keeps what it holds.
	Drains struct {
		Node Named
		Role role.Name
		Held Held
	}
	// HandedBack is an Always Role the Controller takes because no Node holds it next.
	HandedBack struct {
		Role role.Name
		From []Named
		To   Named
	}
	// Pinned is a Controller Role the Controller keeps whatever the Admin ticks.
	Pinned struct {
		Node Named
		Role role.Name
	}
	// Moves is portable work leaving a Node that lost Role, for its heir.
	Moves struct {
		Role     role.Name
		From, To Named
		Held     Held
	}
	// Requeues is portable work leaving a Node that lost Role, back to its
	// owner's queue.
	Requeues struct {
		Role role.Name
		From Named
		Held Held
	}
	WritesOff struct {
		Node Named
		Role role.Name
		Held Held
	}
	Leaves struct{ Node Named }
)

func (Gains) effect()      {}
func (LetsGo) effect()     {}
func (Drains) effect()     {}
func (HandedBack) effect() {}
func (Pinned) effect()     {}
func (Moves) effect()      {}
func (Requeues) effect()   {}
func (WritesOff) effect()  {}
func (Leaves) effect()     {}

func (e Gains) Words() string  { return e.Node.Name + " gets the " + e.Role.Label() + " Role." }
func (e LetsGo) Words() string { return e.Node.Name + " loses the " + e.Role.Label() + " Role." }

// Words says what the drain keeps. Only Downloading's work can be endless
// in part: a seeding goal ends a torrent, and one with no seeding goal
// stays.
func (e Drains) Words() string {
	h, start := e.Held, e.Node.Name+" takes no new "+e.Role.Label()+" work"
	switch {
	case h.EndlessUnits == 0:
		return fmt.Sprintf("%s and finishes the %s it has.", start, h)
	case e.Role != role.Downloading:
		return fmt.Sprintf("%s. It keeps its %s until you move %s or give %s up.", start, h, h.pronoun(), h.pronoun())
	}
	kept := Held{Units: h.EndlessUnits, Noun: h.Noun}
	if kept.Units == h.Units {
		return fmt.Sprintf("%s. %s %s and %s until you remove, move or give %s up.", start, kept, noSeedingGoal(kept), stays(kept), kept.pronoun())
	}
	ending := Held{Units: h.Units - h.EndlessUnits, Noun: h.Noun}
	goals := "they reach their seeding goals"
	if ending.Units == 1 {
		goals = "it reaches its seeding goal"
	}
	return fmt.Sprintf("%s. It finishes %s once %s. %s %s and %s until you remove, move or give %s up.",
		start, ending, goals, kept, noSeedingGoal(kept), stays(kept), kept.pronoun())
}

func noSeedingGoal(h Held) string { return agree(h, "has no seeding goal", "have no seeding goal") }

func stays(h Held) string { return agree(h, "stays", "stay") }

// agree is one when h is a single Unit and many otherwise.
func agree(h Held, one, many string) string {
	if h.Units == 1 {
		return one
	}
	return many
}

func (e HandedBack) Words() string {
	verb := "takes over"
	if len(e.From) == 1 && e.From[0].ID == e.To.ID {
		verb = "keeps"
	}
	return fmt.Sprintf("%s %s %s, since SlipStream always needs a Node that %s.",
		e.To.Name, verb, e.Role.Label(), e.Role.Purpose())
}

func (e Pinned) Words() string {
	return fmt.Sprintf("%s keeps %s, because the Controller always %s.", e.Node.Name, e.Role.Label(), e.Role.Purpose())
}

func (e Moves) Words() string {
	verb := "move"
	if e.Held.Units == 1 {
		verb = "moves"
	}
	return fmt.Sprintf("%s %s to %s.", e.Held, verb, e.To.Name)
}

// Words says what stops and what waits: "2 Conversions running on Attic
// stop. They start again from the beginning on a Node that can run them.",
// then "3 Conversions waiting on Attic go back to the queue."
func (e Requeues) Words() string {
	running := Held{Units: e.Held.Started, Noun: e.Held.Noun}
	waiting := Held{Units: e.Held.Units - e.Held.Started, Noun: e.Held.Noun}
	var out []string
	if running.Units > 0 {
		out = append(out, fmt.Sprintf("%s running on %s %s. %s again from the beginning on a Node that can run %s.",
			running, e.From.Name, agree(running, "stops", "stop"), agree(running, "It starts", "They start"), running.pronoun()))
	}
	if waiting.Units > 0 {
		out = append(out, fmt.Sprintf("%s waiting on %s %s back to the queue.", waiting, e.From.Name, agree(waiting, "goes", "go")))
	}
	return strings.Join(out, " ")
}

func (e WritesOff) Words() string {
	return fmt.Sprintf("SlipStream gives up the %s %s has for %s. No files are deleted.",
		e.Held, e.Node.Name, e.Role.Label())
}

func (e Leaves) Words() string {
	return e.Node.Name + " is removed. It can join again with a new Join Code."
}

// String is the count and size: "3 torrents (41 GB)", "1 Indexer".
func (h Held) String() string {
	noun := h.Noun.Many
	if h.Units == 1 {
		noun = h.Noun.One
	}
	out := strconv.FormatInt(h.Units, 10) + " " + noun
	if h.Bytes > 0 {
		out += " (" + size(h.Bytes) + ")"
	}
	return out
}

func (h Held) pronoun() string {
	if h.Units == 1 {
		return "it"
	}
	return "them"
}

var sizeUnits = [...]string{"bytes", "KB", "MB", "GB", "TB"}

// size writes bytes as the web's bytes() does.
func size(n int64) string {
	v, unit := float64(n), 0
	for v >= 1024 && unit < len(sizeUnits)-1 {
		v /= 1024
		unit++
	}
	digits := 0
	if unit > 0 && v < 10 {
		digits = 1
	}
	return strconv.FormatFloat(v, 'f', digits, 64) + " " + sizeUnits[unit]
}

func newCheck(n Named, r role.Name, v Verdict, why Reason, words, word string) Check {
	return Check{Node: n, Role: r, Verdict: v, Reason: why, Words: words, gestureWord: word}
}

func roleTaken(added Named, r role.Name, holder Named) Check {
	return newCheck(added, r, Block, RoleTaken,
		fmt.Sprintf("%s already %s. Drag %s to move it.", holder.Name, r.Purpose(), r.Label()),
		holder.Name+" already "+r.Purpose())
}

func probed(n Named, r role.Name, v Verdict, why role.ProbeReason) Check {
	if v == Block {
		return newCheck(n, r, v, ProbeFails,
			fmt.Sprintf("%s can't run on %s. %s", r.Label(), n.Name, why.Words()), "Can't run here")
	}
	return newCheck(n, r, v, ProbeWeak,
		fmt.Sprintf("%s may run poorly on %s. %s", r.Label(), n.Name, why.Words()), "Runs poorly here")
}

func offline(n Named, r role.Name) Check {
	return newCheck(n, r, Warn, OfflineUnchecked,
		fmt.Sprintf("Can't check %s while %s is offline.", r.Label(), n.Name), "Can't check while offline")
}

func nothingToTest(n Named, r role.Name) Check {
	return newCheck(n, r, Allow, NothingToTest,
		fmt.Sprintf("No mail server or notification connection is set up yet, so SlipStream can't test the %s Role on %s.", r.Label(), n.Name), "Nothing to test yet")
}

func reached(n Named, r role.Name, t Target, res ReachResult) Check {
	var c Check
	switch {
	case res.OK:
		c = newCheck(n, r, Allow, Reached,
			fmt.Sprintf("%s reached %s in %d ms.", n.Name, t, res.Millis), "Reached")
	case t.Block:
		c = newCheck(n, r, Block, ReachFailed,
			fmt.Sprintf("%s can't reach %s (%s).", n.Name, t, res.Err), "Can't reach "+t.String())
	default:
		c = newCheck(n, r, Warn, ReachWarned,
			fmt.Sprintf("%s can't reach %s. Messages to %s fail while %s sends them.", n.Name, t.Host, t.What, n.Name), "Can't reach "+t.Host)
	}
	c.Target = t.String()
	return c
}

func isController(n Named) Check {
	return newCheck(n, "", Block, IsController, n.Name+" is the Controller, so it can't be removed.", "Controller")
}

func holdsWork(n Named, r role.Name, h Held) Check {
	return newCheck(n, r, Block, HoldsWork,
		fmt.Sprintf("%s still has %s for %s. Untick its Roles and remove %s once that work finishes, or remove it now and give up that work.",
			n.Name, h, r.Label(), n.Name),
		"Has "+h.String())
}

func nothingHeld(n Named, roles role.Set) Check {
	labels := make([]string, 0, roles.Len())
	for _, r := range roles.Names() {
		labels = append(labels, r.Label())
	}
	return newCheck(n, "", Block, NothingHeld,
		fmt.Sprintf("%s has nothing to give up for %s.", n.Name, strings.Join(labels, " or ")), "Nothing to give up")
}

func keepsRole(n Named, r role.Name, h Held) Check {
	return newCheck(n, r, Block, KeepsRole,
		fmt.Sprintf("%s is the last Node that %s, so it keeps %s and can't give up its %s. Tick %s on another Node first.",
			n.Name, r.Purpose(), r.Label(), h, r.Label()),
		"Last Node that "+r.Purpose())
}

func keepsConfig(n Named, r role.Name, kept []string) Check {
	them := "them"
	if len(kept) == 1 {
		them = "it"
	}
	names := strings.Join(kept, " and ")
	if len(kept) > 2 {
		names = strings.Join(kept[:len(kept)-1], ", ") + " and " + kept[len(kept)-1]
	}
	return newCheck(n, r, Block, KeepsConfig,
		fmt.Sprintf("%s still has %s. First delete %s, or move %s to another Node that %s.", n.Name, names, them, them, r.Purpose()),
		"Has "+names)
}

// keepingWords is the Nodes page line for a Node waiting to be adopted.
func keepingWords(v NodeView) string {
	return v.Name + " keeps every torrent until SlipStream has matched its list of torrents with the library."
}
