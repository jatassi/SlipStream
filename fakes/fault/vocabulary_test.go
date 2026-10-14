package fault_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fault"
)

// q20 is #26 Q20's list of failures every fake plays, each with the fault
// text that plays it, or why it waits.
var q20 = []struct {
	failure string
	text    string
	layer   fault.Layer
	pending string
}{
	{failure: "latency", text: "latency=200ms", layer: fault.Conn},
	{failure: "hang", text: "hang", layer: fault.HTTP},
	{failure: "refuse", text: "refuse", layer: fault.Conn},
	{failure: "reset", text: "reset", layer: fault.Conn},
	{failure: "truncate", text: "truncate", layer: fault.HTTP},
	{failure: "stall", text: "stall", layer: fault.Conn},
	{failure: "throttle", text: "throttle=256KiB/s", layer: fault.Conn},
	{failure: "TLS failure", text: "tls=untrusted", layer: fault.TLS},
	{failure: "TLS failure", text: "tls=expired", layer: fault.TLS},
	{failure: "TLS failure", text: "tls=wrong-name", layer: fault.TLS},
	{failure: "DNS failure", pending: "every Node dials literal addresses until the configurable resolver (#26 Q3)"},
	{failure: "any HTTP status", text: "status=503", layer: fault.HTTP},
	{failure: "429 with a reset header", text: "status=429,retry-after=30s,reset-header=X-RateLimit-Reset", layer: fault.HTTP},
	{failure: "malformed body", text: "malformed", layer: fault.HTTP},
	{failure: "slow body", text: "slow-body=16KiB/s", layer: fault.HTTP},
}

func TestVocabulary(t *testing.T) {
	var failures []string
	for _, f := range q20 {
		if !slices.Contains(failures, f.failure) {
			failures = append(failures, f.failure)
		}
		if f.pending != "" {
			continue
		}
		got, err := fault.Parse(f.text)
		if err != nil {
			t.Errorf("%s: Parse(%q): %v", f.failure, f.text, err)
			continue
		}
		if got.Layer() != f.layer || got.String() != f.text {
			t.Errorf("%s: %q parses to %s at the %s layer, want %s", f.failure, f.text, got, got.Layer(), f.layer)
		}
	}
	if len(failures) != 13 {
		t.Errorf("Q20 names 13 failures, the list has %d: %q", len(failures), failures)
	}
}

func TestEveryFaultRoundTrips(t *testing.T) {
	for _, c := range []struct {
		f    fault.Fault
		text string
	}{
		{fault.Refuse{}, "refuse"},
		{fault.Reset{}, "reset"},
		{fault.Stall{}, "stall"},
		{fault.Latency(200 * time.Millisecond), "latency=200ms"},
		{fault.Latency(90 * time.Second), "latency=1m30s"},
		{fault.Throttle(256 << 10), "throttle=256KiB/s"},
		{fault.Throttle(1000), "throttle=1000B/s"},
		{fault.Cut(64 << 10), "cut=64KiB"},
		{fault.Cut(3 << 20), "cut=3MiB"},
		{fault.Cut(0), "cut=0B"},
		{fault.BadTLS(fault.Untrusted), "tls=untrusted"},
		{fault.BadTLS(fault.Expired), "tls=expired"},
		{fault.BadTLS(fault.WrongName), "tls=wrong-name"},
		{fault.Status{Code: 404}, "status=404"},
		{fault.Status{Code: 503, RetryAfter: time.Second}, "status=503,retry-after=1s"},
		{fault.Status{Code: 429, RetryAfter: 30 * time.Second, ResetHeader: "X-RateLimit-Reset"}, "status=429,retry-after=30s,reset-header=X-RateLimit-Reset"},
		{fault.Hang{}, "hang"},
		{fault.Malformed{}, "malformed"},
		{fault.Truncate{}, "truncate"},
		{fault.SlowBody(1 << 30), "slow-body=1GiB/s"},
	} {
		if got := c.f.String(); got != c.text {
			t.Errorf("%#v.String() = %q, want %q", c.f, got, c.text)
		}
		if got, err := fault.Parse(c.text); err != nil || got != c.f {
			t.Errorf("Parse(%q) = %#v, %v; want %#v", c.text, got, err, c.f)
		}
	}
}

