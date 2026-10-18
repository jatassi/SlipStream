// Package role is sync dispatch: a Role's holder half keeps the intent the
// Controller derives for it, reports facts, and sends results the
// Controller applies exactly once, over one Sync stream per session or in
// process on the Controller.
package role

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/storage"
)

// Name is a Role. Code names one by its constant; text from outside is
// parsed with ParseName.
type Name string

const (
	Searching       Name = "searching"
	Downloading     Name = "downloading"
	Storage         Name = "storage"
	Converting      Name = "converting"
	Subtitles       Name = "subtitles"
	Metadata        Name = "metadata"
	MediaServerLink Name = "media-server-link"
	Notifying       Name = "notifying"
	Relaying        Name = "relaying"
	Serving         Name = "serving"
)

// NodeID is a Node's id, nodes_node.id. The nodes package declares its id
// as this type, so role and rpc never import it.
type NodeID struct{ storage.UUID }

// Standing is whether a holder is in step with the Controller.
type Standing uint8

const (
	// Unreachable: no session to the Controller. Role code keeps the last
	// intent and its Reports commit locally.
	Unreachable Standing = iota
	// Live: in step, after the session's CaughtUp.
	Live
	// Mismatched: the link is up but the Controller runs another version.
	Mismatched
)

func (s Standing) String() string {
	switch s {
	case Unreachable:
		return "unreachable"
	case Live:
		return "live"
	case Mismatched:
		return "mismatched"
	}
	return fmt.Sprintf("Standing(%d)", uint8(s))
}
