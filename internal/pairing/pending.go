package pairing

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
)

// PendingState is where a pending claim stands: awaiting the Controller's
// dial, then joined or failed.
type PendingState uint8

const (
	AwaitingDial PendingState = iota
	JoinedState
	FailedState
)

func (s PendingState) String() string {
	return [...]string{"awaiting-dial", "joined", "failed"}[s]
}

// Pending is the last claim that couldn't reach its Controller and waited
// for it to dial this Node at Address instead.
type Pending struct {
	State PendingState
	// Tried is every address in the code, none of which answered.
	Tried   []netip.AddrPort
	Address joincode.NodeAddress
	Until   time.Time
	// Joined is set once JoinedState, and Err once FailedState.
	Joined *Joined
	Err    *Error
}

// Lines are what a person reads while the claim awaits the dial. The code
// carries no Controller name, so they say "the Controller".
func (p Pending) Lines() []string {
	tried := make([]string, len(p.Tried))
	for i, a := range p.Tried {
		tried[i] = a.String()
	}
	return []string{
		"Can't reach the Controller at " + strings.Join(tried, ", ") + ".",
		"Paste this Node address into the Add a Node dialog on the Controller:",
		"  " + p.Address.String(),
		"Waiting for the Controller to dial until " + p.Until.Format("15:04") + ". Ctrl-C stops.",
	}
}

// ErrJoinPending refuses a claim while another awaits its Controller's
// dial.
var ErrJoinPending = &Error{Kind: JoinPending, Reason: "a Join waits for its Controller to dial this Node"}

// Pending is the last pending claim, false when no claim has waited. It
// takes no lock.
func (p *Pairing) Pending() (Pending, bool) {
	pend := p.pending.Load()
	if pend == nil {
		return Pending{}, false
	}
	return *pend, true
}

// Awaiting reports whether a claim awaits its Controller's dial now.
func (p *Pairing) Awaiting() bool {
	pend := p.pending.Load()
	return pend != nil && pend.State == AwaitingDial
}

// await has the link wait for code's Controller to dial this Node, until
// await_dial or ctx ends. Callers hold p.mu; await releases it for the
// wait and holds it again when it returns.
func (p *Pairing) await(ctx context.Context, code joincode.Code, unreachable *Error, awaiting func(Pending)) (*Pending, link.Paired, error) {
	addrs, ok := p.cfg.Link.Advertised()
	if !ok {
		return nil, link.Paired{}, &Error{Kind: Unaddressed, Reason: unreachable.Reason + ", and this Node's link has no address for it to dial"}
	}
	addr, err := joincode.NewNodeAddress(p.cfg.Key[:], addrs.All())
	if err != nil {
		return nil, link.Paired{}, fmt.Errorf("this Node's address: %w", err)
	}
	pend := &Pending{State: AwaitingDial, Tried: code.Addresses(), Address: addr, Until: time.Now().Add(p.cfg.AwaitDial)}
	p.pending.Store(pend)
	p.mu.Unlock()
	defer p.mu.Lock()
	awaiting(*pend)
	wait, cancel := context.WithDeadline(ctx, pend.Until)
	defer cancel()
	paired, err := p.cfg.Link.AwaitPair(wait, code, p.cfg.Name)
	if err == nil {
		return pend, paired, nil
	}
	refused := refusal(err)
	switch {
	case refused.Kind != Unreachable:
	case ctx.Err() != nil:
		refused = &Error{Kind: Unreachable, Reason: "stopped waiting for the Controller to dial this Node"}
	default:
		refused = &Error{Kind: Unreachable, Reason: "the Controller didn't dial this Node by " + pend.Until.Format("15:04") + ". Ask for a new Join Code."}
	}
	p.fail(pend, refused)
	return nil, link.Paired{}, refused
}

// fail ends pend with err. An err that is no *Error has no Kind.
func (p *Pairing) fail(pend *Pending, err error) {
	failed := *pend
	failed.State = FailedState
	var ok bool
	if failed.Err, ok = errors.AsType[*Error](err); !ok {
		failed.Err = &Error{Reason: err.Error()}
	}
	p.pending.Store(&failed)
}
