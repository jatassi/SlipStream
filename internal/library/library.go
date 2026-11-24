// Package library composes the Library's reads and actions over the
// owners of Works, Targets, roots and files, each in one snapshot or one
// Write across them.
package library

import (
	"context"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/searches"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/subtitles"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

type Config struct {
	Library *storage.DB[storage.Library]
	Works   *works.Works
	Targets *targets.Targets
	Roots   *roots.Roots
	Files   *files.Files
	Imports Imports
	Moves   Moves
	Leaver  Leaver
	// Queue takes the searches an Admin's add and monitoring ask for, in
	// their Write.
	Queue searches.Queue
	// History records Works added and Works deleted.
	History entry.Recorder
	// Subtitles marks each row whose Work wants a subtitle nothing meets.
	Subtitles Subtitles
}

// Subtitles is the mark of every Work with an unmet subtitle want.
type Subtitles interface {
	LibraryRows(ctx context.Context, r storage.Reader[storage.Library]) (map[works.WorkID]subtitles.LibraryMark, error)
}

// NoMarks is the Subtitles of a library nothing subtitles.
type NoMarks struct{}

func (NoMarks) LibraryRows(context.Context, storage.Reader[storage.Library]) (map[works.WorkID]subtitles.LibraryMark, error) {
	return nil, nil
}

// Leaver hears of the Works a Leave files delete removes, inside the
// delete's Write while their file rows still stand, so it can keep a
// record of the files left on disk.
type Leaver interface {
	Leave(ctx context.Context, tx *storage.Tx[storage.Library], ws []works.WorkID) error
}

// Imports ends the unfinished Imports of a deleted Work: in the delete's
// Write, before the cascade, then a touch once it commits.
type Imports interface {
	EndOf(ctx context.Context, tx *storage.Tx[storage.Library], ws []works.WorkID) error
	Touch()
}

// Moves cancels the open Moves of a deleted Work before their switch: in
// the delete's Write, then a poke once it commits. Of is a Work's Move as
// its page shows it, nil while none is in flight or failed. NoMoves when
// nil.
type Moves interface {
	CancelOf(ctx context.Context, tx *storage.Tx[storage.Library], ws []works.WorkID) error
	Poke()
	Of(ctx context.Context, r storage.Reader[storage.Library], w works.WorkID) (*Moving, error)
}

// Moving is a Work's Move as its page shows it.
type Moving struct {
	Unit, Batch storage.UUID
	Phase       string
	// Waiting is empty while it waits for nothing.
	Waiting    string
	ToNode     role.NodeID
	ToNodeName string
	// Words is its sentence, such as "Copying to den".
	Words string
}

// NoMoves is the Moves of a library nothing moves.
type NoMoves struct{}

func (NoMoves) CancelOf(context.Context, *storage.Tx[storage.Library], []works.WorkID) error {
	return nil
}
func (NoMoves) Poke() {}

func (NoMoves) Of(context.Context, storage.Reader[storage.Library], works.WorkID) (*Moving, error) {
	return nil, nil
}

type Library struct {
	cfg Config
}

func New(cfg Config) *Library {
	if cfg.Moves == nil {
		cfg.Moves = NoMoves{}
	}
	return &Library{cfg: cfg}
}
