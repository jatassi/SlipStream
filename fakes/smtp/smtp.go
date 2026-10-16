// Package smtp is fakes/smtp: a capturing mail server on go-smtp. One
// front serves plaintext SMTP with STARTTLS and implicit TLS. It accepts
// any AUTH PLAIN and keeps every message until a cue says otherwise.
package smtp

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/emersion/go-sasl"
	gosmtp "github.com/emersion/go-smtp"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
)

// Mail is the capture of every message the server accepted, in
// smtp.message records whose node is the sending origin.
var Mail = fake.CaptureSpec{Name: "mail", Kind: "smtp.message"}

// Each cue turns its play on from now on, and off with {"off": true}.
var (
	// AuthFailure answers every AUTH 535.
	AuthFailure = fake.CueSpec{Name: "smtp-auth-failure"}
	// Defer answers every RCPT 451.
	Defer = fake.CueSpec{Name: "smtp-defer"}
	// Reject answers every RCPT 550.
	Reject = fake.CueSpec{Name: "smtp-reject"}
	// TLSRequired answers MAIL 530 on a connection without TLS.
	TLSRequired = fake.CueSpec{Name: "smtp-tls-required"}
	// Refused closes every connection as it arrives.
	Refused = fake.CueSpec{Name: "smtp-refused"}
	// Slow holds each connection silent for {"delay": "30s"}, its default,
	// before it answers.
	Slow = fake.CueSpec{Name: "smtp-slow"}
)

var Fake = fake.Spec{
	Name:      "smtp",
	TLS:       fake.OwnTLS,
	Scheme:    "smtp",
	New:       newServer,
	Endpoints: func(*node.EndpointsBlock, fake.Address) {},
	Online:    fake.NoOnlineRun("an SMTP server is the Admin's own; there is no shared real one to check against"),
	Captures:  []fake.CaptureSpec{Mail},
	Cues:      []fake.CueSpec{AuthFailure, Defer, Reject, TLSRequired, Refused, Slow},
	Entered:   entered,
}

// Message is one accepted message as fakeworld mail shows it. Security is
// how its connection ran: none, starttls or tls.
type Message struct {
	From     string   `json:"from"`
	To       []string `json:"to"`
	Username string   `json:"username"`
	Security string   `json:"security"`
	Subject  string   `json:"subject"`
	Text     string   `json:"text"`
	HTML     string   `json:"html"`
}

// Toggle is a cue's arguments.
type Toggle struct {
	Off bool `json:"off"`
}

type SlowArgs struct {
	Off   bool   `json:"off"`
	Delay string `json:"delay"`
}

const defaultSlow = 30 * time.Second

// sniffWindow is how long a connection may stay silent before it is a
// plaintext client waiting for the greeting. A client in implicit TLS
// speaks first.
const sniffWindow = 200 * time.Millisecond

const tlsHandshake = 0x16

type server struct {
	env  fake.Env
	mail *fake.Capture

	mu    sync.Mutex
	on    map[string]bool
	delay time.Duration
}

func newServer(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	mail, err := env.Capture(Mail.Name)
	if err != nil {
		return nil, err
	}
	s := &server{env: env, mail: mail, on: map[string]bool{}}
	for _, cue := range []fake.CueSpec{AuthFailure, Defer, Reject, TLSRequired, Refused} {
		if err := fake.OnCue(env, cue.Name, func(_ context.Context, a Toggle) (any, error) { return s.set(cue.Name, !a.Off), nil }); err != nil {
			return nil, err
		}
	}
	if err := fake.OnCue(env, Slow.Name, s.slow); err != nil {
		return nil, err
	}
	return s, nil
}

type was struct {
	Was bool `json:"was"`
}

func (s *server) set(cue string, on bool) was {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := s.on[cue]
	s.on[cue] = on
	return was{before}
}

