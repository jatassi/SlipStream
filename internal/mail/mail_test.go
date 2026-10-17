package mail_test

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"
	"testing/synctest"
	"time"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/mail/mailtest"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
)

var update = flag.Bool("update", false, "rewrite the golden emails under testdata")

var grace = mail.Address{Name: "Grace Hopper", Email: "grace@example.com"}

func TestTestMessageRendersOnTheLayout(t *testing.T) {
	cases := []struct {
		golden, subject string
		email           mail.TestEmail
	}{
		{"test", "Test email from media-pc", mail.TestEmail{ServerName: "media-pc", URL: "http://media-pc:8484"}},
		{"test-no-address", "Test email from SlipStream", mail.TestEmail{}},
	}
	for _, c := range cases {
		t.Run(c.golden, func(t *testing.T) {
			m, err := mail.TestMessage(grace, c.email)
			if err != nil {
				t.Fatal(err)
			}
			if m.To != grace || m.Subject != c.subject {
				t.Fatalf("to %+v subject %q, want %+v %q", m.To, m.Subject, grace, c.subject)
			}
			mailtest.Golden(t, m, "testdata", c.golden, *update)
		})
	}
}

func TestAButtonTakesOnlyAWebLink(t *testing.T) {
	files := fstest.MapFS{
		"link.html.tmpl": {Data: []byte(`{{define "body"}}{{button "Open" .}}{{end}}`)},
		"link.txt.tmpl":  {Data: []byte(`{{define "subject"}}Link{{end}}{{define "body"}}{{button "Open" .}}{{end}}`)},
	}
	link := mail.MustTemplate[string](files, "link")
	if _, err := link.Render(grace, "javascript:alert(1)"); err == nil || !strings.Contains(err.Error(), `button link "javascript:alert(1)" is not an http or https URL`) {
		t.Fatalf("Render with a script link = %v", err)
	}
	m, err := link.Render(grace, "https://slipstream.example.com/signup#token=a&b")
	if err != nil {
		t.Fatal(err)
	}
	if want := `href="https://slipstream.example.com/signup#token=a&amp;b"`; strings.Count(m.HTML, want) != 2 {
		t.Fatalf("HTML has %d of %s, want the VML and the link:\n%s", strings.Count(m.HTML, want), want, m.HTML)
	}
	if want := "Open: https://slipstream.example.com/signup#token=a&b\n"; m.Text != want {
		t.Fatalf("Text = %q, want %q", m.Text, want)
	}
}

func TestATemplateWithoutASubjectPanics(t *testing.T) {
	files := fstest.MapFS{
		"bare.html.tmpl": {Data: []byte(`{{define "body"}}Hi{{end}}`)},
		"bare.txt.tmpl":  {Data: []byte(`{{define "body"}}Hi{{end}}`)},
	}
	defer func() {
		if got := recover(); got != "mail: template bare needs body in both files and subject in its text file" {
			t.Fatalf("panic = %v", got)
		}
	}()
	mail.MustTemplate[struct{}](files, "bare")
}

func TestAServerWithNoHostIsNotSetUp(t *testing.T) {
	got, err := mail.SMTP{Timings: mail.DefaultTimings()}.Send(t.Context(), mail.Message{To: grace})
	if err != nil || got != (mail.Outcome{Kind: mail.NotSetUp}) {
		t.Fatalf("Send = %+v, %v", got, err)
	}
}

func TestSendInsideWritePanics(t *testing.T) {
	lib := storagetest.Library(t)
	for name, s := range map[string]mail.Sender{"smtp": mail.SMTP{}, "mailtest": mailtest.New()} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if got := recover(); got != "mail: Send inside Write; send after the commit" {
					t.Fatalf("panic = %v", got)
				}
			}()
			_ = lib.Write(t.Context(), func(ctx context.Context, _ *storage.Tx[storage.Library]) error {
				_, _ = s.Send(ctx, mail.Message{To: grace})
				return nil
			})
		})
	}
}

func newLog() (*slog.Logger, *bytes.Buffer) {
	var b bytes.Buffer
	noTime := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey {
			return slog.Attr{}
		}
		return a
	}
	return slog.New(slog.NewTextHandler(&b, &slog.HandlerOptions{ReplaceAttr: noTime})), &b
}

func TestTheOutboxLogsAMessageTheServerRefused(t *testing.T) {
	sender := mailtest.New()
	sender.Answer(mail.Outcome{Kind: mail.Failed, Reply: "550 5.1.1 Mailbox unavailable"})
	log, out := newLog()
	box := mail.NewOutbox(sender, log)
	box.Post(mail.Message{To: grace, Subject: "Your SlipStream PIN was turned off"})
	if err := box.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := sender.Sent(); len(got) != 1 || got[0].Subject != "Your SlipStream PIN was turned off" {
		t.Fatalf("sent %+v", got)
	}
	want := `level=WARN msg="mail not sent" to=grace@example.com subject="Your SlipStream PIN was turned off" outcome=failed reply="550 5.1.1 Mailbox unavailable"` + "\n"
	if out.String() != want {
		t.Fatalf("log = %q, want %q", out.String(), want)
	}
}

func TestTheOutboxSendsNothingAfterClose(t *testing.T) {
	sender := mailtest.New()
	log, out := newLog()
	box := mail.NewOutbox(sender, log)
	if err := box.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	box.Post(mail.Message{To: grace, Subject: "Your email changed"})
	if got := sender.Sent(); len(got) != 0 {
		t.Fatalf("sent %+v after Close", got)
	}
	want := `level=WARN msg="mail not sent" to=grace@example.com subject="Your email changed" reason="shutting down"` + "\n"
	if out.String() != want {
		t.Fatalf("log = %q, want %q", out.String(), want)
	}
}

type stuck struct{}

func (stuck) Send(ctx context.Context, _ mail.Message) (mail.Outcome, error) {
	<-ctx.Done()
	return mail.Outcome{Kind: mail.Failed, Reply: ctx.Err().Error()}, nil
}

func TestCloseCancelsSendsStillInFlightAtItsDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		log, out := newLog()
		box := mail.NewOutbox(stuck{}, log)
		box.Post(mail.Message{To: grace, Subject: "Confirm your new email"})
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		start := time.Now()
		if err := box.Close(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Close = %v", err)
		}
		if waited := time.Since(start); waited != 10*time.Second {
			t.Fatalf("Close returned after %v", waited)
		}
		want := `level=WARN msg="mail not sent" to=grace@example.com subject="Confirm your new email" outcome=failed reply="context canceled"` + "\n"
		if out.String() != want {
			t.Fatalf("log = %q, want %q", out.String(), want)
		}
	})
}
