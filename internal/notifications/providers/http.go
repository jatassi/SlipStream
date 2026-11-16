package providers

import (
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/httpretry"
)

// minWait is the shortest slow-down, so a provider that asks for no wait,
// or a wait SlipStream can't read, is never retried at once.
const minWait = time.Second

// reading is how a Post provider reads its responses: its own reading
// first, then the shared status table.
type reading struct {
	// image: the provider takes a poster, so a 413 may pass without it.
	image bool
	// auth and gone are the fields a 401 or 403, and a 404 or 410, name.
	auth, gone string
	// own reads what the provider documents beyond the status table, and
	// reports false to leave the rest to it.
	own func(r Response, now time.Time) (Verdict, bool)
}

func (rd reading) classify(r Reply, now time.Time) Verdict {
	if r.Post == nil {
		return Verdict{Outcome: Transient, Reason: r.Down}
	}
	if rd.own != nil {
		if v, ok := rd.own(*r.Post, now); ok {
			return v
		}
	}
	return rd.status(*r.Post, now)
}

// status is the shared table: 2xx sent, 429 or a 503 with Retry-After a
// slow-down, 401 and 403 a refused connection, 404 and 410 gone, 400, 413
// and 422 a refused message, and anything else a failure a retry may fix.
func (rd reading) status(r Response, now time.Time) Verdict {
	switch s := r.Status; {
	case s == 0:
		return Verdict{Outcome: Transient, Reason: Reason{Code: Network}}
	case s >= 200 && s < 300:
		return Verdict{Outcome: Sent}
	case s == http.StatusTooManyRequests, s == http.StatusServiceUnavailable && r.Header.Get("Retry-After") != "":
		return slowDown(now, httpretry.After(r.Header.Get("Retry-After"), now), s)
	case s == http.StatusUnauthorized || s == http.StatusForbidden:
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: Unauthorized, Status: s, Field: rd.auth}}
	case s == http.StatusNotFound || s == http.StatusGone:
		return Verdict{Outcome: RefusedConnection, Reason: Reason{Code: Gone, Status: s, Field: rd.gone}}
	case s == http.StatusRequestEntityTooLarge:
		return Verdict{Outcome: RefusedContent, Reason: Reason{Code: TooLarge, Status: s}, WithoutImage: rd.image, Response: r.Body}
	case s == http.StatusBadRequest || s == http.StatusUnprocessableEntity:
		return Verdict{Outcome: RefusedContent, Reason: Reason{Code: Refused, Status: s}, Response: r.Body}
	}
	return Verdict{Outcome: Transient, Reason: Reason{Code: ServerError, Status: r.Status}}
}

// slowDown waits wait from now, and at least minWait.
func slowDown(now time.Time, wait time.Duration, status int) Verdict {
	until := now.Add(max(wait, minWait))
	return Verdict{Outcome: SlowDown, Until: until, Reason: Reason{Code: RateLimited, Status: status, Until: until}}
}

// epoch is the time a header of Unix seconds names, zero when it names
// none.
func epoch(v string) time.Time {
	secs, err := strconv.ParseInt(v, 10, 64)
	if err != nil || secs <= 0 {
		return time.Time{}
	}
	return time.Unix(secs, 0)
}

func basicAuth(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

func jsonHeader() http.Header { return http.Header{"Content-Type": {"application/json"}} }
