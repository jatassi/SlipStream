package storing

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/storage"
)

// Disk is every filesystem call the Storage holder makes. Each OS has its
// own; the simulation tier wraps it to play a full disk, a missing
// mount, a hung share or a read-only folder on cue. Every method may block;
// the Store bounds those that answer questions (stat, listing, volume) with
// Timings.Stat.
type Disk interface {
	// Stat follows links and junctions.
	Stat(ctx context.Context, path string) (Info, error)
	ReadDir(ctx context.Context, dir string) ([]Entry, error)
	Mkdir(ctx context.Context, path string) error
	Open(ctx context.Context, path string) (io.ReadCloser, error)
	Create(ctx context.Context, path string) (File, error)
	Remove(ctx context.Context, path string) error
	Rename(ctx context.Context, from, to string) error
	Link(ctx context.Context, from, to string) error
	// Writable writes nothing: access(W_OK) and the volume's read-only flag.
	Writable(ctx context.Context, dir string) (bool, error)
	Volume(ctx context.Context, path string) (Volume, error)
	Places(ctx context.Context) ([]Place, error)
	Host() Host
}

// Info is a stat result.
type Info interface {
	Dir() bool
	Size() int64
	// Same compares file identity, as os.SameFile does.
	Same(o Info) bool
}

type File interface {
	io.Writer
	Sync() error
	io.Closer
}

type Entry struct {
	Name string
	// Dir is a folder, or a link or junction to one.
	Dir bool
	// Hidden is the hidden rule's verdict.
	Hidden   bool
	Readable bool
}

type Host struct {
	Style PathStyle
	// Account is the account SlipStream runs as.
	Account string
	// Service is a Windows service, which sees no mapped drive letters.
	Service bool
	// Data and Downloads are folders no root may overlap.
	Data      string
	Downloads string
}

type Volume struct {
	// ID is the fsid on Unix, or the volume serial on Windows.
	ID string
	// Name reads "D: (Media)", "/mnt/media", "/Volumes/Archive" or "\\nas\media".
	Name  string
	Mount string
	Kind  Kind
	Total int64
	// Free is what SlipStream may use.
	Free      int64
	SampledAt time.Time
}

// Place is where browsing starts: a root this Node holds, a drive, "/" or
// a real mount.
type Place struct {
	Name, Path  string
	Kind        Kind
	Free, Total int64
	// Root is the root the place is, zero for any other place.
	Root storage.UUID
}

// Segment is one breadcrumb as the Node split a path; a drive or a share
// is one segment.
type Segment struct{ Name, Path string }

type Folder struct {
	Name, Path string
	Readable   bool
}

// Listing is a Browse answer: a folder, or the Node's places for an empty
// path.
type Listing struct {
	Path, Parent string
	Segments     []Segment
	Volume       *Volume
	// Folders are in natural, case-insensitive order.
	Folders []Folder
	Hidden  int
	Places  []Place
	// Service: SlipStream runs as a Windows service.
	Service bool
}

// FolderError is a folder the Node could not read or use, never folded
// into an empty answer.
type FolderError struct {
	Kind FolderErrorKind
	Text string
	// Account is the account SlipStream runs as, for FolderDenied.
	Account string
}

func (e *FolderError) Error() string { return fmt.Sprintf("folder %s: %s", e.Kind, e.Text) }

// NotReady is a root that may not take a write now.
type NotReady struct {
	// State is the marker's verdict at the instant of the write.
	State RootState
	// Draining: the Node drains Storage and takes no new imports.
	Draining bool
	// Unknown: the root isn't in its Node's intent yet.
	Unknown bool
}

func (e *NotReady) Error() string {
	switch {
	case e.Unknown:
		return "the root isn't in its Node's intent yet"
	case e.Draining:
		return "the root's Node drains Storage"
	}
	return "the root is " + e.State.String()
}

// Payload is a finished download's file on the root's Node.
type Payload struct {
	InfoHash  bencode.InfoHash
	FileIndex int
}

// Recorded is a file the library records in a root, for Verify.
type Recorded struct {
	Path Rel
	Size int64
}

// MissingFile is a recorded file Verify didn't find as recorded: an index
// into its files.
type MissingFile struct {
	Index  int
	Reason MissingReason
}

// Other is a root a folder may not overlap.
type Other struct {
	ID   storage.UUID
	Path string
}

// CheckResult is Check's verdict on a folder. Refusal is nil when the
// folder is usable.
type CheckResult struct {
	Path    string
	Volume  *Volume
	Refusal *Refusal
	// Marker is the root id the folder's marker names, zero for none.
	Marker storage.UUID
}

// Refusal is why a folder can't be a root, and the root it overlaps.
type Refusal struct {
	Reason Reason
	Root   storage.UUID
}
