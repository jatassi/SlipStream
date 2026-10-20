package metadata

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
	metadatav1 "github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/metadata/v1/metadatav1connect"
	"github.com/jatassi/slipstream/internal/role"
)

// State is how a provider stands, from the latest outcome of a call to it.
type State uint8

const (
	// StateUnused: nothing has called the provider since the Controller
	// started or its key changed.
	StateUnused State = iota + 1
	StateWorking
	StateKeyRefused
	// StateRateLimited until Until.
	StateRateLimited
	// StateOutOfAllowance is a daily allowance spent, until Until.
	StateOutOfAllowance
	StateUnreachable
)

var states = map[OutcomeKind]State{
	OK:          StateWorking,
	NotFound:    StateWorking,
	Refused:     StateKeyRefused,
	Limited:     StateRateLimited,
	Capped:      StateOutOfAllowance,
	Unreachable: StateUnreachable,
}

// ProviderStatus is how a provider stands, built only from the outcomes of
// the calls holders made and the tests they ran.
type ProviderStatus struct {
	Provider ProviderID
	Label    string
	// Summary says how the provider stands and what to do, in plain words.
	// The fields after Key are the machinery under it.
	Summary     string
	Key         KeyInUse
	State       State
	Until       time.Time
	LastSuccess time.Time
	LastError   string
	LastErrorAt time.Time
	// Nodes is each holder that called the provider, by id.
	Nodes []NodeReach
}

// NodeReach is whether a holder reached the provider on its latest call.
type NodeReach struct {
	Node    role.NodeID
	Reached bool
	At      time.Time
}

type Changed struct {
	Provider string `json:"provider" doc:"The provider whose status changed"`
}

// ChangedEvent fires when a provider's status changes in anything but the
// time of its latest success.
var ChangedEvent = live.AdminEvent[Changed]("metadata.changed")

type status struct {
	p     Provider
	state State
	until time.Time
	// at is the time of the outcome state is from.
	at          time.Time
	lastSuccess time.Time
	lastError   string
	lastErrorAt time.Time
	nodes       map[role.NodeID]NodeReach
}

func newStatus(p Provider) *status {
	return &status{p: p, state: StateUnused, nodes: map[role.NodeID]NodeReach{}}
}

