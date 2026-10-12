// Package live owns the event hub, event declarations and their publishers.
// It has no HTTP code and no tables.
package live

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

// Timings is the timings.live section.
type Timings struct {
	// Heartbeat is how often a stream sends a keepalive and checks its
	// session.
	Heartbeat time.Duration `yaml:"heartbeat"`
	// Retry is how long a browser waits before it reconnects a dropped
	// stream.
	Retry time.Duration `yaml:"retry"`
}

func DefaultTimings() Timings {
	return Timings{Heartbeat: 25 * time.Second, Retry: 3 * time.Second}
}

type Audience uint8

const (
	ToAdmins Audience = iota + 1
	ToAccount
	ToEveryone
)

// Event is a declared event. surface.Events lists every one.
type Event interface {
	Name() string
	Audience() Audience
	// Payload is the zero value of the event's payload type.
	Payload() any
}

type decl[T any] struct {
	name     string
	audience Audience
}

func (d decl[T]) Name() string       { return d.name }
func (d decl[T]) Audience() Audience { return d.audience }
func (d decl[T]) Payload() any       { var v T; return v }

type AdminDecl[T any] struct{ decl[T] }

type AccountDecl[T any] struct{ decl[T] }

type EveryoneDecl[T any] struct{ decl[T] }

func AdminEvent[T any](name string) *AdminDecl[T] {
	return &AdminDecl[T]{decl[T]{name, ToAdmins}}
}

func AccountEvent[T any](name string) *AccountDecl[T] {
	return &AccountDecl[T]{decl[T]{name, ToAccount}}
}

func EveryoneEvent[T any](name string) *EveryoneDecl[T] {
	return &EveryoneDecl[T]{decl[T]{name, ToEveryone}}
}

type AdminPublisher[T any] struct {
	hub  *Hub
	decl Event
}

type AccountPublisher[T any] struct {
	hub  *Hub
	decl Event
}

type EveryonePublisher[T any] struct {
	hub  *Hub
	decl Event
}

// Bind panics when the hub was not built with d, so an event missing from
// surface.Events fails at start.
func (d *AdminDecl[T]) Bind(h *Hub) AdminPublisher[T] {
	h.mustDeclare(d)
	return AdminPublisher[T]{h, d}
}

func (d *AccountDecl[T]) Bind(h *Hub) AccountPublisher[T] {
	h.mustDeclare(d)
	return AccountPublisher[T]{h, d}
}

func (d *EveryoneDecl[T]) Bind(h *Hub) EveryonePublisher[T] {
	h.mustDeclare(d)
	return EveryonePublisher[T]{h, d}
}

func (p AdminPublisher[T]) Publish(ctx context.Context, v T) {
	p.hub.publish(ctx, p.decl, storage.UUID{}, v)
}

func (p AccountPublisher[T]) Publish(ctx context.Context, account storage.UUID, v T) {
	p.hub.publish(ctx, p.decl, account, v)
}

func (p EveryonePublisher[T]) Publish(ctx context.Context, v T) {
	p.hub.publish(ctx, p.decl, storage.UUID{}, v)
}

// Resync means refetch everything: events may have been missed.
type Resync struct{}

// SessionEnded means the stream's session ended and the stream closes.
type SessionEnded struct {
	Reason string `json:"reason" enum:"signed-out,expired"`
}

var (
	ResyncEvent       = EveryoneEvent[Resync]("live.resync")
	SessionEndedEvent = AccountEvent[SessionEnded]("live.session-ended")
)

// Viewer is who a subscription reads for, fixed at connect.
type Viewer struct {
	Account storage.UUID
	Session storage.UUID
	Admin   bool
}

// Message is one event as a stream writes it.
type Message struct {
	ID   int64
	Name string
	Data any
}

var ErrEnded = errors.New("live: subscription ended")

const ringSize = 4096

type entry struct {
	msg      Message
	audience Audience
	account  storage.UUID
}

func (e entry) visibleTo(v Viewer) bool {
	switch e.audience {
	case ToAdmins:
		return v.Admin
	case ToAccount:
		return e.account == v.Account
	case ToEveryone:
		return true
	}
	return false
}

type Hub struct {
	events map[string]Event

	mu     sync.Mutex
	ring   [ringSize]entry
	base   int64
	head   int64
	notify chan struct{}
	subs   map[*Subscription]struct{}
	closed bool
}

