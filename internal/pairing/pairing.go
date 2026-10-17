// Package pairing owns a Worker's half of a pairing: pairing.json, Claim,
// the Worker's and Unclaimed Node's Trust, the join control op and the join
// config section.
package pairing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/link"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/system"
)

// Settings is the join config section.
type Settings struct {
	// Code joins an Unclaimed Node at start. It is always redacted.
	Code joincode.Code `yaml:"code"`
}

// Link is the Node link as a pairing uses it.
type Link interface {
	Pair(ctx context.Context, code joincode.Code, name string) (link.Paired, error)
	AwaitPair(ctx context.Context, code joincode.Code, name string) (link.Paired, error)
	Advertised() (link.Addresses, bool)
	Keep(t link.Target)
	Drop(key link.Key)
	Subscribe(ctx context.Context) <-chan link.Change
}

// Claims is the composition root's one definition of the Node's claim.
type Claims interface {
	Claim(ctx context.Context) (system.Claim, error)
}

type Config struct {
	DataDir string
	// Key is this Node's, which its Node address carries.
	Key  link.Key
	Link Link
	// Name is sent once, at Pair.
	Name   string
	Claims Claims
	// AwaitDial bounds a claim's wait for its Controller to dial.
	AwaitDial time.Duration
	Log       *slog.Logger
}

// Pairing is one Node's pairing record and what follows from it.
type Pairing struct {
	cfg  Config
	path string

	mu      sync.Mutex
	rec     atomic.Pointer[Record]
	pending atomic.Pointer[Pending]
}

// Open reads pairing.json when the data dir holds one. A record it cannot
// use is a *RecordError.
func Open(cfg Config) (*Pairing, error) {
	p := &Pairing{cfg: cfg, path: recordPath(cfg.DataDir)}
	rec, ok, err := readRecord(p.path)
	if err != nil {
		return nil, err
	}
	if ok {
		p.rec.Store(&rec)
	}
	return p, nil
}

// Record is the pairing record, false on a Node that has none.
func (p *Pairing) Record() (Record, bool) {
	rec := p.rec.Load()
	if rec == nil {
		return Record{}, false
	}
	return *rec, true
}

// Controller is the pinned Controller's key and recorded addresses. Only
// a Worker, which holds a record, asks.
func (p *Pairing) Controller() link.Target {
	return p.rec.Load().Controller
}

// Admit pins the Controller, answers a superseded one with the key that
// replaced it, and refuses the rest.
func (p *Pairing) Admit(key link.Key) link.Admission {
	rec := p.rec.Load()
	switch {
	case rec == nil:
		return link.Refused{}
	case key == rec.Controller.Key:
		return link.Pinned{}
	case slices.ContainsFunc(rec.Superseded, func(s Superseded) bool { return s.Key == key }):
		return link.Superseded{By: rec.Controller.Key, Addresses: rec.Controller.Addresses}
	}
	return link.Refused{}
}

// Pair refuses: a Worker is not a Controller.
func (p *Pairing) Pair(context.Context, link.Key, *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return &linkv1.PairResponse{Outcome: &linkv1.PairResponse_Refused{
		Refused: &linkv1.Refused{Reason: linkv1.RefusedReason_REFUSED_REASON_NOT_A_CONTROLLER},
	}}, nil
}

// Joined is a Claim that paired: this Node's id and name as the
// Controller recorded them, the Controller's name and key, and its address
// on the session the Pair made live.
type Joined struct {
	NodeID         role.NodeID
	Name           string
	ControllerName string
	Controller     link.Key
	Link           netip.AddrPort
}

// Claim joins the Controller code names, returning once the session is
// live. A code naming the pinned or a superseded Controller changes
// nothing, another Controller's code re-pairs, and a Controller refuses.
// A Controller none of the code's addresses answer within join_dial may
// dial this Node instead: Claim calls awaiting once with the pending
// claim, then waits for that dial within await_dial. Every refusal is an
// *Error, ErrJoinPending while another claim waits.
func (p *Pairing) Claim(ctx context.Context, code joincode.Code, awaiting func(Pending)) (Joined, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Awaiting() {
		return Joined{}, ErrJoinPending
	}
	claim, err := p.cfg.Claims.Claim(ctx)
	if err != nil {
		return Joined{}, err
	}
	if claim == system.Controller {
		return Joined{}, &Error{Kind: IsController, Reason: "this Node is a Controller"}
	}
	prev := p.rec.Load()
	if err := ignored(prev, code); err != nil {
		return Joined{}, err
	}
	var pend *Pending
	paired, err := p.cfg.Link.Pair(ctx, code, p.cfg.Name)
	if err != nil {
		refused := refusal(err)
		if refused.Kind != Unreachable {
			return Joined{}, refused
		}
		if pend, paired, err = p.await(ctx, code, refused, awaiting); err != nil {
			return Joined{}, err
		}
		prev = p.rec.Load()
	}
	joined, err := p.record(prev, code, paired)
	if pend != nil {
		if err != nil {
			p.fail(pend, err)
			return Joined{}, err
		}
		done := *pend
		done.State, done.Joined = JoinedState, &joined
		p.pending.Store(&done)
	}
	return joined, err
}

