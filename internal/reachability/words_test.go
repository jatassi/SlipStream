package reachability

import (
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	jt "github.com/jatassi/slipstream/internal/reachability/judge/judgetest"
	"github.com/jatassi/slipstream/internal/role"
)

var words = Words{Node: func(n role.NodeID) string { return jt.Names[n] }}

// TestWordsRegistry is the registry test: every id has words, every Params
// field a template reads is declared, and the judge sets every declared
// field it needs, over every row of the verdict table and the generator's
// seeds.
func TestWordsRegistry(t *testing.T) {
	everyIDHasWords(t)
	seenFix, seenFact := judgedWords(t)
	for _, id := range judge.FixIDs {
		if !seenFix[id] {
			t.Errorf("no row emits fix %s, so its Params go unchecked", id)
		}
		x := judge.Fix{ID: id, Params: full, Command: true}
		checkText(t, string(id), words.Title(x)+" "+words.Step(x))
	}
	for _, id := range judge.FactIDs {
		if !seenFact[id] {
			t.Errorf("no row emits fact %s, so its Params go unchecked", id)
		}
		checkText(t, string(id), words.FactText(judge.Fact{ID: id, Params: full}))
	}
}

func everyIDHasWords(t *testing.T) {
	t.Helper()
	for _, id := range judge.CheckIDs {
		if meanings[id] == "" {
			t.Errorf("check %s has no meaning", id)
		}
	}
	for _, id := range judge.FixIDs {
		if s, ok := fixes[id]; !ok || s.title == nil || s.step == nil {
			t.Errorf("fix %s has no title or step", id)
		}
	}
	for _, id := range judge.FactIDs {
		if s, ok := facts[id]; !ok || s.title == nil {
			t.Errorf("fact %s has no words", id)
		}
	}
	for id := range fixes {
		if !id.Valid() {
			t.Errorf("words for unknown fix %s", id)
		}
	}
	for id := range facts {
		if !id.Valid() {
			t.Errorf("words for unknown fact %s", id)
		}
	}
}

// judgedWords renders every fix and fact the judge emits over the verdict
// table and the generator's seeds, and returns the ids it saw.
func judgedWords(t *testing.T) (seenFix map[judge.FixID]bool, seenFact map[judge.FactID]bool) {
	t.Helper()
	var all []judge.Facts
	for _, r := range jt.Rows() {
		all = append(all, r.Facts)
	}
	for _, s := range jt.Seeds() {
		all = append(all, jt.Gen(s))
	}
	seenFix, seenFact = map[judge.FixID]bool{}, map[judge.FactID]bool{}
	for _, f := range all {
		fxs, fcs := emitted(judge.Judge(f))
		for _, fx := range fxs {
			seenFix[fx.ID] = true
			checkSentence(t, string(fx.ID), fixes[fx.ID], fx.Params, func(p judge.Params) string {
				x := fx
				x.Params = p
				return words.Title(x) + " " + words.Step(x) + " " + Command(x)
			})
		}
		for _, fc := range fcs {
			seenFact[fc.ID] = true
			checkSentence(t, string(fc.ID), facts[fc.ID], fc.Params, func(p judge.Params) string {
				return words.FactText(judge.Fact{ID: fc.ID, Params: p})
			})
		}
	}
	return seenFix, seenFact
}

var fam6 = netmodel.IPv6

var full = judge.Params{
	Node: jt.Atlas, Tester: jt.Seedbox, Host: "plex.example.org", Port: 32400, LocalPort: 32400,
	Addr: netip.MustParseAddr("203.0.113.9"), LocalAddr: jt.AtlasLAN, Want: netip.MustParseAddr("203.0.113.10"),
	Prefix: netip.MustParsePrefix("2001:db8:2::/64"), Date: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC),
	Tool: netmodel.UFW, Class: netmodel.Mappable, Program: "caddy", Family: &fam6, URL: "https://plex.example.org",
	Raw: "expired", Reason: judge.ReasonPrivate,
}

