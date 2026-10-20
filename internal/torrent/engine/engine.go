// Package engine defines the torrent engine interface. The libtorrent
// package implements it over libtorrent-rasterbar.
package engine

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
	"time"
)

// Engine is a thin port over a torrent engine. Every policy (when to start,
// pause, announce or check) is the caller's; an Engine only does what it is
// told and reports what happened.
type Engine interface {
	// Add adds a torrent paused: nothing connects or announces until Resume.
	// It takes Resume when the caller has resume data, else Metainfo, and sets
	// each counter to the larger of the resume data's value and Baseline's.
	Add(AddParams) (Handle, error)
	// Pause stops the torrent gracefully. A tracker gets stopped only where it
	// answered started, so until each started announce is answered the pause
	// waits, as Remove does.
	Pause(Handle) error
	Resume(Handle) error
	// Reannounce schedules the next announce to tracker, an index into the
	// torrent's tracker list, delay from now. Without ignoreMinInterval the
	// announce waits for the tracker's MinAnnounce. Either way MinAnnounce
	// moves to the scheduled time. A paused torrent, or a tracker it never
	// announced to, is left as it is.
	Reannounce(h Handle, tracker int, delay time.Duration, ignoreMinInterval bool) error
	// Recheck hashes the files of a paused torrent, connecting to no peer
	// and announcing to no tracker. The torrent runs for the check, so
	// Resumed may come first, and pauses once checked: Checked, then Paused.
	Recheck(Handle) error
	// SaveResume asks for resume data: ResumeSaved or ResumeFailed follows.
	SaveResume(Handle) error
	// PostTrackers sends TrackerList for h.
	PostTrackers(Handle) error
	// PostUpdates sends Updates with the torrents whose status changed since
	// the last call, or with every torrent once the engine sent Dropped.
	PostUpdates() error
	// PostStats sends Stats.
	PostStats() error
	// Configure applies l live.
	Configure(l Limits) error
	ConnectPeer(Handle, netip.AddrPort) error
	// Remove leaves once no tracker awaits its answer to started, so stopped
	// goes out wherever started was answered. deleteFiles deletes the
	// torrent's files and then its save path when that is left empty, and
	// FilesDeleted follows Removed.
	Remove(h Handle, deleteFiles bool) error
	// Files lists a torrent's files as they lie under its save path, after
	// the engine sanitized the names the metainfo gives.
	Files(Handle) ([]File, error)
	// FileProgress is the bytes done of each file, in Files' order.
	FileProgress(Handle) ([]int64, error)
	// ListenAddr is ErrNotListening while the engine has no listener: a
	// listen interface naming a port binds that port or none, and the
	// engine sends ListenFailed.
	ListenAddr() (netip.AddrPort, error)
	// Version names the engine and the peer ID prefix and user agent it
	// announces with, which are its own: SlipStream never sets them.
	Version() Version
	// Events is closed by Close. A reader that falls behind gets Dropped,
	// and the next PostUpdates carries every torrent.
	Events() <-chan Event
	Close() error
}

type Config struct {
	ListenInterface netip.AddrPort
	Discovery       Discovery
	// OutgoingUTP lets the engine dial peers over uTP as well as TCP.
	OutgoingUTP bool
	// AnnounceFloor is the least time between two regular announces to a
	// tracker, whatever interval it answers: libtorrent's
	// min_announce_interval. Zero is no floor.
	AnnounceFloor time.Duration
	// Rand draws the ports an engine tries when ListenInterface's port is 0.
	Rand *rand.Rand
}

// Limits is the engine's live configuration, which Configure applies.
type Limits struct {
	// DownloadLimit and UploadLimit are payload bytes a second, 0 for none.
	DownloadLimit, UploadLimit int64
	// Connections is the most peer connections, 0 for as many as the
	// process may open files.
	Connections int
	// UTP lets peers connect over uTP as well as TCP, both ways.
	UTP bool
	// WatchIncoming sends IncomingPeer for each incoming connection.
	WatchIncoming bool
}

// Version is an engine's version and the identity it announces with.
type Version struct {
	// Libtorrent is major.minor.tiny.
	Libtorrent string
	// PeerIDPrefix starts every peer ID: -LT, the version, 0 and -.
	PeerIDPrefix string
	// UserAgent is libtorrent/ and the version with a fourth number.
	UserAgent string
}

