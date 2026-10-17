package identity

import (
	"context"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/mail"
)

// StartEmailChange emails a confirm link to email, refusing ErrEmailTaken
// and ErrAddressUnset, and waits for the outcome. The email changes only
// when the link is followed, so every Account's email is proven.
func (s *Service) StartEmailChange(ctx context.Context, a Account, email string) (mail.Outcome, error) {
	email = normalizeEmail(email)
	me := a.s.profile
	var out issued
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if err := emailFree(ctx, q, email, me.ID); err != nil {
			return err
		}
		var err error
		out, err = s.issue(ctx, q, fx, emailKind, linkTarget{email: email, account: me.ID}, AccountID{})
		return err
	})
	if err != nil {
		return mail.Outcome{}, err
	}
	set, err := readSettings(ctx, db.New(s.lib))
	if err != nil {
		return mail.Outcome{}, err
	}
	data := linkMailData{ServerName: set.Address.Host(), Email: email, URL: out.url, Expires: when(out.expires)}
	return deliver(ctx, s, confirmEmailMail, mail.Address{Name: me.DisplayName, Email: email}, data)
}

// ConfirmEmail follows an email link: in one Write it marks the link used,
// refuses a turned-off Account and a taken email, changes the email and
// tells the old address. It signs nobody in.
func (s *Service) ConfirmEmail(ctx context.Context, _ Anyone, t LinkToken) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		link, err := redeem(ctx, q, fx, emailKind, t)
		if err != nil {
			return err
		}
		row, err := getAccount(ctx, q, link.AccountID)
		if err != nil {
			return err
		}
		if !row.DisabledAt.IsZero() {
			return ErrAccountDisabled
		}
		return changeEmail(ctx, q, fx, row, link.Email)
	})
}

func changeEmail(ctx context.Context, q *db.Queries, fx *effects, row db.IdentityAccount, email string) error {
	if email == row.Email {
		return nil
	}
	if err := emailFree(ctx, q, email, row.ID); err != nil {
		return err
	}
	if err := q.SetEmail(ctx, db.SetEmailParams{Email: email, ID: row.ID}); err != nil {
		return err
	}
	m, err := emailChangedMail.Render(mail.Address{Name: row.DisplayName, Email: row.Email}, emailChangedData{NewEmail: email})
	if err != nil {
		return err
	}
	fx.notice(m)
	fx.accountsChanged()
	fx.sessionsChanged(row.ID)
	return nil
}