func (s *server) slow(_ context.Context, a SlowArgs) (any, error) {
	delay := defaultSlow
	if a.Delay != "" {
		d, err := time.ParseDuration(a.Delay)
		if err != nil || d <= 0 {
			return nil, fmt.Errorf("delay %q is not a positive duration", a.Delay)
		}
		delay = d
	}
	s.mu.Lock()
	s.delay = delay
	s.mu.Unlock()
	return s.set(Slow.Name, !a.Off), nil
}

func (s *server) playing(cue fake.CueSpec) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.on[cue.Name]
}

func (s *server) holding() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.on[Slow.Name] {
		return 0
	}
	return s.delay
}

// Serve hands each connection to the implicit-TLS server when its first
// byte opens a TLS handshake, and to the plaintext server, which offers
// STARTTLS, when it stays silent.
func (s *server) Serve(ctx context.Context, l net.Listener) error {
	plain, implicit := newHandoff(l.Addr()), newHandoff(l.Addr())
	servers := []*gosmtp.Server{s.smtp(ctx, false), s.smtp(ctx, true)}
	var wg sync.WaitGroup
	for i, h := range []*handoff{plain, implicit} {
		wg.Go(func() { _ = servers[i].Serve(h) })
	}
	stop := context.AfterFunc(ctx, func() { _ = l.Close() })
	defer stop()
	for {
		c, err := l.Accept()
		if err != nil {
			_ = plain.Close()
			_ = implicit.Close()
			for _, srv := range servers {
				_ = srv.Close()
			}
			wg.Wait()
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("accept smtp: %w", err)
		}
		go s.route(ctx, c, plain, implicit)
	}
}

func (s *server) smtp(ctx context.Context, implicit bool) *gosmtp.Server {
	srv := gosmtp.NewServer(gosmtp.BackendFunc(func(c *gosmtp.Conn) (gosmtp.Session, error) {
		return &session{srv: s, ctx: fake.ConnContext(ctx, c.Conn()), conn: c, implicit: implicit}, nil
	}))
	srv.Domain = Fake.Name + ".fake.localhost"
	srv.AllowInsecureAuth = true
	if !implicit {
		srv.TLSConfig = s.env.Leaf
	}
	srv.ErrorLog = slog.NewLogLogger(s.env.Log.Handler(), slog.LevelWarn)
	return srv
}

func (s *server) route(ctx context.Context, c net.Conn, plain, implicit *handoff) {
	if s.playing(Refused) {
		_ = c.Close()
		return
	}
	if d := s.holding(); d > 0 {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			_ = c.Close()
			return
		}
	}
	if conn, ok := sniff(c); ok {
		implicit.push(tls.Server(conn, s.env.Leaf))
	} else {
		plain.push(conn)
	}
}

// sniff is c with its first byte still to read, and whether that byte
// opens a TLS handshake.
func sniff(c net.Conn) (net.Conn, bool) {
	r := bufio.NewReader(c)
	_ = c.SetReadDeadline(time.Now().Add(sniffWindow))
	first, err := r.Peek(1)
	_ = c.SetReadDeadline(time.Time{})
	if err != nil {
		return c, false
	}
	return &peeked{Conn: c, r: r}, first[0] == tlsHandshake
}

type peeked struct {
	net.Conn
	r *bufio.Reader
}

func (p *peeked) Read(b []byte) (int, error) { return p.r.Read(b) } //nolint:wrapcheck // a net.Conn's own read

func (p *peeked) NetConn() net.Conn { return p.Conn }

// handoff is a listener whose connections route pushes.
type handoff struct {
	addr  net.Addr
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
}

func newHandoff(addr net.Addr) *handoff {
	return &handoff{addr: addr, conns: make(chan net.Conn), done: make(chan struct{})}
}

func (h *handoff) push(c net.Conn) {
	select {
	case h.conns <- c:
	case <-h.done:
		_ = c.Close()
	}
}

