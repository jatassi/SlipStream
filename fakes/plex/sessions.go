package plex

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/internal/mediaserver"
	"github.com/jatassi/slipstream/fakes/internal/serverfake"
	"github.com/jatassi/slipstream/internal/mediaserverlink/link"
)

type transcode struct {
	Key                  string `json:"key"`
	VideoDecision        string `json:"videoDecision"`
	AudioDecision        string `json:"audioDecision"`
	SubtitleDecision     string `json:"subtitleDecision,omitempty"`
	Container            string `json:"container,omitempty"`
	VideoCodec           string `json:"videoCodec,omitempty"`
	AudioCodec           string `json:"audioCodec,omitempty"`
	Height               int    `json:"height,omitempty"`
	AudioChannels        int    `json:"audioChannels,omitempty"`
	TranscodeHwRequested bool   `json:"transcodeHwRequested"`
	TranscodeHwDecoding  string `json:"transcodeHwDecoding,omitempty"`
	TranscodeHwEncoding  string `json:"transcodeHwEncoding,omitempty"`
}

// transcodeOf is the TranscodeSession of sess, nil for a direct play, as
// the recorded run of 2026-11-26 read it: copy or transcode per stream, a
// burn re-encoding the video, no subtitleDecision but burn, and no height
// once the video is re-encoded.
func transcodeOf(sess mediaserver.Session, key string) *transcode {
	st := sess.Streams
	if st.Decision() == link.DecisionDirectPlay {
		return nil
	}
	word := map[bool]string{false: "copy", true: "transcode"}
	video := st.Video == link.StreamTranscode || st.Subtitles == link.SubtitlesBurned
	rc := sess.Received
	t := &transcode{
		Key: key, VideoDecision: word[video], AudioDecision: word[st.Audio == link.StreamTranscode],
		Container: rc.Container, VideoCodec: rc.VideoCodec, AudioCodec: rc.AudioCodec, AudioChannels: rc.Channels,
		TranscodeHwRequested: sess.Hardware,
	}
	if st.Subtitles == link.SubtitlesBurned {
		t.SubtitleDecision = "burn"
	}
	if !video {
		t.Height = rc.Height
	}
	if sess.Hardware {
		t.TranscodeHwDecoding, t.TranscodeHwEncoding = "vaapi", "vaapi"
	}
	return t
}

func transcodeKey(sess mediaserver.Session) string { return "fake-plex-transcode-" + sess.Key }

// sessions lists each session whose client reported its timeline, as the
// recorded run of 2026-11-26 read it: the item that holds the version,
// every Media of it with only the played one selected, and the player's
// machineIdentifier its key. A direct play carries the TranscodeSession
// of its player's last transcode, as a real Plex's did.
func (s *server) sessions(w http.ResponseWriter, _ *http.Request) {
	out := []metadata{}
	for _, sess := range s.core.Sessions() {
		it, ok := s.core.ItemAt(sess.Version)
		if sess.Unreported || !ok {
			continue
		}
		m := s.render(it)
		m.SessionKey = sess.Key
		m.User = map[string]any{"id": "1", "title": sess.User}
		m.Player = map[string]any{
			"title": sess.Player.Name, "product": cmp.Or(sess.Player.Product, "Plex Web"), "platform": sess.Player.Platform,
			"machineIdentifier": sess.Player.Key, "state": "playing",
		}
		m.Session = map[string]any{"id": "fake-plex-session-" + sess.Key}
		t := transcodeOf(sess, "/transcode/sessions/"+transcodeKey(sess))
		s.mu.Lock()
		m.TranscodeSession = cmp.Or(t, s.stale[sess.Player.Key])
		selected := !s.unselected[sess.Key]
		s.mu.Unlock()
		m.Media = playing(m.Media, sess, t != nil, selected)
		out = append(out, m)
	}
	writeJSON(w, container{map[string]any{"size": len(out), "Metadata": out}})
	s.core.Listed()
}

// playing is ms as a session lists them: the played Media is what the
// player receives, its Part only the streams that play, with no file
// while Plex transcodes, and the others as their item holds them, with no
// streams.
func playing(ms []media, sess mediaserver.Session, transcoding, selected bool) []media {
	out := make([]media, len(ms))
	for i, m := range ms {
		p := m.Part[0]
		if !strings.EqualFold(p.File, sess.Version.String()) {
			p.Stream = nil
			p.ID = p.ID.quote()
			out[i] = media{ID: m.ID.quote(), Part: []part{p}}
			continue
		}
		pt := part{ID: p.ID.quote(), Selected: selected, Stream: played(p.Stream, sess, transcoding), Decision: "directplay"}
		mm := media{ID: m.ID.quote(), Selected: selected}
		if transcoding {
			rc := sess.Received
			pt.Decision, mm.Container, mm.Bitrate, mm.Height = "transcode", rc.Container, rc.Bitrate, rc.Height
		} else {
			pt.Key, pt.File, pt.Size = p.Key, p.File, p.Size
		}
		mm.Part = []part{pt}
		out[i] = mm
	}
	return out
}

