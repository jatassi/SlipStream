package providers_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

var ntfyCase = providertest.Case{
	Settings: providertest.Raw(map[string]any{"priority": "4"}),
	Secrets:  providertest.Raw(map[string]any{"topic": "slipstream-ntfy-topic", "accessToken": "tk_ntfy"}),
}

func TestNtfyBuildsEachMessageAsItsGolden(t *testing.T) {
	providertest.Golden(t, provider(t, "ntfy"), ntfyCase)
}

func TestNtfySendsToTheServerEnteredOrNtfySh(t *testing.T) {
	p := provider(t, "ntfy")
	own := providertest.Case{Settings: providertest.Raw(map[string]any{"serverUrl": "https://ntfy.lan/push", "priority": "3"}), Secrets: ntfyCase.Secrets}
	for c, want := range map[*providertest.Case]string{&ntfyCase: "https://ntfy.sh/slipstream-ntfy-topic", &own: "https://ntfy.lan/push/slipstream-ntfy-topic"} {
		out, err := p.Build(providertest.Payload(p, *c, providertest.FixtureOf("test"), false, providers.DefaultEndpoints()))
		if err != nil {
			t.Fatal(err)
		}
		if out.Post.URL != want {
			t.Errorf("URL = %s, want %s", out.Post.URL, want)
		}
		host, _ := p.Host(providers.Values{"serverUrl": c.Settings["serverUrl"]}, providers.DefaultEndpoints())
		if host.String() != want[:len(host.String())] {
			t.Errorf("host = %s for %s", host, want)
		}
	}
}

func TestNtfyReadsItsResponses(t *testing.T) {
	now := providertest.Now
	providertest.Classify(t, provider(t, "ntfy"), []providertest.ClassifyCase{
		{Name: "a message", Reply: providertest.Post(200, `{"id":"sPs71M8A2T","event":"message"}`), Want: providers.Verdict{Outcome: providers.Sent}},
		{
			Name:  "a request limit waits its Retry-After",
			Reply: providertest.Post(429, `{"code":42901,"http":429,"error":"limit reached: too many requests"}`, "Retry-After", "5"),
			Want:  slow(now.Add(5*time.Second), 429),
		},
		{
			Name:  "a protected topic refuses the connection",
			Reply: providertest.Post(403, `{"code":40301,"http":403,"error":"forbidden"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Unauthorized, Status: 403, Field: "accessToken"}},
		},
		{
			Name:  "an attachment too large may pass without it",
			Reply: providertest.Post(413, `{"code":41301,"http":413,"error":"attachment too large, or bandwidth limit reached"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.TooLarge, Status: 413}, WithoutImage: true, Response: []byte(`{"code":41301,"http":413,"error":"attachment too large, or bandwidth limit reached"}`)},
		},
	})
}
