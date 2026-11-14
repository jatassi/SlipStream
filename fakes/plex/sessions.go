package plex

import (
	"context"
	"net/http"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

type transcode struct {
	Key                  string `json:"key"`
	VideoDecision        string `json:"videoDecision"`
	AudioDecision        string `json:"audioDecision"`
	TranscodeHwRequested bool   `json:"transcodeHwRequested"`
	TranscodeHwDecoding  string `json:"transcodeHwDecoding,omitempty"`
	TranscodeHwEncoding  string `json:"transcodeHwEncoding,omitempty"`
}

func transcodeOf(sess mediaserver.Session, key string) *transcode {
	if sess.Decision == link.DecisionDirectPlay {
		return nil
	}
	t := &transcode{Key: key, VideoDecision: string(sess.Decision), AudioDecision: "copy", TranscodeHwRequested: sess.Hardware}
	if sess.Hardware {
		t.TranscodeHwDecoding, t.TranscodeHwEncoding = "vaapi", "vaapi"
	}
	return t
}

func transcodeKey(sess mediaserver.Session) string { return "fake-plex-transcode-" + sess.Key }

// sessions lists each session whose client reported its timeline. A
// transcode's part decision is "transcode" even for a remux (#71).
func (s *server) sessions(w http.ResponseWriter, _ *http.Request) {
	var out []metadata
	for _, sess := range s.core.Sessions() {
		if sess.Unreported {
			continue
		}
		m := s.render(sess.Item)
		m.SessionKey = sess.Key
		m.User = map[string]any{"id": "1", "title": sess.User}
		m.Player = map[string]any{"title": sess.Player, "product": sess.Product, "machineIdentifier": "fake-player-" + sess.Key, "state": "playing"}
		m.Session = map[string]any{"id": "fake-plex-session-" + sess.Key}
		m.TranscodeSession = transcodeOf(sess, "/transcode/sessions/"+transcodeKey(sess))
		decision := "directplay"
		if m.TranscodeSession != nil {
			decision = "transcode"
		}
		m.Media = filterPart(m.Media, sess.Path, decision)
		out = append(out, m)
	}
	writeJSON(w, container{map[string]any{"size": len(out), "Metadata": out}})
}

// filterPart keeps the one part the session plays.
func filterPart(ms []media, file link.ServerPath, decision string) []media {
	for _, m := range ms {
		for _, p := range m.Part {
			if strings.EqualFold(p.File, file.String()) {
				p.Decision = decision
				return []media{{ID: m.ID, Part: []part{p}}}
			}
		}
	}
	return ms
}

// transcodes lists every transcode, reported or not.
func (s *server) transcodes(w http.ResponseWriter, _ *http.Request) {
	var out []transcode
	for _, sess := range s.core.Sessions() {
		if t := transcodeOf(sess, transcodeKey(sess)); t != nil {
			out = append(out, *t)
		}
	}
	writeJSON(w, container{map[string]any{"size": len(out), "TranscodeSession": out}})
}

type notification struct {
	NotificationContainer map[string]any
}

// notifications pushes timeline, activity and playing notifications until
// the client leaves, the server restarts or the fake stops.
func (s *server) notifications(w http.ResponseWriter, r *http.Request) {
	sub := s.core.Subscribe()
	defer sub.Cancel()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer func() { _ = conn.CloseNow() }()
	ctx := conn.CloseRead(r.Context())
	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.Done:
			_ = conn.Close(websocket.StatusGoingAway, "the server restarted")
			return
		case e := <-sub.C:
			if n, ok := s.notify(e); ok {
				if err := wsjson.Write(ctx, conn, n); err != nil {
					return
				}
			}
		}
	}
}

var (
	timelineStates = map[mediaserver.EventKind]struct {
		state    int
		metadata string
	}{
		mediaserver.ItemAdded:   {5, "created"},
		mediaserver.ItemChanged: {5, ""},
		mediaserver.ItemRemoved: {9, "deleted"},
	}
	activityEvents = map[mediaserver.EventKind]string{mediaserver.ScanStarted: "started", mediaserver.ScanEnded: "ended"}
	playStates     = map[mediaserver.EventKind]string{mediaserver.PlayStarted: "playing", mediaserver.PlayStopped: "stopped"}
)

