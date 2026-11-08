package storing

import "time"

// FileID is a file's identity on its Node: its volume and its id there.
// Zero where the volume keeps no ids (NFS, SMB, CIFS, vfat, exFAT, afpfs,
// webdav, any Windows network drive). Comparable.
type FileID struct{ Volume, File string }

func (id FileID) IsZero() bool { return id.File == "" }

// Seen is what a walk or an import reads of a file besides its size.
type Seen struct {
	Modified time.Time // to the millisecond
	ID       FileID
}

// Same reports whether s and o read the same modification time and id.
func (s Seen) Same(o Seen) bool { return s.Modified.Equal(o.Modified) && s.ID == o.ID }

// Walked is one video file a walk listed.
type Walked struct {
	Path Rel
	Size int64
	Seen Seen
}

// Unread is a name a walk couldn't list: a file or folder whose name a
// Rel can't carry, or a folder it can't open.
type Unread struct {
	// Path is for people: invalid bytes read as U+FFFD.
	Path   string
	Reason UnreadReason
}

type UnreadReason uint8

const (
	UnreadNotUTF8 UnreadReason = iota + 1 // a name that isn't UTF-8
	UnreadRefused                         // a name ParseRel refuses, such as one holding a backslash
	UnreadDenied                          // a folder the Node can't list
)

// Cursor is where a walk's next page starts, opaque to the Controller.
// The zero Cursor starts the walk.
type Cursor struct{ b string }

// Page is one page of a walk. Next starts the page after it, until Done.
// CaseFold is whether the root resolves names ignoring case.
type Page struct {
	Entries  []Walked
	Unread   []Unread
	Next     Cursor
	Done     bool
	CaseFold bool
}
