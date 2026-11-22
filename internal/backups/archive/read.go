package archive

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"

	"filippo.io/age"
	"github.com/klauspost/compress/zstd"
	"github.com/zeebo/blake3"
)

// maxWorkFactor is the highest scrypt work factor the age CLI accepts.
const maxWorkFactor = 22

// Sink receives the four entries after the manifest as they decode. Read
// calls Discard on any error.
type Sink interface {
	Create(e Entry) (io.WriteCloser, error)
	Discard() error
}

// Opened is proof that age reached the end of the stream with no error:
// only Read makes one with a manifest, and only after the last chunk
// authenticated, the zstd frame ended and tar's end-of-archive followed the
// fifth entry. Staging takes an Opened, so nothing can act on bytes from a
// stream that later turns out damaged.
type Opened struct {
	manifest Manifest
	sums     Sums
}

func (o Opened) Manifest() Manifest { return o.manifest }

func (o Opened) Sums() Sums { return o.sums }

var (
	ErrNotBackup       = errors.New("not a SlipStream Backup")
	ErrWrongPassphrase = errors.New("the passphrase doesn't open this Backup")
)

// TooNewError is a manifest format above Format. Version is the SlipStream
// version that took the Backup.
type TooNewError struct {
	Format  int
	Version string
}

func (e *TooNewError) Error() string {
	return fmt.Sprintf("the Backup is format %d from SlipStream %s, and this version opens format %d at most", e.Format, e.Version, Format)
}

type DamageReason int

const (
	DamageCut DamageReason = iota + 1
	DamageChunk
	DamageOrder
	DamageExtra
	DamageOversize
	DamageZstd
	DamageManifest
)

var damageWords = map[DamageReason]string{
	DamageCut:      "the file ends early",
	DamageChunk:    "a chunk doesn't authenticate",
	DamageOrder:    "an entry is out of order",
	DamageExtra:    "data follows the last entry",
	DamageOversize: "an entry is over its size limit",
	DamageZstd:     "the compressed stream doesn't decode",
	DamageManifest: "the manifest doesn't decode",
}

func (r DamageReason) String() string { return damageWords[r] }

// DamagedError is anything wrong after the age header.
type DamagedError struct {
	Reason DamageReason
	Err    error
}

func (e *DamagedError) Error() string {
	return fmt.Sprintf("the Backup is damaged because %s: %v", e.Reason, e.Err)
}

func (e *DamagedError) Unwrap() error { return e.Err }

// OversizeError is an entry over its bound. For library.db the bound is
// the free space a restore checked, so a restore reads it as room.
type OversizeError struct {
	Entry       Entry
	Size, Bound int64
}

func (e *OversizeError) Error() string {
	return fmt.Sprintf("%s is %d bytes, over its %d", e.Entry.Name(), e.Size, e.Bound)
}

// Read decodes r into sink. Its errors are exactly:
//
//	ErrNotBackup        not an age file, not scrypt, or a manifest that isn't one
//	ErrWrongPassphrase  the scrypt stanza doesn't open
//	*TooNewError        a manifest format above Format
//	*DamagedError       anything after the header: a chunk, a cut, an entry out of order, an extra entry, an entry over its bound, zstd
//
// or an error reading r or writing to sink, which it returns as it is. It
// runs one zstd thread, caps window and memory, and accepts a work factor
// up to 22.
func Read(r io.Reader, pass Passphrase, l Limits, sink Sink) (Opened, error) {
	rd := &reading{src: source{r: r, h: blake3.New()}, sink: sink, limits: l}
	o, err := rd.read(pass)
	if err != nil {
		if derr := sink.Discard(); derr != nil {
			err = errors.Join(err, fmt.Errorf("discard the entries: %w", derr))
		}
		return Opened{}, err
	}
	return o, nil
}

// Check reads r to the end and answers its Sums: the read-back step of a
// Backup, and the check of a pulled copy.
func Check(r io.Reader, pass Passphrase, l Limits) (Opened, error) {
	return Read(r, pass, l, discardSink{})
}

type discardSink struct{}

func (discardSink) Create(Entry) (io.WriteCloser, error) { return nopCloser{io.Discard}, nil }

func (discardSink) Discard() error { return nil }

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

