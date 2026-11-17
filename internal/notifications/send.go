package notifications

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/role"
)

// maxResponse is how much of a refusing response a given-up message keeps.
const maxResponse = 1 << 10

// send is one job outside any Write.
func (s *Service) send(ctx context.Context, j job) result {
	r := result{conn: j.conn.ID, event: j.event.ID, kind: j.event.Kind, revision: j.conn.Revision}
	secrets, err := s.open(j.conn.ID, j.conn.Sealed)
	if err != nil {
		s.cfg.Log.Error("open a connection's secrets", "connection", j.conn.ID, "err", err.Error())
		r.verdict = providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: secretsUnreadable}}
		return r
	}
	r.verdict, _ = s.attempt(ctx, j.conn, secrets, j.event, j.accepted)
	return r
}

// attempt renders e for c, builds the provider's request and sends it from
// the Node that sends for c now. A 413 on a message with a poster is built
// and sent once more without it.
// It answers the Node it picked, zero when none may send.
func (s *Service) attempt(ctx context.Context, c Connection, secrets providers.Values, e event, accepted []string) (providers.Verdict, role.NodeID) {
	p := s.provider(c.Provider)
	node, why, ok := s.pick(c)
	if !ok {
		return providers.Verdict{Outcome: providers.Transient, Reason: why}, role.NodeID{}
	}
	msg, err := s.render(ctx, e, s.cfg.Roster.Name(node))
	if err != nil {
		s.cfg.Log.Error("render a notification", "event", e.ID, "err", err.Error())
		return providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.Refused}}, node
	}
	pay := providers.Payload{
		Kind: e.Kind, At: e.At, Message: msg, Settings: c.Settings, Secrets: secrets,
		Poster: s.poster(ctx, p.Image, msg.Poster, c.Posters), Endpoints: s.cfg.Endpoints, Accepted: accepted,
	}
	v := s.try(ctx, p, node, pay)
	if v.Outcome == providers.RefusedContent && v.WithoutImage && pay.Poster != nil {
		pay.Poster = nil
		v = s.try(ctx, p, node, pay)
	}
	v.Response = masked(v.Response, secrets)
	s.logKept(c, v)
	return v, node
}

func (s *Service) try(ctx context.Context, p providers.Provider, node role.NodeID, pay providers.Payload) providers.Verdict {
	out, err := p.Build(pay)
	if err != nil {
		return providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.Refused}}
	}
	v := p.Classify(s.deliver(ctx, node, out, pay.Accepted), time.Now())
	if v.Reason.Node.IsZero() {
		v.Reason.Node = node
	}
	return v
}

// pick is the Node that sends for c now: a Custom Script's own Node, or
// the Node with the Notifying Role, the Controller first.
func (s *Service) pick(c Connection) (role.NodeID, providers.Reason, bool) {
	if c.script() {
		if !s.cfg.Roster.InStep(c.Node) {
			return role.NodeID{}, providers.Reason{Code: nodeOffline, Node: c.Node}, false
		}
		return c.Node, providers.Reason{}, true
	}
	node, ok := s.cfg.Roster.Pick(role.Notifying, s.cfg.Roster.Self())
	if !ok {
		return role.NodeID{}, providers.Reason{Code: providers.NoNode}, false
	}
	return node, providers.Reason{}, true
}

// render is e's message now, its link on the Address, or none while the
// Address isn't set.
func (s *Service) render(ctx context.Context, e event, sender string) (notice.Message, error) {
	msg, err := s.catalog.entry(e.Kind).RenderStored(e.Subject, notice.Context{Now: time.Now(), Zone: s.cfg.Zone, Sender: sender})
	if err != nil || msg.Link.Path == "" {
		return msg, err
	}
	address, err := identity.ReadAddress(ctx, s.cfg.Library)
	if err != nil {
		return notice.Message{}, err
	}
	if address.IsZero() {
		msg.Link = notice.Link{}
		return msg, nil
	}
	msg.Link.Path = address.String() + msg.Link.Path
	return msg, nil
}

// posterNames names a poster file by its type.
var posterNames = map[string]string{"image/jpeg": "poster.jpg", "image/png": "poster.png", "image/webp": "poster.webp", "image/svg+xml": "poster.svg"}

// poster is the medium poster the way the provider takes it, or none when
// posters are off, the message has none, or the store can't produce it.
func (s *Service) poster(ctx context.Context, way providers.ImageWay, ref artwork.Ref, on bool) *providers.Poster {
	if !on || ref.IsZero() {
		return nil
	}
	switch way {
	case providers.Bytes:
		img, err := s.cfg.Artwork.Image(ctx, ref, artwork.SizeMedium)
		if err != nil {
			s.cfg.Log.Info("send a notification without its poster", "err", err.Error())
			return nil
		}
		return &providers.Poster{Bytes: img.Body, Type: img.Type, Name: posterNames[img.Type]}
	case providers.PublicURL, providers.BodyURL:
		u, err := s.cfg.Artwork.PublicURL(ref, artwork.SizeMedium)
		if err != nil {
			s.cfg.Log.Info("send a notification without its poster", "err", err.Error())
			return nil
		}
		return &providers.Poster{URL: u}
	case providers.NoImage:
	}
	return nil
}

// masked is a response with every secret value of the connection replaced
// by [secret], cut to maxResponse on a rune boundary.
func masked(response []byte, secrets providers.Values) []byte {
	if response == nil {
		return nil
	}
	text := string(response)
	for _, raw := range secrets {
		for _, v := range leaves(raw) {
			if v != "" {
				text = strings.ReplaceAll(text, v, "[secret]")
			}
		}
	}
	if len(text) > maxResponse {
		text = text[:maxResponse]
	}
	return []byte(strings.ToValidUTF8(text, ""))
}

// leaves are the strings inside a stored value: the value itself, or each
// string in an object or a list, such as Webhook headers.
func leaves(raw json.RawMessage) []string {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	var out []string
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case string:
			out = append(out, v)
		case []any:
			for _, e := range v {
				walk(e)
			}
		case map[string]any:
			for _, e := range v {
				walk(e)
			}
		}
	}
	walk(v)
	return out
}
