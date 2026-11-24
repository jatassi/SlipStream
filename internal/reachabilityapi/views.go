package reachabilityapi

import (
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/netmodel"
	"github.com/jatassi/slipstream/internal/reachability"
	"github.com/jatassi/slipstream/internal/reachability/judge"
	"github.com/jatassi/slipstream/internal/role"
)

// checkID, fixID and factID are the judge's closed sets, so the web's
// unions of them follow judge.CheckIDs, FixIDs and FactIDs.
type (
	checkID struct{}
	fixID   struct{}
	factID  struct{}
)

func (checkID) Values() []string { return strings(judge.CheckIDs) }
func (fixID) Values() []string   { return strings(judge.FixIDs) }
func (factID) Values() []string  { return strings(judge.FactIDs) }

func strings[S ~string](ids []S) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = string(id)
	}
	return out
}

type remoteAccess struct {
	NothingWanted bool                  `json:"nothingWanted" doc:"Nothing is set up for use away from home: the page shows its Empty state"`
	Status        status                `json:"status"`
	Card          api.Nullable[card]    `json:"card" doc:"The one recommendation, null when every route works and was tested"`
	Routes        []route               `json:"routes" nullable:"false"`
	LastRun       api.Nullable[lastRun] `json:"lastRun" doc:"Null before the first check"`
	Suggestions   []suggestionView      `json:"suggestions" nullable:"false"`
	Nodes         []classLine           `json:"nodes" nullable:"false" doc:"One line per Node, the Controller first; empty while nothing is set up"`
}

type status struct {
	Counts counts       `json:"counts"`
	Parts  []statusPart `json:"parts" nullable:"false" doc:"The status sentence's parts, joined with · on screen"`
}

type counts struct {
	Work      int `json:"work" doc:"Reachable and tested from outside"`
	LookRight int `json:"lookRight" doc:"Reachable by what SlipStream sees here, not tested from outside"`
	Untested  int `json:"untested" doc:"Fixable, resting on Not tested"`
	Fix       int `json:"fix" doc:"Fixable, resting on a failing check"`
	Relay     int `json:"relay" doc:"Needs a relay"`
}

type statusPart struct {
	Tone       string `json:"tone" enum:"ok,neutral,warn,fail"`
	Text       string `json:"text"`
	ExposureID string `json:"exposureId,omitempty" format:"uuid" doc:"The route the part is about, which it links to"`
}

type card struct {
	Kind       string  `json:"kind" enum:"untested,warn,fail"`
	ExposureID string  `json:"exposureId" format:"uuid"`
	Subject    string  `json:"subject"`
	Verdict    string  `json:"verdict" enum:"reachable,fixable,needs-relay"`
	Fix        fixView `json:"fix"`
}

type exposureView struct {
	ID      string `json:"id" format:"uuid"`
	Kind    string `json:"kind" enum:"ui-portal,media-server"`
	Subject string `json:"subject" doc:"SlipStream UI and portal, or the Media Server's name"`
	Public  string `json:"public" doc:"The address players outside the home use; the UI and portal's is Address"`
}

type route struct {
	Exposure exposureView          `json:"exposure"`
	Checked  bool                  `json:"checked" doc:"False until the first check of this route finished"`
	Route    string                `json:"route,omitempty" enum:"direct,own-proxy,through-relay,unplaced"`
	Families string                `json:"families,omitempty" doc:"The families the route offers, as its tag"`
	Verdict  string                `json:"verdict,omitempty" enum:"reachable,fixable,needs-relay"`
	Tested   bool                  `json:"tested" doc:"Every offered family was tested from outside"`
	Folded   bool                  `json:"folded" doc:"Reachable and tested: the route shows as one line"`
	Card     string                `json:"card" enum:"none,untested,warn,fail"`
	Fix      api.Nullable[fixView] `json:"fix" doc:"The step that fixes the deciding check, null with none"`
	Skipped  *string               `json:"skipped" doc:"Why the route wasn't checked since its last result, null while the Node that serves it is online"`
	TestedBy string                `json:"testedBy,omitempty"`
	JudgedAt *time.Time            `json:"judgedAt"`
	Hops     []hopView             `json:"hops" nullable:"false"`
	Facts    []factView            `json:"facts" nullable:"false"`
}

