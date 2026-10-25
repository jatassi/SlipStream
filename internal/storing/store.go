package storing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	nodev1 "github.com/jatassi/slipstream/internal/pb/slipstream/node/v1"
	storagev1 "github.com/jatassi/slipstream/internal/pb/slipstream/storage/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/rpc"
	"github.com/jatassi/slipstream/internal/storage"
)

// Payloads is where this Node's finished downloads lie.
type Payloads interface {
	// Path is where file index of torrent h lies on this Node, an error
	// until the torrent finished here.
	Path(h bencode.InfoHash, index int) (string, error)
}

type Config struct {
	Disk     Disk
	Payloads Payloads
	Timings  Timings
}

// resampleEvery is the least time between two samples a placement asks
// for on one volume, so a season's import doesn't stat the disk per file.
const resampleEvery = 10 * time.Second

// Store is a Node's Storage, one per Node from its start, claimed or not.
// Run keeps the roots its intent lists and reports them; the
// StorageService handlers act on them by id.
type Store struct {
	cfg  Config
	host Host
	poke chan struct{}
	// asks is each Sample's reply channel, which Run answers with the
	// stamp of a pass it began after the ask.
	asks chan chan<- int64

	mu sync.Mutex
	// stopped is closed while Run isn't running.
	stopped chan struct{}
	// roots are the latest intent's, in its order.
	roots    []intentRoot
	draining bool
	// sampled is each root's volume as the last pass read it.
	sampled map[storage.UUID]Volume
	// stuck counts the bounded calls per key still in the kernel past
	// their deadline.
	stuck map[string]int
	// resampled is when a placement last asked for a sample, by volume id.
	resampled map[string]time.Time
}

type intentRoot struct {
	id    storage.UUID
	path  string
	place int64
	adopt bool
	proof []Rel
}

func NewStore(cfg Config) *Store {
	stopped := make(chan struct{})
	close(stopped)
	return &Store{
		cfg:       cfg,
		host:      cfg.Disk.Host(),
		poke:      make(chan struct{}, 1),
		asks:      make(chan chan<- int64),
		stopped:   stopped,
		sampled:   map[storage.UUID]Volume{},
		stuck:     map[string]int{},
		resampled: map[string]time.Time{},
	}
}

// Run is the Role's run function. On each snapshot, each tick of the
// intent's interval, each poke and each Sample it judges every root and
// samples its volume, then sets the facts. The Samples that queue during
// a pass share the next one.
func (s *Store) Run(ctx context.Context, h Holder) error {
	stopped := make(chan struct{})
	s.mu.Lock()
	s.stopped = stopped
	s.mu.Unlock()
	defer close(stopped)
	snapshots := latest(h.Intent(ctx))
	var (
		every  time.Duration
		ticker *time.Ticker
		tick   <-chan time.Time
		passed int64
	)
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()
	for {
		var asks []chan<- int64
		select {
		case snap := <-snapshots:
			if err := s.keep(snap); err != nil {
				return err
			}
			if d := time.Duration(snap.Intent.GetSampleEverySeconds()) * time.Second; d != every {
				every = d
				if ticker != nil {
					ticker.Stop()
				}
				ticker, tick = nil, nil
				if d > 0 {
					ticker = time.NewTicker(d)
					tick = ticker.C
				}
			}
		case <-s.poke:
		case <-tick:
		case ask := <-s.asks:
			asks = append(s.queued(), ask)
		case <-ctx.Done():
			return nil
		}
		passed = s.passAfter(ctx, h, passed, asks)
	}
}

// passAfter passes under a stamp above last, sets the facts and answers
// asks with the stamp. Stamps rise strictly, so a Sample's answer names
// one pass even when two begin in one millisecond.
func (s *Store) passAfter(ctx context.Context, h Holder, last int64, asks []chan<- int64) int64 {
	passed := max(time.Now().UnixMilli(), last+1)
	facts := s.pass(ctx)
	facts.PassedAtUnixMs = passed
	h.SetFacts(facts)
	for _, ask := range asks {
		ask <- passed
	}
	return passed
}

func (s *Store) queued() []chan<- int64 {
	var asks []chan<- int64
	for {
		select {
		case ask := <-s.asks:
			asks = append(asks, ask)
		default:
			return asks
		}
	}
}

