package fault

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/events"
)

// Target names what a rule faults: a fake's name, seeders or
// seeders/<release>, or link/<node>. A rule's Target covers every front
// whose target is it or lies beneath it.
type Target string

func (t Target) covers(front Target) bool {
	return front == t || strings.HasPrefix(string(front), string(t)+"/")
}

// Front is one place faults play: a target as one origin reaches it, and
// the layers its traffic carries.
type Front struct {
	Target Target
	// Origin is the dialing Node, "world" for the harness, "" where no
	// world names one.
	Origin string
	Layers Layer
}

// Rule is one fault and where and when it plays. Its JSON, the form the
// control API and the event log carry, and its YAML, the form a
// Scenario's faults block holds, write Fault as its canonical text.
type Rule struct {
	ID     string `json:"id"     jsonschema:"required" yaml:"id"`
	Target Target `json:"target" jsonschema:"required" yaml:"target"`
	// Origin is the dialing Node; "" is every origin.
	Origin string `json:"origin,omitempty" yaml:"origin,omitempty"`
	// Route is, for an HTTP fault, a route name the fake declares in
	// fake.Routes; "" is every route.
	Route string `json:"route,omitempty" yaml:"route,omitempty"`
	// Params are, for an HTTP fault, path values the request must carry.
	Params map[string]string `json:"params,omitempty" yaml:"params,omitempty"`
	Fault  Fault             `json:"fault"            jsonschema:"required" yaml:"fault"`
	// Skip is the matches let through first: the 3rd request is Skip 2,
	// Times 1.
	Skip int `json:"skip,omitempty" jsonschema:"minimum=0" yaml:"skip,omitempty"`
	// Times is the matches played; 0 plays until cleared.
	Times int `json:"times,omitempty" jsonschema:"minimum=0" yaml:"times,omitempty"`
	// After arms the rule on the first matching record observed after it
	// is set. nil arms it at once.
	After *events.Query `json:"after,omitempty" yaml:"after,omitempty"`
}

// ruleFields is Rule without the methods below, so they can encode its
// other fields the default way.
type ruleFields Rule

func (r Rule) MarshalJSON() ([]byte, error) {
	text := ""
	if r.Fault != nil {
		text = r.Fault.String()
	}
	data, err := json.Marshal(struct {
		ruleFields
		Fault string `json:"fault"`
	}{ruleFields(r), text})
	if err != nil {
		return nil, fmt.Errorf("rule %q: %w", r.ID, err)
	}
	return data, nil
}

// UnmarshalJSON refuses a field Rule lacks and a fault Parse refuses.
func (r *Rule) UnmarshalJSON(data []byte) error {
	var v struct {
		*ruleFields
		Fault string `json:"fault"`
	}
	v.ruleFields = (*ruleFields)(r)
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return fmt.Errorf("rule: %w", err)
	}
	f, err := Parse(v.Fault)
	if err != nil {
		return fmt.Errorf("rule %q: %w", r.ID, err)
	}
	r.Fault = f
	return nil
}

// Covers reports whether f is a front r faults.
func (r Rule) Covers(f Front) bool {
	return r.Target.covers(f.Target) && (r.Origin == "" || r.Origin == f.Origin)
}

// Check refuses r unless some front covers it and every front it covers
// carries its fault's layer. The world runs it when a rule is set and on
// a Scenario's rules at launch.
func Check(r Rule, fronts []Front) error {
	if r.Fault == nil {
		return fmt.Errorf("rule %q has no fault", r.ID)
	}
	layer := r.Fault.Layer()
	if layer != HTTP && (r.Route != "" || len(r.Params) > 0) {
		return fmt.Errorf("%s is %s %s fault and takes no route or params", r.Fault, layer.article(), layer)
	}
	if r.Skip < 0 || r.Times < 0 {
		return fmt.Errorf("rule %q: skip and times can't be negative", r.ID)
	}
	if _, ok := r.Fault.(Refuse); ok && (r.Skip > 0 || r.Times > 0) {
		return errors.New("refuse holds a front until cleared and takes no skip or times")
	}
	if layer == Disk {
		return checkDisk(r, fronts)
	}
	var covered []Front
	for _, f := range fronts {
		if r.Covers(f) {
			covered = append(covered, f)
		}
	}
	if len(covered) == 0 {
		return uncovered(r, fronts)
	}
	for _, f := range covered {
		if f.Layers&layer == 0 {
			return fmt.Errorf("%s is %s %s fault and %s carries no %s", r.Fault, layer.article(), layer, r.Target, layer)
		}
	}
	return nil
}