// reading tracks the first error each layer saw, so a failure that
// surfaces in tar is blamed on the layer it started in: the source, age,
// zstd or the sink, in that order.
type reading struct {
	src     source
	age     tracked
	zstd    tracked
	written error
	sink    Sink
	limits  Limits
}

func (rd *reading) read(pass Passphrase) (Opened, error) {
	id, err := age.NewScryptIdentity(pass.words)
	if err != nil {
		return Opened{}, fmt.Errorf("open the Backup: %w", err)
	}
	id.SetMaxWorkFactor(maxWorkFactor)
	ar, err := age.Decrypt(&rd.src, id)
	if err != nil {
		return Opened{}, rd.header(err)
	}
	rd.age.r = ar
	zr, err := zstd.NewReader(&rd.age,
		zstd.WithDecoderConcurrency(1),
		zstd.WithDecoderMaxWindow(uint64(rd.limits.ZstdWindow)), //nolint:gosec // a configured size, never negative
		zstd.WithDecoderMaxMemory(uint64(rd.limits.ZstdMemory)), //nolint:gosec // a configured size, never negative
	)
	if err != nil {
		return Opened{}, fmt.Errorf("start the zstd decoder: %w", err)
	}
	defer zr.Close()
	rd.zstd.r = zr
	tr := tar.NewReader(&rd.zstd)

	var o Opened
	for e := range entries {
		hdr, err := tr.Next()
		switch {
		case errors.Is(err, io.EOF):
			return Opened{}, rd.fail(DamageCut, fmt.Errorf("the archive ends before %s", e.Name()))
		case err != nil:
			return Opened{}, rd.fail(DamageOrder, err)
		case hdr.Name != e.Name() || hdr.Typeflag != tar.TypeReg:
			return Opened{}, rd.fail(DamageOrder, fmt.Errorf("entry %d is %q where the regular file %s belongs", e+1, hdr.Name, e.Name()))
		case hdr.Size > rd.bound(e):
			return Opened{}, rd.fail(DamageOversize, &OversizeError{Entry: e, Size: hdr.Size, Bound: rd.bound(e)})
		}
		if e == ManifestEntry {
			if o.manifest, o.sums.Entries[e], err = rd.manifest(tr, hdr.Size); err != nil {
				return Opened{}, err
			}
			continue
		}
		if o.sums.Entries[e], err = rd.entry(e, tr, hdr.Size); err != nil {
			return Opened{}, err
		}
	}
	if hdr, err := tr.Next(); !errors.Is(err, io.EOF) {
		if err == nil {
			err = fmt.Errorf("entry %q follows %s", hdr.Name, IndexEntry.Name())
		}
		return Opened{}, rd.fail(DamageExtra, err)
	}
	if err := atEnd(&rd.zstd); err != nil {
		return Opened{}, rd.fail(DamageExtra, fmt.Errorf("after the tar: %w", err))
	}
	if err := atEnd(&rd.age); err != nil {
		return Opened{}, rd.fail(DamageExtra, fmt.Errorf("after the zstd frame: %w", err))
	}
	rd.src.h.Sum(o.sums.Ciphertext[:0])
	o.sums.Size = rd.src.n
	return o, nil
}

// header sorts age's header errors. A wrong passphrase is the only stanza
// error that names scrypt alone.
func (rd *reading) header(err error) error {
	if rd.src.err != nil {
		return fmt.Errorf("read the Backup: %w", rd.src.err)
	}
	if nomatch, ok := errors.AsType[*age.NoIdentityMatchError](err); ok && slices.Equal(nomatch.StanzaTypes, []string{"scrypt"}) {
		return ErrWrongPassphrase
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return &DamagedError{Reason: DamageCut, Err: err}
	}
	return fmt.Errorf("%w: %w", ErrNotBackup, err)
}

func (rd *reading) fail(reason DamageReason, err error) error {
	switch {
	case rd.written != nil:
		return fmt.Errorf("write an entry: %w", rd.written)
	case rd.src.err != nil:
		return fmt.Errorf("read the Backup: %w", rd.src.err)
	case errors.Is(rd.age.err, io.ErrUnexpectedEOF):
		return &DamagedError{Reason: DamageCut, Err: rd.age.err}
	case rd.age.err != nil:
		return &DamagedError{Reason: DamageChunk, Err: rd.age.err}
	case rd.zstd.err != nil:
		return &DamagedError{Reason: DamageZstd, Err: rd.zstd.err}
	case errors.Is(err, io.ErrUnexpectedEOF):
		return &DamagedError{Reason: DamageCut, Err: err}
	}
	return &DamagedError{Reason: reason, Err: err}
}