func TestParseRefuses(t *testing.T) {
	for text, want := range map[string]string{
		"dns":                         `unknown fault "dns"`,
		"stall=1s":                    "fault stall takes no value",
		"latency":                     "fault latency needs a value: latency=…",
		"latency=soon":                `fault latency=soon: duration "soon" is not a Go duration such as 200ms`,
		"latency=0s":                  "fault latency=0s: duration 0s is not positive",
		"throttle=256KiB":             `fault throttle=256KiB: rate "256KiB" needs /s`,
		"throttle=0B/s":               "fault throttle=0B/s: rate 0B/s is not positive",
		"cut=64KB":                    `fault cut=64KB: size "64KB" is not a count of B`,
		"cut=64":                      `fault cut=64: size "64" needs a unit: B, KiB, MiB or GiB`,
		"tls=self-signed":             "fault tls=self-signed: want untrusted, expired or wrong-name",
		"status=99":                   `fault status=99: status "99" is not an HTTP status code`,
		"status=429,reset-header=X-R": "fault status=429,reset-header=X-R: reset-header needs retry-after",
		"status=429,retry-after=1s,retry-after=2s": "fault status=429,retry-after=1s,retry-after=2s: retry-after given twice",
		"status=429,body=x":                        `fault status=429,body=x: unknown status parameter "body=x"`,
	} {
		if _, err := fault.Parse(text); err == nil || err.Error() != want {
			t.Errorf("Parse(%q) = %v, want %q", text, err, want)
		}
	}
}

var (
	tmdbFront = fault.Front{Target: "tmdb", Origin: "home", Layers: fault.Conn | fault.TLS | fault.HTTP}
	tmdbBox   = fault.Front{Target: "tmdb", Origin: "seedbox", Layers: fault.Conn | fault.TLS | fault.HTTP}
	linkFront = fault.Front{Target: "link/home", Layers: fault.Conn}
	seedA     = fault.Front{Target: "seeders/a", Origin: "home", Layers: fault.Conn}
	fronts    = []fault.Front{tmdbFront, tmdbBox, linkFront, seedA}
)

func TestARuleAFrontCantPlayIsRefused(t *testing.T) {
	for _, c := range []struct {
		rule fault.Rule
		want string
	}{
		{fault.Rule{ID: "x", Target: "link/home", Fault: fault.Status{Code: 429}}, "status=429 is an HTTP fault and link/home carries no HTTP"},
		{fault.Rule{ID: "x", Target: "seeders", Fault: fault.BadTLS(fault.Expired)}, "tls=expired is a TLS fault and seeders carries no TLS"},
		{fault.Rule{ID: "x", Target: "tmdb", Route: "movie", Fault: fault.Throttle(1024)}, "throttle=1KiB/s is a connection fault and takes no route or params"},
		{fault.Rule{ID: "x", Target: "tmdbx", Fault: fault.Reset{}}, "no target tmdbx"},
		{fault.Rule{ID: "x", Target: "tmdb", Origin: "nas", Fault: fault.Reset{}}, "tmdb has no front for nas"},
		{fault.Rule{ID: "x", Target: "link/home", Origin: "seedbox", Fault: fault.Stall{}}, "link/home is the one front every peer dials, so a rule can't pick seedbox's connections; per-pair link faults stay in the simulation tier"},
		{fault.Rule{ID: "x", Target: "tmdb", Fault: fault.Refuse{}, Times: 1}, "refuse holds a front until cleared and takes no skip or times"},
		{fault.Rule{ID: "x", Target: "tmdb", Fault: fault.Hang{}, Skip: -1}, `rule "x": skip and times can't be negative`},
		{fault.Rule{ID: "x", Target: "tmdb"}, `rule "x" has no fault`},
	} {
		if _, err := fault.NewPlan(fronts, nil).Set(c.rule); err == nil || err.Error() != c.want {
			t.Errorf("Set(%+v) = %v, want %q", c.rule, err, c.want)
		}
	}
	for _, ok := range []fault.Rule{
		{ID: "x", Target: "seeders", Fault: fault.Throttle(1024)},
		{ID: "x", Target: "tmdb", Origin: "seedbox", Route: "movie", Params: map[string]string{"id": "1"}, Fault: fault.Status{Code: 429}},
		{ID: "x", Target: "link/home", Fault: fault.Cut(256 << 10)},
	} {
		if err := fault.Check(ok, fronts); err != nil {
			t.Errorf("Check(%+v) = %v", ok, err)
		}
	}
}

