package sim_test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	gosmtp "github.com/emersion/go-smtp"

	"github.com/jatassi/slipstream/fakes/fault"
	"github.com/jatassi/slipstream/fakes/world/internal/nodeapi"
	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/fakes/world/sim/simtest"
	"github.com/jatassi/slipstream/fakes/world/worldtest"
	"github.com/jatassi/slipstream/internal/contenttype"
	"github.com/jatassi/slipstream/internal/install"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/nodes"
)

var smtpAt = netip.MustParseAddrPort("127.0.0.40:2525")

// mailbox is an SMTP server on the world's network that records the
// address each message came from.
type mailbox struct {
	mu   sync.Mutex
	from []netip.Addr
}

func serveMail(t *testing.T, w *sim.World) *mailbox {
	t.Helper()
	l, err := w.Listen(smtpAt)
	if err != nil {
		t.Fatal(err)
	}
	box := &mailbox{}
	s := gosmtp.NewServer(box)
	s.Domain = "mail.test"
	go func() { _ = s.Serve(l) }()
	t.Cleanup(func() { _ = s.Close() })
	return box
}

func (m *mailbox) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
	from, err := netip.ParseAddrPort(c.Conn().RemoteAddr().String())
	if err != nil {
		return nil, fmt.Errorf("the mail server's peer: %w", err)
	}
	return &letter{box: m, from: from.Addr()}, nil
}

func (m *mailbox) senders() []netip.Addr {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.from)
}

type letter struct {
	box  *mailbox
	from netip.Addr
}

func (*letter) Mail(string, *gosmtp.MailOptions) error { return nil }
func (*letter) Rcpt(string, *gosmtp.RcptOptions) error { return nil }
func (*letter) Reset()                                 {}
func (*letter) Logout() error                          { return nil }

func (l *letter) Data(r io.Reader) error {
	if _, err := io.Copy(io.Discard, r); err != nil {
		return fmt.Errorf("read the message: %w", err)
	}
	l.box.mu.Lock()
	defer l.box.mu.Unlock()
	l.box.from = append(l.box.from, l.from)
	return nil
}

func setMail(t *testing.T, a *worldtest.API, at netip.AddrPort) {
	t.Helper()
	body := `{"host":"` + at.Addr().String() + `","port":` + strconv.Itoa(int(at.Port())) + `,"security":"none","fromAddress":"slipstream@home.test","fromName":"SlipStream"}`
	if r := a.Admin("home", http.MethodPatch, "/api/v1/mail/settings", body); r.Status != http.StatusOK {
		t.Fatalf("save the email setting = %d %s", r.Status, r.Body)
	}
}

// notifyingOnSeedbox moves Notifying from home to seedbox.
func notifyingOnSeedbox(t *testing.T, a *worldtest.API) worldtest.Reply {
	t.Helper()
	var set []nodeapi.Assignment
	for _, r := range a.Nodes("home") {
		roles := slices.DeleteFunc(r.Assigned(), func(n string) bool { return n == "notifying" })
		if r.Name == "seedbox" {
			roles = append(roles, "notifying")
		}
		set = append(set, nodeapi.Assignment{NodeID: r.ID, Roles: roles})
	}
	return a.Assign("home", set...)
}

type sent struct {
	Outcome string `json:"outcome"`
	Detail  string `json:"detail"`
}

func testMail(t *testing.T, a *worldtest.API, session string) sent {
	t.Helper()
	r := a.Signed("home", session, http.MethodPost, "/api/v1/mail/test", "")
	var out sent
	if err := json.Unmarshal([]byte(r.Body), &out); r.Status != http.StatusOK || err != nil {
		t.Fatalf("sendTestMail = %d %s, want 200", r.Status, r.Body)
	}
	return out
}

