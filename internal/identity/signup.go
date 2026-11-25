package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	Offers    Described
}

// Issued answers every issuing operation: the link, its URL, shown this
// once, and what became of its email. Dropped names the servers Send again
// no longer offers.
type Issued struct {
	Link    PendingLink
	URL     string
	Mail    mail.Outcome
	Dropped []string
}

// IssueSignupLink issues a Signup link for email offering g in one Write,
// refusing ErrAccountExists, ErrAddressUnset and the Granter's
// *NotGrantableError, then sends the Signup email and waits for its
// outcome. A live link for the same email is withdrawn, and the email says
// it replaces it.
func (s *Service) IssueSignupLink(ctx context.Context, by Admin, email string, g Grants) (Issued, error) {
	gr, err := s.granterFor(g)
	if err != nil {
		return Issued{}, err
	}
	email = normalizeEmail(email)
	id, name := issuer(by)
	var out issued
	var offers Described
	err = s.write(ctx, func(ctx context.Context, q *db.Queries, fx *effects) error {
		if _, err := q.GetAccountByEmail(ctx, email); !errors.Is(err, storage.ErrNoRows) {
			if err == nil {
				return ErrAccountExists
			}
			return err
		}
		if err := gr.CheckGrants(ctx, fx.tx, g); err != nil {
			return err
		}
		var err error
		if offers, err = gr.Describe(ctx, fx.tx, g); err != nil {
			return err
		}
		out, err = s.issue(ctx, q, fx, signupKind, linkTarget{email: email, grants: g}, id)
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
		Link: PendingLink{ID: out.id, Email: email, IssuedBy: name, IssuedAt: out.issuedAt, ExpiresAt: out.expires, Offers: offers},
		URL:  out.url,
		Mail: outcome,
	}, nil
}

func (s *Service) SignupLinks(ctx context.Context, _ Admin) ([]PendingLink, error) {
	var out []PendingLink
	err := s.lib.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		rows, err := db.New(r).ListLiveSignupLinks(ctx, storage.Now())
		if err != nil {
			return err
		}
		out = make([]PendingLink, len(rows))
		for i, row := range rows {
			offers, err := s.describe(ctx, r, row.Grants)
			if err != nil {
				return err
			}
			out[i] = PendingLink{ID: row.ID, Email: row.Email, IssuedBy: row.IssuerName, IssuedAt: row.IssuedAt.Time(), ExpiresAt: row.ExpiresAt.Time(), Offers: offers}
		}
		return nil
	})
	return out, err
}

// SignupLink is the live Signup link id and its URL, opened from its
// sealed token, for Copy link.
func (s *Service) SignupLink(ctx context.Context, _ Admin, id LinkID) (PendingLink, string, error) {
	var link PendingLink
	var url string
	err := s.lib.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, offers, err := s.liveSignupLink(ctx, r, id)
		if err != nil {
			return err
		}
		token, err := s.sealer.Open(row.TokenSealed, tokenColumn.Row(id.String()))
		if err != nil {
			return err
		}
		set, err := readSettings(ctx, db.New(r))
		if err != nil {
			return err
		}
		link = PendingLink{ID: row.ID, Email: row.Email, IssuedBy: row.IssuerName, IssuedAt: row.IssuedAt.Time(), ExpiresAt: row.ExpiresAt.Time(), Offers: offers}
		url = set.Address.String() + signupKind.page + "#token=" + string(token)
		return nil
	})
	return link, url, err
}

// ResendSignupLink issues a new link for the email of the live link id,
// which it withdraws. The new link offers the servers the old one did that
// are still present and shown to Members, and Dropped names the others.
func (s *Service) ResendSignupLink(ctx context.Context, by Admin, id LinkID) (Issued, error) {
	var email string
	var kept Grants
	var dropped []string
	err := s.lib.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		row, offers, err := s.liveSignupLink(ctx, r, id)
		if err != nil {
			return err
		}
		email = row.Email
		for _, o := range offers.MediaServers {
			if o.Present && o.Shown {
				kept.MediaServers = append(kept.MediaServers, ServerGrant{Server: o.Server, PlexHome: o.PlexHome})
			} else {
				dropped = append(dropped, o.Name)
			}
		}
		return nil
	})
	if err != nil {
		return Issued{}, err
	}
	out, err := s.IssueSignupLink(ctx, by, email, kept)
	if err != nil {
		return Issued{}, err
	}
	out.Dropped = dropped
	return out, nil
}

func (s *Service) liveSignupLink(ctx context.Context, r storage.Reader[file], id LinkID) (db.GetLiveSignupLinkRow, Described, error) {
	row, err := db.New(r).GetLiveSignupLink(ctx, db.GetLiveSignupLinkParams{ID: id, Now: storage.Now()})
	if errors.Is(err, storage.ErrNoRows) {
		return row, Described{}, ErrNoLink
	}
	if err != nil {
		return row, Described{}, err
	}
	offers, err := s.describe(ctx, r, row.Grants)
	return row, offers, err
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
		g, err := parseGrants(link.Grants)
		if err != nil {
			return AccountID{}, err
		}
		gr, err := s.granterFor(g)
		if err != nil {
			return AccountID{}, err
		}
		after, err := gr.Redeemed(ctx, fx.tx, Redemption{Account: id, IssuedBy: link.IssuedBy, Grants: g})
		if err != nil {
			return AccountID{}, err
		}
		fx.afterCommit(after)
		fx.accountsChanged()
		return id, q.SetLinkAccount(ctx, db.SetLinkAccountParams{AccountID: id, ID: link.ID})
	})
}

// granterFor is what owns g: NoGrants for empty g, which needs no owner,
// and ErrNoGranter while g has grants and nothing is set.
func (s *Service) granterFor(g Grants) (Granter, error) {
	if g.Empty() {
		return NoGrants{}, nil
	}
	return s.boundGranter()
}

func (s *Service) boundGranter() (Granter, error) {
	g := s.granter.Load()
	if g == nil {
		return nil, ErrNoGranter
	}
	return *g, nil
}

// parseGrants is the one decode of identity_link.grants.
func parseGrants(column *string) (Grants, error) {
	var g Grants
	if column == nil {
		return g, nil
	}
	if err := json.Unmarshal([]byte(*column), &g); err != nil {
		return Grants{}, fmt.Errorf("decode the link's grants: %w", err)
	}
	return g, nil
}

// describe is a link's grants column as its owner describes it.
func (s *Service) describe(ctx context.Context, r storage.Reader[file], column *string) (Described, error) {
	g, err := parseGrants(column)
	if err != nil {
		return Described{}, err
	}
	gr, err := s.granterFor(g)
	if err != nil {
		return Described{}, err
	}
	return gr.Describe(ctx, r, g)
}
