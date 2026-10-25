// Package notifying is the Notifying Role: the link service that sends
// mail from the Node holding the Role, the courier that hands the
// Controller's mail to that Node, and the Role's owner.
package notifying

import (
	"context"
	"fmt"
	"net"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodesystem"
	notifyv1 "github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1/notifyv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// Owner is the Notifying Role's owner. It keeps no work, and a Node is
// tested against the SMTP server before it sends mail.
var Owner = nodes.Owner{Role: role.Notifying, Held: nodes.HoldsNothing, Unassigned: nodes.MovesNothing, Reach: target}

// target is the SMTP server's host and port, false while mail isn't set
// up. Only implicit TLS starts with a handshake.
func target(ctx context.Context, r storage.Reader[storage.Library]) (nodesystem.Target, bool, error) {
	s, err := mail.ReadSettings(ctx, r)
	if err != nil || s.Host == "" {
		return nodesystem.Target{}, false, err
	}
	return nodesystem.Target{Host: s.Host, Port: s.Port, TLS: s.Security == mail.TLS}, true, nil
}

// Dial reaches a host from the Node.
type Dial func(ctx context.Context, network, address string) (net.Conn, error)

// Serve is NotifyingService on a Node: it sends each message to the
// server the call carries, dialing it through dial.
func Serve(t mail.Timings, dial Dial) notifyv1connect.NotifyingServiceHandler {
	return sender{timings: t, dial: dial}
}

type sender struct {
	timings mail.Timings
	dial    Dial
}

func (s sender) Send(ctx context.Context, req *notifyv1.SendRequest) (*notifyv1.SendResponse, error) {
	out, err := mail.SMTP{Server: serverOf(req.GetServer()), Timings: s.timings, Dial: s.dial}.Send(ctx, messageOf(req.GetMessage()))
	if err != nil {
		return nil, err
	}
	return &notifyv1.SendResponse{Kind: string(out.Kind), Reply: out.Reply}, nil
}

// Roster is the board the courier picks a Node from.
type Roster interface {
	Board() *nodes.Board
}

// Courier hands each message to the Node that sends mail: the Controller
// while it may, else the first other Node that may, over the link.
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
	res, err := notifyv1connect.NotifyingService.On(c.Cluster, node).Send(ctx, &notifyv1.SendRequest{Server: serverToWire(s), Message: messageToWire(m)})
	if err != nil {
		v, _ := board.Node(node)
		return mail.Outcome{Kind: mail.Failed, Reply: fmt.Sprintf("%s, which sends mail, didn't respond: %v", v.Name, err)}, nil
	}
	return mail.Outcome{Kind: mail.OutcomeKind(res.GetKind()), Reply: res.GetReply()}, nil
}

func serverToWire(s mail.Server) *notifyv1.Server {
	return &notifyv1.Server{
		Host: s.Host, Port: uint32(s.Port), Security: string(s.Security), Username: s.Username, Password: s.Password,
		From: &notifyv1.Address{Name: s.From.Name, Email: s.From.Email},
	}
}

func serverOf(s *notifyv1.Server) mail.Server {
	return mail.Server{
		Host: s.GetHost(), Port: uint16(s.GetPort()), Security: mail.Security(s.GetSecurity()), //nolint:gosec // a port the Controller sent from a uint16
		Username: s.GetUsername(), Password: s.GetPassword(),
		From: mail.Address{Name: s.GetFrom().GetName(), Email: s.GetFrom().GetEmail()},
	}
}

func messageToWire(m mail.Message) *notifyv1.Message {
	return &notifyv1.Message{To: &notifyv1.Address{Name: m.To.Name, Email: m.To.Email}, Subject: m.Subject, Html: m.HTML, Text: m.Text}
}

func messageOf(m *notifyv1.Message) mail.Message {
	return mail.Message{To: mail.Address{Name: m.GetTo().GetName(), Email: m.GetTo().GetEmail()}, Subject: m.GetSubject(), HTML: m.GetHtml(), Text: m.GetText()}
}