func arm(t *testing.T, p *fault.Plan, r fault.Rule) {
	t.Helper()
	if _, err := p.Set(r); err != nil {
		t.Fatal(err)
	}
}

var ok = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("real")) })

func serve(h http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/3/movie/1", http.NoBody))
	return rec
}

// pipeEnd is one end of a pipe whose writes never wait for a reader.
func pipeEnd(t *testing.T) net.Conn {
	t.Helper()
	a, b := fault.Pipe(&net.TCPAddr{}, &net.TCPAddr{})
	t.Cleanup(func() { _ = a.Close(); _ = b.Close() })
	return a
}

type firings struct {
	mu  sync.Mutex
	got []string
}

func (f *firings) add(x fault.Firing) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.got = append(f.got, fmt.Sprintf("%s %s@%s %s #%d", x.Rule, x.Front.Target, x.Front.Origin, x.Fault, x.Match))
}

func (f *firings) sorted() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Sorted(slices.Values(f.got))
}

func TestAPlanCountsEachOccurrenceOnceUnderConcurrency(t *testing.T) {
	var heard firings
	p := fault.NewPlan(fronts, heard.add)
	arm(t, p, fault.Rule{ID: "429", Target: "tmdb", Fault: fault.Status{Code: 429}, Skip: 2, Times: 3})
	arm(t, p, fault.Rule{ID: "slow", Target: "tmdb", Fault: fault.Status{Code: 503}})
	arm(t, p, fault.Rule{ID: "drop", Target: "tmdb", Origin: "home", Fault: fault.Reset{}, Skip: 1, Times: 2})
	handler := p.HTTP("tmdb", "movie", func(*http.Request) string { return "home" }, ok)

	var wg sync.WaitGroup
	codes := make(chan int, 40)
	for range 40 {
		wg.Go(func() { codes <- serve(handler).Code })
	}
	conns := make(chan net.Conn, 10)
	for range 10 {
		wg.Go(func() {
			a, b := net.Pipe()
			_ = b.Close()
			conns <- p.Conn(a, tmdbFront)
		})
	}
	wg.Wait()
	close(codes)
	close(conns)
	count := map[int]int{}
	for c := range codes {
		count[c]++
	}
	if want := map[int]int{200: 2, 429: 3, 503: 35}; !maps.Equal(count, want) {
		t.Errorf("40 requests answered %v, want %v: two skipped, three played, then the next rule", count, want)
	}
	reset := 0
	for c := range conns {
		if _, err := c.Write([]byte("x")); errors.Is(err, fault.ErrReset) {
			reset++
		}
	}
	if reset != 2 {
		t.Errorf("%d of 10 connections were reset, want 2", reset)
	}
	got := heard.sorted()
	want := []string{"429 tmdb@home status=429 #3", "429 tmdb@home status=429 #4", "429 tmdb@home status=429 #5", "drop tmdb@home reset #2", "drop tmdb@home reset #3"}
	got = slices.DeleteFunc(got, func(s string) bool { return strings.HasPrefix(s, "slow ") })
	if !slices.Equal(got, want) {
		t.Errorf("fired %q, want %q", got, want)
	}
}