// record writes the pairing paired made, superseding prev's Controller
// when paired names another, and keeps the Controller. Callers hold p.mu.
func (p *Pairing) record(prev *Record, code joincode.Code, paired link.Paired) (Joined, error) {
	id, err := storage.UUIDFromBytes(paired.NodeID)
	if err != nil {
		return Joined{}, fmt.Errorf("the Controller paired this Node as %x: %w", paired.NodeID, err)
	}
	addrs := paired.Node.Addresses
	if addrs.IsZero() {
		if addrs, err = link.ParseAddresses(code.Addresses()); err != nil {
			return Joined{}, err
		}
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	next := Record{
		NodeID:         role.NodeID{UUID: id},
		Controller:     link.Target{Key: paired.Controller, Addresses: addrs},
		ControllerName: paired.ControllerName,
		PairedAt:       now,
	}
	if prev != nil {
		next.Superseded = slices.Clone(prev.Superseded)
		if prev.Controller.Key != paired.Controller {
			next.Superseded = append(next.Superseded, Superseded{Key: prev.Controller.Key, By: paired.Controller, At: now})
		}
	}
	if err := writeRecord(p.path, next); err != nil {
		return Joined{}, err
	}
	p.rec.Store(&next)
	if prev != nil && prev.Controller.Key != paired.Controller {
		p.cfg.Link.Drop(prev.Controller.Key)
	}
	p.cfg.Link.Keep(next.Controller)
	return Joined{NodeID: next.NodeID, Name: paired.Name, ControllerName: paired.ControllerName, Controller: paired.Controller, Link: paired.Remote}, nil
}

// ignored is the refusal of a code naming rec's pinned or a superseded
// Controller, nil for any other code or with no record.
func ignored(rec *Record, code joincode.Code) *Error {
	switch {
	case rec == nil:
		return nil
	case code.Names(rec.Controller.Key[:]):
		return &Error{Kind: AlreadyJoined, Reason: "already joined"}
	case slices.ContainsFunc(rec.Superseded, func(s Superseded) bool { return code.Names(s.Key[:]) }):
		return &Error{Kind: SupersededController, Reason: "superseded Controller"}
	}
	return nil
}

// Repairs is whether Claim would re-pair this Node with code: it holds a
// record, and code names neither its pinned nor a superseded Controller.
func (p *Pairing) Repairs(code joincode.Code) bool {
	rec := p.rec.Load()
	return rec != nil && !code.IsZero() && ignored(rec, code) == nil
}

// Run follows the link until ctx ends, rewriting the Controller's
// addresses when its Welcome advertises others, so re-dials find a moved
// Controller.
func (p *Pairing) Run(ctx context.Context) error {
	for c := range p.cfg.Link.Subscribe(ctx) {
		live, ok := c.State.(link.Live)
		if !ok || live.Peer.Addresses.IsZero() {
			continue
		}
		if err := p.follow(c.Key, live.Peer.Addresses); err != nil {
			p.cfg.Log.Error("rewrite the pairing record", "err", err.Error())
		}
	}
	return nil
}

func (p *Pairing) follow(key link.Key, addrs link.Addresses) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	rec := p.rec.Load()
	if rec == nil || rec.Controller.Key != key || rec.Controller.Addresses == addrs {
		return nil
	}
	next := *rec
	next.Controller.Addresses = addrs
	if err := writeRecord(p.path, next); err != nil {
		return err
	}
	p.rec.Store(&next)
	p.cfg.Link.Keep(next.Controller)
	return nil
}

// Kind is why a Claim changed nothing. The refusals a Setup request can
// meet are named as its problem codes.
type Kind string

const (
	AlreadyJoined        Kind = "already_joined"
	SupersededController Kind = "superseded_controller"
	IsController         Kind = "controller_node"
	CodeUnknown          Kind = "join_code_unknown"
	CodeExpired          Kind = "join_code_expired"
	CodeUsed             Kind = "join_code_used"
	KeyMismatch          Kind = "controller_key_mismatch"
	Unreachable          Kind = "controller_unreachable"
	RestartRequired      Kind = "restart_required"
	JoinPending          Kind = "join_pending"
	Unaddressed          Kind = "node_unaddressed"
)

// ErrRestartRequired refuses a re-pair of a Worker whose half runs. That
// half stays bound to the Controller it started with, so a Worker re-pairs
// only before its half starts.
var ErrRestartRequired = &Error{
	Kind:   RestartRequired,
	Reason: "a running Worker re-pairs only at start: stop the Node and run slipstream join with this code, or restart it with the code in SLIPSTREAM_JOIN_CODE",
}

// Ignored is whether k left a pairing in place rather than refusing one:
// the code names the pinned or a superseded Controller.
func (k Kind) Ignored() bool { return k == AlreadyJoined || k == SupersededController }

// Error is a Claim that changed nothing, with the reason a person reads.
type Error struct {
	Kind   Kind   `json:"kind"`
	Reason string `json:"reason"`
}

func (e *Error) Error() string { return e.Reason }

// refusedBy is each Pair refusal a Controller can answer a Join with.
var refusedBy = map[linkv1.RefusedReason]Error{
	linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_UNKNOWN: {Kind: CodeUnknown, Reason: "the Controller does not know the Join Code"},
	linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_EXPIRED: {Kind: CodeExpired, Reason: "the Join Code has expired"},
	linkv1.RefusedReason_REFUSED_REASON_JOIN_CODE_USED:    {Kind: CodeUsed, Reason: "the Join Code was already used"},
}

func refusal(err error) *Error {
	if r, ok := errors.AsType[*link.RefusedError](err); ok {
		if e, ok := refusedBy[r.Reason]; ok {
			return &e
		}
	}
	if errors.Is(err, link.ErrKeyMismatch) {
		return &Error{Kind: KeyMismatch, Reason: err.Error()}
	}
	return &Error{Kind: Unreachable, Reason: "the Controller is unreachable: " + err.Error()}
}

// Paired is whether this Node holds a pairing record.
func (p *Pairing) Paired() bool { return p.rec.Load() != nil }
