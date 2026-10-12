package api

import (
	"context"
	"reflect"
	"time"

	"github.com/danielgtaylor/huma/v2/sse"

	"github.com/jatassi/slipstream/internal/live"
)

// StreamMessage is one message on an event stream. Data is a declared
// event's payload, or nil for a message that only carries Retry or a
// Comment, such as a keepalive.
type StreamMessage struct {
	// ID is written only when it is above zero.
	ID      int64
	Data    any
	Retry   time.Duration
	Comment string
}

// Send writes one message and flushes it. It fails once the client is gone
// or misses the per-message write deadline.
type Send func(StreamMessage) error

// RegisterStream adds an event stream, under the same checks as Register.
// events are every event the stream may send; the document lists each
// one's payload. The handler runs until the stream ends; an error it
// returns is logged, since the client already has its 200.
func RegisterStream[P, I any](o *Owner, level Level[P], op Op, events []live.Event, handler func(context.Context, P, *I, Send) error) {
	r := o.router
	hop, _ := humaOperation(o, level, op, reflect.TypeFor[I]())
	payloads := make(map[string]any, len(events))
	for _, e := range events {
		payloads[e.Name()] = e.Payload()
	}
	sse.Register(r.huma, hop, payloads, func(ctx context.Context, in *I, send sse.Sender) {
		p := principal[P](ctx, op)
		defer func() {
			if v := recover(); v != nil {
				r.logPanic(op, v)
			}
		}()
		err := handler(ctx, p, in, func(m StreamMessage) error {
			return send(sse.Message{ID: int(m.ID), Data: m.Data, Retry: int(m.Retry / time.Millisecond), Comment: m.Comment})
		})
		if err != nil {
			r.logFailure(ctx, "operation failed", "method", op.Method, "path", Prefix+op.Path, "operation", op.ID, "error", err)
		}
	})
}