// uncovered says why no front covers r.
func uncovered(r Rule, fronts []Front) error {
	if !slices.ContainsFunc(fronts, func(f Front) bool { return r.Target.covers(f.Target) }) {
		return fmt.Errorf("no target %s", r.Target)
	}
	if !slices.ContainsFunc(fronts, func(f Front) bool { return r.Target.covers(f.Target) && f.Origin != "" }) {
		return fmt.Errorf("%s is the one front every peer dials, so a rule can't pick %s's connections; per-pair link faults stay in the simulation tier", r.Target, r.Origin)
	}
	return fmt.Errorf("%s has no front for %s", r.Target, r.Origin)
}

// Firing is one occurrence a rule played.
type Firing struct {
	Rule  string
	Front Front
	Fault Fault
	// Match counts the rule's matches, skipped ones included.
	Match int
}

// Plan is the one owner of rule state: arming, counting and firing, under
// one lock. A connection rule counts connections, taking those already
// open when it arms; a TLS rule counts handshakes; an HTTP rule counts
// requests. When two rules match one occurrence on one layer, the earlier
// set wins.
type Plan struct {
	fronts []Front
	fired  func(Firing)

	mu      sync.Mutex
	rules   []*entry
	conns   []*conn
	routes  map[Target][]Route
	changed chan struct{}
}

type entry struct {
	Rule
	armed   bool
	matched int
}

// checkRouteLocked refuses a rule whose Route no fake serving its target
// declared, or whose Params name a path value no such route has.
func (p *Plan) checkRouteLocked(r Rule) error {
	if r.Route == "" && len(r.Params) == 0 {
		return nil
	}
	var routes []Route
	for t, rs := range p.routes {
		if r.Target.covers(t) {
			routes = append(routes, rs...)
		}
	}
	missing := "no route of %[1]s has a path value %[3]s"
	if r.Route != "" {
		routes = slices.DeleteFunc(routes, func(rt Route) bool { return rt.Name != r.Route })
		if len(routes) == 0 {
			return fmt.Errorf("%s serves no route %s", r.Target, r.Route)
		}
		missing = "route %[2]s of %[1]s has no path value %[3]s"
	}
	for _, k := range slices.Sorted(maps.Keys(r.Params)) {
		if !slices.ContainsFunc(routes, func(rt Route) bool { return slices.Contains(rt.Params, k) }) {
			return fmt.Errorf(missing, r.Target, r.Route, k)
		}
	}
	return nil
}

func (e *entry) done() bool { return e.Times > 0 && e.matched >= e.Skip+e.Times }

// NewPlan plays rules on fronts. fired, which may be nil, hears each
// occurrence traffic plays, outside the Plan's lock. The occurrences that
// arming plays on open connections are returned by Set and Observe
// instead, so the event log can append them after the record that armed
// the rule.
func NewPlan(fronts []Front, fired func(Firing)) *Plan {
	return &Plan{fronts: fronts, fired: fired, routes: map[Target][]Route{}}
}

