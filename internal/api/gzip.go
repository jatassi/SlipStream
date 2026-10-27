package api

import (
	"compress/gzip"
	"io"
	"strconv"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// humaContext names the embedded context apart from its Context method.
type humaContext = huma.Context

// gzipped is a request's context whose body goes through w.
type gzipped struct {
	humaContext
	w *gzip.Writer
}

func (g gzipped) BodyWriter() io.Writer { return g.w }

// gzipMiddleware compresses the response at gzip's fastest level when the
// request accepts gzip. The Library's 20,000 rows are about 15 MB of JSON
// and about 2 MB compressed, which costs about 55 ms to compress.
func gzipMiddleware(r *Router) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		ctx.AppendHeader("Vary", "Accept-Encoding")
		if !acceptsGzip(ctx.Header("Accept-Encoding")) {
			next(ctx)
			return
		}
		ctx.SetHeader("Content-Encoding", "gzip")
		w, err := gzip.NewWriterLevel(ctx.BodyWriter(), gzip.BestSpeed)
		if err != nil {
			panic(err)
		}
		next(gzipped{humaContext: ctx, w: w})
		if err := w.Close(); err != nil {
			r.logFailure(ctx.Context(), "response not sent", "operation", ctx.Operation().OperationID, "error", err)
		}
	}
}

// acceptsGzip reads an Accept-Encoding header: gzip is accepted unless it
// is absent or given q=0.
func acceptsGzip(header string) bool {
	for part := range strings.SplitSeq(header, ",") {
		coding, params, _ := strings.Cut(part, ";")
		if !strings.EqualFold(strings.TrimSpace(coding), "gzip") {
			continue
		}
		q, given := strings.CutPrefix(strings.ReplaceAll(params, " ", ""), "q=")
		if !given {
			return true
		}
		weight, err := strconv.ParseFloat(q, 64)
		return err != nil || weight > 0
	}
	return false
}
