// Package live owns the event hub, event declarations and their publishers.
// It has no HTTP code and no tables.
package live

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/storage"
)

// Audience is who may see an event.
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

func (d *AdminDecl[T]) Bind(h *Hub) AdminPublisher[T]       { panic("live: not built") }
func (d *AccountDecl[T]) Bind(h *Hub) AccountPublisher[T]   { panic("live: not built") }
func (d *EveryoneDecl[T]) Bind(h *Hub) EveryonePublisher[T] { panic("live: not built") }

func (p AdminPublisher[T]) Publish(ctx context.Context, v T) { panic("live: not built") }
func (p AccountPublisher[T]) Publish(ctx context.Context, account storage.UUID, v T) {
	panic("live: not built")
}
func (p EveryonePublisher[T]) Publish(ctx context.Context, v T) { panic("live: not built") }

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

type Hub struct{}

// NewHub builds a hub over every declared event. It panics on a duplicate
// name or a payload type used twice.
func NewHub(events []Event) *Hub { panic("live: not built") }

// Subscribe starts a cursor for viewer. lastEventID is the raw
// Last-Event-ID header, empty when absent.
func (h *Hub) Subscribe(viewer Viewer, lastEventID string) *Subscription {
	panic("live: not built")
}

// EndSession makes each of the session's subscriptions return
// SessionEnded{"signed-out"} and then ErrEnded.
func (h *Hub) EndSession(session storage.UUID) { panic("live: not built") }

// Close ends every subscription.
func (h *Hub) Close() { panic("live: not built") }

type Subscription struct{}

// Next blocks until the viewer's next visible message, ctx ends, or the
// subscription ends with ErrEnded.
func (s *Subscription) Next(ctx context.Context) (Message, error) { panic("live: not built") }

func (s *Subscription) Close() { panic("live: not built") }
