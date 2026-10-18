package mail

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/textproto"

	gomail "github.com/wneessen/go-mail"
	"github.com/wneessen/go-mail/smtp"

	"github.com/jatassi/slipstream/internal/storage"
)

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

type SMTP struct {
	Server  Server
	Timings Timings
	// Roots are the CAs a server's certificate must chain to, nil for the
	// system's.
	Roots *x509.CertPool
	// Dial reaches the server, nil for the OS network.
	Dial func(ctx context.Context, network, address string) (net.Conn, error)
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
	conn, err := client.DialToSMTPClientWithContext(ctx)
	if err != nil {
		return Outcome{Kind: Failed, Reply: reply(err)}, nil
	}
	defer func() { _ = client.CloseWithSMTPClient(conn) }()
	if err := s.deliver(conn, m.To, msg); err != nil {
		return Outcome{Kind: Failed, Reply: reply(err)}, nil
	}
	return Outcome{Kind: Sent}, nil
}

// deliver runs the envelope and the data itself rather than through
// go-mail's Send, whose errors hide the server's reply.
func (s SMTP) deliver(c *smtp.Client, to Address, msg *gomail.Msg) error {
	if err := c.Mail(s.Server.From.Email); err != nil {
		return fmt.Errorf("MAIL FROM: %w", err)
	}
	if err := c.Rcpt(to.Email); err != nil {
		return fmt.Errorf("RCPT TO: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if _, err := msg.WriteTo(w); err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("DATA: %w", err)
	}
	return nil
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
	config := &tls.Config{ServerName: s.Server.Host, RootCAs: s.Roots, MinVersion: tls.VersionTLS12}
	opts := []gomail.Option{
		gomail.WithPort(int(s.Server.Port)),
		gomail.WithTimeout(s.Timings.Send),
		gomail.WithTLSConfig(config),
		gomail.WithDialContextFunc(s.dialer(config)),
	}
	switch s.Server.Security {
	case None:
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	case StartTLS:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	case TLS:
		// The dial already finished the handshake. go-mail reads it from
		// the connection, so AUTH knows it is encrypted, and the server
		// offers no STARTTLS to take.
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSOpportunistic))
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

// dialer reaches the server through Dial, finishing a TLS handshake there
// for TLS.
func (s SMTP) dialer(config *tls.Config) gomail.DialContextFunc {
	dial := s.Dial
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		c, err := dial(ctx, network, address)
		if err != nil || s.Server.Security != TLS {
			return c, err
		}
		secured := tls.Client(c, config)
		if err := secured.HandshakeContext(ctx); err != nil {
			_ = c.Close()
			return nil, fmt.Errorf("TLS handshake with %s: %w", address, err)
		}
		return secured, nil
	}
}

func reply(err error) string {
	if r, ok := errors.AsType[*textproto.Error](err); ok {
		return fmt.Sprintf("%d %s", r.Code, r.Msg)
	}
	return err.Error()
}
