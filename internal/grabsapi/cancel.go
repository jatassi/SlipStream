package grabsapi

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/grabs"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/storage"
)

type cancelInput struct {
	GrabID string `path:"grabId" format:"uuid"`
	Body   struct {
		Blocklist bool `json:"blocklist" doc:"Also put its Release on the Blocklist; only when the Grab's actions.blocklistable is true"`
	}
}

func (h handlers) cancelGrab(ctx context.Context, by identity.Admin, in *cancelInput) (*struct{}, error) {
	u, err := storage.ParseUUID(in.GrabID)
	if err != nil {
		return nil, grabNotFound
	}
	err = open(h.p).Cancel(ctx, by, grabs.GrabID{UUID: u}, in.Body.Blocklist)
	if busy, ok := errors.AsType[*grabs.GrabBusyError](err); ok {
		if busy.Words == "" {
			return nil, grabBusy
		}
		return nil, grabBusy.Because(busy.Words)
	}
	switch {
	case errors.Is(err, grabs.ErrGrabFinished):
		return nil, grabFinished
	case errors.Is(err, grabs.ErrReconciling):
		return nil, reconciling.Because("Cancel the Grab once Reconciling finishes.")
	}
	return nil, answer(err)
}
