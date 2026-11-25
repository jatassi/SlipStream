package reachability_test

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/reachability/judge/judgetest"
	"github.com/jatassi/slipstream/internal/reachability/reachabilitytest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
)

// wantPlex wants the Plex on runsOn at b's public address and waits for
// its first check.
func wantPlex(t *testing.T, w *reachabilitytest.World, b *judgetest.B, runsOn role.NodeID) reachability.Exposure {
	t.Helper()
	id := plex(t, w, "Plex", runsOn)
	synctest.Wait()
	e, err := w.Reach.Want(t.Context(), w.Admin, reachability.Wanted{Kind: judge.MediaServer, Server: id, Public: b.Facts().Exposure.Public.String()})
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return e.Exposure
}

func sent(calls []reachabilitytest.Call, node role.NodeID, method string) int {
	n := 0
	for _, c := range calls {
		if c.Node == node && c.Method == method {
			n++
		}
	}
	return n
}

// stamp makes node's Read answers claim at as their observed time.
func stamp(node role.NodeID, at time.Time) func(*reachabilitytest.World) {
	return func(w *reachabilitytest.World) {
		w.Site(node, func(s *reachabilitytest.Site) { s.Stamp = &at })
	}
}

// unwatched makes node's Read answer as before it follows its watch
// intent: interfaces alone, with no STUN answer.
func unwatched(node role.NodeID) func(*reachabilitytest.World) {
	return func(w *reachabilitytest.World) {
		w.Site(node, func(s *reachabilitytest.Site) { s.Facts.STUN, s.Facts.Watched = nil, false })
	}
}

func runNow(t *testing.T, w *reachabilitytest.World) {
	t.Helper()
	if _, err := w.Reach.RunNow(t.Context(), storage.NewUUID(), nil); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

func TestAUnitJudgesTheRouteFromTheNodesThatMayTestIt(t *testing.T) {
	for _, c := range []struct {
		name     string
		shape    func() *judgetest.B
		play     func(*reachabilitytest.World)
		verdict  judge.Verdict
		tested   bool
		testedBy role.NodeID
		// never are Nodes no Dial may reach.
		never []role.NodeID
	}{
		{name: "Jackson's shape is tested from atlas", shape: judgetest.OwnProxy, verdict: judge.Reachable, tested: true, testedBy: judgetest.Atlas},
		{name: "an open home is tested from seedbox", shape: judgetest.Open, verdict: judge.Reachable, tested: true, testedBy: judgetest.Seedbox, never: []role.NodeID{judgetest.Atlas}},
		{name: "den at home never tests", shape: func() *judgetest.B { return judgetest.Open().Inside() }, verdict: judge.Reachable, tested: true, testedBy: judgetest.Seedbox, never: []role.NodeID{judgetest.Den}},
		{
			name: "den never tests before atlas's network is read", shape: func() *judgetest.B { return judgetest.Open().Inside() }, verdict: judge.Fixable,
			play:  unwatched(judgetest.Atlas),
			never: []role.NodeID{judgetest.Den},
		},
		{name: "a single machine stays untested", shape: judgetest.SingleMachine, verdict: judge.Fixable},
		{
			name: "an offline tester leaves it untested", shape: judgetest.Open, verdict: judge.Reachable,
			play:  func(w *reachabilitytest.World) { w.Away(judgetest.Seedbox, true) },
			never: []role.NodeID{judgetest.Seedbox},
		},
		{
			name: "a tester claiming no observed time isn't read", shape: judgetest.Open, verdict: judge.Reachable,
			play:  stamp(judgetest.Seedbox, time.Time{}),
			never: []role.NodeID{judgetest.Seedbox},
		},
		{
			name: "a port closed from outside fails, tested", shape: judgetest.Open, verdict: judge.Fixable, tested: true, testedBy: judgetest.Seedbox,
			play: func(w *reachabilitytest.World) {
				w.Site(judgetest.Seedbox, func(s *reachabilitytest.Site) { s.Player = judgetest.Closed })
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := c.shape()
				w := world(t, b)
				if c.play != nil {
					c.play(w)
				}
				wantPlex(t, w, b, judgetest.Atlas)
				rt := w.Route(t, "Plex")
				if !rt.Judged || rt.Result.Verdict != c.verdict || rt.Result.Tested != c.tested || rt.TestedBy != c.testedBy {
					t.Fatalf("judged %v, verdict %v, tested %v by %s; want verdict %v, tested %v by %s",
						rt.Judged, rt.Result.Verdict, rt.Result.Tested, judgetest.Names[rt.TestedBy], c.verdict, c.tested, judgetest.Names[c.testedBy])
				}
				calls := w.Calls()
				for _, n := range c.never {
					if sent(calls, n, "Dial") > 0 {
						t.Fatalf("%s was asked to connect", judgetest.Names[n])
					}
				}
			})
		})
	}
}