func (h *handoff) Accept() (net.Conn, error) {
	select {
	case c := <-h.conns:
		return c, nil
	case <-h.done:
		return nil, net.ErrClosed
	}
}

func (h *handoff) Close() error {
	h.once.Do(func() { close(h.done) })
	return nil
}

func (h *handoff) Addr() net.Addr { return h.addr }

type session struct {
	srv      *server
	ctx      context.Context
	conn     *gosmtp.Conn
	implicit bool
	username string
	from     string
	to       []string
}

var _ gosmtp.AuthSession = (*session)(nil)

func (s *session) security() string {
	if _, ok := s.conn.TLSConnectionState(); !ok {
		return "none"
	}
	if s.implicit {
		return "tls"
	}
	return "starttls"
}

func (s *session) AuthMechanisms() []string { return []string{sasl.Plain} }

func (s *session) Auth(string) (sasl.Server, error) {
	return sasl.NewPlainServer(func(_, username, _ string) error {
		if s.srv.playing(AuthFailure) {
			return gosmtp.ErrAuthFailed
		}
		s.username = username
		return nil
	}), nil
}

func (s *session) Mail(from string, _ *gosmtp.MailOptions) error {
	if s.srv.playing(TLSRequired) && s.security() == "none" {
		return &gosmtp.SMTPError{Code: 530, EnhancedCode: gosmtp.EnhancedCode{5, 7, 0}, Message: "Must issue a STARTTLS command first"}
	}
	s.from = from
	return nil
}

func (s *session) Rcpt(to string, _ *gosmtp.RcptOptions) error {
	switch {
	case s.srv.playing(Defer):
		return &gosmtp.SMTPError{Code: 451, EnhancedCode: gosmtp.EnhancedCode{4, 3, 0}, Message: "Try again later"}
	case s.srv.playing(Reject):
		return &gosmtp.SMTPError{Code: 550, EnhancedCode: gosmtp.EnhancedCode{5, 1, 1}, Message: "Mailbox unavailable"}
	}
	s.to = append(s.to, to)
	return nil
}

func (s *session) Data(r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read message: %w", err)
	}
	parts, err := parse(raw)
	if err != nil {
		return &gosmtp.SMTPError{Code: 554, EnhancedCode: gosmtp.EnhancedCode{5, 6, 0}, Message: err.Error()}
	}
	m := Message{From: s.from, To: s.to, Username: s.username, Security: s.security(), Subject: parts.subject, Text: parts.text, HTML: parts.html}
	s.srv.mail.Record(s.ctx, m,
		fake.File{Name: "message.eml", Data: raw},
		fake.File{Name: "message.html", Data: []byte(parts.html)},
		fake.File{Name: "message.txt", Data: []byte(parts.text)},
	)
	return nil
}

func (s *session) Reset() { s.from, s.to = "", nil }

func (s *session) Logout() error { return nil }

// entered is the SMTP server an Admin saved, which the Controller dials.
func entered(get func(path string) ([]byte, error)) ([]fake.Entered, error) {
	body, err := get("/api/v1/mail/settings")
	if errors.Is(err, fake.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var settings struct {
		Host string `json:"host"`
		Port uint16 `json:"port"`
	}
	if err := json.Unmarshal(body, &settings); err != nil {
		return nil, fmt.Errorf("getMailSettings: %w", err)
	}
	if settings.Host == "" {
		return nil, nil
	}
	if body, err = get("/api/v1/nodes"); err != nil {
		return nil, err
	}
	var list struct {
		Nodes []struct {
			ID   string `json:"id"`
			Self bool   `json:"self"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("listNodes: %w", err)
	}
	var controller string
	for _, n := range list.Nodes {
		if n.Self {
			controller = n.ID
		}
	}
	at := net.JoinHostPort(settings.Host, strconv.Itoa(int(settings.Port)))
	return []fake.Entered{{What: "SMTP server", URL: "smtp://" + at, NodeID: controller}}, nil
}
