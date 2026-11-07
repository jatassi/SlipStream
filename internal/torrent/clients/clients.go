// Package clients is what SlipStream knows of an external Download
// Client apart from any one product's protocol: each kind's facts, the
// address, label and Client folders an Admin enters, the problems a client
// can have, and the contract each kind's package fills. It holds no
// policy: when to poll, how long to wait and what absence means belong to
// the Node's actor.
package clients

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
)

// Session is what one Client needs to reach its product.
type Session struct {
	HTTP     *http.Client
	Address  Address
	Username string
	Secrets  Secrets
	// Daemon is Deluge's chosen daemon, nil for the only one.
	Daemon *Daemon
	// TrackerEvery is the least time between two reads of one torrent's
	// tracker messages, for a kind that reads them apart from its status.
	TrackerEvery time.Duration
	// Call bounds each request.
	Call time.Duration
	Log  *slog.Logger
}

// Secrets are the values sealed on the Controller and held in Node
// memory only.
type Secrets struct {
	Password string
	APIKey   string
}

func (Secrets) LogValue() slog.Value { return slog.StringValue("[secrets]") }

type Daemon struct {
	Host string
	Port int
}

// Client is one session with one product. Its owner calls it from one
// goroutine, so calls never overlap. Every method signs in first when the
// session has none, and signs in again once when the product says the
// session expired. A failure is a *Problem, a context error, or a plain
// error only for a response the kind can't parse.
type Client interface {
	// Test signs in, reads the version and checks what only the product
	// can say: that label can be used, and the warnings. It changes
	// nothing in the client. A version under the kind's Floor is a Problem
	// with Cause VersionTooOld. A Deluge Test that fails with DaemonNeeded
	// or WrongDaemon still answers Tested.Daemons, for the Admin to choose
	// from.
	Test(ctx context.Context, label Label) (Tested, error)
	// Version is the product's version, read at the last sign-in.
	Version(ctx context.Context) (Version, error)
	// DefaultFolder is where the client saves a torrent added with no
	// folder under label, as the client sees it.
	DefaultFolder(ctx context.Context, label Label) (string, error)
	// Add sends a .torrent's bytes. A torrent already in the client is
	// nil: the next poll resolves it. Add creates the label first where
	// the kind needs one (qBittorrent's category, Deluge's label). Deluge
	// sets the label in a call of its own, so an Add of a torrent already
	// there sets a label it lacks. The Node heals a torrent it added whose
	// Status.Label is not the label by sending the Add again.
	Add(ctx context.Context, a Add) error
	// Poll is the status of each of hashes the client lists. A hash it
	// doesn't list is missing from the map. A session to the wrong Deluge
	// daemon is a Problem, never an empty map.
	Poll(ctx context.Context, hashes []bencode.InfoHash) (map[bencode.InfoHash]Status, error)
	// Files is one torrent's files, in index order.
	Files(ctx context.Context, h bencode.InfoHash) ([]File, error)
	Act(ctx context.Context, a Act, hashes []bencode.InfoHash) error
	// Remove removes one torrent, deleting its data when data is set and
	// the kind DeletesData.
	Remove(ctx context.Context, h bencode.InfoHash, data bool) error
	// Close signs out where the kind has a sign-out, and drops the session.
	Close(ctx context.Context) error
}

type Act uint8

const (
	Pause Act = iota + 1
	Resume
	Recheck
	Announce
)

type Add struct {
	Metainfo []byte
	Hash     bencode.InfoHash
	// Folder is as the client sees it: <download folder>/<Content Type
	// name>.
	Folder string
	Label  Label
	Paused bool
}

// Tested is what a passing Test learned.
type Tested struct {
	Version  Version
	Warnings []Warning
	// Daemons are the daemons Deluge's web UI lists, nil for other kinds.
	Daemons []Daemon
	// MovesAfterFinish is a client set to move a finished torrent's files,
	// which the Node warns about while the Import delay is 0.
	MovesAfterFinish bool
}

type Warning uint8

const (
	// RemovesAtLimit is a client that removes torrents at a seeding limit.
	RemovesAtLimit Warning = iota + 1
	// MovesAfterFinish is a client that moves finished files while the
	// Import delay is 0. The Node adds it, since only the Node knows the
	// delay.
	MovesAfterFinish
	// NoMovingState is a kind that can't report moving files while the
	// Import delay is 0, added by the Node from the kind's facts.
	NoMovingState
)

// Status is one torrent as the client reports it, in SlipStream's terms.
type Status struct {
	Name       string
	Size, Done int64
	// Run is what the client means to do with it: started, or paused or
	// stopped by anyone.
	Run      Run
	Activity Activity
	// QueuePosition is its place in the client's queue, from 1, 0 for none.
	QueuePosition int
	// StoppedAtLimit is a torrent the client stopped at its seeding goal.
	StoppedAtLimit bool
	// Finished is the kind's completion signal with no move in progress.
	Finished bool
	// FinishedAt is the client's own completion time, zero when it gives
	// none.
	FinishedAt time.Time
	// Folder is the torrent's save folder as the client sees it.
	Folder string
	// Label is the label the torrent has now, "" for none.
	Label    string
	Counters Counters
	// LocalError is the client's own words for a local error, "" for none.
	LocalError string
	Trackers   []TrackerMessage
	Rates      Rates
}

type Run uint8

const (
	Started Run = iota + 1
	Paused
)

type Activity uint8

const (
	Checking Activity = iota + 1
	Downloading
	Seeding
	Queued
	Moving
)

// Counters are the client's raw all-time counters. They can drop, as when
// a torrent is added again in the client.
type Counters struct {
	Uploaded, Downloaded int64
	Active, Seeding      time.Duration
}

type TrackerMessage struct {
	Message string
	// Error is a tracker error, false for a warning. A warning never
	// blocks completion.
	Error bool
}

type Rates struct {
	Down, Up     int64
	Peers, Seeds int
}

// File is one of a torrent's files. Path is relative to Status.Folder,
// with slashes, and includes the torrent's own folder for a multi-file
// torrent, as the built-in client's engine.File does.
type File struct {
	Index int
	Path  string
	Size  int64
	Done  int64
}