// TestMailGoesThroughTheNodeThatSendsIt moves Notifying to seedbox, whose
// reach test finds the mail server, so a test mail reaches the server from
// seedbox. Cut off within the offline grace, seedbox sends nothing and
// neither does home, and once seedbox lapses home covers Notifying and
// sends the mail itself.
func TestMailGoesThroughTheNodeThatSendsIt(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, simtest.Home(t))
		a := w.API()
		box := serveMail(t, w)
		setMail(t, a, smtpAt)
		var moved worldtest.Outcome
		if r := notifyingOnSeedbox(t, a); r.Status != http.StatusOK || json.Unmarshal([]byte(r.Body), &moved) != nil {
			t.Fatalf("setRoles moving Notifying to seedbox = %d %s, want 200", r.Status, r.Body)
		}
		session := a.SignIn("home")

		if got := testMail(t, a, session); got.Outcome != "sent" {
			t.Fatalf("a test mail through seedbox = %+v, want sent", got)
		}
		if got := box.senders(); !slices.Equal(got, []netip.Addr{netip.MustParseAddr("127.0.0.3")}) {
			t.Fatalf("the mail server heard from %v, want seedbox at 127.0.0.3", got)
		}

		simtest.Flip(t, w.Path("home", "seedbox"), fault.Stall{})
		link := node.Defaults(contenttype.Set{}, install.Dev).Timings.Link
		time.Sleep(link.Keepalive + link.WriteTimeout)
		synctest.Wait()
		row := seedbox(t, a)
		if row.Link.State != "down" {
			t.Fatalf("seedbox is %s after the stall, want down", row.Link.State)
		}
		if got := testMail(t, a, session); got.Outcome != "failed" || !strings.Contains(got.Detail, "seedbox is") {
			t.Fatalf("a test mail while seedbox is away = %+v, want failed naming seedbox", got)
		}
		seen, err := time.Parse(time.RFC3339Nano, row.Link.Since)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Until(seen.Add(nodes.OfflineGrace.Default)) + time.Second)
		synctest.Wait()
		if got := testMail(t, a, session); got.Outcome != "sent" {
			t.Fatalf("a test mail once seedbox lapsed = %+v, want sent", got)
		}
		if got := box.senders(); !slices.Equal(got, []netip.Addr{netip.MustParseAddr("127.0.0.3"), netip.MustParseAddr("127.0.0.2")}) {
			t.Fatalf("the mail server heard from %v, want seedbox and then home", got)
		}
	})
}

// TestAMailServerSeedboxCantReachBlocksNotifyingThere saves a mail server
// nothing listens at, so seedbox's reach test fails and the move is
// blocked.
func TestAMailServerSeedboxCantReachBlocksNotifyingThere(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, simtest.Home(t))
		a := w.API()
		setMail(t, a, smtpAt)
		checks := blockedChecks(t, notifyingOnSeedbox(t, a))
		if len(checks) != 1 || checks[0].Reason != "reach_failed" || checks[0].Target != "127.0.0.40:2525" || !strings.HasPrefix(checks[0].Words, "seedbox can't reach 127.0.0.40:2525 (") {
			t.Fatalf("the blocked set checks %+v, want seedbox's failed reach test", checks)
		}
	})
}

type check struct {
	Reason string `json:"reason"`
	Target string `json:"target"`
	Words  string `json:"words"`
}

// blockedChecks is the checks of a setRoles a guardrail blocked, failing
// the test unless r is that refusal.
func blockedChecks(t *testing.T, r worldtest.Reply) []check {
	t.Helper()
	var blocked struct {
		Code    string `json:"code"`
		Outcome struct {
			Checks []check `json:"checks"`
		} `json:"outcome"`
	}
	if err := json.Unmarshal([]byte(r.Body), &blocked); err != nil || r.Status != http.StatusConflict || blocked.Code != "roles_blocked" {
		t.Fatalf("setRoles = %d %s, want 409 roles_blocked", r.Status, r.Body)
	}
	return blocked.Outcome.Checks
}

var hookAt = netip.MustParseAddrPort("127.0.0.41:80")

// hook is a webhook server on the world's network that records the address
// each Post came from. It closes every connection from refused, as a
// host that one Node's network can't reach.
type hook struct {
	mu   sync.Mutex
	from []netip.Addr
}

func serveHook(t *testing.T, w *sim.World, refused netip.Addr) *hook {
	t.Helper()
	l, err := w.Listen(hookAt)
	if err != nil {
		t.Fatal(err)
	}
	h := &hook{}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			if from, err := netip.ParseAddrPort(c.RemoteAddr().String()); err == nil && from.Addr() == refused {
				_ = c.Close()
				continue
			}
			go h.serve(c)
		}
	}()
	t.Cleanup(func() { _ = l.Close() })
	return h
}

func (h *hook) serve(c net.Conn) {
	defer func() { _ = c.Close() }()
	from, err := netip.ParseAddrPort(c.RemoteAddr().String())
	if err != nil {
		return
	}
	r := bufio.NewReader(c)
	for {
		req, err := http.ReadRequest(r)
		if err != nil {
			return
		}
		_, _ = io.Copy(io.Discard, req.Body)
		h.mu.Lock()
		h.from = append(h.from, from.Addr())
		h.mu.Unlock()
		if _, err := io.WriteString(c, "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"); err != nil {
			return
		}
	}
}

