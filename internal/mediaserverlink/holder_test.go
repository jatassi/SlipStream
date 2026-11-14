package mediaserverlink_test

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// fakeKind dials connections that record the credential they hold and
// pass on each look a test sends.
type fakeKind struct {
	mu     sync.Mutex
	dialed []string
	steps  []string
	seen   chan link.Seen
}

func (k *fakeKind) About() link.About        { return link.About{Name: "fake", Label: "Fake"} }
func (k *fakeKind) Plan(link.Due) []link.Job { return nil }

func (k *fakeKind) Dial(d link.Dialed) link.Conn {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.dialed = append(k.dialed, fmt.Sprintf("%s at %d", d.Credential.APIKey, d.Server.SettingsVersion))
	return fakeConn{k: k, cred: d.Credential}
}

func (k *fakeKind) record() (dialed, steps []string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	return slices.Clone(k.dialed), slices.Clone(k.steps)
}

type fakeConn struct {
	k    *fakeKind
	cred link.Credential
}

func (fakeConn) Reach(context.Context) (link.Identity, error) { return link.Identity{}, nil }
func (fakeConn) Check(context.Context) (link.Checked, error)  { return link.Checked{}, nil }
func (fakeConn) SignIn(context.Context, string, string) (link.Credential, error) {
	return link.Credential{}, nil
}

func (c fakeConn) Advance(_ context.Context, j link.Job) (link.Advanced, error) {
	c.k.mu.Lock()
	defer c.k.mu.Unlock()
	c.k.steps = append(c.k.steps, c.cred.APIKey)
	return link.Advanced{Job: j, Done: true}, nil
}

func (c fakeConn) Watch(ctx context.Context, _ time.Duration, seen func(link.Seen)) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case s := <-c.k.seen:
			seen(s)
		}
	}
}

// control answers each server's credential at the version the Controller
// holds, key- and the version.
type control struct {
	mu      sync.Mutex
	version int64
	fetches int
}

func (c *control) MediaServerSecrets(context.Context, *msv1.MediaServerSecretsRequest) (*msv1.MediaServerSecretsResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fetches++
	return &msv1.MediaServerSecretsResponse{Credential: &msv1.Credential{ApiKey: fmt.Sprintf("key-%d", c.version)}, SettingsVersion: c.version}, nil
}

func (c *control) set(version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.version = version
}

func (c *control) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fetches
}

// holder is the Role framework's side of Run: the intents a test sends,
// and the reports Run queues.
type holder struct {
	intents chan *msv1.Intent
	mu      sync.Mutex
	reports []string
	facts   *msv1.Facts
	plays   map[string]int
}

func (h *holder) Intent(ctx context.Context) iter.Seq[role.Snapshot[*msv1.Intent]] {
	return func(yield func(role.Snapshot[*msv1.Intent]) bool) {
		for {
			select {
			case <-ctx.Done():
				return
			case in := <-h.intents:
				if !yield(role.Snapshot[*msv1.Intent]{Intent: in, Standing: role.Live}) {
					return
				}
			}
		}
	}
}

func (h *holder) SetFacts(f *msv1.Facts) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.facts = f
}

// Report records r as the test reads it, each play named by the order it
// started in.
func (h *holder) Report(_ context.Context, _ *storage.Tx[storage.Local], r *msv1.Report) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	play := func(id []byte) int {
		if _, ok := h.plays[string(id)]; !ok {
			h.plays[string(id)] = len(h.plays) + 1
		}
		return h.plays[string(id)]
	}
	switch {
	case r.GetContact() != nil:
		c := r.GetContact()
		h.reports = append(h.reports, fmt.Sprintf("%s at %d", c.GetState(), c.GetSettingsVersion()))
	case r.GetPlayStarted() != nil:
		h.reports = append(h.reports, fmt.Sprintf("play %d started", play(r.GetPlayStarted().GetSession().GetPlayId())))
	case r.GetPlayEnded() != nil:
		h.reports = append(h.reports, fmt.Sprintf("play %d ended", play(r.GetPlayEnded().GetPlayId())))
	case r.GetIndexHint() != nil:
		h.reports = append(h.reports, fmt.Sprintf("items %v changed", r.GetIndexHint().GetItemKeys()))
	case r.GetWatching() != nil:
		var ps []int
		for _, id := range r.GetWatching().GetPlayIds() {
			ps = append(ps, play(id))
		}
		h.reports = append(h.reports, fmt.Sprintf("watching %v", ps))
	}
	return nil
}

func (h *holder) sessions() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, s := range h.facts.GetServers() {
		n += len(s.GetSessions())
	}
	return n
}

func (h *holder) reported() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.reports)
}

type world struct {
	kind    *fakeKind
	control *control
	holder  *holder
	link    *mediaserverlink.Link
}

func start(t *testing.T) *world {
	t.Helper()
	w := &world{kind: &fakeKind{seen: make(chan link.Seen)}, control: &control{version: 1}, holder: &holder{intents: make(chan *msv1.Intent), plays: map[string]int{}}}
	w.link = mediaserverlink.NewLink(mediaserverlink.Config{
		Kinds: link.Registry{w.kind}, Timings: link.DefaultTimings(), Log: slog.New(slog.DiscardHandler),
	})
	local := storagetest.Local(t, role.LocalSchema())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = w.link.Run(local, w.control)(ctx, w.holder)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	return w
}

var den = link.ServerID{UUID: storage.NewUUID()}

func intent(versions ...int64) *msv1.Intent {
	in := &msv1.Intent{SessionsEveryMs: 15_000}
	for _, v := range versions {
		in.Servers = append(in.Servers, &msv1.Server{
			Id: den.Bytes(), Kind: "fake", Name: "Den", Address: "http://127.0.0.1:8096", Identity: "id-den", SettingsVersion: v,
		})
	}
	return in
}

