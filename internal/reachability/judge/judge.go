package judge

import (
	"cmp"
	"slices"

	"github.com/jatassi/slipstream/internal/netmodel"
)

// Judge is the diagnostic: one pure function over the facts a run gathered.
func Judge(f Facts) Judged {
	pl := Place(f)
	var g judging
	var fams []FamilyVerdict
	tested := true
	for _, fam := range pl.Offered {
		fv := g.family(newFamilyIn(f, pl, fam))
		tested = tested && fv.Tested
		fams = append(fams, fv)
	}
	if in6 := newFamilyIn(f, pl, netmodel.IPv6); !slices.Contains(pl.Offered, netmodel.IPv6) && rowDNSAAAA.Applies(in6) {
		g.run(in6, rowDNSAAAA, true)
	}
	in4 := newFamilyIn(f, pl, netmodel.IPv4)
	j := Judged{V: Version, Exposure: f.Exposure.ID, Placement: pl, Tested: tested, Forward: in4.forward()}
	for _, r := range exposureRows {
		if r.Applies(in4) {
			g.run(in4, r, false)
		}
	}
	j.Verdict, j.Deciding = decide(g.checks, tested)
	for i := range fams {
		fams[i].Verdict = worst(g.checks, &fams[i].Family)
	}
	j.Families = fams
	if len(fams) == 2 && fams[1].Verdict == Reachable && fams[0].Verdict != Reachable {
		j.Facts = append(j.Facts, Fact{ID: FactIPv6ViewersDirect, Params: Params{Node: pl.Origin}})
	}
	for _, n := range pl.Notes {
		g.facts = append(g.facts, placedFact{DNSHop, n})
	}
	j.Hops = assemble(pl, g.checks, g.facts)
	return j
}

// judging collects the checks and hop facts of one Judge.
type judging struct {
	checks []Check
	facts  []placedFact
}

func (g *judging) run(in *familyIn, r row, perFamily bool) outcome {
	o := r.Eval(in)
	c := in.check(r, o, perFamily)
	g.checks = append(g.checks, c)
	for _, x := range o.Facts {
		if perFamily && x.Family == nil {
			x.Family = c.Family
		}
		g.facts = append(g.facts, placedFact{c.Hop, x})
	}
	return o
}

// family runs one family's rows. A passing outside test overrules every
// local guess of the family into a fact.
func (g *judging) family(in *familyIn) FamilyVerdict {
	start := len(g.checks)
	fv := FamilyVerdict{Family: in.fam}
	for _, r := range familyRows {
		if r.Only != nil && *r.Only != in.fam || !r.Applies(in) {
			continue
		}
		if g.run(in, r, true).byPlex {
			fv.ByPlex = true
		}
	}
	fv.Tested = fv.ByPlex || provenByTester(g.checks[start:])
	if fv.Tested && !fv.ByPlex {
		fv.Tester = in.player.Tester
	}
	if fv.Tested {
		for i := range g.checks[start:] {
			c := &g.checks[start+i]
			c.Fact = c.Fact || slices.ContainsFunc(familyRows, func(r row) bool { return r.LocalGuess && r.ID == c.ID })
		}
	}
	g.facts = append(g.facts, in.hopFacts()...)
	return fv
}

// provenByTester is the spec's proof for a family: one tester's connection,
// TLS when https, and identity all pass.
func provenByTester(cs []Check) bool {
	result := func(id CheckID) (Result, bool) {
		i := slices.IndexFunc(cs, func(c Check) bool { return c.ID == id })
		if i < 0 {
			return 0, false
		}
		return cs[i].Result, true
	}
	connect, _ := result(CheckOutsideConnect)
	identity, _ := result(CheckIdentity)
	tls, https := result(CheckTLSCertificate)
	return connect == Passes && identity == Passes && (!https || tls == Passes)
}

// open is a check that points: not a fact, and neither passing nor out of
// play.
func open(c Check) bool {
	return !c.Fact && (c.Result == Fails || c.Result == NotTested)
}

// worst is the worst Points of the open checks of a family.
func worst(cs []Check, fam *netmodel.Family) Verdict {
	v := Reachable
	for _, c := range cs {
		if open(c) && sameFamily(c.Family, fam) {
			v = max(v, c.Points)
		}
	}
	return v
}

