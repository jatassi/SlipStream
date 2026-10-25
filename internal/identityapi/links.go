package identityapi

import (
	"context"
	"errors"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mailapi"
	"github.com/jatassi/slipstream/internal/storage"
)

type pendingLink struct {
	ID        string    `json:"id" format:"uuid"`
	Email     string    `json:"email" format:"email"`
	IssuedBy  string    `json:"issuedBy,omitempty" doc:"The Admin who issued it; absent for the API key"`
	IssuedAt  time.Time `json:"issuedAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func pendingOf(l identity.PendingLink) pendingLink {
	return pendingLink{ID: l.ID.String(), Email: l.Email, IssuedBy: l.IssuedBy, IssuedAt: l.IssuedAt, ExpiresAt: l.ExpiresAt}
}

type pendingLinks struct {
	Links []pendingLink `json:"links" nullable:"false"`
}

type issuedSignupLink struct {
	Link  pendingLink     `json:"link"`
	URL   string          `json:"url" format:"uri" doc:"The link itself, to copy"`
	Email mailapi.Outcome `json:"email"`
}

type issuedResetLink struct {
	URL   string          `json:"url" format:"uri" doc:"The link itself, shown only in this response"`
	Email mailapi.Outcome `json:"email"`
}

type signupLinkURL struct {
	Link pendingLink `json:"link"`
	URL  string      `json:"url" format:"uri"`
}

type liveLink struct {
	State      string    `json:"state" enum:"live"`
	AdminName  string    `json:"adminName,omitempty" doc:"Who issued the link; absent when the API key or the host command did"`
	Email      string    `json:"email" format:"email" doc:"Where the link went: the Account's email, or the new one it confirms"`
	ServerName string    `json:"serverName"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type deadLink struct {
	State     string     `json:"state" enum:"dead"`
	Reason    string     `json:"reason" enum:"unknown,withdrawn,expired,used"`
	AdminName string     `json:"adminName,omitempty" doc:"Who to ask: absent for unknown, which names nobody, and for a link the API key or the host command issued"`
	IssuedAt  *time.Time `json:"issuedAt,omitempty" doc:"Every reason but unknown"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty" doc:"Every reason but unknown"`
}

type linkView struct{}

func (linkView) Variants() (discriminator string, variants []any) {
	return "state", []any{liveLink{}, deadLink{}}
}

type linkIDInput struct {
	ID string `path:"id" format:"uuid"`
}

type tokenInput struct {
	Token string `path:"token" maxLength:"128"`
}

type newSignupLinkInput struct {
	Body struct {
		Email string `json:"email" format:"email" maxLength:"254"`
	}
}

type resetLinkInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		RemovePasskeys bool `json:"removePasskeys,omitempty" doc:"Also remove the Account's passkeys, for when someone else may have had access"`
	}
}

type joinInput struct {
	Token string `path:"token" maxLength:"128"`
	Body  struct {
		DisplayName string `json:"displayName" minLength:"1" maxLength:"100"`
		PIN         string `json:"pin" pattern:"^[0-9]{4}$"`
	}
}

type newPINInput struct {
	Token string `path:"token" maxLength:"128"`
	Body  struct {
		PIN string `json:"pin" pattern:"^[0-9]{4}$"`
	}
}

type emailChangeInput struct {
	Body struct {
		Email string `json:"email" format:"email" maxLength:"254"`
	}
}

type emailChangeStarted struct {
	Email mailapi.Outcome `json:"email" doc:"What became of the confirm link's email to the new address"`
}

type (
	pendingLinksOutput  struct{ Body pendingLinks }
	signupIssuedOutput  struct{ Body issuedSignupLink }
	resetIssuedOutput   struct{ Body issuedResetLink }
	signupLinkURLOutput struct{ Body signupLinkURL }
	linkViewOutput      struct{ Body api.OneOf[linkView] }
	emailChangeOutput   struct{ Body emailChangeStarted }
)

func linkID(s string) (identity.LinkID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return identity.LinkID{}, linkNotFound
	}
	return identity.LinkID{UUID: u}, nil
}

func accountID(s string) (identity.AccountID, error) {
	u, err := storage.ParseUUID(s)
	if err != nil {
		return identity.AccountID{}, accountNotFound
	}
	return identity.AccountID{UUID: u}, nil
}

func signupIssued(i identity.Issued, err error) (*signupIssuedOutput, error) {
	if err != nil {
		return nil, answer(err)
	}
	return &signupIssuedOutput{Body: issuedSignupLink{Link: pendingOf(i.Link), URL: i.URL, Email: mailapi.OutcomeOf(i.Mail)}}, nil
}

func issueSignupLink(lib Library) func(context.Context, identity.Admin, *newSignupLinkInput) (*signupIssuedOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *newSignupLinkInput) (*signupIssuedOutput, error) {
		return signupIssued(open(lib).IssueSignupLink(ctx, a, in.Body.Email))
	}
}

func listSignupLinks(lib Library) func(context.Context, identity.Admin, *struct{}) (*pendingLinksOutput, error) {
	return func(ctx context.Context, a identity.Admin, _ *struct{}) (*pendingLinksOutput, error) {
		links, err := open(lib).SignupLinks(ctx, a)
		if err != nil {
			return nil, err
		}
		out := pendingLinks{Links: make([]pendingLink, len(links))}
		for i, l := range links {
			out.Links[i] = pendingOf(l)
		}
		return &pendingLinksOutput{Body: out}, nil
	}
}

