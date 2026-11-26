package mediaservers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
	"github.com/jatassi/slipstream/internal/mediaservers/internal/db"
	msv1 "github.com/jatassi/slipstream/internal/pb/slipstream/mediaserver/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Sessions are what the server's reaching Node last saw playing, empty
// while that Node is away.
func (s service) Sessions(ctx context.Context, id link.ServerID) (Sessions, error) {
	var reaching role.NodeID
	err := s.m.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[storage.Library]) error {
		cur, err := db.New(r).GetContact(ctx, id)
		if errors.Is(err, storage.ErrNoRows) {
			return ErrNotFound
		}
		reaching = cur.ReachingNodeID
		return err
	})
	if err != nil {
		return Sessions{}, err
	}
	f, ok := s.m.cfg.Facts(reaching)
	out := Sessions{Reachable: ok, Sessions: []SessionView{}}
	for _, p := range s.m.caps().facts(serverFactsOf(f, id), time.Now()) {
		out.Sessions = append(out.Sessions, SessionView{
			PlayID: p.id.String(), User: p.user, Player: p.player, Product: p.product, Path: p.path.String(),
			Decision: string(p.streams.Decision()), Video: string(p.streams.Video), Audio: string(p.streams.Audio),
			Subtitles: string(p.streams.Subtitles), Hardware: p.hardware, Reasons: p.reasons, StartedAt: p.started,
		})
	}
	return out, nil
}

func serverFactsOf(f *msv1.Facts, id link.ServerID) *msv1.ServerFacts {
	for _, sf := range f.GetServers() {
		if bytes.Equal(sf.GetServerId(), id.Bytes()) {
			return sf
		}
	}
	return nil
}

// seenSessions is each Node's servers' sessions as
// media-servers.sessions-changed last told Admins, and the last looks at
// open plays not yet written.
type seenSessions struct {
	mu      sync.Mutex
	byNode  map[role.NodeID]map[string]string
	pending map[link.ServerID]looked
	wake    chan struct{}
}

// looked is the Node whose facts listed a server's plays, and when it
// last saw each.
type looked struct {
	node  role.NodeID
	plays map[link.PlayID]time.Time
}

func newSeenSessions() seenSessions {
	return seenSessions{byNode: map[role.NodeID]map[string]string{}, pending: map[link.ServerID]looked{}, wake: make(chan struct{}, 1)}
}

// Noted follows a Node's new facts: it publishes
// media-servers.sessions-changed for each server whose sessions on that
// Node started, changed or ended, or that left its facts.
func (m *MediaServers) Noted(node role.NodeID, scope string) {
	if scope != string(role.MediaServerLink) {
		return
	}
	f, _ := m.cfg.Facts(node)
	now := time.Now()
	next := map[string]string{}
	m.seen.mu.Lock()
	for _, sf := range f.GetServers() {
		u, err := storage.UUIDFromBytes(sf.GetServerId())
		if err != nil {
			continue
		}
		live := m.caps().facts(sf, now)
		next[u.String()] = digest(live)
		l := looked{node: node, plays: map[link.PlayID]time.Time{}}
		for _, p := range live {
			l.plays[p.id] = p.seen
		}
		if len(l.plays) > 0 {
			m.seen.pending[link.ServerID{UUID: u}] = l
		}
	}
	prev := m.seen.byNode[node]
	m.seen.byNode[node] = next
	m.seen.mu.Unlock()
	select {
	case m.seen.wake <- struct{}{}:
	default:
	}
	var changed []string
	for id, d := range next {
		if p, ok := prev[id]; !ok || p != d {
			changed = append(changed, id)
		}
	}
	for id := range prev {
		if _, ok := next[id]; !ok {
			changed = append(changed, id)
		}
	}
	if len(changed) > 0 {
		slices.Sort(changed)
		m.cfg.Events.SessionsChanged.Publish(context.Background(), SessionsChanged{ServerIDs: changed})
	}
}

// digest is what an Admin sees of a server's sessions: each play's
// streams, hardware, reasons, tracks and received facts but the bitrate,
// which Plex moves on every poll.
func digest(live []reported) string {
	var b strings.Builder
	for _, p := range live {
		b.Write(p.id.Bytes())
		st := p.streams
		fmt.Fprintf(&b, "%s/%s/%s/%t", st.Video, st.Audio, st.Subtitles, st.ContainerChanged)
		if p.hardware {
			b.WriteString("hw")
		}
		b.WriteString(strings.Join(p.reasons, ","))
		r := p.received
		fmt.Fprintf(&b, "|%s/%s/%d/%s/%d|%s|%s", r.Container, r.VideoCodec, r.Height, r.AudioCodec, r.Channels,
			trackKey(p.audio), trackKey(p.subtitle))
		b.WriteByte(0)
	}
	return b.String()
}

// writeSeen moves each open play's last look to the one its reaching
// Node's facts last gave, so a play the Controller ends without a report
// ends there.
func (m *MediaServers) writeSeen(ctx context.Context) {
	m.seen.mu.Lock()
	pending := m.seen.pending
	m.seen.pending = map[link.ServerID]looked{}
	m.seen.mu.Unlock()
	if len(pending) == 0 {
		return
	}
	err := m.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		q := db.New(tx)
		for server, l := range pending {
			if err := see(ctx, q, server, l); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && ctx.Err() == nil {
		m.cfg.Log.Error("record when Media Server plays were last seen", "err", err.Error())
	}
}

// see moves the last look of server's open plays to l's, while l's Node
// still reaches server.
func see(ctx context.Context, q *db.Queries, server link.ServerID, l looked) error {
	cur, err := q.GetContact(ctx, server)
	switch {
	case errors.Is(err, storage.ErrNoRows):
		return nil
	case err != nil:
		return err
	case cur.ReachingNodeID != l.node:
		return nil
	}
	for id, at := range l.plays {
		seen, open, err := laterSeen(ctx, q, server, id, at)
		if err != nil {
			return err
		}
		if !open {
			continue
		}
		if err := q.SeePlay(ctx, db.SeePlayParams{ServerID: server, ID: id, LastSeenAt: seen}); err != nil {
			return err
		}
	}
	return nil
}

func trackKey(t *link.Track) string {
	switch {
	case t == nil:
		return ""
	case t.External != nil:
		return t.External.Codec + "/" + t.External.Language
	}
	return strconv.Itoa(t.Position)
}
