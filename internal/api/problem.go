package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jatassi/slipstream/internal/identity"
)

// Code is one error an operation may answer: a status, a stable snake_case
// name the client branches on, and a title a person reads. Owners declare
// codes as package-level values and list the ones an operation returns in
// its Op. A handler returns a Code as its error.
type Code struct {
	status int
	name   string
	title  string
}

func NewCode(status int, name, title string) *Code {
	return &Code{status: status, name: name, title: title}
}

func (c *Code) Error() string { return c.name }

func (c *Code) problem() *Problem {
	return &Problem{Status: c.status, Code: c.name, Title: c.title}
}

// Because answers c with detail, a reason a person reads beside the title.
func (c *Code) Because(detail string) error { return &explained{code: c, detail: detail} }

// Until answers c with Retry-After set to the whole seconds until t, and t
// in detail.
func (c *Code) Until(t time.Time) error {
	return &explained{code: c, detail: t.UTC().Format(time.RFC3339), retryAfter: t}
}

// Fields answers c with one field error per entry, for a check the
// operation's schema can't express.
func (c *Code) Fields(errs ...FieldError) error { return &explained{code: c, fields: errs} }

type explained struct {
	code       *Code
	detail     string
	retryAfter time.Time
	fields     []FieldError
}

func (e *explained) Error() string { return e.code.name + ": " + e.detail }

// answer is code's problem as e explains it; e is nil for a bare code.
func (e *explained) answer(code *Code) error {
	p := code.problem()
	if e == nil {
		return p
	}
	p.Detail, p.Errors = e.detail, e.fields
	if e.retryAfter.IsZero() {
		return p
	}
	wait := max(time.Until(e.retryAfter), 0)
	seconds := int64((wait + time.Second - 1) / time.Second)
	p.headers = http.Header{"Retry-After": {strconv.FormatInt(seconds, 10)}}
	return p
}

var (
	BadRequest           = NewCode(http.StatusBadRequest, "bad_request", "The request is malformed")
	NotFound             = NewCode(http.StatusNotFound, "not_found", "Nothing is here")
	BodyTimeout          = NewCode(http.StatusRequestTimeout, "body_timeout", "The request body took too long to arrive")
	TooLarge             = NewCode(http.StatusRequestEntityTooLarge, "too_large", "The request body is too large")
	UnsupportedMediaType = NewCode(http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the request body as JSON")
	Invalid              = NewCode(http.StatusUnprocessableEntity, "invalid", "Some fields are invalid")
	Internal             = NewCode(http.StatusInternalServerError, "internal", "Something went wrong")
)

var (
	SignedOut      = NewCode(http.StatusUnauthorized, "signed_out", "Sign in to continue")
	APIKeyWrong    = NewCode(http.StatusUnauthorized, "api_key_wrong", "The API key is wrong")
	NeedsAccount   = NewCode(http.StatusForbidden, "needs_account", "Only a signed-in person can do this")
	NeedsAdmin     = NewCode(http.StatusForbidden, "needs_admin", "Only an Admin can do this")
	SetupCodeWrong = NewCode(http.StatusForbidden, "setup_code_wrong", "The Setup code is wrong")
	SetupBusy      = NewCode(http.StatusConflict, "setup_busy", "Another setup is running")
	AlreadyClaimed = NewCode(http.StatusConflict, "already_claimed", "This Node is already set up")
	CrossOrigin    = NewCode(http.StatusForbidden, "cross_origin", "Another site cannot do this")
	WorkerNode     = NewCode(http.StatusConflict, "worker_node", "This Node is a Worker. Manage it from its Controller.")
	PINRequired    = NewCode(http.StatusForbidden, "pin_required", "Set a new PIN to continue")
	NeedsReconfirm = NewCode(http.StatusForbidden, "needs_reconfirm", "Confirm it's you to continue")
)

var humaStatusCodes = map[int]*Code{
	http.StatusBadRequest:            BadRequest,
	http.StatusNotFound:              NotFound,
	http.StatusRequestTimeout:        BodyTimeout,
	http.StatusRequestEntityTooLarge: TooLarge,
	http.StatusUnsupportedMediaType:  UnsupportedMediaType,
	http.StatusUnprocessableEntity:   Invalid,
	http.StatusInternalServerError:   Internal,
}

var identityCodes = []struct {
	err  error
	code *Code
}{
	{identity.ErrSignedOut, SignedOut},
	{identity.ErrAPIKeyWrong, APIKeyWrong},
	{identity.ErrNeedsAccount, NeedsAccount},
	{identity.ErrNeedsAdmin, NeedsAdmin},
	{identity.ErrSetupCodeWrong, SetupCodeWrong},
	{identity.ErrSetupBusy, SetupBusy},
	{identity.ErrAlreadyClaimed, AlreadyClaimed},
	{identity.ErrWorkerNode, WorkerNode},
	{identity.ErrPINRequired, PINRequired},
	{identity.ErrNeedsReconfirm, NeedsReconfirm},
}

func identityCode(err error) (*Code, bool) {
	for _, m := range identityCodes {
		if errors.Is(err, m.err) {
			return m.code, true
		}
	}
	return nil, false
}

// Problem is the one error body the API writes: an RFC 9457 problem with
// the extension member code. It has no value member, so a rejected input is
// never echoed.
type Problem struct {
	Status int          `json:"status"`
	Code   string       `json:"code"`
	Title  string       `json:"title"`
	Detail string       `json:"detail,omitempty"`
	Errors []FieldError `json:"errors,omitempty" nullable:"false"`

	headers http.Header
}

type FieldError struct {
	Location string `json:"location"`
	Message  string `json:"message"`
}

func (p *Problem) Error() string { return p.Code }

func (p *Problem) GetStatus() int { return p.Status }

func (p *Problem) GetHeaders() http.Header { return p.headers }

func (p *Problem) ContentType(string) string { return problemType }

const problemType = "application/problem+json"

func init() {
	huma.NewError = newError
}

// newError maps Huma's own errors to codes by status. Only 422 keeps its
// field errors; any status the map lacks answers internal.
func newError(status int, _ string, errs ...error) huma.StatusError {
	code, ok := humaStatusCodes[status]
	if !ok {
		code = Internal
	}
	p := code.problem()
	if code != Invalid {
		return p
	}
	for _, err := range errs {
		if d, ok := err.(huma.ErrorDetailer); ok {
			detail := d.ErrorDetail()
			p.Errors = append(p.Errors, FieldError{Location: detail.Location, Message: detail.Message})
		}
	}
	return p
}

func writeProblem(w http.ResponseWriter, p *Problem) {
	body, err := json.Marshal(p)
	if err != nil {
		panic(fmt.Sprintf("api: marshal a problem: %v", err))
	}
	w.Header().Set("Content-Type", problemType)
	w.WriteHeader(p.Status)
	_, _ = w.Write(append(body, '\n'))
}
