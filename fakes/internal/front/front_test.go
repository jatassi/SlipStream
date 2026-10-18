package front_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/internal/ca"
	"github.com/jatassi/slipstream/fakes/internal/front"
	"github.com/jatassi/slipstream/fakes/scenario"
)

// journal keeps the records the fronts write.
type journal struct {
	mu   sync.Mutex
	recs []events.Record
}

func (j *journal) record(r events.Record) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.recs = append(j.recs, r)
}

// lines are the records as "kind node data", once n of them are written.
func (j *journal) lines(t *testing.T, n int) []string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		j.mu.Lock()
		var out []string
		for _, r := range j.recs {
			out = append(out, r.Kind+" "+r.Node+" "+string(r.Data))
		}
		j.mu.Unlock()
		if len(out) >= n || time.Now().After(deadline) {
			return out
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func newSet(t *testing.T, fronts ...fault.Front) (*fault.Plan, *front.Set, *journal) {
	t.Helper()
	j := &journal{}
	plan := fault.NewPlan(fronts, func(f fault.Firing) {
		j.record(events.Record{Kind: events.FaultFired, Node: f.Front.Origin, Data: json.RawMessage(fmt.Sprintf(`{"rule":%q,"match":%d}`, f.Rule, f.Match))})
	})
	set := front.NewSet(plan, j.record, slog.New(slog.NewTextHandler(t.Output(), nil)))
	t.Cleanup(set.Close)
	return plan, set, j
}

func listen(t *testing.T, set *front.Set, f fault.Front) *front.Front {
	t.Helper()
	fr, err := set.Listen(t.Context(), f)
	if err != nil {
		t.Fatal(err)
	}
	return fr
}

// served serves an HTTP fake that names the origin of each request behind
// a front for each of origins, with TLS from authority.
func served(t *testing.T, plan *fault.Plan, set *front.Set, authority *ca.CA, origins ...string) map[string]*front.Front {
	t.Helper()
	good, err := authority.Leaf("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	expired, err := authority.ExpiredLeaf("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	fronts := map[string]*front.Front{}
	for _, o := range origins {
		fronts[o] = listen(t, set, fault.Front{Target: "echo", Origin: o, Layers: fault.Conn | fault.TLS | fault.HTTP})
	}
	in := front.NewInbox(fronts[origins[0]].Addr())
	for _, fr := range fronts {
		fr.Serve(front.Served(in, front.TLS(plan, fr.Fault(), good, map[fault.TLSFailure]tls.Certificate{fault.Expired: expired})))
	}
	s := scenario.Scenario{Seed: "front"}
	server := fake.HTTP(fake.NewEnv(s, fake.Spec{Name: "echo"}, slog.New(slog.NewTextHandler(t.Output(), nil)), plan, nil), fake.Routes{"echo": {Pattern: "/", Handler: func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "from "+fake.OriginOf(r.Context()))
	}}})
	ctx, cancel := context.WithCancel(context.WithoutCancel(t.Context()))
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, in) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	return fronts
}

func get(t *testing.T, authority *ca.CA, fr *front.Front) (string, error) {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: authority.Pool(), MinVersion: tls.VersionTLS12}}}
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+fr.Addr().String()+"/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET through the front: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	return string(body), err
}

func TestAServedFrontNamesItsOriginAndLogsEachConnection(t *testing.T) {
	authority, err := ca.New("front")
	if err != nil {
		t.Fatal(err)
	}
	fronts := []fault.Front{{Target: "echo", Origin: "home", Layers: fault.Conn | fault.TLS | fault.HTTP}, {Target: "echo", Origin: "seedbox", Layers: fault.Conn | fault.TLS | fault.HTTP}}
	plan, set, j := newSet(t, fronts...)
	served := served(t, plan, set, authority, "home", "seedbox")

	for _, o := range []string{"seedbox", "home"} {
		if got, err := get(t, authority, served[o]); err != nil || got != "from "+o {
			t.Fatalf("GET through %s's front = %q, %v; want from %s", o, got, err, o)
		}
	}
	got := j.lines(t, 4)
	var opens, closes []string
	for _, l := range got {
		kind, rest, _ := strings.Cut(l, " ")
		node, raw, _ := strings.Cut(rest, " ")
		if kind == front.Open {
			opens = append(opens, l)
			continue
		}
		var c struct {
			Conn    uint64
			In, Out int64
			Cause   string
		}
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			t.Fatal(err)
		}
		if c.In == 0 || c.Out == 0 {
			t.Errorf("%s carried no bytes", l)
		}
		closes = append(closes, fmt.Sprintf("%s %d %s", node, c.Conn, c.Cause))
	}
	if want := []string{
		front.Open + ` seedbox {"conn":1,"target":"echo"}`,
		front.Open + ` home {"conn":2,"target":"echo"}`,
	}; !slices.Equal(opens, want) {
		t.Fatalf("front.open records = %q, want %q", opens, want)
	}
	slices.Sort(closes)
	if want := []string{"home 2 closed", "seedbox 1 closed"}; !slices.Equal(closes, want) {
		t.Fatalf("front.close records end %q, want %q", closes, want)
	}
}

