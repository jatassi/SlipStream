package api

import (
	"context"
	"net/http"
	"reflect"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Image is an image a GET answers inline.
type Image struct {
	Type string
	Body []byte
}

var imageTypes = []string{"image/png", "image/jpeg", "image/webp", "image/svg+xml"}

const (
	// immutable is the caching of an image whose address names its bytes
	// for good.
	immutable = "private, max-age=31536000, immutable"
	// svgPolicy lets an SVG draw and style itself and run nothing, even
	// opened on its own.
	svgPolicy = "default-src 'none'; style-src 'unsafe-inline'; sandbox"
)

// RegisterImage adds a GET answering an image inline, under the same
// checks as Register. The image's address must name its bytes for good:
// it is cached privately for a year as immutable. An SVG carries a
// Content-Security-Policy that runs no script. A handler error answers a
// problem, which no cache keeps.
func RegisterImage[P, I any](o *Owner, level Level[P], op Op, handler func(context.Context, P, *I) (*Image, error)) {
	r := o.router
	hop, declared := humaOperation(o, level, op, reflect.TypeFor[I]())
	content := map[string]*huma.MediaType{}
	for _, t := range imageTypes {
		content[t] = &huma.MediaType{Schema: &huma.Schema{Type: huma.TypeString, Format: "binary"}}
	}
	hop.Responses["200"] = &huma.Response{Description: http.StatusText(http.StatusOK), Content: content}
	huma.Register(r.huma, hop, func(ctx context.Context, in *I) (out *huma.StreamResponse, err error) {
		p := principal[P](ctx, op)
		defer func() {
			if v := recover(); v != nil {
				r.logPanic(op, v)
				out, err = nil, Internal.problem()
			}
		}()
		img, err := handler(ctx, p, in)
		if err != nil {
			return nil, r.failure(ctx, op, declared, err)
		}
		return &huma.StreamResponse{Body: func(hc huma.Context) { r.image(hc, op, img) }}, nil
	})
}

func (r *Router) image(hc huma.Context, op Op, img *Image) {
	req, w := humago.Unwrap(hc)
	h := w.Header()
	h.Set("Content-Type", img.Type)
	h.Set("Content-Length", strconv.Itoa(len(img.Body)))
	h.Set("Cache-Control", immutable)
	if img.Type == "image/svg+xml" {
		h.Set("Content-Security-Policy", svgPolicy)
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(img.Body); err != nil {
		r.logFailure(req.Context(), "operation failed", "method", op.Method, "path", Prefix+op.Path, "operation", op.ID, "error", err)
	}
}
