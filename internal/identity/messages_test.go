package identity

import (
	"flag"
	"testing"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mail/mailtest"
)

var update = flag.Bool("update", false, "rewrite the golden emails under testdata/mail")

func TestEveryEmailMatchesItsGolden(t *testing.T) {
	grace := mail.Address{Name: "Grace Hopper", Email: "grace@example.com"}
	link := func(page string) linkMailData {
		return linkMailData{
			AdminName:  "Ada Okafor",
			ServerName: "slipstream.example.com",
			Email:      "grace@example.com",
			URL:        "https://slipstream.example.com/" + page + "#token=Zm9vYmFy",
			Expires:    "Friday, October 23, 2026 at 3:04 PM UTC",
		}
	}
	resent := link("signup")
	resent.Replaces = "Thursday, October 15, 2026 at 9:30 AM UTC"
	hostReset := link("reset")
	hostReset.AdminName = orYourAdmin("")
	render := func(m mail.Message, err error) mail.Message {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
		return m
	}
	cases := []struct {
		golden, subject string
		m               mail.Message
	}{
		{"signup", "Ada Okafor invited you to slipstream.example.com", render(signupMail.Render(grace, link("signup")))},
		{"signup-resent", "Ada Okafor invited you to slipstream.example.com", render(signupMail.Render(grace, resent))},
		{"reset", "Reset your PIN for slipstream.example.com", render(resetMail.Render(grace, link("reset")))},
		{"reset-from-host", "Reset your PIN for slipstream.example.com", render(resetMail.Render(grace, hostReset))},
		{"confirm-email", "Confirm your new email for slipstream.example.com", render(confirmEmailMail.Render(grace, link("confirm-email")))},
		{"pin-off", "Your SlipStream PIN was turned off", render(pinOffMail.Render(grace, pinOffData{Email: "grace@example.com", Count: 10}))},
		{"email-changed", "Your SlipStream email was changed", render(emailChangedMail.Render(grace, emailChangedData{NewEmail: "grace@hopper.example"}))},
	}
	for _, c := range cases {
		t.Run(c.golden, func(t *testing.T) {
			if c.m.Subject != c.subject || c.m.To != grace {
				t.Fatalf("to %+v subject %q, want %+v %q", c.m.To, c.m.Subject, grace, c.subject)
			}
			mailtest.Golden(t, c.m, "testdata/mail", c.golden, *update)
		})
	}
}