func (w *world) send(in *msv1.Intent) {
	w.holder.intents <- in
	synctest.Wait()
}

func (w *world) advance(t *testing.T, version int64) error {
	t.Helper()
	_, err := w.link.Service().Advance(t.Context(), &msv1.AdvanceRequest{
		ServerId: den.Bytes(), SettingsVersion: version,
		Job: mediaserverlink.JobOnWire(link.Job{Work: link.Refresh{}, Step: link.StepNone}),
	})
	return err
}

// TestTheHolderFetchesACredentialOncePerSettingsVersion holds Den at
// settings version 1, then 2, then not at all: the holder fetches Den's
// credential once for each version, its watch and its steps use the one
// it holds, and once Den leaves the intent no step runs on it.
func TestTheHolderFetchesACredentialOncePerSettingsVersion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t)
		w.send(intent(1))
		if err := w.advance(t, 1); err != nil {
			t.Fatal(err)
		}
		w.send(intent(1))
		if got := w.control.count(); got != 1 {
			t.Errorf("at settings version 1 the holder fetched Den's credential %d times, want once", got)
		}

		w.control.set(2)
		w.send(intent(2))
		if err := w.advance(t, 2); err != nil {
			t.Fatal(err)
		}
		if got := w.control.count(); got != 2 {
			t.Errorf("through settings version 2 the holder fetched Den's credential %d times, want twice", got)
		}
		dialed, steps := w.kind.record()
		if want := []string{"key-1 at 1", "key-1 at 1", "key-2 at 2", "key-2 at 2"}; !slices.Equal(dialed, want) {
			t.Errorf("Den was dialed with %q, want %q", dialed, want)
		}
		if want := []string{"key-1", "key-2"}; !slices.Equal(steps, want) {
			t.Errorf("Den's steps ran with %q, want %q", steps, want)
		}

		w.send(intent())
		if err := w.advance(t, 2); rpc.CodeOf(err) != rpc.CodeFailedPrecondition {
			t.Errorf("a step on Den once it left the intent = %v, want FailedPrecondition", err)
		}
	})
}

// TestTheHolderReportsEachChangeInHowAServerResponds has Den respond,
// respond again, not respond, refuse the credential and respond: the
// holder reports each change once, with the credential's settings
// version.
func TestTheHolderReportsEachChangeInHowAServerResponds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		w := start(t)
		w.send(intent(1))
		for _, p := range []*link.Problem{nil, nil, {Reason: link.ReasonUnreachable}, {Reason: link.ReasonRefused, Status: 401}, nil} {
			w.kind.seen <- link.Seen{At: time.Now(), Polled: true, Problem: p}
			synctest.Wait()
		}
		want := []string{"responding at 1", "watching []", "not_responding at 1", "sign_in_refused at 1", "responding at 1"}
		if got := w.holder.reported(); !slices.Equal(got, want) {
			t.Errorf("the holder reported %q, want %q", got, want)
		}
	})
}

func (w *world) look(sessions ...link.Session) {
	w.kind.seen <- link.Seen{At: time.Now(), Polled: true, Sessions: sessions}
	synctest.Wait()
}

// TestTheHolderReportsPlaysAndKeepsThemAcrossAWatchStartedAgain plays
// Metropolis on Den, has a push name a changed item, starts Den's watch
// again with a new settings version, then polls twice without Metropolis:
// the play starts once, each watch says it is watching it, the item is
// reported, and the play ends after the two polls.
func TestTheHolderReportsPlaysAndKeepsThemAcrossAWatchStartedAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p, err := link.ParsePath("/media/Movies/Metropolis (1927)/Metropolis (1927).mkv")
		if err != nil {
			t.Fatal(err)
		}
		metropolis := link.Session{Key: "1", Path: p, User: "jackson", Decision: link.DecisionDirectPlay}
		w := start(t)
		w.send(intent(1))
		w.look(metropolis)
		w.kind.seen <- link.Seen{At: time.Now(), Items: []string{"42"}}
		synctest.Wait()
		if n := w.holder.sessions(); n != 1 {
			t.Errorf("Den's facts list %d sessions while Metropolis plays, want 1", n)
		}
		w.control.set(2)
		w.send(intent(2))
		w.look(metropolis)
		w.look()
		w.look()
		want := []string{"responding at 1", "play 1 started", "watching [1]", "items [42] changed", "responding at 2", "watching [1]", "play 1 ended"}
		if got := w.holder.reported(); !slices.Equal(got, want) {
			t.Errorf("the holder reported %q, want %q", got, want)
		}
		if n := w.holder.sessions(); n != 0 {
			t.Errorf("Den's facts list %d sessions once Metropolis ended, want none", n)
		}
	})
}

func TestOnlyAChangeInHowManyServersRespondIsNotable(t *testing.T) {
	facts := func(responding ...bool) *msv1.Facts {
		f := &msv1.Facts{}
		for _, r := range responding {
			f.Servers = append(f.Servers, &msv1.ServerFacts{Responding: r, Sessions: []*msv1.Session{{SessionKey: "1"}}})
		}
		return f
	}
	for _, c := range []struct {
		last, next *msv1.Facts
		want       bool
	}{
		{&msv1.Facts{}, facts(true), true},
		{facts(true), facts(false), true},
		{facts(true, false), facts(false, true), false},
		{facts(true), &msv1.Facts{Servers: []*msv1.ServerFacts{{Responding: true}}}, false},
	} {
		if got := mediaserverlink.Def.Notable(c.last, c.next); got != c.want {
			t.Errorf("Notable(%v, %v) = %v, want %v", c.last, c.next, got, c.want)
		}
	}
}
