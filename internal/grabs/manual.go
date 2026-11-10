package grabs

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/works"
)

var (
	ErrNoManualSearch       = errors.New("the Work has no Manual search")
	ErrManualSearchReplaced = errors.New("another Manual search of the Work replaced this one")
	ErrQueryNeedsOneItem    = errors.New("a typed query searches one Item")
	ErrNoCell               = errors.New("the Work's Item has no cell of that Edition and Target")
)

// ManualScope is the part of a Work a Manual search covers. Cell narrows
// the plan to one cell of the one Item Part names. Query is typed text
// for one Item. Again starts a new search even when the Work's Manual
// search has this scope and query.
type ManualScope struct {
	Part  Scope
	Cell  *files.CellKey
	Query string
	Again bool
}

// StartManualSearch records a Manual search of work within s: one run per
// Item in scope, monitoring and release dates aside, each asking every on
// Indexer that can search, Automatic search off included. Unless s.Again,
// a Manual search of work with s's scope and query is answered, started
// false. Otherwise it replaces the Work's Manual search in the same Write
// and, once committed, stops the runs it replaced. It fails with
// ErrReconciling, ErrQueryNeedsOneItem, ErrNoCell, ErrNotSearchable,
// ErrNoIndexer, ErrNoRoot, ErrNoWork and *works.NotInWorkError. An
// unreachable root doesn't stop it.
func (gr *Grabs) StartManualSearch(context.Context, identity.Admin, works.WorkID, ManualScope) (id ManualID, started bool, err error) {
	return ManualID{}, false, ErrNotBuilt
}

// listen is a Manual run's goroutine: it asks the run's lines with no
// outcome and stores each answer as it lands. A restart starts it again,
// and it asks only the lines still unanswered.
func (gr *Grabs) listen(context.Context, Search) result {
	return halted
}
