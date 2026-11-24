package providers_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestAppriseSendsToItsURLsWithNoKey(t *testing.T) {
	p := provider(t, "apprise")
	c := providertest.Case{
		Settings: providertest.Raw(map[string]any{"serverUrl": "http://apprise.lan:8000"}),
		Secrets:  providertest.Raw(map[string]any{"urls": []string{"join://join-key/phone", "mailto://user:pass@mail.example"}}),
	}
	out, err := p.Build(providertest.Payload(p, c, providertest.FixtureOf("test"), false, providers.DefaultEndpoints()))
	if err != nil {
		t.Fatal(err)
	}
	providertest.Compare(t, "testdata/apprise/test-urls.golden", outboundDump(out, c))
}

func TestAppriseReadsItsResponses(t *testing.T) {
	providertest.Classify(t, provider(t, "apprise"), []providertest.ClassifyCase{
		{Name: "a message", Reply: providertest.Post(200, ``), Want: providers.Verdict{Outcome: providers.Sent}},
		{
			Name:  "a 424 is sent with its response",
			Reply: providertest.Post(424, `one of 2 services failed`),
			Want:  providers.Verdict{Outcome: providers.Sent, Response: []byte(`one of 2 services failed`)},
		},
		{
			Name:  "a 400 refuses the connection",
			Reply: providertest.Post(400, `invalid tag`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.BadRequest, Status: 400, Field: "key"}, Response: []byte(`invalid tag`)},
		},
		{
			Name:  "an unknown key is gone",
			Reply: providertest.Post(404, ``),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Gone, Status: 404, Field: "key"}},
		},
	})
}
