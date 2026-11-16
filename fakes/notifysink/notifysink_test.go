package notifysink_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/notifysink"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifications/providers/providertest"
)

// posted are the providers that post to a host, which the sink stands in
// for. Email and Custom Script have their own fakes.
var posted = []providers.ID{"discord", "telegram", "slack", "pushover", "gotify", "ntfy", "pushbullet", "apprise", "webhook"}

// sinkCase is id's fixture connection with each URL the Admin types moved
// to the sink's front, under the profile's prefix.
func sinkCase(t *testing.T, f faketest.Served, id providers.ID) providertest.Case {
	t.Helper()
	c := providertest.Cases[id]
	moved := func(v providers.Values) providers.Values {
		out := providers.Values{}
		for k, raw := range v {
			out[k] = raw
			var s string
			if json.Unmarshal(raw, &s) != nil || !strings.HasPrefix(s, "http") {
				continue
			}
			i := strings.Index(s[len("https://"):], "/")
			path := ""
			if i >= 0 {
				path = s[len("https://")+i:]
			}
			out[k], _ = json.Marshal(f.Address.URL(notifysink.Prefix(id) + path))
		}
		return out
	}
	return providertest.Case{Settings: moved(c.Settings), Secrets: moved(c.Secrets)}
}

func sinkEndpoints(f faketest.Served) providers.Endpoints {
	var e node.EndpointsBlock
	notifysink.Fake.Endpoints(&e, f.Address)
	return e.Notify
}

// send builds the imported fixture for id, with its poster when id takes
// one, posts it to the sink and reads the reply as the provider does.
func send(t *testing.T, f faketest.Served, id providers.ID) (providers.Verdict, string) {
	t.Helper()
	p, _ := providers.ByID(id)
	poster := p.Image != providers.NoImage
	out, err := p.Build(providertest.Payload(p, sinkCase(t, f, id), providertest.FixtureOf("imported"), poster, sinkEndpoints(f)))
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequestWithContext(t.Context(), out.Post.Method, out.Post.URL, bytes.NewReader(out.Post.Body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header = out.Post.Header
	res, err := f.Client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	body, _ := io.ReadAll(res.Body)
	header := http.Header{}
	for name, values := range res.Header {
		if name == "Retry-After" || strings.HasPrefix(name, "X-Ratelimit-") || strings.HasPrefix(name, "X-Limit-App-") {
			header[name] = values
		}
	}
	return p.Classify(providers.Reply{Post: &providers.Response{Status: res.StatusCode, Header: header, Body: body}}, time.Now()), providertest.Name(providertest.FixtureOf("imported"), poster)
}

func TestEachProfileCapturesTheRequestItsProviderGoldenShows(t *testing.T) {
	f := faketest.Loopback(t, notifysink.Fake, scenario.Scenario{Seed: "notify"}, faketest.As("den"))
	for _, id := range posted {
		t.Run(string(id), func(t *testing.T) {
			before := len(f.Records(t))
			v, golden := send(t, f, id)
			if v.Outcome != providers.Sent {
				t.Fatalf("the sink's answer reads as %+v, want sent", v)
			}
			records := f.Records(t)[before:]
			if len(records) != 1 || records[0].Kind != notifysink.Notify.Kind || records[0].Node != "den" {
				t.Fatalf("captured %+v, want one notify.request from den", records)
			}
			var got notifysink.Request
			if err := json.Unmarshal(records[0].Data, &got); err != nil {
				t.Fatal(err)
			}
			want, err := os.ReadFile(filepath.Join("..", "..", "internal", "notifications", "providers", "testdata", string(id), golden+".golden"))
			if err != nil {
				t.Fatal(err)
			}
			if got.Profile != string(id) || got.Dump != string(want) {
				t.Errorf("profile %s captured\n%s\nwant its golden %s\n%s", got.Profile, got.Dump, golden, want)
			}
			if p, _ := providers.ByID(id); p.Image == providers.Bytes {
				kept, err := os.ReadFile(filepath.Join(f.RunDir, records[0].Files["poster.jpg"]))
				if err != nil || !bytes.Equal(kept, providertest.PosterBytes) {
					t.Errorf("kept poster %q, %v", kept, err)
				}
			}
		})
	}
}

func TestEachProfilesCuesAnswerTheShapesItsProviderReads(t *testing.T) {
	f := faketest.Loopback(t, notifysink.Fake, scenario.Scenario{Seed: "notify"})
	refusedAs := map[providers.ID]providers.ReasonCode{"telegram": providers.ChatNotFound, "pushover": providers.UserUnknown}
	for _, id := range posted {
		t.Run(string(id), func(t *testing.T) {
			for _, c := range []struct {
				cue     string
				args    any
				outcome providers.Outcome
				code    providers.ReasonCode
			}{
				{notifysink.RateLimited, map[string]int{"seconds": 30}, providers.SlowDown, providers.RateLimited},
				{notifysink.Refused, nil, providers.RefusedConnection, cmp(refusedAs[id], providers.Unauthorized)},
				{notifysink.Gone, nil, providers.RefusedConnection, providers.Gone},
				{notifysink.Down, nil, providers.Transient, providers.ServerError},
			} {
				f.Cue(t, notifysink.Cue(id, c.cue), c.args)
				v, _ := send(t, f, id)
				f.Cue(t, notifysink.Cue(id, c.cue), map[string]bool{"off": true})
				if v.Outcome != c.outcome || v.Reason.Code != c.code {
					t.Errorf("%s reads as %+v, want outcome %d with %s", c.cue, v, c.outcome, c.code)
				}
				if c.cue == notifysink.RateLimited && time.Until(v.Until) < 20*time.Second {
					t.Errorf("%s waits until %s, want about 30 s", c.cue, v.Until)
				}
			}
			if v, _ := send(t, f, id); v.Outcome != providers.Sent {
				t.Errorf("with every cue off, the reply reads as %+v", v)
			}
		})
	}
}

func TestSlowHoldsEachAnswerForItsDelay(t *testing.T) {
	f := faketest.Loopback(t, notifysink.Fake, scenario.Scenario{Seed: "notify"})
	f.Cue(t, notifysink.Cue("discord", notifysink.Slow), map[string]string{"delay": "300ms"})
	start := time.Now()
	if v, _ := send(t, f, "discord"); v.Outcome != providers.Sent || time.Since(start) < 300*time.Millisecond {
		t.Errorf("a slow answer came after %s as %+v", time.Since(start), v)
	}
}

func cmp(a, b providers.ReasonCode) providers.ReasonCode {
	if a != "" {
		return a
	}
	return b
}