func TestTheUIAndPortalIsProvedByItsEcho(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.OpenUI()
		w := world(t, b)
		address(t, w, b.Facts().Exposure.Public.String())
		if _, err := w.Reach.Want(t.Context(), w.Admin, reachability.Wanted{Kind: judge.UIPortal}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		rt := w.Route(t, "SlipStream UI and portal")
		if !rt.Judged || !rt.Result.Folded() || rt.TestedBy != judgetest.Seedbox {
			t.Fatalf("verdict %v, tested %v by %s, facts %v; want Reachable, tested by seedbox", rt.Result.Verdict, rt.Result.Tested, judgetest.Names[rt.TestedBy], judgetest.Shown(rt.Result))
		}
	})
}

func TestNothingWantedSendsNothing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := world(t, judgetest.Open().Inside())
		plex(t, w, "Plex", judgetest.Atlas)
		synctest.Wait()
		w.Page(t)
		if calls := w.Calls(); len(calls) > 0 {
			t.Fatalf("sent %v", calls)
		}
		for _, n := range []role.NodeID{judgetest.Atlas, judgetest.Den, judgetest.Seedbox} {
			if w.Intent(t, n).GetWatch() {
				t.Fatalf("%s watches", judgetest.Names[n])
			}
		}
	})
}

// A Worker's Read answer that names another Node changes only its own
// row (T2).
func TestAWorkerChangesOnlyItsOwnRow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.Open().Inside()
		w := world(t, b)
		w.Site(judgetest.Seedbox, func(s *reachabilitytest.Site) { s.Claims = judgetest.Den })
		wantPlex(t, w, b, judgetest.Atlas)
		w.Away(judgetest.Den, true)
		w.Away(judgetest.Seedbox, true)
		for n, local := range map[role.NodeID]netip.Addr{judgetest.Den: netip.MustParseAddr("192.168.1.30"), judgetest.Seedbox: judgetest.SeedboxIP} {
			nn, err := w.Reach.NodeNetwork(t.Context(), w.Admin, n)
			if err != nil {
				t.Fatal(err)
			}
			if nn.Now || nn.Facts.Egress.Local != local {
				t.Fatalf("%s's stored egress is %v, want %v", judgetest.Names[n], nn.Facts.Egress.Local, local)
			}
		}
	})
}

func TestAnOfflineOriginIsCheckedOnItsReturn(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.Open().Inside()
		w := world(t, b)
		w.Away(judgetest.Den, true)
		wantPlex(t, w, b, judgetest.Den)
		if rt := w.Route(t, "Plex"); rt.Judged {
			t.Fatal("judged while its origin is away")
		}
		w.Away(judgetest.Den, false)
		synctest.Wait()
		if rt := w.Route(t, "Plex"); !rt.Judged || rt.Origin != judgetest.Den {
			t.Fatalf("judged %v from %s, want judged from den", rt.Judged, judgetest.Names[rt.Origin])
		}
		w.Away(judgetest.Den, true)
		if rt := w.Route(t, "Plex"); !rt.Away {
			t.Fatal("the route doesn't show its origin away")
		}
	})
}

