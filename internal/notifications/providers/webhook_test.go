package providers_test

import (
	"encoding/json"
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestTheWebhookBodyIsVersionOne(t *testing.T) {
	p := provider(t, "webhook")
	out, err := p.Build(providertest.Payload(p, providertest.Cases["webhook"], providertest.FixtureOf("upgraded"), true, providers.DefaultEndpoints()))
	if err != nil {
		t.Fatal(err)
	}
	var body struct {
		Version int               `json:"version"`
		Event   string            `json:"event"`
		At      string            `json:"at"`
		Image   string            `json:"image"`
		Work    map[string]string `json:"work"`
		Files   []map[string]any  `json:"files"`
		Health  []map[string]any  `json:"health"`
	}
	if err := json.Unmarshal(out.Post.Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Version != 1 || body.Event != "upgraded" || body.At != "2026-11-16T14:02:00Z" || body.Image == "" || body.Work["id"] != "w1" || len(body.Files) != 1 || body.Health != nil {
		t.Errorf("body = %+v", body)
	}
}

func TestWebhookReadsItsResponses(t *testing.T) {
	providertest.Classify(t, provider(t, "webhook"), []providertest.ClassifyCase{
		{Name: "a 204", Reply: providertest.Post(204, ``), Want: providers.Verdict{Outcome: providers.Sent}},
		{
			Name:  "a 401 refuses the connection",
			Reply: providertest.Post(401, ``),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Unauthorized, Status: 401, Field: "password"}},
		},
		{
			Name:  "a 413 may pass without the image URL",
			Reply: providertest.Post(413, ``),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.TooLarge, Status: 413}, WithoutImage: true, Response: []byte(``)},
		},
		{
			Name:  "a 503 with Retry-After slows down",
			Reply: providertest.Post(503, ``, "Retry-After", "120"),
			Want:  slow(providertest.Now.Add(120e9), 503),
		},
		{Name: "a 503 without one is retried", Reply: providertest.Post(503, ``), Want: transient(providers.Reason{Code: providers.ServerError, Status: 503})},
	})
}
