// Package pairingapi holds the pairing owner's operations: joining a fresh
// Node to a Controller as a Worker, and reading a Join that awaits its
// Controller's dial.
package pairingapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/pairing"
)

// Joiner is the Node as the Join reaches it.
type Joiner interface {
	// Join runs pairing.Claim and, once joined, makes the Node a Worker.
	// It calls awaiting once if the Claim awaits its Controller's dial.
	Join(ctx context.Context, code joincode.Code, awaiting func(pairing.Pending)) (pairing.Joined, error)
	// Pending is the last Join that awaited its Controller's dial.
	Pending() (pairing.Pending, bool)
}

var (
	joinCodeDamaged       = api.NewCode(http.StatusUnprocessableEntity, "join_code_damaged", "The Join Code is mistyped, cut short or not a SlipStream code")
	joinCodeUnknown       = api.NewCode(http.StatusForbidden, "join_code_unknown", "The Controller does not know this Join Code")
	joinCodeExpired       = api.NewCode(http.StatusForbidden, "join_code_expired", "The Join Code has expired")
	joinCodeUsed          = api.NewCode(http.StatusForbidden, "join_code_used", "The Join Code was already used")
	controllerKeyMismatch = api.NewCode(http.StatusBadGateway, "controller_key_mismatch", "The Controller's key does not match the Join Code")
	restartRequired       = api.NewCode(http.StatusConflict, "restart_required", "A running Worker re-pairs only at start")
	nodeUnaddressed       = api.NewCode(http.StatusConflict, "node_unaddressed", "The Controller is unreachable, and this Node's link has no address for it to dial")
	noJoinPending         = api.NewCode(http.StatusNotFound, "no_join_pending", "No Join has awaited its Controller's dial")
)

var refusals = map[pairing.Kind]*api.Code{
	pairing.CodeUnknown:          joinCodeUnknown,
	pairing.CodeExpired:          joinCodeExpired,
	pairing.CodeUsed:             joinCodeUsed,
	pairing.KeyMismatch:          controllerKeyMismatch,
	pairing.AlreadyJoined:        api.AlreadyClaimed,
	pairing.SupersededController: api.AlreadyClaimed,
	pairing.IsController:         api.AlreadyClaimed,
	pairing.RestartRequired:      restartRequired,
	pairing.JoinPending:          api.JoinPending,
	pairing.Unaddressed:          nodeUnaddressed,
}

type joinInput struct {
	Body joinBody
}

type joinBody struct {
	Code string `json:"code" minLength:"1" maxLength:"1024"`
}

type claimOutput struct {
	Status int
	Body   workerClaim
}

// workerClaim is a Join as the first-run page follows it: joined, awaiting
// the Controller's dial at a Node address, or failed while it waited.
type workerClaim struct {
	State       string      `json:"state" enum:"awaiting-dial,joined,failed"`
	NodeID      string      `json:"nodeId,omitempty" format:"uuid" doc:"This Node's id; set once joined"`
	Controller  *controller `json:"controller,omitempty" doc:"Set once joined"`
	NodeAddress string      `json:"nodeAddress,omitempty" doc:"The SLN1 text to paste into the Controller's Add a Node dialog; absent once joined without a wait"`
	Until       *time.Time  `json:"until,omitempty" doc:"When this Node stops waiting for the dial"`
	Problem     *problem    `json:"problem,omitempty" doc:"Why a failed Join failed"`
}

type controller struct {
	Name string `json:"name"`
	Key  string `json:"key" doc:"The Controller key's fingerprint"`
	// Link is the Controller's address on the live session.
	Link string `json:"link"`
}

type problem struct {
	Code   string `json:"code" enum:"controller_unreachable,join_code_unknown,join_code_expired,join_code_used,controller_key_mismatch,internal"`
	Detail string `json:"detail"`
}

func joinedOf(j pairing.Joined) workerClaim {
	return workerClaim{
		State:      pairing.JoinedState.String(),
		NodeID:     j.NodeID.String(),
		Controller: &controller{Name: j.ControllerName, Key: j.Controller.Fingerprint(), Link: j.Link.String()},
	}
}

func pendingOf(p pairing.Pending) workerClaim {
	until := p.Until
	out := workerClaim{State: p.State.String(), NodeAddress: p.Address.String(), Until: &until}
	switch {
	case p.Joined != nil:
		joined := joinedOf(*p.Joined)
		out.NodeID, out.Controller = joined.NodeID, joined.Controller
	case p.Err != nil:
		code := string(p.Err.Kind)
		if code == "" {
			code = "internal"
		}
		out.Problem = &problem{Code: code, Detail: p.Err.Reason}
	}
	return out
}

func Routes(r *api.Router, j Joiner) {
	o := r.Owner("pairing", api.Setup)

	api.Register(o, api.Setup, api.Op{
		ID:      "claimWorker",
		Method:  http.MethodPost,
		Path:    "/setup/worker",
		Summary: "Join this Node to a Controller as a Worker with a Join Code; 202 while it awaits the Controller's dial",
		Status:  http.StatusCreated,
		Also:    []int{http.StatusAccepted},
		Errors:  []*api.Code{joinCodeDamaged, joinCodeUnknown, joinCodeExpired, joinCodeUsed, controllerKeyMismatch, restartRequired, api.JoinPending, nodeUnaddressed},
	}, func(ctx context.Context, _ identity.Claim, in *joinInput) (*claimOutput, error) {
		code, err := joincode.Parse(in.Body.Code)
		if err != nil {
			return nil, joinCodeDamaged
		}
		type result struct {
			joined pairing.Joined
			err    error
		}
		done := make(chan result, 1)
		awaiting := make(chan pairing.Pending, 1)
		go func() {
			joined, err := j.Join(context.WithoutCancel(ctx), code, func(p pairing.Pending) { awaiting <- p })
			done <- result{joined, err}
		}()
		select {
		case p := <-awaiting:
			return &claimOutput{Status: http.StatusAccepted, Body: pendingOf(p)}, nil
		case res := <-done:
			select {
			case p := <-awaiting:
				return &claimOutput{Status: http.StatusAccepted, Body: pendingOf(p)}, nil
			default:
			}
			if e, ok := errors.AsType[*pairing.Error](res.err); ok {
				return nil, refusals[e.Kind]
			}
			if res.err != nil {
				return nil, res.err
			}
			return &claimOutput{Status: http.StatusCreated, Body: joinedOf(res.joined)}, nil
		}
	})

	api.Register(o, api.Setup, api.Op{
		ID:      "getWorkerClaim",
		Method:  http.MethodGet,
		Path:    "/setup/worker",
		Summary: "Get the Join that awaited its Controller's dial: still awaiting, joined or failed",
		Status:  http.StatusOK,
		Errors:  []*api.Code{noJoinPending},
	}, func(context.Context, identity.Claim, *struct{}) (*claimOutput, error) {
		p, ok := j.Pending()
		if !ok {
			return nil, noJoinPending
		}
		return &claimOutput{Status: http.StatusOK, Body: pendingOf(p)}, nil
	})
}
