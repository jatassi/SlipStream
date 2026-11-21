package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// LifecycleBusy is a restore, an update or a Handover while another one
// runs, even across a restart.
var LifecycleBusy = NewCode(http.StatusConflict, "lifecycle_busy", "A restore or an update is already running. Try again once it finishes.")

// uploadIdle is how long an upload may send nothing before it is cut. It
// stands in for a body timeout, since a Backup of a large library takes
// minutes to send.
const uploadIdle = time.Minute

// TooLargeError is an upload that sent more than its operation's cap.
type TooLargeError struct {
	Max int64
	// Sent is the upload's Content-Length when it gave one, else Max+1.
	Sent int64
}

func (e *TooLargeError) Error() string {
	return fmt.Sprintf("the upload is %d bytes, over the %d it may send", e.Sent, e.Max)
}

type uploadBody struct{}

type uploadInput struct{}

// RegisterUpload adds an operation whose request body is a file streamed
// as application/octet-stream, outside Huma's buffered body. Its level is
// checked first, then max answers how many bytes it may take, and body
// fails with a *TooLargeError once the upload passes it. No body timeout
// applies: a read that waits longer than a minute fails instead.
func RegisterUpload[P, O any](o *Owner, level Level[P], op Op, max func(ctx context.Context) (int64, error), handler func(ctx context.Context, p P, body io.Reader) (*O, error)) {
	r := o.router
	hop, declared := humaOperation(o, level, op, reflect.TypeFor[uploadInput]())
	hop.MaxBodyBytes = 0
	hop.RequestBody = &huma.RequestBody{
		Required: true,
		Content: map[string]*huma.MediaType{"application/octet-stream": {
			Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"},
		}},
	}
	hop.Middlewares = append(hop.Middlewares, func(ctx huma.Context, next func(huma.Context)) {
		req, w := humago.Unwrap(ctx)
		next(huma.WithValue(ctx, uploadBody{}, &upload{req: req, rc: http.NewResponseController(w)}))
	})
	mustDocumentEmbedded(reflect.TypeFor[O](), map[reflect.Type]bool{})
	huma.Register(r.huma, hop, func(ctx context.Context, _ *uploadInput) (out *O, err error) {
		p := principal[P](ctx, op)
		u, ok := ctx.Value(uploadBody{}).(*upload)
		if !ok {
			panic("api: " + op.ID + " ran without its upload body")
		}
		defer func() {
			if v := recover(); v != nil {
				r.logPanic(op, v)
				out, err = nil, Internal.problem()
			}
		}()
		limit, err := max(ctx)
		if err == nil {
			u.max = limit
			if u.req.ContentLength > limit {
				err = &TooLargeError{Max: limit, Sent: u.req.ContentLength}
			}
		}
		if err == nil {
			out, err = handler(ctx, p, u)
		}
		if err != nil {
			return nil, r.failure(ctx, op, declared, err)
		}
		return out, nil
	})
}

// upload is the request body, cut at max bytes and after a read that waits
// uploadIdle.
type upload struct {
	req  *http.Request
	rc   *http.ResponseController
	max  int64
	read int64
}

func (u *upload) Read(b []byte) (int, error) {
	_ = u.rc.SetReadDeadline(time.Now().Add(uploadIdle))
	if room := u.max - u.read + 1; int64(len(b)) > room {
		b = b[:room]
	}
	n, err := u.req.Body.Read(b)
	u.read += int64(n)
	if u.read > u.max {
		sent := u.req.ContentLength
		if sent <= u.max {
			sent = u.max + 1
		}
		return n, &TooLargeError{Max: u.max, Sent: sent}
	}
	return n, err
}
