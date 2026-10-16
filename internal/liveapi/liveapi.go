// Package liveapi holds the event stream: one per browser tab, carrying the
// events its session may see.
package liveapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
)

type streamInput struct {
	LastEventID string `header:"Last-Event-ID" doc:"The id of the last event a dropped stream received"`
}

// Identities is where the stream finds the identity service. Only a
// Controller admits an Account, and its library.db is open.
type Identities interface {
	Identity() (*identity.Service, bool)
}

func Routes(r *api.Router, hub *live.Hub, ids Identities, t live.Timings) {
	o := r.Owner("live", api.Account)

	api.RegisterStream(o, api.Account, api.Op{
		ID:      "streamEvents",
		Method:  http.MethodGet,
		Path:    "/events",
		Summary: "Stream the events this session may see",
		Status:  http.StatusOK,
	}, hub.Events(), func(ctx context.Context, a identity.Account, in *streamInput, send api.Send) error {
		sessions, _ := ids.Identity()
		s := stream{sessions: sessions, timings: t, account: a, send: send, retry: t.Retry}
		return s.run(ctx, hub.Subscribe(live.Viewer{
			Account: a.Profile().ID.UUID,
			Session: a.Session().UUID,
			Admin:   a.Profile().Admin,
		}, in.LastEventID))
	})
}

type stream struct {
	sessions *identity.Service
	timings  live.Timings
	account  identity.Account
	send     api.Send
	// retry rides on the connection's first message, whatever it is.
	retry time.Duration
}

// run writes sub's messages until the stream ends. At each heartbeat it
// renews the session and sends a keepalive; a session that has ended sends
// live.session-ended with expired and closes.
func (s *stream) run(ctx context.Context, sub *live.Subscription) error {
	defer sub.Close()
	// A sign-out that deleted the session after authentication but before
	// Subscribe ended no subscription of ours; reading the session again
	// now catches it.
	err := s.sessions.Renew(ctx, s.account)
	if errors.Is(err, identity.ErrSignedOut) {
		s.write(api.StreamMessage{Data: live.SessionEnded{Reason: live.SignedOut}})
		return nil
	}
	if err != nil {
		return err
	}
	beat := time.Now().Add(s.timings.Heartbeat)
	for {
		wait, cancel := context.WithDeadline(ctx, beat)
		msg, err := sub.Next(wait)
		cancel()
		switch {
		case err == nil:
			if !s.write(api.StreamMessage{ID: msg.ID, Data: msg.Data}) {
				return nil
			}
		case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
			if ended, err := s.heartbeat(ctx); ended {
				return err
			}
			beat = time.Now().Add(s.timings.Heartbeat)
		default:
			return nil
		}
	}
}

// heartbeat reports whether the stream ended.
func (s *stream) heartbeat(ctx context.Context) (bool, error) {
	err := s.sessions.Renew(ctx, s.account)
	if errors.Is(err, identity.ErrSignedOut) {
		s.write(api.StreamMessage{Data: live.SessionEnded{Reason: live.Expired}})
		return true, nil
	}
	if err != nil {
		return true, err
	}
	return !s.write(api.StreamMessage{Comment: "keepalive"}), nil
}

// write reports whether the message reached the client.
func (s *stream) write(m api.StreamMessage) bool {
	m.Retry, s.retry = s.retry, 0
	return s.send(m) == nil
}
