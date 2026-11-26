package mediaservers

import (
	"context"
	"fmt"
	"sync"

	"github.com/jatassi/slipstream/internal/identity"
)

// turns serializes every access change of one Account, and of one row
// whose Account is gone. A second caller waits, reads the row again and
// finds it moved. Leaving holds an Account's turn until identity's delete
// Write ends. A turn isn't re-entrant.
type turns struct {
	mu   sync.Mutex
	held map[turn]chan struct{}
}

type turn struct {
	// account is zero for a row whose Account is gone.
	account identity.AccountID
	row     AccessID
}

func accountTurn(a identity.AccountID) turn { return turn{account: a} }

// take waits for k to be free and holds it until release.
func (t *turns) take(ctx context.Context, k turn) (release func(), err error) {
	for {
		t.mu.Lock()
		busy, ok := t.held[k]
		if !ok {
			if t.held == nil {
				t.held = map[turn]chan struct{}{}
			}
			done := make(chan struct{})
			t.held[k] = done
			t.mu.Unlock()
			return func() {
				t.mu.Lock()
				delete(t.held, k)
				t.mu.Unlock()
				close(done)
			}, nil
		}
		t.mu.Unlock()
		select {
		case <-busy:
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for a Member's access to change: %w", ctx.Err())
		}
	}
}
