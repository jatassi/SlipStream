// Package exitcode holds the process exit codes, a contract with the tray
// launcher, the service manager and the health trial. It imports nothing, so
// the release tool can render a unit's restart rule without the Node.
package exitcode

// 2 is left to Go's runtime, which exits 2 on an unrecovered panic.
const (
	OK     = 0
	Failed = 1
	Usage  = 3
	InUse  = 4
	// LibraryRefused is a library.db newer than this binary, holding a
	// schema set it lacks, or unreadable.
	LibraryRefused = 5
	// RecordRefused is a node key or pairing record refused: an unreadable
	// or malformed node.key or pairing.json.
	RecordRefused = 6
)

// NoRestart are the codes a restart cannot fix, so a service manager must
// not restart on them.
var NoRestart = []int{Usage, InUse, LibraryRefused, RecordRefused}
