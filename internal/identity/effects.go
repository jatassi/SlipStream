package identity

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

// effects is what one Write did that the world hears about once it
// commits. A Write that fails plays none of it.
type effects struct {
	// now is the Write's one clock reading.
	now      storage.Time
	ended    []endedSession
	sessions map[AccountID]struct{}
}

type endedSession struct {
	id  SessionID
	why live.EndReason
}

// endSession records that the Write deletes account's session id. The
// caller reads the id inside the same Write, so a cascade never leaves a
// stream running.
func (fx *effects) endSession(account AccountID, id SessionID, why live.EndReason) {
	fx.ended = append(fx.ended, endedSession{id: id, why: why})
	fx.sessionsChanged(account)
}

func (fx *effects) sessionsChanged(account AccountID) { fx.sessions[account] = struct{}{} }

// write is identity's only lib.Write. After commit it ends the sessions'
// streams, then publishes.
func (s *Service) write(ctx context.Context, fn func(ctx context.Context, q *db.Queries, fx *effects) error) error {
	fx := &effects{now: storage.Now(), sessions: map[AccountID]struct{}{}}
	err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		return fn(ctx, db.New(tx), fx)
	})
	if err != nil {
		return err
	}
	for _, e := range fx.ended {
		s.hub.EndSession(ctx, e.id.UUID, e.why)
	}
	for account := range fx.sessions {
		s.sessions.Publish(ctx, account.UUID, SessionsChanged{Account: account.String()})
	}
	return nil
}
