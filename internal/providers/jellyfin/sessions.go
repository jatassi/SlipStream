package jellyfin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// Session is one client session, with what it plays.
type Session struct {
	ID              string           `json:"Id"`
	UserID          string           `json:"UserId"`
	UserName        string           `json:"UserName"`
	Client          string           `json:"Client"`
	DeviceName      string           `json:"DeviceName"`
	DeviceID        string           `json:"DeviceId"`
	RemoteEndPoint  string           `json:"RemoteEndPoint"`
	NowPlayingItem  *Item            `json:"NowPlayingItem"`
	PlayState       *PlayState       `json:"PlayState"`
	TranscodingInfo *TranscodingInfo `json:"TranscodingInfo"`
}

// Play methods a PlayState reads.
const (
	DirectPlay   = "DirectPlay"
	DirectStream = "DirectStream"
	Transcode    = "Transcode"
)

type PlayState struct {
	PlayMethod    string `json:"PlayMethod"`
	PositionTicks int64  `json:"PositionTicks"`
	MediaSourceID string `json:"MediaSourceId"`
}

// TranscodingInfo is a transcode. The video is re-encoded when
// IsVideoDirect is false. Emby lists it only once the client reported its
// playback (#71). It goes stale on Jellyfin when a client never reports
// a stop.
type TranscodingInfo struct {
	IsVideoDirect            bool     `json:"IsVideoDirect"`
	IsAudioDirect            bool     `json:"IsAudioDirect"`
	VideoCodec               string   `json:"VideoCodec"`
	HardwareAccelerationType string   `json:"HardwareAccelerationType"`
	VideoDecoderIsHardware   bool     `json:"VideoDecoderIsHardware"`
	VideoEncoderIsHardware   bool     `json:"VideoEncoderIsHardware"`
	TranscodeReasons         []string `json:"TranscodeReasons"`
}

// Sessions are GET /Sessions. An API key sees every user's.
func (c *Client) Sessions(ctx context.Context) ([]Session, error) {
	var out []Session
	err := c.do(ctx, http.MethodGet, "/Sessions", nil, true, nil, &out)
	return out, err
}

// Playing is a player's playback report.
type Playing struct {
	ItemID        string `json:"ItemId"`
	MediaSourceID string `json:"MediaSourceId,omitempty"`
	PlaySessionID string `json:"PlaySessionId,omitempty"`
	PlayMethod    string `json:"PlayMethod"`
	PositionTicks int64  `json:"PositionTicks"`
}

// ReportPlaying is a player's POST /Sessions/Playing, which SlipStream
// never sends: a conformance case sends it as the player.
func (c *Client) ReportPlaying(ctx context.Context, p Playing) error {
	return c.do(ctx, http.MethodPost, "/Sessions/Playing", nil, true, p, nil)
}

// Message is one socket message.
type Message struct {
	MessageType string          `json:"MessageType"`
	Data        json.RawMessage `json:"Data,omitempty"`
}

// Socket message types.
const (
	MessageSessions       = "Sessions"
	MessageSessionsStart  = "SessionsStart"
	MessageForceKeepAlive = "ForceKeepAlive"
	MessageKeepAlive      = "KeepAlive"
	MessageLibraryChanged = "LibraryChanged"
)

// Sessions is a Sessions message's list.
func (m Message) Sessions() ([]Session, error) {
	var out []Session
	if err := json.Unmarshal(m.Data, &out); err != nil {
		return nil, fmt.Errorf("a %s message: %w", m.MessageType, err)
	}
	return out, nil
}

// LibraryChanged is a LibraryChanged message's data.
type LibraryChanged struct {
	ItemsAdded   []string `json:"ItemsAdded"`
	ItemsUpdated []string `json:"ItemsUpdated"`
	ItemsRemoved []string `json:"ItemsRemoved"`
}

func (c *Client) socketURL() (string, http.Header) {
	u := *c.base
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	h := http.Header{}
	if c.flavor == Emby {
		u = *u.JoinPath("/embywebsocket")
		u.RawQuery = url.Values{"api_key": {c.token}, "deviceId": {c.app.DeviceID}}.Encode()
		return u.String(), h
	}
	u = *u.JoinPath("/socket")
	h.Set("Authorization", c.authorization(c.token))
	return u.String(), h
}

// Socket reads the server's socket until ctx ends or the server closes
// it: Jellyfin's /socket signed with the Authorization header, Emby's
// /embywebsocket with ?api_key=. Once open it calls opened, asks for the
// session list every period with SessionsStart, and calls on for each
// message. It answers ForceKeepAlive with KeepAlive, and keeps sending one
// every half of the timeout the server gave, which drops a socket silent
// for that long.
func (c *Client) Socket(ctx context.Context, period time.Duration, opened func(), on func(Message)) error {
	address, header := c.socketURL()
	conn, resp, err := websocket.Dial(ctx, address, &websocket.DialOptions{HTTPClient: c.http, HTTPHeader: header})
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		if resp != nil && resp.StatusCode/100 != 1 {
			return fmt.Errorf("%s socket: %w", c.flavor, &Error{Status: resp.StatusCode, Detail: err.Error()})
		}
		return fmt.Errorf("%s socket: %w", c.flavor, err)
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(16 << 20)
	start, err := json.Marshal(fmt.Sprintf("0,%d", period.Milliseconds()))
	if err != nil {
		return fmt.Errorf("%s socket: %w", c.flavor, err)
	}
	if err := wsjson.Write(ctx, conn, Message{MessageType: MessageSessionsStart, Data: start}); err != nil {
		return fmt.Errorf("%s socket: %w", c.flavor, err)
	}
	opened()
	keepAlive := make(chan time.Duration, 1)
	kctx, stop := context.WithCancel(ctx)
	defer stop()
	go c.keepAlive(kctx, conn, keepAlive)
	for {
		var m Message
		if err := wsjson.Read(ctx, conn, &m); err != nil {
			if ctx.Err() != nil {
				return ctx.Err() //nolint:wrapcheck // the caller's own context
			}
			return fmt.Errorf("%s socket: %w", c.flavor, err)
		}
		if m.MessageType == MessageForceKeepAlive {
			if err := c.askedToKeepAlive(m, keepAlive); err != nil {
				return err
			}
		}
		on(m)
	}
}

// askedToKeepAlive hands keepAlive the timeout a ForceKeepAlive gives,
// in place of any it hasn't taken yet.
func (c *Client) askedToKeepAlive(m Message, keepAlive chan time.Duration) error {
	var seconds int
	if err := json.Unmarshal(m.Data, &seconds); err != nil {
		return fmt.Errorf("%s socket: a %s message: %w", c.flavor, m.MessageType, err)
	}
	select {
	case <-keepAlive:
	default:
	}
	keepAlive <- time.Duration(seconds) * time.Second
	return nil
}

// keepAlive sends KeepAlive at once for each timeout it is handed and
// every half of the last one after.
func (c *Client) keepAlive(ctx context.Context, conn *websocket.Conn, timeouts <-chan time.Duration) {
	var tick <-chan time.Time
	var ticker *time.Ticker
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case d := <-timeouts:
			if ticker != nil {
				ticker.Stop()
			}
			ticker = time.NewTicker(max(d/2, time.Second))
			tick = ticker.C
		case <-tick:
		}
		if err := wsjson.Write(ctx, conn, Message{MessageType: MessageKeepAlive}); err != nil {
			return
		}
	}
}
