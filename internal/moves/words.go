package moves

import (
	"errors"
	"fmt"

	"github.com/jatassi/slipstream/internal/files"
)

var (
	// ErrReconciling refuses a Start while a restored Controller is
	// Reconciling.
	ErrReconciling = files.ErrReconciling
	ErrPlanExpired = errors.New("the Move's preview expired")
	ErrNotFound    = errors.New("no Move has this id")
	ErrNoRoot      = errors.New("no library root has this id")
	// ErrUnitSwitched refuses a cancel of a unit past its switch.
	ErrUnitSwitched = errors.New("the files already switched to the new root")
	// ErrSwitchNotWaiting refuses "Switch now" of a unit that waits for
	// no playback and no Media Server.
	ErrSwitchNotWaiting = errors.New("the Move isn't waiting for playback or a Media Server")
	ErrUnitNotFailed    = errors.New("the Move hasn't failed")
)

// PlanChangedError is a Start whose confirm isn't the plan's movable count
// now.
type PlanChangedError struct{ Was, Now int }

func (e *PlanChangedError) Error() string {
	return fmt.Sprintf("the preview showed %d to move and now shows %d", e.Was, e.Now)
}
