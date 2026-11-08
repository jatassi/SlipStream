package tracker

import (
	"time"

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
	*p = peer{uploaded: a.Uploaded, downloaded: a.Downloaded, seeder: a.Left == 0, active: event != "stopped", at: now}
	if event == "completed" {
		s.completed[hash]++
	}
	if account, ok := s.accounts[a.Passkey]; ok && account != releases.Seeders {
		s.upsert(historyKey{account, hash}, origin, p, up, down, now)
	}
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