type fixView struct {
	ID      api.Enum[fixID]          `json:"id"`
	Title   string                   `json:"title"`
	Step    string                   `json:"step"`
	Command string                   `json:"command,omitempty" doc:"The firewall command, when the fix carries one"`
	Details string                   `json:"details,omitempty"`
	Action  api.Nullable[actionView] `json:"action" doc:"The fix's button, null with none"`
}

type actionView struct {
	Kind   string `json:"kind" enum:"pair-node,map-ports"`
	NodeID string `json:"nodeId,omitempty" format:"uuid"`
	Label  string `json:"label"`
}

type hopView struct {
	Kind   string      `json:"kind" enum:"player,dns,proxy,relay,tls,node-link,origin,media-server"`
	Worst  string      `json:"worst" enum:"passes,fails,not-tested,not-used,not-reached"`
	Checks []checkView `json:"checks" nullable:"false"`
	Facts  []factView  `json:"facts" nullable:"false"`
}

type checkView struct {
	ID       api.Enum[checkID]     `json:"id"`
	Family   string                `json:"family,omitempty" enum:"ipv4,ipv6"`
	Result   string                `json:"result" enum:"passes,fails,not-tested,not-used,not-reached"`
	Points   string                `json:"points,omitempty" enum:"reachable,fixable,needs-relay" doc:"What a failing or untested result points to"`
	Fact     bool                  `json:"fact" doc:"Shown, never pointing: an overruled local guess or a connect-back"`
	Meaning  string                `json:"meaning"`
	Fix      api.Nullable[fixView] `json:"fix"`
	Observed string                `json:"observed,omitempty"`
	By       string                `json:"by,omitempty" doc:"The Node that saw it"`
}

type factView struct {
	ID   api.Enum[factID] `json:"id"`
	Text string           `json:"text"`
}

type lastRun struct {
	At      time.Time  `json:"at"`
	Testers []nodeName `json:"testers" nullable:"false"`
}

type nodeName struct {
	NodeID string `json:"nodeId" format:"uuid"`
	Name   string `json:"name"`
}

type suggestionView struct {
	Kind           string `json:"kind" enum:"ui-portal,media-server"`
	ServerID       string `json:"serverId,omitempty" format:"uuid"`
	Subject        string `json:"subject"`
	Public         string `json:"public" doc:"The prefilled public address, empty when none is known"`
	AddressPrivate bool   `json:"addressPrivate" doc:"The UI and portal has no Address outsiders can use: set one first"`
}

type classLine struct {
	NodeID string `json:"nodeId" format:"uuid"`
	Name   string `json:"name"`
	Line   string `json:"line"`
}

func exposureViewOf(e reachability.Exposure, subject string) exposureView {
	v := exposureView{ID: e.ID.String(), Kind: reachability.KindName(e.Kind), Subject: subject}
	if e.Public != nil {
		v.Public = e.Public.String()
	}
	return v
}

func family(f *netmodel.Family) string {
	if f == nil {
		return ""
	}
	return map[netmodel.Family]string{netmodel.IPv4: "ipv4", netmodel.IPv6: "ipv6"}[*f]
}

func fixViewOf(w reachability.Words, f judge.Fix) fixView {
	v := fixView{ID: api.NewEnum[fixID](string(f.ID)), Title: w.Title(f), Step: w.Step(f), Command: reachability.Command(f), Details: w.Details(f)}
	if a := f.Action; a != nil {
		action := actionView{Kind: map[judge.ActionKind]string{judge.PairNode: "pair-node", judge.MapPorts: "map-ports"}[a.Kind], Label: reachability.ActionLabel(a.Kind)}
		if !a.Node.IsZero() {
			action.NodeID = a.Node.String()
		}
		v.Action = api.Some(action)
	}
	return v
}

func factViews(w reachability.Words, fs []judge.Fact) []factView {
	out := make([]factView, len(fs))
	for i, f := range fs {
		out[i] = factView{ID: api.NewEnum[factID](string(f.ID)), Text: w.FactText(f)}
	}
	return out
}

