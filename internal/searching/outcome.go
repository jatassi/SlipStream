package searching

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Limited is a site that limited requests: a 429 or Torznab's "Request
// limit reached".
type Limited struct {
	// RetryAfter is zero when the site gave no time.
	RetryAfter time.Duration
}

func (e *Limited) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("the Indexer limited requests for %s", e.RetryAfter)
	}
	return "the Indexer limited requests"
}

// Failing is a site that couldn't be reached or failed the request: a 5xx,
// a timeout, a connection failure or a response SlipStream can't read.
type Failing struct{ Err error }

func (e *Failing) Error() string { return e.Err.Error() }

func (e *Failing) Unwrap() error { return e.Err }

// Stopped is a site that refused the Indexer in a way only a change to it
// clears.
type Stopped struct {
	Cause StopCause
	// Text is the site's or feed's own words.
	Text string
	// SolverTried is a challenge_unsolved that a Challenge solver couldn't
	// clear.
	SolverTried bool
}

func (e *Stopped) Error() string {
	if e.Text == "" {
		return stopWords[e.Cause]
	}
	return stopWords[e.Cause] + ": " + e.Text
}

type StopCause string

const (
	CredentialsRejected  StopCause = "credentials_rejected"
	CaptchaNeeded        StopCause = "captcha_needed"
	ChallengeUnsolved    StopCause = "challenge_unsolved"
	CookieExpired        StopCause = "cookie_expired"
	DefinitionUnloadable StopCause = "definition_unloadable"
	// AccountRefused is Torznab codes 100 to 199 and 910.
	AccountRefused StopCause = "account_refused"
	// RequestRejected is Torznab codes 200 to 203.
	RequestRejected StopCause = "request_rejected"
)

var stopWords = map[StopCause]string{
	CredentialsRejected:  "the site rejected the credentials",
	CaptchaNeeded:        "the site needs a captcha answer",
	ChallengeUnsolved:    "the site's browser challenge wasn't cleared",
	CookieExpired:        "the site's cookie expired",
	DefinitionUnloadable: "the Indexer's definition doesn't load",
	AccountRefused:       "the Indexer refused the account",
	RequestRejected:      "the Indexer rejected the request",
}

// outcomeOf is err as an Answer's Outcome: nil, an outcome a driver
// returned, or any other error as Failing. jobCap words a job that ran
// past it.
func outcomeOf(ctx context.Context, err error, jobCap time.Duration) error {
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &Failing{Err: fmt.Errorf("the job ran past %s", jobCap)}
	}
	if l, ok := errors.AsType[*Limited](err); ok {
		return l
	}
	if s, ok := errors.AsType[*Stopped](err); ok {
		return s
	}
	if f, ok := errors.AsType[*Failing](err); ok {
		return f
	}
	return &Failing{Err: err}
}
