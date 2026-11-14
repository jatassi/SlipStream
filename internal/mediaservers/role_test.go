package mediaservers_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/mediaserverlink"
	mslink "github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/nodes/nodestest"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/role/roletest"
	"github.com/jatassi/slipstream/internal/secret"
	"github.com/jatassi/slipstream/internal/secret/secrettest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

// side is the Controller link's Trust, Pair and lockstep, answered by the
// roster once it is open.
type side struct{ svc *nodes.Service }

func (s *side) Admit(key link.Key) link.Admission { return s.svc.Admit(key) }

func (s *side) Pair(ctx context.Context, peer link.Key, req *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return s.svc.Pair(ctx, peer, req)
}

func (s *side) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, h := s.svc.Controls().Handler()
	h.ServeHTTP(w, r)
}

type unheard struct{}

func (unheard) Publish(context.Context, nodes.Changed) {}

// logs is every line the Controller logged.
type logs struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logs) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	n, _ := l.buf.Write(p)
	return n, nil
}

func (l *logs) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// holder is a Node's Media Server link holder, which keeps its last
// intent and queues the reports a test gives it.
type holder struct {
	local *storage.DB[storage.Local]
	mu    sync.Mutex
	h     mediaserverlink.Holder
	last  *msv1.Intent
}

func (h *holder) held() role.Held {
	return mediaserverlink.Def.Hold(func(ctx context.Context, rh mediaserverlink.Holder) error {
		for s := range rh.Intent(ctx) {
			h.mu.Lock()
			h.h, h.last = rh, s.Intent
			h.mu.Unlock()
		}
		return nil
	}, func(context.Context) error { return nil })
}

// servers is the names of the servers in h's last intent.
func (h *holder) servers() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []string
	for _, s := range h.last.GetServers() {
		out = append(out, s.GetName()+" at "+s.GetAddress())
	}
	return out
}

func (h *holder) report(t *testing.T, r *msv1.Report) {
	t.Helper()
	h.mu.Lock()
	rh := h.h
	h.mu.Unlock()
	err := h.local.Write(t.Context(), func(ctx context.Context, tx *storage.Tx[storage.Local]) error {
		return rh.Report(ctx, tx, r)
	})
	if err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
}

// rig is home, the Controller, with seedbox paired, both holding Media
// Server link through their holders.
type rig struct {
	lib              *storage.DB[storage.Library]
	sealer           *secret.Sealer
	svc              *nodes.Service
	logs             *logs
	home, seedbox    role.NodeID
	atHome, onSeedbx *holder
}

