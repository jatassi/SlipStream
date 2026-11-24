package judge_test

import (
	"errors"
	"net/netip"
	"reflect"
	"slices"
	"testing"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	jt "github.com/jatassi/slipstream/internal/reachability/judge/judgetest"
	"github.com/jatassi/slipstream/internal/role"
)

func TestJudge(t *testing.T) {
	for _, r := range jt.Rows() {
		jt.Check(t, r)
		properties(t, r.Name, r.Facts)
	}
}

func FuzzJudge(f *testing.F) {
	for _, s := range jt.Seeds() {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		properties(t, "generated", jt.Gen(b))
	})
}

var localGuesses = []judge.CheckID{judge.CheckOriginBind, judge.CheckOriginFirewall, judge.CheckOriginClass}

// properties are what holds for every Judged, whatever the facts.
func properties(t *testing.T, name string, f judge.Facts) {
	t.Helper()
	j := judge.Judge(f)
	decidingProps(t, name, j)
	checkProps(t, name, j)
	if again := judge.Judge(f); !reflect.DeepEqual(again, j) {
		t.Errorf("%s: Judge isn't stable", name)
	}
	if pl := judge.Place(f); !reflect.DeepEqual(pl, j.Placement) {
		t.Errorf("%s: Place = %+v; Judge placed %+v", name, pl, j.Placement)
	}
	b, err := judge.Encode(j)
	if err != nil {
		t.Fatalf("%s: Encode: %v", name, err)
	}
	if back, err := judge.Decode(b); err != nil || !reflect.DeepEqual(back, j) {
		t.Errorf("%s: Decode(Encode(j)) = %+v, %v", name, back, err)
	}
}

func decidingProps(t *testing.T, name string, j judge.Judged) {
	t.Helper()
	if (j.Deciding == nil) != (j.Verdict == judge.Reachable && j.Tested) {
		t.Errorf("%s: deciding %v with %v tested=%v", name, j.Deciding, j.Verdict, j.Tested)
	}
	if j.Deciding != nil {
		c := j.Check(*j.Deciding)
		if c.ID == "" || c.Points != j.Verdict || c.Fix == nil || j.Fix() != c.Fix {
			t.Errorf("%s: deciding %s points %v with fix %v; the verdict is %v", name, c.ID, c.Points, c.Fix, j.Verdict)
		}
	}
	if j.Failing() && (j.Check(*j.Deciding).Result != judge.Fails || j.Card() != judge.Warn && j.Card() != judge.Fail) {
		t.Errorf("%s: failing with card %v", name, j.Card())
	}
}

func checkProps(t *testing.T, name string, j judge.Judged) {
	t.Helper()
	for _, h := range j.Hops {
		for _, c := range h.Checks {
			if c.Family == nil {
				continue
			}
			if *c.Family == netmodel.IPv6 && c.Points > judge.Fixable {
				t.Errorf("%s: IPv6 check %s points %v", name, c.ID, c.Points)
			}
			if slices.Contains(localGuesses, c.ID) && !c.Fact && proven(j, *c.Family) {
				t.Errorf("%s: %s of a proven family isn't a fact", name, c.ID)
			}
		}
	}
}

// proven is a family tested from outside with none of the tester's checks failing.
func proven(j judge.Judged, fam netmodel.Family) bool {
	i := slices.IndexFunc(j.Families, func(f judge.FamilyVerdict) bool { return f.Family == fam })
	if i < 0 || !j.Families[i].Tested {
		return false
	}
	for _, h := range j.Hops {
		for _, c := range h.Checks {
			tester := c.ID == judge.CheckOutsideConnect || c.ID == judge.CheckTLSCertificate || c.ID == judge.CheckIdentity
			if tester && c.Result == judge.Fails && c.Family != nil && *c.Family == fam {
				return false
			}
		}
	}
	return true
}

func TestDecodeOtherVersion(t *testing.T) {
	if _, err := judge.Decode([]byte(`{"V":0,"Verdict":2}`)); !errors.Is(err, judge.ErrOtherVersion) {
		t.Errorf("Decode of version 0 = %v; want ErrOtherVersion", err)
	}
}

