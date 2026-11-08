// Package torrentclient is the one fake torrent client engine the four
// fake Download Clients share: their torrents on the Scenario clock,
// before any protocol. It writes each torrent's payload from the release
// catalog into a real folder as the torrent downloads, under the client's
// own name for that folder, and plays the faults row 16 of
// docs/offline-boundaries.md lists. It computes progress when it is read,
// so it runs no goroutine and fake time under synctest moves it.
package torrentclient

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/fakes/internal/releases"
	"github.com/jatassi/slipstream/internal/bencode"
)

type Config struct {
	Releases *releases.Catalog
	// Dir is the real folder the files land in, and Name the client's own
	// name for it, such as /downloads or D:\Torrents. Every folder the
	// engine reports starts with Name.
	Dir, Name string
	// Windows is a client whose paths use backslashes.
	Windows bool
	// Rate is how many bytes a second each torrent downloads, and uploads
	// while it seeds.
	Rate int64
	// Slots is how many torrents download at once, the rest queued in add
	// order, 0 for no queue.
	Slots int
	// SeedRatio is the ratio at which the client stops a seeding torrent,
	// 0 for no limit.
	SeedRatio float64
	// Check is how long the client checks a torrent's data when it is
	// added or rechecked, 0 for at once.
	Check time.Duration
}

type State uint8

const (
	Stopped State = iota + 1
	// Checking is a torrent whose data the client checks, started or
	// stopped.
	Checking
	Queued
	Downloading
	Seeding
)

// Torrent is one torrent as the engine holds it.
type Torrent struct {
	// ID is the client's own number for it, in add order from 1.
	ID   int
	Hash bencode.InfoHash
	Name string
	// Folder is its save folder as the client names it.
	Folder string
	// Files are its files in metainfo order, Path relative to Folder with
	// slashes and the torrent's own folder first for a multi-file torrent.
	Files      []File
	Size, Done int64
	State      State
	// Order is its place in the client's queue among every torrent, from 0.
	Order  int
	Labels []string
	// CompletedAt is when its last byte arrived, zero before.
	CompletedAt          time.Time
	Uploaded, Downloaded int64
	Downloading, Seeding time.Duration
	// LocalError is the client's words for a local error, such as a full
	// disk, which stops the torrent.
	LocalError string
	Tracker    Tracker
	// SeedGoalMet is a torrent the client stopped at its seed ratio.
	SeedGoalMet bool
	Rates       Rates
}

type File struct {
	Path       string
	Size, Done int64
}

// Tracker is the tracker's last message, "" for none.
type Tracker struct {
	Message string
	Error   bool
}

type Rates struct {
	Down, Up     int64
	Peers, Seeds int
}

var (
	ErrNotFound = errors.New("the client lists no such torrent")
	ErrCorrupt  = errors.New("the metainfo isn't a torrent")
)

// Engine is one fake client's torrents. It is safe for concurrent use.
type Engine struct {
	cfg Config

	mu       sync.Mutex
	at       time.Time
	nextID   int
	torrents []*torrent
	// dropNext has the next Add answer as though it kept the torrent.
	dropNext bool
	// lateNext hides the next added torrent from reads for that long.
	lateNext time.Duration
	restarts int
}

type torrent struct {
	Torrent
	payload string
	written int64
	stalled bool
	hidden  time.Time
	// checked is when a check of its data ends.
	checked time.Time
}

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg, at: time.Now()}
}

// DefaultFolder is where the client saves a torrent added with no folder.
func (e *Engine) DefaultFolder() string { return e.cfg.Name }

