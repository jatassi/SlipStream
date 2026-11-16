// Package noticetest keeps what an owner records, for the owner's tests.
package noticetest

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
