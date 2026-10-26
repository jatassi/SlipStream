// Package transfers copies files between Nodes for the jobs that ask:
// the Controller records each Transfer a purpose starts and judges its
// live state, and every Node sends and receives the Transfers its intent
// makes active, chunk by verified chunk, resuming from what it verified.
package transfers

import (
	"embed"
	"io"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/db"
	"github.com/jatassi/slipstream/internal/transfers/internal/engine"
)

type file = storage.Local

//go:embed schema/*.sql
var schemaFiles embed.FS

// Schema is the transfers local set: what a receiver staged and verified.
func Schema() storage.Schema[file] { return storage.NewSchema[file]("transfers", schemaFiles) }

// ID is a Transfer's own id, never its job's.
type ID = db.TransferID

// ChunkSize is every chunk's size but a file's last.
const ChunkSize = engine.ChunkSize

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
	// RouteRetry is how long a pair starts on a higher route that moved
	// bytes before it tries the first route again.
	RouteRetry time.Duration `yaml:"route_retry"`
	// PairWrite is how often the Controller records a pair's speed.
	PairWrite time.Duration `yaml:"pair_write"`
	// Changed is how often the Controller publishes transfers.changed.
	Changed time.Duration `yaml:"changed"`
}

func DefaultTimings() Timings {
	return Timings{
		Chunk:      30 * time.Second,
		Report:     time.Second,
		Dial:       30 * time.Second,
		Estimate:   20 * time.Second,
		Ahead:      2,
		SyncBytes:  64 << 20,
		SyncEvery:  time.Second,
		RetryMin:   time.Second,
		RetryMax:   time.Minute,
		Damaged:    4,
		Piece:      64 << 10,
		RouteRetry: time.Hour,
		PairWrite:  time.Minute,
		Changed:    time.Second,
	}
}

// Hooks are the simulation tier's view into a Node's sender. Production
// leaves them zero.
type Hooks struct {
	// Read wraps the file each chunk is read from, so a test can fail a
	// read.
	Read func(io.ReaderAt) io.ReaderAt
	// Sent sees each chunk a Node sends after it hashed it, and may change
	// it.
	Sent func(transfer ID, position, index int, data []byte)
}
