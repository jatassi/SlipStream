// Package notifysink is fake-notify-sink: one front that stands in for
// every notification provider SlipStream posts to. Each provider is a
// profile with its own routes, its success answer and the answers its
// cues play: a slow-down, a refusal, a gone webhook, an outage and a slow
// answer. Every request is kept as a notify capture, with the request as
// outbound.Dump writes it, so a capture compares with the provider's
// golden file, and with each poster it carried as a file.
//
// The fixed hosts sit under a path prefix per profile, which
// Endpoints.Notify points at, and so do the servers an Admin types for
// Gotify, ntfy, Apprise and the Webhook: https://<front>/gotify and so on.
// Discord and Slack webhook URLs keep their own paths at the front's root.
// A capture's dump drops the prefix, so it reads as the provider's own
// host would have received it.
package notifysink

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/fake"
	"github.com/jatassi/slipstream/fakes/scenario"
	"github.com/jatassi/slipstream/internal/node"
	"github.com/jatassi/slipstream/internal/node/endpoint"
	"github.com/jatassi/slipstream/internal/notifications/outbound"
	"github.com/jatassi/slipstream/internal/notifications/providers"
)

// Notify is the capture of every request the sink answered, in
// notify.request records whose node is the sending origin.
var Notify = fake.CaptureSpec{Name: "notify", Kind: "notify.request"}

// Fake is the sink. It reads nothing from the Scenario.
var Fake = fake.Spec{
	Name:      "notify-sink",
	TLS:       fake.FrontTLS,
	New:       newSink,
	Endpoints: endpoints,
	Online:    fake.NoOnlineRun("a real Discord, Telegram or Pushover account is personal, so the sink stands in for every provider"),
	Captures:  []fake.CaptureSpec{Notify},
	Cues:      cueSpecs(),
}

// Request is one request as the sink kept it.
type Request struct {
	Profile string `json:"profile"`
	// Method and Path are the dump's first line: the path with its
	// secrets as [set].
	Method string `json:"method"`
	Path   string `json:"path"`
	// Status is what the sink answered.
	Status int `json:"status"`
	// Dump is outbound.Dump of the request, comparable with the provider's
	// golden file.
	Dump string `json:"dump"`
}

// Prefix is the path the sink serves a profile's host under, empty for a
// profile served at the front's root.
func Prefix(id providers.ID) string {
	for _, p := range profiles {
		if p.id == id {
			return p.prefix
		}
	}
	panic("notifysink: no profile " + string(id))
}

func endpoints(e *node.EndpointsBlock, at fake.Address) {
	for _, p := range []struct {
		to *endpoint.URL
		id providers.ID
	}{{&e.Notify.Telegram, "telegram"}, {&e.Notify.Pushover, "pushover"}, {&e.Notify.Pushbullet, "pushbullet"}, {&e.Notify.Ntfy, "ntfy"}} {
		*p.to = endpoint.MustParseURL(at.URL(Prefix(p.id)))
	}
}

// Cue names, one of each per profile, each on until {"off": true}.
const (
	// RateLimited answers the profile's slow-down, waiting {"seconds": 2}
	// by default.
	RateLimited = "rate-limited"
	// Refused answers the profile's refusal of the connection: 401, or
	// Telegram's unknown chat and Pushover's unknown user.
	Refused = "refused"
	// Gone answers 404, or 410 for Slack's archived channel.
	Gone = "gone"
	// Down answers 503 with no Retry-After.
	Down = "down"
	// Slow holds each answer for {"delay": "40s"}, its default.
	Slow = "slow"
)

var cues = []string{RateLimited, Refused, Gone, Down, Slow}

// Cue is the name of the cue that plays what on the profile id.
func Cue(id providers.ID, what string) string { return string(id) + "-" + what }

func cueSpecs() []fake.CueSpec {
	var out []fake.CueSpec
	for _, p := range profiles {
		for _, c := range cues {
			out = append(out, fake.CueSpec{Name: Cue(p.id, c)})
		}
	}
	return out
}

const (
	defaultWait = 2 * time.Second
	defaultSlow = 40 * time.Second
	// maxBody bounds what the sink reads of one request, well over any
	// provider's limit on a poster.
	maxBody = 32 << 20
)

type LimitArgs struct {
	Off     bool `json:"off"`
	Seconds int  `json:"seconds"`
}

type SlowArgs struct {
	Off   bool   `json:"off"`
	Delay string `json:"delay"`
}

type Toggle struct {
	Off bool `json:"off"`
}

type was struct {
	Was bool `json:"was"`
}

type sink struct {
	notify *fake.Capture

	mu    sync.Mutex
	on    map[string]bool
	wait  map[providers.ID]time.Duration
	delay map[providers.ID]time.Duration
}

func newSink(_ scenario.Scenario, env fake.Env) (fake.Server, error) {
	notify, err := env.Capture(Notify.Name)
	if err != nil {
		return nil, err
	}
	s := &sink{notify: notify, on: map[string]bool{}, wait: map[providers.ID]time.Duration{}, delay: map[providers.ID]time.Duration{}}
	routes := fake.Routes{}
	for _, p := range profiles {
		if err := s.cues(env, p.id); err != nil {
			return nil, err
		}
		for i, pattern := range p.patterns {
			routes[fmt.Sprintf("%s-%d", p.id, i)] = fake.Route{Pattern: pattern, Handler: s.serve(p)}
		}
	}
	return fake.HTTP(env, routes), nil
}

