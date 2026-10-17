package identitytest

import (
	"log/slog"
	"testing"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mail/mailtest"
)

// Mail captures every email identity sends, whether its caller waits or
// it goes through the Outbox, which closes when t ends.
func Mail(t testing.TB) (*mailtest.Sender, *mail.Outbox) {
	s := mailtest.New()
	o := mail.NewOutbox(s, slog.New(slog.NewTextHandler(t.Output(), nil)))
	t.Cleanup(func() { _ = o.Close(t.Context()) })
	return s, o
}
