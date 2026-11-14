package mediaservers

import (
	"bytes"
	"context"
	"errors"
	"slices"
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
	for _, sf := range f.GetServers() {
		if !bytes.Equal(sf.GetServerId(), id.Bytes()) {
			continue
		}
		for _, x := range sf.GetSessions() {
			v, err := sessionView(x)
			if err != nil {
				return Sessions{}, err
			}
			out.Sessions = append(out.Sessions, v)
		}
	}
	return out, nil
}

func sessionView(x *msv1.Session) (SessionView, error) {
	id, err := playIDOf(x.GetPlayId())
	if err != nil {
		return SessionView{}, err
	}
	return SessionView{
		PlayID: id.String(), User: x.GetUser(), Player: x.GetPlayer(), Product: x.GetProduct(), Path: x.GetServerPath(),
		Decision: x.GetDecision(), Hardware: x.GetHardware(), Reasons: append([]string{}, x.GetReasons()...),
		StartedAt: time.UnixMilli(x.GetStartedUnixMs()).UTC(),
	}, nil
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
	next := map[string]string{}
	m.seen.mu.Lock()
	for _, sf := range f.GetServers() {
		u, err := storage.UUIDFromBytes(sf.GetServerId())
		if err != nil {
			continue
		}
		next[u.String()] = digest(sf)
		l := looked{node: node, plays: map[link.PlayID]time.Time{}}
		for _, x := range sf.GetSessions() {
			if id, err := playIDOf(x.GetPlayId()); err == nil {
				l.plays[id] = timeOf(x.GetSeenUnixMs())
			}
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

// digest is what an Admin sees of a server's sessions.
func digest(sf *msv1.ServerFacts) string {
	var b strings.Builder
	for _, x := range sf.GetSessions() {
		b.Write(x.GetPlayId())
		b.WriteString(x.GetDecision())
		if x.GetHardware() {
			b.WriteString("hw")
		}
		b.WriteString(strings.Join(x.GetReasons(), ","))
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
