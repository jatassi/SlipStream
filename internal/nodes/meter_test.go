package nodes

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/link"
)

func TestALinksRatesAreItsCountersSinceTheLastSample(t *testing.T) {
	m := newMeter()
	if sent, received := m.link(box, start, link.Stats{Sent: 5000, Received: 9000}); sent != 0 || received != 0 {
		t.Fatalf("the first sample reads %d sent and %d received a second, want none", sent, received)
	}
	if sent, received := m.link(box, start.Add(2*time.Second), link.Stats{Sent: 7000, Received: 10000}); sent != 1000 || received != 500 {
		t.Fatalf("2000 bytes sent and 1000 received over two seconds read %d and %d a second, want 1000 and 500", sent, received)
	}
	if sent, received := m.link(box, start.Add(3*time.Second), link.Stats{Sent: 100, Received: 100}); sent != 0 || received != 0 {
		t.Fatalf("a new session's counters read %d and %d a second, want none", sent, received)
	}
	m.linkDown(box)
	if sent, received := m.link(box, start.Add(4*time.Second), link.Stats{Sent: 900, Received: 900}); sent != 0 || received != 0 {
		t.Fatalf("the first sample after the link went down reads %d and %d, want none", sent, received)
	}
}
