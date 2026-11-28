package conversions

import "context"

// MoveUp puts a queued Conversion at the head of Moved up.
func (c *Conversions) MoveUp(context.Context, ID) (ListRow, error) { return ListRow{}, errNotBuilt }

// MoveAfter puts a queued Conversion right after after in its lane, or at
// the head of Moved up when after is nil.
func (c *Conversions) MoveAfter(context.Context, ID, *ID) (ListRow, error) {
	return ListRow{}, errNotBuilt
}

// Retry queues a failed Conversion at the head of its lane with a fresh
// budget.
func (c *Conversions) Retry(context.Context, ID) (ListRow, error) { return ListRow{}, errNotBuilt }

// Cancel cancels an unfinished Conversion and declines its pair.
func (c *Conversions) Cancel(context.Context, ID) (ListRow, error) { return ListRow{}, errNotBuilt }

// CancelGroup cancels every unfinished Conversion of a group, answering
// how many.
func (c *Conversions) CancelGroup(context.Context, GroupID) (int, error) { return 0, errNotBuilt }