func (s *status) take(node role.NodeID, o Outcome) {
	if !o.At.Before(s.at) {
		s.state, s.until, s.at = states[o.Kind], o.Until, o.At
	}
	if o.Kind == OK || o.Kind == NotFound {
		s.lastSuccess = maxTime(s.lastSuccess, o.At)
	} else if !o.At.Before(s.lastErrorAt) {
		s.lastError, s.lastErrorAt = s.p.Label+" "+o.Words, o.At
	}
	if !o.At.Before(s.nodes[node].At) {
		s.nodes[node] = NodeReach{Node: node, Reached: o.Kind != Unreachable, At: o.At}
	}
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

// reach is one provider on one holder.
type reach struct {
	p    ProviderID
	node role.NodeID
}

// retest is a provider a holder couldn't reach, tested there again at due,
// each time wait doubles.
type retest struct {
	wait time.Duration
	due  time.Time
}

// Providers is every provider's status, in display order.
func (m *Metadata) Providers(context.Context, identity.Admin) []ProviderStatus {
	out := make([]ProviderStatus, len(m.cfg.Providers))
	for i, p := range m.cfg.Providers {
		out[i] = m.statusOf(p.ID)
	}
	return out
}

// Test runs p's test call on a holder with the key in use, and answers
// p's status after it. The test spends one call of p's allowance.
func (m *Metadata) Test(ctx context.Context, _ identity.Admin, p ProviderID) (ProviderStatus, error) {
	if _, ok := m.label(p); !ok {
		return ProviderStatus{}, ErrNoProvider
	}
	_, err := send(ctx, m, role.NodeID{}, func(ctx context.Context, s metadatav1connect.MetadataServiceHandler) (*metadatav1.TestResponse, error) {
		return s.Test(ctx, m.test(p))
	})
	if err != nil {
		return ProviderStatus{}, err
	}
	return m.statusOf(p), nil
}

func (m *Metadata) test(p ProviderID) *metadatav1.TestRequest {
	return &metadatav1.TestRequest{Ask: askToWire(Ask{keys: m.inEffect(), fresh: true, generation: m.cacheGeneration()}), Provider: string(p)}
}

// Run retests each provider a holder couldn't reach, on that holder, a
// RetestMin after the call that failed and then each RetestMax at most,
// doubling the wait each time, until it answers. Beside it, Run fetches
// the images Prefetch queues.
func (m *Metadata) Run(ctx context.Context) error {
	var prefetching sync.WaitGroup
	prefetching.Go(func() { m.prefetching(ctx) })
	defer prefetching.Wait()
	defer m.gallery.close()
	for {
		var fire <-chan time.Time
		if next, ok := m.nextRetest(); ok {
			fire = time.After(time.Until(next))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-m.wake:
		case <-fire:
			m.retestDue(ctx)
		}
	}
}

func (m *Metadata) nextRetest() (time.Time, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var next time.Time
	for _, t := range m.retests {
		if next.IsZero() || t.due.Before(next) {
			next = t.due
		}
	}
	return next, !next.IsZero()
}

func (m *Metadata) retestDue(ctx context.Context) {
	now := time.Now()
	m.mu.Lock()
	var due []reach
	for r, t := range m.retests {
		if !now.Before(t.due) {
			due = append(due, r)
		}
	}
	m.mu.Unlock()
	slices.SortFunc(due, func(a, b reach) int {
		if a.p != b.p {
			return strings.Compare(string(a.p), string(b.p))
		}
		return strings.Compare(a.node.String(), b.node.String())
	})
	for _, r := range due {
		if node, ok := m.cfg.Roster.Pick(role.Metadata, r.node); ok && node == r.node {
			res, err := m.cfg.Service(r.node).Test(ctx, m.test(r.p))
			if err == nil {
				m.fold(ctx, r.node, outcomesOf(res.GetOutcomes()))
			}
		}
		m.mu.Lock()
		if t, ok := m.retests[r]; ok {
			t.wait = min(2*t.wait, m.cfg.Timings.RetestMax)
			t.due = time.Now().Add(t.wait)
		}
		m.mu.Unlock()
	}
}

// fold takes the outcomes node answered with into status and announces
// each provider whose status changed.
func (m *Metadata) fold(ctx context.Context, node role.NodeID, outcomes []Outcome) {
	m.mu.Lock()
	changed := m.take(node, outcomes)
	m.mu.Unlock()
	m.announce(ctx, changed)
}

// rekeyed puts key in use for p, or SlipStream's when key is empty, and
// starts p's status afresh from outcomes, which node answered testing it.
func (m *Metadata) rekeyed(ctx context.Context, p ProviderID, key string, node role.NodeID, outcomes []Outcome) {
	m.mu.Lock()
	if key == "" {
		delete(m.yours, p)
	} else {
		m.yours[p] = key
	}
	m.statuses[p] = newStatus(m.statuses[p].p)
	changed := m.take(node, outcomes)
	m.mu.Unlock()
	if !slices.Contains(changed, p) {
		changed = append([]ProviderID{p}, changed...)
	}
	m.announce(ctx, changed)
}

// take folds outcomes into status and schedules or ends retests, and
// answers each provider whose status changed; m.mu is held.
func (m *Metadata) take(node role.NodeID, outcomes []Outcome) []ProviderID {
	before := map[ProviderID]ProviderStatus{}
	var touched []ProviderID
	for _, o := range outcomes {
		s, ok := m.statuses[o.Provider]
		if !ok {
			continue
		}
		if _, seen := before[o.Provider]; !seen {
			before[o.Provider] = m.view(s)
			touched = append(touched, o.Provider)
		}
		s.take(node, o)
		m.schedule(reach{p: o.Provider, node: node}, o.Kind)
	}
	return slices.DeleteFunc(touched, func(p ProviderID) bool { return before[p].sameAs(m.view(m.statuses[p])) })
}

// schedule starts retesting r once it is unreachable and ends it once it
// answers; m.mu is held.
func (m *Metadata) schedule(r reach, k OutcomeKind) {
	if k != Unreachable {
		delete(m.retests, r)
		return
	}
	if _, ok := m.retests[r]; ok {
		return
	}
	m.retests[r] = &retest{wait: m.cfg.Timings.RetestMin, due: time.Now().Add(m.cfg.Timings.RetestMin)}
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func (m *Metadata) announce(ctx context.Context, changed []ProviderID) {
	for _, p := range changed {
		m.cfg.Changed.Publish(ctx, Changed{Provider: string(p)})
	}
}

func (m *Metadata) statusOf(p ProviderID) ProviderStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.view(m.statuses[p])
}

func (m *Metadata) label(p ProviderID) (string, bool) {
	s, ok := m.statuses[p]
	if !ok {
		return "", false
	}
	return s.p.Label, true
}

// view is s as the API reads it; m.mu is held.
func (m *Metadata) view(s *status) ProviderStatus {
	v := ProviderStatus{
		Provider: s.p.ID, Label: s.p.Label, Key: m.keyInUse(s.p.ID), State: s.state, Until: s.until,
		LastSuccess: s.lastSuccess, LastError: s.lastError, LastErrorAt: s.lastErrorAt, Nodes: []NodeReach{},
	}
	for _, n := range s.nodes {
		v.Nodes = append(v.Nodes, n)
	}
	slices.SortFunc(v.Nodes, func(a, b NodeReach) int { return strings.Compare(a.Node.String(), b.Node.String()) })
	v.Summary = summary(v)
	return v
}

// sameAs is s and o alike but for when things last went right.
func (s ProviderStatus) sameAs(o ProviderStatus) bool {
	reached := func(x, y NodeReach) bool { return x.Node == y.Node && x.Reached == y.Reached }
	return s.Summary == o.Summary && s.Key == o.Key && s.State == o.State && s.Until.Equal(o.Until) &&
		s.LastError == o.LastError && slices.EqualFunc(s.Nodes, o.Nodes, reached)
}

func summary(s ProviderStatus) string {
	l := s.Label
	switch s.State {
	case StateUnused:
		if s.Key == KeyNone {
			return noKey(l)
		}
		return l + " hasn't been called yet."
	case StateWorking:
		return l + " is working."
	case StateKeyRefused:
		switch s.Key {
		case KeyNone:
			return noKey(l)
		case KeyBundled:
			return l + " refused SlipStream's key. Add your own."
		case KeyYours:
			return l + " refused your key. Check it and save it again, or remove it."
		}
	case StateRateLimited:
		return l + " is rate limiting SlipStream. Calls go again once the limit ends."
	case StateOutOfAllowance:
		if s.Key == KeyBundled {
			return l + " is out of today's allowance. It resets on its own, or add your own key to go on now."
		}
		return l + " is out of today's allowance. It resets on its own."
	case StateUnreachable:
		return "SlipStream can't reach " + l + ". It tries again on its own; check the network of each Node that can't reach it."
	}
	panic(fmt.Sprintf("metadata: %s stands as state %d with key %d", l, s.State, s.Key))
}

func noKey(label string) string { return "This build has no " + label + " key. Add your own." }

// lastOf is the latest outcome for p in outcomes.
func lastOf(outcomes []Outcome, p ProviderID) (Outcome, bool) {
	for _, o := range slices.Backward(outcomes) {
		if o.Provider == p {
			return o, true
		}
	}
	return Outcome{}, false
}