func routeOf(p reachability.Page, rt reachability.Route) route {
	w := p.Words
	v := route{Exposure: exposureViewOf(rt.Exposure, rt.Subject), Checked: rt.Judged, Card: judge.NoCard.String(), Hops: []hopView{}, Facts: []factView{}}
	v.Exposure.Public = rt.Public
	if rt.Away {
		v.Skipped = new(w.Skipped(rt.Origin, rt.JudgedAt.In(p.Zone).Format("15:04")))
	}
	if !rt.Judged {
		return v
	}
	j := rt.Result
	v.JudgedAt = &rt.JudgedAt
	v.Route = j.Routes[netmodel.IPv4].Kind.String()
	v.Families, v.Verdict, v.Tested, v.Folded, v.Card = reachability.Families(j.Offered), j.Verdict.String(), j.Tested, j.Folded(), j.Card().String()
	v.TestedBy = w.TestedBy(j, rt.TestedBy)
	if f := j.Fix(); f != nil {
		v.Fix = api.Some(fixViewOf(w, *f))
	}
	v.Facts = factViews(w, j.Facts)
	for _, h := range j.Hops {
		hv := hopView{Kind: h.Kind.String(), Worst: h.Worst.String(), Checks: []checkView{}, Facts: factViews(w, h.Facts)}
		for _, c := range h.Checks {
			cv := checkView{ID: api.NewEnum[checkID](string(c.ID)), Family: family(c.Family), Result: c.Result.String(), Fact: c.Fact, Meaning: reachability.CheckMeaning(c.ID), Observed: c.Observed.Text}
			if !c.Fact && (c.Result == judge.Fails || c.Result == judge.NotTested) {
				cv.Points = c.Points.String()
			}
			if c.Fix != nil {
				cv.Fix = api.Some(fixViewOf(w, *c.Fix))
			}
			if !c.By.IsZero() {
				cv.By = w.Node(c.By)
			}
			hv.Checks = append(hv.Checks, cv)
		}
		v.Hops = append(v.Hops, hv)
	}
	return v
}

func pageOf(p reachability.Page) remoteAccess {
	w := p.Words
	out := remoteAccess{NothingWanted: p.NothingWanted(), Routes: []route{}, Suggestions: []suggestionView{}, Nodes: []classLine{}}
	s := p.Status
	out.Status.Counts = counts{Work: len(s.Work), LookRight: len(s.LookRight), Untested: len(s.Untested), Fix: len(s.Fix), Relay: len(s.Relay)}
	subjects := map[judge.ExposureID]string{}
	for _, rt := range p.Routes {
		subjects[rt.Exposure.ID] = rt.Subject
		out.Routes = append(out.Routes, routeOf(p, rt))
	}
	out.Status.Parts = []statusPart{}
	for _, part := range reachability.StatusParts(s, func(id judge.ExposureID) string { return subjects[id] }) {
		sp := statusPart{Tone: string(part.Tone), Text: part.Text}
		if part.Exposure != nil {
			sp.ExposureID = part.Exposure.String()
		}
		out.Status.Parts = append(out.Status.Parts, sp)
	}
	out.Card = cardOf(p)
	if !p.LastRun.IsZero() {
		run := lastRun{At: p.LastRun, Testers: []nodeName{}}
		for _, t := range p.Testers {
			run.Testers = append(run.Testers, nodeName{NodeID: t.String(), Name: w.Node(t)})
		}
		out.LastRun = api.Some(run)
	}
	for _, sg := range p.Suggestions {
		v := suggestionView{Kind: reachability.KindName(sg.Kind), Subject: sg.Subject, Public: sg.Public, AddressPrivate: sg.AddressPrivate}
		if !sg.Server.IsZero() {
			v.ServerID = sg.Server.String()
		}
		out.Suggestions = append(out.Suggestions, v)
	}
	for _, n := range p.Nodes {
		out.Nodes = append(out.Nodes, classLine{NodeID: n.Node.String(), Name: n.Name, Line: lineOf(w, n)})
	}
	return out
}

func lineOf(w reachability.Words, n reachability.NodeLine) string {
	switch {
	case !n.Watched:
		return w.NotWatched(n.Node)
	case !n.Read:
		return w.NotReadYet(n.Node)
	}
	return w.ClassLine(n.Node, n.Class)
}

// cardOf is the one recommendation: a route that needs a relay, else one
// that needs a fix, else one that wasn't tested from outside.
func cardOf(p reachability.Page) api.Nullable[card] {
	for _, want := range []judge.Card{judge.Fail, judge.Warn, judge.Untested} {
		for _, rt := range p.Routes {
			if !rt.Judged || rt.Result.Card() != want {
				continue
			}
			return api.Some(card{Kind: want.String(), ExposureID: rt.Exposure.ID.String(), Subject: rt.Subject, Verdict: rt.Result.Verdict.String(), Fix: fixViewOf(p.Words, *rt.Result.Fix())})
		}
	}
	return api.Nullable[card]{}
}

