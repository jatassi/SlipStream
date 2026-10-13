// Package identityapi holds identity's operations: claiming a fresh Node
// and signing in and out.
package identityapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

// Library is the Node's library.db as identity's operations reach it.
type Library interface {
	// Identity is the identity service, false while library.db is closed.
	Identity() (*identity.Service, bool)
	// ClaimController opens library.db and library.key, creating them,
	// runs claim on the identity service, and once claim succeeds makes the
	// Node a Controller.
	ClaimController(ctx context.Context, claim func(*identity.Service) error) error
}

// open is the identity service of an operation an Account check admitted,
// which only a Controller, with library.db open, admits.
func open(lib Library) *identity.Service {
	s, ok := lib.Identity()
	if !ok {
		panic("identityapi: an Account operation ran while library.db is closed")
	}
	return s
}

var pinWrong = api.NewCode(http.StatusUnauthorized, "pin_wrong", "The email or PIN is wrong")

type account struct {
	ID          string `json:"id" format:"uuid"`
	Email       string `json:"email" format:"email" maxLength:"254"`
	DisplayName string `json:"displayName" minLength:"1" maxLength:"100"`
	Admin       bool   `json:"admin"`
}

func accountOf(p identity.Profile) account {
	return account{ID: p.ID.String(), Email: p.Email, DisplayName: p.DisplayName, Admin: p.Admin}
}

type claimInput struct {
	Body claimBody
}

type claimBody struct {
	Email       string `json:"email" format:"email" maxLength:"254"`
	DisplayName string `json:"displayName" minLength:"1" maxLength:"100"`
	PIN         string `json:"pin" pattern:"^[0-9]{4}$"`
}

type claimOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      claimed
}

type claimed struct {
	Account account `json:"account"`
	APIKey  string  `json:"apiKey" pattern:"^ssk_[A-Za-z0-9_-]{43}$"`
}

type signInInput struct {
	Body signInBody
}

type signInBody struct {
	Email string `json:"email" format:"email" maxLength:"254"`
	PIN   string `json:"pin" pattern:"^[0-9]{4}$"`
}

type sessionOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      session
}

type session struct {
	Account account `json:"account"`
}

type getSessionOutput struct {
	Body session
}

type signOutOutput struct {
	SetCookie string `header:"Set-Cookie"`
}

func Routes(r *api.Router, lib Library) {
	o := r.Owner("identity", api.Public)

	api.Register(o, api.Setup, api.Op{
		ID:      "claimController",
		Method:  http.MethodPost,
		Path:    "/setup/controller",
		Summary: "Create the first Admin and make this Node a Controller",
		Status:  http.StatusCreated,
	}, func(ctx context.Context, c identity.Claim, in *claimInput) (*claimOutput, error) {
		pin, err := identity.ParsePIN(in.Body.PIN)
		if err != nil {
			return nil, err
		}
		var res identity.Claimed
		err = lib.ClaimController(ctx, func(s *identity.Service) error {
			var err error
			res, err = s.ClaimController(ctx, c, in.Body.Email, in.Body.DisplayName, pin)
			return err
		})
		if errors.Is(err, identity.ErrAlreadyClaimed) {
			return nil, api.AlreadyClaimed
		}
		if err != nil {
			return nil, err
		}
		return &claimOutput{
			SetCookie: res.Cookie.String(),
			Body:      claimed{Account: accountOf(res.Profile), APIKey: res.APIKey},
		}, nil
	})

	api.Register(o, api.Public, api.Op{
		ID:      "createSession",
		Method:  http.MethodPost,
		Path:    "/session",
		Summary: "Sign in with email and PIN",
		Status:  http.StatusOK,
		Errors:  []*api.Code{pinWrong},
	}, func(ctx context.Context, a identity.Anyone, in *signInInput) (*sessionOutput, error) {
		pin, err := identity.ParsePIN(in.Body.PIN)
		if err != nil {
			return nil, err
		}
		s, ok := lib.Identity()
		if !ok {
			return nil, pinWrong
		}
		profile, cookie, err := s.SignIn(ctx, a, in.Body.Email, pin)
		if errors.Is(err, identity.ErrPINWrong) {
			return nil, pinWrong
		}
		if err != nil {
			return nil, err
		}
		return &sessionOutput{SetCookie: cookie.String(), Body: session{Account: accountOf(profile)}}, nil
	})

	api.Register(o, api.Account, api.Op{
		ID:      "getSession",
		Method:  http.MethodGet,
		Path:    "/session",
		Summary: "Get the signed-in Account",
		Status:  http.StatusOK,
	}, func(_ context.Context, a identity.Account, _ *struct{}) (*getSessionOutput, error) {
		return &getSessionOutput{Body: session{Account: accountOf(a.Profile())}}, nil
	})

	api.Register(o, api.Account, api.Op{
		ID:      "deleteSession",
		Method:  http.MethodDelete,
		Path:    "/session",
		Summary: "Sign out",
		Status:  http.StatusNoContent,
	}, func(ctx context.Context, a identity.Account, _ *struct{}) (*signOutOutput, error) {
		cookie, err := open(lib).SignOut(ctx, a)
		if err != nil {
			return nil, err
		}
		return &signOutOutput{SetCookie: cookie.String()}, nil
	})
}
