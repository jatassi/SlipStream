// Package searches is the vocabulary every owner uses to ask grabs to
// search: a Trigger, the Items asked, and the Queue that takes them in the
// caller's Write. grabs is its one implementation. It imports only works
// and storage, so owners below grabs can use it.
package searches

import (
	"context"
	"sync/atomic"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Trigger is what asked for a search. It is stored on grabs' queue rows and
// runs, so a value never changes once released.
type Trigger string

const (
	// Search is the Admin's press on a Work, level, Item, cell or Library
	// row.
	Search Trigger = "search"
	// SearchAll is Search all, and the Library's bulk Search.
	SearchAll Trigger = "search_all"
	Add       Trigger = "add"
	Monitor   Trigger = "monitor"
	Profile   Trigger = "profile"
	Editions  Trigger = "editions"
	// Retarget is a Target or Profile edit with Impact: it resets the gap
	// and queues nothing.
	Retarget    Trigger = "retarget"
	ReleaseDate Trigger = "release_date"
	// Refresh is a refresh that added a monitored, released Item.
	Refresh    Trigger = "refresh"
	Backlog    Trigger = "backlog"
	FailedGrab Trigger = "failed_grab"
	// RSS is an RSS sync run's trigger only, never a queued one.
	RSS Trigger = "rss"
)

// Triggers is every Trigger, in the order the wire lists them.
func Triggers() []Trigger {
	return []Trigger{Search, SearchAll, Add, Monitor, Profile, Editions, Retarget, ReleaseDate, Refresh, Backlog, FailedGrab, RSS}
}

// Ask is a Work's Items to search. Items nil is the whole Work.
type Ask struct {
	Work  works.WorkID
	Items []works.ItemID
}

// Queue takes asks in the caller's Write. One call is one event: a call
// with one Ask is one Work, with several is several. It never opens a
// Write, reads cells or calls a Facet. Asking twice is asking once.
type Queue interface {
	Queue(ctx context.Context, tx *storage.Tx[storage.Library], t Trigger, asks ...Ask) error
}

// None queues nothing: until grabs is bound, and in tests.
type None struct{}

func (None) Queue(context.Context, *storage.Tx[storage.Library], Trigger, ...Ask) error { return nil }

// Late is a Queue bound after the owner that holds it opens, as files
// opens before grabs. Queue panics before Bind.
type Late struct{ q atomic.Pointer[Queue] }

func (l *Late) Bind(q Queue) { l.q.Store(&q) }

func (l *Late) Queue(ctx context.Context, tx *storage.Tx[storage.Library], t Trigger, asks ...Ask) error {
	q := l.q.Load()
	if q == nil {
		panic("searches: Queue called before Bind")
	}
	return (*q).Queue(ctx, tx, t, asks...)
}
