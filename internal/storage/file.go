package storage

// The fixed file names. internal/node's DataDir joins them onto the data
// dir, and every message about a file names it by these.
const (
	LibraryFile = "library.db"
	LocalFile   = "local.db"
)

// File is the constraint every Schema, DB, Tx and Reader is generic over.
// Library and Local are its only members, and its unexported method seals
// it, so no other package can add a third file.
type File interface {
	Library | Local
	policy() policy
}

// Library marks library.db: Works, history, settings, Accounts, secrets;
// everything a Backup copies (ADR 0018).
type Library struct{}

// Local marks local.db: a Worker's outbox, torrent resume data, the
// Controller's own runtime state. Losing it costs work, never user data.
type Local struct{}

// policy is every way the two files differ. Generic code reads it through
// policyOf[F], so the differences live in these two literals.
type policy struct {
	name string
	// rank orders nested Writes: inside a Write, only a file of higher rank
	// may be written, so library.db's writer is always taken first.
	rank        int
	synchronous string
	// quickCheck runs PRAGMA quick_check before migrating. Only the file
	// that may be discarded pays for it on every start.
	quickCheck bool
	// discardUnreadable sets a corrupt or too-new file aside and starts
	// empty, instead of refusing it.
	discardUnreadable bool
	newerAdvice       string
	corruptAdvice     string
	migrationAdvice   string
}

func (Library) policy() policy {
	return policy{
		name:              LibraryFile,
		rank:              0,
		synchronous:       "FULL",
		quickCheck:        false,
		discardUnreadable: false,
		newerAdvice:       "; restore a Backup this version took, or update SlipStream",
		corruptAdvice:     "; restore a Backup",
		migrationAdvice:   "",
	}
}

func (Local) policy() policy {
	return policy{
		name:              LocalFile,
		rank:              1,
		synchronous:       "NORMAL",
		quickCheck:        true,
		discardUnreadable: true,
		newerAdvice:       "",
		corruptAdvice:     "",
		migrationAdvice:   "; " + LocalFile + " is safe to delete",
	}
}

// files is the number of File members; holding indexes by rank.
const files = 2

func policyOf[F File]() policy {
	var f F
	return f.policy()
}
