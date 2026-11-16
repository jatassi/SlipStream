// Package notifying is the Notifying Role's wire: the link service that
// sends mail, posts HTTP requests and runs scripts from the Node with the
// Role, the Client the Controller calls it through, the courier that hands
// the Controller's mail to that Node, and the Role's owner.
package notifying

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	notifyv1 "github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1/notifyv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// Owner is the Notifying Role's owner. It keeps no work, and a Node that
// can't reach the SMTP server can't gain the Role.
var Owner = nodes.Owner{Role: role.Notifying, Held: nodes.HoldsNothing, Unassigned: nodes.MovesNothing, Reach: targets}

// targets is the SMTP server's host and port, none while mail isn't set
// up. Only implicit TLS starts with a handshake.
func targets(ctx context.Context, r storage.Reader[storage.Library]) ([]nodes.Target, error) {
	s, err := mail.ReadSettings(ctx, r)
	if err != nil || s.Host == "" {
		return nil, err
	}
	return []nodes.Target{{Target: nodesystem.Target{Host: s.Host, Port: s.Port, TLS: s.Security == mail.TLS}, What: "the mail server", Block: true}}, nil
}

// Dial reaches a host from the Node.
type Dial func(ctx context.Context, network, address string) (net.Conn, error)

// Serve is NotifyingService on a Node: Send and Post dial through dial,
// and Run runs scripts through run.
func Serve(t mail.Timings, dial Dial, run Runner) notifyv1connect.NotifyingServiceHandler {
	return server{timings: t, dial: dial, run: run, client: postClient(dial)}
}

type server struct {
	timings mail.Timings
	dial    Dial
	run     Runner
	client  *http.Client
}

func (s server) Send(ctx context.Context, req *notifyv1.SendRequest) (*notifyv1.SendResponse, error) {
	out, err := mail.SMTP{Server: serverOf(req.GetServer()), Timings: s.timings, Dial: s.dial}.Send(ctx, messageOf(req.GetMessage()))
	if err != nil {
		return nil, err
	}
	return &notifyv1.SendResponse{Kind: string(out.Kind), Reply: out.Reply, Permanent: out.Permanent, Step: string(out.Step)}, nil
}

func (s server) Post(ctx context.Context, req *notifyv1.PostRequest) (*notifyv1.PostResponse, error) {
	r, err := requestOf(req)
	if err != nil {
		return nil, rpc.Errorf(rpc.CodeInvalidArgument, "%v", err)
	}
	return responseToWire(post(ctx, s.client, r)), nil
}

func (s server) Run(ctx context.Context, req *notifyv1.RunRequest) (*notifyv1.RunResponse, error) {
	return ranToWire(s.run.Run(ctx, scriptOf(req))), nil
}

// Roster is the board the courier picks a Node from.
type Roster interface {
	Board() *nodes.Board
}

// Courier hands each message to the Node that sends notifications and
// mail: the Controller while it may, else the first other Node that may,
// over the link.
type Courier struct {
	Roster  Roster
	Self    role.NodeID
	Cluster *rpc.Cluster
}

var _ mail.Courier = Courier{}

// Deliver fails the message, with why, while no Node may send it or the
// one picked doesn't answer.
func (c Courier) Deliver(ctx context.Context, s mail.Server, m mail.Message) (mail.Outcome, error) {
	board := c.Roster.Board()
	who := board.Who(role.Notifying)
	node, ok := who.Pick(c.Self)
	if !ok {
		return mail.Outcome{Kind: mail.Failed, Reply: who.Why()}, nil
	}
	out, err := Client{Cluster: c.Cluster}.Send(ctx, node, s, m)
	if err != nil {
		v, _ := board.Node(node)
		return mail.Outcome{Kind: mail.Failed, Reply: fmt.Sprintf("%s, which sends notifications and mail, didn't respond: %v", v.Name, err)}, nil
	}
	return out, nil
}