func (rd *reading) bound(e Entry) int64 {
	if e == LibraryEntry {
		return rd.limits.Library
	}
	return smallEntry
}

func (rd *reading) manifest(tr io.Reader, size int64) (Manifest, [32]byte, error) {
	b := make([]byte, size)
	if _, err := io.ReadFull(tr, b); err != nil {
		return Manifest{}, [32]byte{}, rd.fail(DamageCut, err)
	}
	m, err := parseManifest(b)
	return m, blake3.Sum256(b), err
}

func (rd *reading) entry(e Entry, tr io.Reader, size int64) ([32]byte, error) {
	w, err := rd.sink.Create(e)
	if err != nil {
		return [32]byte{}, fmt.Errorf("create %s: %w", e.Name(), err)
	}
	h := blake3.New()
	_, err = io.CopyN(io.MultiWriter(&trackedWriter{w: w, err: &rd.written}, h), tr, size)
	if cerr := w.Close(); err == nil && cerr != nil {
		return [32]byte{}, fmt.Errorf("write %s: %w", e.Name(), cerr)
	}
	if err != nil {
		return [32]byte{}, rd.fail(DamageCut, err)
	}
	var sum [32]byte
	h.Sum(sum[:0])
	return sum, nil
}

func parseManifest(b []byte) (Manifest, error) {
	var head struct {
		Format  int    `json:"format"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(b, &head); err != nil || head.Format < 1 {
		return Manifest{}, fmt.Errorf("%w: manifest.json names no format", ErrNotBackup)
	}
	if head.Format > Format {
		return Manifest{}, &TooNewError{Format: head.Format, Version: head.Version}
	}
	var m Manifest
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&m); err != nil {
		return Manifest{}, &DamagedError{Reason: DamageManifest, Err: err}
	}
	var problem string
	switch {
	case d.More():
		problem = "data follows the manifest"
	case m.Version == "":
		problem = "the manifest names no version"
	case !slices.Contains(triggers, m.Trigger):
		problem = fmt.Sprintf("the trigger %q isn't one SlipStream writes", m.Trigger)
	case len(m.ControllerKey) != 32:
		problem = fmt.Sprintf("the Controller key is %d bytes, not 32", len(m.ControllerKey))
	case m.Generation == 0:
		problem = "the manifest names no passphrase generation"
	case m.TakenAt.IsZero():
		problem = "the manifest names no time"
	}
	if problem != "" {
		return Manifest{}, &DamagedError{Reason: DamageManifest, Err: errors.New(problem)}
	}
	return m, nil
}

// atEnd reads once more from a layer whose consumer has finished, which
// must answer nothing and io.EOF.
func atEnd(r io.Reader) error {
	n, err := r.Read(make([]byte, 1))
	switch {
	case n > 0:
		return errors.New("more data")
	case errors.Is(err, io.EOF):
		return nil
	case err == nil:
		return errors.New("no end of stream")
	}
	return err //nolint:wrapcheck // fail blames the layer that saw it
}

// source hashes and counts the ciphertext and keeps the first error
// reading it, which is never damage.
type source struct {
	r   io.Reader
	h   *blake3.Hasher
	n   int64
	err error
}

func (s *source) Read(p []byte) (int, error) {
	n, err := s.r.Read(p)
	_, _ = s.h.Write(p[:n])
	s.n += int64(n)
	if err != nil && !errors.Is(err, io.EOF) && s.err == nil {
		s.err = err
	}
	return n, err //nolint:wrapcheck // a Reader returns io.EOF as itself
}

type tracked struct {
	r   io.Reader
	err error
}

func (t *tracked) Read(p []byte) (int, error) {
	n, err := t.r.Read(p)
	if err != nil && !errors.Is(err, io.EOF) && t.err == nil {
		t.err = err
	}
	return n, err //nolint:wrapcheck // a Reader returns io.EOF as itself
}

type trackedWriter struct {
	w   io.Writer
	err *error
}

func (t *trackedWriter) Write(p []byte) (int, error) {
	n, err := t.w.Write(p)
	if err != nil && *t.err == nil {
		*t.err = err
	}
	return n, err //nolint:wrapcheck // Read wraps it as the sink's
}
