package downloadclients

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/downloadclients/internal/db"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Target is one Download Client: Node's built-in client when Client is
// nil, else the external client Client on Node.
type Target struct {
	Node   role.NodeID
	Client *ClientID
}

// Ask is what a Grab's client is chosen from: the Node that holds its
// Work's library root, the Indexer that found its Release and the Tracker
// that Release reached, and the client an Admin sent it to, nil for none.
type Ask struct {
	Root    role.NodeID
	Indexer indexers.ID
	Tracker indexers.TrackerID
	Sent    *Choice
	// Pinned makes a Sent client that can't take the Grab now a
	// *PinnedError instead of a fall-through to Tracker routing: the
	// client was an Admin's choice in a Manual search.
	Pinned bool
}

// PinnedError is a pinned client that can't take a Grab now: it is off or
// deleted, or its Node doesn't hold Downloading. Name is the client in
// words, such as "qBittorrent on Den".
type PinnedError struct {
	Sent  Choice
	Name  string
	Cause error
}

func (e *PinnedError) Error() string { return e.Name + " can't take the Grab: " + e.Cause.Error() }

func (e *PinnedError) Unwrap() error { return e.Cause }

// Routes is Tracker routing, as trackers answers it: the client a Grab of
// tracker that Indexer indexer found goes to, false when it goes by its
// Work's library root. It reads in the caller's tx.
type Routes interface {
	Route(ctx context.Context, r storage.Reader[file], tracker indexers.TrackerID, indexer indexers.ID) (Target, bool, error)
}

// RouteRoster is what routing a Grab reads of the Nodes.
type RouteRoster interface {
	// Route is where r work homed on home goes now. It fails with
	// *nodes.Unrouted while no Node may take it.
	Route(r role.Name, home role.NodeID) (role.NodeID, error)
	Holds(r role.Name, node role.NodeID) bool
	WrittenOff(r role.Name, node role.NodeID) bool
}

// Router chooses the client each Grab goes to.
type Router struct {
	Roster RouteRoster
	Routes Routes
}

// Route is the client a Grab goes to now: the one an Admin sent it to
// while that client may take it, a *PinnedError while a pinned one can't,
// else its Tracker's route, else the client where New Grabs go on the
// Downloading Node its Work's root routes to, the built-in client by default and never an off one. A
// Tracker's route may name a Node out of step, which the Grab waits for.
// It reads in the caller's tx, and fails with *nodes.Unrouted while no
// Node may take a Grab that goes by its root.
func (rt Router) Route(ctx context.Context, r storage.Reader[file], a Ask) (Target, error) {
	if a.Sent != nil {
		t, err := Usable(ctx, r, *a.Sent, rt.downloading)
		if err == nil || !Unusable(err) {
			return t, err
		}
		if a.Pinned {
			return Target{}, pinned(ctx, r, *a.Sent, err)
		}
	}
	if t, ok, err := rt.Routes.Route(ctx, r, a.Tracker, a.Indexer); err != nil || ok {
		return t, err
	}
	node, err := rt.Roster.Route(role.Downloading, a.Root)
	if err != nil {
		return Target{}, err
	}
	id, err := db.New(r).GetGrabsTo(ctx, node)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return Target{Node: node}, nil
	case err != nil:
		return Target{}, err
	}
	return Target{Node: node, Client: &id}, nil
}

// pinned is the *PinnedError of pinned client c, which Usable refused
// with cause.
func pinned(ctx context.Context, r storage.Reader[file], c Choice, cause error) error {
	names, err := namesIn(ctx, r)
	if err != nil {
		return err
	}
	name := names[c.BuiltInOn] + "'s built-in client"
	if c.Client != nil {
		row, err := db.New(r).GetClientNamed(ctx, *c.Client)
		if err != nil {
			return err
		}
		name = row.Name + " on " + names[row.NodeID]
	}
	return &PinnedError{Sent: c, Name: name, Cause: cause}
}

// downloading reports whether node may take a Grab's download now: it
// holds Downloading and the Admin hasn't written that off.
func (rt Router) downloading(node role.NodeID) bool {
	return rt.Roster.Holds(role.Downloading, node) && !rt.Roster.WrittenOff(role.Downloading, node)
}

// Unusable reports whether err is Usable refusing a client because it
// can't take a Grab now: it is off or deleted, or its Node doesn't hold
// Downloading.
func Unusable(err error) bool {
	_, off := errors.AsType[*ClientOffError](err)
	return off || errors.Is(err, ErrNotDownloading) || errors.Is(err, ErrNoClient)
}

// Choice is a client as an Admin picks it: an external client by id, or
// the built-in client of BuiltInOn when Client is nil.
type Choice struct {
	Client    *ClientID
	BuiltInOn role.NodeID
}

// ClientOffError refuses an off client, Name, a Grab can't go to.
type ClientOffError struct{ Name string }

func (e *ClientOffError) Error() string { return e.Name + " is off" }

// Usable is the Target c names when a Grab can go to it now: downloading
// says its Node holds Downloading, and an external client is on and not
// deleted. It fails with ErrNotDownloading, ErrNoClient or
// *ClientOffError, and reads in the caller's tx.
func Usable(ctx context.Context, r storage.Reader[file], c Choice, downloading func(role.NodeID) bool) (Target, error) {
	t := Target{Node: c.BuiltInOn}
	if c.Client != nil {
		e, err := ExternalOf(ctx, r, *c.Client)
		if err != nil {
			return Target{}, err
		}
		if !e.On {
			return Target{}, &ClientOffError{Name: e.Name}
		}
		t = Target{Node: e.Node, Client: &e.ID}
	}
	if !downloading(t.Node) {
		return Target{}, ErrNotDownloading
	}
	return t, nil
}
