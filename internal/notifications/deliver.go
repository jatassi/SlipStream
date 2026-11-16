package notifications

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/httpretry"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifying"
	"github.com/jatassi/slipstream/internal/role"
)

// deliver is the one place providers' types meet the wire. It hands o to
// node and answers the reply as the Controller checked it. Email sends
// once to each address not in accepted.
func (s *Service) deliver(ctx context.Context, node role.NodeID, o providers.Outbound, accepted []string) providers.Reply {
	send := s.cfg.Timings.Send
	switch {
	case o.Post != nil:
		// The Node's own timeout ends the request first; the call's
		// deadline covers a Node that never answers.
		call, cancel := context.WithTimeout(ctx, 2*send)
		defer cancel()
		res, err := s.cfg.Courier.Post(call, node, toRequest(*o.Post, send))
		if err != nil {
			return providers.Reply{Down: s.down(node, err)}
		}
		r := fromResponse(res)
		return providers.Reply{Post: &r}
	case o.Run != nil:
		call, cancel := context.WithTimeout(ctx, o.Run.Stop+send)
		defer cancel()
		res, err := s.cfg.Courier.Run(call, node, toScript(*o.Run))
		if err != nil {
			return providers.Reply{Down: s.down(node, err)}
		}
		r := fromRan(res)
		return providers.Reply{Run: &r}
	}
	server, ok, err := s.cfg.Mail.Server(ctx)
	if err != nil {
		s.cfg.Log.Error("read the mail server", "err", err.Error())
		return providers.Reply{Down: providers.Reason{Code: providers.Unreadable, Node: node}}
	}
	var replies []providers.MailReply
	for _, m := range o.Mail {
		switch {
		case slices.Contains(accepted, m.To):
			continue
		case !ok:
			replies = append(replies, providers.MailReply{To: m.To, NotSetUp: true})
			continue
		}
		call, cancel := context.WithTimeout(ctx, 2*send)
		out, err := s.cfg.Courier.Send(call, node, server, toMail(m))
		cancel()
		if err != nil {
			return providers.Reply{Down: s.down(node, err)}
		}
		replies = append(replies, fromOutcome(m.To, out))
	}
	return providers.Reply{Mail: replies}
}

// logKept writes a Sent verdict's response, such as Apprise's 424, at
// info. Its secret values are already masked.
func (s *Service) logKept(c Connection, v providers.Verdict) {
	if v.Outcome == providers.Sent && len(v.Response) > 0 {
		s.cfg.Log.Info("a provider took a message with a response", "connection", s.label(c), "response", string(v.Response))
	}
}

// down is why no answer came from node.
func (s *Service) down(node role.NodeID, err error) providers.Reason {
	s.cfg.Log.Info("a Node didn't answer a notification", "node", node, "err", httpretry.WithoutURL(err).Error())
	if errors.Is(err, notifying.ErrUnchecked) {
		return providers.Reason{Code: providers.Unreadable, Node: node}
	}
	return providers.Reason{Code: providers.NodeSilent, Node: node}
}

func toRequest(r providers.Request, timeout time.Duration) notifying.Request {
	return notifying.Request{Method: r.Method, URL: r.URL, Header: r.Header, Body: r.Body, Timeout: timeout, Dial: notifying.AnyAddress}
}

// fromResponse drops why no response came: a status of 0 says so.
func fromResponse(r notifying.Response) providers.Response {
	return providers.Response{Status: r.Status, Header: r.Header, Body: r.Body}
}

func toScript(s providers.Script) notifying.Script {
	return notifying.Script{Path: s.Path, Args: s.Args, Event: s.Event, EventFile: s.EventFile, Stop: s.Stop}
}

// fromRan answers a script the Node couldn't start as one that failed with
// why as its output.
func fromRan(r notifying.Ran) providers.Ran {
	if r.Err != "" {
		return providers.Ran{Exit: r.Exit, Tail: []byte(r.Err)}
	}
	return providers.Ran{Exit: r.Exit, Tail: r.Tail, TimedOut: r.TimedOut, Missing: r.Missing}
}

func toMail(m providers.Mail) mail.Message {
	parts := make([]mail.Part, len(m.Inline))
	for i, p := range m.Inline {
		parts[i] = mail.Part{ContentID: p.ContentID, Type: p.Type, Name: p.Name, Body: p.Body}
	}
	return mail.Message{To: mail.Address{Email: m.To}, Subject: m.Subject, HTML: m.HTML, Text: m.Text, Inline: parts}
}

func fromOutcome(to string, o mail.Outcome) providers.MailReply {
	return providers.MailReply{To: to, Sent: o.Kind == mail.Sent, NotSetUp: o.Kind == mail.NotSetUp, Permanent: o.Permanent, Step: providers.MailStep(o.Step)}
}
