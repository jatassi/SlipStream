package notifying_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/notifying"
	notifyv1 "github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/notify/v1/notifyv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

var self = role.NodeID{UUID: storage.NewUUID()}

// alone is a Client calling impl on this Node, as the Controller calls the
// Notifying Role on itself.
func alone(impl notifyv1connect.NotifyingServiceHandler) notifying.Client {
	d, cluster := rpc.Alone(self, func(role.Name) bool { return true })
	notifyv1connect.NotifyingService.Mount(d, impl)
	return notifying.Client{Cluster: cluster}
}

func serving() notifying.Client {
	return alone(notifying.Serve(mail.DefaultTimings(), (&net.Dialer{}).DialContext, nil))
}

func postTo(t *testing.T, c notifying.Client, url string) notifying.Response {
	t.Helper()
	res, err := c.Post(t.Context(), self, notifying.Request{Method: http.MethodPost, URL: url, Header: http.Header{"Content-Type": {"application/json"}}, Body: []byte(`{}`), Timeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestAPostFollowsNoRedirect(t *testing.T) {
	followed := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/elsewhere" {
			followed = true
			return
		}
		http.Redirect(w, r, "/elsewhere", http.StatusTemporaryRedirect)
	}))
	defer srv.Close()
	res := postTo(t, serving(), srv.URL+"/hook")
	if res.Status != http.StatusTemporaryRedirect || followed {
		t.Fatalf("Post = %d, followed %v; want the 307 itself", res.Status, followed)
	}
}

func TestAPostAnswersTheFirst4KiBOfALargeBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, strings.Repeat("x", notifying.MaxBody)+strings.Repeat("y", 5<<20))
	}))
	defer srv.Close()
	res := postTo(t, serving(), srv.URL)
	if res.Status != http.StatusBadRequest || string(res.Body) != strings.Repeat("x", notifying.MaxBody) {
		t.Fatalf("Post = %d with a %d-byte body, want 400 with the first %d bytes", res.Status, len(res.Body), notifying.MaxBody)
	}
}

func TestAPostAnswersOnlyTheHeadersAboutWaiting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("the request's Content-Type = %q", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Retry-After", "30")
		w.Header().Set("X-RateLimit-Reset-After", "1.5")
		w.Header().Set("X-Limit-App-Remaining", "9000")
		w.Header().Set("Set-Cookie", "session=secret")
		w.Header().Set("Server", "nginx")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	res := postTo(t, serving(), srv.URL)
	want := http.Header{"Retry-After": {"30"}, "X-Ratelimit-Reset-After": {"1.5"}, "X-Limit-App-Remaining": {"9000"}}
	if res.Status != http.StatusTooManyRequests || len(res.Header) != len(want) {
		t.Fatalf("Post = %d with headers %v, want %v", res.Status, res.Header, want)
	}
	for name, v := range want {
		if res.Header.Get(name) != v[0] {
			t.Errorf("header %s = %q, want %q", name, res.Header.Get(name), v[0])
		}
	}
}

func TestAPostErrorCarriesNoURL(t *testing.T) {
	l, err := new(net.ListenConfig).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := l.Addr().String()
	_ = l.Close()
	for _, url := range []string{"http://" + closed + "/bot123:SECRET/sendMessage", "http://" + closed + "/bot123:SECRET/%zz"} {
		res := postTo(t, serving(), url)
		if res.Status != 0 || res.Err == "" || strings.Contains(res.Err, "SECRET") {
			t.Errorf("Post %s = %+v, want an error without the token", url, res)
		}
	}
}

// answers is a Node answering whatever the test sets.
type answers struct {
	notifyv1connect.UnimplementedNotifyingServiceHandler
	post *notifyv1.PostResponse
	run  *notifyv1.RunResponse
	send *notifyv1.SendResponse
}

func (a answers) Post(context.Context, *notifyv1.PostRequest) (*notifyv1.PostResponse, error) {
	return a.post, nil
}

func (a answers) Run(context.Context, *notifyv1.RunRequest) (*notifyv1.RunResponse, error) {
	return a.run, nil
}

func (a answers) Send(context.Context, *notifyv1.SendRequest) (*notifyv1.SendResponse, error) {
	return a.send, nil
}

func TestTheClientRefusesAnAnswerOverACapOrMissingAField(t *testing.T) {
	cases := []struct {
		name string
		node answers
	}{
		{"a body over 4 KiB", answers{post: &notifyv1.PostResponse{Status: 200, Body: make([]byte, notifying.MaxBody+1)}}},
		{"a status under 100", answers{post: &notifyv1.PostResponse{Status: 99}}},
		{"a status over 599", answers{post: &notifyv1.PostResponse{Status: 600}}},
		{"no status and no error", answers{post: &notifyv1.PostResponse{}}},
		{"a header not about waiting", answers{post: &notifyv1.PostResponse{Status: 200, Headers: []*notifyv1.Header{{Name: "Set-Cookie", Value: "x"}}}}},
		{"a tail over 1 KiB", answers{run: &notifyv1.RunResponse{Tail: make([]byte, notifying.MaxTail+1)}}},
		{"no outcome", answers{send: &notifyv1.SendResponse{}}},
		{"an unknown step", answers{send: &notifyv1.SendResponse{Kind: "failed", Step: "helo"}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			client := alone(c.node)
			var err error
			switch {
			case c.node.post != nil:
				_, err = client.Post(t.Context(), self, notifying.Request{Method: http.MethodPost, URL: "https://example.com"})
			case c.node.run != nil:
				_, err = client.Run(t.Context(), self, notifying.Script{Path: "/opt/notify.sh"})
			default:
				_, err = client.Send(t.Context(), self, mail.Server{}, mail.Message{})
			}
			if !errors.Is(err, notifying.ErrUnchecked) {
				t.Fatalf("err = %v, want ErrUnchecked", err)
			}
		})
	}
}

func TestTheClientTakesAnAnswerWithinItsCaps(t *testing.T) {
	client := alone(answers{
		post: &notifyv1.PostResponse{Status: 429, Headers: []*notifyv1.Header{{Name: "Retry-After", Value: "5"}}, Body: make([]byte, notifying.MaxBody)},
		run:  &notifyv1.RunResponse{Exit: 3, Tail: make([]byte, notifying.MaxTail)},
		send: &notifyv1.SendResponse{Kind: "failed", Reply: "550 5.1.1 Mailbox unavailable", Permanent: true, Step: "rcpt"},
	})
	res, err := client.Post(t.Context(), self, notifying.Request{Method: http.MethodPost, URL: "https://example.com"})
	if err != nil || res.Status != 429 || res.Header.Get("Retry-After") != "5" {
		t.Errorf("Post = %+v, %v", res, err)
	}
	if ran, err := client.Run(t.Context(), self, notifying.Script{Path: "/opt/notify.sh"}); err != nil || ran.Exit != 3 {
		t.Errorf("Run = %+v, %v", ran, err)
	}
	out, err := client.Send(t.Context(), self, mail.Server{}, mail.Message{})
	if want := (mail.Outcome{Kind: mail.Failed, Reply: "550 5.1.1 Mailbox unavailable", Permanent: true, Step: mail.StepRcpt}); err != nil || out != want {
		t.Errorf("Send = %+v, %v; want %+v", out, err, want)
	}
}
