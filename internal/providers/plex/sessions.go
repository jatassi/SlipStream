package plex

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type User struct {
	ID    ID     `json:"id"`
	Title string `json:"title"`
}

type Player struct {
	Title             string `json:"title"`
	Product           string `json:"product"`
	MachineIdentifier string `json:"machineIdentifier"`
	State             string `json:"state"`
}

type SessionInfo struct {
	ID string `json:"id"`
}

// Transcode is a transcode session. VideoDecision is "transcode" when the
// video is re-encoded and "copy" for a remux. Direct play has none.
type Transcode struct {
	Key                     string `json:"key"`
	VideoDecision           string `json:"videoDecision"`
	AudioDecision           string `json:"audioDecision"`
	TranscodeHwRequested    bool   `json:"transcodeHwRequested"`
	TranscodeHwDecoding     string `json:"transcodeHwDecoding"`
	TranscodeHwEncoding     string `json:"transcodeHwEncoding"`
	TranscodeHwFullPipeline bool   `json:"transcodeHwFullPipeline"`
}

// Session is one playback with its transcode, if any.
type Session struct {
	Metadata
	// Transcode is the session's entry in /transcode/sessions, nil for
	// direct play.
	Transcode *Transcode
}

// Sessions are /status/sessions joined with /transcode/sessions. A session
// is listed in /status/sessions only once its client reported a timeline,
// and /transcode/sessions always lists a transcode (#71).
func (c *Client) Sessions(ctx context.Context) ([]Session, error) {
	sessions, _, err := c.Playing(ctx)
	return sessions, err
}

// Playing is Sessions and every transcode, which lists those of sessions
// /status/sessions doesn't list yet.
func (c *Client) Playing(ctx context.Context) ([]Session, []Transcode, error) {
	var status struct {
		MediaContainer struct {
			Metadata []Metadata `json:"Metadata"`
		}
	}
	if err := c.do(ctx, http.MethodGet, "/status/sessions", nil, true, &status); err != nil {
		return nil, nil, err
	}
	transcodes, err := c.Transcodes(ctx)
	if err != nil {
		return nil, nil, err
	}
	byKey := map[string]*Transcode{}
	for i := range transcodes {
		byKey[transcodes[i].Key] = &transcodes[i]
	}
	out := make([]Session, len(status.MediaContainer.Metadata))
	for i, m := range status.MediaContainer.Metadata {
		out[i] = Session{Metadata: m, Transcode: m.TranscodeSession}
		if m.TranscodeSession != nil {
			if t, ok := byKey[transcodeKey(m.TranscodeSession.Key)]; ok {
				out[i].Transcode = t
			}
		}
	}
	return out, transcodes, nil
}

// transcodeKey is a session's transcode key, which /status/sessions gives
// as /transcode/sessions/<key>.
func transcodeKey(k string) string { return k[strings.LastIndexByte(k, '/')+1:] }

// Transcodes are /transcode/sessions.
func (c *Client) Transcodes(ctx context.Context) ([]Transcode, error) {
	var out struct {
		MediaContainer struct {
			TranscodeSession []Transcode `json:"TranscodeSession"`
		}
	}
	err := c.do(ctx, http.MethodGet, "/transcode/sessions", nil, true, &out)
	return out.MediaContainer.TranscodeSession, err
}

// Notification is one message on the notification socket.
type Notification struct {
	Type     string          `json:"type"`
	Timeline []TimelineEntry `json:"TimelineEntry"`
	Playing  []PlaySession   `json:"PlaySessionStateNotification"`
	Activity []ActivityEntry `json:"ActivityNotification"`
}

// Timeline states Plex sends for library items.
const (
	TimelineCreated = 0
	TimelineDone    = 5
	TimelineDeleted = 9
)

type TimelineEntry struct {
	SectionID     ID     `json:"sectionID"`
	ItemID        ID     `json:"itemID"`
	Type          int    `json:"type"`
	Title         string `json:"title"`
	State         int    `json:"state"`
	MetadataState string `json:"metadataState"`
}

type PlaySession struct {
	SessionKey       ID     `json:"sessionKey"`
	ClientIdentifier string `json:"clientIdentifier"`
	RatingKey        ID     `json:"ratingKey"`
	Key              string `json:"key"`
	State            string `json:"state"`
	ViewOffset       int64  `json:"viewOffset"`
	TranscodeSession string `json:"transcodeSession"`
}

type ActivityEntry struct {
	Event    string   `json:"event"`
	UUID     string   `json:"uuid"`
	Activity Activity `json:"Activity"`
}

type Activity struct {
	UUID     string `json:"uuid"`
	Type     string `json:"type"`
	Title    string `json:"title"`
	Progress int    `json:"progress"`
	Context  struct {
		LibrarySectionID ID `json:"librarySectionID"`
	} `json:"Context"`
}

// Notifications reads /:/websockets/notifications until ctx ends or the
// server closes it, calling opened once the socket is open and on for each
// message. It needs no subscription message.
func (c *Client) Notifications(ctx context.Context, opened func(), on func(Notification)) error {
	u := c.base.JoinPath("/:/websockets/notifications")
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	u.RawQuery = url.Values{"X-Plex-Token": {c.token}}.Encode()
	conn, resp, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{HTTPClient: c.http, HTTPHeader: c.h.header()})
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil && resp.StatusCode/100 != 1 {
			return fmt.Errorf("plex notifications: %w", &Error{Status: resp.StatusCode, Detail: err.Error()})
		}
		return fmt.Errorf("plex notifications: %w", err)
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(16 << 20)
	opened()
	for {
		var msg struct {
			NotificationContainer Notification `json:"NotificationContainer"`
		}
		if err := wsjson.Read(ctx, conn, &msg); err != nil {
			if ctx.Err() != nil {
				return ctx.Err() //nolint:wrapcheck // the caller's own context
			}
			return fmt.Errorf("plex notifications: %w", err)
		}
		on(msg.NotificationContainer)
	}
}
