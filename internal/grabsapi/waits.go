package grabsapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type grabWaitingInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   struct {
		Cells []cellKey `json:"cells,omitempty" doc:"The waiting Items and Targets to grab now; left out, every waiting one of the Work"`
	}
}

type grabbing struct {
	Grabbing int `json:"grabbing" minimum:"0" doc:"The Releases grabbed now. 0 when nothing waited, because the wait already ended"`
}

type grabbingOutput struct{ Body grabbing }

var grabWaitingOp = api.Op{
	ID:      "grabWaiting",
	Method:  http.MethodPost,
	Path:    "/works/{workId}/waits/grab",
	Summary: "End a Work's waits for a better Release and grab the best one found for each now",
	Status:  http.StatusAccepted,
	Errors:  []*api.Code{workNotFound, api.Reconciling},
}

func (h handlers) grabWaiting(ctx context.Context, by identity.Admin, in *grabWaitingInput) (*grabbingOutput, error) {
	u, err := storage.ParseUUID(in.WorkID)
	if err != nil {
		return nil, workNotFound
	}
	cells := make([]files.CellKey, len(in.Body.Cells))
	for i, c := range in.Body.Cells {
		item, err := storage.ParseUUID(c.ItemID)
		if err != nil {
			return nil, api.Invalid
		}
		target, err := storage.ParseUUID(c.TargetID)
		if err != nil {
			return nil, api.Invalid
		}
		cells[i] = files.CellKey{Item: works.ItemID{UUID: item}, Edition: targets.Edition(c.Edition), Target: targets.TargetID{UUID: target}}
	}
	n, err := open(h.p).GrabWaiting(ctx, by, works.WorkID{UUID: u}, cells)
	if errors.Is(err, grabs.ErrReconciling) {
		return nil, api.Reconciling
	}
	if err != nil {
		return nil, answer(err)
	}
	return &grabbingOutput{Body: grabbing{Grabbing: n}}, nil
}
