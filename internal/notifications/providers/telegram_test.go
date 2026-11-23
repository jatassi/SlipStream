package providers_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestTelegramEscapesHTMLAndCutsTheTextToItsLimit(t *testing.T) {
	p := provider(t, "telegram")
	f := providertest.FixtureOf("imported")
	f.Message.Title = "Tom & Jerry <Uncut>"
	f.Message.Lines = []string{strings.Repeat("🎬", 3000)}
	out, err := p.Build(providertest.Payload(p, providertest.Cases["telegram"], f, false, providers.DefaultEndpoints()))
	if err != nil {
		t.Fatal(err)
	}
	var m struct{ Text string }
	if err := json.Unmarshal(out.Post.Body, &m); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(m.Text, "<b>Tom &amp; Jerry &lt;Uncut&gt;</b>\n") {
		t.Errorf("the title isn't escaped in %q", m.Text[:60])
	}
	parsed := strings.NewReplacer("<b>", "", "</b>", "", "&amp;", "&", "&lt;", "<", "&gt;", ">", `<a href="https://slipstream.example/works/w2">`, "", "</a>", "").Replace(m.Text)
	if n := len(utf16.Encode([]rune(parsed))); n > 4096 || !strings.Contains(parsed, "🎬…\nOpen in SlipStream") {
		t.Errorf("the text has %d characters once parsed, want at most 4096 with the line cut before the link", n)
	}
}

func TestTelegramReadsItsAnswersByErrorCode(t *testing.T) {
	now := providertest.Now
	providertest.Classify(t, provider(t, "telegram"), []providertest.ClassifyCase{
		{Name: "a message", Reply: providertest.Post(200, `{"ok":true,"result":{"message_id":7}}`), Want: providers.Verdict{Outcome: providers.Sent}},
		{
			Name:  "a flood wait waits parameters.retry_after",
			Reply: providertest.Post(429, `{"ok":false,"error_code":429,"description":"Too Many Requests: retry after 7","parameters":{"retry_after":7}}`),
			Want:  slow(now.Add(7*time.Second), 429),
		},
		{
			Name:  "an unknown chat refuses the connection",
			Reply: providertest.Post(400, `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.ChatNotFound, Status: 400, Field: "chatId"}},
		},
		{
			Name:  "a blocked bot refuses the connection",
			Reply: providertest.Post(403, `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Unauthorized, Status: 403, Field: "chatId"}},
		},
		{
			Name:  "a wrong token refuses the connection",
			Reply: providertest.Post(401, `{"ok":false,"error_code":401,"description":"Unauthorized"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Unauthorized, Status: 401, Field: "botToken"}},
		},
		{
			Name:  "another 400 gives the message up",
			Reply: providertest.Post(400, `{"ok":false,"error_code":400,"description":"Bad Request: can't parse entities"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.Refused, Status: 400}, Response: []byte(`{"ok":false,"error_code":400,"description":"Bad Request: can't parse entities"}`)},
		},
	})
}
