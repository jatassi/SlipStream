package liveapi_test

import (
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
	gate, _, err := identity.NewGate(ctx, identitytest.Node{Service: svc}.Claim, log)
	if err != nil {
		t.Fatal(err)
	}
	r := api.NewRouter(api.Services{Node: identitytest.Controller{Node: identitytest.Node{Service: svc}}, Gate: gate, Web: fstest.MapFS{}, Log: log})
	identityapi.Routes(r, identitytest.Node{Service: svc})
	liveapi.Routes(r, hub, identitytest.Node{Service: svc}, live.Timings{Heartbeat: heartbeat, Retry: 3 * time.Second})
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

func (n *node) status(path, session string) int {
	n.t.Helper()
	res := n.get(path, session)
	_ = res.Body.Close()
	return res.StatusCode
}

func (n *node) stream(session string) *apitest.Stream {
	n.t.Helper()
	res := n.get("/api/v1/events", session) //nolint:bodyclose // the Stream closes it
	if res.StatusCode != http.StatusOK {
		n.t.Fatalf("GET /api/v1/events = %d", res.StatusCode)
	}
	s := apitest.NewStream(res.Body)
	n.t.Cleanup(func() { _ = s.Close() })
	return s
}

func next(t *testing.T, s *apitest.Stream) string {
	t.Helper()
	msg, err := s.Next()
	if err != nil {
		t.Fatalf("the stream ended: %v", err)
	}
	return msg
}

func TestAQuietStreamSendsKeepalivesAndCountsAsUse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := serve(t, 2*time.Hour)
		session := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
		s := n.stream(session)
		if got := next(t, s); got != "id: 946684800000000\nretry: 3000\nevent: live.resync\ndata: {}\n" {
			t.Fatalf("first message = %q", got)
		}
		opened := time.Now()
		if got := next(t, s); got != ": keepalive\n" || time.Since(opened) != 2*time.Hour {
			t.Fatalf("after %v read %q, want a keepalive after 2h0m0s", time.Since(opened), got)
		}
		time.Sleep(31 * 24 * time.Hour)
		if status := n.status("/api/v1/session", session); status != http.StatusOK {
			t.Fatalf("after 31 days with a stream open, getSession = %d, want 200", status)
		}
	})
}

func TestAStreamWhoseSessionExpiresEndsWithExpired(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := serve(t, 30*24*time.Hour+time.Second)
		session := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
		s := n.stream(session)
		next(t, s)
		if got, want := next(t, s), "event: live.session-ended\ndata: {\"reason\":\"expired\"}\n"; got != want {
			t.Fatalf("read %q, want %q", got, want)
		}
		if msg, err := s.Next(); !errors.Is(err, io.EOF) {
			t.Fatalf("read %q, %v; want the stream to end", msg, err)
		}
		if status := n.status("/api/v1/session", session); status != http.StatusUnauthorized {
			t.Fatalf("getSession after expiry = %d, want 401", status)
		}
	})
}

func TestASignOutBetweenAuthenticationAndSubscribeEndsTheStream(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := serve(t, 2*time.Hour)
		session := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
		time.Sleep(2 * time.Hour)
		held, release, signedOut := make(chan struct{}), make(chan struct{}), make(chan error, 1)
		go func() {
			signedOut <- n.lib.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
				close(held)
				<-release
				_, err := tx.ExecContext(ctx, "DELETE FROM identity_session")
				return err
			})
		}()
		<-held
		opened := make(chan *http.Response, 1)
		go func() {
			req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://slipstream/api/v1/events", http.NoBody)
			req.Header.Set("Cookie", identity.CookieName+"="+session)
			res, _ := n.client.Do(req) //nolint:bodyclose // the Stream closes it
			opened <- res
		}()
		synctest.Wait() // the stream's authentication waits on the writer to slide its session
		close(release)
		if err := <-signedOut; err != nil {
			t.Fatal(err)
		}
		res := <-opened
		if res == nil || res.StatusCode != http.StatusOK {
			t.Fatalf("GET /api/v1/events = %v, want 200", res)
		}
		s := apitest.NewStream(res.Body)
		t.Cleanup(func() { _ = s.Close() })
		if got, want := next(t, s), "retry: 3000\nevent: live.session-ended\ndata: {\"reason\":\"signed-out\"}\n"; got != want {
			t.Fatalf("read %q, want %q", got, want)
		}
		if msg, err := s.Next(); !errors.Is(err, io.EOF) {
			t.Fatalf("read %q, %v; want the stream to end", msg, err)
		}
	})
}

func TestASlowHeartbeatDelaysTheNextOne(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		n := serve(t, 2*time.Hour)
		session := identitytest.Session(t, n.lib, identitytest.Member(t, n.lib, "member@example.com"))
		s := n.stream(session)
		next(t, s)
		opened := time.Now()
		time.Sleep(time.Hour)
		// Holding the writer until 5h makes the 2h heartbeat's slide wait.
		go func() {
			_ = n.lib.Write(t.Context(), func(context.Context, *storage.Tx[storage.Library]) error {
				time.Sleep(4 * time.Hour)
				return nil
			})
		}()
		for _, want := range []time.Duration{5 * time.Hour, 7 * time.Hour} {
			if got := next(t, s); got != ": keepalive\n" || time.Since(opened) != want {
				t.Fatalf("after %v read %q, want a keepalive after %v", time.Since(opened), got, want)
			}
		}
	})
}
