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

// Ask is what a Grab's client is chosen from: the Downloading Node its
// fetch picked by the Work's root, and the Tracker its Release reached.
type Ask struct {
	Node    role.NodeID
	Tracker indexers.TrackerID
}

// Route is the one function that chooses the client a Grab goes to: the
// client a.Node's New Grabs go to, its built-in client by default and
// never an off one. S29 (Trackers and seeding goals) puts Tracker
// routing in front of it here. It reads in the caller's tx.
func Route(ctx context.Context, r storage.Reader[file], a Ask) (Target, error) {
	id, err := db.New(r).GetGrabsTo(ctx, a.Node)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return Target{Node: a.Node}, nil
	case err != nil:
		return Target{}, err
	}
	return Target{Node: a.Node, Client: &id}, nil
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
