// Package source names what a Transfer reads on its source Node, why the
// source can't be read now, and how the Controller judges it ready from
// its own rows. It is a leaf, so the holders that serve sources can return
// its types without importing transfers.
package source

import (
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
)

type Kind string

const TorrentKind Kind = "torrent"

// Ref names one file a sender reads. The only kind is Torrent, and Refs
// are comparable, so they key maps.
type Ref interface {
	Kind() Kind
	ref()
}

// Torrent is file index File of the torrent with info hash Hash, as the
// source Node's torrent client holds it.
type Torrent struct {
	Hash bencode.InfoHash
	File int
}

func (Torrent) Kind() Kind { return TorrentKind }
func (Torrent) ref()       {}

// Reason is why a sender can't read a source.
type Reason uint8

const (
	// Gone is a source the Node no longer has, and never will again.
	Gone Reason = iota + 1
	// Failed is a source its holder failed, and won't recover by itself.
	Failed
	// NotReady is a source the Node has but can't read yet.
	NotReady
)

// Absent is why a sender can't read a source now. Detail is the holder's
// own text: the engine's error for Failed, the open error or what the
// source waits on for NotReady.
type Absent struct {
	Reason Reason
	Detail string
}

func (a *Absent) Error() string {
	var what string
	switch a.Reason {
	case Gone:
		what = "the source no longer has the file"
	case Failed:
		what = "the source failed"
	case NotReady:
		what = "the source isn't ready"
	}
	if a.Detail == "" {
		return what
	}
	return what + ": " + a.Detail
}

// ReadyState is a source's readiness as the Controller's rows say it.
type ReadyState uint8

const (
	// Unknown is a source the rows say nothing of: no holder record, no
	// report yet, or removed other than by a write-off. The sender's
	// Absent is the authority then.
	Unknown ReadyState = iota
	Ready
	Checking
	// Incomplete is a source still downloading, or found missing pieces
	// by a check.
	Incomplete
	// FailedSource is a source its holder failed.
	FailedSource
	// WrittenOff is a source whose Node the Admin wrote off.
	WrittenOff
)

// Readiness is the Controller's view of a source from its own rows. Name
// is the source as people read it, and Detail the holder's error text for
// FailedSource.
type Readiness struct {
	State  ReadyState
	Name   string
	Detail string
}

// Identity is what a resume keys on: size, modification time and, where
// the volume keeps it stable, the file id. FileID is nil where it isn't.
type Identity struct {
	Size    int64
	ModTime time.Time
	FileID  []byte
}
