// Package mailtest is a capturing mail.Sender and golden files for
// rendered emails.
package mailtest

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/storage"
)

// Sender keeps every message and answers the outcome it was last told to,
// Sent until then.
type Sender struct {
	mu      sync.Mutex
	answer  mail.Outcome
	sent    []mail.Message
	taken   []bool
	arrived chan struct{}
}

var _ mail.Sender = (*Sender)(nil)

func New() *Sender {
	return &Sender{answer: mail.Outcome{Kind: mail.Sent}, arrived: make(chan struct{})}
}

// Answer is the outcome of every Send from now on.
func (s *Sender) Answer(o mail.Outcome) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.answer = o
}

func (s *Sender) Send(ctx context.Context, m mail.Message) (mail.Outcome, error) {
	if storage.InWrite(ctx) {
		panic("mail: Send inside Write; send after the commit")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, m)
	s.taken = append(s.taken, false)
	close(s.arrived)
	s.arrived = make(chan struct{})
	return s.answer, nil
}

func (s *Sender) Sent() []mail.Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.sent)
}

// Wait is the first message to the email to that no Wait has answered,
// waiting for it to be sent until the test ends.
func (s *Sender) Wait(t testing.TB, to string) mail.Message {
	t.Helper()
	for {
		s.mu.Lock()
		for i, m := range s.sent {
			if !s.taken[i] && m.To.Email == to {
				s.taken[i] = true
				s.mu.Unlock()
				return m
			}
		}
		arrived := s.arrived
		s.mu.Unlock()
		select {
		case <-arrived:
		case <-t.Context().Done():
			t.Fatalf("no mail to %s", to)
		}
	}
}

// Golden checks m against dir/<name>.html and dir/<name>.txt, whose text
// starts with the subject. update rewrites them instead.
func Golden(t testing.TB, m mail.Message, dir, name string, update bool) {
	t.Helper()
	parts := map[string]string{
		name + ".html": m.HTML,
		name + ".txt":  "Subject: " + m.Subject + "\n\n" + m.Text,
	}
	for file, got := range parts {
		path := filepath.Join(dir, file)
		if update {
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil { //nolint:gosec // a golden file the test names
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path) //nolint:gosec // a golden file the test names
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("%s is stale; rerun with -update and review the diff\ngot:\n%s", path, got)
		}
	}
}
