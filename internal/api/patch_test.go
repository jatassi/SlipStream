package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/identity"
)

type fixtureSpan struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

type fixturePatch struct {
	Body struct {
		Span  Patch[fixtureSpan] `json:"span,omitempty"`
		Count Patch[int64]       `json:"count,omitempty"`
	}
}

type fixtureSaid struct {
	Body struct {
		Said string `json:"said"`
	}
}

func describe[T any](name string, p Patch[T]) string {
	switch {
	case !p.Sent():
		return name + " absent"
	case p.Value() == nil:
		return name + " null"
	}
	return fmt.Sprintf("%s %v", name, *p.Value())
}

func TestAPatchMemberIsAbsentNullOrAValue(t *testing.T) {
	r := newRouter(t, &bytes.Buffer{})
	Register(r.Owner("fixture", Public), Public, Op{ID: "updateFixture", Method: http.MethodPatch, Path: "/fixture"},
		func(_ context.Context, _ identity.Anyone, in *fixturePatch) (*fixtureSaid, error) {
			out := &fixtureSaid{}
			out.Body.Said = describe("span", in.Body.Span) + ", " + describe("count", in.Body.Count)
			return out, nil
		})
	h := r.Handler()
	for _, tc := range []struct{ body, want string }{
		{`{}`, `{"said":"span absent, count absent"}`},
		{`{"span":null,"count":3}`, `{"said":"span null, count 3"}`},
		{`{"span":{"start":"01:00","end":"03:00"},"count":null}`, `{"said":"span {01:00 03:00}, count null"}`},
		{`{"count":"three"}`, `{"status":422,"code":"invalid","title":"Some fields are invalid","errors":[{"location":"body.count","message":"expected integer"}]}`},
	} {
		t.Run(tc.body, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/api/v1/fixture", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			if w.Body.String() != tc.want+"\n" {
				t.Fatalf("PATCH %s = %d %s, want %s", tc.body, w.Code, w.Body, tc.want)
			}
		})
	}
}
