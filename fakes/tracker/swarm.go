package tracker

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/internal/bencode"
)

// peerKey is one peer of one account's torrent, as UNIT3D keys its peers
// table: by user, torrent and peer ID.
type peerKey struct {
	passkey string
	hash    bencode.InfoHash
	peerID  string
}

// peer is what the peer's last accepted announce said: the totals it
// reported, whether it was seeding, and whether it hadn't stopped.
type peer struct {
	uploaded, downloaded int64
	seeder, active       bool
	at                   time.Time
	// startedAt is its last accepted started, zero before one.
	startedAt time.Time
}

// accept applies the accepted announce a, from origin at now, to the
// swarm and to a site account's history of the torrent. A BEP 15
// announce carries no passkey, so it keeps no history.
func (s *server) accept(a Announce, hash bencode.InfoHash, origin string, now time.Time) {
	key := peerKey{a.Passkey, hash, a.PeerID}
	p, known := s.peers[key]
	event := a.Event
	if !known {
		p = &peer{}
		s.peers[key] = p
	}
	up, down := max(a.Uploaded-p.uploaded, 0), max(a.Downloaded-p.downloaded, 0)
	// UNIT3D takes a peer it doesn't know that reports totals as started,
	// and credits a started announce nothing.
	if !known && (a.Uploaded > 0 || a.Downloaded > 0) {
		event = "started"
	}
	if event == "started" {
		up, down = 0, 0
	}
	startedAt := p.startedAt
	if event == "started" {
		startedAt = now
	}
	*p = peer{uploaded: a.Uploaded, downloaded: a.Downloaded, seeder: a.Left == 0, active: event != "stopped", at: now, startedAt: startedAt}
	switch {
	case event == "completed":
		s.completed[hash]++
	case event == "started":
		s.recordPeer(key, origin, Peer{StartedAt: now})
	case event == "stopped" && !startedAt.IsZero():
		s.recordPeer(key, origin, Peer{StartedAt: startedAt, StoppedAt: &now})
	}
	if account, ok := s.accounts[a.Passkey]; ok && account != releases.Seeders {
		s.upsert(historyKey{account, hash}, key, origin, p, up, down, now)
	}
}

// Peer is a peers record's data: a peer's last started, and its stopped
// once it sent one.
type Peer struct {
	Release   string     `json:"release"`
	InfoHash  string     `json:"infoHash"`
	PeerID    string     `json:"peerId"`
	StartedAt time.Time  `json:"startedAt"`
	StoppedAt *time.Time `json:"stoppedAt,omitempty"`
}

func (s *server) recordPeer(k peerKey, origin string, rec Peer) {
	rec.Release, rec.InfoHash, rec.PeerID = s.torrents[k.hash].release, k.hash.String(), k.peerID
	s.peerRecords.Record(fake.WithOrigin(context.Background(), origin), rec)
}

// seedsElsewhere reports whether a peer of k's passkey and torrent other
// than k's still seeds at now.
func (s *server) seedsElsewhere(k peerKey, now time.Time) bool {
	for other, p := range s.peers {
		if other.passkey == k.passkey && other.hash == k.hash && other.peerID != k.peerID &&
			p.active && p.seeder && now.Before(p.at.Add(s.hitrun.inactiveAfter)) {
			return true
		}
	}
	return false
}

// swarm is the torrent's seeders and leechers: the peers whose last
// announce hadn't stopped and came within the inactive time, and the
// seeders the feeds list.
func (s *server) swarm(hash bencode.InfoHash, now time.Time) (complete, incomplete int64) {
	complete = s.torrents[hash].listed
	for k, p := range s.peers {
		if k.hash != hash || !p.active || !now.Before(p.at.Add(s.hitrun.inactiveAfter)) {
			continue
		}
		if p.seeder {
			complete++
		} else {
			incomplete++
		}
	}
	return complete, incomplete
}
