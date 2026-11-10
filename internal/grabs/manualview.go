package grabs

import (
	"context"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/formats"
	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/indexers"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// ManualView is a Work's Manual search as the sheet shows it, every
// sentence written.
type ManualView struct {
	ID ManualID
	// Title is the scope in words: "Heat (1995)", "Season 2 of Severance".
	Title string
	// Query is the typed query, "" when each run sent its Item's terms.
	Query     string
	Started   time.Time
	StartedBy Actor
	// Searching is true while a run has a line with no outcome.
	Searching bool
	// Queryable is true when the scope is one Item, so a typed query can
	// search it.
	Queryable bool
	// Line is "Searching 6 Indexers. 4 have responded." or "SlipStream
	// searched 6 Indexers. 2 didn't respond."
	Line     string
	Indexers []IndexerLine
	Plan     ManualPlan
	Folded   Folded
	// Releases is how many Releases All Releases lists.
	Releases int
	// Pins is the pins still in effect, for the sheet to keep. A pin that
	// names no Release, or that no longer fills its cell, is dropped.
	Pins []Pin
}

// IndexerLine is one Indexer across the search's runs.
type IndexerLine struct {
	// Indexer is zero once the Indexer is deleted.
	Indexer indexers.ID
	Name    string
	// Sentence is S26 (Indexers and Trackers)'s outcome sentence, or
	// "TorrentLeech searched 6 of 10 episodes. Its limit resets at 14:05."
	Sentence string
	// Done is true once it has an outcome for every run that asked it.
	Done bool
	// Automatic is true when automatic search uses it.
	Automatic bool
	// Failed is a fault at the site, which the sheet shows red with Retry
	// and Open Indexer.
	Failed bool
}

// ManualPlan is the fewest Grabs that fill the scope's missing, failed and
// upgradable cells, worked out around the pins.
type ManualPlan struct {
	// Headline is "2 Grabs get you everything.", "1 Grab gets you 2 of the
	// 3 files you're missing. SlipStream hasn't found a Release for 4K
	// yet." or "Every Target here already has its file."
	Headline string
	// Rows is a Grab or a cell nothing fills, in Want order.
	Rows []Row
	// Held is each cell an unfinished Grab downloads, left out of the plan.
	Held []HeldCell
}

// Row is a plan row in words. Pick is nil for a cell nothing fills.
type Row struct {
	// Cells is the cells it fills, the first leading it. A pin chosen from
	// its choices replaces the pin of its first cell.
	Cells   []files.CellKey
	Pick    *PlanPick
	Pinned  bool
	Outcome string // "It gets you 1080p and 720p for E05."
	// Missing is a cell nothing fills: "Missing: 720p for E05.", "Could be
	// better: 1080p SDR." or "Failed: 1080p for E10."
	Missing string
	// Cause is Missing's plain cause: "No Indexer listed a 720p Release."
	Cause string
	// Failure is the cell's last Grab's failure, for a failed cell.
	Failure string
	Choices []Choice
}

// PlanPick is what a pick of the row sends back verbatim, with the
// Release it names.
type PlanPick struct {
	Release ReleaseDetails
	// Lifts is the soft refusals it gets past, each a sentence.
	Lifts []Worded
	// Edition is the Edition it is grabbed as, zero unless that is another
	// Edition than its cut's.
	Edition targets.Edition
	// Routed is "Where Tracker routing sends it (qBittorrent on Den)".
	Routed string
}

// Choice is one other Release that could fill a row's first cell.
type Choice struct {
	Pin     Pin
	Release ReleaseDetails
	Lifts   []Worded
	Edition targets.Edition
	// Note is "From TorrentLeech, which automatic search doesn't use."
	Note string
}

// HeldCell is a cell a Grab downloads, left out of the plan.
type HeldCell struct {
	Cell files.CellKey
	Grab GrabID
	// Words is "A Grab is downloading Heat.1995.1080p.WEB for 1080p."
	Words string
	Stage Stage
}

// Folded is "SlipStream wouldn't grab 14 more Releases." and its groups.
type Folded struct {
	Line   string
	Groups []FoldGroup
}

// FoldGroup is one group of the Releases SlipStream wouldn't grab: "8 are
// 4K, which no Target here takes." Fault is a real fault, which the sheet
// shows red.
type FoldGroup struct {
	Words string
	Fault bool
}

// ReleaseRef names a Manual run's Release: its run and its position in
// arrival order. A pick names its group's primary when the sheet drew
// it, and the grab Write takes whichever copy is primary then.
type ReleaseRef struct {
	Run      SearchID
	Position int
}

// Pin is an Admin's swap: Release fills Cell, whatever ranks above it,
// grabbed as Cell's Edition.
type Pin struct {
	Cell    files.CellKey
	Release ReleaseRef
}

// ReleaseDetails is one Release with its facts and its verdicts in words.
type ReleaseDetails struct {
	Ref   ReleaseRef
	Title string
	// Words is the Release in words, as release.Describe writes it.
	Words string
	// Indexers is the Indexers that listed it, by priority.
	Indexers []string
	Tracker  string
	Size     int64
	// Published is zero when the Indexer sent no date.
	Published time.Time
	Seeders   int
	Freeleech bool
	Formats   []formats.ID
	// Verdicts is one per Target of its Item: "It would fill 1080p.", "It
	// meets 720p. It isn't better than the file you have."
	Verdicts []string
	// LostAt is where it lost to its first cell's pick, nil when it
	// didn't.
	LostAt *Loss
	// Blocked is "It's been on the Blocklist since 2 Nov because
	// TorrentLeech reported it unregistered.", "" when it isn't on it.
	Blocked string
}

// Offer is one Release as All Releases lists it, one per group of copies,
// in rank order.
type Offer struct {
	Release ReleaseDetails
	// Rank is its place in the ranking from 1, 0 when no cell's Target
	// takes it. Score and Quality are its best Target's, for sorting.
	Rank    int
	Score   int
	Quality int
	Grab    GrabOffer
}

// GrabOffer is what grabbing an Offer by itself does, or why it can't.
type GrabOffer struct {
	// Grabbable is true when it fills some cell and its Indexer would take
	// it now. A Release on the Blocklist is grabbable only with Unblock.
	Grabbable bool
	Outcome   string
	Lifts     []Worded
	Edition   targets.Edition
	// Cancels is each Grab downloading a cell it would also fill.
	Cancels []HeldCell
	// Waits is "TorrentLeech is at its grab limit until 14:05."
	Waits string
	// Refusal is "It's a season pack, and SlipStream can't import packs
	// yet."
	Refusal string
}

// ManualSearch is work's Manual search with its plan worked out around
// pins. It fails with ErrNoManualSearch and ErrNoWork.
func (gr *Grabs) ManualSearch(context.Context, identity.Admin, works.WorkID, []Pin) (ManualView, error) {
	return ManualView{}, ErrNotBuilt
}

// ManualReleases is every Release of work's Manual search, one per group
// of copies, in rank order. It fails with ErrNoManualSearch and ErrNoWork.
func (gr *Grabs) ManualReleases(context.Context, identity.Admin, works.WorkID) ([]Offer, error) {
	return nil, ErrNotBuilt
}
