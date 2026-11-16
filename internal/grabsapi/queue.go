package grabsapi

import (
	"context"
	"net/http"

	"github.com/jatassi/slipstream/internal/api"
	"github.com/jatassi/slipstream/internal/identity"
)

type cancelQueuedInput struct {
	Lane string `query:"lane" enum:"bulk" required:"true" doc:"The lane to empty. Only bulk can be canceled"`
}

type queuedCanceled struct {
	Canceled int `json:"canceled" minimum:"0" doc:"The queued searches canceled. A search already running finishes"`
}

type queuedCanceledOutput struct{ Body queuedCanceled }

var cancelQueuedSearchesOp = api.Op{
	ID:      "cancelQueuedSearches",
	Method:  http.MethodDelete,
	Path:    "/search-queue",
	Summary: "Cancel every search queued in a lane",
	Status:  http.StatusOK,
}

func (h handlers) cancelQueuedSearches(ctx context.Context, by identity.Admin, _ *cancelQueuedInput) (*queuedCanceledOutput, error) {
	n, err := open(h.p).CancelQueued(ctx, by)
	if err != nil {
		return nil, err
	}
	return &queuedCanceledOutput{Body: queuedCanceled{Canceled: n}}, nil
}
