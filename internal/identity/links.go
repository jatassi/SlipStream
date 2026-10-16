package identity

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jatassi/slipstream/internal/identity/internal/db"
	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
)

type LinkID = db.LinkID

// linkKind is everything that differs between Signup, Reset and email
// confirm links. The schema's CHECK on kind lists the same names.
type linkKind struct {
	name     string
	page     string
	lifetime func(Settings) time.Duration
	// keep is whether the token is kept sealed for Copy link: Signup
	// links only, since reading another Admin's Reset link would take over
	// their Account.
	keep bool
}

var (
	signupKind = &linkKind{name: "signup", page: "/signup", lifetime: func(s Settings) time.Duration { return s.SignupLinkLifetime }, keep: true}
	resetKind  = &linkKind{name: "reset", page: "/reset", lifetime: func(s Settings) time.Duration { return s.ResetLinkLifetime }}
	emailKind  = &linkKind{name: "email", page: "/confirm-email", lifetime: func(s Settings) time.Duration { return s.EmailLinkLifetime }}
)

// LinkKind names a kind of link to Link: SignupLink, ResetLink or EmailLink.
type LinkKind struct{ k *linkKind }

var (
	SignupLink = LinkKind{signupKind}
	ResetLink  = LinkKind{resetKind}
	EmailLink  = LinkKind{emailKind}
)

var tokenColumn = secret.NewColumn("identity_link.token_sealed")

// LinkToken is a link's secret: 32 random bytes, base64url. It is looked
// up by its SHA-256 and logs masked.
type LinkToken struct{ s string }

// ParseLinkToken never fails: a token of the wrong shape matches nothing,
// so its link is unknown, the same answer as a made-up one.
func ParseLinkToken(s string) LinkToken { return LinkToken{s: s} }

func (LinkToken) LogValue() slog.Value { return slog.StringValue("[link token]") }

// DeadReason is why a link doesn't work, a closed enum later specs extend.
// The client's own "unreached" never comes from here.
type DeadReason string

const (
	Unknown   DeadReason = "unknown"
	Withdrawn DeadReason = "withdrawn"
	Expired   DeadReason = "expired"
	Used      DeadReason = "used"
)