// A confirmed network change reruns the Node's routes at once; another
// inside the watch interval waits for its end and shows address-churn.
func TestANetworkChangeRerunsOncePerWatchInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.Open()
		w := world(t, b)
		wantPlex(t, w, b, judgetest.Atlas)
		w.Calls()
		w.Reach.Noted(judgetest.Atlas, string(role.Network))
		synctest.Wait()
		if n := sent(w.Calls(), judgetest.Atlas, "Read"); n != 1 {
			t.Fatalf("the first change read atlas %d times, want 1", n)
		}
		w.Reach.Noted(judgetest.Atlas, string(role.Network))
		w.Reach.Noted(judgetest.Atlas, string(role.Network))
		synctest.Wait()
		if n := sent(w.Calls(), judgetest.Atlas, "Read"); n != 0 {
			t.Fatalf("a change inside the interval read atlas %d times, want 0", n)
		}
		time.Sleep(reachability.WatchEvery.Default)
		synctest.Wait()
		if n := sent(w.Calls(), judgetest.Atlas, "Read"); n != 1 {
			t.Fatalf("the interval's end read atlas %d times, want 1", n)
		}
		if shown := judgetest.Shown(w.Route(t, "Plex").Result); !slices.Contains(shown, judge.FactAddressChurn) {
			t.Fatalf("shown %v, want %s", shown, judge.FactAddressChurn)
		}
	})
}

// After a Worker restart its facts carry no previous address, and the
// Controller's stored one still finds an A that names the old one.
func TestAStaleAOutlivesAWorkerRestart(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.Open()
		w := world(t, b)
		wantPlex(t, w, b, judgetest.Atlas)
		moved := netip.MustParseAddr("203.0.113.50")
		w.Site(judgetest.Atlas, func(s *reachabilitytest.Site) {
			s.Facts.Interfaces[0].Addrs[0].Prefix = netip.PrefixFrom(moved, 24)
			s.Facts.Egress.Local = moved
			for i := range s.Facts.STUN {
				s.Facts.STUN[i].Mapped = netip.AddrPortFrom(moved, s.Facts.STUN[i].Mapped.Port())
			}
			s.Facts.Previous4 = netip.Addr{}
		})
		runNow(t, w)
		if fix := w.Route(t, "Plex").Result.Fix(); fix == nil || fix.ID != judge.FixStaleA || fix.Params.Addr != judgetest.HomeWAN {
			t.Fatalf("fix %+v, want %s naming %v", fix, judge.FixStaleA, judgetest.HomeWAN)
		}
	})
}

func items(t *testing.T, w *reachabilitytest.World) health.Answer {
	t.Helper()
	a, err := w.Reach.HealthSource().Answer(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// Health warns once a route has failed for Warn in Health after, holds it
// unsure while the origin is away, and clears on a pass and on removal.
func TestHealthFollowsAFailingRoute(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.Open().Inside()
		w := world(t, b)
		closed := func(s *reachabilitytest.Site) { s.Player = judgetest.Closed }
		w.Site(judgetest.Seedbox, closed)
		e := wantPlex(t, w, b, judgetest.Den)
		failing := time.Now()
		if a := items(t, w); len(a.Items) != 0 || !a.Next.Equal(failing.Add(reachability.WarnAfter.Default)) {
			t.Fatalf("items %v due %v, want none until %v", a.Items, a.Next, failing.Add(reachability.WarnAfter.Default))
		}
		time.Sleep(reachability.WarnAfter.Default)
		if a := items(t, w); len(a.Items) != 1 || a.Items[0].Level != health.Warn {
			t.Fatalf("items %v, want one warning", a.Items)
		}
		w.Away(judgetest.Den, true)
		if a := items(t, w); len(a.Items) != 0 || len(a.Unsure) != 1 {
			t.Fatalf("items %v unsure %v, want one unsure while den is away", a.Items, a.Unsure)
		}
		w.Away(judgetest.Den, false)
		w.Site(judgetest.Seedbox, func(s *reachabilitytest.Site) { s.Player = judgetest.Passed })
		runNow(t, w)
		if a := items(t, w); len(a.Items)+len(a.Unsure) != 0 || !a.Next.IsZero() {
			t.Fatalf("items %v unsure %v due %v after a pass, want none", a.Items, a.Unsure, a.Next)
		}
		w.Site(judgetest.Seedbox, closed)
		runNow(t, w)
		time.Sleep(reachability.WarnAfter.Default)
		if err := w.Reach.Unwant(t.Context(), w.Admin, e.ID); err != nil {
			t.Fatal(err)
		}
		if a := items(t, w); len(a.Items) != 0 {
			t.Fatalf("items %v after the Exposure's removal, want none", a.Items)
		}
	})
}

