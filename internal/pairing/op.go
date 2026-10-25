package pairing

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/node/op"
)

type JoinRequest struct {
	Code string `json:"code"`
	// NodePort is --node-port: the link.listen port the Join needs, 0 for
	// the Node's own.
	NodePort uint16 `json:"nodePort,omitempty"`
}

// JoinReply holds one of: the lines of a Claim awaiting its Controller's
// dial, the joined line, or why the Claim changed nothing. Awaiting comes
// first and again every minute; one of the others ends the op.
type JoinReply struct {
	Awaiting []string `json:"awaiting,omitempty"`
	Joined   string   `json:"joined,omitempty"`
	Refused  *Error   `json:"refused,omitempty"`
}

// JoinOp joins a running Node to the Controller a Join Code names. A Node
// serves it once ready.
var JoinOp = op.Op[JoinRequest, JoinReply]{Name: "join"}

// String is the line slipstream join prints.
func (j Joined) String() string {
	return fmt.Sprintf("joined %s as %s (node %s), linked to %s", j.ControllerName, j.Name, j.NodeID, j.Link)
}
