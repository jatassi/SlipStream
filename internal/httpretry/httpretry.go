// Package httpretry reads what an HTTP answer says about trying again, and
// keeps a request's URL, which may carry a key, out of its errors.
package httpretry

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// After is the wait a Retry-After value of seconds or an HTTP date asks
// for, zero when there is none.
func After(v string, now time.Time) time.Duration {
	if secs, err := strconv.Atoi(v); err == nil {
		return max(time.Duration(secs)*time.Second, 0)
	}
	if at, err := http.ParseTime(v); err == nil {
		return max(at.Sub(now), 0)
	}
	return 0
}

// WithoutURL drops the request URL from err, since it may carry a key.
func WithoutURL(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return ue.Err
	}
	return err
}
