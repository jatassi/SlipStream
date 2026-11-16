package notifying

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/httpretry"
)

// Request is one HTTP request the Controller built, which the Node sends
// as it is. Its URL and headers may carry a provider's secret.
type Request struct {
	Method  string
	URL     string
	Header  http.Header
	Body    []byte
	Timeout time.Duration
	Dial    DialRule
}

// LogValue is the method and host alone, since the path, query and
// headers may carry a token.
func (r Request) LogValue() slog.Value {
	host := ""
	if u, err := url.Parse(r.URL); err == nil {
		host = u.Host
	}
	return slog.GroupValue(slog.String("method", r.Method), slog.String("host", host))
}

// DialRule is which addresses a Post may reach.
type DialRule uint8

// AnyAddress lets a Post reach any address, the Admin's own LAN included.
const AnyAddress DialRule = iota

// Response is what the Node read: the status, the headers about waiting
// and at most the first MaxBody bytes of the body, or, with no status, why
// no response came.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
	Err    string
}

// MaxBody is how much of a response body comes back. A provider's error
// detail fits in it, and a page a misconfigured URL returns doesn't cross
// the link whole.
const MaxBody = 4 << 10

// waitHeader is whether a response header says how long to wait, the only
// headers a Post answers with.
func waitHeader(name string) bool {
	name = http.CanonicalHeaderKey(name)
	return name == "Retry-After" || strings.HasPrefix(name, "X-Ratelimit-") || strings.HasPrefix(name, "X-Limit-App-")
}

// postClient follows no redirect, since a redirected request would carry
// its secret to a host the Admin never named, and keeps no idle
// connection, since a Node posts rarely.
func postClient(dial Dial) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext:         dial,
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: 10 * time.Second,
			DisableKeepAlives:   true,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func post(ctx context.Context, client *http.Client, r Request) Response {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, r.Method, r.URL, bytes.NewReader(r.Body))
	if err != nil {
		return Response{Err: httpretry.WithoutURL(err).Error()}
	}
	req.Header = r.Header
	res, err := client.Do(req)
	if err != nil {
		return Response{Err: httpretry.WithoutURL(err).Error()}
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, MaxBody))
	if err != nil {
		return Response{Err: "read the response: " + httpretry.WithoutURL(err).Error()}
	}
	header := http.Header{}
	for name, values := range res.Header {
		if waitHeader(name) {
			header[name] = values
		}
	}
	return Response{Status: res.StatusCode, Header: header, Body: body}
}
