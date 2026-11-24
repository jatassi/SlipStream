// Package entrytest keeps what an owner records, for the owner's tests.
package entrytest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/storage"
)

// Recorder keeps every entry an owner records.
type Recorder struct {
	mu     sync.Mutex
	events []kept
}

type kept struct {
	kind    entry.Kind
	subject []byte
	about   entry.About
}

// New answers the Recorder to put in the owner's test Config, which refuses
// a kind not in rows the way notifications does, and the keeper to assert
// on.
func New(owner string, rows []entry.Entry) (rec entry.Recorder, kept *Recorder) {
	kept = &Recorder{}
	return entry.Sink(owner, rows, kept.put), kept
}

// put keeps the entry at once, even from a Write that later rolls back.
func (r *Recorder) put(_ context.Context, _ *storage.Tx[storage.Library], k entry.Kind, subject []byte, a entry.About) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, kept{kind: k, subject: subject, about: a})
	return nil
}

// Len is how many entries r keeps, of any kind.
func (r *Recorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

// Abouts is what each recorded entry of row's kind is about, oldest first.
func Abouts[F entry.Facts](r *Recorder, row entry.Row[F]) []entry.About {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []entry.About
	for _, e := range r.events {
		if e.kind == row.Kind {
			out = append(out, e.about)
		}
	}
	return out
}

// Of decodes the recorded entries of row's kind, oldest first.
func Of[F entry.Facts](r *Recorder, row entry.Row[F]) []F {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []F
	for _, e := range r.events {
		if e.kind != row.Kind {
			continue
		}
		var f F
		if err := json.Unmarshal(e.subject, &f); err != nil {
			panic(fmt.Sprintf("entrytest: decode a %s entry: %v", row.Kind, err))
		}
		out = append(out, f)
	}
	return out
}

// Context is the moment a golden renders at: 14:02 on 16 November 2026 in
// UTC, sent by Den.
var Context = entry.Context{Now: time.Date(2026, 11, 16, 14, 2, 0, 0, time.UTC), Zone: time.UTC, Sender: "Den"}

// Stored renders the facts stored at dir/<name>.json through e and checks
// the message has a title, no empty line, and names each of names in its
// title, lines or link. The stored facts are written from facts only when
// missing, so a later change to the facts' shape has to decode what is
// already stored.
func Stored(t testing.TB, e entry.Entry, facts any, dir, name string, names ...string) entry.Message {
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
	if d := e.Declared(); d.Label == "" {
		t.Fatalf("%s has no Label", d.Kind)
	}
	m, err := e.RenderStored(subject, Context)
	if err != nil {
		t.Fatal(err)
	}
	read := append([]string{m.Title}, m.Lines...)
	if m.Title == "" || slices.Contains(m.Lines, "") {
		t.Errorf("%s reads %q, want a title and no empty line", name, read)
	}
	all := strings.Join(append(read, m.Link.Path), "\n")
	for _, n := range names {
		if !strings.Contains(all, n) {
			t.Errorf("%s reads %q linking %q, want it to name %q", name, read, m.Link.Path, n)
		}
	}
	return m
}

// Ignored is a Recorder for an owner whose entries a test doesn't read. It
// still refuses a kind not in rows.
func Ignored(owner string, rows []entry.Entry) entry.Recorder {
	rec, _ := New(owner, rows)
	return rec
}
