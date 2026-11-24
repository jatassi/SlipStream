package reachability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/reachability/judge"
)

// echoes are the live single-use answers a test of the UI and portal
// fetches. They live in memory only: a restart mid-test leaves that test's
// identity check Not tested, which is honest.
type echoes struct {
	mu   sync.Mutex
	live map[string]*echo
}

type echo struct {
	answer  string
	until   time.Time
	arrived *judge.Arrival
}

func token() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// issue draws an id and its answer, kept until now+life.
func (e *echoes) issue(now time.Time, life time.Duration) (id, answer string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for k, x := range e.live {
		if now.After(x.until) {
			delete(e.live, k)
		}
	}
	id, answer = token(), token()
	e.live[id] = &echo{answer: answer, until: now.Add(life)}
	return id, answer
}

// fetch answers id's answer once and records how the fetch arrived.
func (e *echoes) fetch(id string, a judge.Arrival, now time.Time) (string, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	x, ok := e.live[id]
	if !ok || x.arrived != nil || now.After(x.until) {
		return "", false
	}
	x.arrived = &a
	return x.answer, true
}

// take ends id and answers how its fetch arrived, nil when none did.
func (e *echoes) take(id string) *judge.Arrival {
	e.mu.Lock()
	defer e.mu.Unlock()
	x := e.live[id]
	delete(e.live, id)
	if x == nil {
		return nil
	}
	return x.arrived
}

// Echo answers GET /remote-access/echo/{id}: the answer once, then
// nothing. It records the arrival, which judge.AcceptEcho weighs.
func (r *Reachability) Echo(_ context.Context, id string, a api.Arrival) (string, bool) {
	at := a.Via
	if at.IsZero() {
		at = r.cfg.Roster.Self()
	}
	return r.echoes.fetch(id, judge.Arrival{At: at, Via: a.Via, Source: a.Source}, r.now())
}
