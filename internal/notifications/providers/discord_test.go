package providers_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/outbound"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func provider(t *testing.T, id providers.ID) providers.Provider {
	t.Helper()
	p, ok := providers.ByID(id)
	if !ok {
		t.Fatalf("no provider %s", id)
	}
	return p
}

func TestDiscordReadsItsResponses(t *testing.T) {
	now := providertest.Now
	providertest.Classify(t, provider(t, "discord"), []providertest.ClassifyCase{
		{Name: "a message", Reply: providertest.Post(200, `{"id":"1"}`), Want: providers.Verdict{Outcome: providers.Sent}},
		{
			Name:  "a 429 waits its retry_after in float seconds",
			Reply: providertest.Post(429, `{"message":"You are being rate limited.","retry_after":2.5,"global":false}`, "Retry-After", "3"),
			Want:  slow(now.Add(2500*time.Millisecond), 429),
		},
		{
			Name:  "a 429 with no body waits its Retry-After",
			Reply: providertest.Post(429, ``, "Retry-After", "3"),
			Want:  slow(now.Add(3*time.Second), 429),
		},
		{
			Name:  "a deleted webhook is gone",
			Reply: providertest.Post(404, `{"message":"Unknown Webhook","code":10015}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Gone, Status: 404, Field: "webhookUrl"}},
		},
		{
			Name:  "a 413 may pass without the poster",
			Reply: providertest.Post(413, `{"message":"Request entity too large"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.TooLarge, Status: 413}, WithoutImage: true, Response: []byte(`{"message":"Request entity too large"}`)},
		},
		{
			Name:  "a 400 gives the message up",
			Reply: providertest.Post(400, `{"message":"Invalid Form Body"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.Refused, Status: 400}, Response: []byte(`{"message":"Invalid Form Body"}`)},
		},
		{Name: "a 502 is retried", Reply: providertest.Post(502, ``), Want: transient(providers.Reason{Code: providers.ServerError, Status: 502})},
		{Name: "no response is retried", Reply: providertest.Post(0, ``), Want: transient(providers.Reason{Code: providers.Network})},
		{
			Name:  "a silent Node is retried",
			Reply: providers.Reply{Down: providers.Reason{Code: providers.NodeSilent}},
			Want:  transient(providers.Reason{Code: providers.NodeSilent}),
		},
	})
}

func slow(until time.Time, status int) providers.Verdict {
	return providers.Verdict{Outcome: providers.SlowDown, Until: until, Reason: providers.Reason{Code: providers.RateLimited, Status: status, Until: until}}
}

func transient(r providers.Reason) providers.Verdict {
	return providers.Verdict{Outcome: providers.Transient, Reason: r}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func outboundDump(o providers.Outbound, c providertest.Case) string {
	return outbound.DumpOutbound(o, providertest.Masks(c.Secrets))
}