func getSignupLink(lib Library) func(context.Context, identity.Admin, *linkIDInput) (*signupLinkURLOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *linkIDInput) (*signupLinkURLOutput, error) {
		id, err := linkID(in.ID)
		if err != nil {
			return nil, err
		}
		s := open(lib)
		link, url, err := s.SignupLink(ctx, a, id)
		if err != nil {
			return nil, answer(err)
		}
		return &signupLinkURLOutput{Body: signupLinkURL{Link: pendingOf(link), URL: url}}, nil
	}
}

func resendSignupLink(lib Library) func(context.Context, identity.Admin, *linkIDInput) (*signupIssuedOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *linkIDInput) (*signupIssuedOutput, error) {
		id, err := linkID(in.ID)
		if err != nil {
			return nil, err
		}
		return signupIssued(open(lib).ResendSignupLink(ctx, a, id))
	}
}

func withdrawSignupLink(lib Library) func(context.Context, identity.Admin, *linkIDInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Admin, in *linkIDInput) (*struct{}, error) {
		id, err := linkID(in.ID)
		if err != nil {
			return nil, err
		}
		return nil, answer(open(lib).WithdrawSignupLink(ctx, a, id))
	}
}

func issueResetLink(lib Library) func(context.Context, identity.Admin, *resetLinkInput) (*resetIssuedOutput, error) {
	return func(ctx context.Context, a identity.Admin, in *resetLinkInput) (*resetIssuedOutput, error) {
		id, err := accountID(in.ID)
		if err != nil {
			return nil, err
		}
		i, err := open(lib).IssueResetLink(ctx, a, id, in.Body.RemovePasskeys)
		if err != nil {
			return nil, answer(err)
		}
		return &resetIssuedOutput{Body: issuedResetLink{URL: i.URL, Email: mailapi.OutcomeOf(i.Mail)}}, nil
	}
}

func startEmailChange(lib Library) func(context.Context, identity.Account, *emailChangeInput) (*emailChangeOutput, error) {
	return func(ctx context.Context, a identity.Account, in *emailChangeInput) (*emailChangeOutput, error) {
		out, err := open(lib).StartEmailChange(ctx, a, in.Body.Email)
		if err != nil {
			return nil, answer(err)
		}
		return &emailChangeOutput{Body: emailChangeStarted{Email: mailapi.OutcomeOf(out)}}, nil
	}
}

// viewLink is a by-token GET. An Unclaimed Node holds no link, so every
// token there is unknown.
func viewLink(lib Library, k identity.LinkKind) func(context.Context, identity.Anyone, *tokenInput) (*linkViewOutput, error) {
	return func(ctx context.Context, _ identity.Anyone, in *tokenInput) (*linkViewOutput, error) {
		s, ok := lib.Identity()
		if !ok {
			return &linkViewOutput{Body: api.NewOneOf[linkView](deadLink{State: "dead", Reason: string(identity.Unknown)})}, nil
		}
		live, err := s.Link(ctx, k, identity.ParseLinkToken(in.Token))
		if dead, ok := errors.AsType[*identity.DeadError](err); ok {
			v := deadLink{State: "dead", Reason: string(dead.Reason), AdminName: dead.AdminName}
			if dead.Reason != identity.Unknown {
				v.IssuedAt, v.ExpiresAt = &dead.IssuedAt, &dead.ExpiresAt
			}
			return &linkViewOutput{Body: api.NewOneOf[linkView](v)}, nil
		}
		if err != nil {
			return nil, err
		}
		return &linkViewOutput{Body: api.NewOneOf[linkView](liveLink{
			State: "live", AdminName: live.AdminName, Email: live.Email, ServerName: live.ServerName, ExpiresAt: live.ExpiresAt,
		})}, nil
	}
}

func opened(lib Library) (*identity.Service, error) {
	s, ok := lib.Identity()
	if !ok {
		return nil, linkDead
	}
	return s, nil
}

func redeemSignupLink(lib Library) func(context.Context, identity.Anyone, *joinInput) (*signedInOutput, error) {
	return func(ctx context.Context, a identity.Anyone, in *joinInput) (*signedInOutput, error) {
		pin, err := identity.ParsePIN(in.Body.PIN)
		if err != nil {
			return nil, err
		}
		s, err := opened(lib)
		if err != nil {
			return nil, err
		}
		got, err := s.RedeemSignup(ctx, a, identity.ParseLinkToken(in.Token), in.Body.DisplayName, pin)
		if err != nil {
			return nil, answer(err)
		}
		return signedInOf(got), nil
	}
}

func redeemResetLink(lib Library) func(context.Context, identity.Anyone, *newPINInput) (*signedInOutput, error) {
	return func(ctx context.Context, a identity.Anyone, in *newPINInput) (*signedInOutput, error) {
		pin, err := identity.ParsePIN(in.Body.PIN)
		if err != nil {
			return nil, err
		}
		s, err := opened(lib)
		if err != nil {
			return nil, err
		}
		got, err := s.RedeemReset(ctx, a, identity.ParseLinkToken(in.Token), pin)
		if err != nil {
			return nil, answer(err)
		}
		return signedInOf(got), nil
	}
}

func confirmEmail(lib Library) func(context.Context, identity.Anyone, *tokenInput) (*struct{}, error) {
	return func(ctx context.Context, a identity.Anyone, in *tokenInput) (*struct{}, error) {
		s, err := opened(lib)
		if err != nil {
			return nil, err
		}
		return nil, answer(s.ConfirmEmail(ctx, a, identity.ParseLinkToken(in.Token)))
	}
}
