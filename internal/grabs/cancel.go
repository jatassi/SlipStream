package grabs

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
)

var ErrGrabFinished = errors.New("the Grab is imported or failed")

// GrabBusyError is a cancel of a Grab placing its file. Words says what it
// waits for, "" when it waits for nothing.
type GrabBusyError struct {
	Stage Stage
	Words string
}

func (e *GrabBusyError) Error() string { return "the Grab is " + string(e.Stage) }

// Cancel stops Grab id: at fetching it fails; at downloading its Torrent
// goes with reason canceled and it fails; at transferring it fails and
// the Transfer is canceled. With blocklist and an info hash known, its
// Release gets a by_hand entry. Canceling a canceled Grab changes nothing.
// It fails with ErrNoGrab, ErrGrabFinished, *GrabBusyError and
// ErrReconciling.
func (gr *Grabs) Cancel(context.Context, identity.Admin, GrabID, bool) error {
	return ErrNotBuilt
}