func (s *server) notify(e mediaserver.Event) (notification, bool) {
	if ts, ok := timelineStates[e.Kind]; ok {
		entry := map[string]any{
			"identifier": "com.plexapp.plugins.library", "sectionID": e.Library, "itemID": e.Item.Key,
			"type": plexType(e.Item), "title": e.Item.Title, "state": ts.state,
		}
		if ts.metadata != "" {
			entry["metadataState"] = ts.metadata
		}
		return pushed("timeline", "TimelineEntry", entry), true
	}
	if event, ok := activityEvents[e.Kind]; ok {
		uuid := "fake-plex-scan-" + e.Library
		return pushed("activity", "ActivityNotification", map[string]any{
			"event": event, "uuid": uuid,
			"Activity": map[string]any{
				"uuid": uuid, "type": "library.update.section", "cancellable": false, "title": "Scanning", "progress": 0,
				"Context": map[string]any{"librarySectionID": e.Library},
			},
		}), true
	}
	state, ok := playStates[e.Kind]
	if !ok {
		return notification{}, false
	}
	entry := map[string]any{
		"sessionKey": e.Session.Key, "clientIdentifier": "fake-player-" + e.Session.Key, "ratingKey": e.Session.Item.Key,
		"key": "/library/metadata/" + e.Session.Item.Key, "state": state, "viewOffset": 0,
	}
	if e.Session.Decision != link.DecisionDirectPlay {
		entry["transcodeSession"] = transcodeKey(e.Session)
	}
	return pushed("playing", "PlaySessionStateNotification", entry), true
}

func pushed(typ, field string, entry map[string]any) notification {
	return notification{NotificationContainer: map[string]any{"type": typ, "size": 1, field: []map[string]any{entry}}}
}

// Play is a plex-play cue's arguments: a session on the file at Path, as
// the server stores it.
type Play struct {
	Path     string        `json:"path"`
	User     string        `json:"user"`
	Player   string        `json:"player"`
	Product  string        `json:"product,omitempty"`
	Decision link.Decision `json:"decision"`
	Hardware bool          `json:"hardware,omitempty"`
	// Unreported is a client that never sent its timeline, which only
	// /transcode/sessions lists.
	Unreported bool `json:"unreported,omitempty"`
}

// Played is a plex-play cue's result.
type Played struct {
	Session string `json:"session"`
}

type Stop struct {
	Session string `json:"session"`
}

type Scan struct {
	Section string `json:"section"`
}

func (s *server) cues() error {
	if err := fake.OnCue(s.env, "plex-play", func(_ context.Context, a Play) (any, error) {
		p, err := link.ParsePath(a.Path)
		if err != nil {
			return nil, err
		}
		if _, err := link.ParseDecision(string(a.Decision)); err != nil {
			return nil, err
		}
		sess, err := s.core.Start(mediaserver.Play{
			Path: p, User: a.User, Player: a.Player, Product: a.Product, Decision: a.Decision, Hardware: a.Hardware, Unreported: a.Unreported,
		})
		if err != nil {
			return nil, err
		}
		return Played{Session: sess.Key}, nil
	}); err != nil {
		return err
	}
	if err := fake.OnCue(s.env, "plex-stop", func(_ context.Context, a Stop) (any, error) {
		return map[string]bool{"stopped": s.core.Stop(a.Session)}, nil
	}); err != nil {
		return err
	}
	if err := fake.OnCue(s.env, "plex-restart", func(context.Context, struct{}) (any, error) {
		s.core.Restart()
		return map[string]bool{"restarted": true}, nil
	}); err != nil {
		return err
	}
	return fake.OnCue(s.env, "plex-scan", func(_ context.Context, a Scan) (any, error) {
		started, err := s.core.RefreshLibrary(a.Section)
		if err != nil {
			return nil, err
		}
		return map[string]any{"started": started}, nil
	})
}