// Settings is the torrent section of a Node's configuration.
type Settings struct {
	Listen netip.AddrPort `yaml:"listen"`
	// Downloads is the folder torrents download into. Its consumer resolves
	// a relative path against the data dir.
	Downloads string `yaml:"downloads"`
	// OutgoingUTP dials peers over uTP as well as TCP. Off, the engine
	// dials TCP only and still accepts uTP.
	OutgoingUTP bool `yaml:"outgoing_utp"`
	// LoopbackPeers has a peer on loopback count as one on the internet,
	// for a world whose every peer is on loopback.
	LoopbackPeers bool `yaml:"loopback_peers"`
}

func DefaultSettings() Settings {
	return Settings{Listen: netip.MustParseAddrPort("0.0.0.0:6881"), Downloads: "downloads", OutgoingUTP: true}
}

// Discovery selects the peer discovery services. Its zero value turns
// all four off.
type Discovery struct {
	DHT, LSD, UPnP, NATPMP bool
}

type AddParams struct {
	// Metainfo is nil when Resume carries the info dictionary.
	Metainfo []byte
	// Resume is resume data from ResumeSaved, nil for none.
	Resume   []byte
	SavePath string
	// SeedMode trusts the data under SavePath as complete without hashing it.
	SeedMode bool
	Baseline Totals
}

// Totals are a torrent's all-time counters.
type Totals struct {
	Uploaded, Downloaded int64
	Active, Seeding      time.Duration
}

type Handle int64

// File is one file of a torrent: its index in the metainfo, its path under
// the save path with / separators, and its size.
type File struct {
	Index int
	Path  string
	Size  int64
}

type State uint8

const (
	StateCheckingFiles State = iota + 1
	StateDownloadingMetadata
	StateDownloading
	StateFinished
	StateSeeding
	StateCheckingResumeData
)

var stateNames = [...]string{
	StateCheckingFiles:       "checking files",
	StateDownloadingMetadata: "downloading metadata",
	StateDownloading:         "downloading",
	StateFinished:            "finished",
	StateSeeding:             "seeding",
	StateCheckingResumeData:  "checking resume data",
}

func (s State) String() string { return stateNames[s] }

type Status struct {
	Torrent Handle
	State   State
	Paused  bool
	Done    int64
	Wanted  int64
	// DownBps and UpBps are payload rates in bytes a second.
	DownBps, UpBps int64
	Peers, Seeds   int
	// Totals include the baseline the torrent was added with.
	Totals Totals
	// LastPayload is when the torrent last downloaded payload, zero for never.
	LastPayload time.Time
	// NeedSave is whether the torrent changed since its resume data was saved.
	NeedSave bool
	// Err is the torrent's error, nil for none.
	Err *Error
}

type AnnounceEvent uint8

const (
	// AnnounceNone is a regular announce.
	AnnounceNone AnnounceEvent = iota
	AnnounceCompleted
	AnnounceStarted
	AnnounceStopped
)

var announceEventNames = [...]string{
	AnnounceNone:      "none",
	AnnounceCompleted: "completed",
	AnnounceStarted:   "started",
	AnnounceStopped:   "stopped",
}

func (a AnnounceEvent) String() string { return announceEventNames[a] }

// Tracker is one entry of a torrent's tracker list. The torrent announces
// to it from each of its Endpoints, its local listen addresses, and the
// announce state is that of the first.
type Tracker struct {
	Index     int
	URL       string
	Tier      int
	Endpoints int
	// NextAnnounce is when the next regular announce goes out, and
	// MinAnnounce the earliest one may. Zero for never announced.
	NextAnnounce, MinAnnounce time.Time
	// Message is the tracker's last warning or failure reason.
	Message string
	// LastError is the last announce's error, nil once one succeeds.
	LastError *Error
	// Fails is the announces that failed in a row.
	Fails int
	// Updating is an announce awaiting its answer.
	Updating bool
	// StartSent is whether the tracker answered started.
	StartSent bool
}

type Event interface{ event() }

