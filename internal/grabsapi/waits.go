package grabsapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

type grabWaitingInput struct {
	WorkID string `path:"workId" format:"uuid"`
	Body   struct {
		Cells []cellKey `json:"cells,omitempty" doc:"The waiting cells to grab now; left out, every waiting cell of the Work"`
	}
}

type grabbing struct {
	Grabbing int `json:"grabbing" minimum:"0" doc:"The waits ended now. 0 when nothing waited, because the wait already ended"`
}

type grabbingOutput struct{ Body grabbing }

var grabWaitingOp = api.Op{
	ID:      "grabWaiting",
	Method:  http.MethodPost,
	Path:    "/works/{workId}/waits/grab",
	Summary: "End a Work's waits for a better Release and grab the best Release each holds now",
	Status:  http.StatusAccepted,
	Errors:  []*api.Code{workNotFound, reconciling, notYet},
}

func (handlers) grabWaiting(context.Context, identity.Admin, *grabWaitingInput) (*grabbingOutput, error) {
	return nil, notYet
}
