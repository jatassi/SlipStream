package providers_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

var gotifyCase = providertest.Case{
	Settings: providertest.Raw(map[string]any{"serverUrl": "https://gotify.example.com", "priority": 8}),
	Secrets:  providertest.Raw(map[string]any{"appToken": "gotify-app-token"}), //nolint:gosec // a fixture token
}

func TestGotifyBuildsEachMessageAsItsGolden(t *testing.T) {
	providertest.Golden(t, provider(t, "gotify"), gotifyCase)
}

func TestGotifyReadsItsResponses(t *testing.T) {
	providertest.Classify(t, provider(t, "gotify"), []providertest.ClassifyCase{
		{Name: "a message", Reply: providertest.Post(200, `{"id":25}`), Want: providers.Verdict{Outcome: providers.Sent}},
		{
			Name:  "a wrong token refuses the connection",
			Reply: providertest.Post(401, `{"error":"Unauthorized","errorCode":401}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Unauthorized, Status: 401, Field: "appToken"}},
		},
		{
			Name:  "no server at the URL refuses the connection",
			Reply: providertest.Post(404, `404 page not found`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Gone, Status: 404, Field: "serverUrl"}},
		},
	})
}
