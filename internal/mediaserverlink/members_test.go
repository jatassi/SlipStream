package mediaserverlink_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	mslink "github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1/mediaserverv1connect"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// capture keeps every record at debug and above as text, attributes
// resolved.
type capture struct {
	mu    *sync.Mutex
	lines *[]string
	attrs []slog.Attr
}

func newCapture() capture { return capture{mu: &sync.Mutex{}, lines: &[]string{}} }

func (capture) Enabled(context.Context, slog.Level) bool { return true }

func (c capture) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	write := func(a slog.Attr) bool {
		fmt.Fprintf(&b, " %s=%v", a.Key, a.Value.Resolve())
		return true
	}
	for _, a := range c.attrs {
		write(a)
	}
	r.Attrs(write)
	c.mu.Lock()
	defer c.mu.Unlock()
	*c.lines = append(*c.lines, b.String())
	return nil
}

func (c capture) WithAttrs(as []slog.Attr) slog.Handler {
	c.attrs = append(append([]slog.Attr{}, c.attrs...), as...)
	return c
}

func (c capture) WithGroup(string) slog.Handler { return c }

func (c capture) holding(s string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for _, l := range *c.lines {
		if strings.Contains(l, s) {
			out = append(out, l)
		}
	}
	return out
}

// denKey answers Den's API key to its reaching Node.
type denKey struct{}

func (denKey) MediaServerSecrets(context.Context, *msv1.MediaServerSecretsRequest) (*msv1.MediaServerSecretsResponse, error) {
	return &msv1.MediaServerSecretsResponse{Credential: &msv1.Credential{ApiKey: "den-key"}, SettingsVersion: 1}, nil
}

type pairRoster map[role.NodeID]link.Key

func (r pairRoster) Key(node role.NodeID) (link.Key, bool) {
	k, ok := r[node]
	return k, ok
}

func (pairRoster) Serves(n role.Name, _ role.NodeID) bool { return n == role.MediaServerLink }

// TestMembersLogsNoPassword sets a Member's password on Den through a
// Worker's Members, over the link with every side logging at debug: the
// password signs in, and no log record holds it.
func TestMembersLogsNoPassword(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		kind, _ := mediaserverlink.Kinds.Of("jellyfin")
		f, _, admin := served(t, kind)
		logs := newCapture()
		log := slog.New(logs)

		ml := mediaserverlink.NewLink(mediaserverlink.Config{Kinds: mediaserverlink.Kinds, HTTP: f.Client, Timings: mslink.DefaultTimings(), Log: log})
		h := &holder{intents: make(chan *msv1.Intent), plays: map[string]int{}}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = ml.Run(storagetest.Local(t, role.LocalSchema()), denKey{})(ctx, h)
		}()
		t.Cleanup(func() {
			cancel()
			<-done
		})
		h.intents <- &msv1.Intent{SessionsEveryMs: 15_000, Servers: []*msv1.Server{{
			Id: den.Bytes(), Kind: "jellyfin", Name: "Den", Address: f.Address.URL(""), Identity: denServerID, SettingsVersion: 1,
		}}}
		synctest.Wait()

		cm, wm := http.NewServeMux(), http.NewServeMux()
		p := linktest.New(t, linktest.Side{Lockstep: cm, Log: log}, linktest.Side{Lockstep: wm, Log: log})
		holds := func(role.Name) bool { return true }
		controller := rpc.NewDispatch(p.Controller.Link, p.Controller.Key, p.Controller.Key, holds)
		worker := rpc.NewDispatch(p.Worker.Link, p.Worker.Key, p.Controller.Key, holds)
		cm.Handle("/", controller)
		wm.Handle("/", worker)
		mediaserverv1connect.MediaServerLinkService.Mount(worker, ml.Service())
		workerID := role.NodeID{UUID: storage.NewUUID()}
		p.Connect(t)
		members := mediaserverv1connect.MediaServerLinkService.On(rpc.NewCluster(controller, pairRoster{workerID: p.Worker.Key}), workerID)

		users := mediaserverlink.RemoteUsers(members, mslink.ServerRef{ID: den, SettingsVersion: 1})
		sam, err := users.CreateUser(t.Context(), "sam")
		if err != nil {
			t.Fatal(err)
		}
		const secret = "correct-horse-battery-staple"
		pw, err := mslink.NewPassword(secret)
		if err != nil {
			t.Fatal(err)
		}
		if err := users.SetPassword(t.Context(), sam, pw); err != nil {
			t.Fatal(err)
		}
		if _, err := admin.AuthenticateByName(t.Context(), "sam", secret); err != nil {
			t.Fatalf("sam signs in with the password: %v", err)
		}
		if leaked := logs.holding(secret); len(leaked) > 0 {
			t.Errorf("the password is in %d log records, first %q", len(leaked), leaked[0])
		}
	})
}