// Sample has Run pass now and answers the pass's stamp. While Run isn't
// running it refuses as a Node that doesn't hold Storage.
func (s *Store) Sample(ctx context.Context, _ *storagev1.SampleRequest) (*storagev1.SampleResponse, error) {
	s.mu.Lock()
	stopped := s.stopped
	s.mu.Unlock()
	reply := make(chan int64, 1)
	select {
	case s.asks <- reply:
	case <-stopped:
		return nil, rpc.ErrorWith(rpc.CodeFailedPrecondition, &nodev1.NotHolder{Role: string(role.Storage)}, "this Node doesn't hold the storage Role")
	case <-ctx.Done():
		return nil, fmt.Errorf("ask for a sample: %w", ctx.Err())
	}
	select {
	case passed := <-reply:
		return &storagev1.SampleResponse{PassedAtUnixMs: passed}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("await a sample: %w", ctx.Err())
	}
}

// latest pumps seq into a channel that holds only the newest snapshot, so
// the run loop never falls behind its intent.
func latest(seq iter.Seq[role.Snapshot[*storagev1.Intent]]) <-chan role.Snapshot[*storagev1.Intent] {
	ch := make(chan role.Snapshot[*storagev1.Intent], 1)
	go func() {
		for snap := range seq {
			select {
			case <-ch:
			default:
			}
			ch <- snap
		}
	}()
	return ch
}

func (s *Store) keep(snap role.Snapshot[*storagev1.Intent]) error {
	roots, err := parseIntent(snap.Intent)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roots, s.draining = roots, snap.Draining
	return nil
}

func parseIntent(in *storagev1.Intent) ([]intentRoot, error) {
	roots := make([]intentRoot, 0, len(in.GetRoots()))
	for _, r := range in.GetRoots() {
		id, err := storage.UUIDFromBytes(r.GetId())
		if err != nil {
			return nil, fmt.Errorf("intent: root id: %w", err)
		}
		proof := make([]Rel, 0, len(r.GetProof()))
		for _, p := range r.GetProof() {
			rel, err := ParseRel(p)
			if err != nil {
				return nil, fmt.Errorf("intent: root %s's proof: %w", id, err)
			}
			proof = append(proof, rel)
		}
		roots = append(roots, intentRoot{id: id, path: r.GetPath(), place: r.GetPlace(), adopt: r.GetAdopt(), proof: proof})
	}
	return roots, nil
}

// Release forgets every root: the Node let Storage go.
func (s *Store) Release(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roots, s.draining = nil, false
	clear(s.sampled)
	return nil
}

type verdict struct {
	state RootState
	// detail is the OS error text behind the state, if any.
	detail string
}

func (s *Store) pass(ctx context.Context) *storagev1.Facts {
	s.mu.Lock()
	roots := s.roots
	s.mu.Unlock()
	known := map[storage.UUID]bool{}
	for _, r := range roots {
		known[r.id] = true
	}
	verdicts := make([]verdict, len(roots))
	var samples []rootSample
	for i, r := range roots {
		verdicts[i] = s.judge(ctx, r, known)
		if st := verdicts[i].state; st == Missing || st == Unresponsive {
			continue
		}
		v, err := timed(ctx, s, r.path, func(ctx context.Context) (Volume, error) { return s.cfg.Disk.Volume(ctx, r.path) })
		switch {
		case errors.Is(err, errUnresponsive):
			verdicts[i] = verdict{Unresponsive, err.Error()}
		case err == nil:
			samples = append(samples, rootSample{root: r.id, volume: v})
		}
	}
	vols, of := volumesOf(samples)
	s.mu.Lock()
	clear(s.sampled)
	for _, smp := range samples {
		s.sampled[smp.root] = smp.volume
	}
	s.mu.Unlock()
	facts := &storagev1.Facts{}
	for i, r := range roots {
		facts.Roots = append(facts.Roots, &storagev1.RootFacts{
			Id: r.id.Bytes(), Place: r.place, State: verdicts[i].state.Wire(), VolumeId: of[r.id], Detail: verdicts[i].detail,
		})
	}
	for _, v := range vols {
		facts.Volumes = append(facts.Volumes, volumeWire(v))
	}
	return facts
}

