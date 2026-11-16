// Package notificationstest runs notifications over a real library in a
// synctest bubble, with a scripted Courier and roster, and asserts what
// the Courier received, Health and the bell.
package notificationstest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/identity/identitytest"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	"github.com/jatassi/slipstream/internal/notifications"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// World is one Controller's notifications. Make it inside synctest.Test.
type World struct {
	t        *testing.T
	Lib      *storage.DB[storage.Library]
	Service  *notifications.Service
	Courier  *Courier
	Roster   *Roster
	Identity *identity.Service
	// Words is the Hook provider's own wording, changed in place by a test.
	Words map[providers.ReasonCode]string
	// Self is the Controller, which has the Notifying Role.
	Self role.NodeID

	sealer   *secret.Sealer
	log      *lockedBuffer
	failNext atomic.Bool
	passes   atomic.Int64
	running  func()
	eachPass atomic.Pointer[func()]
	rec      notice.Recorder
}

func New(t *testing.T) *World {
	t.Helper()
	ctx := t.Context()
	lib := storagetest.Library(t, secret.Schema(), identity.Schema(), nodes.Schema(), notifications.Schema())
	keys, err := secret.Open(ctx, lib, filepath.Join(t.TempDir(), secret.KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	id, err := identity.New(ctx, identity.Config{Library: lib, Pepper: keys.PINPepper(), Sealer: keys.Sealer(), Hub: identitytest.Hub(t)})
	if err != nil {
		t.Fatal(err)
	}
	w := &World{
		t: t, Lib: lib, Identity: id, Courier: newCourier(), Words: map[providers.ReasonCode]string{},
		sealer: keys.Sealer(), log: &lockedBuffer{},
	}
	w.Roster = newRoster()
	w.Self = w.AddNode("Den")
	w.Roster.self = w.Self
	w.Restart()
	t.Cleanup(w.Stop)
	return w
}

// Restart stops the dispatcher, if it runs, and starts a new Service over
// the same library.
func (w *World) Restart() {
	w.Stop()
	w.Service = notifications.New(notifications.Config{
		Library: w.Lib, Sealer: w.sealer, Providers: []providers.Provider{hookProvider(w.Words), scriptProvider()},
		Courier: w.Courier, Roster: w.Roster, Mail: noMail{}, Artwork: posters{},
		Changed: nopChanged{}, Seen: nopSeen{}, Timings: notifications.DefaultTimings(),
		Rand: rand.New(rand.NewPCG(1, 2)), Zone: time.UTC,
		Log:        slog.New(slog.NewTextHandler(w.log, &slog.HandlerOptions{Level: slog.LevelInfo})),
		BeforePass: w.beforePass,
	})
	w.rec = w.Service.Recorder("pings", []notice.Entry{PingRow, PongRow})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = w.Service.Run(ctx)
	}()
	w.running = func() {
		cancel()
		<-done
	}
	synctest.Wait()
}

// Stop ends the dispatcher. A Service with none still records.
func (w *World) Stop() {
	if w.running != nil {
		w.running()
		w.running = nil
		synctest.Wait()
	}
}

// ErrInjected is the storage error FailNextWrite puts on a pass.
var ErrInjected = errors.New("notificationstest: an injected storage error")

// FailNextWrite fails the dispatcher's next pass before it commits.
func (w *World) FailNextWrite() { w.failNext.Store(true) }

func (w *World) beforePass(context.Context) error {
	w.passes.Add(1)
	if f := w.eachPass.Load(); f != nil {
		(*f)()
	}
	if w.failNext.Swap(false) {
		return ErrInjected
	}
	return nil
}

// EachPass runs f before each dispatcher pass, on the dispatcher's
// goroutine, until it is called with nil.
func (w *World) EachPass(f func()) {
	if f == nil {
		w.eachPass.Store(nil)
		return
	}
	w.eachPass.Store(&f)
}

// Passes is how many passes the dispatcher has started.
func (w *World) Passes() int { return int(w.passes.Load()) }

// Advance moves the bubble's clock by d and waits for the dispatcher to
// settle.
func (w *World) Advance(d time.Duration) {
	time.Sleep(d)
	synctest.Wait()
}

// AdvanceTo moves the clock to t.
func (w *World) AdvanceTo(t time.Time) { w.Advance(time.Until(t)) }

// Logs is everything the service logged.
func (w *World) Logs() string { return w.log.String() }

// AddNode pairs a Node with the Notifying Role, online.
func (w *World) AddNode(name string) role.NodeID {
	w.t.Helper()
	id := role.NodeID{UUID: storage.NewUUID()}
	err := w.Lib.Write(w.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return nodestest.Insert(ctx, tx, id, name)
	})
	if err != nil {
		w.t.Fatal(err)
	}
	w.Roster.add(id, name)
	return id
}

