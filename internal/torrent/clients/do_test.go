package clients_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/torrent/clients"
)

func TestDoLogsTheRoundTripButNeverItsPathHeadersOrBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"result":"torrent-secret-name"}`))
	}))
	defer srv.Close()
	var logs bytes.Buffer
	s := session(srv.Client(), &logs, time.Minute)
	req := request(t, srv.URL+"/transmission/rpc/private-path")
	req.Header.Set("Authorization", "Basic amFja3NvbjpodW50ZXIy")

	res, err := clients.Do(t.Context(), s, req, 1<<10)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusConflict || string(res.Body) != `{"result":"torrent-secret-name"}` {
		t.Errorf("Do = %d %q, want the 409 and its body for the kind to read", res.Status, res.Body)
	}
	line := logs.String()
	for _, want := range []string{`"method":"POST"`, `"host":"` + strings.TrimPrefix(srv.URL, "http://") + `"`, `"status":409`, `"took":`} {
		if !strings.Contains(line, want) {
			t.Errorf("log %s lacks %s", line, want)
		}
	}
	for _, never := range []string{"private-path", "transmission/rpc", "torrent-secret-name", "amFja3NvbjpodW50ZXIy"} {
		if strings.Contains(line, never) {
			t.Errorf("log %s holds %q", line, never)
		}
	}
}

func TestDoTurnsAServerErrorIntoAProblem(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, err := clients.Do(t.Context(), session(srv.Client(), nil, time.Minute), request(t, srv.URL), 1<<10)
	p := problem(t, err, clients.ServerError)
	if p.Status != http.StatusServiceUnavailable {
		t.Errorf("Status = %d, want 503", p.Status)
	}
}

func TestDoClassifiesTransportFailures(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	release := make(chan struct{})
	slow := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer slow.Close()
	defer close(release)
	tlsSrv := httptest.NewTLSServer(http.NotFoundHandler())
	defer tlsSrv.Close()

	cases := []struct {
		name string
		url  string
		call time.Duration
		want clients.Cause
	}{
		{"nothing listens", closed.URL, time.Minute, clients.NoResponse},
		{"no response in time", slow.URL, 50 * time.Millisecond, clients.TimedOut},
		{"a certificate the Node doesn't trust", tlsSrv.URL, time.Minute, clients.TLSFailed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var logs bytes.Buffer
			_, err := clients.Do(t.Context(), session(&http.Client{}, &logs, c.call), request(t, c.url+"/rpc"), 1<<10)
			_ = problem(t, err, c.want)
			if strings.Contains(logs.String(), "/rpc") {
				t.Errorf("log %s holds the path", logs.String())
			}
		})
	}
}

func TestDoAnswersTheCallersCancelAsAContextError(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(20*time.Millisecond, cancel)
	_, err := clients.Do(ctx, session(srv.Client(), nil, time.Minute), request(t, srv.URL), 1<<10)
	if _, ok := errors.AsType[*clients.Problem](err); ok || !errors.Is(err, context.Canceled) {
		t.Errorf("Do = %v, want context.Canceled and no Problem", err)
	}
}

func TestDoRefusesABodyOverTheLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 11))
	}))
	defer srv.Close()
	_, err := clients.Do(t.Context(), session(srv.Client(), nil, time.Minute), request(t, srv.URL), 10)
	if _, ok := errors.AsType[*clients.Problem](err); ok || err == nil {
		t.Errorf("Do = %v, want a plain error", err)
	}
}

func session(c *http.Client, logs *bytes.Buffer, call time.Duration) clients.Session {
	if logs == nil {
		logs = &bytes.Buffer{}
	}
	return clients.Session{HTTP: c, Call: call, Log: slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}
}

func request(t *testing.T, url string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(`{"method":"torrent_get"}`))
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func problem(t *testing.T, err error, want clients.Cause) *clients.Problem {
	t.Helper()
	p, ok := errors.AsType[*clients.Problem](err)
	if !ok || p.Cause != want {
		t.Fatalf("Do = %v, want a %s Problem", err, want)
	}
	return p
}