type nodeNetwork struct {
	NodeID     string                    `json:"nodeId" format:"uuid"`
	Name       string                    `json:"name"`
	Line       string                    `json:"line" doc:"The Node's class, in no signal color"`
	Now        bool                      `json:"now" doc:"Read on opening; false is the last stored look, for a Node that is away"`
	ObservedAt *time.Time                `json:"observedAt"`
	Interfaces []ifaceView               `json:"interfaces" nullable:"false"`
	Listeners  []listenView              `json:"listeners" nullable:"false"`
	Firewall   []ruleView                `json:"firewall" nullable:"false"`
	Gateway    api.Nullable[gatewayView] `json:"gateway" doc:"Null when no router was asked"`
	Mappings   []mappingView             `json:"mappings" nullable:"false"`
}

type ifaceView struct {
	Name      string   `json:"name"`
	Tunnel    bool     `json:"tunnel"`
	Addresses []string `json:"addresses" nullable:"false"`
}

type listenView struct {
	Port     int    `json:"port"`
	Program  string `json:"program" doc:"Empty when SlipStream couldn't see the owner"`
	IPv4     string `json:"ipv4" enum:"unknown,yes,no"`
	IPv6     string `json:"ipv6" enum:"unknown,yes,no"`
	Loopback bool   `json:"loopback"`
}

type ruleView struct {
	Port   int    `json:"port"`
	Family string `json:"family" enum:"ipv4,ipv6"`
	Policy string `json:"policy" enum:"unknown,allows,blocks"`
}

type gatewayView struct {
	Address string `json:"address"`
	WAN     string `json:"wan,omitempty" doc:"The address the router says it has on the internet, empty when it said none"`
}

type mappingView struct {
	External  int        `json:"external"`
	Internal  int        `json:"internal"`
	Permanent bool       `json:"permanent" doc:"The router took the mapping only with no expiry"`
	Expires   *time.Time `json:"expires"`
}

func tri(t netmodel.Tri) string { return [...]string{"unknown", "yes", "no"}[t] }

func addr(a netip.Addr) string {
	if !a.IsValid() {
		return ""
	}
	return a.String()
}

func nodeNetworkOf(n reachability.NodeNetwork) nodeNetwork {
	w := reachability.Words{Node: func(role.NodeID) string { return n.Name }}
	f := n.Facts
	out := nodeNetwork{
		NodeID: n.Node.String(), Name: n.Name, Line: lineOf(w, n.Line), Now: n.Now,
		Interfaces: []ifaceView{}, Listeners: []listenView{}, Firewall: []ruleView{}, Mappings: []mappingView{},
	}
	if !f.At.IsZero() {
		out.ObservedAt = &f.At
	}
	for _, i := range f.Interfaces {
		iv := ifaceView{Name: i.Name, Tunnel: i.Tunnel, Addresses: []string{}}
		for _, a := range i.Addrs {
			iv.Addresses = append(iv.Addresses, a.Prefix.String())
		}
		out.Interfaces = append(out.Interfaces, iv)
	}
	for _, l := range f.Listeners {
		out.Listeners = append(out.Listeners, listenView{Port: int(l.Port), Program: l.Program.Name, IPv4: tri(l.V4), IPv6: tri(l.V6), Loopback: l.Loopback})
	}
	for _, r := range f.Firewall {
		out.Firewall = append(out.Firewall, ruleView{Port: int(r.Port), Family: family(&r.Family), Policy: [...]string{"unknown", "allows", "blocks"}[r.Policy]})
	}
	if f.Gateway.Addr.IsValid() {
		out.Gateway = api.Some(gatewayView{Address: f.Gateway.Addr.String(), WAN: addr(f.Gateway.WAN)})
	}
	for _, m := range f.Mappings {
		if m.Refused != netmodel.RefusedNone {
			continue
		}
		mv := mappingView{External: int(m.External), Internal: int(m.Internal), Permanent: m.Lifetime == 0}
		if !m.Expires.IsZero() {
			mv.Expires = &m.Expires
		}
		out.Mappings = append(out.Mappings, mv)
	}
	return out
}