// judge reads r's folder and marker. A root to adopt whose folder lacks
// its marker, or holds a stray one naming no root of the intent, gets its
// marker once the folder proves to be the root's.
func (s *Store) judge(ctx context.Context, r intentRoot, known map[storage.UUID]bool) verdict {
	m, err := s.read(ctx, r.path)
	switch {
	case err != nil:
		return verdictOf(err)
	case m.names(r.id):
	case !r.adopt && m.present:
		return verdict{state: Foreign}
	case !r.adopt:
		return verdict{state: Unmounted}
	case m.present && known[m.id]:
		return verdict{state: Foreign}
	default:
		return s.adopt(ctx, r)
	}
	ok, err := timed(ctx, s, r.path, func(ctx context.Context) (bool, error) { return s.cfg.Disk.Writable(ctx, r.path) })
	switch {
	case err != nil:
		return verdictOf(err)
	case !ok:
		return verdict{state: Unwritable}
	}
	return verdict{state: Ready}
}

// adopt writes r's marker when its folder is writable and holds one of
// its proof files, or r has none yet. Otherwise the folder may be an empty
// mount point, which reads unmounted until a later pass finds the proof.
func (s *Store) adopt(ctx context.Context, r intentRoot) verdict {
	ok, err := timed(ctx, s, r.path, func(ctx context.Context) (bool, error) { return s.cfg.Disk.Writable(ctx, r.path) })
	switch {
	case err != nil:
		return verdictOf(err)
	case !ok:
		return verdict{state: Unwritable}
	}
	proven, err := timed(ctx, s, r.path, func(ctx context.Context) (bool, error) { return s.proven(ctx, r) })
	switch {
	case err != nil:
		return verdictOf(err)
	case !proven:
		return verdict{state: Unmounted}
	}
	if _, err := timed(ctx, s, r.path, func(ctx context.Context) (struct{}, error) { return struct{}{}, s.mark(ctx, r) }); err != nil {
		return verdictOf(err)
	}
	return verdict{state: Ready}
}

func (s *Store) proven(ctx context.Context, r intentRoot) (bool, error) {
	if len(r.proof) == 0 {
		return true, nil
	}
	for _, p := range r.proof {
		_, err := s.cfg.Disk.Stat(ctx, s.host.Style.Under(r.path, p))
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return false, err
		}
	}
	return false, nil
}

// mark writes r's marker beside it and renames it into place, so no
// reader sees a half-written one.
func (s *Store) mark(ctx context.Context, r intentRoot) error {
	marker := s.host.Style.Join(r.path, MarkerName)
	part := marker + ".part"
	f, err := s.cfg.Disk.Create(ctx, part)
	if err != nil {
		return fmt.Errorf("write the marker: %w", err)
	}
	_, err = io.WriteString(f, r.id.String())
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = s.cfg.Disk.Rename(ctx, part, marker)
	}
	if err != nil {
		_ = s.cfg.Disk.Remove(ctx, part)
		return fmt.Errorf("write the marker: %w", err)
	}
	return nil
}

type marking struct {
	present bool
	// id is the root the marker names, zero when it names none.
	id storage.UUID
}

func (m marking) names(id storage.UUID) bool { return m.present && m.id == id }

var errNotAFolder = errors.New("not a folder")

// read stats folder and reads its marker under the stat deadline.
func (s *Store) read(ctx context.Context, folder string) (marking, error) {
	return timed(ctx, s, folder, func(ctx context.Context) (marking, error) {
		info, err := s.cfg.Disk.Stat(ctx, folder)
		if err != nil {
			return marking{}, err
		}
		if !info.Dir() {
			return marking{}, fmt.Errorf("%s: %w", folder, errNotAFolder)
		}
		f, err := s.cfg.Disk.Open(ctx, s.host.Style.Join(folder, MarkerName))
		if errors.Is(err, fs.ErrNotExist) {
			return marking{}, nil
		}
		if err != nil {
			return marking{}, fmt.Errorf("read the marker: %w", err)
		}
		defer func() { _ = f.Close() }()
		b, err := io.ReadAll(io.LimitReader(f, 64))
		if err != nil {
			return marking{}, fmt.Errorf("read the marker: %w", err)
		}
		id, err := storage.ParseUUID(strings.TrimSpace(string(b)))
		if err != nil {
			return marking{present: true}, nil
		}
		return marking{present: true, id: id}, nil
	})
}

