package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/live"
)

type fixtureTick struct {
	N int `json:"n"`
}

var fixtureTickEvent = live.EveryoneEvent[fixtureTick]("fixture.tick")

func streamFixture(r *Router) http.Handler {
	RegisterStream(r.Owner("fixture", Public), Public, Op{ID: "streamFixture", Method: http.MethodGet, Path: "/fixture", Status: http.StatusOK},
		[]live.Event{fixtureTickEvent}, func(_ context.Context, _ identity.Anyone, _ *struct{}, send Send) error {
			if err := send(StreamMessage{ID: 7, Data: fixtureTick{N: 1}, Retry: 3 * time.Second}); err != nil {
				return err
			}
			return send(StreamMessage{Comment: "keepalive"})
		})
	return r.Handler()
}

const fixtureStream = "id: 7\nretry: 3000\nevent: fixture.tick\ndata: {\"n\":1}\n\n: keepalive\n\n"

// stderr runs f with os.Stderr on a pipe and returns what was written. f
// starts and stops any server itself, so its goroutines see the swap.
func stderr(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stderr
	os.Stderr = w
	f()
	os.Stderr = saved
	_ = w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestAStreamOnARealConnectionKeepsHumaQuiet(t *testing.T) {
	r := newRouter(t, &bytes.Buffer{})
	h := streamFixture(r)
	var body string
	printed := stderr(t, func() {
		srv := httptest.NewServer(h)
		defer srv.Close()
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/v1/fixture", http.NoBody)
		if err != nil {
			t.Fatal(err)
		}
		res, err := srv.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = res.Body.Close() }()
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		body = res.Header.Get("Content-Type") + "\n" + string(b)
	})
	if want := "text/event-stream\n" + fixtureStream; body != want {
		t.Errorf("stream = %q, want %q", body, want)
	}
	if printed != "" {
		t.Errorf("Huma's sse package printed %q", printed)
	}
}

// httptest.NewRecorder has no write deadline, so Huma's sse package warns
// on every message; only a real connection keeps it quiet.
func TestAStreamUnderARecorderWarnsOnEveryMessage(t *testing.T) {
	h := streamFixture(newRouter(t, &bytes.Buffer{}))
	w := httptest.NewRecorder()
	printed := stderr(t, func() {
		h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/fixture", http.NoBody))
	})
	if w.Body.String() != fixtureStream {
		t.Errorf("stream = %q, want %q", w.Body, fixtureStream)
	}
	warning := "warning: unable to set write deadline: feature not supported\n"
	if printed != warning+warning {
		t.Errorf("Huma's sse package printed %q, want the warning twice", printed)
	}
}

func TestAStreamWhoseClientLeftLogsAtDebug(t *testing.T) {
	var log bytes.Buffer
	r := newRouter(t, &log)
	r.services.Log = slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: noTime}))
	ctx, leave := context.WithCancel(t.Context())
	RegisterStream(r.Owner("fixture", Public), Public, Op{ID: "streamFixture", Method: http.MethodGet, Path: "/fixture", Status: http.StatusOK},
		[]live.Event{fixtureTickEvent}, func(ctx context.Context, _ identity.Anyone, _ *struct{}, _ Send) error {
			leave()
			return fmt.Errorf("renew: %w", ctx.Err())
		})
	r.Handler().ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/fixture", http.NoBody))
	if want := "level=DEBUG msg=\"operation failed\" method=GET path=/api/v1/fixture operation=streamFixture error=\"renew: context canceled\"\n"; log.String() != want {
		t.Fatalf("log = %q, want %q", log.String(), want)
	}
}
