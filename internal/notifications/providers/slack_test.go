package providers_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestSlackEscapesMrkdwn(t *testing.T) {
	p := provider(t, "slack")
	f := providertest.FixtureOf("works_added")
	f.Message.Title = "Added <Tom & Jerry> to Movies"
	out, err := p.Build(providertest.Payload(p, providertest.Cases["slack"], f, false, providers.DefaultEndpoints()))
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Blocks []struct{ Text struct{ Text string } }
	}
	if err := json.Unmarshal(out.Post.Body, &m); err != nil {
		t.Fatal(err)
	}
	if got, want := m.Blocks[0].Text.Text, "*<https://slipstream.example/works/w1|Added &lt;Tom &amp; Jerry&gt; to Movies>*\nSam added it."; got != want {
		t.Errorf("section = %q, want %q", got, want)
	}
}

func TestSlackReadsItsResponses(t *testing.T) {
	now := providertest.Now
	providertest.Classify(t, provider(t, "slack"), []providertest.ClassifyCase{
		{Name: "a message", Reply: providertest.Post(200, `ok`), Want: providers.Verdict{Outcome: providers.Sent}},
		{Name: "a 429 waits its Retry-After", Reply: providertest.Post(429, ``, "Retry-After", "30"), Want: slow(now.Add(30*time.Second), 429)},
		{
			Name:  "an archived channel is gone",
			Reply: providertest.Post(410, `channel_is_archived`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Gone, Status: 410, Field: "webhookUrl"}},
		},
		{
			Name:  "a removed webhook is gone",
			Reply: providertest.Post(404, `no_service`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Gone, Status: 404, Field: "webhookUrl"}},
		},
		{
			Name:  "a payload Slack can't read gives the message up",
			Reply: providertest.Post(400, `invalid_payload`),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.Refused, Status: 400}, Response: []byte(`invalid_payload`)},
		},
	})
}
