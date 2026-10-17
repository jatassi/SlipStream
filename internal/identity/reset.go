package identity

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/mail"
)

// IssueResetLink issues a Reset link for id and ends every session it
// holds, in one Write, then emails the link and waits for the outcome. The
// token is never kept: the URL is only in this answer.
func (s *Service) IssueResetLink(ctx context.Context, by Admin, id AccountID, removePasskeys bool) (Issued, error) {
	issuerID, name := issuer(by)
	var out issued
	var row db.IdentityAccount
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		var err error
		if row, err = getAccount(ctx, q, id); err != nil {
			return err
		}
		if !row.DisabledAt.IsZero() {
			return ErrAccountDisabled
		}
		if out, err = s.issue(ctx, q, fx, resetKind, linkTarget{email: row.Email, account: id, removePasskeys: removePasskeys}, issuerID); err != nil {
			return err
		}
		return endAllSessions(ctx, q, fx, id)
	})
	if err != nil {
		return Issued{}, err
	}
	set, err := readSettings(ctx, db.New(s.lib))
	if err != nil {
		return Issued{}, err
	}
	data := linkMailData{AdminName: orYourAdmin(name), ServerName: set.Address.Host(), Email: row.Email, URL: out.url, Expires: when(out.expires)}
	outcome, err := deliver(ctx, s, resetMail, mail.Address{Name: row.DisplayName, Email: row.Email}, data)
	if err != nil {
		return Issued{}, err
	}
	return Issued{
		Link: PendingLink{ID: out.id, Email: row.Email, IssuedBy: name, IssuedAt: out.issuedAt, ExpiresAt: out.expires},
		URL:  out.url,
		Mail: outcome,
	}, nil
}

// RedeemReset does, in one Write: marks the link used, refuses a turned-off
// Account, ends every session again, removes the passkeys when the link
// says so, sets the new PIN, which clears the PIN's failures, and signs in.
func (s *Service) RedeemReset(ctx context.Context, a Anyone, t LinkToken, pin PIN) (SignedIn, error) {
	if _, err := lookup(ctx, db.New(s.lib), resetKind, t, time.Now()); err != nil {
		return SignedIn{}, err
	}
	hash, err := s.pins.hash(ctx, pin)
	if err != nil {
		return SignedIn{}, err
	}
	return s.signInBy(ctx, a.client, func(ctx context.Context, q *db.Queries, fx *effects) (AccountID, error) {
		link, err := redeem(ctx, q, fx, resetKind, t)
		if err != nil {
			return AccountID{}, err
		}
		row, err := getAccount(ctx, q, link.AccountID)
		if err != nil {
			return AccountID{}, err
		}
		if !row.DisabledAt.IsZero() {
			return AccountID{}, ErrAccountDisabled
		}
		if err := endAllSessions(ctx, q, fx, row.ID); err != nil {
			return AccountID{}, err
		}
		if link.RemovePasskeys {
			if err := q.DeletePasskeys(ctx, row.ID); err != nil {
				return AccountID{}, err
			}
			fx.passkeysChanged(row.ID)
		}
		return row.ID, q.SetPIN(ctx, db.SetPINParams{PinHash: hash, ID: row.ID})
	})
}

func endAllSessions(ctx context.Context, q *db.Queries, fx *effects, account AccountID) error {
	ids, err := q.ListSessionIDs(ctx, account)
	if err != nil {
		return err
	}
	fx.endSessions(account, ids, live.Revoked)
	fx.accountsChanged()
	return q.DeleteAccountSessions(ctx, account)
}
