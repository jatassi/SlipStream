package library

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

// errNotImplemented answers a choice the contract names and no change
// applies yet: a preset other than All or None, Include, a Monitor new
// override, a scoped monitoring change or a Work's options.
var errNotImplemented = errors.New("not implemented")

// MonitoringOutcome is one Work's result: its Outcome, what moved (or
// would), and on a preview what each preset would monitor of it.
type MonitoringOutcome struct {
	Outcome
	Delta  works.Delta
	Counts works.PresetCounts
}

// SetMonitoring applies c to each of ws in one Write, all or none;
// preview rolls it back. A Work that moves nothing is skipped as already;
// an unknown one fails. Searches nothing.
func (l *Library) SetMonitoring(context.Context, identity.Admin, []works.WorkID, works.Choice, bool) ([]MonitoringOutcome, error) {
	return nil, errNotImplemented
}

// SetMonitored turns monitoring on or off for the levels and Items part
// names of Work id, each level standing for every Item under it.
// works.ErrNotInWork answers an id of another Work.
func (l *Library) SetMonitored(context.Context, identity.Admin, works.WorkID, works.Within, bool) error {
	return errNotImplemented
}

// SetOptions changes Work id's options; preview reports what the change
// would withdraw and writes nothing.
func (l *Library) SetOptions(ctx context.Context, by identity.Admin, id works.WorkID, o works.Optioning, preview bool) (works.OptionsChange, error) {
	return l.cfg.Works.SetOptions(ctx, by, id, o, preview)
}
