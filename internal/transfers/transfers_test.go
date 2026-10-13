package transfers_test

import (
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/zeebo/blake3"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/link/linktest"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/storage/storagetest"
	"github.com/jatassi/slipstream/internal/transfers"
)

// size is five chunks, the last one short.
const size = 4*transfers.ChunkSize + 1000

var (
	held  = bencode.InfoHash{1}
	every = []int{0, 1, 2, 3, 4}
)

// rig is a Controller receiving from a Worker over a real link, with the
// Worker's Source reading one held payload.
type rig struct {
	t        *testing.T
	pair     *linktest.Pair
	receiver *transfers.Receiver
	payload  string
	job      transfers.Job

	mu     sync.Mutex
	wanted [][]int
	seen   []progress
	// fault sees each chunk the Source reads and may change it or fail
	// the read.
	fault func(chunk int, b []byte) error
}

type progress struct {
	verified, size int64
	ok             bool
}

type side struct {
	version string
	// serves picks the peer the Worker's Source serves.
	serves func(*linktest.Pair) link.Key
}

func newRig(t *testing.T, s side) *rig {
	t.Helper()
	if s.serves == nil {
		s.serves = func(p *linktest.Pair) link.Key { return p.Controller.Key }
	}
	r := &rig{t: t}
	var source *transfers.Source
	r.pair = linktest.New(t, linktest.Side{}, linktest.Side{Version: s.version, Streams: map[link.Kind]func(link.Key, link.Stream){
		link.KindTransfer: func(peer link.Key, st link.Stream) { source.Serve(peer, st) },
	}})
	dir := t.TempDir()
	r.payload = filepath.Join(dir, "seedbox", "The Quiet Orchard.mkv")
	r.write(size, 1)
	source = transfers.NewSource(transfers.SourceConfig{
		Payloads:   payloads{held: r.payload},
		Controller: s.serves(r.pair),
		Hooks:      transfers.Hooks{Read: r.read, Wanted: r.want},
		Log:        slog.New(slog.DiscardHandler),
	})
	seedbox := role.NodeID{UUID: storage.NewUUID()}
	r.receiver = transfers.NewReceiver(transfers.Config{
		Local:   storagetest.Local(t, transfers.Schema()),
		Link:    r.pair.Controller.Link,
		Keys:    keys{seedbox: r.pair.Worker.Key},
		Timings: transfers.DefaultTimings(),
	})
	r.job = transfers.Job{
		ID:       transfers.ID{UUID: storage.NewUUID()},
		Source:   seedbox,
		InfoHash: held,
		Staging:  filepath.Join(dir, "movies", ".slipstream-staging", "grab.part"),
	}
	return r
}

// write fills the payload with n bytes drawn from seed.
func (r *rig) write(n int, seed byte) {
	b := make([]byte, n)
	_, _ = rand.NewChaCha8([32]byte{seed}).Read(b)
	if err := os.MkdirAll(filepath.Dir(r.payload), 0o750); err != nil {
		r.t.Fatal(err)
	}
	if err := os.WriteFile(r.payload, b, 0o600); err != nil {
		r.t.Fatal(err)
	}
}

func (r *rig) receive() error {
	err := r.receiver.Receive(r.t.Context(), r.job)
	synctest.Wait()
	return err
}

func (r *rig) read(in io.ReaderAt) io.ReaderAt { return faulty{in, r} }

type faulty struct {
	io.ReaderAt
	r *rig
}

func (f faulty) ReadAt(b []byte, off int64) (int, error) {
	n, err := f.ReaderAt.ReadAt(b, off)
	if err == nil && f.r.fault != nil {
		err = f.r.fault(int(off/transfers.ChunkSize), b)
	}
	return n, err
}

func (r *rig) want(hash bencode.InfoHash, index int, chunks []int) {
	if hash != held || index != 0 {
		r.t.Errorf("the Source answered a Want for %s file %d", hash, index)
	}
	verified, size, ok := r.receiver.Progress(r.job.ID)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.wanted = append(r.wanted, chunks)
	r.seen = append(r.seen, progress{verified, size, ok})
}

func (r *rig) expectWanted(want ...[]int) {
	r.t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if !reflect.DeepEqual(r.wanted, want) {
		r.t.Fatalf("the Source was asked for %v, want %v", r.wanted, want)
	}
}

// expectStaged checks the staged file is the payload, by BLAKE3.
func (r *rig) expectStaged() {
	r.t.Helper()
	got, want := sum(r.t, r.job.Staging), sum(r.t, r.payload)
	if got != want {
		r.t.Fatalf("the staged file's BLAKE3 is %x, want the payload's %x", got, want)
	}
}

func sum(t *testing.T, path string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return blake3.Sum256(b)
}

// breakOnce fails the Source's first read of chunk, ending the stream.
func breakOnce(chunk int) func(int, []byte) error {
	done := false
	return func(i int, _ []byte) error {
		if i != chunk || done {
			return nil
		}
		done = true
		return errors.New("the disk went away")
	}
}

// flip changes a byte of chunk in the Source's first copies of it.
func flip(chunk, copies int) func(int, []byte) error {
	return func(i int, b []byte) error {
		if i == chunk && copies > 0 {
			copies--
			b[100] ^= 0xff
		}
		return nil
	}
}

type keys map[role.NodeID]link.Key

func (k keys) Key(node role.NodeID) (link.Key, bool) {
	key, ok := k[node]
	return key, ok
}

type payloads map[bencode.InfoHash]string

