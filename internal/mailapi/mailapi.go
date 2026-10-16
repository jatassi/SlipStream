// Package mailapi holds the SMTP setting's operations and Send test email.
package mailapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/mail"
)

type Library interface {
	// Mail is the SMTP setting, false while library.db is closed.
	Mail() (*mail.Mail, bool)
	Identity() (*identity.Service, bool)
}

func open(lib Library) (*mail.Mail, *identity.Service) {
	m, ok := lib.Mail()
	ids, open := lib.Identity()
	if !ok || !open {
		panic("mailapi: an Admin operation ran while library.db is closed")
	}
	return m, ids
}

var noRecipient = api.NewCode(http.StatusUnprocessableEntity, "no_recipient", "The API key has no email address to send a test to. Sign in to send one.")

// Outcome is what became of an email SlipStream sent while the caller
// waited.
type Outcome struct {
	Outcome string `json:"outcome" enum:"sent,failed,not-set-up"`
	Detail  string `json:"detail,omitempty" doc:"The mail server's reply, or why none answered, when the email failed"`
}

func OutcomeOf(o mail.Outcome) Outcome { return Outcome{Outcome: string(o.Kind), Detail: o.Reply} }

type settings struct {
	Host        string `json:"host" doc:"Empty until email is set up"`
	Port        uint16 `json:"port" minimum:"1"`
	Security    string `json:"security" enum:"none,starttls,tls"`
	Username    string `json:"username" doc:"Empty when the server takes no sign-in"`
	PasswordSet bool   `json:"passwordSet" doc:"Whether a password is saved; it is never shown"`
	FromAddress string `json:"fromAddress"`
	FromName    string `json:"fromName"`
}

type settingsPatch struct {
	Host        *string `json:"host,omitempty" maxLength:"253"`
	Port        *uint16 `json:"port,omitempty" minimum:"1"`
	Security    *string `json:"security,omitempty" enum:"none,starttls,tls"`
	Username    *string `json:"username,omitempty" maxLength:"254"`
	Password    *string `json:"password,omitempty" maxLength:"1024" doc:"Write-only; empty clears it"`
	FromAddress *string `json:"fromAddress,omitempty" maxLength:"254"`
	FromName    *string `json:"fromName,omitempty" maxLength:"100"`
}

type settingsInput struct{ Body settingsPatch }

type settingsOutput struct{ Body settings }

type testOutput struct{ Body Outcome }

func settingsOf(s mail.Settings) *settingsOutput {
	return &settingsOutput{Body: settings{
		Host:        s.Host,
		Port:        s.Port,
		Security:    string(s.Security),
		Username:    s.Username,
		PasswordSet: s.PasswordSet,
		FromAddress: s.From.Email,
		FromName:    s.From.Name,
	}}
}

func (p settingsPatch) patch() mail.Patch {
	out := mail.Patch{Host: p.Host, Port: p.Port, Username: p.Username, Password: p.Password, FromAddress: p.FromAddress, FromName: p.FromName}
	if p.Security != nil {
		s := mail.Security(*p.Security)
		out.Security = &s
	}
	return out
}

func Routes(r *api.Router, lib Library) {
	o := r.Owner("mail", api.Admin)

	api.Register(o, api.Admin, api.Op{
		ID:      "getMailSettings",
		Method:  http.MethodGet,
		Path:    "/mail/settings",
		Summary: "Get the SMTP server every email goes through",
		Status:  http.StatusOK,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*settingsOutput, error) {
		m, _ := open(lib)
		s, err := m.Settings(ctx)
		if err != nil {
			return nil, err
		}
		return settingsOf(s), nil
	})

	api.Register(o, api.Reconfirmed, api.Op{
		ID:      "updateMailSettings",
		Method:  http.MethodPatch,
		Path:    "/mail/settings",
		Summary: "Change the SMTP server every email goes through",
		Status:  http.StatusOK,
	}, func(ctx context.Context, _ identity.Admin, in *settingsInput) (*settingsOutput, error) {
		m, _ := open(lib)
		s, err := m.Update(ctx, in.Body.patch())
		if err != nil {
			return nil, err
		}
		return settingsOf(s), nil
	})

	api.Register(o, api.Admin, api.Op{
		ID:      "sendTestMail",
		Method:  http.MethodPost,
		Path:    "/mail/test",
		Summary: "Send a test email to your own address and answer the mail server's reply",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noRecipient},
	}, func(ctx context.Context, a identity.Admin, _ *struct{}) (*testOutput, error) {
		account, ok := a.Account()
		if !ok {
			return nil, noRecipient
		}
		m, ids := open(lib)
		set, err := ids.Settings(ctx, a)
		if err != nil {
			return nil, err
		}
		p := account.Profile()
		msg, err := mail.TestMessage(mail.Address{Name: p.DisplayName, Email: p.Email}, mail.TestEmail{ServerName: set.Address.Host(), URL: set.Address.String()})
		if err != nil {
			return nil, err
		}
		out, err := m.Send(ctx, msg)
		if err != nil {
			return nil, err
		}
		return &testOutput{Body: OutcomeOf(out)}, nil
	})
}
