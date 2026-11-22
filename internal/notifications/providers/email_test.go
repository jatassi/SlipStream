package providers_test

import (
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestEmailSendsAgainOnlyToTheAddressesThatDidntTakeIt(t *testing.T) {
	p := provider(t, "email")
	payload := providertest.Payload(p, providertest.Cases["email"], providertest.FixtureOf("test"), false, providers.DefaultEndpoints())
	payload.Accepted = []string{"sam@example.com"}
	out, err := p.Build(payload)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Mail) != 1 || out.Mail[0].To != "alex@example.com" {
		t.Errorf("Build = %+v, want one mail to alex@example.com", out.Mail)
	}
}

func TestEmailReadsEachReplyByStepAndPermanence(t *testing.T) {
	sent := providers.MailReply{To: "sam@example.com", Sent: true}
	failed := func(step providers.MailStep, permanent bool) providers.Reply {
		return providers.Reply{Mail: []providers.MailReply{sent, {To: "alex@example.com", Permanent: permanent, Step: step}}}
	}
	took := []string{"sam@example.com"}
	providertest.Classify(t, provider(t, "email"), []providertest.ClassifyCase{
		{
			Name:  "every address took it",
			Reply: providers.Reply{Mail: []providers.MailReply{sent, {To: "alex@example.com", Sent: true}}},
			Want:  providers.Verdict{Outcome: providers.Sent, Accepted: []string{"sam@example.com", "alex@example.com"}},
		},
		{
			Name:  "no mail server is set up",
			Reply: providers.Reply{Mail: []providers.MailReply{{To: "sam@example.com", NotSetUp: true}}},
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.MailNotSetUp}},
		},
		{
			Name:  "a refused sign-in refuses the connection",
			Reply: failed(providers.StepConnect, true),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.MailAuth}, Accepted: took},
		},
		{
			Name:  "a refused address refuses the connection",
			Reply: failed(providers.StepRcpt, true),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.MailRecipient, To: "alex@example.com", Field: "addresses"}, Accepted: took},
		},
		{
			Name:  "a refused message gives it up",
			Reply: failed(providers.StepData, true),
			Want:  providers.Verdict{Outcome: providers.RefusedContent, Reason: providers.Reason{Code: providers.MailData}, Accepted: took},
		},
		{
			Name:  "a deferral is retried to the address that didn't take it",
			Reply: failed(providers.StepRcpt, false),
			Want:  providers.Verdict{Outcome: providers.Transient, Reason: providers.Reason{Code: providers.Network}, Accepted: took},
		},
		{Name: "a silent Node is retried", Reply: providers.Reply{Down: providers.Reason{Code: providers.NodeSilent}}, Want: transient(providers.Reason{Code: providers.NodeSilent})},
	})
}