func TestTesters(t *testing.T) {
	for _, c := range []struct {
		name  string
		b     *jt.B
		fam   netmodel.Family
		want  []role.NodeID
		facts []judge.FactID
	}{
		{"an outside Worker tests a Direct route", jt.Open(), netmodel.IPv4, []role.NodeID{jt.Seedbox}, nil},
		{"an inside Worker is refused", jt.Open().Inside(), netmodel.IPv4, []role.NodeID{jt.Seedbox},
			[]judge.FactID{judge.FactSameNetwork}},
		{"an offline Worker is skipped", jt.Open().Away(jt.Seedbox), netmodel.IPv4, nil,
			[]judge.FactID{judge.FactTesterAway}},
		{"a Worker that serves the Exposure is skipped", jt.Open().Serves(jt.Seedbox), netmodel.IPv4, nil,
			[]judge.FactID{judge.FactServesIt}},
		{"a link that contradicts the claim", jt.Open().LinkFrom(jt.Seedbox, "198.51.100.20"), netmodel.IPv4, nil,
			[]judge.FactID{judge.FactClaimContradicted}},
		{"a link over a private network leaves the claim unconfirmed", jt.Open().LinkFrom(jt.Seedbox, "100.101.7.3"), netmodel.IPv4,
			[]role.NodeID{jt.Seedbox}, []judge.FactID{judge.FactLinkUnconfirmed}},
		{"an IPv4 link leaves an IPv6 claim unconfirmed", jt.IPv6(), netmodel.IPv6,
			[]role.NodeID{jt.Seedbox}, []judge.FactID{judge.FactLinkUnconfirmed}},
		{"an IPv6 link outside the claimed /64", jt.IPv6().LinkFrom(jt.Seedbox, "2001:db8:7::1"), netmodel.IPv6, nil,
			[]judge.FactID{judge.FactClaimContradicted}},
		{"the origin tests its own-proxy route", jt.OwnProxy().Outside(), netmodel.IPv4, []role.NodeID{jt.Atlas, jt.Seedbox}, nil},
		{"the proxy at the dialed address never tests", jt.PairedProxy().Outside(), netmodel.IPv4,
			[]role.NodeID{jt.Atlas, jt.Seedbox}, []judge.FactID{judge.FactAtAddress}},
		{"a tester whose DNS answers a private address", jt.Open().ResolvesAt(jt.Seedbox, "192.168.1.20"), netmodel.IPv4, nil,
			[]judge.FactID{judge.FactTesterDNS}},
		{"no tester on an Unplaced route", jt.SingleMachine().Resolves("192.168.1.20").Outside(), netmodel.IPv4, nil, nil},
		{"IPv6 needs a Global IPv6 address", jt.IPv6().Outside().Inside(), netmodel.IPv6,
			[]role.NodeID{jt.Seedbox}, []judge.FactID{judge.FactLinkUnconfirmed}},
	} {
		f := c.b.Facts()
		got, facts := judge.Testers(f, judge.Place(f), c.fam)
		var ids []judge.FactID
		for _, x := range facts {
			ids = append(ids, x.ID)
		}
		if !slices.Equal(got, c.want) || !slices.Equal(ids, c.facts) {
			t.Errorf("%s: Testers = %v %v; want %v %v", c.name, got, ids, c.want, c.facts)
		}
	}
}

func TestVet(t *testing.T) {
	f := jt.IPv6().Facts()
	pl := judge.Place(f)
	ap := netip.MustParseAddrPort
	for _, c := range []struct {
		target string
		from   judge.TargetSource
		want   bool
	}{
		{"203.0.113.9:32400", judge.FromPublic4, true},
		{"192.168.1.20:32400", judge.FromPublic4, false},
		{"192.0.2.44:32400", judge.FromPublic4, false},
		{"192.0.2.45:32400", judge.FromPublic4, false},
		{"203.0.113.9:22", judge.FromPublic4, false},
		{"[2001:db8:1::99]:32400", judge.FromPublished, true},
		{"[2001:db8:5::1]:32400", judge.FromPublished, false},
		{"[fd72::20]:32400", judge.FromPublished, false},
		{"[2001:db8:9::1]:32400", judge.FromPublished, false},
	} {
		if got := judge.Vet(f, pl, jt.Seedbox, ap(c.target), c.from); got != c.want {
			t.Errorf("Vet(%s) = %v; want %v", c.target, got, c.want)
		}
	}
}

func TestAcceptEcho(t *testing.T) {
	for _, c := range []struct {
		name string
		b    *jt.B
		echo jt.Arrival
		ok   bool
		fact judge.FactID
	}{
		{"from the tester's public address", jt.OpenUI(), jt.Outside, true, ""},
		{"through Caddy on the origin", jt.OpenUI().Caddy(), jt.ThroughCaddy, true, judge.FactEchoThroughProgram},
		{"over loopback with no program in front", jt.OpenUI(), jt.ThroughCaddy, false, judge.FactEchoSource},
		{"through the tester's own Serve stream", jt.OpenUI(), jt.OwnStream, false, judge.FactEchoOwnStream},
		{"from a loopback on the tester", jt.OpenUI(), jt.TesterLoopback, false, judge.FactEchoTesterAddress},
		{"from another address on a Direct route", jt.OpenUI(), jt.OtherSource, false, judge.FactEchoSource},
		{"from another address through the Admin's proxy", jt.OwnProxy().Outside().UI("https://slipstream.proxy.fake.localhost"), jt.OtherSource, true, ""},
		{"nothing arrived", jt.OpenUI(), jt.Dropped, false, ""},
	} {
		f := c.b.Tested(netmodel.IPv4, jt.Passed).Echo(c.echo).Facts()
		pl := judge.Place(f)
		ok, fact := judge.AcceptEcho(f, pl, f.Dials[0])
		if ok != c.ok || fact.ID != c.fact {
			t.Errorf("%s: AcceptEcho = %v %s; want %v %s", c.name, ok, fact.ID, c.ok, c.fact)
		}
	}
}
