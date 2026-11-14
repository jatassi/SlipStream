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

func queueRoutes(o *api.Owner, h handlers) {
	api.Register(o, api.Admin, api.Op{
		ID:      "cancelQueuedSearches",
		Method:  http.MethodDelete,
		Path:    "/search-queue",
		Summary: "Cancel every search queued in a lane",
		Status:  http.StatusOK,
		Errors:  []*api.Code{notYet},
	}, h.cancelQueuedSearches)
}

func (handlers) cancelQueuedSearches(context.Context, identity.Admin, *cancelQueuedInput) (*queuedCanceledOutput, error) {
	return nil, notYet
}
