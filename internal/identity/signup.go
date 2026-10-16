package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/storage"
)

// PendingLink is a live Signup link as the Accounts page lists it.
type PendingLink struct {
	ID        LinkID
	Email     string
	IssuedBy  string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// Issued answers every issuing operation: the link, its URL, shown this
// once, and what became of its email.
type Issued struct {
	Link PendingLink
	URL  string
	Mail mail.Outcome
}

// IssueSignupLink issues a Signup link for email in one Write, refusing
// ErrAccountExists and ErrAddressUnset, then sends the Signup email and
// waits for its outcome. A live link for the same email is withdrawn, and
// the email says it replaces it.
func (s *Service) IssueSignupLink(ctx context.Context, by Admin, email string) (Issued, error) {
	email = normalizeEmail(email)
	id, name := issuer(by)
	var out issued
	err := s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if _, err := q.GetAccountByEmail(ctx, email); !errors.Is(err, storage.ErrNoRows) {
			if err == nil {
				return ErrAccountExists
			}
			return err
		}
		var err error
		out, err = s.issue(ctx, q, fx, signupKind, linkTarget{email: email}, id)
		return err
	})
	if err != nil {
		return Issued{}, err
	}
	set, err := readSettings(ctx, db.New(s.lib))
	if err != nil {
		return Issued{}, err
	}
	data := linkMailData{AdminName: orYourAdmin(name), ServerName: set.Address.Host(), Email: email, URL: out.url, Expires: when(out.expires)}
	if !out.replaces.IsZero() {
		data.Replaces = when(out.replaces)
	}
	outcome, err := deliver(ctx, s, signupMail, mail.Address{Email: email}, data)
	if err != nil {
		return Issued{}, err
	}
	return Issued{
		Link: PendingLink{ID: out.id, Email: email, IssuedBy: name, IssuedAt: out.issuedAt, ExpiresAt: out.expires},
		URL:  out.url,
		Mail: outcome,
	}, nil
}

func (s *Service) SignupLinks(ctx context.Context, _ Admin) ([]PendingLink, error) {
	rows, err := db.New(s.lib).ListLiveSignupLinks(ctx, storage.Now())
	if err != nil {
		return nil, err
	}
	out := make([]PendingLink, len(rows))
	for i, r := range rows {
		out[i] = PendingLink{ID: r.ID, Email: r.Email, IssuedBy: r.IssuerName, IssuedAt: r.IssuedAt.Time(), ExpiresAt: r.ExpiresAt.Time()}
	}
	return out, nil
}

// SignupLink is the live Signup link id and its URL, opened from its
// sealed token, for Copy link.
func (s *Service) SignupLink(ctx context.Context, _ Admin, id LinkID) (PendingLink, string, error) {
	q := db.New(s.lib)
	row, err := q.GetLiveSignupLink(ctx, db.GetLiveSignupLinkParams{ID: id, Now: storage.Now()})
	if errors.Is(err, storage.ErrNoRows) {
		return PendingLink{}, "", ErrNoLink
	}
	if err != nil {
		return PendingLink{}, "", err
	}
	token, err := s.sealer.Open(row.TokenSealed, tokenColumn.Row(id.String()))
	if err != nil {
		return PendingLink{}, "", err
	}
	set, err := readSettings(ctx, q)
	if err != nil {
		return PendingLink{}, "", err
	}
	link := PendingLink{ID: row.ID, Email: row.Email, IssuedBy: row.IssuerName, IssuedAt: row.IssuedAt.Time(), ExpiresAt: row.ExpiresAt.Time()}
	return link, set.Address.String() + signupKind.page + "#token=" + string(token), nil
}

// ResendSignupLink issues a new link for the email of the live link id,
// which it withdraws.
func (s *Service) ResendSignupLink(ctx context.Context, by Admin, id LinkID) (Issued, error) {
	row, err := db.New(s.lib).GetLiveSignupLink(ctx, db.GetLiveSignupLinkParams{ID: id, Now: storage.Now()})
	if errors.Is(err, storage.ErrNoRows) {
		return Issued{}, ErrNoLink
	}
	if err != nil {
		return Issued{}, err
	}
	return s.IssueSignupLink(ctx, by, row.Email)
}

// WithdrawSignupLink cancels the live link id, which then answers
// withdrawn.
func (s *Service) WithdrawSignupLink(ctx context.Context, _ Admin, id LinkID) error {
	return s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		_, err := q.WithdrawSignupLink(ctx, db.WithdrawSignupLinkParams{Now: fx.now, ID: id})
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNoLink
		}
		fx.linksChanged()
		return err
	})
}

// RedeemSignup creates the Member with the link's email, marks the link
// used and signs in, in one Write. A dead token is refused before the PIN
// is hashed, so made-up tokens cost no hash.
func (s *Service) RedeemSignup(ctx context.Context, a Anyone, t LinkToken, displayName string, pin PIN) (SignedIn, error) {
	if _, err := lookup(ctx, db.New(s.lib), signupKind, t, time.Now()); err != nil {
		return SignedIn{}, err
	}
	hash, err := s.pins.hash(ctx, pin)
	if err != nil {
		return SignedIn{}, err
	}
	return s.signInBy(ctx, a.client, func(ctx context.Context, q *db.Queries, fx *effects) (AccountID, error) {
		link, err := redeem(ctx, q, fx, signupKind, t)
		if err != nil {
			return AccountID{}, err
		}
		if _, err := q.GetAccountByEmail(ctx, link.Email); !errors.Is(err, storage.ErrNoRows) {
			if err == nil {
				return AccountID{}, ErrAccountExists
			}
			return AccountID{}, err
		}
		id := AccountID{UUID: storage.NewUUID()}
		err = q.InsertAccount(ctx, db.InsertAccountParams{ID: id, Email: link.Email, DisplayName: displayName, Admin: false, PinHash: hash, CreatedAt: fx.now})
		if err != nil {
			return AccountID{}, err
		}
		fx.accountsChanged()
		return id, q.SetLinkAccount(ctx, db.SetLinkAccountParams{AccountID: id, ID: link.ID})
	})
}
