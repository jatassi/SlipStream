package api

import (
	"context"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Download is a file a GET answers, saved under Name.
type Download struct {
	Name string
	Size int64
	Body io.ReadCloser
}

// RegisterDownload adds a GET answering a file of mediaType, under the
// same checks as Register. A handler error answers a problem, since
// nothing has been written yet. A HEAD passes the access check and
// answers the 200's headers without running handler.
func RegisterDownload[P, I any](o *Owner, level Level[P], op Op, mediaType string, handler func(context.Context, P, *I) (*Download, error)) {
	r := o.router
	hop, declared := humaOperation(o, level, op, reflect.TypeFor[I]())
	hop.Responses["200"] = &huma.Response{
		Description: http.StatusText(http.StatusOK),
		Content:     map[string]*huma.MediaType{mediaType: {Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"}}},
	}
	hop.Middlewares = append(hop.Middlewares, func(hc huma.Context, next func(huma.Context)) {
		if hc.Method() != http.MethodHead {
			next(hc)
			return
		}
		hc.SetHeader("Content-Type", mediaType)
		hc.SetStatus(http.StatusOK)
	})
	huma.Register(r.huma, hop, func(ctx context.Context, in *I) (out *huma.StreamResponse, err error) {
		p := principal[P](ctx, op)
		defer func() {
			if v := recover(); v != nil {
				r.logPanic(op, v)
				out, err = nil, Internal.problem()
			}
		}()
		d, err := handler(ctx, p, in)
		if err != nil {
			return nil, r.failure(ctx, op, declared, err)
		}
		return &huma.StreamResponse{Body: func(hc huma.Context) { r.download(hc, op, mediaType, d) }}, nil
	})
}

func (r *Router) download(hc huma.Context, op Op, mediaType string, d *Download) {
	req, w := humago.Unwrap(hc)
	defer func() { _ = d.Body.Close() }()
	h := w.Header()
	h.Set("Content-Type", mediaType)
	h.Set("Content-Length", strconv.FormatInt(d.Size, 10))
	h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": d.Name}))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, d.Body); err != nil {
		r.logFailure(req.Context(), "operation failed", "method", op.Method, "path", Prefix+op.Path, "operation", op.ID, "error", err)
	}
}