func TestRemovalsForgetWhatTheyTook(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		b := judgetest.Open().Inside()
		w := world(t, b)
		id := plex(t, w, "Plex", judgetest.Atlas)
		synctest.Wait()
		if _, err := w.Reach.Want(t.Context(), w.Admin, reachability.Wanted{Kind: judge.MediaServer, Server: id, Public: b.Facts().Exposure.Public.String()}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if n := count(t, w, "reachability_node"); n != 3 {
			t.Fatalf("%d Node rows, want 3", n)
		}
		removeNode(t, w, judgetest.Den)
		synctest.Wait()
		if n := count(t, w, "reachability_node WHERE node_id = ?", judgetest.Den); n != 0 {
			t.Fatal("den's row outlived its removal")
		}
		removeServer(t, w, id)
		synctest.Wait()
		if p := w.Page(t); !p.NothingWanted() || count(t, w, "reachability_result") != 0 {
			t.Fatalf("routes %v after the Media Server's removal, want none", p.Routes)
		}
		if w.Intent(t, judgetest.Atlas).GetWatch() {
			t.Fatal("atlas still watches with nothing wanted")
		}
	})
}

// Twenty Exposures over every Node of a shape are judged inside one
// unit's bound, on the real clock.
func TestManyRoutesAreJudgedInsideTheUnitBound(t *testing.T) {
	if testing.Short() {
		t.Skip("the scale test runs without -short")
	}
	b := judgetest.PairedProxy().Inside().Outside()
	w := world(t, b)
	for i := range 20 {
		id := plex(t, w, fmt.Sprintf("Plex %d", i), judgetest.Atlas)
		if _, err := w.Reach.Want(t.Context(), w.Admin, reachability.Wanted{Kind: judge.MediaServer, Server: id, Public: fmt.Sprintf("https://plex%d.example.org", i)}); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now()
	for bound := start.Add(reachability.DefaultTimings().Unit); ; {
		judged := 0
		for _, rt := range w.Page(t).Routes {
			if rt.Judged {
				judged++
			}
		}
		if judged == 20 {
			t.Logf("20 routes judged in %v", time.Since(start))
			return
		}
		if time.Now().After(bound) {
			t.Fatalf("%d of 20 routes judged after %v", judged, reachability.DefaultTimings().Unit)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// With "Open router ports automatically" on, the origin's intent maps the
// forward each Direct route needs, unless the route goes through a proxy
// or the origin only stands in for a Media Server on no Node. Turning it on checks the origin's routes again.
func TestTheIntentMapsWhatADirectRouteNeeds(t *testing.T) {
	for _, c := range []struct {
		name   string
		shape  func() *judgetest.B
		runsOn role.NodeID
		off    bool
		want   []string
	}{
		{name: "a Direct route", shape: judgetest.Mappable, runsOn: judgetest.Atlas, want: []string{"32400>32400"}},
		{name: "the setting off", shape: judgetest.Mappable, runsOn: judgetest.Atlas, off: true},
		{name: "a route through a proxy", shape: judgetest.PairedProxy, runsOn: judgetest.Atlas},
		{name: "a Media Server on no Node", shape: judgetest.Mappable},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				b := c.shape()
				w := world(t, b)
				wantPlex(t, w, b, c.runsOn)
				w.Calls()
				on := map[bool]string{false: "true", true: "false"}[c.off]
				if err := w.Reach.NodePanel(judgetest.Atlas).Apply(t.Context(), setting.Patch{reachability.MapPorts.Key: json.RawMessage(on)}); err != nil {
					t.Fatal(err)
				}
				synctest.Wait()
				var got []string
				for _, m := range w.Intent(t, judgetest.Atlas).GetMap() {
					got = append(got, fmt.Sprintf("%d>%d", m.GetExternal(), m.GetInternal()))
				}
				if !slices.Equal(got, c.want) {
					t.Errorf("atlas maps %v, want %v", got, c.want)
				}
				if sent(w.Calls(), judgetest.Atlas, "Read") == 0 {
					t.Error("the setting's change didn't check atlas's routes again")
				}
			})
		})
	}
}
