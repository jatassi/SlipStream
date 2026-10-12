package liveapi_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/api/apitest"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/identityapi"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/liveapi"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

type node struct {
	t      *testing.T
	lib    *storage.DB[storage.Library]
	client *http.Client
}

// serve runs identity's operations and the stream on an in-memory listener,
// with a stream heartbeat of heartbeat.
func serve(t *testing.T, heartbeat time.Duration) *node {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema())
	keys, err := secret.Open(ctx, lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	hub := live.NewHub([]live.Event{identity.SessionsChangedEvent})
	svc, err := identity.New(ctx, lib, keys.PINPepper(), identity.SessionsChangedEvent.Bind(hub), hub)
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(t.Output(), nil))
	gate, _, err := identity.NewGate(ctx, svc.HasAdmin, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Identity: svc, Gate: gate, Web: fstest.MapFS{}, Log: log})
	identityapi.Routes(r, svc)
	liveapi.Routes(r, hub, svc, live.Timings{Heartbeat: heartbeat, Retry: 3 * time.Second})
	ln := apitest.NewListener(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8484})
	srv := &http.Server{Handler: r.Handler(), ReadHeaderTimeout: time.Minute}
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()
	t.Cleanup(func() {
		hub.Close()
		if err := srv.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
		if err := <-served; !errors.Is(err, http.ErrServerClosed) {
			t.Error(err)
		}
	})
	return &node{t: t, lib: lib, client: ln.Client()}
}

func (n *node) get(path, session string) *http.Response {
	n.t.Helper()
	req, err := http.NewRequestWithContext(n.t.Context(), http.MethodGet, "http://slipstream"+path, http.NoBody)
	if err != nil {
		n.t.Fatal(err)
	}
	req.Header.Set("Cookie", identity.CookieName+"="+session)
	res, err := n.client.Do(req)
	if err != nil {
		n.t.Fatal(err)
	}
	return res
}

// stream reads its messages as a browser does, as they arrive: a net.Pipe
// has no buffer, so a write the client doesn't read would miss its deadline.
type stream struct {
	t        *testing.T
	body     io.ReadCloser
	messages chan string
	end      chan error
}

func (n *node) stream(session string) *stream {
	n.t.Helper()
	res := n.get("/api/v1/events", session)
	if res.StatusCode != http.StatusOK {
		n.t.Fatalf("GET /api/v1/events = %d", res.StatusCode)
	}
	s := &stream{t: n.t, body: res.Body, messages: make(chan string, 1024), end: make(chan error, 1)}
	go func() {
		r := bufio.NewReader(res.Body)
		for {
			msg, err := apitest.ReadMessage(r)
			if err != nil {
				s.end <- err
				close(s.messages)
				return
			}
			s.messages <- msg
		}
	}()
	n.t.Cleanup(func() { _ = res.Body.Close() })
	return s
}

func (s *stream) next() string {
	s.t.Helper()
	msg, ok := <-s.messages
	if !ok {
		s.t.Fatalf("the stream ended: %v", <-s.end)
	}
	return msg
}

func (s *stream) ended() {
	s.t.Helper()
	if msg, ok := <-s.messages; ok {
		s.t.Fatalf("read %q, want the stream to end", msg)
	}
	if err := <-s.end; !errors.Is(err, io.EOF) {
		s.t.Fatalf("the stream ended with %v, want EOF", err)
	}
}

func TestAQuietStreamSendsKeepalivesAndCountsAsUse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := serve(t, 2*time.Hour)
		session := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
		s := n.stream(session)
		if got := s.next(); got != "id: 946684800000000\nretry: 3000\nevent: live.resync\ndata: {}\n" {
			t.Fatalf("first message = %q", got)
		}
		opened := time.Now()
		if got := s.next(); got != ": keepalive\n" || time.Since(opened) != 2*time.Hour {
			t.Fatalf("after %v read %q, want a keepalive after 2h0m0s", time.Since(opened), got)
		}
		time.Sleep(31 * 24 * time.Hour)
		if res := n.get("/api/v1/session", session); res.StatusCode != http.StatusOK {
			t.Fatalf("after 31 days with a stream open, getSession = %d, want 200", res.StatusCode)
		}
		_ = s.body.Close()
	})
}

func TestAStreamWhoseSessionExpiresEndsWithExpired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := serve(t, 30*24*time.Hour+time.Second)
		session := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
		s := n.stream(session)
		s.next()
		if got, want := s.next(), "event: live.session-ended\ndata: {\"reason\":\"expired\"}\n"; got != want {
			t.Fatalf("read %q, want %q", got, want)
		}
		s.ended()
		if res := n.get("/api/v1/session", session); res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("getSession after expiry = %d, want 401", res.StatusCode)
		}
	})
}
