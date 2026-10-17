package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/identity"
)

type fixtureTick struct {
	N int `json:"n"`
}

var fixtureTickEvent = StreamEvent{Name: "fixture.tick", Payload: fixtureTick{}}

type goneInput struct {
	Gone bool `query:"gone"`
}

func streamFixture(r *Router, run StreamFunc) http.Handler {
	RegisterStream(r.Owner("fixture", Public), Public, Op{ID: "streamFixture", Method: http.MethodGet, Path: "/fixture", Status: http.StatusOK, Errors: []*Code{fixtureGone}},
		[]StreamEvent{fixtureTickEvent}, func(_ context.Context, _ identity.Anyone, in *goneInput) (StreamFunc, error) {
			if in.Gone {
				return nil, fixtureGone
			}
			return run, nil
		})
	return r.Handler()
}

func ticks(_ context.Context, send Send) error {
	if err := send(StreamMessage{ID: 7, Data: fixtureTick{N: 1}, Retry: 3 * time.Second}); err != nil {
		return err
	}
	return send(StreamMessage{Comment: "keepalive"})
}

// openStream opens target on a real connection and returns the response's
// Content-Type and a read of its whole body.
func openStream(t *testing.T, h http.Handler, target string) (contentType string, read func() string) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+target, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res.Header.Get("Content-Type"), func() string {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
}

func TestAStreamWritesEachMessageAsAnEvent(t *testing.T) {
	contentType, body := openStream(t, streamFixture(newRouter(t, &bytes.Buffer{}), ticks), "/api/v1/fixture")
	got := contentType + "\n" + body()
	if want := "text/event-stream\nid: 7\nretry: 3000\nevent: fixture.tick\ndata: {\"n\":1}\n\n: keepalive\n\n"; got != want {
		t.Errorf("stream = %q, want %q", got, want)
	}
}

func TestAStreamAnswersAProblemBeforeItOpens(t *testing.T) {
	w := get(t, streamFixture(newRouter(t, &bytes.Buffer{}), ticks), "/api/v1/fixture?gone=true")
	want := `{"status":410,"code":"fixture_gone","title":"The fixture is gone"}` + "\n"
	if w.Code != http.StatusGone || w.Body.String() != want {
		t.Fatalf("got %d %q, want 410 %q", w.Code, w.Body, want)
	}
}

func TestEndStreamsEndsOpenAndLaterStreams(t *testing.T) {
	r := newRouter(t, &bytes.Buffer{})
	ended := make(chan struct{}, 2)
	h := streamFixture(r, func(ctx context.Context, send Send) error {
		if err := send(StreamMessage{Comment: "open"}); err != nil {
			return err
		}
		<-ctx.Done()
		ended <- struct{}{}
		return nil
	})
	_, open := openStream(t, h, "/api/v1/fixture")
	r.EndStreams()
	_, later := openStream(t, h, "/api/v1/fixture")
	if a, b := open(), later(); a != ": open\n\n" || b != ": open\n\n" || len(ended) != 2 {
		t.Fatalf("streams read %q and %q, %d ended", a, b, len(ended))
	}
}

func TestAStreamWhoseClientLeftLogsAtDebug(t *testing.T) {
	var log bytes.Buffer
	r := newRouter(t, &log)
	r.services.Log = slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: noTime}))
	ctx, leave := context.WithCancel(t.Context())
	h := streamFixture(r, func(ctx context.Context, _ Send) error {
		leave()
		return fmt.Errorf("renew: %w", ctx.Err())
	})
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/fixture", http.NoBody))
	if want := "level=DEBUG msg=\"operation failed\" method=GET path=/api/v1/fixture operation=streamFixture error=\"renew: context canceled\"\n"; log.String() != want {
		t.Fatalf("log = %q, want %q", log.String(), want)
	}
}

func downloadFixture(r *Router, calls *int) http.Handler {
	RegisterDownload(r.Owner("fixture", Public), Public, Op{ID: "downloadFixture", Method: http.MethodGet, Path: "/fixture", Status: http.StatusOK, Errors: []*Code{fixtureGone}},
		"application/zip", func(_ context.Context, _ identity.Anyone, in *goneInput) (*Download, error) {
			*calls++
			if in.Gone {
				return nil, fixtureGone
			}
			return &Download{Name: "logs é.zip", Size: 5, Body: io.NopCloser(strings.NewReader("PK..."))}, nil
		})
	return r.Handler()
}

func TestADownloadAnswersItsFileAsAnAttachment(t *testing.T) {
	calls := 0
	h := downloadFixture(newRouter(t, &bytes.Buffer{}), &calls)
	w := get(t, h, "/api/v1/fixture")
	got := fmt.Sprintf("%d %s %s %s %q", w.Code, w.Header().Get("Content-Type"), w.Header().Get("Content-Length"), w.Header().Get("Content-Disposition"), w.Body)
	if want := `200 application/zip 5 attachment; filename*=utf-8''logs%20%C3%A9.zip "PK..."`; got != want {
		t.Fatalf("download = %s, want %s", got, want)
	}
	gone := get(t, h, "/api/v1/fixture?gone=true")
	if gone.Code != http.StatusGone || gone.Header().Get("Content-Type") != "application/problem+json" || calls != 2 {
		t.Fatalf("a refused download = %d %s after %d calls", gone.Code, gone.Header().Get("Content-Type"), calls)
	}
}

func TestAHeadOfADownloadNeverRunsItsHandler(t *testing.T) {
	calls := 0
	h := downloadFixture(newRouter(t, &bytes.Buffer{}), &calls)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodHead, "/api/v1/fixture", http.NoBody))
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "application/zip" || w.Body.Len() != 0 || calls != 0 {
		t.Fatalf("HEAD = %d %s with %d body bytes after %d handler calls", w.Code, w.Header().Get("Content-Type"), w.Body.Len(), calls)
	}
}