// Add adds the torrent metainfo describes to folder, or to the default
// folder for "", and answers it. A torrent already in the client is
// answered unchanged with duplicate set.
func (e *Engine) Add(metainfo []byte, folder string, labels []string, paused bool) (t Torrent, duplicate bool, err error) {
	m, err := bencode.DecodeMetainfo(metainfo)
	if err != nil {
		return Torrent{}, false, ErrCorrupt
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.advance()
	if i := e.index(m.InfoHash); i >= 0 {
		return e.snapshot(i), true, nil
	}
	if folder == "" {
		folder = e.cfg.Name
	}
	e.nextID++
	added := &torrent{Torrent: Torrent{
		ID: e.nextID, Hash: m.InfoHash, Name: m.Info.Name, Folder: folder, Labels: slices.Clone(labels),
		State: Downloading,
	}}
	for _, f := range m.Info.Files {
		p := m.Info.Name
		if f.Path != nil {
			p = path.Join(append([]string{m.Info.Name}, f.Path...)...)
		}
		added.Files = append(added.Files, File{Path: p, Size: f.Length})
		added.Size += f.Length
	}
	if r, ok := e.cfg.Releases.ByInfoHash(m.InfoHash); ok {
		added.payload = r.File
	}
	if paused {
		added.State = Stopped
	}
	added.checked = e.at.Add(e.cfg.Check)
	if e.lateNext > 0 {
		added.hidden = e.at.Add(e.lateNext)
		e.lateNext = 0
	}
	if e.dropNext {
		e.dropNext = false
		return added.Torrent, false, nil
	}
	e.torrents = append(e.torrents, added)
	e.settle()
	return e.snapshot(len(e.torrents) - 1), false, nil
}

// List is each torrent of hashes the client lists, in add order, or every
// torrent for nil.
func (e *Engine) List(hashes []bencode.InfoHash) []Torrent {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.advance()
	var out []Torrent
	for i, t := range e.torrents {
		if t.hidden.After(e.at) || (hashes != nil && !slices.Contains(hashes, t.Hash)) {
			continue
		}
		out = append(out, e.snapshot(i))
	}
	return out
}

func (e *Engine) Get(h bencode.InfoHash) (Torrent, bool) {
	ts := e.List([]bencode.InfoHash{h})
	if len(ts) == 0 {
		return Torrent{}, false
	}
	return ts[0], true
}

// Stop stops a torrent, as pausing it in the client does.
func (e *Engine) Stop(h bencode.InfoHash) error {
	return e.change(h, func(t *torrent) { t.State = Stopped })
}

// Start starts a stopped torrent and clears its local error.
func (e *Engine) Start(h bencode.InfoHash) error {
	return e.change(h, func(t *torrent) {
		if t.State == Stopped {
			t.State, t.LocalError, t.SeedGoalMet = Downloading, "", false
		}
	})
}

// Verify rechecks a torrent's data, which the engine holds as written.
func (e *Engine) Verify(h bencode.InfoHash) error {
	return e.change(h, func(t *torrent) { t.checked = e.at.Add(e.cfg.Check) })
}

// Announce is a reannounce, which changes nothing the engine holds.
func (e *Engine) Announce(h bencode.InfoHash) error { return e.change(h, func(*torrent) {}) }

func (e *Engine) SetLabels(h bencode.InfoHash, labels []string) error {
	return e.change(h, func(t *torrent) { t.Labels = slices.Clone(labels) })
}

// Remove removes a torrent, deleting its files and the folders they
// leave empty under its folder when data is set.
func (e *Engine) Remove(h bencode.InfoHash, data bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.advance()
	i := e.index(h)
	if i < 0 {
		return ErrNotFound
	}
	t := e.torrents[i]
	e.torrents = slices.Delete(e.torrents, i, i+1)
	if !data {
		return nil
	}
	return e.deleteFiles(t)
}

// Move moves a torrent's files to folder, as a client's move on
// completion does.
func (e *Engine) Move(h bencode.InfoHash, folder string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.advance()
	i := e.index(h)
	if i < 0 {
		return ErrNotFound
	}
	t := e.torrents[i]
	from, _ := e.local(t.Folder)
	to, ok := e.local(folder)
	if !ok {
		return fmt.Errorf("no such file or directory (%s)", folder)
	}
	for _, f := range t.Files {
		name := filepath.Join(to, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
			return fmt.Errorf("make the folder: %w", err)
		}
		if err := os.Rename(filepath.Join(from, filepath.FromSlash(f.Path)), name); err != nil {
			return fmt.Errorf("move the file: %w", err)
		}
	}
	t.Folder = folder
	return nil
}

// Stall stops a torrent's data from arriving, or lets it arrive again.
func (e *Engine) Stall(h bencode.InfoHash, on bool) error {
	return e.change(h, func(t *torrent) { t.stalled = on })
}

// Fail gives a torrent a local error, such as a full disk, and stops it.
// "" clears the error and leaves the torrent stopped.
func (e *Engine) Fail(h bencode.InfoHash, message string) error {
	return e.change(h, func(t *torrent) {
		t.LocalError = message
		if message != "" {
			t.State = Stopped
		}
	})
}

// Announced sets a torrent's tracker message, "" for none.
func (e *Engine) Announced(h bencode.InfoHash, tr Tracker) error {
	return e.change(h, func(t *torrent) { t.Tracker = tr })
}

// Empty removes every torrent and keeps their files.
func (e *Engine) Empty() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.torrents = nil
}

