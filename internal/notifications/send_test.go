package notifications

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/notifications/providers"
)

func TestAKeptResponseMasksEverySecretAndKeepsOneKiB(t *testing.T) {
	secrets := providers.Values{
		"token":   json.RawMessage(`"s3cret"`),
		"headers": json.RawMessage(`[{"name":"X-Key","value":"h3ader"}]`),
	}
	got := string(masked([]byte(`{"error":"s3cret and h3ader refused"}`), secrets))
	if want := `{"error":"[secret] and [secret] refused"}`; got != want {
		t.Fatalf("masked = %q, want %q", got, want)
	}
	long := masked([]byte(strings.Repeat("é", maxResponse)), nil)
	if len(long) > maxResponse || !utf8.Valid(long) {
		t.Fatalf("a long response kept %d bytes, valid %v", len(long), utf8.Valid(long))
	}
}