func (p payloads) Path(hash bencode.InfoHash, index int) (string, error) {
	if path, ok := p[hash]; ok && index == 0 {
		return path, nil
	}
	return "", transfers.ErrNotPresent
}

func TestAFileTransfersVerified(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		if err := r.receive(); err != nil {
			t.Fatal(err)
		}
		r.expectStaged()
		if err := r.receive(); err != nil {
			t.Fatalf("receiving a finished Transfer again = %v", err)
		}
		r.expectWanted(every)
		if _, _, ok := r.receiver.Progress(r.job.ID); ok {
			t.Fatal("a finished Transfer still reports progress")
		}
	})
}

func TestABrokenStreamResumesFromTheVerifiedChunks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		r.fault = breakOnce(2)
		if err := r.receive(); !errors.Is(err, transfers.ErrBroken) {
			t.Fatalf("a stream that broke at chunk 2 = %v, want ErrBroken", err)
		}
		if err := r.receive(); err != nil {
			t.Fatal(err)
		}
		r.expectStaged()
		r.expectWanted(every, []int{2, 3, 4})
		want := []progress{{0, size, true}, {2 * transfers.ChunkSize, size, true}}
		if !reflect.DeepEqual(r.seen, want) {
			t.Fatalf("the progress at each Want = %v, want %v", r.seen, want)
		}
	})
}

func TestAFlippedByteIsAskedForAgain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		r.fault = flip(1, 1)
		if err := r.receive(); err != nil {
			t.Fatal(err)
		}
		r.expectStaged()
		r.expectWanted(every, []int{1})
	})
}

func TestAChunkThatNeverVerifiesFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		r.fault = flip(1, 4)
		var failed *transfers.FailedError
		if err := r.receive(); !errors.As(err, &failed) || err.Error() != "transfers: chunk 1 failed verification 4 times" {
			t.Fatalf("four bad copies of chunk 1 = %v", err)
		}
		r.expectWanted(every, []int{1}, []int{1}, []int{1})
	})
}

func TestAChangedFileRestarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		r.fault = breakOnce(2)
		if err := r.receive(); !errors.Is(err, transfers.ErrBroken) {
			t.Fatalf("a stream that broke at chunk 2 = %v, want ErrBroken", err)
		}
		r.write(size+1000, 2)
		if err := r.receive(); err != nil {
			t.Fatal(err)
		}
		r.expectStaged()
		r.expectWanted(every, every)
	})
}

func TestAStalledChunkBreaksTheStream(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		stall := make(chan struct{})
		defer close(stall)
		r.fault = func(i int, _ []byte) error {
			if i == 2 {
				<-stall
			}
			return nil
		}
		start := time.Now()
		if err := r.receiver.Receive(t.Context(), r.job); !errors.Is(err, transfers.ErrBroken) {
			t.Fatalf("a stalled chunk = %v, want ErrBroken", err)
		}
		if waited := time.Since(start); waited != 30*time.Second {
			t.Fatalf("the receiver waited %v for the stalled chunk, want 30s", waited)
		}
	})
}

func TestASourceWithoutTheFileIsNotPresent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		r.job.InfoHash = bencode.InfoHash{2}
		if err := r.receive(); !errors.Is(err, transfers.ErrNotPresent) {
			t.Fatalf("a file the source doesn't hold = %v, want ErrNotPresent", err)
		}
	})
}

func TestATransferWithNoSessionIsNotSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		if why, ok := rpc.NotSent(r.receive()); !ok || why != rpc.Disconnected {
			t.Fatalf("a Transfer with no session = %v, %v; want it not sent as disconnected", why, ok)
		}
	})
}

func TestATransferBetweenVersionsIsNotSent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{version: "0.0.0-b"})
		r.pair.Connect(t)
		if why, ok := rpc.NotSent(r.receive()); !ok || why != rpc.Mismatched {
			t.Fatalf("a Transfer between versions = %v, %v; want it not sent as mismatched", why, ok)
		}
		r.expectWanted()
	})
}

func TestASourceServesOnlyItsController(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{serves: func(p *linktest.Pair) link.Key { return p.Worker.Key }})
		r.pair.Connect(t)
		if err := r.receive(); !errors.Is(err, transfers.ErrBroken) {
			t.Fatalf("a Transfer from a source serving another Controller = %v, want ErrBroken", err)
		}
		r.expectWanted()
	})
}

func TestForgetDropsTheStagedFile(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		if err := r.receive(); err != nil {
			t.Fatal(err)
		}
		for range 2 {
			if err := r.receiver.Forget(t.Context(), r.job.ID); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := os.Stat(r.job.Staging); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("the staged file after Forget: %v", err)
		}
		if err := r.receive(); err != nil {
			t.Fatal(err)
		}
		r.expectWanted(every, every)
	})
}

func TestRetainDropsTheTransfersNotKept(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRig(t, side{})
		r.pair.Connect(t)
		r.fault = breakOnce(2)
		if err := r.receive(); !errors.Is(err, transfers.ErrBroken) {
			t.Fatalf("a stream that broke at chunk 2 = %v, want ErrBroken", err)
		}
		if err := r.receiver.Retain(t.Context(), []transfers.ID{r.job.ID}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(r.job.Staging); err != nil {
			t.Fatalf("the kept Transfer's staged file: %v", err)
		}
		if err := r.receiver.Retain(t.Context(), nil); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(r.job.Staging); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("the dropped Transfer's staged file: %v", err)
		}
		if err := r.receive(); err != nil {
			t.Fatal(err)
		}
		r.expectWanted(every, every)
	})
}