func (h *hook) senders() []netip.Addr {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.from)
}

type tested struct {
	Node   string `json:"node"`
	Detail string `json:"detail"`
}

func testConnection(t *testing.T, a *worldtest.API, id string) (int, tested) {
	t.Helper()
	r := a.Admin("home", http.MethodPost, "/api/v1/notifications/connections/"+id+"/test", "{}")
	var out tested
	if err := json.Unmarshal([]byte(r.Body), &out); err != nil {
		t.Fatalf("testNotificationConnection = %d %s", r.Status, r.Body)
	}
	return r.Status, out
}

func addConnection(t *testing.T, a *worldtest.API, body string) string {
	t.Helper()
	r := a.Admin("home", http.MethodPost, "/api/v1/notifications/connections", body)
	var added struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(r.Body), &added); r.Status != http.StatusCreated || err != nil {
		t.Fatalf("addNotificationConnection = %d %s, want 201", r.Status, r.Body)
	}
	return added.ID
}

// TestANotificationGoesFromTheWorkerWhenTheControllerCant saves a webhook
// whose host closes home's connections. A Test from home fails; with
// Notifying moved to seedbox, the Test passes and the host heard only
// from seedbox.
func TestANotificationGoesFromTheWorkerWhenTheControllerCant(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, simtest.Home(t))
		a := w.API()
		h := serveHook(t, w, netip.MustParseAddr("127.0.0.2"))
		id := addConnection(t, a, `{"name":"Family","provider":"webhook","on":true,"secrets":{"url":"http://`+hookAt.String()+`/hook"}}`)

		if status, got := testConnection(t, a, id); status != http.StatusUnprocessableEntity {
			t.Fatalf("a Test from home = %d %+v, want 422", status, got)
		}
		if r := notifyingOnSeedbox(t, a); r.Status != http.StatusOK {
			t.Fatalf("setRoles moving Notifying to seedbox = %d %s, want 200", r.Status, r.Body)
		}
		if status, got := testConnection(t, a, id); status != http.StatusOK || got.Node != "seedbox" {
			t.Fatalf("a Test through seedbox = %d %+v, want 200 from seedbox", status, got)
		}
		if got := h.senders(); !slices.Equal(got, []netip.Addr{netip.MustParseAddr("127.0.0.3")}) {
			t.Fatalf("the webhook host heard from %v, want seedbox at 127.0.0.3 alone", got)
		}
	})
}

// TestAScriptOnAnOfflineNodeWaitsThenRunsOnceItIsBack adds a Custom Script
// on seedbox, which gets Works added. With seedbox cut off inside its
// offline grace, a Work added runs no script anywhere; once the link
// heals, the script runs on seedbox once.
func TestAScriptOnAnOfflineNodeWaitsThenRunsOnceItIsBack(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		w := sim.Launch(t, simtest.Home(t), sim.WithWalk())
		a := w.API()
		if r := notifyingOnSeedbox(t, a); r.Status != http.StatusOK {
			t.Fatalf("setRoles moving Notifying to seedbox = %d %s, want 200", r.Status, r.Body)
		}
		addConnection(t, a, `{"name":"Den","provider":"script","on":true,"events":["works_added"],"node":"`+seedbox(t, a).ID+`","values":{"path":"/scripts/den"}}`)

		down := simtest.Partition(t, w)
		simtest.AddOrchard(t, w)
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		if got := append(w.Node("home").Scripts(), w.Node("seedbox").Scripts()...); len(got) != 0 {
			t.Fatalf("with seedbox cut off the script ran %+v, want it to wait", got)
		}
		simtest.Heal(w, down)
		simtest.Eventually(t, "the script runs on seedbox once it is back", func() bool { return len(w.Node("seedbox").Scripts()) > 0 })
		time.Sleep(time.Minute)
		synctest.Wait()
		got := w.Node("seedbox").Scripts()
		if len(got) != 1 || got[0].Path != "/scripts/den" || got[0].Event != "works_added" || len(w.Node("home").Scripts()) != 0 {
			t.Fatalf("back online the scripts ran %+v on seedbox and %+v on home, want Works added once on seedbox", got, w.Node("home").Scripts())
		}
	})
}
