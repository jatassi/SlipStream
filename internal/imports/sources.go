package imports

import "github.com/jatassi/slipstream/internal/files"

// sourceRule is what an Import's Source decides about it.
type sourceRule struct {
	// download: its files are a Grab's torrent payload, its faults and a
	// misdescribed title go back to the Grab, and it outlives its files.
	download bool
	// picked: the Admin picked its files, so they aren't one Release. Each
	// that fills nothing waits, each that fills something waits for its
	// confirm, and each reads its quality from a release name the Admin
	// may edit.
	picked bool
	// landsInPlace: its files already sit in its own root, where they
	// land without a Transfer. It finishes once every file is final, and
	// a Work delete leaves its waiting files to Pending imports, where
	// its root's Leads show beside it.
	landsInPlace bool
	// ownRoot: its files land in its own root, from their Path there.
	ownRoot bool
	// sends: a file on another Node than its root's goes there in imports'
	// own Transfer.
	sends bool
	// placedBy is how each of its files reached its root before it
	// lands, empty when its place step moves the bytes.
	placedBy PlacedBy
	probe    probeAt
	announce announcing
	// arrival is why a landed file's Work has its Conversions converge.
	arrival files.Arrival
}

// probeAt is where a file's media record comes from.
type probeAt int

const (
	onImportNode probeAt = iota
	// carried: the file comes with its record.
	carried
)

// announcing is what history hears when a file lands.
type announcing int

const (
	loud announcing = iota
	none
	// quiet: History lists it, and connections and the bell never hear.
	quiet
)

var sourceRules = map[Source]sourceRule{
	FromGrab:       {download: true, sends: true, probe: onImportNode, announce: loud, arrival: files.ArrivalNew},
	FromFolder:     {picked: true, sends: true, probe: onImportNode, announce: loud, arrival: files.ArrivalNew},
	FromRoot:       {landsInPlace: true, ownRoot: true, probe: carried, announce: none, arrival: files.ArrivalAdopted},
	FromConversion: {ownRoot: true, placedBy: Converted, probe: carried, announce: quiet, arrival: files.ArrivalMade},
}
