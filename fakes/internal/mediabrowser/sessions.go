package mediabrowser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

// Keepalive timings Jellyfin's session socket uses: a ForceKeepAlive
// carrying the timeout once the client has been silent keepAliveAsk, and
// the socket dropped once silent keepAliveTimeout.
const (
	keepAliveAsk     = 45 * time.Second
	keepAliveTimeout = 60 * time.Second
)

var playMethods = map[link.Decision]string{link.DecisionDirectPlay: "DirectPlay", link.DecisionDirectStream: "Transcode", link.DecisionTranscode: "Transcode"}

// renderSession renders sess, whose User is the id the play cue looked
// up by name.
func (s *server) renderSession(sess mediaserver.Session) map[string]any {
	s.mu.Lock()
	name := ""
	if u, ok := s.userAt(sess.User); ok {
		name = u.name
	}
	s.mu.Unlock()
	item := s.renderItem(sess.Item, sess.User)
	m := map[string]any{
		"Id": sess.Key, "UserId": sess.User, "UserName": name, "Client": either(sess.Product, s.p.ProductName+" Web"),
		"DeviceName": sess.Player, "DeviceId": "fake-player-" + sess.Key, "RemoteEndPoint": "192.168.1.30", "IsActive": true,
		"NowPlayingItem": item,
		"PlayState":      map[string]any{"PlayMethod": playMethods[sess.Decision], "PositionTicks": 0, "MediaSourceId": id("source " + sess.Path.String())},
	}
	if sess.Decision == link.DecisionDirectPlay || (s.p.Flavor == Emby && sess.Unreported) {
		return m
	}
	t := map[string]any{
		"IsVideoDirect": sess.Decision == link.DecisionDirectStream, "IsAudioDirect": false, "VideoCodec": "h264", "AudioCodec": "aac",
		"TranscodeReasons": nonNilStrings(sess.Reasons),
	}
	if sess.Hardware {
		if s.p.Flavor == Emby {
			t["VideoDecoderIsHardware"], t["VideoEncoderIsHardware"] = true, true
		} else {
			t["HardwareAccelerationType"] = "vaapi"
		}
	} else if s.p.Flavor == Jellyfin {
		t["HardwareAccelerationType"] = "none"
	}
	m["TranscodingInfo"] = t
	return m
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func (s *server) sessionList() []map[string]any {
	out := []map[string]any{}
	for _, sess := range s.core.Sessions() {
		out = append(out, s.renderSession(sess))
	}
	return out
}

func (s *server) sessions(w http.ResponseWriter, _ *http.Request, _ caller) {
	writeJSON(w, s.sessionList())
}

// playing is a player's POST /Sessions/Playing. Its PlaySessionId names
// the session, and on Emby the session's transcode shows from then on.
func (s *server) playing(w http.ResponseWriter, r *http.Request, _ caller) {
	var in struct {
		PlaySessionID string `json:"PlaySessionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.core.Reported(in.PlaySessionID)
	w.WriteHeader(http.StatusNoContent)
}

type message struct {
	MessageType string `json:"MessageType"`
	Data        any    `json:"Data,omitempty"`
	MessageID   string `json:"MessageId,omitempty"`
}

type inbound struct {
	MessageType string          `json:"MessageType"`
	Data        json.RawMessage `json:"Data"`
}

// socket is Jellyfin's /socket or Emby's /embywebsocket. A refused
// Jellyfin socket answers 403, as ?api_key= did with legacy authorization
// off (#71). Once a SessionsStart subscribes, it pushes Sessions when a
// play or a signed-in client starts or ends, and at no other time. That
// is Jellyfin's SessionInfoWebSocketListener, which sends on session
// events alone, and neither Jellyfin 12.2, 12.4 nor Emby 4.10 sent one in
// five minutes with no session changing (online run 2026-11-14). Jellyfin
// asks for KeepAlive and drops a silent client. Emby pushes
// LibraryChanged on each item change; a Jellyfin key's socket gets none.
func (s *server) socket(w http.ResponseWriter, r *http.Request) {
	if s.srv == nil {
		http.NotFound(w, r)
		return
	}
	if _, ok := s.caller(r); !ok {
		status := http.StatusUnauthorized
		if s.p.Flavor == Jellyfin {
			status = http.StatusForbidden
		}
		w.WriteHeader(status)
		return
	}
	sub := s.core.Subscribe()
	defer sub.Cancel()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	in := make(chan inbound)
	go func() {
		defer cancel()
		for {
			var m inbound
			if err := wsjson.Read(ctx, conn, &m); err != nil {
				return
			}
			select {
			case in <- m:
			case <-ctx.Done():
				return
			}
		}
	}()
	(&push{s: s, conn: conn, sub: sub, in: in}).run(ctx)
}

type push struct {
	s       *server
	conn    *websocket.Conn
	sub     *mediaserver.Subscription
	in      <-chan inbound
	heard   time.Time
	asked   bool
	started bool
}

func (p *push) send(ctx context.Context, m message) bool {
	return wsjson.Write(ctx, p.conn, m) == nil
}

func (p *push) run(ctx context.Context) {
	p.heard = time.Now()
	var silence <-chan time.Time
	if p.s.p.Flavor == Jellyfin {
		if !p.forceKeepAlive(ctx) {
			return
		}
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		silence = t.C
	}
	for ok := true; ok; {
		select {
		case <-ctx.Done():
			return
		case <-p.sub.Done:
			_ = p.conn.Close(websocket.StatusGoingAway, "the server restarted")
			return
		case m := <-p.in:
			p.heard, p.asked = time.Now(), false
			p.take(m)
		case <-silence:
			ok = p.silent(ctx)
		case e := <-p.sub.C:
			ok = p.event(ctx, e)
		}
	}
}

func (p *push) forceKeepAlive(ctx context.Context) bool {
	return p.send(ctx, message{MessageType: "ForceKeepAlive", Data: int(keepAliveTimeout / time.Second)})
}

// silent asks a client silent keepAliveAsk for a KeepAlive, once, and
// drops it once silent keepAliveTimeout.
func (p *push) silent(ctx context.Context) bool {
	quiet := time.Since(p.heard)
	if quiet >= keepAliveTimeout {
		_ = p.conn.Close(websocket.StatusPolicyViolation, "no KeepAlive")
		return false
	}
	if quiet < keepAliveAsk || p.asked {
		return true
	}
	p.asked = true
	return p.forceKeepAlive(ctx)
}

// take handles one client message: SessionsStart "<dueMs>,<periodMs>",
// SessionsStop and KeepAlive.
func (p *push) take(m inbound) {
	switch m.MessageType {
	case "SessionsStart":
		var arg string
		if err := json.Unmarshal(m.Data, &arg); err != nil {
			return
		}
		_, after, _ := strings.Cut(arg, ",")
		if ms, err := strconv.Atoi(after); err == nil && ms > 0 {
			p.started = true
		}
	case "SessionsStop":
		p.started = false
	}
}

func (p *push) sessions(ctx context.Context) bool {
	return p.send(ctx, message{MessageType: "Sessions", Data: p.s.sessionList(), MessageID: id("message " + time.Now().String())})
}

func (p *push) event(ctx context.Context, e mediaserver.Event) bool {
	switch e.Kind {
	case mediaserver.ScanStarted, mediaserver.ScanEnded:
	case mediaserver.PlayStarted, mediaserver.PlayStopped, mediaserver.ClientStarted, mediaserver.ClientEnded:
		if p.started {
			return p.sessions(ctx)
		}
	case mediaserver.ItemAdded, mediaserver.ItemChanged, mediaserver.ItemRemoved:
		if p.s.p.Flavor != Emby {
			return true
		}
		changed := map[string][]string{"ItemsAdded": {}, "ItemsUpdated": {}, "ItemsRemoved": {}, "FoldersAddedTo": {}, "FoldersRemovedFrom": {}}
		field := map[mediaserver.EventKind]string{mediaserver.ItemAdded: "ItemsAdded", mediaserver.ItemChanged: "ItemsUpdated", mediaserver.ItemRemoved: "ItemsRemoved"}[e.Kind]
		changed[field] = []string{e.Item.Key}
		return p.send(ctx, message{MessageType: "LibraryChanged", Data: changed})
	}
	return true
}

// Play is a play cue's arguments: a session on the file at Path, as the
// server stores it, by the user named User. On Emby its transcode shows
// only once the player reports its playback.
type Play struct {
	Path     string        `json:"path"`
	User     string        `json:"user"`
	Player   string        `json:"player"`
	Product  string        `json:"product,omitempty"`
	Decision link.Decision `json:"decision"`
	Reasons  []string      `json:"reasons,omitempty"`
	Hardware bool          `json:"hardware,omitempty"`
}

// Played is a play cue's result: the session's Id, the PlaySessionId a
// player reports with.
type Played struct {
	Session string `json:"session"`
	Item    string `json:"item"`
}

type Stop struct {
	Session string `json:"session"`
}

func (s *server) cues(routes fake.Routes) error {
	n := s.p.Name
	if err := fake.OnCue(s.env, n+"-play", func(_ context.Context, a Play) (any, error) {
		p, err := link.ParsePath(a.Path)
		if err != nil {
			return nil, err
		}
		if _, err := link.ParseDecision(string(a.Decision)); err != nil {
			return nil, err
		}
		s.mu.Lock()
		u, ok := s.userNamed(a.User, nil)
		s.mu.Unlock()
		if !ok {
			return nil, fmt.Errorf("%s has no user %s", s.srv.Name, a.User)
		}
		sess, err := s.core.Start(mediaserver.Play{
			Path: p, User: u.id, Player: a.Player, Product: a.Product, Decision: a.Decision, Reasons: a.Reasons,
			Hardware: a.Hardware, Unreported: s.p.Flavor == Emby,
		})
		if err != nil {
			return nil, err
		}
		return Played{Session: sess.Key, Item: sess.Item.Key}, nil
	}); err != nil {
		return err
	}
	if err := fake.OnCue(s.env, n+"-stop", func(_ context.Context, a Stop) (any, error) {
		return map[string]bool{"stopped": s.core.Stop(a.Session)}, nil
	}); err != nil {
		return err
	}
	if err := fake.OnCue(s.env, n+"-restart", func(context.Context, struct{}) (any, error) {
		s.core.Restart()
		return map[string]bool{"restarted": true}, nil
	}); err != nil {
		return err
	}
	if err := fake.OnCue(s.env, n+"-scan", func(context.Context, struct{}) (any, error) {
		return map[string]any{"started": s.core.RefreshAll()}, nil
	}); err != nil {
		return err
	}
	return fake.OnCue(s.env, n+"-fail-next", func(_ context.Context, a FailNext) (any, error) {
		if _, ok := routes[a.Route]; !ok {
			return nil, fmt.Errorf("fake %s has no route %s", n, a.Route)
		}
		s.mu.Lock()
		s.failNext[a.Route] = true
		s.mu.Unlock()
		return map[string]string{"failing": a.Route}, nil
	})
}
