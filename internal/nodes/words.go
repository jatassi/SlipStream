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
func (WritesOff) effect()  {}
func (Leaves) effect()     {}

func (e Gains) Words() string  { return e.Node.Name + " starts " + e.Role.Label() + "." }
func (e LetsGo) Words() string { return e.Node.Name + " stops " + e.Role.Label() + "." }

func (e Drains) Words() string {
	if e.Held.Endless {
		return fmt.Sprintf("%s takes no new %s work. It keeps its %s until you move or write %s off.",
			e.Node.Name, e.Role.Label(), e.Held, e.Held.pronoun())
	}
	return fmt.Sprintf("%s takes no new %s work and finishes the %s it holds.", e.Node.Name, e.Role.Label(), e.Held)
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

func (e WritesOff) Words() string {
	return fmt.Sprintf("The %s %s holds for %s %s written off. No files are deleted.",
		e.Held, e.Node.Name, e.Role.Label(), e.Held.verbBe())
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

func (h Held) verbBe() string {
	if h.Units == 1 {
		return "is"
	}
	return "are"
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

var probeSubjects = map[role.Probe]string{
	role.EngineStarts: "torrent engine",
	role.DataWritable: "data folder",
	role.ListenerUp:   "HTTP listener",
}

func newCheck(n Named, r role.Name, v Verdict, why Reason, words, word string) Check {
	return Check{Node: n, Role: r, Verdict: v, Reason: why, Words: words, gestureWord: word}
}

func roleTaken(added Named, r role.Name, holder Named) Check {
	return newCheck(added, r, Block, RoleTaken,
		fmt.Sprintf("%s already %s. Drag %s to move it.", holder.Name, r.Purpose(), r.Label()),
		holder.Name+" already "+r.Purpose())
}

func probed(n Named, r role.Name, v Verdict) Check {
	subject := probeSubjects[r.Rule().Probe]
	if v == Block {
		return newCheck(n, r, v, ProbeFails,
			fmt.Sprintf("%s can't run on %s. Its %s check failed.", r.Label(), n.Name, subject), "Can't run here")
	}
	return newCheck(n, r, v, ProbeWeak,
		fmt.Sprintf("%s may run poorly on %s. Its %s check found problems.", r.Label(), n.Name, subject), "Runs poorly here")
}

func offline(n Named, r role.Name) Check {
	return newCheck(n, r, Warn, OfflineUnchecked,
		fmt.Sprintf("Can't check %s while %s is offline.", r.Label(), n.Name), "Can't check while offline")
}

func nothingToTest(n Named, r role.Name) Check {
	return newCheck(n, r, Allow, NothingToTest,
		fmt.Sprintf("Nothing to test yet for %s on %s.", r.Label(), n.Name), "Nothing to test yet")
}

func reached(n Named, r role.Name, res ReachResult) Check {
	if res.OK {
		return newCheck(n, r, Allow, Reached,
			fmt.Sprintf("%s reached %s in %d ms.", n.Name, res.Target, res.Millis), "Reached")
	}
	return newCheck(n, r, Block, ReachFailed,
		fmt.Sprintf("%s can't reach %s (%s).", n.Name, res.Target, res.Err), "Can't reach "+res.Target)
}

func isController(n Named) Check {
	return newCheck(n, "", Block, IsController, n.Name+" is the Controller, so it can't be removed.", "Controller")
}

func holdsWork(n Named, r role.Name, h Held) Check {
	return newCheck(n, r, Block, HoldsWork,
		fmt.Sprintf("%s still holds %s for %s. Untick its Roles and remove it when they finish, or remove it now and write them off.",
			n.Name, h, r.Label()),
		"Holds "+h.String())
}

func nothingHeld(n Named, roles role.Set) Check {
	labels := make([]string, 0, roles.Len())
	for _, r := range roles.Names() {
		labels = append(labels, r.Label())
	}
	return newCheck(n, "", Block, NothingHeld,
		fmt.Sprintf("%s holds nothing to write off for %s.", n.Name, strings.Join(labels, " or ")), "Nothing held")
}

func keepsRole(n Named, r role.Name, h Held) Check {
	return newCheck(n, r, Block, KeepsRole,
		fmt.Sprintf("%s is the last Node that %s, so it keeps %s and can't write off its %s. Tick %s on another Node first.",
			n.Name, r.Purpose(), r.Label(), h, r.Label()),
		"Last Node that "+r.Purpose())
}
