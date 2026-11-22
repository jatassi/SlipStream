package archive_test

import (
	"archive/tar"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"filippo.io/age"
	"github.com/klauspost/compress/zstd"

	"github.com/jatassi/slipstream/internal/backups/archive"
)

var (
	params = archive.Params{ScryptLog: 10, ZstdWindow: 8 << 20}
	limits = archive.Limits{Library: 64 << 20, ZstdWindow: 8 << 20, ZstdMemory: 64 << 20}
)

const words = "abandon ability able about above absent absorb abstract absurd abuse"

func pass(t *testing.T) archive.Passphrase {
	t.Helper()
	p, err := archive.ParsePassphrase(words)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

type fixture struct {
	m                                   archive.Manifest
	library, libraryKey, backupKey, idx []byte
}

func small() fixture {
	return fixture{
		m:          manifest(),
		library:    []byte("SQLite format 3\x00 a library"),
		libraryKey: []byte("000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f\n"),
		backupKey:  []byte(`{"format":1}`),
		idx:        []byte(`{"backups":[]}`),
	}
}

// large spreads the archive over many age chunks, with backups.json big
// enough that library.db ends well before the last chunk.
func large() fixture {
	f := small()
	f.library = random(256 << 10)
	f.idx = random(512 << 10)
	return f
}

func random(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func manifest() archive.Manifest {
	return archive.Manifest{
		Format:        archive.Format,
		Version:       "2.1.0",
		Schemas:       map[string]int64{"secret": 1, "nodes": 4},
		BackupID:      archive.ID{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10},
		TakenAt:       time.Date(2026, 11, 14, 4, 0, 12, 0, time.UTC),
		Trigger:       archive.Scheduled,
		ControllerKey: bytes.Repeat([]byte{7}, 32),
		Generation:    1,
	}
}

func (f fixture) write(t *testing.T, p archive.Params) ([]byte, archive.Sums) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "snapshot.db")
	if err := os.WriteFile(path, f.library, 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	sums, err := archive.Write(&out, f.m, archive.Sources{LibraryDB: path, LibraryKey: f.libraryKey, BackupKey: f.backupKey, Index: f.idx}, pass(t), p)
	if err != nil {
		t.Fatal(err)
	}
	return out.Bytes(), sums
}

func (f fixture) body(e archive.Entry) []byte {
	return [][]byte{nil, f.library, f.libraryKey, f.backupKey, f.idx}[e]
}

type tarFile struct {
	name string
	body []byte
	typ  byte
}

func (f fixture) files(t *testing.T) []tarFile {
	t.Helper()
	m, err := json.Marshal(f.m)
	if err != nil {
		t.Fatal(err)
	}
	return []tarFile{
		{name: "manifest.json", body: m},
		{name: "library.db", body: f.library},
		{name: "library.key", body: f.libraryKey},
		{name: "backup.key", body: f.backupKey},
		{name: "backups.json", body: f.idx},
	}
}

// seal packs files with the stock tar, zstd and age writers, so a test
// can cut any archive the format forbids.
func seal(t *testing.T, files []tarFile) []byte {
	t.Helper()
	var out bytes.Buffer
	r, err := age.NewScryptRecipient(words)
	if err != nil {
		t.Fatal(err)
	}
	r.SetWorkFactor(params.ScryptLog)
	aw, err := age.Encrypt(&out, r)
	if err != nil {
		t.Fatal(err)
	}
	zw, err := zstd.NewWriter(aw)
	if err != nil {
		t.Fatal(err)
	}
	tw := tar.NewWriter(zw)
	for _, f := range files {
		if f.typ != 0 {
			if err := tw.WriteHeader(&tar.Header{Name: f.name, Typeflag: f.typ, Mode: 0o600, Linkname: "elsewhere"}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Typeflag: tar.TypeReg, Mode: 0o600, Size: int64(len(f.body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(f.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := errors.Join(tw.Close(), zw.Close(), aw.Close()); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// recorder is a Sink that keeps what it was given.
type recorder struct {
	mu        sync.Mutex
	got       map[archive.Entry]*bytes.Buffer
	closed    map[archive.Entry]bool
	discarded bool
	onClose   func(archive.Entry)
}

func newRecorder() *recorder {
	return &recorder{got: map[archive.Entry]*bytes.Buffer{}, closed: map[archive.Entry]bool{}}
}

func (r *recorder) Create(e archive.Entry) (io.WriteCloser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got[e] = &bytes.Buffer{}
	return &recorded{r: r, e: e}, nil
}

func (r *recorder) size(e archive.Entry) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.got[e].Len()
}

func (r *recorder) Discard() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.discarded = true
	return nil
}

type recorded struct {
	r *recorder
	e archive.Entry
}

func (w *recorded) Write(p []byte) (int, error) {
	w.r.mu.Lock()
	defer w.r.mu.Unlock()
	return w.r.got[w.e].Write(p) //nolint:wrapcheck // a test buffer
}

func (w *recorded) Close() error {
	w.r.mu.Lock()
	w.r.closed[w.e] = true
	onClose := w.r.onClose
	w.r.mu.Unlock()
	if onClose != nil {
		onClose(w.e)
	}
	return nil
}

func read(t *testing.T, b []byte) (archive.Opened, *recorder, error) {
	t.Helper()
	rec := newRecorder()
	o, err := archive.Read(bytes.NewReader(b), pass(t), limits, rec)
	return o, rec, err
}

// refused checks that Read gave nothing to act on: no Opened, and the
// Sink told to discard what it wrote.
func refused(t *testing.T, o archive.Opened, rec *recorder) {
	t.Helper()
	if o.Manifest().Format != 0 || o.Sums() != (archive.Sums{}) {
		t.Fatalf("a refused Read returned %+v", o.Manifest())
	}
	if !rec.discarded {
		t.Fatal("a refused Read left the Sink's entries in place")
	}
}

func damaged(t *testing.T, err error, reasons ...archive.DamageReason) {
	t.Helper()
	d, ok := errors.AsType[*archive.DamagedError](err)
	if !ok {
		t.Fatalf("err = %v, want *archive.DamagedError", err)
	}
	for _, r := range reasons {
		if d.Reason == r {
			return
		}
	}
	t.Fatalf("damage reason = %q (%v), want one of %q", d.Reason, d.Err, reasons)
}
