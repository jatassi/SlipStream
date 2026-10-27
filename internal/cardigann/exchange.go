package cardigann

import (
	"errors"
	"net/http"
	"net/url"
)

// Exchange is one flow of requests to a site, run without IO. Next is the
// request to send, false once the flow has ended. Feed takes that request's
// response. An error from Feed ends the flow with that error.
type Exchange interface {
	Next() (Request, bool)
	Feed(Response) error
}

// Request is one request an Exchange asks the caller to send.
type Request struct {
	Method string
	URL    *url.URL
	Header http.Header
	Body   []byte
	// FollowRedirects is whether the caller follows redirects before it
	// feeds the response back. A redirect to a magnet link isn't followed:
	// the caller feeds back the redirect response itself.
	FollowRedirects bool
	// ReplaceCookies is whether the response's cookies replace the site's
	// jar, as a Prowlarr login does.
	ReplaceCookies bool
	Purpose        Purpose
}

// Purpose is what a request is for.
type Purpose int

// The purposes.
const (
	ForLogin Purpose = iota + 1
	ForSearch
	ForDownload
	ForCaptcha
)

// Response is a request's response as the caller received it, its body
// undecoded.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
	// URL is the address the response came from, after any redirect the
	// caller followed.
	URL *url.URL
}

// step is one request of a flow and what the flow does with its response:
// the next step, or nil when the flow ends.
type step struct {
	req  Request
	then func(Response) (*step, error)
}

// flow runs a chain of steps as an Exchange. Login, Search and Download
// embed one.
type flow struct {
	cur *step
	err error
}

var (
	errEnded      = errors.New("the exchange has already ended")
	errUnfinished = errors.New("the exchange hasn't ended")
)

func startFlow(first *step, err error) flow {
	if err != nil {
		return flow{err: err}
	}
	return flow{cur: first}
}

// Next is the request to send, false once the flow has ended.
func (f *flow) Next() (Request, bool) {
	if f.cur == nil {
		return Request{}, false
	}
	return f.cur.req, true
}

// Feed takes the response to the request Next gave.
func (f *flow) Feed(r Response) error {
	if f.cur == nil {
		return errEnded
	}
	next, err := f.cur.then(r)
	f.cur = next
	if err != nil {
		f.cur, f.err = nil, err
	}
	return err
}

// ended is the error the flow ended with, or errUnfinished while a request
// is still to be sent.
func (f *flow) ended() error {
	if f.cur != nil {
		return errUnfinished
	}
	return f.err
}
