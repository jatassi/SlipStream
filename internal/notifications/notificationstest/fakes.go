package notificationstest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/mail"
	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/notifications/providers"
	"github.com/jatassi/slipstream/internal/notifying"
	"github.com/jatassi/slipstream/internal/role"
)

// Ping is the facts of the World's own kinds.
type Ping struct {
	Name   string      `json:"name"`
	Poster artwork.Ref `json:"poster,omitzero"`
}

var (
	PingRow = notice.Row[Ping]{
		Spec:   notice.Spec{Kind: "ping", Label: "Ping", Group: notice.System, OnByDefault: true, BellByDefault: true},
		Render: func(f Ping, _ notice.Context) notice.Message { return pingMessage("Ping", f) },
	}
	PongRow = notice.Row[Ping]{
		Spec:   notice.Spec{Kind: "pong", Label: "Pong", Group: notice.System, OnByDefault: true, BellByDefault: true},
		Render: func(f Ping, _ notice.Context) notice.Message { return pingMessage("Pong", f) },
	}
)

func pingMessage(label string, f Ping) notice.Message {
	return notice.Message{Title: label + " " + f.Name, Lines: []string{"It happened."}, Link: notice.Link{Path: "/pings/" + f.Name}, Poster: f.Poster}
}

// APoster is a poster reference the World's artwork answers.
var APoster = artwork.Ref{Source: "test", Shape: artwork.ShapePoster, Path: "/poster.jpg"}

const (
	HookID   providers.ID = "hook"
	ScriptID providers.ID = "script"
)

// hook is the World's provider: one POST to its url setting with its token
// as the bearer, and a reading of the reply that a real provider might
// make.
func hookProvider(words map[providers.ReasonCode]string) providers.Provider {
	return providers.Provider{
		ID: HookID, Label: "Hook", Image: providers.Bytes, Words: words,
		Fields: []providers.Field{
			{Key: "url", Label: "URL", Kind: providers.URL},
			{Key: "token", Label: "Token", Kind: providers.Text, Secret: true},
			{Key: "signing", Label: "Signing key", Kind: providers.Text, Secret: true, Optional: true},
		},
		Host:     hookHost,
		Build:    buildHook,
		Classify: classifyHook,
	}
}

func hookHost(values providers.Values, _ providers.Endpoints) (*url.URL, bool) {
	var raw string
	if err := json.Unmarshal(values["url"], &raw); err != nil {
		return nil, false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return nil, false
	}
	return &url.URL{Scheme: u.Scheme, Host: u.Host}, true
}

// body is what the Hook posts.
type body struct {
	Kind   notice.Kind `json:"kind"`
	Title  string      `json:"title"`
	Lines  []string    `json:"lines"`
	Link   string      `json:"link"`
	Tone   notice.Tone `json:"tone"`
	Poster string      `json:"poster,omitempty"`
}

func buildHook(p providers.Payload) (providers.Outbound, error) {
	var target, token string
	if err := json.Unmarshal(p.Settings["url"], &target); err != nil {
		return providers.Outbound{}, fmt.Errorf("read the url: %w", err)
	}
	if err := json.Unmarshal(p.Secrets["token"], &token); err != nil {
		return providers.Outbound{}, fmt.Errorf("read the token: %w", err)
	}
	b := body{Kind: p.Kind, Title: p.Message.Title, Lines: p.Message.Lines, Link: p.Message.Link.Path, Tone: p.Message.Tone}
	if p.Poster != nil {
		b.Poster = p.Poster.Name
	}
	raw, err := json.Marshal(b)
	if err != nil {
		return providers.Outbound{}, fmt.Errorf("encode the body: %w", err)
	}
	header := http.Header{"Authorization": {"Bearer " + token}}
	if key, ok := p.Secrets["signing"]; ok {
		var signing string
		if err := json.Unmarshal(key, &signing); err != nil {
			return providers.Outbound{}, fmt.Errorf("read the signing key: %w", err)
		}
		header.Set("X-Signing", signing)
	}
	return providers.Outbound{Post: &providers.Request{Method: http.MethodPost, URL: target, Header: header, Body: raw}}, nil
}