// decide is the verdict and the deciding check in one pass, the only place
// either is chosen. The verdict is the worst Points of the open checks;
// the deciding check is the first of those with that Points, Fails before
// Not tested, then IPv4, then no family, then IPv6, then hop order, then
// table order. Reachable has a deciding check only while untested: the
// one pointing to Reachable carries the step that confirms it.
func decide(cs []Check, tested bool) (Verdict, *CheckRef) {
	var pointing []Check
	v := Reachable
	for _, c := range cs {
		if open(c) {
			pointing = append(pointing, c)
			v = max(v, c.Points)
		}
	}
	if v == Reachable && tested {
		return v, nil
	}
	slices.SortStableFunc(pointing, func(a, b Check) int {
		return cmp.Or(
			cmp.Compare(resultRank(a.Result), resultRank(b.Result)),
			cmp.Compare(familyRank(a.Family), familyRank(b.Family)),
			cmp.Compare(a.Hop, b.Hop),
		)
	})
	for _, c := range pointing {
		if c.Points == v {
			return v, &CheckRef{ID: c.ID, Family: c.Family}
		}
	}
	panic("judge: no check points to the verdict")
}

func resultRank(r Result) int {
	if r == Fails {
		return 0
	}
	return 1
}

func familyRank(f *netmodel.Family) int {
	switch {
	case f == nil:
		return 1
	case *f == netmodel.IPv4:
		return 0
	}
	return 2
}

// shownRank orders a hop's results for its worst: Fails, then Not tested,
// then Not reached, then Passes.
func shownRank(r Result) int {
	return [...]int{Passes: 1, Fails: 4, NotTested: 3, NotUsed: 0, NotReached: 2}[r]
}

// assemble lays the checks and facts on the routes' hops. A hop past a
// failing one reads Not reached, except the Origin's facts about itself and
// the Media Server's report.
func assemble(pl Placement, cs []Check, facts []placedFact) []Hop {
	var kinds []HopKind
	for _, fam := range pl.Offered {
		kinds = append(kinds, pl.Routes[fam].Hops...)
	}
	slices.Sort(kinds)
	kinds = slices.Compact(kinds)
	hops := make([]Hop, len(kinds))
	failed := false
	for i, k := range kinds {
		h := hopOf(k, cs, facts)
		if failed && h.Worst == Passes && k != OriginHop && k != MediaServerHop {
			h.Worst = NotReached
		}
		failed = failed || h.Worst == Fails && k != PlayerHop
		hops[i] = h
	}
	return hops
}

func hopOf(k HopKind, cs []Check, facts []placedFact) Hop {
	h := Hop{Kind: k, Worst: NotUsed}
	for _, c := range cs {
		if c.Hop != k {
			continue
		}
		h.Checks = append(h.Checks, c)
		r := c.Result
		if c.Fact {
			r = Passes
		}
		if shownRank(r) > shownRank(h.Worst) {
			h.Worst = r
		}
	}
	for _, f := range facts {
		if f.hop == k {
			h.Facts = append(h.Facts, f.fact)
		}
	}
	return h
}

// Status is every Exposure by the card its verdict shows, for the status
// sentence and the Nodes summary line.
type Status struct {
	Work      []ExposureID // Reachable and tested
	LookRight []ExposureID // Reachable, not tested from outside
	Untested  []ExposureID // Fixable, resting on Not tested
	Fix       []ExposureID // Fixable, resting on a failing check
	Relay     []ExposureID // Needs a relay
}

func Count(js []Judged) Status {
	var s Status
	for _, j := range js {
		switch j.Card() {
		case NoCard:
			s.Work = append(s.Work, j.Exposure)
		case Untested:
			if j.Verdict == Reachable {
				s.LookRight = append(s.LookRight, j.Exposure)
			} else {
				s.Untested = append(s.Untested, j.Exposure)
			}
		case Warn:
			s.Fix = append(s.Fix, j.Exposure)
		case Fail:
			s.Relay = append(s.Relay, j.Exposure)
		}
	}
	return s
}
