package providers_test

import (
	"bytes"
	"testing"

	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

func TestTheScriptsEventFileIsTheWebhookBody(t *testing.T) {
	for _, f := range providertest.Fixtures {
		run, err := provider(t, "script").Build(providertest.Payload(provider(t, "script"), providertest.Cases["script"], f, false, providers.DefaultEndpoints()))
		if err != nil {
			t.Fatal(err)
		}
		hook, err := provider(t, "webhook").Build(providertest.Payload(provider(t, "webhook"), providertest.Cases["webhook"], f, false, providers.DefaultEndpoints()))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(run.Run.EventFile, hook.Post.Body) || run.Run.Event != string(f.Kind) {
			t.Errorf("%s: event %s, file %s, want the Webhook body %s", f.Kind, run.Run.Event, run.Run.EventFile, hook.Post.Body)
		}
	}
}

func TestCustomScriptReadsHowARunEnded(t *testing.T) {
	ran := func(r providers.Ran) providers.Reply { return providers.Reply{Run: &r} }
	providertest.Classify(t, provider(t, "script"), []providertest.ClassifyCase{
		{Name: "exit 0", Reply: ran(providers.Ran{}), Want: providers.Verdict{Outcome: providers.Sent}},
		{Name: "exit 3 is retried with its output", Reply: ran(providers.Ran{Exit: 3, Tail: []byte("it broke")}), Want: transient(providers.Reason{Code: providers.ScriptExit, Exit: 3, Tail: "it broke"})},
		{Name: "a stopped run is retried", Reply: ran(providers.Ran{Exit: -1, TimedOut: true, Tail: []byte("still going")}), Want: transient(providers.Reason{Code: providers.ScriptStopped, Tail: "still going"})},
		{
			Name:  "a missing path refuses the connection",
			Reply: ran(providers.Ran{Missing: true}),
			Want:  providers.Verdict{Outcome: providers.RefusedConnection, Reason: providers.Reason{Code: providers.ScriptMissing, Field: "path"}},
		},
		{Name: "an offline Node holds it", Reply: providers.Reply{Down: providers.Reason{Code: providers.NodeSilent}}, Want: transient(providers.Reason{Code: providers.NodeSilent})},
	})
}