func TestATLSRulePicksTheLeaf(t *testing.T) {
	authority, err := ca.New("front")
	if err != nil {
		t.Fatal(err)
	}
	f := fault.Front{Target: "echo", Origin: "home", Layers: fault.Conn | fault.TLS | fault.HTTP}
	plan, set, _ := newSet(t, f)
	fr := served(t, plan, set, authority, "home")["home"]
	if _, err := plan.Set(fault.Rule{ID: "old", Target: "echo", Fault: fault.BadTLS(fault.Expired), Times: 1}); err != nil {
		t.Fatal(err)
	}
	_, err = get(t, authority, fr)
	if ce, ok := errors.AsType[*tls.CertificateVerificationError](err); !ok || !strings.Contains(ce.Err.Error(), "expired") {
		t.Fatalf("GET under tls=expired = %v, want an expired certificate", err)
	}
	var invalid x509.CertificateInvalidError
	if !errors.As(err, &invalid) || invalid.Reason != x509.Expired {
		t.Fatalf("GET under tls=expired = %v, want x509 expired", err)
	}
	if got, err := get(t, authority, fr); err != nil || got != "from home" {
		t.Fatalf("GET once the rule played = %q, %v", got, err)
	}
}

func TestARefusedFrontBindsItsPortAgainWhenCleared(t *testing.T) {
	f := fault.Front{Target: "seeders/a", Origin: "home", Layers: fault.Conn}
	plan, set, _ := newSet(t, f)
	upstream := echoServer(t)
	fr := listen(t, set, f)
	fr.Serve(front.Forward(func() (netip.AddrPort, error) { return upstream, nil }))
	if got := roundTrip(t, fr.Addr(), "one"); got != "one" {
		t.Fatalf("echo = %q", got)
	}
	if _, err := plan.Set(fault.Rule{ID: "down", Target: "seeders", Fault: fault.Refuse{}}); err != nil {
		t.Fatal(err)
	}
	until(t, "the front refuses", func() bool {
		c, err := dial(t, fr.Addr())
		if err == nil {
			_ = c.Close()
		}
		return errors.Is(err, connRefused)
	})
	plan.Clear("down")
	until(t, "the front listens again", func() bool {
		c, err := dial(t, fr.Addr())
		if err == nil {
			_ = c.Close()
		}
		return err == nil
	})
	if got := roundTrip(t, fr.Addr(), "two"); got != "two" {
		t.Fatalf("echo after the refuse cleared = %q", got)
	}
}

func TestAForwardedFrontLogsTheFaultThatClosedIt(t *testing.T) {
	f := fault.Front{Target: "seeders/a", Origin: "home", Layers: fault.Conn}
	plan, set, j := newSet(t, f)
	upstream := echoServer(t)
	fr := listen(t, set, f)
	fr.Serve(front.Forward(func() (netip.AddrPort, error) { return upstream, nil }))
	if _, err := plan.Set(fault.Rule{ID: "short", Target: "seeders/a", Fault: fault.Cut(3), Times: 1}); err != nil {
		t.Fatal(err)
	}
	c, err := dial(t, fr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if _, err := c.Write([]byte("hey")); err != nil {
		t.Fatal(err)
	}
	if got, err := io.ReadAll(c); len(got) != 0 || err != nil {
		t.Fatalf("read through a cut=3 front = %q, %v; want the connection closed", got, err)
	}
	want := []string{
		front.Open + ` home {"conn":1,"target":"seeders/a"}`,
		events.FaultFired + ` home {"rule":"short","match":1}`,
		front.Close + ` home {"conn":1,"target":"seeders/a","in":3,"out":0,"cause":"cut"}`,
	}
	if got := j.lines(t, 3); !slices.Equal(got, want) {
		t.Fatalf("records =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func echoServer(t *testing.T) netip.AddrPort {
	t.Helper()
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = l.Close() })
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				_, _ = io.Copy(c, c)
				_ = c.Close()
			}()
		}
	}()
	return netip.MustParseAddrPort(l.Addr().String())
}

func dial(t *testing.T, addr netip.AddrPort) (net.Conn, error) {
	t.Helper()
	return new(net.Dialer).DialContext(t.Context(), "tcp", addr.String()) //nolint:wrapcheck // the dial's own error, which the test reads
}

func roundTrip(t *testing.T, addr netip.AddrPort, s string) string {
	t.Helper()
	c, err := dial(t, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if _, err := c.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, len(s))
	if _, err := io.ReadFull(c, b); err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func until(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("waiting until %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
