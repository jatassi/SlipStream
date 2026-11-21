package metadata

import "errors"

// ErrNoChanges answers a changes list asked from further back than the
// source lists, or from a source that keeps none. The caller relies on its
// other refresh tiers instead.
var ErrNoChanges = errors.New("the metadata source lists no changes that far back")
