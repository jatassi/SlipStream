package storage

const (
	LibraryFile = "library.db"
	LocalFile   = "local.db"
)

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

type policy struct {
	name              string
	rank              int
	synchronous       string
	quickCheck        bool
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

const files = 2

func policyOf[F File]() policy {
	var f F
	return f.policy()
}
