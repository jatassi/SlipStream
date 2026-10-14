package world

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/events"
	"github.com/jatassi/slipstream/internal/identity"
)

// AdminIdentity is the manifest identity CaptureSSE signs in as: the run's
// Admin, the only identity a manifest holds.
const AdminIdentity = "admin"

// captures are the SSE streams the run reads until it stops, one per
// identity and Node.
type captures struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu      sync.Mutex
	running map[string]bool
}

func newCaptures(life context.Context) *captures {
	ctx, cancel := context.WithCancel(life)
	return &captures{ctx: ctx, cancel: cancel, running: map[string]bool{}}
}

// sseMessage is an sse.message record's data: one message of the stream
// the identity As opened on Node.
type sseMessage struct {
	As    string `json:"as"`
	Node  string `json:"node"`
	ID    string `json:"id,omitempty"`
	Event string `json:"event"`
	Data  string `json:"data"`
}

const sessionEnded = "live.session-ended"

// CaptureSSE signs in to the Node node once as the manifest identity as,
// opens its event stream and appends each message as an sse.message
// record until the run stops. A stream that ends opens again, with the
// last id it read, on the Node's next node.ready; live.session-ended ends
// the capture.
func (r *Run) CaptureSSE(ctx context.Context, as, node string) (events.Record, error) {
	m := r.Manifest()
	if as != AdminIdentity || m.Admin == nil {
		return events.Record{}, fmt.Errorf("the manifest holds no identity %s", as)
	}
	i := slices.IndexFunc(m.Nodes, func(n Node) bool { return n.Name == node })
	if i < 0 {
		return events.Record{}, fmt.Errorf("the run has no node %s", node)
	}
	return r.captureSSE(ctx, *m.Admin, m.Nodes[i])
}

// captureSSE returns once the stream's first message is in the event log,
// so a capture started at launch holds the Node's opening live.resync
// before run.ready.
func (r *Run) captureSSE(ctx context.Context, admin Admin, n Node) (events.Record, error) {
	rec, first, err := r.openCapture(ctx, admin, n)
	if err != nil {
		return events.Record{}, err
	}
	select {
	case <-first:
		return rec, nil
	case <-ctx.Done():
		return events.Record{}, fmt.Errorf("capture %s's stream on %s: %w", AdminIdentity, n.Name, ctx.Err())
	}
}

func (r *Run) openCapture(ctx context.Context, admin Admin, n Node) (events.Record, <-chan struct{}, error) {
	key := AdminIdentity + "@" + n.Name
	r.captures.mu.Lock()
	defer r.captures.mu.Unlock()
	if r.captures.ctx.Err() != nil {
		return events.Record{}, nil, errors.New("the run is stopping")
	}
	if r.captures.running[key] {
		return events.Record{}, nil, fmt.Errorf("the run already captures %s's stream on %s", AdminIdentity, n.Name)
	}
	s := &sseStream{run: r, as: AdminIdentity, node: n.Name, url: n.URL, first: make(chan struct{})}
	if err := s.signIn(ctx, admin); err != nil {
		return events.Record{}, nil, err
	}
	body, err := s.open()
	if err != nil {
		return events.Record{}, nil, err
	}
	rec, err := r.events.Append(events.Record{Kind: events.Cue, Node: events.World, Data: stepData(ctx, cued{Name: "sse", Args: data(sseCapture{AdminIdentity, n.Name})})})
	if err != nil {
		_ = body.Close()
		return events.Record{}, nil, err
	}
	r.captures.running[key] = true
	r.captures.wg.Go(func() { s.read(body, rec.Seq) })
	return rec, s.first, nil
}

// stopCaptures ends every capture and returns once none can append.
func (r *Run) stopCaptures() {
	r.captures.mu.Lock()
	r.captures.cancel()
	r.captures.mu.Unlock()
	r.captures.wg.Wait()
}

type sseStream struct {
	run            *Run
	as, node, url  string
	cookie, lastID string
	// first closes once the stream's first message is appended, or when
	// the capture ends before one arrives.
	first     chan struct{}
	firstOnce sync.Once
}

func (s *sseStream) signIn(ctx context.Context, admin Admin) error {
	body, err := json.Marshal(struct {
		Email string `json:"email"`
		PIN   string `json:"pin"`
	}{admin.Email, admin.PIN})
	if err != nil {
		return fmt.Errorf("sign in to %s: %w", s.node, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url+"/api/v1/session", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sign in to %s: %w", s.node, err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sign in to %s: %w", s.node, err)
	}
	defer func() { _ = res.Body.Close() }()
	reply, _ := io.ReadAll(res.Body)
	for _, c := range res.Cookies() {
		if c.Name == identity.CookieName && res.StatusCode == http.StatusOK {
			s.cookie = c.Value
			return nil
		}
	}
	return fmt.Errorf("sign in to %s as %s: %d %s", s.node, admin.Email, res.StatusCode, bytes.TrimSpace(reply))
}

// open opens the stream under the run's captures, whose end closes it.
func (s *sseStream) open() (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(s.run.captures.ctx, http.MethodGet, s.url+"/api/v1/events", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("open %s's stream: %w", s.node, err)
	}
	req.Header.Set("Cookie", identity.CookieName+"="+s.cookie)
	if s.lastID != "" {
		req.Header.Set("Last-Event-ID", s.lastID)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open %s's stream: %w", s.node, err)
	}
	if res.StatusCode != http.StatusOK {
		reply, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		return nil, fmt.Errorf("open %s's stream: %d %s", s.node, res.StatusCode, bytes.TrimSpace(reply))
	}
	return res.Body, nil
}

// read appends body's messages, then reopens the stream after each of the
// Node's readiness records past since, until the session ends or the run
// stops.
func (s *sseStream) read(body io.ReadCloser, since uint64) {
	ctx := s.run.captures.ctx
	defer s.firstOnce.Do(func() { close(s.first) })
	for {
		ended := s.drain(body)
		_ = body.Close()
		if ended || ctx.Err() != nil {
			return
		}
		for {
			ready, err := events.Wait(ctx, s.run.runDir, events.Query{Kind: events.NodeReady, Node: s.node, After: since})
			if err != nil {
				return
			}
			since = ready.Seq
			if body, err = s.open(); err == nil {
				break
			}
			s.run.log.Error("sse", "as", s.as, "node", s.node, "err", err.Error())
		}
	}
}

// drain appends each message body holds and reports whether the session
// ended.
func (s *sseStream) drain(body io.Reader) bool {
	r := bufio.NewReader(body)
	for {
		msg, err := readMessage(r)
		if err != nil {
			return false
		}
		if msg.Event == "" {
			continue
		}
		msg.As, msg.Node = s.as, s.node
		if msg.ID != "" {
			s.lastID = msg.ID
		}
		s.run.appendRecord(events.Record{Kind: events.SSEMessage, Node: s.node, Data: data(msg)})
		s.firstOnce.Do(func() { close(s.first) })
		if msg.Event == sessionEnded {
			return true
		}
	}
}

// readMessage reads one message: its fields up to the blank line that ends
// it. A comment, such as a keepalive, is a message with no event.
func readMessage(r *bufio.Reader) (sseMessage, error) {
	var msg sseMessage
	var data []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return sseMessage{}, err //nolint:wrapcheck // any read error ends the stream, which drain reopens
		}
		line = strings.TrimSuffix(line, "\n")
		if line == "" {
			msg.Data = strings.Join(data, "\n")
			return msg, nil
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "id":
			msg.ID = value
		case "event":
			msg.Event = value
		case "data":
			data = append(data, value)
		}
	}
}
