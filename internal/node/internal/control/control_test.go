package control_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jatassi/slipstream/internal/node/internal/control"
)

func serve(t *testing.T, handlers map[string]control.Handler) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ctl") //nolint:usetesting // t.TempDir's path is too long for the socket limit on macOS
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "control.sock")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := control.Listen(t.Context(), new(net.ListenConfig).Listen, path, func(op string) (control.Handler, error) {
		if h, ok := handlers[op]; ok {
			return h, nil
		}
		return nil, &control.Error{Code: control.CodeUnknownOp, Message: op}
	})
	if err != nil {
		t.Fatalf("listen over a stale socket file: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
		s.Wait()
	})
	return path
}

func raw(t *testing.T, path, request string) string {
	t.Helper()
	out, err := rawErr(t, path, request)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func rawErr(t *testing.T, path, request string) (string, error) {
	t.Helper()
	var d net.Dialer
	conn, err := d.DialContext(t.Context(), "unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := io.WriteString(conn, request); err != nil {
		return "", fmt.Errorf("write request: %w", err)
	}
	out, err := io.ReadAll(conn)
	if err != nil {
		return string(out), fmt.Errorf("read reply: %w", err)
	}
	return string(out), nil
}

func TestOneRequestManyReplies(t *testing.T) {
	path := serve(t, map[string]control.Handler{
		"count": func(_ context.Context, body json.RawMessage, send func(any) error) error {
			var req struct{ To int }
			if err := control.Decode(body, &req); err != nil {
				return err
			}
			for i := 1; i <= req.To; i++ {
				if err := send(i); err != nil {
					return err
				}
			}
			return errors.New("ran out")
		},
	})
	c, err := control.Dial(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if err := c.Send("count", map[string]int{"To": 2}); err != nil {
		t.Fatal(err)
	}
	var got []int
	for {
		var n int
		err := c.Recv(&n)
		if err != nil {
			want := &control.Error{Code: control.CodeFailed, Message: "ran out"}
			if e, ok := errors.AsType[*control.Error](err); !ok || *e != *want {
				t.Fatalf("last reply = %v, want %v", err, want)
			}
			break
		}
		got = append(got, n)
	}
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("replies = %v, want [1 2]", got)
	}
	if err := c.Recv(new(int)); !errors.Is(err, io.EOF) {
		t.Fatalf("after the last reply: %v, want EOF", err)
	}
}

func TestMalformedRequestsAreBadRequests(t *testing.T) {
	path := serve(t, map[string]control.Handler{
		"echo": func(_ context.Context, body json.RawMessage, send func(any) error) error {
			var req struct{ Text string }
			if err := control.Decode(body, &req); err != nil {
				return err
			}
			return send(req.Text)
		},
	})
	for _, c := range []struct{ request, want string }{
		{`{"op":"echo","body":{"Text":"hi"}}` + "\n", `{"body":"hi"}` + "\n"},
		{"not json\n", `{"error":{"code":"bad_request","message":"invalid character 'o' in literal null (expecting 'u')"}}` + "\n"},
		{`{"op":"echo","body":{"Txt":"hi"}}` + "\n", `{"error":{"code":"bad_request","message":"json: unknown field \"Txt\""}}` + "\n"},
		{`{"op":"nope"}` + "\n", `{"error":{"code":"unknown_op","message":"nope"}}` + "\n"},
	} {
		if got := raw(t, path, c.request); got != c.want {
			t.Errorf("request %.40q: reply %q, want %q", c.request, got, c.want)
		}
	}
}

// The Node stops reading an oversized request, so the client may see its
// connection reset instead of the bad_request reply.
func TestOversizedRequestNeverReachesTheHandler(t *testing.T) {
	called := make(chan struct{}, 1)
	path := serve(t, map[string]control.Handler{
		"echo": func(context.Context, json.RawMessage, func(any) error) error {
			called <- struct{}{}
			return nil
		},
	})
	out, err := rawErr(t, path, `{"op":"echo","body":"`+strings.Repeat("x", 1<<20)+`"}`+"\n")
	want := `{"error":{"code":"bad_request","message":"line exceeds 1048576 bytes"}}` + "\n"
	if err == nil && out != want {
		t.Fatalf("reply %q, want %q or a reset", out, want)
	}
	select {
	case <-called:
		t.Fatal("handler ran for an oversized request")
	default:
	}
}
