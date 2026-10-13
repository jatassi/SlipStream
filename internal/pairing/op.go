package pairing

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/node/op"
)

type JoinRequest struct {
	Code string `json:"code"`
}

// JoinReply holds one of: the joined line, or why the Claim changed
// nothing.
type JoinReply struct {
	Joined  string `json:"joined,omitempty"`
	Refused *Error `json:"refused,omitempty"`
}

// JoinOp joins a running Node to the Controller a Join Code names. A Node
// serves it once ready.
var JoinOp = op.Op[JoinRequest, JoinReply]{Name: "join"}

// String is the line slipstream join prints.
func (j Joined) String() string {
	return fmt.Sprintf("joined %s as %s (node %s); link up to %s", j.ControllerName, j.Name, j.NodeID, j.Link)
}