// played are the streams of all that play in sess, ids as strings: the
// video and the chosen audio and subtitle.
func played(all []stream, sess mediaserver.Session, transcoding bool) []stream {
	var out []stream
	st := sess.Streams
	pick := func(i int, decision string) {
		if i < 0 {
			return
		}
		x := all[i]
		x.ID = x.ID.quote()
		if transcoding {
			x.Decision = decision
		}
		out = append(out, x)
	}
	word := map[bool]string{false: "copy", true: "transcode"}
	pick(nth(all, 1, 0), word[st.Video == link.StreamTranscode || st.Subtitles == link.SubtitlesBurned])
	if n := sess.AudioTrack; n != nil {
		pick(nth(all, 2, *n), word[st.Audio == link.StreamTranscode])
	}
	if n := sess.SubtitleTrack; n != nil {
		sub := nth(all, 3, *n)
		if *n == -1 {
			sub = slices.IndexFunc(all, func(x stream) bool { return x.Key != "" })
		}
		if sub >= 0 && st.Subtitles == link.SubtitlesBurned {
			all[sub].Burn = "1"
		}
		pick(sub, map[bool]string{false: "copy", true: "burn"}[st.Subtitles == link.SubtitlesBurned])
	}
	return out
}

// nth is the index in all of the n-th embedded stream of typ, -1 when it
// has none.
func nth(all []stream, typ, n int) int {
	for i, x := range all {
		if x.StreamType != typ || x.Key != "" {
			continue
		}
		if n == 0 {
			return i
		}
		n--
	}
	return -1
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
		"sessionKey": e.Session.Key, "clientIdentifier": e.Session.Player.Key, "ratingKey": e.Session.Item.Key,
		"key": "/library/metadata/" + e.Session.Item.Key, "state": state, "viewOffset": 0,
	}
	if e.Session.Streams.Decision() != link.DecisionDirectPlay {
		entry["transcodeSession"] = transcodeKey(e.Session)
	}
	return pushed("playing", "PlaySessionStateNotification", entry), true
}

func pushed(typ, field string, entry map[string]any) notification {
	return notification{NotificationContainer: map[string]any{"type": typ, "size": 1, field: []map[string]any{entry}}}
}

// Play is a plex-play cue's arguments. An unreported client never sent
// its timeline, which only /transcode/sessions lists. Unselected lists
// the item's Media with none selected.
type Play struct {
	serverfake.Play
	Unselected bool `json:"unselected,omitempty"`
}

// Played is a plex-play cue's result: the session's key and the ratingKey
// of the item it plays.
type Played struct {
	Session string `json:"session"`
	Item    string `json:"item"`
}

// Select is a plex-select cue's arguments: whether the session's played
// Media is selected.
type Select struct {
	Session  string `json:"session"`
	Selected bool   `json:"selected"`
}

type Stop struct {
	Session string `json:"session"`
}

type Scan struct {
	Section string `json:"section"`
}

func (s *server) cues() error {
	if err := fake.OnCue(s.env, "plex-play", func(_ context.Context, a Play) (any, error) {
		p, err := a.Core(a.User, false)
		if err != nil {
			return nil, err
		}
		sess, err := s.core.Start(p)
		if err != nil {
			return nil, err
		}
		it, _ := s.core.ItemAt(sess.Version)
		s.mu.Lock()
		defer s.mu.Unlock()
		s.unselected[sess.Key] = a.Unselected
		if t := transcodeOf(sess, "/transcode/sessions/"+transcodeKey(sess)); t != nil {
			s.stale[sess.Player.Key] = t
		}
		return Played{Session: sess.Key, Item: it.Key}, nil
	}); err != nil {
		return err
	}
	if err := fake.OnCue(s.env, "plex-select", func(_ context.Context, a Select) (any, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.unselected[a.Session] = !a.Selected
		return map[string]bool{"selected": a.Selected}, nil
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

// productNames are Plex's names for the release words a conformance play
// receives.
var productNames = map[string]string{
	"MKV": "mkv", "MP4": "mp4", "TS": "mpegts", "x264": "h264", "x265": "hevc", "AAC": "aac", "DTS": "dca", "DD": "ac3", "DD+": "eac3",
}

// ReceivedOf is r, in release words and bits a second, in Plex's names
// and kbps.
func ReceivedOf(r link.Received) mediaserver.Received {
	name := func(s string) string { return cmp.Or(productNames[s], s) }
	return mediaserver.Received{
		Container: name(r.Container), VideoCodec: name(r.VideoCodec), AudioCodec: name(r.AudioCodec),
		Height: r.Height, Bitrate: r.Bitrate / 1000, Channels: r.Channels,
	}
}
