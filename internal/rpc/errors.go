package rpc

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connectproto"
	"google.golang.org/protobuf/proto"

	"github.com/jatassi/slipstream/internal/link"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	"github.com/jatassi/slipstream/internal/role"
)

// Code is a call's error code, the same on the wire and in process.
type Code = connect.Code

const (
	CodeCanceled           = connect.CodeCanceled
	CodeUnknown            = connect.CodeUnknown
	CodeInvalidArgument    = connect.CodeInvalidArgument
	CodeDeadlineExceeded   = connect.CodeDeadlineExceeded
	CodeNotFound           = connect.CodeNotFound
	CodeAlreadyExists      = connect.CodeAlreadyExists
	CodePermissionDenied   = connect.CodePermissionDenied
	CodeResourceExhausted  = connect.CodeResourceExhausted
	CodeFailedPrecondition = connect.CodeFailedPrecondition
	CodeAborted            = connect.CodeAborted
	CodeOutOfRange         = connect.CodeOutOfRange
	CodeUnimplemented      = connect.CodeUnimplemented
	CodeInternal           = connect.CodeInternal
	CodeUnavailable        = connect.CodeUnavailable
	CodeDataLoss           = connect.CodeDataLoss
	CodeUnauthenticated    = connect.CodeUnauthenticated
)

// Errorf is a handler's error: its code and message reach the caller, and
// nothing else does.
func Errorf(code Code, format string, args ...any) error {
	return connect.Errorf(code, format, args...)
}

// CodeOf is err's code, CodeUnknown for an error no call returned.
func CodeOf(err error) Code { return connect.CodeOf(err) }

// MessageOf is the message a call's error carries without its code, err's
// text for an error no call returned.
func MessageOf(err error) string {
	if ce, ok := errors.AsType[*connect.Error](err); ok {
		return ce.Message()
	}
	return err.Error()
}

// Why is why a call never ran.
type Why uint8

const (
	// Disconnected: no live link session to the target.
	Disconnected Why = iota + 1
	// Mismatched: the session is live, but the target runs another version.
	Mismatched
	// NotHolder: the roster, or the target when its Roles changed in
	// between, says the target does not hold the Role.
	NotHolder
)

func (w Why) String() string {
	switch w {
	case Disconnected:
		return "disconnected"
	case Mismatched:
		return "mismatched"
	case NotHolder:
		return "not holder"
	}
	return fmt.Sprintf("Why(%d)", uint8(w))
}

var notHolderType = string((&nodev1.NotHolder{}).ProtoReflect().Descriptor().FullName())

// NotSent reports a call that never ran, so any call may be retried or
// sent elsewhere.
func NotSent(err error) (Why, bool) {
	switch {
	case errors.Is(err, link.ErrDisconnected):
		return Disconnected, true
	case errors.Is(err, link.ErrMismatched):
		return Mismatched, true
	}
	if ce, ok := errors.AsType[*connect.Error](err); ok {
		for _, d := range ce.Details() {
			if d.Type == notHolderType {
				return NotHolder, true
			}
		}
	}
	return 0, false
}

// Unheld is the Role a NotHolder refusal names, "" when err is none or
// names no Role, as for a Node the roster lacks.
func Unheld(err error) role.Name {
	nh, _ := DetailOf[*nodev1.NotHolder](err)
	return role.Name(nh.GetRole())
}

// ErrorWith is a handler's error carrying detail, which reaches the caller
// beside the code and message for DetailOf to read.
func ErrorWith(code Code, detail proto.Message, message string) error {
	d, err := connectproto.NewErrorDetail(detail)
	if err != nil {
		panic(fmt.Sprintf("rpc: pack %s: %v", detail.ProtoReflect().Descriptor().FullName(), err))
	}
	return connect.NewError(code, message).WithDetail(d) //nolint:wrapcheck // the error a handler answers, built here
}

// DetailOf is err's first detail of type M, false when it carries none.
func DetailOf[M proto.Message](err error) (M, bool) {
	var none M
	ce, ok := errors.AsType[*connect.Error](err)
	if !ok {
		return none, false
	}
	for _, d := range ce.Details() {
		if v, err := connectproto.UnmarshalErrorDetail(d); err == nil {
			if m, ok := v.(M); ok {
				return m, true
			}
		}
	}
	return none, false
}

// Lost reports a call whose request left before the link died or the call
// was canceled, so it may have run; only an idempotent call retries it.
func Lost(err error) bool {
	if _, ok := NotSent(err); ok {
		return false
	}
	code := CodeOf(err)
	return code == CodeUnavailable || code == CodeCanceled
}

// Answered reports an error the called Node's handler returned, as against
// one the link or the caller's side gave: a response cut short as the
// peer's link closed reads as neither an answer nor Lost.
func Answered(err error) bool {
	ce, ok := errors.AsType[*connect.Error](err)
	return ok && ce.IsRemote()
}

// CutShort reports a call that ended with no answer from the called
// Node's handler, as when that Node stopped mid-call and its link closed
// the response, so it may have run; only an idempotent call retries it.
func CutShort(err error) bool {
	ce, ok := errors.AsType[*connect.Error](err)
	return ok && !ce.IsRemote()
}

type callerKey struct{}

// CallerOf is the key of the Node whose call ctx serves: this Node's own on
// the local path, the peer's over the link.
func CallerOf(ctx context.Context) (link.Key, bool) {
	if k, ok := ctx.Value(callerKey{}).(link.Key); ok {
		return k, true
	}
	return link.PeerOf(ctx)
}
