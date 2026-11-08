package tracker

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/internal/bencode"
)

// historyGap is the longest gap between two seeder announces UNIT3D
// still credits as seed time: its AutoUpsertHistories' 5400 seconds. The
// fake keeps its own copy, so its count shares nothing with SlipStream's.
const historyGap = 5400 * time.Second

// hitrunBuffer is the percent of a torrent's size an account must have
// downloaded before UNIT3D holds it to the site's rule.
const hitrunBuffer = 10

type hitrunTimings struct {
	prewarn, grace, inactiveAfter time.Duration
}

// historyKey is one account's torrent, as UNIT3D keys its history table:
// by user and torrent.
type historyKey struct {
	account string
	hash    bencode.InfoHash
}

// history is UNIT3D's history row: what the account's peers of the
// torrent last said, the seed time credited, and the hit and run flags.
type history struct {
	seeder, active       bool
	updatedAt            time.Time
	seedtime             time.Duration
	uploaded, downloaded int64
	prewarnedAt          time.Time
	hitrun               bool
	// origin is the origin of the account's last announce of the torrent,
	// which its records name.
	origin   string
	timer    *time.Timer
	recorded History
}

// History is a seeding record's data: an account's history of one
// torrent. Owes is whether the account has downloaded more than UNIT3D's
// buffer of it and meets neither its site's ratio nor its seed time.
type History struct {
	Account     string `json:"account"`
	Release     string `json:"release"`
	InfoHash    string `json:"infoHash"`
	SeedSeconds int64  `json:"seedSeconds"`
	Uploaded    int64  `json:"uploaded"`
	Downloaded  int64  `json:"downloaded"`
	Seeder      bool   `json:"seeder"`
	Active      bool   `json:"active"`
	Owes        bool   `json:"owes"`
	Prewarned   bool   `json:"prewarned"`
	HitAndRun   bool   `json:"hitAndRun"`
}

// upsert is AutoUpsertHistories for one accepted announce by peer p: it
// credits the time since the history's last announce when that came less
// than historyGap ago from an active seeder and p still seeds, then takes
// p's seeder and active and adds the transfer.
func (s *server) upsert(k historyKey, origin string, p *peer, up, down int64, now time.Time) {
	h, ok := s.histories[k]
	if !ok {
		h = &history{}
		s.histories[k] = h
	}
	if ok && h.updatedAt.Add(historyGap).After(now) && h.seeder && h.active && p.seeder {
		h.seedtime += now.Sub(h.updatedAt).Truncate(time.Second)
	}
	h.seeder, h.active, h.updatedAt, h.origin = p.seeder, p.active, now, origin
	h.uploaded += up
	h.downloaded += down
	s.settle(k, h, now)
}

// owes reports whether h is held to its site's rule and meets neither
// half of it.
func (s *server) owes(k historyKey, h *history) bool {
	rule, stated := s.tracker.Rule(k.account)
	if !stated || h.downloaded*100 <= s.torrents[k.hash].size*hitrunBuffer {
		return false
	}
	ratio := rule.MinRatio > 0 && float64(h.uploaded) >= rule.MinRatio*float64(h.downloaded)
	seeded := rule.MinSeedTime > 0 && h.seedtime >= rule.MinSeedTime
	return !ratio && !seeded
}

// settle plays UNIT3D's scheduled commands on h as they stand at now: the
// ghost peer flush, the prewarning and the hit and run. It records h when
// it changed, and sets h's timer for the next time a command would change
// it. Each command leaves updated_at alone, as UNIT3D's do.
func (s *server) settle(k historyKey, h *history, now time.Time) {
	t := s.hitrun
	owes := s.owes(k, h)
	if h.active && !now.Before(h.updatedAt.Add(t.inactiveAfter)) {
		h.active = false
	}
	if !h.active && owes && !h.hitrun {
		if h.prewarnedAt.IsZero() && !now.Before(h.updatedAt.Add(t.prewarn)) {
			h.prewarnedAt = now
		}
		if !h.prewarnedAt.IsZero() && !now.Before(h.prewarnedAt.Add(t.prewarn)) && !now.Before(h.updatedAt.Add(t.grace)) {
			h.hitrun = true
		}
	}
	if rec := s.view(k, h, owes); rec != h.recorded {
		h.recorded = rec
		s.seeding.Record(fake.WithOrigin(context.Background(), h.origin), rec)
	}
	if h.timer != nil {
		h.timer.Stop()
	}
	var next time.Time
	switch {
	case h.active:
		next = h.updatedAt.Add(t.inactiveAfter)
	case !owes || h.hitrun:
		return
	case h.prewarnedAt.IsZero():
		next = h.updatedAt.Add(t.prewarn)
	default:
		next = h.prewarnedAt.Add(t.prewarn)
		if grace := h.updatedAt.Add(t.grace); grace.After(next) {
			next = grace
		}
	}
	h.timer = time.AfterFunc(next.Sub(now), func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.closed {
			s.settle(k, h, time.Now())
		}
	})
}

func (s *server) view(k historyKey, h *history, owes bool) History {
	return History{
		Account: k.account, Release: s.torrents[k.hash].release, InfoHash: k.hash.String(),
		SeedSeconds: int64(h.seedtime / time.Second), Uploaded: h.uploaded, Downloaded: h.downloaded,
		Seeder: h.seeder, Active: h.active, Owes: owes, Prewarned: !h.prewarnedAt.IsZero(), HitAndRun: h.hitrun,
	}
}

func (s *server) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	for _, h := range s.histories {
		if h.timer != nil {
			h.timer.Stop()
		}
	}
}