func classifyHook(r providers.Reply, now time.Time) providers.Verdict {
	if r.Down.Code != "" {
		return providers.Verdict{Outcome: providers.Transient, Reason: r.Down}
	}
	if r.Run != nil {
		if r.Run.Exit == 0 {
			return providers.Verdict{Outcome: providers.Sent}
		}
		return providers.Verdict{Outcome: providers.Transient, Reason: providers.Reason{Code: providers.ScriptExit, Exit: r.Run.Exit, Tail: string(r.Run.Tail)}}
	}
	res := r.Post
	status := providers.Reason{Status: res.Status}
	switch {
	case res.Status == 0:
		status.Code = providers.Network
		return providers.Verdict{Outcome: providers.Transient, Reason: status}
	case res.Status < 300:
		return providers.Verdict{Outcome: providers.Sent}
	case res.Status == http.StatusFailedDependency:
		return providers.Verdict{Outcome: providers.Sent, Response: res.Body}
	case res.Status == http.StatusTooManyRequests:
		seconds, _ := strconv.Atoi(res.Header.Get("Retry-After"))
		until := now.Add(time.Duration(seconds) * time.Second)
		return providers.Verdict{Outcome: providers.SlowDown, Until: until, Reason: providers.Reason{Code: providers.RateLimited, Status: res.Status, Until: until}}
	case res.Status == http.StatusNotFound:
		status.Code, status.Field = providers.Gone, "url"
		return providers.Verdict{Outcome: providers.RefusedConnection, Reason: status}
	case res.Status == http.StatusUnauthorized:
		status.Code, status.Field = providers.Unauthorized, "token"
		return providers.Verdict{Outcome: providers.RefusedConnection, Reason: status}
	case res.Status == http.StatusRequestEntityTooLarge:
		status.Code = providers.TooLarge
		return providers.Verdict{Outcome: providers.RefusedContent, Reason: status, WithoutImage: true, Response: res.Body}
	case res.Status < 500:
		status.Code = providers.Refused
		return providers.Verdict{Outcome: providers.RefusedContent, Reason: status, Response: res.Body}
	}
	status.Code = providers.ServerError
	return providers.Verdict{Outcome: providers.Transient, Reason: status}
}

// scriptProvider is Custom Script as far as the dispatcher needs: it runs
// its path on its Node.
func scriptProvider() providers.Provider {
	return providers.Provider{
		ID: ScriptID, Label: "Custom Script",
		Fields: []providers.Field{
			{Key: "path", Label: "Path", Kind: providers.Text},
			{Key: "node", Label: "Node", Kind: providers.NodeField},
		},
		Host: func(providers.Values, providers.Endpoints) (*url.URL, bool) { return nil, false },
		Build: func(p providers.Payload) (providers.Outbound, error) {
			var path string
			if err := json.Unmarshal(p.Settings["path"], &path); err != nil {
				return providers.Outbound{}, fmt.Errorf("read the path: %w", err)
			}
			file, err := json.Marshal(body{Kind: p.Kind, Title: p.Message.Title, Lines: p.Message.Lines})
			if err != nil {
				return providers.Outbound{}, fmt.Errorf("encode the event file: %w", err)
			}
			return providers.Outbound{Run: &providers.Script{Path: path, Event: string(p.Kind), EventFile: file, Stop: 30 * time.Second}}, nil
		},
		Classify: classifyHook,
	}
}

// Reply is one scripted answer.
type Reply struct {
	// Status is 200 when zero.
	Status int
	Header http.Header
	Body   string
	// Exit is a script's exit code.
	Exit int
	// Silent: the Node doesn't respond.
	Silent bool
	// Delay is how long the answer takes.
	Delay time.Duration
	// Hold keeps the call open after it is received until Hold closes or
	// the dispatcher stops.
	Hold chan struct{}
}

// Received is one message the Courier was handed.
type Received struct {
	At   time.Time
	Node role.NodeID
	// Target is the URL posted to, or the script's path.
	Target string
	Kind   notice.Kind
	Title  string
	Lines  []string
	Link   string
	Tone   notice.Tone
	Poster string
	Header http.Header
}

// Courier answers each target's queued replies in order, then 200 or exit
// 0.
type Courier struct {
	mu       sync.Mutex
	replies  map[string][]Reply
	received []Received
}

func newCourier() *Courier { return &Courier{replies: map[string][]Reply{}} }

// Queue adds replies to target's queue.
func (c *Courier) Queue(target string, rs ...Reply) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.replies[target] = append(c.replies[target], rs...)
}

// Received is every message handed to target, in order.
func (c *Courier) Received(target string) []Received {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []Received
	for _, r := range c.received {
		if r.Target == target {
			out = append(out, r)
		}
	}
	return out
}

// All is every message the Courier was handed.
func (c *Courier) All() []Received {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Received(nil), c.received...)
}

var errSilent = errors.New("notificationstest: the Node didn't respond")

