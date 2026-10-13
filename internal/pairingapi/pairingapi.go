// Package pairingapi holds the pairing owner's operation: joining a fresh
// Node to a Controller as a Worker.
package pairingapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/pairing"
)

// Joiner is the Node as the Join reaches it.
type Joiner interface {
	// Join runs pairing.Claim and, once joined, makes the Node a Worker.
	Join(ctx context.Context, code joincode.Code) (pairing.Joined, error)
}

var (
	joinCodeDamaged       = api.NewCode(http.StatusUnprocessableEntity, "join_code_damaged", "The Join Code is mistyped, cut short or not a SlipStream code")
	joinCodeUnknown       = api.NewCode(http.StatusForbidden, "join_code_unknown", "The Controller does not know this Join Code")
	joinCodeExpired       = api.NewCode(http.StatusForbidden, "join_code_expired", "The Join Code has expired")
	joinCodeUsed          = api.NewCode(http.StatusForbidden, "join_code_used", "The Join Code was already used")
	controllerUnreachable = api.NewCode(http.StatusBadGateway, "controller_unreachable", "The Controller did not answer")
	controllerKeyMismatch = api.NewCode(http.StatusBadGateway, "controller_key_mismatch", "The Controller's key does not match the Join Code")
)

var refusals = map[pairing.Kind]*api.Code{
	pairing.CodeUnknown:          joinCodeUnknown,
	pairing.CodeExpired:          joinCodeExpired,
	pairing.CodeUsed:             joinCodeUsed,
	pairing.Unreachable:          controllerUnreachable,
	pairing.KeyMismatch:          controllerKeyMismatch,
	pairing.AlreadyJoined:        api.AlreadyClaimed,
	pairing.SupersededController: api.AlreadyClaimed,
	pairing.IsController:         api.AlreadyClaimed,
}

type joinInput struct {
	Body joinBody
}

type joinBody struct {
	Code string `json:"code" minLength:"1" maxLength:"1024"`
}

type joinOutput struct {
	Body worker
}

type worker struct {
	NodeID     string     `json:"nodeId" format:"uuid"`
	Controller controller `json:"controller"`
}

type controller struct {
	Name string `json:"name"`
	Key  string `json:"key" doc:"The Controller key's fingerprint"`
	// Link is the Controller's address on the live session.
	Link string `json:"link"`
}

func Routes(r *api.Router, j Joiner) {
	o := r.Owner("pairing", api.Setup)

	api.Register(o, api.Setup, api.Op{
		ID:      "claimWorker",
		Method:  http.MethodPost,
		Path:    "/setup/worker",
		Summary: "Join this Node to a Controller as a Worker with a Join Code",
		Status:  http.StatusCreated,
		Errors:  []*api.Code{joinCodeDamaged, joinCodeUnknown, joinCodeExpired, joinCodeUsed, controllerUnreachable, controllerKeyMismatch},
	}, func(ctx context.Context, _ identity.Claim, in *joinInput) (*joinOutput, error) {
		code, err := joincode.Parse(in.Body.Code)
		if err != nil {
			return nil, joinCodeDamaged
		}
		joined, err := j.Join(ctx, code)
		if e, ok := errors.AsType[*pairing.Error](err); ok {
			return nil, refusals[e.Kind]
		}
		if err != nil {
			return nil, err
		}
		return &joinOutput{Body: worker{
			NodeID:     joined.NodeID.String(),
			Controller: controller{Name: joined.ControllerName, Key: joined.Controller.Fingerprint(), Link: joined.Link.String()},
		}}, nil
	})
}
