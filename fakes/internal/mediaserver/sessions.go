package mediaserver

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// EventKind is what changed on the server, for its push socket.
type EventKind string

const (
	ItemAdded   EventKind = "item_added"
	ItemChanged EventKind = "item_changed"
	ItemRemoved EventKind = "item_removed"
	ScanStarted EventKind = "scan_started"
	ScanEnded   EventKind = "scan_ended"
	PlayStarted EventKind = "play_started"
	PlayStopped EventKind = "play_stopped"
)

// Event is one change on the server. Item is set for the item kinds and
// Session for the play kinds.
type Event struct {
	Kind    EventKind
	Library string
	Item    Item
	Session Session
}

// subscriptionBuffer is how many events a socket may lag behind before
// the server drops it, as a server drops a client that stops reading.
const subscriptionBuffer = 256

// Subscription is one push socket's feed. Done closes when the server
// drops it: on Cancel, on a restart, or when it lags too far behind.
type Subscription struct {
	C    <-chan Event
	Done <-chan struct{}
	c    chan Event
	done chan struct{}
	s    *Server
}

// Subscribe opens a feed of every event from now on.
func (s *Server) Subscribe() *Subscription {
	s.lock()
	defer s.mu.Unlock()
	c, done := make(chan Event, subscriptionBuffer), make(chan struct{})
	sub := &Subscription{C: c, Done: done, c: c, done: done, s: s}
	s.subs[sub] = struct{}{}
	return sub
}

func (sub *Subscription) Cancel() {
	sub.s.mu.Lock()
	defer sub.s.mu.Unlock()
	sub.s.drop(sub)
}

func (s *Server) drop(sub *Subscription) {
	if _, ok := s.subs[sub]; ok {
		delete(s.subs, sub)
		close(sub.done)
	}
}

func (s *Server) publish(e Event) {
	for sub := range s.subs {
		select {
		case sub.c <- e:
		default:
			s.drop(sub)
		}
	}
}

// Play is a playback a cue starts.
type Play struct {
	Path     link.ServerPath
	User     string
	Player   string
	Product  string
	Decision link.Decision
	Reasons  []string
	Hardware bool
	// Unreported is a client that never reported its timeline, which Plex
	// lists only in /transcode/sessions (#71).
	Unreported bool
}

// Session is one playback on the server.
type Session struct {
	Play
	Key     string
	Item    Item
	Started time.Time
}

// ErrNoItem is a call naming a path or key the server holds no item at.
var ErrNoItem = errors.New("no item")

// Start begins p on the item at its path.
func (s *Server) Start(p Play) (Session, error) {
	s.lock()
	defer s.mu.Unlock()
	it, _, ok := s.partAt(p.Path)
	if !ok {
		return Session{}, fmt.Errorf("%s: %w", p.Path, ErrNoItem)
	}
	s.sessionKeys++
	sess := &Session{Play: p, Key: strconv.Itoa(s.sessionKeys), Item: it.clone(), Started: time.Now()}
	sess.Reasons = slices.Clone(p.Reasons)
	s.sessions = append(s.sessions, sess)
	s.publish(Event{Kind: PlayStarted, Library: it.Library, Session: *sess})
	return *sess, nil
}

// Stop ends the session key, false when none plays.
func (s *Server) Stop(key string) bool {
	s.lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.sessions, func(x *Session) bool { return x.Key == key })
	if i < 0 {
		return false
	}
	sess := s.sessions[i]
	s.sessions = slices.Delete(s.sessions, i, i+1)
	s.publish(Event{Kind: PlayStopped, Library: sess.Item.Library, Session: *sess})
	return true
}

// Sessions are the playbacks now, in the order they started.
func (s *Server) Sessions() []Session {
	s.lock()
	defer s.mu.Unlock()
	out := make([]Session, len(s.sessions))
	for i, x := range s.sessions {
		out[i] = *x
	}
	return out
}

// Restart is the server restarting: every push socket drops, running
// scans stop without ending and pending reports are gone.
func (s *Server) Restart() {
	s.lock()
	defer s.mu.Unlock()
	for sub := range s.subs {
		s.drop(sub)
	}
	for _, l := range s.libs {
		if l.timer != nil {
			l.timer.Stop()
			l.timer, l.scan.Running = nil, false
		}
	}
	s.dropPending()
}