// URL is the Hook URL a connection named name posts to.
func URL(name string) string { return "https://hook.example/" + name }

// Token is the secret of a connection named name.
func Token(name string) string { return "token-of-" + name }

// Connect makes a Hook connection named name, on, with posters and the
// recommended events. Each change edits the draft first.
func (w *World) Connect(name string, changes ...func(*notifications.Draft)) notifications.ConnectionID {
	w.t.Helper()
	d := notifications.Draft{
		Name: name, Provider: HookID, On: true, Posters: true,
		Settings: providers.Values{"url": Text(URL(name))}, Secrets: map[string]string{"token": Token(name)},
	}
	for _, change := range changes {
		change(&d)
	}
	return w.create(d)
}

// ConnectScript makes a Custom Script connection named name that runs on
// node.
func (w *World) ConnectScript(name string, node role.NodeID) notifications.ConnectionID {
	w.t.Helper()
	return w.create(notifications.Draft{
		Name: name, Provider: ScriptID, On: true, Node: node, Settings: providers.Values{"path": Text(ScriptPath(name))},
	})
}

// ScriptPath is the path a script connection named name runs.
func ScriptPath(name string) string { return "/scripts/" + name }

func (w *World) create(d notifications.Draft) notifications.ConnectionID {
	w.t.Helper()
	v, err := w.Service.Create(w.t.Context(), d)
	if err != nil {
		w.t.Fatal(err)
	}
	synctest.Wait()
	return v.ID
}

// Edit applies e to connection id.
func (w *World) Edit(id notifications.ConnectionID, e notifications.Edit) {
	w.t.Helper()
	if _, err := w.Service.Edit(w.t.Context(), id, e); err != nil {
		w.t.Fatal(err)
	}
	synctest.Wait()
}

// Status is what connection id's row says now.
func (w *World) Status(id notifications.ConnectionID) notifications.Status {
	w.t.Helper()
	v, err := w.Service.Connection(w.t.Context(), id)
	if err != nil {
		w.t.Fatal(err)
	}
	return v.Status
}

// Record records each of fs in one Write, as PingRow or PongRow.
func (w *World) Record(row notice.Row[Ping], fs ...Ping) {
	w.t.Helper()
	err := w.Lib.Write(w.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		for _, f := range fs {
			if err := row.Record(ctx, tx, w.rec, f); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		w.t.Fatal(err)
	}
	synctest.Wait()
}

// Transitions hands ts to the sink in one Write, as one Health evaluation
// does.
func (w *World) Transitions(ts ...health.Transition) {
	w.t.Helper()
	err := w.Lib.Write(w.t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return w.Service.Transitions(ctx, tx, ts)
	})
	if err != nil {
		w.t.Fatal(err)
	}
	synctest.Wait()
}

// Health is the Notifications category's answer now.
func (w *World) Health() health.Answer {
	w.t.Helper()
	a, err := w.Service.HealthSource().Answer(w.t.Context())
	if err != nil {
		w.t.Fatal(err)
	}
	return a
}

// Admin makes an Admin Account signed in now.
func (w *World) Admin(email string) identity.Admin {
	w.t.Helper()
	ctx := w.t.Context()
	token := identitytest.Session(w.t, w.Lib, identitytest.Admin(w.t, w.Lib, email))
	caller, _, err := w.Identity.Authenticate(ctx, identity.Request{Session: token})
	if err != nil {
		w.t.Fatal(err)
	}
	account, err := caller.Account()
	if err != nil {
		w.t.Fatal(err)
	}
	admin, ok := account.AsAdmin()
	if !ok {
		w.t.Fatalf("%s isn't an Admin", email)
	}
	return admin
}

// Feed is the newest page of the bell for by.
func (w *World) Feed(by identity.Admin) notifications.Feed {
	w.t.Helper()
	f, err := w.Service.Feed(w.t.Context(), by, 0, 50)
	if err != nil {
		w.t.Fatal(err)
	}
	return f
}

// Set applies one Notifications setting.
func (w *World) Set(d setting.Duration, v time.Duration) {
	w.t.Helper()
	raw, err := json.Marshal(int64(v / time.Second))
	if err != nil {
		w.t.Fatal(err)
	}
	if err := w.Service.Settings().Apply(w.t.Context(), setting.Patch{d.Key: raw}); err != nil {
		w.t.Fatal(err)
	}
	synctest.Wait()
}

// Text is s as a stored setting.
func Text(s string) json.RawMessage {
	b, _ := json.Marshal(s)
	return b
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type (
	nopChanged struct{}
	nopSeen    struct{}
)

func (nopChanged) Publish(context.Context, notifications.Changed) {}

func (nopSeen) Publish(context.Context, storage.UUID, notifications.Seen) {}
