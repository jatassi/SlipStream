package mail

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/textproto"

	gomail "github.com/wneessen/go-mail"

	"github.com/jatassi/slipstream/internal/storage"
)

// Security is how a connection to the SMTP server is protected.
type Security string

const (
	None     Security = "none"
	StartTLS Security = "starttls"
	TLS      Security = "tls"
)

// Server is the SMTP server every email goes through, with its password in
// plaintext. A Server with no Host is not set up.
type Server struct {
	Host     string
	Port     uint16
	Security Security
	Username string
	Password string
	From     Address
}

// SMTP is the Sender that hands each message to Server.
type SMTP struct {
	Server  Server
	Timings Timings
	// Roots are the CAs a server's certificate must chain to, nil for the
	// system's.
	Roots *x509.CertPool
}

var _ Sender = SMTP{}

func (s SMTP) Send(ctx context.Context, m Message) (Outcome, error) {
	if storage.InWrite(ctx) {
		panic("mail: Send inside Write; send after the commit")
	}
	if s.Server.Host == "" {
		return Outcome{Kind: NotSetUp}, nil
	}
	msg, err := s.message(m)
	if err != nil {
		return Outcome{Kind: Failed, Reply: err.Error()}, nil
	}
	client, err := gomail.NewClient(s.Server.Host, s.options()...)
	if err != nil {
		return Outcome{Kind: Failed, Reply: err.Error()}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, s.Timings.Send)
	defer cancel()
	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return Outcome{Kind: Failed, Reply: reply(err)}, nil
	}
	return Outcome{Kind: Sent}, nil
}

func (s SMTP) message(m Message) (*gomail.Msg, error) {
	msg := gomail.NewMsg()
	if err := msg.FromFormat(s.Server.From.Name, s.Server.From.Email); err != nil {
		return nil, fmt.Errorf("from address %q: %w", s.Server.From.Email, err)
	}
	if err := msg.AddToFormat(m.To.Name, m.To.Email); err != nil {
		return nil, fmt.Errorf("to address %q: %w", m.To.Email, err)
	}
	msg.Subject(m.Subject)
	msg.SetBodyString(gomail.TypeTextPlain, m.Text)
	msg.AddAlternativeString(gomail.TypeTextHTML, m.HTML)
	return msg, nil
}

func (s SMTP) options() []gomail.Option {
	opts := []gomail.Option{
		gomail.WithPort(int(s.Server.Port)),
		gomail.WithTimeout(s.Timings.Send),
		gomail.WithTLSConfig(&tls.Config{ServerName: s.Server.Host, RootCAs: s.Roots, MinVersion: tls.VersionTLS12}),
	}
	switch s.Server.Security {
	case None:
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	case StartTLS:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	case TLS:
		opts = append(opts, gomail.WithSSL())
	}
	if s.Server.Username != "" {
		auth := gomail.SMTPAuthAutoDiscover
		if s.Server.Security == None {
			auth = gomail.SMTPAuthPlainNoEnc
		}
		opts = append(opts, gomail.WithSMTPAuth(auth), gomail.WithUsername(s.Server.Username), gomail.WithPassword(s.Server.Password))
	}
	return opts
}

// reply is the server's reply within err, or err itself when no server
// replied.
func reply(err error) string {
	if r, ok := errors.AsType[*textproto.Error](err); ok {
		return fmt.Sprintf("%d %s", r.Code, r.Msg)
	}
	return err.Error()
}