func (c *Courier) take(ctx context.Context, rec Received, raw []byte) (Reply, error) {
	var b body
	_ = json.Unmarshal(raw, &b)
	rec.At, rec.Kind, rec.Title, rec.Lines, rec.Link, rec.Tone, rec.Poster = time.Now(), b.Kind, b.Title, b.Lines, b.Link, b.Tone, b.Poster
	c.mu.Lock()
	r := Reply{Status: http.StatusOK}
	if q := c.replies[rec.Target]; len(q) > 0 {
		r, c.replies[rec.Target] = q[0], q[1:]
	}
	c.received = append(c.received, rec)
	c.mu.Unlock()
	if r.Hold != nil {
		select {
		case <-r.Hold:
		case <-ctx.Done():
			return Reply{}, fmt.Errorf("notificationstest: the call ended: %w", context.Cause(ctx))
		}
	}
	if r.Delay > 0 {
		select {
		case <-time.After(r.Delay):
		case <-ctx.Done():
			return Reply{}, fmt.Errorf("notificationstest: the call ended: %w", context.Cause(ctx))
		}
	}
	if r.Silent {
		return Reply{}, errSilent
	}
	if r.Status == 0 {
		r.Status = http.StatusOK
	}
	return r, nil
}

func (c *Courier) Post(ctx context.Context, node role.NodeID, req notifying.Request) (notifying.Response, error) {
	r, err := c.take(ctx, Received{Node: node, Target: req.URL, Header: req.Header}, req.Body)
	if err != nil {
		return notifying.Response{}, err
	}
	return notifying.Response{Status: r.Status, Header: r.Header, Body: []byte(r.Body)}, nil
}

func (c *Courier) Run(ctx context.Context, node role.NodeID, s notifying.Script) (notifying.Ran, error) {
	r, err := c.take(ctx, Received{Node: node, Target: s.Path}, s.EventFile)
	if err != nil {
		return notifying.Ran{}, err
	}
	return notifying.Ran{Exit: r.Exit, Tail: []byte(r.Body)}, nil
}

func (c *Courier) Send(context.Context, role.NodeID, mail.Server, mail.Message) (mail.Outcome, error) {
	return mail.Outcome{}, errSilent
}

// Roster is the Nodes: each with the Notifying Role, online until a test
// says otherwise.
type Roster struct {
	mu      sync.Mutex
	self    role.NodeID
	names   map[role.NodeID]string
	online  map[role.NodeID]bool
	removed map[role.NodeID]bool
	changed chan struct{}
}

func newRoster() *Roster {
	return &Roster{names: map[role.NodeID]string{}, online: map[role.NodeID]bool{}, removed: map[role.NodeID]bool{}, changed: make(chan struct{})}
}

func (r *Roster) add(id role.NodeID, name string) {
	r.change(func() { r.names[id], r.online[id] = name, true })
}

// SetOnline takes node offline or brings it back.
func (r *Roster) SetOnline(node role.NodeID, on bool) { r.change(func() { r.online[node] = on }) }

// Remove writes node off.
func (r *Roster) Remove(node role.NodeID) {
	r.change(func() { r.online[node], r.removed[node] = false, true })
}

func (r *Roster) change(f func()) {
	r.mu.Lock()
	f()
	close(r.changed)
	r.changed = make(chan struct{})
	r.mu.Unlock()
}

func (r *Roster) Self() role.NodeID { return r.self }

func (r *Roster) Changed() <-chan struct{} {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.changed
}

func (r *Roster) Pick(_ role.Name, prefer role.NodeID) (role.NodeID, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.online[prefer] {
		return prefer, true
	}
	for id, on := range r.online {
		if on {
			return id, true
		}
	}
	return role.NodeID{}, false
}

func (r *Roster) InStep(node role.NodeID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.online[node]
}

func (r *Roster) Name(node role.NodeID) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.names[node]
}

func (r *Roster) Holds(_ role.Name, node role.NodeID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.names[node]
	return ok && !r.removed[node]
}

func (r *Roster) WrittenOff(_ role.Name, node role.NodeID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.removed[node]
}

type noMail struct{}

func (noMail) Server(context.Context) (mail.Server, bool, error) { return mail.Server{}, false, nil }

// posters answers APoster as a small JPEG.
type posters struct{}

func (posters) Image(_ context.Context, ref artwork.Ref, _ artwork.Size) (artwork.Image, error) {
	if ref != APoster {
		return artwork.Image{}, errors.New("notificationstest: no such poster")
	}
	return artwork.Image{Type: "image/jpeg", Body: []byte("a poster")}, nil
}

func (posters) PublicURL(ref artwork.Ref, _ artwork.Size) (string, error) {
	return "https://images.example" + ref.Path, nil
}