// DeadError is a link that can't be used: why, and who to ask. AdminName
// is "" for Unknown, which names nobody, and for a link the API key or the
// host command issued, which the card calls "your Admin".
type DeadError struct {
	Reason    DeadReason
	AdminName string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

func (e *DeadError) Error() string { return "identity: the link is " + string(e.Reason) }

var (
	ErrAddressUnset  = errors.New("identity: SlipStream's address is not set")
	ErrAccountExists = errors.New("identity: an Account has that email")
	ErrEmailTaken    = errors.New("identity: another Account has that email")
	ErrNoLink        = errors.New("identity: no pending Signup link has that id")
)

// deadOf is why row doesn't work, derived from its one row: its end, or
// expired once expires_at has passed. A live row answers nil.
func deadOf(row db.IdentityLink, issuer string, now time.Time) *DeadError {
	dead := &DeadError{AdminName: issuer, IssuedAt: row.IssuedAt.Time(), ExpiresAt: row.ExpiresAt.Time()}
	switch {
	case row.EndReason.String == "used":
		dead.Reason = Used
	case row.EndReason.String == "withdrawn":
		dead.Reason = Withdrawn
	case !now.Before(row.ExpiresAt.Time()):
		dead.Reason = Expired
	default:
		return nil
	}
	return dead
}

func linkOf(r db.GetLinkByHashRow) db.IdentityLink {
	return db.IdentityLink{
		ID: r.ID, Kind: r.Kind, TokenHash: r.TokenHash, TokenSealed: r.TokenSealed, Email: r.Email, AccountID: r.AccountID,
		RemovePasskeys: r.RemovePasskeys, IssuedBy: r.IssuedBy, IssuedAt: r.IssuedAt, ExpiresAt: r.ExpiresAt, EndedAt: r.EndedAt, EndReason: r.EndReason,
	}
}

// LiveLink is what a link's page shows before it is used.
type LiveLink struct {
	AdminName  string
	Email      string
	ServerName string
	ExpiresAt  time.Time
}

// Link answers the by-token GET for kind: the live link, or *DeadError.
// A turned-off Account's link still answers live; using it refuses.
func (s *Service) Link(ctx context.Context, k LinkKind, t LinkToken) (LiveLink, error) {
	q := db.New(s.lib)
	row, err := lookup(ctx, q, k.k, t, time.Now())
	if err != nil {
		return LiveLink{}, err
	}
	set, err := readSettings(ctx, q)
	if err != nil {
		return LiveLink{}, err
	}
	return LiveLink{AdminName: row.IssuerName, Email: row.Email, ServerName: set.Address.Host(), ExpiresAt: row.ExpiresAt.Time()}, nil
}

func lookup(ctx context.Context, q *db.Queries, k *linkKind, t LinkToken, now time.Time) (db.GetLinkByHashRow, error) {
	row, err := q.GetLinkByHash(ctx, digest(t.s))
	if errors.Is(err, storage.ErrNoRows) || err == nil && row.Kind != k.name {
		return db.GetLinkByHashRow{}, &DeadError{Reason: Unknown}
	}
	if err != nil {
		return db.GetLinkByHashRow{}, err
	}
	if dead := deadOf(linkOf(row), row.IssuerName, now); dead != nil {
		return db.GetLinkByHashRow{}, dead
	}
	return row, nil
}

// linkTarget is who a link is for: an email for a Signup link, an Account
// for the others, whose email is where the link goes.
type linkTarget struct {
	email          string
	account        AccountID
	removePasskeys bool
}

// issued is a link just inserted: its URL, shown once, and when it was sent
// and ends. replaces is when the live link it withdrew was sent.
type issued struct {
	id       LinkID
	url      string
	issuedAt time.Time
	expires  time.Time
	replaces time.Time
}

// issue is the one issuing core. It refuses ErrAddressUnset, withdraws the
// live link of k for the same target, so at most one is ever live and the
// withdraw finds one or none, and inserts the new one.
func (s *Service) issue(ctx context.Context, q *db.Queries, fx *effects, k *linkKind, to linkTarget, by AccountID) (issued, error) {
	set, err := readSettings(ctx, q)
	if err != nil {
		return issued{}, err
	}
	if set.Address.IsZero() {
		return issued{}, ErrAddressUnset
	}
	var withdrawn storage.Time
	if k == signupKind {
		withdrawn, err = q.WithdrawLiveSignupLink(ctx, db.WithdrawLiveSignupLinkParams{Now: fx.now, Email: to.email})
	} else {
		withdrawn, err = q.WithdrawLiveAccountLink(ctx, db.WithdrawLiveAccountLinkParams{Now: fx.now, Kind: k.name, AccountID: to.account})
	}
	if err != nil && !errors.Is(err, storage.ErrNoRows) {
		return issued{}, err
	}
	if k == signupKind {
		if err := q.EraseExpiredSignupTokens(ctx, fx.now); err != nil {
			return issued{}, err
		}
	}
	token := newToken()
	out := issued{
		id:       LinkID{UUID: storage.NewUUID()},
		url:      set.Address.String() + k.page + "#token=" + token,
		issuedAt: fx.now.Time(),
		expires:  fx.now.Time().Add(k.lifetime(set)),
	}
	if !withdrawn.IsZero() {
		out.replaces = withdrawn.Time()
	}
	var sealed secret.Sealed
	if k.keep {
		sealed = s.sealer.Seal([]byte(token), tokenColumn.Row(out.id.String()))
	}
	err = q.InsertLink(ctx, db.InsertLinkParams{
		ID:             out.id,
		Kind:           k.name,
		TokenHash:      digest(token),
		TokenSealed:    sealed,
		Email:          to.email,
		AccountID:      to.account,
		RemovePasskeys: to.removePasskeys,
		IssuedBy:       by,
		IssuedAt:       fx.now,
		ExpiresAt:      storage.FromTime(out.expires),
	})
	if err != nil {
		return issued{}, err
	}
	fx.linksChanged()
	return out, nil
}

// redeem marks the live link of k with token used, in the Write that does
// its work. The UPDATE holds only while the link is still live, so two
// tabs racing get one row and one *DeadError{Used}.
func redeem(ctx context.Context, q *db.Queries, fx *effects, k *linkKind, t LinkToken) (db.IdentityLink, error) {
	row, err := q.RedeemLink(ctx, db.RedeemLinkParams{Now: fx.now, TokenHash: digest(t.s), Kind: k.name})
	if errors.Is(err, storage.ErrNoRows) {
		_, err = lookup(ctx, q, k, t, fx.now.Time())
		return db.IdentityLink{}, err
	}
	fx.linksChanged()
	return row, err
}

// deliver renders t and emails it, waiting for the outcome. It runs after
// the Write that issued the link has committed.
func deliver[D any](ctx context.Context, s *Service, t mail.Template[D], to mail.Address, data D) (mail.Outcome, error) {
	m, err := t.Render(to, data)
	if err != nil {
		return mail.Outcome{}, err
	}
	return s.mail.Send(ctx, m)
}

// issuer is who an Admin principal names on a link: its Account, or none
// for the API key.
func issuer(a Admin) (id AccountID, name string) {
	if acc, ok := a.Account(); ok {
		return acc.s.profile.ID, acc.s.profile.DisplayName
	}
	return AccountID{}, ""
}
