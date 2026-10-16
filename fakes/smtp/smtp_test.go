package smtp_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/fake/faketest"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/fakes/smtp"
	"github.com/jatassi/slipstream/internal/mail"
)

var grace = mail.Address{Name: "Grace Hopper", Email: "grace@example.com"}

func serve(t *testing.T) faketest.Served {
	t.Helper()
	return faketest.Loopback(t, smtp.Fake, scenario.Scenario{Name: "mail", Seed: "mail"}, faketest.As("home"))
}

func sender(f faketest.Served, security mail.Security, username string) mail.SMTP {
	return mail.SMTP{
		Server: mail.Server{
			Host:     "127.0.0.1",
			Port:     f.Address.AddrPort.Port(),
			Security: security,
			Username: username,
			Password: "hunter2",
			From:     mail.Address{Name: "SlipStream", Email: "slipstream@media-pc.example"},
		},
		Timings: mail.DefaultTimings(),
		Roots:   f.Roots,
	}
}

func testMessage(t *testing.T) mail.Message {
	t.Helper()
	m, err := mail.TestMessage(grace, mail.TestEmail{ServerName: "media-pc", URL: "http://media-pc:8484"})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func send(t *testing.T, s mail.SMTP, m mail.Message) mail.Outcome {
	t.Helper()
	out, err := s.Send(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// captured is every message the fake kept, and the files each kept.
func captured(t *testing.T, f faketest.Served) (msgs []smtp.Message, files []map[string]string) {
	t.Helper()
	for _, r := range f.Records(t) {
		if r.Kind != smtp.Mail.Kind {
			continue
		}
		if r.Node != "home" {
			t.Fatalf("captured from %q, want home", r.Node)
		}
		var m smtp.Message
		if err := json.Unmarshal(r.Data, &m); err != nil {
			t.Fatal(err)
		}
		msgs = append(msgs, m)
		kept := map[string]string{}
		for name, rel := range r.Files {
			b, err := os.ReadFile(filepath.Join(f.RunDir, rel))
			if err != nil {
				t.Fatal(err)
			}
			kept[name] = string(b)
		}
		files = append(files, kept)
	}
	return msgs, files
}

func TestEverySecurityDeliversTheMessage(t *testing.T) {
	cases := []struct {
		security mail.Security
		username string
	}{
		{mail.None, ""},
		{mail.None, "slipstream"},
		{mail.StartTLS, "slipstream"},
		{mail.TLS, "slipstream"},
	}
	for _, c := range cases {
		t.Run(string(c.security)+"/"+c.username, func(t *testing.T) {
			t.Parallel()
			f := serve(t)
			m := testMessage(t)
			if out := send(t, sender(f, c.security, c.username), m); out != (mail.Outcome{Kind: mail.Sent}) {
				t.Fatalf("Send = %+v", out)
			}
			msgs, files := captured(t, f)
			want := smtp.Message{
				From:     "slipstream@media-pc.example",
				To:       []string{"grace@example.com"},
				Username: c.username,
				Security: string(c.security),
				Subject:  "Test email from media-pc",
				Text:     crlf(m.Text),
				HTML:     crlf(m.HTML),
			}
			if len(msgs) != 1 || !reflect.DeepEqual(msgs[0], want) {
				t.Fatalf("captured %+v, want %+v", msgs, want)
			}
			if files[0]["message.html"] != want.HTML || files[0]["message.txt"] != want.Text || !strings.Contains(files[0]["message.eml"], "Subject: Test email from media-pc\r\n") {
				t.Fatalf("kept files %v", files[0])
			}
		})
	}
}

// crlf is s as it crosses SMTP, every line ending CRLF.
func crlf(s string) string { return strings.ReplaceAll(s, "\n", "\r\n") }

func TestCuesAnswerTheServersReply(t *testing.T) {
	cases := []struct {
		cue      fake.CueSpec
		security mail.Security
		reply    string
	}{
		{smtp.AuthFailure, mail.StartTLS, "535 5.7.8 Authentication failed"},
		{smtp.Defer, mail.StartTLS, "451 4.3.0 Try again later"},
		{smtp.Reject, mail.TLS, "550 5.1.1 Mailbox unavailable"},
		{smtp.TLSRequired, mail.None, "530 5.7.0 Must issue a STARTTLS command first"},
	}
	for _, c := range cases {
		t.Run(c.cue.Name, func(t *testing.T) {
			t.Parallel()
			f := serve(t)
			s := sender(f, c.security, "slipstream")
			f.Cue(t, c.cue.Name, smtp.Toggle{})
			if out := send(t, s, testMessage(t)); out != (mail.Outcome{Kind: mail.Failed, Reply: c.reply}) {
				t.Fatalf("Send = %+v, want failed %q", out, c.reply)
			}
			f.Cue(t, c.cue.Name, smtp.Toggle{Off: true})
			if out := send(t, s, testMessage(t)); out != (mail.Outcome{Kind: mail.Sent}) {
				t.Fatalf("Send after the cue is off = %+v", out)
			}
			if msgs, _ := captured(t, f); len(msgs) != 1 {
				t.Fatalf("captured %d messages, want only the one sent after the cue", len(msgs))
			}
		})
	}
}

func TestTLSRequiredStillTakesSTARTTLS(t *testing.T) {
	f := serve(t)
	f.Cue(t, smtp.TLSRequired.Name, smtp.Toggle{})
	if out := send(t, sender(f, mail.StartTLS, ""), testMessage(t)); out != (mail.Outcome{Kind: mail.Sent}) {
		t.Fatalf("Send = %+v", out)
	}
}

func TestARefusedConnectionFails(t *testing.T) {
	f := serve(t)
	f.Cue(t, smtp.Refused.Name, smtp.Toggle{})
	out := send(t, sender(f, mail.StartTLS, ""), testMessage(t))
	if out.Kind != mail.Failed || !strings.Contains(out.Reply, "EOF") {
		t.Fatalf("Send = %+v, want failed at EOF", out)
	}
}

func TestASlowServerRunsOutTheSendTimeout(t *testing.T) {
	f := serve(t)
	f.Cue(t, smtp.Slow.Name, smtp.SlowArgs{Delay: "5s"})
	s := sender(f, mail.StartTLS, "")
	s.Timings.Send = 500 * time.Millisecond
	start := time.Now()
	out := send(t, s, testMessage(t))
	if out.Kind != mail.Failed || !strings.Contains(out.Reply, "i/o timeout") {
		t.Fatalf("Send = %+v, want failed by the timeout", out)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Fatalf("Send took %v past a 500ms timeout", took)
	}
}

func TestEnteredIsTheSavedServerTheControllerDials(t *testing.T) {
	api := func(settings string) func(string) ([]byte, error) {
		return func(path string) ([]byte, error) {
			switch path {
			case "/api/v1/mail/settings":
				if settings == "" {
					return nil, fake.ErrNotFound
				}
				return []byte(settings), nil
			case "/api/v1/nodes":
				return []byte(`{"nodes":[{"id":"worker-id","self":false},{"id":"controller-id","self":true}]}`), nil
			}
			return nil, errors.New("unexpected GET " + path)
		}
	}
	cases := []struct {
		name, settings string
		want           []fake.Entered
	}{
		{"saved", `{"host":"127.0.0.1","port":2525,"security":"starttls","username":"","passwordSet":false,"fromAddress":"slipstream@media-pc.example","fromName":"SlipStream"}`,
			[]fake.Entered{{What: "SMTP server", URL: "smtp://127.0.0.1:2525", NodeID: "controller-id"}}},
		{"not set up", `{"host":"","port":587,"security":"starttls","username":"","passwordSet":false,"fromAddress":"","fromName":"SlipStream"}`, nil},
		{"no mail settings yet", "", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := smtp.Fake.Entered(api(c.settings))
			if err != nil || !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Entered = %+v, %v, want %+v", got, err, c.want)
			}
		})
	}
}
