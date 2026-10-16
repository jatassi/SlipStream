package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// StreamEvent declares one event a stream may send. The document names it
// by Payload's Go type, so each event of a stream has its own type.
type StreamEvent struct {
	Name    string
	Payload any
}

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

// StreamFunc runs an accepted stream until ctx ends, which it also does at
// Router.EndStreams. An error it returns is logged, since the client
// already has its 200.
type StreamFunc func(ctx context.Context, send Send) error

// writeTimeout bounds writing one message.
const writeTimeout = 5 * time.Second

// RegisterStream adds an event stream, under the same checks as Register.
// events are every event the stream may send. handler runs first and may
// answer any declared code as a problem; once it returns a StreamFunc, the
// 200 and the events follow.
func RegisterStream[P, I any](o *Owner, level Level[P], op Op, events []StreamEvent, handler func(context.Context, P, *I) (StreamFunc, error)) {
	r := o.router
	hop, declared := humaOperation(o, level, op, reflect.TypeFor[I]())
	names := documentStream(r.huma, &hop, events)
	huma.Register(r.huma, hop, func(ctx context.Context, in *I) (out *huma.StreamResponse, err error) {
		p := principal[P](ctx, op)
		defer func() {
			if v := recover(); v != nil {
				r.logPanic(op, v)
				out, err = nil, Internal.problem()
			}
		}()
		run, err := handler(ctx, p, in)
		if err != nil {
			return nil, r.failure(ctx, op, declared, err)
		}
		return &huma.StreamResponse{Body: func(hc huma.Context) { r.stream(hc, op, names, run) }}, nil
	})
}

// documentStream documents events as the 200's text/event-stream schema,
// one oneOf branch per event, and returns each payload type's event name.
func documentStream(api huma.API, op *huma.Operation, events []StreamEvent) map[reflect.Type]string {
	names := make(map[reflect.Type]string, len(events))
	branches := make([]*huma.Schema, 0, len(events))
	for _, e := range events {
		t := deref(reflect.TypeOf(e.Payload))
		names[t] = e.Name
		branches = append(branches, &huma.Schema{
			Title: "Event " + e.Name,
			Type:  huma.TypeObject,
			Properties: map[string]*huma.Schema{
				"id":    {Type: huma.TypeInteger, Description: "The event ID."},
				"event": {Type: huma.TypeString, Description: "The event name.", Extensions: map[string]any{"const": e.Name}},
				"data":  api.OpenAPI().Components.Schemas.Schema(t, true, e.Name),
				"retry": {Type: huma.TypeInteger, Description: "The retry time in milliseconds."},
			},
			Required: []string{"data", "event"},
		})
	}
	slices.SortFunc(branches, func(a, b *huma.Schema) int { return strings.Compare(a.Title, b.Title) })
	op.Responses["200"] = &huma.Response{Content: map[string]*huma.MediaType{"text/event-stream": {Schema: &huma.Schema{
		Title:       "Server Sent Events",
		Description: "Each oneOf object in the array represents one possible Server Sent Events (SSE) message, serialized as UTF-8 text according to the SSE specification.",
		Type:        huma.TypeArray,
		Items:       &huma.Schema{Extensions: map[string]any{"oneOf": branches}},
	}}}}
	return names
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// stream commits the 200 and runs the stream until its request or the
// Router's streams end.
func (r *Router) stream(hc huma.Context, op Op, names map[reflect.Type]string, run StreamFunc) {
	req, w := humago.Unwrap(hc)
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.Flush()
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	stop := context.AfterFunc(r.streams, cancel)
	defer stop()
	defer func() {
		if v := recover(); v != nil {
			r.logPanic(op, v)
		}
	}()
	send := func(m StreamMessage) error { return writeMessage(w, rc, names, m) }
	if err := run(ctx, send); err != nil {
		r.logFailure(ctx, "operation failed", "method", op.Method, "path", Prefix+op.Path, "operation", op.ID, "error", err)
	}
}

// writeMessage writes m as the SSE wire spells it and flushes it.
func writeMessage(w http.ResponseWriter, rc *http.ResponseController, names map[reflect.Type]string, m StreamMessage) error {
	if err := rc.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("set the write deadline: %w", err)
	}
	var b strings.Builder
	if m.ID > 0 {
		fmt.Fprintf(&b, "id: %d\n", m.ID)
	}
	if m.Retry > 0 {
		fmt.Fprintf(&b, "retry: %d\n", m.Retry.Milliseconds())
	}
	if m.Comment != "" {
		for line := range strings.SplitSeq(strings.ReplaceAll(strings.ReplaceAll(m.Comment, "\r\n", "\n"), "\r", "\n"), "\n") {
			fmt.Fprintf(&b, ": %s\n", line)
		}
	}
	if m.Data != nil {
		name, ok := names[deref(reflect.TypeOf(m.Data))]
		if !ok {
			return fmt.Errorf("api: %T is no event of this stream", m.Data)
		}
		data, err := json.Marshal(m.Data)
		if err != nil {
			return fmt.Errorf("api: encode %T: %w", m.Data, err)
		}
		fmt.Fprintf(&b, "event: %s\ndata: %s\n", name, data)
	}
	b.WriteString("\n")
	if _, err := w.Write([]byte(b.String())); err != nil {
		return fmt.Errorf("write an event: %w", err)
	}
	if err := rc.Flush(); err != nil {
		return fmt.Errorf("flush an event: %w", err)
	}
	return nil
}

// EndStreams ends every open stream's context, and any stream's that opens
// later. The Node calls it as it stops, so no stream holds the stop.
func (r *Router) EndStreams() { r.endStreams() }
