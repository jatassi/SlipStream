package clients

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Response is a client's response, its body read and closed.
type Response struct {
	Status int
	Header http.Header
	Body   []byte
}

// Do sends req with s's HTTP client under s.Call and reads at most limit
// bytes of the body. It logs method, host, status and how long the
// request took, never a path, query, header or body. A dial error, a
// timeout, a TLS failure or a 5xx is a *Problem, and the end of ctx is
// ctx's error. Any other status is the kind's to read.
func Do(ctx context.Context, s Session, req *http.Request, limit int64) (Response, error) {
	call, cancel := context.WithTimeout(ctx, s.Call)
	defer cancel()
	sent := time.Now()
	res, err := s.HTTP.Do(req.WithContext(call)) //nolint:gosec // the Admin typed this client's address
	if err != nil {
		err = withoutURL(err)
		s.Log.DebugContext(ctx, "A request to a Download Client failed", "method", req.Method, "host", req.URL.Host,
			"took", time.Since(sent).String(), "err", err.Error())
		return Response{}, failed(ctx, err)
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	s.Log.DebugContext(ctx, "A Download Client responded", "method", req.Method, "host", req.URL.Host,
		"status", res.StatusCode, "bytes", len(body), "took", time.Since(sent).String())
	switch {
	case err != nil:
		return Response{}, failed(ctx, withoutURL(err))
	case int64(len(body)) > limit:
		return Response{}, fmt.Errorf("the Download Client's response is over %d bytes", limit)
	case res.StatusCode >= http.StatusInternalServerError:
		return Response{}, &Problem{Cause: ServerError, Status: res.StatusCode}
	}
	return Response{Status: res.StatusCode, Header: res.Header, Body: body}, nil
}

func failed(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return fmt.Errorf("reach the Download Client: %w", context.Cause(ctx))
	}
	return &Problem{Cause: transportCause(err), Err: err}
}

func transportCause(err error) Cause {
	if errors.Is(err, context.DeadlineExceeded) {
		return TimedOut
	}
	if t, ok := errors.AsType[interface {
		error
		Timeout() bool
	}](err); ok && t.Timeout() {
		return TimedOut
	}
	if tlsFailed(err) {
		return TLSFailed
	}
	return NoResponse
}

func tlsFailed(err error) bool {
	_, header := errors.AsType[tls.RecordHeaderError](err)
	_, verify := errors.AsType[*tls.CertificateVerificationError](err)
	_, authority := errors.AsType[x509.UnknownAuthorityError](err)
	_, hostname := errors.AsType[x509.HostnameError](err)
	_, invalid := errors.AsType[x509.CertificateInvalidError](err)
	return header || verify || authority || hostname || invalid
}

// withoutURL drops the request URL from err, since its path is the
// client's to keep out of logs.
func withoutURL(err error) error {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		return fmt.Errorf("%s: %w", ue.Op, ue.Err)
	}
	return err
}