// Set checks r and arms it, replacing the rule with its ID, and returns
// the occurrences arming played. Beyond Check, an HTTP rule's route and
// params must be ones a fake serving its target declared through HTTP.
func (p *Plan) Set(r Rule) ([]Firing, error) {
	if err := Check(r, p.fronts); err != nil {
		return nil, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.checkRouteLocked(r); err != nil {
		return nil, err
	}
	p.removeLocked(r.ID)
	e := &entry{Rule: r}
	p.rules = append(p.rules, e)
	var fired []Firing
	if r.After == nil {
		fired = p.armLocked(e)
	}
	p.signalLocked()
	return fired, nil
}

// Clear removes the rule with id, or every rule for "", and reports
// whether it removed any. Connections it held go back to no fault.
func (p *Plan) Clear(id string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	before := len(p.rules)
	if id == "" {
		for _, e := range p.rules {
			p.releaseLocked(e)
		}
		p.rules = nil
	} else {
		p.removeLocked(id)
	}
	p.signalLocked()
	return len(p.rules) < before
}

// Observe arms each waiting rule whose After matches rec, and returns the
// occurrences that arming played. The event log hands it each record
// before Append returns, so rules arm in log order.
func (p *Plan) Observe(rec events.Record) []Firing {
	p.mu.Lock()
	defer p.mu.Unlock()
	var fired []Firing
	armed := false
	for _, e := range p.rules {
		if !e.armed && e.After.Matches(rec) {
			fired = append(fired, p.armLocked(e)...)
			armed = true
		}
	}
	if armed {
		p.signalLocked()
	}
	return fired
}

// Refused reports whether a refuse rule holds f, and returns a channel
// closed when any rule next changes.
func (p *Plan) Refused(f Front) (refused bool, changed <-chan struct{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	refused = slices.ContainsFunc(p.rules, func(e *entry) bool {
		_, ok := e.Fault.(Refuse)
		return ok && e.armed && e.Covers(f)
	})
	return refused, p.changedLocked()
}

// dialing is what a dial on a Network path meets: Refuse while a refuse
// rule holds the path, Stall when a stall would take the connection,
// since a stalled path drops the handshake, and nil otherwise.
func (p *Plan) dialing() Fault {
	if refused, _ := p.Refused(path); refused {
		return Refuse{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, e := range p.rules {
		if e.takes(path, Conn) {
			if e.Fault == (Stall{}) {
				return Stall{}
			}
			return nil
		}
	}
	return nil
}

// TLS counts one handshake on f and returns the failure to play on it.
func (p *Plan) TLS(f Front) (BadTLS, bool) {
	firing, ok := p.take(f, TLS, func(*entry) bool { return true })
	if !ok {
		return 0, false
	}
	bad, _ := firing.Fault.(BadTLS)
	return bad, true
}

// request counts one request on route at f and returns the fault to play.
func (p *Plan) request(f Front, route string, r *http.Request) (Fault, bool) {
	firing, ok := p.take(f, HTTP, func(e *entry) bool {
		if e.Route != "" && e.Route != route {
			return false
		}
		for k, v := range e.Params {
			if r.PathValue(k) != v {
				return false
			}
		}
		return true
	})
	return firing.Fault, ok
}

func (p *Plan) take(f Front, layer Layer, match func(*entry) bool) (Firing, bool) {
	p.mu.Lock()
	e, n := p.firstLocked(f, layer, match)
	p.mu.Unlock()
	if e == nil {
		return Firing{}, false
	}
	firing := Firing{Rule: e.ID, Front: f, Fault: e.Fault, Match: n}
	p.report([]Firing{firing})
	return firing, true
}

// firstLocked counts one occurrence on f against the earliest armed rule
// that matches it, and returns that rule when it plays the occurrence.
func (p *Plan) firstLocked(f Front, layer Layer, match func(*entry) bool) (played *entry, matched int) {
	for _, e := range p.rules {
		if e.takes(f, layer) && match(e) {
			return e.countLocked()
		}
	}
	return nil, 0
}

// takes reports whether e counts an occurrence on layer at f. A refuse
// rule holds a front and counts nothing.
func (e *entry) takes(f Front, layer Layer) bool {
	_, refuse := e.Fault.(Refuse)
	return !refuse && e.armed && !e.done() && e.Fault.Layer() == layer && e.Covers(f)
}

func (e *entry) countLocked() (played *entry, matched int) {
	e.matched++
	if e.matched <= e.Skip {
		return nil, 0
	}
	return e, e.matched
}

// armLocked arms e, which takes each open connection it covers that holds
// no rule.
func (p *Plan) armLocked(e *entry) []Firing {
	e.armed = true
	var fired []Firing
	for _, c := range slices.Clone(p.conns) {
		if c.held != nil || !e.takes(c.front, Conn) {
			continue
		}
		if played, n := e.countLocked(); played != nil {
			p.holdLocked(c, e)
			fired = append(fired, Firing{Rule: e.ID, Front: c.front, Fault: e.Fault, Match: n})
		}
	}
	return fired
}

// openLocked registers c and lets the earliest matching rule take it.
func (p *Plan) openLocked(c *conn) *Firing {
	p.conns = append(p.conns, c)
	e, n := p.firstLocked(c.front, Conn, func(*entry) bool { return true })
	if e == nil {
		return nil
	}
	p.holdLocked(c, e)
	return &Firing{Rule: e.ID, Front: c.front, Fault: e.Fault, Match: n}
}

func (p *Plan) holdLocked(c *conn, e *entry) {
	c.held, c.gen = e, c.gen+1
	if _, ok := e.Fault.(Reset); ok {
		p.dropLocked(c)
		c.kill(ErrReset, true)
	}
}

func (p *Plan) dropLocked(c *conn) {
	p.conns = slices.DeleteFunc(p.conns, func(o *conn) bool { return o == c })
}

func (p *Plan) removeLocked(id string) {
	p.rules = slices.DeleteFunc(p.rules, func(e *entry) bool {
		if e.ID != id {
			return false
		}
		p.releaseLocked(e)
		return true
	})
}

func (p *Plan) releaseLocked(e *entry) {
	for _, c := range p.conns {
		if c.held == e {
			c.held, c.gen = nil, c.gen+1
		}
	}
}

// changedLocked returns a channel closed at the next change. The channel
// is made by the caller, so inside its synctest bubble.
func (p *Plan) changedLocked() <-chan struct{} {
	if p.changed == nil {
		p.changed = make(chan struct{})
	}
	return p.changed
}

func (p *Plan) signalLocked() {
	if p.changed != nil {
		close(p.changed)
		p.changed = nil
	}
}

func (p *Plan) report(fired []Firing) {
	if p.fired == nil {
		return
	}
	for _, f := range fired {
		p.fired(f)
	}
}
