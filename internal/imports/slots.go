package imports

import (
	"context"
	"sync"

	"github.com/jatassi/slipstream/internal/role"
)

// slots is library.scan.probes per Node, shared by every scan unit and
// every Import probing there.
type slots struct {
	mu    sync.Mutex
	inUse map[role.NodeID]int
	freed chan struct{}
}

func newSlots() *slots { return &slots{inUse: map[role.NodeID]int{}, freed: make(chan struct{})} }

// take waits for one of node's limit slots, false when ctx ends first.
// The limit is read by each taker, so a lowered setting holds new probes
// until the running ones finish.
func (s *slots) take(ctx context.Context, node role.NodeID, limit int) bool {
	for {
		s.mu.Lock()
		if s.inUse[node] < limit {
			s.inUse[node]++
			s.mu.Unlock()
			return true
		}
		freed := s.freed
		s.mu.Unlock()
		select {
		case <-freed:
		case <-ctx.Done():
			return false
		}
	}
}

func (s *slots) give(node role.NodeID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inUse[node]--
	close(s.freed)
	s.freed = make(chan struct{})
}