// DropNextAdd has the next Add answer as the client does but keep nothing.
func (e *Engine) DropNextAdd() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.dropNext = true
}

// ListNextAddAfter has the next added torrent missing from reads for d.
func (e *Engine) ListNextAddAfter(d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lateNext = d
}

// Restart restarts the client, keeping its torrents. A protocol that
// holds sessions starts a new one.
func (e *Engine) Restart() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.restarts++
}

// Restarts is how many times the client has restarted.
func (e *Engine) Restarts() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.restarts
}

func (e *Engine) change(h bencode.InfoHash, f func(*torrent)) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.advance()
	i := e.index(h)
	if i < 0 {
		return ErrNotFound
	}
	f(e.torrents[i])
	e.settle()
	return nil
}

func (e *Engine) index(h bencode.InfoHash) int {
	return slices.IndexFunc(e.torrents, func(t *torrent) bool { return t.Hash == h })
}

func (e *Engine) snapshot(i int) Torrent {
	t := e.torrents[i]
	out := t.Torrent
	out.Order = i
	out.Labels = slices.Clone(t.Labels)
	out.Files = make([]File, len(t.Files))
	at := int64(0)
	for j, f := range t.Files {
		out.Files[j] = File{Path: f.Path, Size: f.Size, Done: min(max(t.Done-at, 0), f.Size)}
		at += f.Size
	}
	if e.checking(t) {
		out.State = Checking
	}
	if e.moving(t) {
		switch out.State {
		case Downloading:
			out.Rates = Rates{Down: e.cfg.Rate, Peers: 1, Seeds: 1}
		case Seeding:
			out.Rates = Rates{Up: e.cfg.Rate, Peers: 1}
		case Stopped, Checking, Queued:
		}
	}
	return out
}

func (e *Engine) checking(t *torrent) bool { return t.checked.After(e.at) }

// moving is a torrent whose data flows: one with a payload that isn't
// stalled.
func (e *Engine) moving(t *torrent) bool { return t.payload != "" && !t.stalled }

// settle puts each started torrent in its state: seeding once complete,
// downloading in the first Slots of the incomplete ones, queued after.
func (e *Engine) settle() {
	slots := 0
	for _, t := range e.torrents {
		switch {
		case t.State == Stopped:
		case t.Done == t.Size:
			t.State = Seeding
		case e.cfg.Slots == 0 || slots < e.cfg.Slots:
			t.State = Downloading
			slots++
		default:
			t.State = Queued
		}
	}
}

// advance runs the clock from the last read to now, one step to each
// moment a torrent ends a check, completes or meets its seed ratio, then
// writes the bytes that arrived.
func (e *Engine) advance() {
	now := time.Now()
	for e.at.Before(now) {
		step := now.Sub(e.at)
		for _, t := range e.torrents {
			if need, ok := e.untilNext(t); ok {
				step = min(step, need)
			}
		}
		for _, t := range e.torrents {
			e.run(t, step)
		}
		e.at = e.at.Add(step)
		e.settle()
	}
	for _, t := range e.torrents {
		if t.written < t.Done && t.LocalError == "" {
			if err := e.write(t); err != nil {
				t.LocalError, t.State = err.Error(), Stopped
				e.settle()
			}
		}
	}
}

// untilNext is how long until t ends a check, completes or meets the
// seed ratio, false for a torrent that does none of them.
func (e *Engine) untilNext(t *torrent) (time.Duration, bool) {
	if e.checking(t) {
		return t.checked.Sub(e.at), true
	}
	if !e.moving(t) {
		return 0, false
	}
	switch t.State {
	case Downloading:
		return e.timeFor(t.Size - t.Done), true
	case Seeding:
		if e.cfg.SeedRatio > 0 {
			return e.timeFor(int64(math.Ceil(e.cfg.SeedRatio*float64(t.Size))) - t.Uploaded), true
		}
	case Stopped, Checking, Queued:
	}
	return 0, false
}

