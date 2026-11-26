// Package access is the pure model of S43 (Member access to Media Servers):
// the states of one Member's access to one Media Server, the events that
// move it, the outcomes of the remote calls behind them and the one Table
// of every move. mediaservers runs a move and records it. This package
// answers which move a row makes and what a viewer may start on it.
package access

import (
	"fmt"
	"slices"
)

// State is a Media Server access row's state, as stored and sent.
type State string

const (
	// None is no row: the From of an offer and the To of a move that
	// deletes the row.
	None     State = ""
	Offered  State = "offered"
	Waiting  State = "waiting" // plex_tv only
	Shared   State = "shared"  // plex_tv only
	Making   State = "making"  // users only
	Active   State = "active"
	Removing State = "removing"
	Removed  State = "removed" // users only
)

// States are the stored states, in the order the schema's CHECK and the
// API list them.
func States() []State {
	return []State{Offered, Waiting, Shared, Making, Active, Removing, Removed}
}

// Pending is whether the Member still has something to do or wait for.
func (s State) Pending() bool {
	return s == Offered || s == Waiting || s == Shared || s == Making
}

// MemberSees is whether the row shows to its Member.
func (s State) MemberSees() bool { return s != Removing && s != Removed }

// NeedsYou is whether the row is listed under Needs you.
func (s State) NeedsYou() bool { return s == Waiting || s == Shared || s == Removing }

// Way is how a row's server gives access, stored on the row.
type Way string

const (
	// WayPlexTV is a share or Plex Home invite through plex.tv, from the
	// Controller.
	WayPlexTV Way = "plex_tv"
	// WayUsers is a user account on the server, along a route.
	WayUsers Way = "users"
)

func Ways() []Way { return []Way{WayPlexTV, WayUsers} }

// Event is what moves a row. The ones a person starts are its Actions.
type Event string

const (
	Offer       Event = "offer"       // Admin, from the Account page; the Granter on redemption
	PlexSignIn  Event = "plexSignIn"  // the Controller, when the Member's PIN completes
	AnotherPlex Event = "anotherPlex" // Member: Use another Plex account
	ShareFriend Event = "shareFriend" // Admin: Share on Plex
	ShareHome   Event = "shareHome"   // Admin: Add to Plex Home
	MarkAdded   Event = "markAdded"   // Admin: Share by hand instead, then Mark as added
	Check       Event = "check"       // Admin, Member, and the Plex sign-in task
	MakeAccount Event = "makeAccount" // Member: Make my account
	Created     Event = "created"     // the Controller: the create returned a user id
	Finish      Event = "finish"      // the Controller: policy, password and name set
	Unmake      Event = "unmake"      // the Controller: a failed step, a retry, a start, a roster change
	Link        Event = "link"        // Admin: Link an existing account
	Remove      Event = "remove"      // Admin: Remove access
	RetryRemove Event = "retryRemove" // Admin: Try again on a removing row
	Leave       Event = "leave"       // the Granter: the Account is being deleted
)

// Outcome is what the remote side said: the branch a move takes.
type Outcome string

const (
	// Done is the event's only outcome, or its success.
	Done Outcome = ""
	// NoIdentity is an offer to a Member who holds no Plex identity.
	NoIdentity   Outcome = "noIdentity"
	IdentityHeld Outcome = "identityHeld"
	// AlreadyThere is the server already shared with the Plex account, or
	// the account already in the Admin's Plex Home.
	AlreadyThere Outcome = "alreadyThere"
	NotThere     Outcome = "notThere"
	Accepted     Outcome = "accepted"
	// NotAccepted is an accept that failed, or no token held to make it.
	NotAccepted Outcome = "notAccepted"
	// Refused is plex.tv declining a sharing call.
	Refused Outcome = "refused"
	// Failed is any other remote failure.
	Failed Outcome = "failed"
	// Gone is the server's 404 for the row's user.
	Gone Outcome = "gone"
	// Left is Done for a row whose Account is already deleted, which keeps
	// nothing.
	Left Outcome = "left"
)

// By is who may start an event.
type By uint8

const (
	ByAdmin By = 1 << iota
	ByMember
	ByController
)

// Mark is what a move sets on the row besides its state.
type Mark uint8

