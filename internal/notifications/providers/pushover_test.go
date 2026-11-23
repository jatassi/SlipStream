package providers_test

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestPushoverReadsItsResponses(t *testing.T) {
	now := providertest.Now
	reset := now.Add(15 * 24 * time.Hour)
	providertest.Classify(t, provider(t, "pushover"), []providertest.ClassifyCase{
		{Name: "a message", Reply: providertest.Post(200, `{"status":1,"request":"5042853c"}`), Want: providers.Verdict{Outcome: providers.Sent}},
		{
			Name:  "an unknown user refuses the connection",
			Reply: providertest.Post(400, `{"user":"invalid","errors":["user identifier is not a valid user, group, or subscribed user key"],"status":0,"request":"5042853c"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.UserUnknown, Status: 400, Field: "userKey"}},
		},
		{
			Name:  "a wrong application token refuses the connection",
			Reply: providertest.Post(400, `{"token":"invalid","errors":["application token is invalid"],"status":0,"request":"5042853c"}`),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.Unauthorized, Status: 400, Field: "appToken"}},
		},
		{
			Name:  "a spent monthly limit waits until it resets",
			Reply: providertest.Post(429, `{"status":0}`, "X-Limit-App-Remaining", "0", "X-Limit-App-Reset", itoa(reset.Unix())),
			Want:  slow(reset, 429),
		},
		{
			Name:  "a 413 may pass without the poster",
			Reply: providertest.Post(413, `{"status":0}`),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.TooLarge, Status: 413}, WithoutImage: true, Response: []byte(`{"status":0}`)},
		},
	})
}