// folderErrorOf classifies a Disk error. Every answer to a failed call reads
// through it.
func folderErrorOf(err error) FolderErrorKind {
	switch {
	case errors.Is(err, errUnresponsive):
		return FolderUnresponsive
	case errors.Is(err, fs.ErrNotExist):
		return FolderNotFound
	case errors.Is(err, errNotAFolder):
		return FolderNotAFolder
	case errors.Is(err, fs.ErrPermission):
		return FolderDenied
	}
	return FolderIO
}

var stateOf = map[FolderErrorKind]RootState{
	FolderNotFound:     Missing,
	FolderNotAFolder:   Missing,
	FolderDenied:       Unwritable,
	FolderUnresponsive: Unresponsive,
	FolderIO:           Unresponsive,
}

func verdictOf(err error) verdict { return verdict{stateOf[folderErrorOf(err)], err.Error()} }

// resolve is root's folder for a write: in the intent, its marker naming
// it at this instant, and for a placement not draining. Else *NotReady,
// and a poke so the facts follow at once.
func (s *Store) resolve(ctx context.Context, root storage.UUID, place bool) (string, error) {
	s.mu.Lock()
	r, known := s.rootLocked(root)
	draining := s.draining
	s.mu.Unlock()
	switch {
	case !known:
		return "", &NotReady{Unknown: true}
	case place && draining:
		return "", &NotReady{Draining: true}
	}
	m, err := s.read(ctx, r.path)
	if err == nil && m.names(root) {
		return r.path, nil
	}
	s.wake()
	state := Unmounted
	switch {
	case err != nil:
		state = verdictOf(err).state
	case m.present:
		state = Foreign
	}
	return "", &NotReady{State: state}
}

func (s *Store) rootLocked(id storage.UUID) (intentRoot, bool) {
	for _, r := range s.roots {
		if r.id == id {
			return r, true
		}
	}
	return intentRoot{}, false
}

// Local is rel under root on this Node, for the Transfer receiver's
// staging, which brings new bytes in as a placement does.
func (s *Store) Local(ctx context.Context, root storage.UUID, rel Rel) (string, error) {
	dir, err := s.resolve(ctx, root, true)
	if err != nil {
		return "", err
	}
	return s.host.Style.Under(dir, rel), nil
}

func (s *Store) wake() {
	select {
	case s.poke <- struct{}{}:
	default:
	}
}

// resample asks for a pass after a placement into root, at most once per
// resampleEvery for its volume.
func (s *Store) resample(root storage.UUID) {
	s.mu.Lock()
	key := root.String()
	if v, ok := s.sampled[root]; ok {
		key = v.ID
	}
	now := time.Now()
	due := now.Sub(s.resampled[key]) >= resampleEvery
	if due {
		s.resampled[key] = now
	}
	s.mu.Unlock()
	if due {
		s.wake()
	}
}

// errUnresponsive is a Disk call that missed the stat deadline. The call
// goes on in its goroutine until the kernel answers.
var errUnresponsive = errors.New("the disk didn't respond in time")

// timed runs fn under the stat deadline and answers errUnresponsive once it
// passes, or ctx's error when ctx ends first. A filesystem call can't be
// interrupted, so a miss leaves it running: while an earlier call on key is
// still in the kernel, timed answers errUnresponsive at once rather than
// stack another call behind it.
func timed[T any](ctx context.Context, s *Store, key string, fn func(context.Context) (T, error)) (T, error) {
	var none T
	s.mu.Lock()
	stuck := s.stuck[key] > 0
	s.mu.Unlock()
	if stuck {
		return none, fmt.Errorf("%s is still not responding: %w", key, errUnresponsive)
	}
	type answer struct {
		v   T
		err error
	}
	done := make(chan answer, 1)
	go func() {
		v, err := fn(ctx)
		done <- answer{v, err}
	}()
	deadline := time.NewTimer(s.cfg.Timings.Stat)
	defer deadline.Stop()
	select {
	case a := <-done:
		return a.v, a.err
	case <-ctx.Done():
		return none, fmt.Errorf("%s: %w", key, ctx.Err())
	case <-deadline.C:
	}
	s.mu.Lock()
	s.stuck[key]++
	s.mu.Unlock()
	go func() {
		<-done
		s.mu.Lock()
		defer s.mu.Unlock()
		s.stuck[key]--
		if s.stuck[key] == 0 {
			delete(s.stuck, key)
		}
	}()
	return none, fmt.Errorf("%s hasn't responded within %s: %w", key, s.cfg.Timings.Stat, errUnresponsive)
}
