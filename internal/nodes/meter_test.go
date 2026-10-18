package nodes

import (
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	dlv1 "github.com/jatassi/slipstream/internal/pb/slipstream/downloading/v1"
)

func facts(torrents ...*dlv1.TorrentFacts) *dlv1.Facts { return &dlv1.Facts{Torrents: torrents} }

func torrent(h byte, done, uploaded int64) *dlv1.TorrentFacts {
	var hash bencode.InfoHash
	hash[0] = h
	return &dlv1.TorrentFacts{InfoHash: hash[:], Done: done, Uploaded: uploaded}
}

func TestAClientsRatesSpanItsLastTwoFacts(t *testing.T) {
	m := newMeter(2 * time.Second)
	m.arrive(box, start, facts(torrent(1, 0, 0)))
	if down, up := m.rates(box, start); down != 0 || up != 0 {
		t.Fatalf("after the first facts the rates are %d down and %d up, want none", down, up)
	}
	m.arrive(box, start.Add(500*time.Millisecond), facts(torrent(1, 1000, 250), torrent(2, 4000, 4000)))
	if down, up := m.rates(box, start.Add(time.Second)); down != 2000 || up != 500 {
		t.Fatalf("1000 bytes down and 250 up in half a second read %d down and %d up a second, want 2000 and 500; a torrent first seen adds nothing", down, up)
	}
	m.arrive(box, start.Add(1500*time.Millisecond), facts(torrent(2, 5000, 4100)))
	if down, up := m.rates(box, start.Add(2*time.Second)); down != 1000 || up != 100 {
		t.Fatalf("over the next second the rates read %d down and %d up, want 1000 and 100 from the torrent that stayed", down, up)
	}
	if down, up := m.rates(box, start.Add(3500*time.Millisecond)); down != 1000 || up != 100 {
		t.Fatalf("two seconds after the last facts the rates read %d and %d, want the last pair's", down, up)
	}
	if down, up := m.rates(box, start.Add(3501*time.Millisecond)); down != 0 || up != 0 {
		t.Fatalf("with no facts for longer than quiet the rates read %d and %d, want none: nothing moved", down, up)
	}
	m.arrive(box, start.Add(4*time.Second), facts(torrent(2, 0, 0)))
	if down, up := m.rates(box, start.Add(4*time.Second)); down != 0 || up != 0 {
		t.Fatalf("a torrent added again from zero reads %d down and %d up, want none", down, up)
	}
}

func TestALinksRatesAreItsCountersSinceTheLastSample(t *testing.T) {
	m := newMeter(2 * time.Second)
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