const (
	MarkByHand    Mark = 1 << iota
	SetPlexHome        // plex_home = 1
	ClearPlexHome      // plex_home = 0: a friend share
	ClearPlex          // plex_id, plex_uuid and shared_server_id to NULL: Use another Plex account
	ClearUser          // server_user_id and username to NULL
	ClearAccount       // account_id to NULL: a Leave that couldn't finish outlives the Account
	KeepProblem        // the move records the call's problem; every other move clears it
)

// Transition is one move of the Table.
type Transition struct {
	From State
	On   Event
	When Outcome
	To   State
	By   By
	Way  Way
	Mark Mark
}

// MoveError is a move the Table doesn't hold.
type MoveError struct {
	From State
	On   Event
	When Outcome
	Way  Way
}

func (e *MoveError) Error() string {
	return fmt.Sprintf("access: no %s move from %q on %s with outcome %q", e.Way, e.From, e.On, e.When)
}

// Next is the move from from on on when the remote side said when.
func Next(from State, on Event, when Outcome, way Way) (Transition, error) {
	i := slices.IndexFunc(Table, func(t Transition) bool {
		return t.From == from && t.On == on && t.When == when && t.Way == way
	})
	if i < 0 {
		return Transition{}, &MoveError{From: from, On: on, When: when, Way: way}
	}
	return Table[i], nil
}

// Must is Next for a move the caller knows the Table holds. It panics
// otherwise, as regexp.MustCompile does.
func Must(from State, on Event, when Outcome, way Way) Transition {
	t, err := Next(from, on, when, way)
	if err != nil {
		panic(err)
	}
	return t
}

// Starts is whether on can start from from at all.
func Starts(from State, on Event, way Way) bool {
	return startsBy(from, on, way, ByAdmin|ByMember|ByController)
}

func startsBy(from State, on Event, way Way, by By) bool {
	return slices.ContainsFunc(Table, func(t Transition) bool {
		return t.From == from && t.On == on && t.Way == way && t.By&by != 0
	})
}

// Past is whether from was reached by some outcome of on, so a repeat of
// on answers the row as it is.
func Past(from State, on Event, way Way) bool {
	return slices.ContainsFunc(Table, func(t Transition) bool {
		return t.To == from && t.On == on && t.Way == way
	})
}

// Facts are what Actions needs beyond the state.
type Facts struct {
	State    State
	Way      Way
	PlexHome bool
	// PlexInUse is whether one of the Account's Plex rows is shared or
	// active.
	PlexInUse bool
	// SharesWork is whether the Admin's sign-in can share through plex.tv.
	SharesWork bool
	// HomeWorks is whether the Admin's sign-in can invite into the Plex
	// Home now, Plex Pass included.
	HomeWorks bool
}

// screen is every event a person starts, in the order the screen shows
// them. homeFirst is the order for a row offered as Plex Home.
var (
	screen    = []Event{ShareFriend, ShareHome, MarkAdded, Check, AnotherPlex, MakeAccount, Link, Offer, RetryRemove, Remove}
	homeFirst = []Event{ShareHome, ShareFriend, MarkAdded, Check, AnotherPlex, MakeAccount, Link, Offer, RetryRemove, Remove}
)

// Actions is what by may start on the row, in the order the screen shows
// them, with the share the row was offered as first.
func Actions(f Facts, by By) []Event {
	order := screen
	if f.PlexHome {
		order = homeFirst
	}
	out := []Event{}
	for _, on := range order {
		if gate, ok := gates[on]; startsBy(f.State, on, f.Way, by) && (!ok || gate(f)) {
			out = append(out, on)
		}
	}
	return out
}

// gates are the events Actions offers only on some rows. Offer is an
// action only where it moves the row: on an offered or waiting row it is
// the Account page's choice of Friend or Plex Home.
var gates = map[Event]func(Facts) bool{
	AnotherPlex: func(f Facts) bool { return !f.PlexInUse },
	ShareFriend: func(f Facts) bool { return f.SharesWork },
	ShareHome:   func(f Facts) bool { return f.HomeWorks },
	Offer: func(f Facts) bool {
		return slices.ContainsFunc(Table, func(t Transition) bool {
			return t.From == f.State && t.On == Offer && t.Way == f.Way && t.To != f.State
		})
	},
}
