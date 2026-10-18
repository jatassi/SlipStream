// Package engine defines the torrent engine interface. The libtorrent
// package implements it over libtorrent-rasterbar.
package engine

import (
	"fmt"
	"math/rand/v2"
	"net/netip"
)

type Engine interface {
	Add(AddParams) (Handle, error)
	ConnectPeer(Handle, netip.AddrPort) error
	Status(Handle) (Status, error)
	Remove(Handle) error
	// Files lists a torrent's files as they lie under its save path, after
	// the engine sanitized the names the metainfo gives.
	Files(Handle) ([]File, error)
	ListenAddr() (netip.AddrPort, error)
	// Events is closed by Close. A reader that falls behind gets Dropped and
	// should re-read Status for the torrents it tracks.
	Events() <-chan Event
	Close() error
}

type Config struct {
	ListenInterface netip.AddrPort
	Discovery       Discovery
	// OutgoingUTP lets the engine dial peers over uTP as well as TCP.
	OutgoingUTP bool
	// Rand draws the ports an engine tries when ListenInterface's port is 0.
	Rand *rand.Rand
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
	Metainfo []byte
	SavePath string
	// SeedMode trusts the data under SavePath as complete without hashing it.
	SeedMode bool
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
	State  State
	Peers  int
	Done   int64
	Wanted int64
	// Uploaded is the payload bytes sent to peers since the torrent was
	// added.
	Uploaded int64
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
	// TrackerReplied and TrackerFailed report each announce to a tracker.
	TrackerReplied struct{ Torrent Handle }
	TrackerFailed  struct {
		Torrent Handle
		Err     *Error
	}
	ListenFailed struct{ Err *Error }
	Dropped      struct{}
)

func (Added) event()          {}
func (StateChanged) event()   {}
func (Finished) event()       {}
func (Removed) event()        {}
func (TorrentFailed) event()  {}
func (TrackerReplied) event() {}
func (TrackerFailed) event()  {}
func (ListenFailed) event()   {}
func (Dropped) event()        {}

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
)

var (
	ErrUnknownTorrent = &Error{Category: CategoryShim, Code: CodeUnknownTorrent, Text: "unknown torrent"}
	ErrClosed         = &Error{Category: CategoryShim, Code: CodeClosed, Text: "engine closed"}
	ErrNotListening   = &Error{Category: CategoryShim, Code: CodeNotListening, Text: "not listening"}
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