func (e *Engine) timeFor(bytes int64) time.Duration {
	return max(time.Duration(math.Ceil(float64(bytes)/float64(e.cfg.Rate)*float64(time.Second))), 0)
}

func (e *Engine) bytesIn(d time.Duration) int64 { return int64(float64(e.cfg.Rate) * d.Seconds()) }

func (e *Engine) run(t *torrent, step time.Duration) {
	if e.checking(t) {
		return
	}
	switch t.State {
	case Downloading:
		t.Downloading += step
		if !e.moving(t) {
			return
		}
		got := e.bytesIn(step)
		if step >= e.timeFor(t.Size-t.Done) {
			got = t.Size - t.Done
		}
		t.Done += got
		t.Downloaded += got
		if t.Done == t.Size {
			t.CompletedAt = e.at.Add(step)
		}
	case Seeding:
		t.Seeding += step
		if !e.moving(t) {
			return
		}
		goal := int64(math.Ceil(e.cfg.SeedRatio * float64(t.Size)))
		if e.cfg.SeedRatio > 0 && step >= e.timeFor(goal-t.Uploaded) {
			t.Uploaded = goal
			t.State, t.SeedGoalMet = Stopped, true
			return
		}
		t.Uploaded += e.bytesIn(step)
	case Stopped, Checking, Queued:
	}
}

// local is folder, as the client names it, on the real disk: Dir with the
// path under Name. A folder outside Name has none.
func (e *Engine) local(folder string) (string, bool) {
	sep, same := "/", func(a, b string) bool { return a == b }
	if e.cfg.Windows {
		sep, same = `\`, strings.EqualFold
	}
	name := strings.TrimSuffix(e.cfg.Name, sep)
	if len(folder) < len(name) || !same(folder[:len(name)], name) {
		return "", false
	}
	under := folder[len(name):]
	if under != "" && !strings.HasPrefix(under, sep) {
		return "", false
	}
	parts := strings.Split(strings.Trim(under, sep), sep)
	return filepath.Join(append([]string{e.cfg.Dir}, parts...)...), true
}

// write copies the payload bytes t has received and not yet written into
// its files.
func (e *Engine) write(t *torrent) error {
	dir, ok := e.local(t.Folder)
	if !ok {
		return fmt.Errorf("no such file or directory (%s)", t.Folder)
	}
	src, err := os.Open(t.payload)
	if err != nil {
		return fmt.Errorf("open the payload: %w", err)
	}
	defer func() { _ = src.Close() }()
	at := int64(0)
	for _, f := range t.Files {
		from, to := max(t.written, at), min(t.Done, at+f.Size)
		if from < to {
			if err := copyRange(src, filepath.Join(dir, filepath.FromSlash(f.Path)), from, at, to-from); err != nil {
				return err
			}
		}
		at += f.Size
	}
	t.written = t.Done
	return nil
}

// copyRange copies n bytes of src from offset into the file at name,
// whose first byte is src's byte start.
func copyRange(src io.ReaderAt, name string, offset, start, n int64) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o750); err != nil {
		return fmt.Errorf("make the folder: %w", err)
	}
	dst, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0o640) //nolint:gosec // the path is the torrent's under the engine's folder
	if err != nil {
		return fmt.Errorf("open the file: %w", err)
	}
	_, err = io.Copy(io.NewOffsetWriter(dst, offset-start), io.NewSectionReader(src, offset, n))
	if closeErr := dst.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("write the file: %w", err)
	}
	return nil
}

func (e *Engine) deleteFiles(t *torrent) error {
	dir, ok := e.local(t.Folder)
	if !ok {
		return nil
	}
	for _, f := range t.Files {
		name := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete the file: %w", err)
		}
		for parent := filepath.Dir(name); parent != dir && strings.HasPrefix(parent, dir); parent = filepath.Dir(parent) {
			if os.Remove(parent) != nil {
				break
			}
		}
	}
	return nil
}