func TestAfterArmsOnARecordObservedAfterSet(t *testing.T) {
	var heard firings
	p := fault.NewPlan(fronts, heard.add)
	completed := events.Record{Seq: 1, Kind: "tracker.announce", Node: "seedbox", Data: []byte(`{"event":"completed"}`)}
	started := events.Record{Seq: 2, Kind: "tracker.announce", Node: "seedbox", Data: []byte(`{"event":"started"}`)}
	if fired := p.Observe(completed); len(fired) != 0 {
		t.Fatalf("Observe on an empty plan fired %+v", fired)
	}
	open := p.Conn(pipeEnd(t), linkFront)
	arm(t, p, fault.Rule{ID: "cut", Target: "link/home", Fault: fault.Cut(0), Times: 1, After: &events.Query{Kind: "tracker.announce", Match: map[string]string{"event": "completed"}}})
	if fired := p.Observe(started); len(fired) != 0 {
		t.Fatalf("a started announce fired %+v", fired)
	}
	if _, err := open.Write([]byte("x")); err != nil {
		t.Fatalf("before the rule armed a write failed: %v", err)
	}
	completed.Seq = 3
	fired := p.Observe(completed)
	if len(fired) != 1 || fired[0] != (fault.Firing{Rule: "cut", Front: linkFront, Fault: fault.Cut(0), Match: 1}) {
		t.Fatalf("the completed announce fired %+v, want the open link connection, match 1", fired)
	}
	if _, err := open.Write([]byte("x")); !errors.Is(err, fault.ErrCut) {
		t.Fatalf("after the rule armed a write gave %v, want ErrCut", err)
	}
	later := p.Conn(pipeEnd(t), linkFront)
	if _, err := later.Write([]byte("x")); err != nil {
		t.Fatalf("a connection after the one occurrence failed: %v", err)
	}
	if got := heard.sorted(); len(got) != 0 {
		t.Fatalf("fired to the callback %q; arming's occurrences belong to Observe", got)
	}
}

func TestSetReplacesByIDAndClearRemoves(t *testing.T) {
	p := fault.NewPlan(fronts, nil)
	handler := p.HTTP("tmdb", "movie", func(*http.Request) string { return "home" }, ok)
	arm(t, p, fault.Rule{ID: "a", Target: "tmdb", Fault: fault.Status{Code: 429}})
	arm(t, p, fault.Rule{ID: "a", Target: "tmdb", Fault: fault.Status{Code: 503}})
	arm(t, p, fault.Rule{ID: "b", Target: "tmdb", Fault: fault.Status{Code: 404}})
	if got := serve(handler).Code; got != 503 {
		t.Fatalf("after replacing a, answered %d, want 503", got)
	}
	if !p.Clear("a") || serve(handler).Code != 404 {
		t.Fatal("clearing a did not leave b answering 404")
	}
	if p.Clear("a") {
		t.Fatal("clearing a twice reported a removal")
	}
	if !p.Clear("") || serve(handler).Code != 200 {
		t.Fatal(`Clear("") left a rule`)
	}
}

func TestTLSCountsHandshakes(t *testing.T) {
	p := fault.NewPlan(fronts, nil)
	arm(t, p, fault.Rule{ID: "tls", Target: "tmdb", Origin: "seedbox", Fault: fault.BadTLS(fault.WrongName), Skip: 1, Times: 1})
	var got []string
	for _, f := range []fault.Front{tmdbBox, tmdbFront, tmdbBox, tmdbBox} {
		play := "-"
		if bad, ok := p.TLS(f); ok {
			play = bad.String()
		}
		got = append(got, f.Origin+":"+play)
	}
	if want := []string{"seedbox:-", "home:-", "seedbox:tls=wrong-name", "seedbox:-"}; !slices.Equal(got, want) {
		t.Fatalf("handshakes played %q, want %q", got, want)
	}
}

func TestRefusedHoldsAFrontUntilCleared(t *testing.T) {
	p := fault.NewPlan(fronts, nil)
	arm(t, p, fault.Rule{ID: "down", Target: "tmdb", Origin: "home", Fault: fault.Refuse{}})
	refused, changed := p.Refused(tmdbFront)
	other, _ := p.Refused(tmdbBox)
	if !refused || other {
		t.Fatalf("Refused = %t at home and %t at seedbox, want only home", refused, other)
	}
	p.Clear("down")
	select {
	case <-changed:
	default:
		t.Fatal("clearing the rule did not signal the change")
	}
	if refused, _ := p.Refused(tmdbFront); refused {
		t.Fatal("the front stays refused after the clear")
	}
}