func (s *sink) cues(env fake.Env, id providers.ID) error {
	if err := fake.OnCue(env, Cue(id, RateLimited), func(_ context.Context, a LimitArgs) (any, error) {
		wait := defaultWait
		if a.Seconds > 0 {
			wait = time.Duration(a.Seconds) * time.Second
		}
		s.mu.Lock()
		s.wait[id] = wait
		s.mu.Unlock()
		return s.set(Cue(id, RateLimited), !a.Off), nil
	}); err != nil {
		return err
	}
	for _, c := range []string{Refused, Gone, Down} {
		if err := fake.OnCue(env, Cue(id, c), func(_ context.Context, a Toggle) (any, error) { return s.set(Cue(id, c), !a.Off), nil }); err != nil {
			return err
		}
	}
	return fake.OnCue(env, Cue(id, Slow), func(_ context.Context, a SlowArgs) (any, error) {
		delay := defaultSlow
		if a.Delay != "" {
			d, err := time.ParseDuration(a.Delay)
			if err != nil || d <= 0 {
				return nil, fmt.Errorf("delay %q is not a positive duration", a.Delay)
			}
			delay = d
		}
		s.mu.Lock()
		s.delay[id] = delay
		s.mu.Unlock()
		return s.set(Cue(id, Slow), !a.Off), nil
	})
}

func (s *sink) set(cue string, on bool) was {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := s.on[cue]
	s.on[cue] = on
	return was{before}
}

// play is the answer the profile's cues pick now, and how long to hold it.
func (s *sink) play(id providers.ID) (cue string, wait, hold time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.on[Cue(id, Slow)] {
		hold = s.delay[id]
	}
	for _, c := range []string{Down, Gone, Refused, RateLimited} {
		if s.on[Cue(id, c)] {
			return c, s.wait[id], hold
		}
	}
	return "", 0, hold
}

func (s *sink) serve(p profile) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
		if err != nil {
			http.Error(w, "The request body didn't arrive whole.", http.StatusBadRequest)
			return
		}
		cue, wait, hold := s.play(p.id)
		if hold > 0 {
			select {
			case <-time.After(hold):
			case <-r.Context().Done():
				return
			}
		}
		rec := &recorder{ResponseWriter: w}
		switch cue {
		case Down:
			http.Error(rec, "Service Unavailable", http.StatusServiceUnavailable)
		case Gone:
			p.gone(rec)
		case Refused:
			p.refused(rec)
		case RateLimited:
			p.limited(rec, wait)
		default:
			p.ok(rec, r)
		}
		s.keep(r, p, body, rec.status)
	}
}

func (s *sink) keep(r *http.Request, p profile, body []byte, status int) {
	sent := asSent(r, p, body)
	dump := outbound.Dump(sent, p.secrets(r, sent))
	line, _, _ := strings.Cut(dump, "\n")
	method, path, _ := strings.Cut(line, " ")
	if status == 0 {
		status = http.StatusOK
	}
	s.notify.Record(r.Context(), Request{Profile: string(p.id), Method: method, Path: path, Status: status, Dump: dump}, posters(sent)...)
}

// transport are the headers Go's client adds itself, which no provider
// builds.
var transport = []string{"Accept-Encoding", "Content-Length", "User-Agent"}

// asSent is r as the provider built it: its path without the profile's
// prefix, and its headers without those the transport added.
func asSent(r *http.Request, p profile, body []byte) providers.Request {
	header := r.Header.Clone()
	for _, name := range transport {
		header.Del(name)
	}
	u := url.URL{Scheme: "https", Host: "sink", Path: strings.TrimPrefix(r.URL.Path, p.prefix), RawQuery: r.URL.RawQuery}
	return providers.Request{Method: r.Method, URL: u.String(), Header: header, Body: body}
}

// posters are the files a request carried: each multipart file part, or a
// body that is itself an image.
func posters(r providers.Request) []fake.File {
	kind, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch {
	case strings.HasPrefix(kind, "image/"):
		name := r.Header.Get("Filename")
		if name == "" {
			name = "body"
		}
		return []fake.File{{Name: name, Data: r.Body}}
	case kind != "multipart/form-data":
		return nil
	}
	var out []fake.File
	mr := multipart.NewReader(bytes.NewReader(r.Body), params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			return out
		}
		if part.FileName() != "" {
			data, _ := io.ReadAll(part)
			out = append(out, fake.File{Name: part.FileName(), Data: data})
		}
	}
}

// fields are r's named text values, from a form, a multipart body or a
// JSON object.
func fields(r providers.Request) map[string]string {
	out := map[string]string{}
	kind, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch kind {
	case "application/x-www-form-urlencoded":
		form, _ := url.ParseQuery(string(r.Body))
		for k := range form {
			out[k] = form.Get(k)
		}
	case "multipart/form-data":
		mr := multipart.NewReader(bytes.NewReader(r.Body), params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			if part.FileName() == "" {
				data, _ := io.ReadAll(part)
				out[part.FormName()] = string(data)
			}
		}
	case "application/json":
		var obj map[string]any
		_ = json.Unmarshal(r.Body, &obj)
		for k, v := range obj {
			if s, ok := v.(string); ok {
				out[k] = s
			}
		}
	}
	return out
}

type recorder struct {
	http.ResponseWriter
	status int
}

func (r *recorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *recorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b) //nolint:wrapcheck // the response's own write
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson // a fake's answer to a client that went away
}

func writeText(w http.ResponseWriter, status int, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, text)
}

func retryAfter(w http.ResponseWriter, wait time.Duration) {
	w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())))
}
