// Package identityapi holds identity's operations: claiming a fresh Node,
// signing in and out, re-confirm and the PIN, the API key, and the sign-in
// settings.
package identityapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/settingapi"
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

type session struct {
	Account     account `json:"account"`
	PINRequired bool    `json:"pinRequired" doc:"The session must set a new PIN before anything but reading the session and signing out"`
	KeyPrompt   string  `json:"keyPrompt" enum:"ask,never" doc:"Whether the prompt to add a passkey may show after sign-in"`
}

type getSessionOutput struct {
	Body session
}

type signedInOutput struct {
	SetCookie string `header:"Set-Cookie"`
	Body      signedIn
}

type signedIn struct {
	Account       account `json:"account"`
	PINRequired   bool    `json:"pinRequired" doc:"The session must set a new PIN before anything but reading the session and signing out"`
	KeyPrompt     string  `json:"keyPrompt" enum:"ask,never" doc:"Whether the prompt to add a passkey may show after sign-in"`
	PasskeyPrompt bool    `json:"passkeyPrompt" doc:"Show the prompt to add a passkey for this address, once in this session"`
}

func signedInOf(s identity.SignedIn) *signedInOutput {
	return &signedInOutput{
		SetCookie: s.Cookie.String(),
		Body:      signedIn{Account: accountOf(s.Profile), PINRequired: s.PINRequired, KeyPrompt: string(s.KeyPrompt), PasskeyPrompt: s.PasskeyPrompt},
	}
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
		Errors:  []*api.Code{api.JoinPending},
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
		if errors.Is(err, identity.ErrJoinPending) {
			return nil, api.JoinPending
		}
		if err != nil {
			return nil, err
		}
		return &claimOutput{
			SetCookie: res.Cookie.String(),
			Body:      claimed{Account: accountOf(res.Profile), APIKey: res.APIKey.Reveal()},
		}, nil
	})

	api.Register(o, api.Public, api.Op{
		ID:      "createSession",
		Method:  http.MethodPost,
		Path:    "/session",
		Summary: "Sign in with email and PIN",
		Status:  http.StatusOK,
		Errors:  append([]*api.Code{accountDisabled}, pinChecked...),
	}, func(ctx context.Context, a identity.Anyone, in *signInInput) (*signedInOutput, error) {
		pin, err := identity.ParsePIN(in.Body.PIN)
		if err != nil {
			return nil, err
		}
		s, ok := lib.Identity()
		if !ok {
			return nil, pinWrong
		}
		got, err := s.SignIn(ctx, a, in.Body.Email, pin)
		if err != nil {
			return nil, answer(err)
		}
		return signedInOf(got), nil
	})

	api.Register(o, api.Public, api.Op{
		ID:      "startPasskeySignIn",
		Method:  http.MethodPost,
		Path:    "/passkeys/assertion",
		Summary: "Start signing in with a passkey for this address",
		Status:  http.StatusOK,
		Errors:  []*api.Code{passkeyUnavailable, tooManyAttempts},
	}, startPasskeySignIn(lib))

	api.Register(o, api.Public, api.Op{
		ID:      "finishPasskeySignIn",
		Method:  http.MethodPost,
		Path:    "/passkeys/assertion/{ceremony}",
		Summary: "Sign in with the browser's passkey response. On a turned-off PIN, the session must set a new PIN first",
		Status:  http.StatusOK,
		Errors:  []*api.Code{passkeyUnavailable, passkeyFailed, tooManyAttempts, accountDisabled},
	}, finishPasskeySignIn(lib))

	api.Register(o, api.Bearer, api.Op{
		ID:      "getSession",
		Method:  http.MethodGet,
		Path:    "/session",
		Summary: "Get the signed-in Account",
		Status:  http.StatusOK,
	}, func(_ context.Context, b identity.Bearer, _ *struct{}) (*getSessionOutput, error) {
		return &getSessionOutput{Body: session{Account: accountOf(b.Profile()), PINRequired: b.PINRequired(), KeyPrompt: string(b.KeyPrompt())}}, nil
	})

	api.Register(o, api.Bearer, api.Op{
		ID:      "deleteSession",
		Method:  http.MethodDelete,
		Path:    "/session",
		Summary: "Sign out",
		Status:  http.StatusNoContent,
	}, func(ctx context.Context, b identity.Bearer, _ *struct{}) (*signOutOutput, error) {
		cookie, err := open(lib).SignOut(ctx, b)
		if err != nil {
			return nil, err
		}
		return &signOutOutput{SetCookie: cookie.String()}, nil
	})

	api.Register(o, api.Account, api.Op{
		ID:      "reconfirmSession",
		Method:  http.MethodPost,
		Path:    "/session/reconfirm",
		Summary: "Confirm it's you with your PIN, for actions that change who can get in",
		Status:  http.StatusNoContent,
		Errors:  pinChecked,
	}, reconfirm(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "startPasskeyReconfirm",
		Method:  http.MethodPost,
		Path:    "/session/reconfirm/passkey",
		Summary: "Start confirming it's you with one of your passkeys for this address",
		Status:  http.StatusOK,
		Errors:  []*api.Code{passkeyUnavailable, tooManyAttempts},
	}, startPasskeyReconfirm(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "finishPasskeyReconfirm",
		Method:  http.MethodPost,
		Path:    "/session/reconfirm/passkey/{ceremony}",
		Summary: "Confirm it's you with the browser's passkey response",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{passkeyUnavailable, passkeyFailed, tooManyAttempts},
	}, finishPasskeyReconfirm(lib))

	api.Register(o, api.Bearer, api.Op{
		ID:      "changePIN",
		Method:  http.MethodPost,
		Path:    "/account/pin",
		Summary: "Change your PIN, or set a new one after a passkey sign-in on a turned-off PIN",
		Status:  http.StatusNoContent,
		Errors:  pinChecked,
	}, changePIN(lib))

	api.Register(o, api.Admin, api.Op{
		ID:      "getAPIKey",
		Method:  http.MethodGet,
		Path:    "/api-key",
		Summary: "Get when the API key was made",
		Status:  http.StatusOK,
	}, getAPIKey(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "regenerateAPIKey",
		Method:  http.MethodPost,
		Path:    "/api-key",
		Summary: "Replace the API key, shown this once. The old key stops working",
		Status:  http.StatusOK,
	}, regenerateAPIKey(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "updateOwnAccount",
		Method:  http.MethodPatch,
		Path:    "/account",
		Summary: "Change your display name, or whether the passkey prompt may show",
		Status:  http.StatusOK,
	}, updateOwnAccount(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "listSessions",
		Method:  http.MethodGet,
		Path:    "/sessions",
		Summary: "List the devices your Account is signed in on",
		Status:  http.StatusOK,
	}, listSessions(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "endSession",
		Method:  http.MethodDelete,
		Path:    "/sessions/{id}",
		Summary: "Sign one of your devices out",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{sessionNotFound},
	}, endSession(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "endOtherSessions",
		Method:  http.MethodDelete,
		Path:    "/sessions",
		Summary: "Sign out every device but this one",
		Status:  http.StatusNoContent,
	}, endOtherSessions(lib))

	api.Register(o, api.Confirmed, api.Op{
		ID:      "startPasskeyRegistration",
		Method:  http.MethodPost,
		Path:    "/passkeys/registration",
		Summary: "Start adding a passkey for this address",
		Status:  http.StatusOK,
		Errors:  []*api.Code{passkeyUnavailable},
	}, startPasskeyRegistration(lib))

	api.Register(o, api.Confirmed, api.Op{
		ID:      "finishPasskeyRegistration",
		Method:  http.MethodPost,
		Path:    "/passkeys/registration/{ceremony}",
		Summary: "Finish adding a passkey with the browser's response",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{passkeyUnavailable, passkeyFailed},
	}, finishPasskeyRegistration(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "listPasskeys",
		Method:  http.MethodGet,
		Path:    "/passkeys",
		Summary: "List your passkeys at every address",
		Status:  http.StatusOK,
	}, listPasskeys(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "updatePasskey",
		Method:  http.MethodPatch,
		Path:    "/passkeys/{id}",
		Summary: "Rename one of your passkeys",
		Status:  http.StatusOK,
		Errors:  []*api.Code{passkeyNotFound},
	}, renamePasskey(lib))

	api.Register(o, api.Account, api.Op{
		ID:      "deletePasskey",
		Method:  http.MethodDelete,
		Path:    "/passkeys/{id}",
		Summary: "Delete one of your passkeys",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{passkeyNotFound},
	}, deletePasskey(lib))

	api.Register(o, api.Admin, api.Op{
		ID:      "listAccounts",
		Method:  http.MethodGet,
		Path:    "/accounts",
		Summary: "List every Account",
		Status:  http.StatusOK,
	}, listAccounts(lib))

	api.Register(o, api.Admin, api.Op{
		ID:      "getAccount",
		Method:  http.MethodGet,
		Path:    "/accounts/{id}",
		Summary: "Get an Account with its passkeys and sessions",
		Status:  http.StatusOK,
		Errors:  []*api.Code{accountNotFound},
	}, getAccount(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "updateAccount",
		Method:  http.MethodPatch,
		Path:    "/accounts/{id}",
		Summary: "Change an Account's name, email or role, or turn it off or on",
		Status:  http.StatusOK,
		Errors:  []*api.Code{accountNotFound, emailTaken, lastAdmin},
	}, updateAccount(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "deleteAccount",
		Method:  http.MethodDelete,
		Path:    "/accounts/{id}",
		Summary: "Delete an Account with its sessions, passkeys and links",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{accountNotFound, lastAdmin},
	}, deleteAccount(lib))

	api.Register(o, api.Admin, api.Op{
		ID:      "listAccountSessions",
		Method:  http.MethodGet,
		Path:    "/accounts/{id}/sessions",
		Summary: "List the devices an Account is signed in on",
		Status:  http.StatusOK,
		Errors:  []*api.Code{accountNotFound},
	}, listAccountSessions(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "endAccountSessions",
		Method:  http.MethodDelete,
		Path:    "/accounts/{id}/sessions",
		Summary: "Sign an Account out everywhere",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{accountNotFound},
	}, endAccountSessions(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "endAccountSession",
		Method:  http.MethodDelete,
		Path:    "/accounts/{id}/sessions/{session}",
		Summary: "Sign one of an Account's devices out",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{accountNotFound, sessionNotFound},
	}, endAccountSession(lib))

	api.Register(o, api.Confirmed, api.Op{
		ID:      "startEmailChange",
		Method:  http.MethodPost,
		Path:    "/account/email",
		Summary: "Change your email by sending a confirm link to the new address. The email changes when the link is followed",
		Status:  http.StatusOK,
		Errors:  []*api.Code{emailTaken, addressUnset},
	}, startEmailChange(lib))

	api.Register(o, api.Admin, api.Op{
		ID:      "listSignupLinks",
		Method:  http.MethodGet,
		Path:    "/signup-links",
		Summary: "List the Signup links nobody has used yet",
		Status:  http.StatusOK,
	}, listSignupLinks(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "createSignupLink",
		Method:  http.MethodPost,
		Path:    "/signup-links",
		Summary: "Issue a Signup link for an email, email it, and return it to copy",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{addressUnset, accountExists},
	}, issueSignupLink(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "getSignupLink",
		Method:  http.MethodGet,
		Path:    "/signup-links/{id}",
		Summary: "Get a pending Signup link with its URL, to copy again",
		Status:  http.StatusOK,
		Errors:  []*api.Code{linkNotFound},
	}, getSignupLink(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "resendSignupLink",
		Method:  http.MethodPost,
		Path:    "/signup-links/{id}/resend",
		Summary: "Send a pending Signup link again as a new link. The old link stops working",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{linkNotFound, addressUnset, accountExists},
	}, resendSignupLink(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "deleteSignupLink",
		Method:  http.MethodDelete,
		Path:    "/signup-links/{id}",
		Summary: "Cancel a pending Signup link",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{linkNotFound},
	}, withdrawSignupLink(lib))

	api.Register(o, api.Public, api.Op{
		ID:      "getSignupLinkByToken",
		Method:  http.MethodGet,
		Path:    "/signup-links/by-token/{token}",
		Summary: "Get what a Signup link's page shows: who it is from, or why it no longer works",
		Status:  http.StatusOK,
	}, viewLink(lib, identity.SignupLink))

	api.Register(o, api.Public, api.Op{
		ID:      "redeemSignupLink",
		Method:  http.MethodPost,
		Path:    "/signup-links/by-token/{token}/account",
		Summary: "Make the Member's Account with a Signup link and sign in",
		Status:  http.StatusOK,
		Errors:  []*api.Code{linkDead, accountExists},
	}, redeemSignupLink(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "createResetLink",
		Method:  http.MethodPost,
		Path:    "/accounts/{id}/reset-link",
		Summary: "Issue a Reset link for an Account and end its sessions. Only this response shows the link",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{accountNotFound, accountDisabled, addressUnset},
	}, issueResetLink(lib))

	api.Register(o, api.Public, api.Op{
		ID:      "getResetLinkByToken",
		Method:  http.MethodGet,
		Path:    "/reset-links/by-token/{token}",
		Summary: "Get what a Reset link's page shows: who it is from, or why it no longer works",
		Status:  http.StatusOK,
	}, viewLink(lib, identity.ResetLink))

	api.Register(o, api.Public, api.Op{
		ID:      "redeemResetLink",
		Method:  http.MethodPost,
		Path:    "/reset-links/by-token/{token}/pin",
		Summary: "Set a new PIN with a Reset link, ending every session, and sign in",
		Status:  http.StatusOK,
		Errors:  []*api.Code{linkDead, accountDisabled},
	}, redeemResetLink(lib))

	api.Register(o, api.Public, api.Op{
		ID:      "getEmailLinkByToken",
		Method:  http.MethodGet,
		Path:    "/email-links/by-token/{token}",
		Summary: "Get what a confirm link's page shows: the new email, or why it no longer works",
		Status:  http.StatusOK,
	}, viewLink(lib, identity.EmailLink))

	api.Register(o, api.Public, api.Op{
		ID:      "confirmEmail",
		Method:  http.MethodPost,
		Path:    "/email-links/by-token/{token}/confirm",
		Summary: "Change an Account's email to the one a confirm link proved",
		Status:  http.StatusNoContent,
		Errors:  []*api.Code{linkDead, accountDisabled, emailTaken},
	}, confirmEmail(lib))

	api.Register(o, api.Admin, api.Op{
		ID:      "getIdentitySettings",
		Method:  http.MethodGet,
		Path:    "/identity/settings",
		Summary: "Get the sign-in limits and lifetimes",
		Status:  http.StatusOK,
	}, getSettings(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "updateIdentitySettings",
		Method:  http.MethodPatch,
		Path:    "/identity/settings",
		Summary: "Change the sign-in limits and lifetimes",
		Status:  http.StatusOK,
		Errors:  []*api.Code{settingapi.ErrOutOfRange},
	}, updateSettings(lib))

	api.Register(o, api.Admin, api.Op{
		ID:      "getAddress",
		Method:  http.MethodGet,
		Path:    "/identity/address",
		Summary: "Get the address links use, and the one Settings > Address suggests",
		Status:  http.StatusOK,
	}, getAddress(lib))

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "setAddress",
		Method:  http.MethodPut,
		Path:    "/identity/address",
		Summary: "Set the address links use",
		Status:  http.StatusOK,
		Errors:  []*api.Code{addressIP},
	}, setAddress(lib))
}