type (
	Added        struct{ Torrent Handle }
	StateChanged struct {
		Torrent Handle
		State   State
	}
	Finished      struct{ Torrent Handle }
	Removed       struct{ Torrent Handle }
	TorrentFailed struct {
		Torrent Handle
		Err     *Error
	}
	// TrackerAnnounced reports each announce sent to a tracker, an index
	// into the torrent's tracker list, and TrackerReplied or TrackerFailed
	// its answer.
	TrackerAnnounced struct {
		Torrent Handle
		Tracker int
		Event   AnnounceEvent
		At      time.Time
	}
	TrackerReplied struct {
		Torrent Handle
		Tracker int
		At      time.Time
	}
	// TrackerWarned precedes the TrackerReplied of an answer that carried a
	// warning.
	TrackerWarned struct {
		Torrent Handle
		Tracker int
		Message string
	}
	TrackerFailed struct {
		Torrent Handle
		Tracker int
		// Times is the announces that failed in a row, this one included.
		Times int
		// Reason is the failure reason the tracker answered, empty when it
		// never answered.
		Reason string
		Err    *Error
		At     time.Time
	}
	TrackerList struct {
		Torrent  Handle
		Trackers []Tracker
	}
	ListenFailed struct{ Err *Error }
	Dropped      struct{}
	Paused       struct{ Torrent Handle }
	Resumed      struct{ Torrent Handle }
	ResumeSaved  struct {
		Torrent Handle
		Data    []byte
	}
	ResumeFailed struct {
		Torrent Handle
		Err     *Error
	}
	// FilesDeleted follows Removed when the removal deleted files. Err is
	// nil when every file and folder went.
	FilesDeleted struct {
		Torrent Handle
		Err     *Error
	}
	Updates struct{ Statuses []Status }
	// Checked ends a Recheck.
	Checked struct{ Torrent Handle }
	// IncomingPeer is a peer that connected to the listener, sent while
	// Limits.WatchIncoming is set.
	IncomingPeer struct{ Addr netip.Addr }
	// FileError is a failed read or write of a torrent's files. A write
	// that finds the disk full, whose Err matches ErrDiskFull, pauses the
	// torrent.
	FileError struct {
		Torrent Handle
		Err     *Error
	}
	// HashFailed is a piece whose data failed its hash check: a peer sent
	// data that is not the torrent's.
	HashFailed struct{ Torrent Handle }
	// Stats is the payload the engine sent and received since it opened.
	Stats struct {
		SentPayload, RecvPayload int64
		At                       time.Time
	}
)

func (Added) event()            {}
func (StateChanged) event()     {}
func (Finished) event()         {}
func (Removed) event()          {}
func (TorrentFailed) event()    {}
func (TrackerAnnounced) event() {}
func (TrackerReplied) event()   {}
func (TrackerWarned) event()    {}
func (TrackerFailed) event()    {}
func (TrackerList) event()      {}
func (ListenFailed) event()     {}
func (Dropped) event()          {}
func (Paused) event()           {}
func (Resumed) event()          {}
func (ResumeSaved) event()      {}
func (ResumeFailed) event()     {}
func (FilesDeleted) event()     {}
func (Updates) event()          {}
func (Checked) event()          {}
func (IncomingPeer) event()     {}
func (FileError) event()        {}
func (HashFailed) event()       {}
func (Stats) event()            {}

// Category implements error so errors.Is(err, CategoryBdecode) matches any
// *Error in that category.
type Category uint8

const (
	CategoryShim Category = iota + 1
	CategorySystem
	CategoryLibtorrent
	CategoryBdecode
	CategoryOther
	CategoryException
)

var categoryNames = [...]string{
	CategoryShim:       "shim",
	CategorySystem:     "system",
	CategoryLibtorrent: "libtorrent",
	CategoryBdecode:    "bdecode",
	CategoryOther:      "other",
	CategoryException:  "exception",
}

func (c Category) Error() string { return categoryNames[c] }

// Shim codes, the Code of an Error in CategoryShim.
const (
	CodeUnknownTorrent = 1
	CodeClosed         = 2
	CodeNotListening   = 3
	CodeDiskFull       = 4
)

var (
	ErrUnknownTorrent = &Error{Category: CategoryShim, Code: CodeUnknownTorrent, Text: "unknown torrent"}
	ErrClosed         = &Error{Category: CategoryShim, Code: CodeClosed, Text: "engine closed"}
	ErrNotListening   = &Error{Category: CategoryShim, Code: CodeNotListening, Text: "not listening"}
	// ErrDiskFull matches an error out of space on any OS.
	ErrDiskFull = &Error{Category: CategoryShim, Code: CodeDiskFull, Text: "disk full"}
)

// Error compares by Category and Code. Text is for logs.
type Error struct {
	Category Category
	Code     int
	Text     string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s error %d: %s", e.Category, e.Code, e.Text)
}

func (e *Error) Is(target error) bool {
	switch t := target.(type) {
	case Category:
		return e.Category == t
	case *Error:
		return e.Category == t.Category && e.Code == t.Code
	}
	return false
}
