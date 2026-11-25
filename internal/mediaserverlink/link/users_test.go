package link_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

func TestAPasswordPrintsRedacted(t *testing.T) {
	if _, err := link.NewPassword(""); err == nil {
		t.Error("an empty password was taken")
	}
	const secret = "correct horse battery"
	pw, err := link.NewPassword(secret)
	if err != nil {
		t.Fatal(err)
	}
	held := struct{ Password link.Password }{pw}
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("make", "password", pw, "held", held)
	for _, out := range []string{
		fmt.Sprintf("%v %s %+v %#v", pw, pw, pw, pw), fmt.Sprintf("%v %+v %#v", held, held, held), logged.String(),
	} {
		if strings.Contains(out, secret) || !strings.Contains(out, "[password]") {
			t.Errorf("%q shows the password", out)
		}
	}
	if pw.Reveal() != secret {
		t.Errorf("Reveal = %q, want the password", pw.Reveal())
	}
}
