package notifying

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/mail"
	notifyv1 "github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1/notifyv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
)

// Client is the Controller's side of NotifyingService. A Worker's answer
// is checked before it is used: a call whose answer is over a cap or
// missing a field fails with ErrUnchecked.
type Client struct {
	Cluster *rpc.Cluster
}

// ErrUnchecked is an answer from the Node that the Controller refuses.
var ErrUnchecked = errors.New("notifying: the Node's response is over a cap or missing a field")

func (c Client) Post(ctx context.Context, node role.NodeID, r Request) (Response, error) {
	res, err := notifyv1connect.NotifyingService.On(c.Cluster, node).Post(ctx, requestToWire(r))
	if err != nil {
		return Response{}, err
	}
	return responseOf(res)
}

func (c Client) Run(ctx context.Context, node role.NodeID, s Script) (Ran, error) {
	res, err := notifyv1connect.NotifyingService.On(c.Cluster, node).Run(ctx, scriptToWire(s))
	if err != nil {
		return Ran{}, err
	}
	return ranOf(res)
}

func (c Client) Send(ctx context.Context, node role.NodeID, s mail.Server, m mail.Message) (mail.Outcome, error) {
	res, err := notifyv1connect.NotifyingService.On(c.Cluster, node).Send(ctx, &notifyv1.SendRequest{Server: serverToWire(s), Message: messageToWire(m)})
	if err != nil {
		return mail.Outcome{}, err
	}
	return outcomeOf(res)
}

// maxErr bounds the text a Node gives for why no answer came.
const maxErr = MaxBody

func responseOf(res *notifyv1.PostResponse) (Response, error) {
	status, body, why := int(res.GetStatus()), res.GetBody(), res.GetError()
	switch {
	case len(body) > MaxBody:
		return Response{}, fmt.Errorf("%w: a body of %d bytes", ErrUnchecked, len(body))
	case len(why) > maxErr:
		return Response{}, fmt.Errorf("%w: an error of %d bytes", ErrUnchecked, len(why))
	case why == "" && (status < 100 || status > 599):
		return Response{}, fmt.Errorf("%w: status %d", ErrUnchecked, status)
	case why != "" && status != 0:
		return Response{}, fmt.Errorf("%w: status %d with an error", ErrUnchecked, status)
	}
	header := http.Header{}
	for _, h := range res.GetHeaders() {
		if !waitHeader(h.GetName()) {
			return Response{}, fmt.Errorf("%w: header %q", ErrUnchecked, h.GetName())
		}
		header.Add(h.GetName(), h.GetValue())
	}
	return Response{Status: status, Header: header, Body: body, Err: why}, nil
}

func ranOf(res *notifyv1.RunResponse) (Ran, error) {
	switch {
	case len(res.GetTail()) > MaxTail:
		return Ran{}, fmt.Errorf("%w: a tail of %d bytes", ErrUnchecked, len(res.GetTail()))
	case len(res.GetError()) > maxErr:
		return Ran{}, fmt.Errorf("%w: an error of %d bytes", ErrUnchecked, len(res.GetError()))
	}
	return Ran{Exit: int(res.GetExit()), Tail: res.GetTail(), TimedOut: res.GetTimedOut(), Missing: res.GetMissing(), Err: res.GetError()}, nil
}

var steps = []mail.Step{"", mail.StepConnect, mail.StepFrom, mail.StepRcpt, mail.StepData}

func outcomeOf(res *notifyv1.SendResponse) (mail.Outcome, error) {
	kind, step := mail.OutcomeKind(res.GetKind()), mail.Step(res.GetStep())
	switch {
	case !slices.Contains([]mail.OutcomeKind{mail.Sent, mail.Failed, mail.NotSetUp}, kind):
		return mail.Outcome{}, fmt.Errorf("%w: outcome %q", ErrUnchecked, kind)
	case !slices.Contains(steps, step):
		return mail.Outcome{}, fmt.Errorf("%w: step %q", ErrUnchecked, step)
	case len(res.GetReply()) > maxErr:
		return mail.Outcome{}, fmt.Errorf("%w: a reply of %d bytes", ErrUnchecked, len(res.GetReply()))
	}
	return mail.Outcome{Kind: kind, Reply: res.GetReply(), Permanent: res.GetPermanent(), Step: step}, nil
}

const anyAddress = "any"

func requestToWire(r Request) *notifyv1.PostRequest {
	req := &notifyv1.PostRequest{Method: r.Method, Url: r.URL, Body: r.Body, TimeoutMs: r.Timeout.Milliseconds(), Dial: anyAddress}
	for name, values := range r.Header {
		for _, v := range values {
			req.Headers = append(req.Headers, &notifyv1.Header{Name: name, Value: v})
		}
	}
	return req
}

func requestOf(req *notifyv1.PostRequest) (Request, error) {
	if req.GetDial() != anyAddress {
		return Request{}, fmt.Errorf("this Node has no dial rule %q", req.GetDial())
	}
	header := http.Header{}
	for _, h := range req.GetHeaders() {
		header.Add(h.GetName(), h.GetValue())
	}
	return Request{Method: req.GetMethod(), URL: req.GetUrl(), Header: header, Body: req.GetBody(), Timeout: time.Duration(req.GetTimeoutMs()) * time.Millisecond, Dial: AnyAddress}, nil
}

func responseToWire(r Response) *notifyv1.PostResponse {
	res := &notifyv1.PostResponse{Status: uint32(r.Status), Body: r.Body, Error: r.Err} //nolint:gosec // an HTTP status
	for name, values := range r.Header {
		for _, v := range values {
			res.Headers = append(res.Headers, &notifyv1.Header{Name: name, Value: v})
		}
	}
	return res
}

func scriptToWire(s Script) *notifyv1.RunRequest {
	return &notifyv1.RunRequest{Path: s.Path, Args: s.Args, Event: s.Event, EventFile: s.EventFile, StopMs: s.Stop.Milliseconds()}
}

func scriptOf(req *notifyv1.RunRequest) Script {
	return Script{Path: req.GetPath(), Args: req.GetArgs(), Event: req.GetEvent(), EventFile: req.GetEventFile(), Stop: time.Duration(req.GetStopMs()) * time.Millisecond}
}

func ranToWire(r Ran) *notifyv1.RunResponse {
	return &notifyv1.RunResponse{Exit: int32(r.Exit), Tail: r.Tail, TimedOut: r.TimedOut, Missing: r.Missing, Error: r.Err} //nolint:gosec // a process's exit code
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
	msg := &notifyv1.Message{To: &notifyv1.Address{Name: m.To.Name, Email: m.To.Email}, Subject: m.Subject, Html: m.HTML, Text: m.Text}
	for _, p := range m.Inline {
		msg.Inline = append(msg.Inline, &notifyv1.Inline{ContentId: p.ContentID, Type: p.Type, Name: p.Name, Body: p.Body})
	}
	return msg
}

func messageOf(m *notifyv1.Message) mail.Message {
	msg := mail.Message{To: mail.Address{Name: m.GetTo().GetName(), Email: m.GetTo().GetEmail()}, Subject: m.GetSubject(), HTML: m.GetHtml(), Text: m.GetText()}
	for _, p := range m.GetInline() {
		msg.Inline = append(msg.Inline, mail.Part{ContentID: p.GetContentId(), Type: p.GetType(), Name: p.GetName(), Body: p.GetBody()})
	}
	return msg
}