// NewHub builds a hub over every declared event. It panics on a duplicate
// name or a payload type used twice. events is surface.Events; the hub adds
// ResyncEvent and SessionEndedEvent itself.
func NewHub(events []Event) *Hub {
	all := append([]Event{ResyncEvent, SessionEndedEvent}, events...)
	byName := make(map[string]Event, len(all))
	byPayload := make(map[reflect.Type]Event, len(all))
	for _, e := range all {
		if _, ok := byName[e.Name()]; ok {
			panic("live: event " + e.Name() + " is declared twice")
		}
		payload := reflect.TypeOf(e.Payload())
		if other, ok := byPayload[payload]; ok {
			panic(fmt.Sprintf("live: events %s and %s share payload type %v; give each event its own type", other.Name(), e.Name(), payload))
		}
		byName[e.Name()] = e
		byPayload[payload] = e
	}
	base := time.Now().UnixMicro()
	return &Hub{
		events: byName,
		base:   base,
		head:   base,
		notify: make(chan struct{}),
		subs:   make(map[*Subscription]struct{}),
	}
}

// Events is every event the hub carries, its two control events included,
// ordered by name.
func (h *Hub) Events() []Event {
	return slices.SortedFunc(maps.Values(h.events), func(a, b Event) int { return strings.Compare(a.Name(), b.Name()) })
}

func (h *Hub) mustDeclare(e Event) {
	if h.events[e.Name()] != e {
		panic("live: event " + e.Name() + " is not in surface.Events")
	}
}

func (h *Hub) publish(ctx context.Context, e Event, account storage.UUID, data any) {
	if storage.InWrite(ctx) {
		panic("live: Publish inside Write; publish after the commit")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.head++
	h.ring[h.head%ringSize] = entry{
		msg:      Message{ID: h.head, Name: e.Name(), Data: data},
		audience: e.Audience(),
		account:  account,
	}
	h.wake()
}

// wake runs under mu. Closing the channel wakes every waiting reader at
// once, so a publisher never blocks on a slow one.
func (h *Hub) wake() {
	close(h.notify)
	h.notify = make(chan struct{})
}

// oldest is the smallest id still in the ring. It runs under mu.
func (h *Hub) oldest() int64 {
	return max(h.base+1, h.head-ringSize+1)
}

// Subscribe starts a cursor for viewer. lastEventID is the raw
// Last-Event-ID header, empty when absent. When the ring still holds every
// event after it, the subscription replays them; otherwise its first
// message is live.resync at the head.
func (h *Hub) Subscribe(viewer Viewer, lastEventID string) *Subscription {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := &Subscription{hub: h, viewer: viewer}
	if h.closed {
		s.state = ended
		return s
	}
	last, err := strconv.ParseInt(lastEventID, 10, 64)
	if err == nil && last+1 >= h.oldest() && last <= h.head {
		s.next = last + 1
	} else {
		s.next = h.head + 1
		s.resync = true
	}
	h.subs[s] = struct{}{}
	return s
}

// EndSession makes each of the session's subscriptions return
// SessionEnded{"signed-out"} and then ErrEnded.
func (h *Hub) EndSession(session storage.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		if s.viewer.Session == session {
			s.state = signingOut
		}
	}
	h.wake()
}

func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for s := range h.subs {
		s.state = ended
	}
	clear(h.subs)
	h.wake()
}

type subscriptionState uint8

const (
	reading subscriptionState = iota
	signingOut
	ended
)

// Subscription is one viewer's cursor into the hub's ring. Its fields are
// guarded by hub.mu.
type Subscription struct {
	hub    *Hub
	viewer Viewer
	state  subscriptionState
	next   int64
	resync bool
}

// Next blocks until the viewer's next visible message, ctx ends, or the
// subscription ends with ErrEnded.
func (s *Subscription) Next(ctx context.Context) (Message, error) {
	h := s.hub
	for {
		h.mu.Lock()
		msg, ok, err := s.take()
		wait := h.notify
		h.mu.Unlock()
		if ok || err != nil {
			return msg, err
		}
		select {
		case <-wait:
		case <-ctx.Done():
			return Message{}, fmt.Errorf("live: %w", ctx.Err())
		}
	}
}

// take runs under hub.mu.
func (s *Subscription) take() (Message, bool, error) {
	h := s.hub
	switch s.state {
	case reading:
	case ended:
		return Message{}, false, ErrEnded
	case signingOut:
		s.end()
		return Message{Name: SessionEndedEvent.Name(), Data: SessionEnded{Reason: "signed-out"}}, true, nil
	}
	if s.next < h.oldest() {
		s.next = h.head + 1
		s.resync = true
	}
	if s.resync {
		s.resync = false
		return Message{ID: s.next - 1, Name: ResyncEvent.Name(), Data: Resync{}}, true, nil
	}
	for ; s.next <= h.head; s.next++ {
		if e := h.ring[s.next%ringSize]; e.visibleTo(s.viewer) {
			s.next++
			return e.msg, true, nil
		}
	}
	return Message{}, false, nil
}

// end runs under hub.mu.
func (s *Subscription) end() {
	s.state = ended
	delete(s.hub.subs, s)
}

func (s *Subscription) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	s.end()
}