func newRig(t *testing.T) *rig {
	t.Helper()
	nw := fault.NewNetwork()
	s := &side{}
	homeNode := linktest.Start(t, nw, "home", []netip.Addr{linktest.ControllerAddr}, linktest.Side{Trust: s, Pair: s.Pair, Lockstep: s})
	r := &rig{lib: storagetest.Library(t, schemas()...), sealer: secrettest.Sealer(t), logs: &logs{}}
	log := slog.New(slog.NewTextHandler(r.logs, nil))
	var control *role.Control[*msv1.Intent, *msv1.Facts, *msv1.Report]
	var svc *nodes.Service
	ms := mediaservers.New(mediaservers.Config{
		Library:      r.lib,
		Roster:       roster{&svc},
		Touch:        func(nodes ...role.NodeID) { control.Touch(nodes...) },
		Log:          log,
		Events:       mediaservers.Events{Changed: unpublished[mediaservers.Changed]{}},
		RootsChanged: func(context.Context) <-chan struct{} { return nil },
	})
	control = mediaserverlink.Def.Serve(ms.Intent, ms.Apply)
	owners := nodestest.Unowned()
	for i, o := range owners {
		if o.Role == role.MediaServerLink {
			owners[i] = ms.Owner()
		}
	}
	var err error
	svc, err = nodes.Open(t.Context(), nodes.Config{
		Library: r.lib, Key: homeNode.Key, Name: "home", Link: homeNode.Link, Changed: unheard{},
		Served: []role.Served{control}, Owners: owners, Timings: nodes.DefaultTimings(), Stall: 45 * time.Second, Log: log,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.svc, r.svc, r.home = svc, svc, svc.Self()
	run(t, svc.Run)
	run(t, ms.Run)
	r.atHome = r.hold(t, role.InProcess(svc.Controls()))

	box := linktest.Start(t, nw, "seedbox", []netip.Addr{netip.MustParseAddr("127.0.0.3")}, linktest.Side{})
	code, err := svc.MintJoinCode(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	paired, err := box.Pair(t.Context(), code.Code, "seedbox")
	if err != nil {
		t.Fatal(err)
	}
	u, err := storage.UUIDFromBytes(paired.NodeID)
	if err != nil {
		t.Fatal(err)
	}
	r.seedbox = role.NodeID{UUID: u}
	r.onSeedbx = r.hold(t, role.OverLink(box.Link, homeNode.Key, linktest.Timings))
	synctest.Wait()
	if _, err := r.setRoles(t, role.Of(role.MediaServerLink)); err != nil {
		t.Fatal(err)
	}
	synctest.Wait()
	return r
}

// setRoles ticks roles on seedbox.
func (r *rig) setRoles(t *testing.T, roles role.Set) (nodes.Outcome, error) {
	t.Helper()
	board, err := r.svc.RoleBoard(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return r.svc.Apply(t.Context(), nodes.SetRoles{Basis: board.Basis(), Want: nodes.Roles{r.seedbox: roles}})
}

func (r *rig) hold(t *testing.T, upstream role.Upstream) *holder {
	t.Helper()
	h := &holder{local: storagetest.Local(t, role.LocalSchema())}
	holding, err := role.NewHolding(t.Context(), role.HoldingConfig{
		Local:    h.local,
		Upstream: upstream,
		Held:     []role.Held{h.held()},
		Timings:  role.Timings{RestartMin: time.Second, RestartMax: time.Minute, Beat: 15 * time.Second},
		Rand:     rand.New(rand.NewPCG(1, 2)), //nolint:gosec // seeded on purpose so tests replay
		Log:      slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	run(t, holding.Run)
	return h
}

func (r *rig) names() map[role.NodeID]string {
	return map[role.NodeID]string{r.home: "home", r.seedbox: "seedbox"}
}

// contact is the contact columns of server id.
func (r *rig) contact(t *testing.T, id mslink.ServerID) string {
	t.Helper()
	var got string
	err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
		return rd.QueryRowContext(ctx, `SELECT contact || ' ' || ifnull(contact_since, '-') || ' ' || ifnull(last_contact_at, '-')
			FROM mediaservers_server WHERE id = ?`, id).Scan(&got)
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// roster is the roster once it opens.
type roster struct{ svc **nodes.Service }

func (r roster) Self() role.NodeID        { return (*r.svc).Self() }
func (r roster) Changed() <-chan struct{} { return (*r.svc).Changed() }
func (r roster) InStep(node role.NodeID) bool {
	return (*r.svc).InStep(node)
}
func (r roster) Holds(name role.Name, node role.NodeID) bool {
	return (*r.svc).Holds(name, node)
}

func (r roster) Holding(name role.Name) []role.NodeID { return (*r.svc).Holding(name) }

func (r roster) Standing(node role.NodeID) (nodes.Standing, time.Time) {
	return (*r.svc).Standing(node)
}

type unpublished[T any] struct{}

func (unpublished[T]) Publish(context.Context, T) {}

func run(t *testing.T, f func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = f(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
}

func contactReport(id mslink.ServerID, state mslink.Contact, at time.Time) *msv1.Report {
	return &msv1.Report{ServerId: id.Bytes(), Report: &msv1.Report_Contact{Contact: &msv1.Contact{
		State: string(state), SettingsVersion: 1, AtUnixMs: at.UnixMilli(),
	}}}
}

func ms(t time.Time) string { return strconv.FormatInt(t.UnixMicro(), 10) }

// TestAReportFromANodeThatDoesntReachTheServerIsDropped has seedbox report
// contact with Loft, which home reaches, and then with Den, which it
// reaches itself: the first is dropped with a warning and no skipped
// result, and the second applies.
func TestAReportFromANodeThatDoesntReachTheServerIsDropped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		den := addServer(t, r.lib, r.sealer, "Den", r.seedbox, "http://127.0.0.1:8096", nil)
		loft := addServer(t, r.lib, r.sealer, "Loft", r.home, "http://192.168.1.30:8096", nil)
		at := time.Now()

		r.onSeedbx.report(t, contactReport(loft, mslink.ContactNotResponding, at))
		if got := r.contact(t, loft); got != "unknown - -" {
			t.Errorf("after seedbox's report Loft's contact reads %q, want unknown as before", got)
		}
		if !strings.Contains(r.logs.String(), "level=WARN msg=\"dropped a Media Server report from a Node that doesn't reach the server\"") {
			t.Errorf("the log has no warning for the dropped report:\n%s", r.logs.String())
		}
		rejected, err := r.svc.Controls().Rejected(t.Context())
		if err != nil || len(rejected) != 0 {
			t.Fatalf("after the dropped report the skipped results are %v (%v), want none", rejected, err)
		}

		r.onSeedbx.report(t, contactReport(den, mslink.ContactResponding, at))
		if got, want := r.contact(t, den), "responding "+ms(at)+" "+ms(at); got != want {
			t.Errorf("after seedbox's report Den's contact reads %q, want %q", got, want)
		}
	})
}

// TestUntickingTheRoleOnANodeServersNameIsRefused unticks Media Server
// link on seedbox, which Den and Attic name: the change is refused with
// both servers named, and seedbox keeps the Role.
func TestUntickingTheRoleOnANodeServersNameIsRefused(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		addServer(t, r.lib, r.sealer, "Den", r.seedbox, "http://127.0.0.1:8096", nil)
		addServer(t, r.lib, r.sealer, "Attic", r.seedbox, "http://127.0.0.1:8920", nil)

		_, err := r.setRoles(t, role.Of())
		blocked, ok := errors.AsType[*nodes.BlockedError](err)
		if want := "seedbox still has Attic and Den. Delete them, or move them to another Node that talks to Media Servers, first."; !ok || blocked.Error() != want {
			t.Fatalf("unticking Media Server link on seedbox = %v, want refused with %q", err, want)
		}
		if !r.svc.Holds(role.MediaServerLink, r.seedbox) {
			t.Fatal("after the refused change seedbox doesn't hold Media Server link")
		}
	})
}

// TestRemovingAReachingNodeWithAWriteOffNamesTheController removes
// seedbox, which Den names: the removal is refused until it writes off,
// and then Den names home, which calls its fallback address, and home's
// intent lists it.
func TestRemovingAReachingNodeWithAWriteOffNamesTheController(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t)
		fallback := "http://192.168.1.20:8096"
		den := addServer(t, r.lib, r.sealer, "Den", r.seedbox, "http://127.0.0.1:8096", &fallback)
		at := time.Now()
		r.onSeedbx.report(t, contactReport(den, mslink.ContactResponding, at))

		if _, err := r.svc.Apply(t.Context(), nodes.Remove{Node: r.seedbox}); !errors.As(err, new(*nodes.BlockedError)) {
			t.Fatalf("removing seedbox with no write-off = %v, want refused", err)
		}
		if _, err := r.svc.Apply(t.Context(), nodes.Remove{Node: r.seedbox, WriteOff: true}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()

		var reaching role.NodeID
		var address string
		var fallbackAfter *string
		err := r.lib.Read(t.Context(), func(ctx context.Context, rd storage.Reader[storage.Library]) error {
			return rd.QueryRowContext(ctx, "SELECT reaching_node_id, address, fallback_address FROM mediaservers_server WHERE id = ?", den).
				Scan(&reaching, &address, &fallbackAfter)
		})
		if err != nil {
			t.Fatal(err)
		}
		if reaching != r.home || address != fallback || fallbackAfter != nil {
			t.Errorf("after the removal Den names %s at %s with fallback %v, want home at %s with none", reaching, address, fallbackAfter, fallback)
		}
		if got := r.contact(t, den); got != "unknown - "+ms(at) {
			t.Errorf("after the removal Den's contact reads %q, want unknown until home reports", got)
		}
		if got := r.atHome.servers(); len(got) != 1 || got[0] != "Den at "+fallback {
			t.Errorf("home's intent lists %q, want Den at %s", got, fallback)
		}
		roletest.CheckFresh(t, r.svc.Controls(), r.names())
	})
}
