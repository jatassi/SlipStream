package providers_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestPushbulletReadsItsResponses(t *testing.T) {
	now := providertest.Now
	reset := now.Add(40 * time.Minute)
	providertest.Classify(t, provider(t, "pushbullet"), []providertest.ClassifyCase{
		{Name: "a push", Reply: providertest.Post(200, `{"iden":"ujpah72o0","active":true}`), Want: providers.Verdict{Outcome: providers.Sent}},
		{Name: "a limit waits until it resets", Reply: providertest.Post(429, `{"error":{"code":"too_many_requests"}}`, "X-Ratelimit-Reset", itoa(reset.Unix())), Want: slow(reset, 429)},
		{
			Name:  "a wrong token refuses the connection",
			Reply: providertest.Post(401, `{"error":{"code":"invalid_access_token","type":"invalid_request"}}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Unauthorized, Status: 401, Field: "accessToken"}},
		},
		{
			Name:  "a 413 gives the message up, since it has no poster to drop",
			Reply: providertest.Post(413, ``),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.TooLarge, Status: 413}, Response: []byte(``)},
		},
	})
}
