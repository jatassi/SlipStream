package fault

import (
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"
)

// malformed parses as none of JSON, XML or bencode.
const malformed = "\x00malformed"

// HTTP serves h, the handler of route at target, behind p's HTTP rules.
// origin reads a request's dialing Node. A status or a hang answers
// without calling h, so a faulted request never reaches the fake's state;
// the body faults take h's answer and spoil it.
func (p *Plan) HTTP(target Target, route string, origin func(*http.Request) string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, ok := p.request(Front{Target: target, Origin: origin(r)}, route, r)
		if !ok {
			h.ServeHTTP(w, r)
			return
		}
		switch f := f.(type) {
		case Status:
			if f.RetryAfter > 0 {
				seconds := strconv.Itoa(int(math.Ceil(f.RetryAfter.Seconds())))
				w.Header().Set("Retry-After", seconds)
				if f.ResetHeader != "" {
					w.Header().Set(f.ResetHeader, seconds)
				}
			}
			http.Error(w, http.StatusText(f.Code), f.Code)
		case Hang:
			<-r.Context().Done()
			panic(http.ErrAbortHandler)
		case Malformed:
			code, _ := head(w, h, r, len(malformed))
			w.WriteHeader(code)
			_, _ = w.Write([]byte(malformed))
		case Truncate:
			code, body := head(w, h, r, -1)
			w.WriteHeader(code)
			_, _ = w.Write(body[:len(body)/2])
			_ = http.NewResponseController(w).Flush()
			panic(http.ErrAbortHandler)
		case SlowBody:
			code, body := head(w, h, r, -1)
			w.WriteHeader(code)
			slowly(w, r, body, Rate(f))
		default:
			panic(fmt.Sprintf("fault: %s is not an HTTP fault", f))
		}
	})
}

// head runs h, copies its headers to w, and declares length, or the real
// body's length when length is -1. It returns h's status and body.
func head(w http.ResponseWriter, h http.Handler, r *http.Request, length int) (code int, body []byte) {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	body = rec.Body.Bytes()
	if length < 0 {
		length = len(body)
	}
	maps.Copy(w.Header(), rec.Header())
	w.Header().Set("Content-Length", strconv.Itoa(length))
	return rec.Code, body
}

// slowly sends the headers at once and body at rate.
func slowly(w http.ResponseWriter, r *http.Request, body []byte, rate Rate) {
	rc := http.NewResponseController(w)
	if rc.Flush() != nil {
		return
	}
	for len(body) > 0 {
		n := min(step(rate), len(body))
		t := time.NewTimer(pace(n, rate))
		select {
		case <-t.C:
		case <-r.Context().Done():
			t.Stop()
			return
		}
		if _, err := w.Write(body[:n]); err != nil || rc.Flush() != nil {
			return
		}
		body = body[n:]
	}
}
