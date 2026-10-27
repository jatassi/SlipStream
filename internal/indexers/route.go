package indexers

import (
	"time"

	"github.com/jatassi/slipstream/internal/cardigann"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
)

// Roster is the Nodes and the Roles they hold.
type Roster interface {
	Self() role.NodeID
	// Holding is every Node assigned r, not only covering it: the
	// Controller first, then by id.
	Holding(r role.Name) []role.NodeID
	// Pick is prefer when it may take new r work, else the Controller when
	// it may, else the first Node by id that may.
	Pick(r role.Name, prefer role.NodeID) (role.NodeID, bool)
	// PickOther is Pick for work besides can't take.
	PickOther(r role.Name, besides role.NodeID) (role.NodeID, bool)
	// Away is how node stands while it is assigned r and out of step.
	Away(r role.Name, node role.NodeID) (nodes.Absence, bool)
	Name(node role.NodeID) string
}

// defaultHome is the planner's heir: the Controller when it holds
// Searching, else the first Node by id that does.
func defaultHome(r Roster) (role.NodeID, error) {
	if holding := r.Holding(role.Searching); len(holding) > 0 {
		return holding[0], nil
	}
	return role.NodeID{}, ErrNoSearchingNode
}

// holdsSession reports whether idx signs in, so its calls stay on the Node
// that earned its session: a catalog Indexer whose pinned definition's
// sign-in keeps a session. A Torznab feed is stateless.
func holdsSession(idx Indexer) bool {
	d, ok := idx.Source.(Definition)
	return ok && d.Login == cardigann.SessionLogin
}

// portable reports whether idx's calls may go to a Node other than its
// home: it is stateless or allows failover, and isn't waiting for home.
func portable(idx Indexer, st Status) bool {
	return (!holdsSession(idx) || idx.AllowFailover) && st.State != HomeOnly
}

// route is where one call for idx goes now, or the Waiting outcome: home
// while home may take it; inside the offline grace, wait; past it, a
// holder other than home, the Controller covering included, when idx is
// portable, else wait. The stored home never changes. Pure over the
// roster.
func route(r Roster, idx Indexer, st Status) (role.NodeID, *Outcome) {
	home := idx.Home
	moves := portable(idx, st)
	if a, away := r.Away(role.Searching, home); away {
		if a.Lapsed && moves {
			if to, ok := r.PickOther(role.Searching, home); ok {
				return to, nil
			}
		}
		return role.NodeID{}, &Outcome{Kind: Waiting, Node: home, Since: a.Since, Lapsed: a.Lapsed}
	}
	if to, ok := r.Pick(role.Searching, home); ok && (to == home || moves) {
		return to, nil
	}
	return role.NodeID{}, &Outcome{Kind: Waiting, Node: home}
}

// OutcomeKind is every way an Indexer stands for a call or a search.
type OutcomeKind string

const (
	Found        OutcomeKind = "found"
	NoReleases   OutcomeKind = "no_releases"
	IsLimited    OutcomeKind = "limited"
	IsFailing    OutcomeKind = "failing"
	IsStopped    OutcomeKind = "stopped"
	Waiting      OutcomeKind = "waiting"
	Late         OutcomeKind = "late"
	LimitReached OutcomeKind = "limit_reached"
	NoCategories OutcomeKind = "no_categories"
	Removed      OutcomeKind = "removed"
)

// Outcome is one Indexer's line in a search, a skip, or why a call waits.
type Outcome struct {
	Kind     OutcomeKind
	Releases int
	// Until is set for limited, failing and limit_reached.
	Until time.Time
	// Since, Node and Lapsed are set for waiting: since when home is away,
	// and whether past the grace.
	Since  time.Time
	Node   role.NodeID
	Lapsed bool
	// Cause and Text are set for stopped, and Node is where it was met.
	Cause Cause
	Text  string
}

// gate is the Node u may run on for idx now, else the Outcome that says
// why it can't: stopped, limited or failing inside its backoff, at its
// limit, or waiting for a Node. It is the one rule for every call and the
// Indexer page.
func gate(idx Indexer, st Status, used Usage, r Roster, u Use, now time.Time) (role.NodeID, *Outcome) {
	switch {
	case st.State == Stopped:
		return role.NodeID{}, &Outcome{Kind: IsStopped, Cause: st.Cause, Text: st.Text, Node: st.Node}
	case st.State == Limited && now.Before(st.Until):
		return role.NodeID{}, &Outcome{Kind: IsLimited, Until: st.Until}
	case st.State == Failing && now.Before(st.Until):
		return role.NodeID{}, &Outcome{Kind: IsFailing, Until: st.Until}
	}
	if until := used.free(u); now.Before(until) {
		return role.NodeID{}, &Outcome{Kind: LimitReached, Until: until}
	}
	return route(r, idx, st)
}

func (u Usage) free(use Use) time.Time {
	switch use {
	case UseSearch, UseRecent:
		return u.SearchesFree
	case UseFetch:
		return u.GrabsFree
	case UseTest:
	}
	return time.Time{}
}
