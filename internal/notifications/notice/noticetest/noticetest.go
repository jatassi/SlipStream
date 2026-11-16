// Package noticetest keeps what an owner records, for the owner's tests.
package noticetest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/storage"
)

// Recorder keeps every event an owner records.
type Recorder struct {
	mu     sync.Mutex
	events []kept
}

type kept struct {
	kind    notice.Kind
	subject []byte
}

// New answers the Recorder to put in the owner's test Config, which refuses
// a kind not in rows the way notifications does, and the keeper to assert
// on.
func New(owner string, rows []notice.Entry) (rec notice.Recorder, kept *Recorder) {
	kept = &Recorder{}
	return notice.Sink(owner, rows, kept.put), kept
}

// put keeps the event at once, even from a Write that later rolls back.
func (r *Recorder) put(_ context.Context, _ *storage.Tx[storage.Library], k notice.Kind, subject []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, kept{kind: k, subject: subject})
	return nil
}

// Of decodes the recorded events of row's kind, oldest first.
func Of[F any](r *Recorder, row notice.Row[F]) []F {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []F
	for _, e := range r.events {
		if e.kind != row.Kind {
			continue
		}
		var f F
		if err := json.Unmarshal(e.subject, &f); err != nil {
			panic(fmt.Sprintf("noticetest: decode a %s event: %v", row.Kind, err))
		}
		out = append(out, f)
	}
	return out
}

// Context is the moment a golden renders at: 14:02 on 16 November 2026 in
// UTC, sent by Den.
var Context = notice.Context{Now: time.Date(2026, 11, 16, 14, 2, 0, 0, time.UTC), Zone: time.UTC, Sender: "Den"}

// Golden renders the facts stored at dir/<name>.json through e and checks
// the message against dir/<name>.txt. The stored facts are written from
// facts only when missing, so a later change to the facts' shape has to
// decode what is already stored. update rewrites the message.
func Golden(t testing.TB, e notice.Entry, facts any, dir, name string, update bool) {
	t.Helper()
	stored := filepath.Join(dir, name+".json")
	subject, err := os.ReadFile(stored) //nolint:gosec // a fixture the test names
	if errors.Is(err, fs.ErrNotExist) {
		if subject, err = json.MarshalIndent(facts, "", "  "); err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(stored, append(subject, '\n'), 0o644) //nolint:gosec // a fixture the test names
	}
	if err != nil {
		t.Fatal(err)
	}
	m, err := e.RenderStored(subject, Context)
	if err != nil {
		t.Fatal(err)
	}
	got := Text(m)
	path := filepath.Join(dir, name+".txt")
	if update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil { //nolint:gosec // a golden file the test names
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // a golden file the test names
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s is stale; rerun with -update and review the diff\ngot:\n%s", path, got)
	}
}

// Text is m as a golden file reads it.
func Text(m notice.Message) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", m.Title)
	for _, l := range m.Lines {
		fmt.Fprintf(&b, "Line: %s\n", l)
	}
	fmt.Fprintf(&b, "Link: %s\nTone: %s\n", m.Link.Path, [...]string{"info", "good", "warn", "fail"}[m.Tone])
	if m.Poster.Path != "" {
		fmt.Fprintf(&b, "Poster: %s %s\n", m.Poster.Source, m.Poster.Path)
	}
	if m.Body.Work != nil {
		fmt.Fprintf(&b, "Work: %s\n", m.Body.Work.Title)
	}
	for _, f := range m.Body.Files {
		fmt.Fprintf(&b, "File: %s %s replaced %q by %s\n", strings.Join(f.Items, ", "), f.Quality, f.Replaced, f.PlacedBy)
	}
	return b.String()
}
