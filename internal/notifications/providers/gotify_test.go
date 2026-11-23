package providers_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

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
