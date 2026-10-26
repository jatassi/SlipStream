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

// Timings are protocol internals, config only.
type Timings struct {
	// Chunk bounds the wait for each frame from the source, and each
	// write to a lane.
	Chunk time.Duration `yaml:"chunk"`
	// Report is the most often a Node sends facts while bytes move.
	Report time.Duration `yaml:"report"`
	// Dial is how long a route this end dials may fail before it gives
	// the route up.
	Dial time.Duration `yaml:"dial"`
	// Estimate is the window of a receiver's speed average.
	Estimate time.Duration `yaml:"estimate"`
	// LanesStart is the lanes a route opens first. Each LanesGrow the
	// receiver adds one while the last one added raised the rate by at
	// least LanesGain percent, up to LanesMax.
	LanesStart int           `yaml:"lanes_start"`
	LanesGrow  time.Duration `yaml:"lanes_grow"`
	LanesGain  int           `yaml:"lanes_gain"`
	LanesMax   int           `yaml:"lanes_max"`
	// Ahead is the chunks each lane keeps asked for ahead.
	Ahead int `yaml:"ahead"`
	// SyncBytes and SyncEvery bound a receiver's batch: it syncs, reads
	// back and records its chunks at whichever comes first.
	SyncBytes int64         `yaml:"sync_bytes"`
	SyncEvery time.Duration `yaml:"sync_every"`
	// RetryMin and RetryMax bound the backoff of a chunk, a dial or a
	// source that isn't ready.
	RetryMin time.Duration `yaml:"retry_min"`
	RetryMax time.Duration `yaml:"retry_max"`
	// Damaged is how many failed checks of one chunk fail the Transfer.
	Damaged int `yaml:"damaged"`
	// Piece is the largest write the send limit paces at once, in bytes.
	Piece int64 `yaml:"piece"`
}

func DefaultTimings() Timings {
	return Timings{
		Chunk:      30 * time.Second,
		Report:     time.Second,
		Dial:       30 * time.Second,
		Estimate:   20 * time.Second,
		LanesStart: 4,
		LanesGrow:  10 * time.Second,
		LanesGain:  10,
		LanesMax:   16,
		Ahead:      2,
		SyncBytes:  64 << 20,
		SyncEvery:  time.Second,
		RetryMin:   time.Second,
		RetryMax:   time.Minute,
		Damaged:    4,
		Piece:      64 << 10,
	}
}

// Payloads finds a torrent's file on this Node.
type Payloads interface {
	Path(hash bencode.InfoHash, index int) (string, error)
}

// Hooks are the simulation tier's view into a Source and a Node.
// Production leaves them zero.
type Hooks struct {
	// Read wraps the file each chunk is read from, so a test can change
	// the bytes or fail a read.
	Read func(io.ReaderAt) io.ReaderAt
	// Wanted sees each Want the Source answers.
	Wanted func(hash bencode.InfoHash, index int, chunks []int)
	// Sent sees each chunk a Node sends after it hashed it, and may change
	// it.
	Sent func(transfer ID, position, index int, data []byte)
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