func emitted(j judge.Judged) ([]judge.Fix, []judge.Fact) {
	var fxs []judge.Fix
	fcs := slices.Concat(j.Notes, j.Facts)
	for _, h := range j.Hops {
		fcs = append(fcs, h.Facts...)
		for _, c := range h.Checks {
			if c.Fix != nil {
				fxs = append(fxs, *c.Fix)
			}
			if c.Found != nil {
				fcs = append(fcs, *c.Found)
			}
		}
	}
	return fxs, fcs
}

func checkSentence(t *testing.T, id string, s sentence, p judge.Params, render func(judge.Params) string) {
	t.Helper()
	for _, f := range s.needs {
		if !set(p, f) {
			t.Errorf("%s: the judge left Params field %d unset: %+v", id, f, p)
		}
	}
	text := render(p)
	checkText(t, id, text)
	if declared := render(only(p, slices.Concat(s.needs, s.may))); declared != text {
		t.Errorf("%s reads a Params field it doesn't declare:\n %s\n %s", id, text, declared)
	}
}

func checkText(t *testing.T, id, text string) {
	t.Helper()
	for _, bad := range []string{"%!", "invalid IP", "invalid Prefix", "<nil>", "  ", " ."} {
		if strings.Contains(text, bad) {
			t.Errorf("%s renders %q", id, text)
		}
	}
}

func set(p judge.Params, f param) bool {
	return only(p, []param{f}) != judge.Params{}
}

func only(p judge.Params, fs []param) judge.Params {
	var o judge.Params
	for _, f := range fs {
		switch f {
		case pNode:
			o.Node = p.Node
		case pTester:
			o.Tester = p.Tester
		case pHost:
			o.Host = p.Host
		case pPort:
			o.Port = p.Port
		case pLocalPort:
			o.LocalPort = p.LocalPort
		case pAddr:
			o.Addr = p.Addr
		case pLocalAddr:
			o.LocalAddr = p.LocalAddr
		case pWant:
			o.Want = p.Want
		case pPrefix:
			o.Prefix = p.Prefix
		case pDate:
			o.Date = p.Date
		case pTool:
			o.Tool = p.Tool
		case pClass:
			o.Class = p.Class
		case pProgram:
			o.Program = p.Program
		case pFamily:
			o.Family = p.Family
		case pURL:
			o.URL = p.URL
		case pRaw:
			o.Raw = p.Raw
		case pReason:
			o.Reason = p.Reason
		}
	}
	return o
}

func TestStatusParts(t *testing.T) {
	a, b, c := judge.ExposureID{UUID: jt.Atlas.UUID}, judge.ExposureID{UUID: jt.Den.UUID}, judge.ExposureID{UUID: jt.Seedbox.UUID}
	type part struct {
		tone Tone
		on   *judge.ExposureID
	}
	for _, x := range []struct {
		name string
		s    judge.Status
		want []part
	}{
		{"everything works", judge.Status{Work: []judge.ExposureID{a, b}}, []part{{ToneOK, nil}}},
		{"nothing tested yet", judge.Status{LookRight: []judge.ExposureID{a, b}}, []part{{ToneNeutral, nil}, {ToneNeutral, nil}}},
		{"one of each", judge.Status{Work: []judge.ExposureID{a}, LookRight: []judge.ExposureID{b}, Fix: []judge.ExposureID{c}},
			[]part{{ToneOK, nil}, {ToneNeutral, nil}, {ToneWarn, &c}}},
		{"two relays", judge.Status{Untested: []judge.ExposureID{a}, Relay: []judge.ExposureID{b, c}},
			[]part{{ToneNeutral, nil}, {ToneFail, nil}}},
	} {
		got := StatusParts(x.s, func(judge.ExposureID) string { return "Plex" })
		var parts []part
		for _, p := range got {
			parts = append(parts, part{p.Tone, p.Exposure})
			checkText(t, x.name, p.Text)
		}
		if len(parts) != len(x.want) {
			t.Errorf("%s: %d parts; want %d", x.name, len(parts), len(x.want))
			continue
		}
		for i := range parts {
			if parts[i].tone != x.want[i].tone || (parts[i].on == nil) != (x.want[i].on == nil) || parts[i].on != nil && *parts[i].on != *x.want[i].on {
				t.Errorf("%s: part %d = %v; want %v", x.name, i, parts[i], x.want[i])
			}
		}
	}
}
