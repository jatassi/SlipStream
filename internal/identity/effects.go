package identity

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/storage"
)

// effects is what one Write did that the world hears about once it
// commits. A Write that fails plays none of it.
type effects struct {
	// now is the Write's one clock reading.
	now      storage.Time
	ended    []endedSession
	regated  []AccountID
	sessions map[AccountID]struct{}
	passkeys map[AccountID]struct{}
	accounts bool
	links    bool
	address  bool
	notices  []mail.Message
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

func (fx *effects) endSessions(account AccountID, ids []SessionID, why live.EndReason) {
	for _, id := range ids {
		fx.endSession(account, id, why)
	}
}

func (fx *effects) sessionsChanged(account AccountID) { fx.sessions[account] = struct{}{} }

// passkeysChanged reaches account's streams and, for the passkey count on
// the Accounts list, the Admins'.
func (fx *effects) passkeysChanged(account AccountID) {
	fx.passkeys[account] = struct{}{}
	fx.accountsChanged()
}

func (fx *effects) passkeyUsed(account AccountID) { fx.passkeys[account] = struct{}{} }

// regate records that account's access changed: its streams end with
// live.access-changed, and its tabs read the session again.
func (fx *effects) regate(account AccountID) {
	fx.regated = append(fx.regated, account)
	fx.accountsChanged()
}

func (fx *effects) accountsChanged() { fx.accounts = true }

func (fx *effects) linksChanged() { fx.links = true }

func (fx *effects) addressChanged() { fx.address = true }

// notice records an email that goes out after the commit, which nobody
// waits for.
func (fx *effects) notice(m mail.Message) { fx.notices = append(fx.notices, m) }

// write is identity's only lib.Write. It holds the last-Admin rule: no
// Write may take a Library that has an enabled Admin to none. After commit
// it ends the sessions' streams, then re-gated Accounts' streams, then
// publishes, then posts the notices.
func (s *Service) write(ctx context.Context, fn func(ctx context.Context, q *db.Queries, fx *effects) error) error {
	fx := &effects{now: storage.Now(), sessions: map[AccountID]struct{}{}, passkeys: map[AccountID]struct{}{}}
	err := s.lib.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		q := db.New(tx)
		before, err := q.CountEnabledAdmins(ctx)
		if err != nil {
			return err
		}
		if err := fn(ctx, q, fx); err != nil {
			return err
		}
		after, err := q.CountEnabledAdmins(ctx)
		if err != nil {
			return err
		}
		if before > 0 && after == 0 {
			return ErrLastAdmin
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, e := range fx.ended {
		s.hub.EndSession(ctx, e.id.UUID, e.why)
	}
	for _, account := range fx.regated {
		s.hub.EndAccountStreams(ctx, account.UUID)
	}
	for account := range fx.sessions {
		s.sessions.Publish(ctx, account.UUID, SessionsChanged{Account: account.String()})
	}
	for account := range fx.passkeys {
		s.passkeys.Publish(ctx, account.UUID, PasskeysChanged{Account: account.String()})
	}
	if fx.accounts {
		s.accounts.Publish(ctx, AccountsChanged{})
	}
	if fx.links {
		s.links.Publish(ctx, LinksChanged{})
	}
	if fx.address {
		s.address.Publish(ctx, AddressChanged{})
	}
	for _, m := range fx.notices {
		s.outbox.Post(m)
	}
	return nil
}
