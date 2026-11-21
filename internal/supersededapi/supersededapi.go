// Package supersededapi is the Superseded page: what fenced this
// Controller, Stop this Controller after an unproven reply, Resume, and
// Wipe and join as a Worker.
package supersededapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/joincode"
	"github.com/jatassi/slipstream/internal/lifecycle"
	"github.com/jatassi/slipstream/internal/nodes"
)

// Fence is the Controller's fence and what ends or follows it.
type Fence interface {
	Fenced(ctx context.Context) (nodes.Fence, bool, error)
	// Name is the name of the Node that carried the claim.
	Name(node nodes.ID) string
	Stop(ctx context.Context) error
	// Retire makes this Controller Superseded where no Worker carries a
	// proof.
	Retire(ctx context.Context) error
	Resume(ctx context.Context) error
	// Wipe takes the superseded Backup, writes the wipe record and
	// restarts, so the next start deletes the library and joins with code.
	Wipe(ctx context.Context, code joincode.Code) error
}

// Fencer is the fence, false unless the Node is a Controller.
type Fencer interface {
	Fence() (Fence, bool)
}

// ErrBackupFailed is Wipe when the superseded Backup failed, so nothing
// was deleted.
var ErrBackupFailed = errors.New("the Backup before the wipe failed")

var (
	notTold          = api.NewCode(http.StatusConflict, "not_told", "No Worker said a restore replaced this Controller")
	notSuperseded    = api.NewCode(http.StatusConflict, "not_superseded", "This Controller wasn't replaced")
	joinCodeDamaged  = api.NewCode(http.StatusUnprocessableEntity, "join_code_damaged", "The Join Code is mistyped, cut short or not a SlipStream code")
	wipeBackupFailed = api.NewCode(http.StatusConflict, "wipe_backup_failed", "The Backup before the wipe failed, so nothing was wiped. Open the Backup task for the cause.")
)

var codes = []struct {
	err  error
	code *api.Code
}{
	{nodes.ErrNotTold, notTold},
	{nodes.ErrNotSuperseded, notSuperseded},
	{lifecycle.ErrBusy, api.LifecycleBusy},
	{ErrBackupFailed, wipeBackupFailed},
}

func answer(err error) error {
	for _, c := range codes {
		if errors.Is(err, c.err) {
			return c.code
		}
	}
	return err
}

// State is the fence's state, none while there is no fence.
type State string

const None State = "none"

type view struct {
	State State `json:"state" enum:"none,unproven,proven,stopped,retired"`
	// At is when a Worker said a restore replaced this Controller, or an
	// Admin stopped or retired it.
	At *time.Time `json:"at,omitempty"`
	// Replacement is the Controller a restore made, as the Worker gave it.
	Replacement *replacement `json:"replacement,omitempty"`
	// Worker is the name of the Node that carried the claim.
	Worker *string `json:"worker,omitempty"`
	// WorkerID is that Node's id, so its page can offer Stop.
	WorkerID *nodes.ID `json:"workerId,omitempty"`
}

type replacement struct {
	Key string `json:"key"`
	// Address is the host the Controller listens at, "" when the Worker
	// gave none.
	Address string `json:"address"`
}

type viewOutput struct {
	Body view
}

type wipeInput struct {
	Body struct {
		JoinCode string `json:"joinCode" minLength:"1" maxLength:"1024"`
	}
}

func viewOf(f nodes.Fence, fenced bool, name func(nodes.ID) string) view {
	if !fenced {
		return view{State: None}
	}
	at := f.At.UTC()
	v := view{State: State(f.State), At: &at}
	if f.State != nodes.Retired {
		r := replacement{Key: f.By.Fingerprint()}
		if all := f.Addresses.All(); len(all) > 0 {
			r.Address = all[0].Addr().String()
		}
		v.Replacement = &r
	}
	if !f.ByNode.IsZero() {
		worker, id := name(f.ByNode), f.ByNode
		v.Worker, v.WorkerID = &worker, &id
	}
	return v
}

func open(n Fencer) Fence {
	f, ok := n.Fence()
	if !ok {
		panic("supersededapi: an operation ran on a Node that is no Controller")
	}
	return f
}

func Routes(r *api.Router, n Fencer) {
	o := r.Owner("superseded", api.Admin)

	api.Register(o, api.Admin, api.Op{
		ID: "getSuperseded", Method: http.MethodGet, Path: "/superseded",
		Summary: "Read whether a restore on another machine replaced this Controller", Status: http.StatusOK,
		WhileSuperseded: true,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*viewOutput, error) {
		f := open(n)
		fence, fenced, err := f.Fenced(ctx)
		if err != nil {
			return nil, err
		}
		return &viewOutput{Body: viewOf(fence, fenced, f.Name)}, nil
	})

	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "stopSupersededController", Method: http.MethodPost, Path: "/superseded/stop",
		Summary: "Stop this Controller after a Worker said a restore replaced it", Status: http.StatusNoContent,
		Errors: []*api.Code{notTold}, WhileSuperseded: true,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*struct{}, error) {
		return nil, answer(open(n).Stop(ctx))
	})

	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "retireController", Method: http.MethodPost, Path: "/backups/retire",
		Summary: "Retire this Controller after its library was restored on another machine", Status: http.StatusNoContent,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*struct{}, error) {
		return nil, open(n).Retire(ctx)
	})

	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "resumeSupersededController", Method: http.MethodPost, Path: "/superseded/resume",
		Summary: "Resume this Controller, so no earlier proof of a restore fences it again", Status: http.StatusNoContent,
		Errors: []*api.Code{notSuperseded}, WhileSuperseded: true,
	}, func(ctx context.Context, _ identity.Admin, _ *struct{}) (*struct{}, error) {
		return nil, answer(open(n).Resume(ctx))
	})

	api.Register(o, api.ReconfirmedSession, api.Op{
		ID: "wipeSupersededController", Method: http.MethodPost, Path: "/superseded/wipe",
		Summary: "Back up and wipe this Controller, then join it to the new Controller as a Worker", Status: http.StatusAccepted,
		Errors: []*api.Code{joinCodeDamaged, notSuperseded, api.LifecycleBusy, wipeBackupFailed}, WhileSuperseded: true,
	}, func(ctx context.Context, _ identity.Admin, in *wipeInput) (*struct{}, error) {
		code, err := joincode.Parse(in.Body.JoinCode)
		if err != nil {
			return nil, joinCodeDamaged
		}
		return nil, answer(open(n).Wipe(ctx, code))
	})
}
