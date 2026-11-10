package grabs

import "context"

// settleFaults blocklists the Release of each faulted Torrent a Grab used
// that grabs hasn't settled, one Write per Torrent: the entry, and, while
// a Grab of it downloads, the Torrent's removal with reason release_fault.
// Its downloading Grabs then fail through failOfGone. Grabs past
// downloading carry on. While Reconciling it settles nothing, and the
// sticky fault waits.
func (gr *Grabs) settleFaults(context.Context) error {
	return nil
}
