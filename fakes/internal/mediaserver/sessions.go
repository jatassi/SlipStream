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
	// ClientStarted and ClientEnded are a client's session starting and
	// ending, as a sign-in and its logout do on Jellyfin and Emby.
	ClientStarted EventKind = "client_started"
	ClientEnded   EventKind = "client_ended"
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

// Client tells every push socket a client's session started or ended,
// with ClientStarted or ClientEnded.
func (s *Server) Client(k EventKind) {
	s.lock()
	defer s.mu.Unlock()
	s.publish(Event{Kind: k})
}

// Play is a playback a cue starts.
type Play struct {
	// Version is the file that plays, Path's when zero.
	Path, Version link.ServerPath
	User          string
	// Player.Key is stable across plays and required.
	Player  Player
	Streams link.Streams
	// Received is in the product's own names and units, as it reports them.
	Received Received
	// AudioTrack is a position among the version's embedded audio streams,
	// and SubtitleTrack among its embedded subtitles, or -1 for its first
	// Sidecar.
	AudioTrack, SubtitleTrack *int
	Reasons                   []string
	Hardware                  bool
	// Unreported is a client that hasn't reported its playback: Plex lists
	// it only in /transcode/sessions, and Emby gives it no transcode info
	// (#71).
	Unreported bool
}

type Player struct{ Key, Name, Product, Platform string }

// Received is what a transcode sends the player.
type Received struct {
	Container  string `json:"container,omitempty"`
	VideoCodec string `json:"videoCodec,omitempty"`
	Height     int    `json:"height,omitempty"`
	Bitrate    int64  `json:"bitrate,omitempty"`
	AudioCodec string `json:"audioCodec,omitempty"`
	Channels   int    `json:"channels,omitempty"`
}

// Session is one playback on the server.
type Session struct {
	Play
	Key     string
	Item    Item
	Started time.Time
}

var (
	// ErrNoItem is a call naming a path or key the server holds no item at.
	ErrNoItem = errors.New("no item")
	// ErrNoPlayerKey is a play whose player has no key.
	ErrNoPlayerKey = errors.New("the play names no player key")
)

// Start begins p on the item at its path.
func (s *Server) Start(p Play) (Session, error) {
	s.lock()
	defer s.mu.Unlock()
	if p.Player.Key == "" {
		return Session{}, ErrNoPlayerKey
	}
	it, _, ok := s.partAt(p.Path)
	if !ok {
		return Session{}, fmt.Errorf("%s: %w", p.Path, ErrNoItem)
	}
	if p.Version.IsZero() {
		p.Version = p.Path
	} else if _, _, ok := s.partAt(p.Version); !ok {
		return Session{}, fmt.Errorf("%s: %w", p.Version, ErrNoItem)
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

// Reported is the client of session key reporting its playback, false
// when none plays.
func (s *Server) Reported(key string) bool {
	s.lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.sessions, func(x *Session) bool { return x.Key == key })
	if i < 0 {
		return false
	}
	s.sessions[i].Unreported = false
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
	s.task.Running = false
	s.dropPending()
}
