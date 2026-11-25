package plextv_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/providers/plextv"
)

func TestAMemberTokenPrintsRedacted(t *testing.T) {
	const issued = "member-legacy-token"
	token := plextv.NewMemberToken(issued)
	held := struct{ Token plextv.MemberToken }{token}
	var logged bytes.Buffer
	slog.New(slog.NewJSONHandler(&logged, nil)).Info("pin", "token", token, "held", held)
	for _, out := range []string{
		fmt.Sprintf("%v %s %+v %#v", token, token, token, token), fmt.Sprintf("%v %+v %#v", held, held, held), logged.String(),
	} {
		if strings.Contains(out, issued) || !strings.Contains(out, "[token]") {
			t.Errorf("%q shows the token", out)
		}
	}
	if token.Reveal() != issued {
		t.Errorf("Reveal = %q, want the token", token.Reveal())
	}
}
