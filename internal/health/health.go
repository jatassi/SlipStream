// Package health owns what is wrong with SlipStream's own machinery: the
// registry of categories, the evaluator that alone writes health_item, the
// Report every read serves, the Storage levels and the built-in Storage
// category. Owners import health and answer for their own state; health
// imports none of them.
package health

import (
	"context"
	"embed"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/storage"
)

type file = storage.Library

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("health", schemaFiles) }

// Level is how bad an item is. None is never an item's level; it is the
// worst level of a category or a Report with nothing wrong.
type Level uint8

const (
	None Level = iota
	// Warn: something will go wrong if nobody acts.
	Warn
	// Fail: something is wrong now. The page calls it Error.
	Fail
)

func (l Level) String() string { return [...]string{"", "warn", "fail"}[l] }

func levelOf(s string) Level {
	switch s {
	case "warn":
		return Warn
	case "fail":
		return Fail
	}
	return None
}

// ID names a category: a short stable word, one of IDs.
type ID string

// Key is the owner's stable id for an item's subject, unique within its
// category and built by KeyOf. Health never shows it.
type Key string

// Says is everything shown about an item that Health never stores: the
// owner renders it from its current rows on every answer, so a rename
// shows at once.
type Says struct {
	// Subject names the item in the Admin's words: "D: (Media) on Media server".
	Subject string
	// Text is what to do first, as a sentence that stands alone.
	Text string
	// Detail is the labeled facts under Details, in order.
	Detail []Fact
	// Action is the item's one action; the zero Action is none.
	Action Action
}

// Item is one thing wrong now.
type Item struct {
	Key   Key
	Level Level
	Says
}

// Fact is one labeled fact. Exactly one of Text, Bytes and At is set, so
// the web formats sizes and times in the viewer's locale.
type Fact struct {
	Label string
	Text  string
	Bytes *int64
	At    time.Time
}

func (f Fact) set() int {
	n := 0
	for _, ok := range []bool{f.Text != "", f.Bytes != nil, !f.At.IsZero()} {
		if ok {
			n++
		}
	}
	return n
}

type ActionKind uint8

const (
	NoAction ActionKind = iota
	// ActionCheck is Check now on the item, answered by its category's
	// Source.Check.
	ActionCheck
	// ActionRun is Run now on a scheduled task.
	ActionRun
	// ActionLink is a page in the web.
	ActionLink
)

func (k ActionKind) String() string { return [...]string{"", "check", "run", "link"}[k] }

// Action is at most one of Check now, Run now on a task, or a link.
type Action struct {
	kind  ActionKind
	task  string
	href  string
	label string
}

func CheckAction() Action { return Action{kind: ActionCheck} }

func RunAction(task string) Action { return Action{kind: ActionRun, task: task} }

// LinkAction is a link to href, a route in the web such as "/nodes/<id>".
func LinkAction(href, label string) Action { return Action{kind: ActionLink, href: href, label: label} }

func (a Action) Kind() ActionKind { return a.kind }

func (a Action) Task() string { return a.task }

func (a Action) Href() string { return a.href }

func (a Action) Label() string { return a.label }

// Answer is a source's view now.
type Answer struct {
	Items []Item
	// Unsure is keys the owner can't judge now: its Node is away, its
	// facts aren't in since a restart. An open item under one keeps its
	// level and since and shows these words; none is raised or cleared.
	Unsure []Unsure
	// Next is the earliest instant at which Answer may differ with none of
	// the category's Events published, such as a task unit going overdue;
	// zero when only an event changes it. Health asks again then.
	Next time.Time
}

type Unsure struct {
	Key Key
	Says
}

// Unreached is a Check now that couldn't reach what it checks: "Media
// server is offline, so its figures are as of 14:05." Key is empty for
// the whole category.
type Unreached struct {
	Key   Key
	Words string
}

// About is a category's declaration. Its place in the registry is its
// place on the page.
type About struct {
	ID    ID
	Label string
	// Phrase is the label inside a sentence: "storage", "Nodes", "scheduled tasks".
	Phrase string
	// Events are the live events after which Answer may differ.
	Events []live.Event
}

// Source is the owner's side: Answer derives what is wrong from the
// owner's own state on every call and stores nothing.
type Source struct {
	Answer func(ctx context.Context) (Answer, error)
	// Check samples afresh what Answer reads, for one item's Key or for
	// the whole category when item is "". It answers what it couldn't
	// reach; an error is a failure of the check itself. Nil: no Check now.
	Check func(ctx context.Context, item Key) ([]Unreached, error)
}

type Category struct {
	About
	Source
}
