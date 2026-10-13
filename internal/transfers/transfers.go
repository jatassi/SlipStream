// Package transfers copies one file of a torrent from the Node that holds
// it to the Node that receives it, over a kind-2 link stream. The receiver
// drives: it asks for every chunk it hasn't verified, checks each against
// the source's BLAKE3 hash, and records it in local.db, so a broken stream
// resumes from the last verified chunk.
package transfers

import (
	"embed"
	"errors"
	"io"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/db"
)

type file = storage.Local

//go:embed schema/*.sql
var schemaFiles embed.FS

func Schema() storage.Schema[file] { return storage.NewSchema[file]("transfers", schemaFiles) }

// ID is a Transfer's: the Grab's it serves, one Transfer per Grab.
type ID = db.TransferID

const ChunkSize = 4 << 20

var (
	// ErrBroken is a stream that ended or stalled mid-Transfer. Receive
	// again resumes from the verified chunks.
	ErrBroken = errors.New("transfers: the stream broke")
	// ErrNotPresent is a source that doesn't hold the file yet.
	ErrNotPresent = errors.New("transfers: the source doesn't hold the file")
)

// FailedError is a Transfer that retrying can't finish.
type FailedError struct{ Reason string }

func (e *FailedError) Error() string { return "transfers: " + e.Reason }

type Timings struct {
	// Chunk bounds the wait for each frame from the source.
	Chunk time.Duration `yaml:"chunk"`
}

func DefaultTimings() Timings { return Timings{Chunk: 30 * time.Second} }

// Payloads finds a torrent's file on this Node.
type Payloads interface {
	Path(hash bencode.InfoHash, index int) (string, error)
}

// Hooks are the simulation tier's view into a Source. Production leaves
// them zero.
type Hooks struct {
	// Read wraps the file each chunk is read from, so a test can change
	// the bytes or fail a read.
	Read func(io.ReaderAt) io.ReaderAt
	// Wanted sees each Want the Source answers.
	Wanted func(hash bencode.InfoHash, index int, chunks []int)
}

// Hash is a chunk's BLAKE3-256.
type Hash [32]byte

// Manifest is a file's size and the hash of each ChunkSize chunk, the
// last one short.
type Manifest struct {
	Size   int64
	Hashes []Hash
}

func chunkCount(size int64) int { return int((size + ChunkSize - 1) / ChunkSize) }

func (m Manifest) offset(i int) int64 { return int64(i) * ChunkSize }

func (m Manifest) chunkLen(i int) int64 { return min(ChunkSize, m.Size-m.offset(i)) }

func (m Manifest) equal(o Manifest) bool { return m.Size == o.Size && slices.Equal(m.Hashes, o.Hashes) }

// plan is the chunks to ask for: every one not verified, in order.
func plan(chunks int, verified map[int]bool) []int {
	var want []int
	for i := range chunks {
		if !verified[i] {
			want = append(want, i)
		}
	}
	return want
}
