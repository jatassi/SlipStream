// Package enginetest is an in-memory torrent engine behind the engine
// interface. Its engines share a World: the payloads its swarm holds and
// the trackers it runs. It starts no goroutine and no thread, only timers,
// so a testing/synctest bubble can hold it, which libtorrent's threads
// cannot (ADR 0014). It adds torrents paused, as libtorrent does for the
// engine. A torrent added in seed mode seeds, and one added from resume
// data holds what that says. Any other checks the files it finds under its
// save path once resumed, then downloads when the World holds its payload,
// at once or at the rates Transfer sets, and else never finishes.
package enginetest

import (
	"cmp"
	"crypto/sha1" //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
	"errors"
	"fmt"
	"io"
	"maps"
	"math"
	"net/netip"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

// events is how many events wait for a reader before the engine drops
// them and sends Dropped.
const events = 64

// firstPort is the first port a listen interface with port 0 tries.
const firstPort = 6881

// version is the in-memory engine's identity, in libtorrent's encoding.
var version = engine.Version{Libtorrent: "0.0.0", PeerIDPrefix: "-LT0000-", UserAgent: "libtorrent/0.0.0.0"}

var errAddressInUse = &engine.Error{Category: engine.CategorySystem, Code: 98, Text: "address already in use"}

// World is the swarm, the trackers and the host every engine it opens
// shares.
type World struct {
	// mu guards the World and every engine it opened.
	mu        sync.Mutex
	payloads  map[bencode.InfoHash][]byte
	transfers map[bencode.InfoHash]rates
	trackers  map[string]*tracker
	// engines are the open engines.
	engines []*Engine
	// taken are ports something other than an engine holds.
	taken map[netip.AddrPort]bool
	full  bool
	// adds counts the torrents added, which mints each one's peer ID.
	adds int
}

// rates are payload bytes a second.
type rates struct{ down, up int64 }

func NewWorld() *World {
	return &World{
		payloads: map[bencode.InfoHash][]byte{}, transfers: map[bencode.InfoHash]rates{},
		trackers: map[string]*tracker{}, taken: map[netip.AddrPort]bool{},
	}
}

// Open builds an engine listening on cfg's interface, or on its address's
// first free port from 6881 when it asks for any port, 0.0.0.0 standing in
// for no address. A port taken by TakePort or another engine leaves it not
// listening, as libtorrent's strict bind does, and it sends ListenFailed.
func (w *World) Open(cfg engine.Config) (engine.Engine, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	e := &Engine{
		w:        w,
		floor:    cfg.AnnounceFloor,
		events:   make(chan engine.Event, events),
		torrents: map[engine.Handle]*torrent{},
		parting:  map[bencode.InfoHash]*torrent{},
		posted:   map[engine.Handle]engine.Status{},
	}
	listen := cfg.ListenInterface
	if !listen.Addr().IsValid() {
		listen = netip.AddrPortFrom(netip.IPv4Unspecified(), listen.Port())
	}
	if listen.Port() == 0 {
		port := uint16(firstPort)
		for w.inUse(netip.AddrPortFrom(listen.Addr(), port)) {
			port++
		}
		listen = netip.AddrPortFrom(listen.Addr(), port)
	}
	if w.inUse(listen) {
		e.emit(engine.ListenFailed{Err: errAddressInUse})
	} else {
		e.listen = listen
	}
	w.engines = append(w.engines, e)
	return e, nil
}

func (w *World) inUse(ap netip.AddrPort) bool {
	return w.taken[ap] || slices.ContainsFunc(w.engines, func(e *Engine) bool { return e.listen == ap })
}

// TakePort has something other than an engine hold ap.
func (w *World) TakePort(ap netip.AddrPort) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.taken[ap] = true
}

// Move has the engine listening on from listen on to instead, as one on
// an unspecified address follows the host's default route: it sends no
// event, and only ListenAddr tells.
func (w *World) Move(from, to netip.AddrPort) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, e := range w.engines {
		if e.listen == from {
			e.listen = to
		}
	}
}

// Transfer has the swarm move a torrent with hash at these payload rates,
// down while it downloads a payload the World holds and up while it runs,
// each held to its engine's limit. 0 stalls it.
func (w *World) Transfer(h bencode.InfoHash, down, up int64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.each(func(t *torrent) bool { return t.meta.InfoHash == h }, func(t *torrent) { t.fold() })
	w.transfers[h] = rates{down, up}
	w.each(func(t *torrent) bool { return t.meta.InfoHash == h }, (*torrent).reschedule)
}

// DiskFull fills the disk every engine writes to, or frees it. A torrent
// that writes to a full disk sends FileError and pauses, as libtorrent's
// shim has it.
func (w *World) DiskFull(full bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.each(func(*torrent) bool { return true }, func(t *torrent) { t.fold() })
	w.full = full
	w.each(func(*torrent) bool { return true }, (*torrent).reschedule)
}

// Fail has every engine holding a torrent with hash fail it with err, as
// libtorrent fails a torrent it found an error in.
func (w *World) Fail(h bencode.InfoHash, err *engine.Error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.each(func(t *torrent) bool { return t.meta.InfoHash == h }, func(t *torrent) {
		t.e.emit(engine.TorrentFailed{Torrent: t.handle, Err: err})
	})
}

// Incoming has a peer at from connect to the engine listening on to. An
// engine reports it while it watches incoming peers and runs a torrent,
// since libtorrent turns a peer away before it reports it otherwise.
func (w *World) Incoming(to netip.AddrPort, from netip.Addr) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, e := range w.engines {
		running := slices.ContainsFunc(slices.Collect(maps.Values(e.torrents)), func(t *torrent) bool { return !t.paused })
		if e.listen == to && e.limits.WatchIncoming && running {
			e.emit(engine.IncomingPeer{Addr: from})
		}
	}
}

// each calls f on every torrent of an open engine that match picks, in
// handle order.
func (w *World) each(match func(*torrent) bool, f func(*torrent)) {
	for _, e := range w.engines {
		for _, h := range slices.Sorted(maps.Keys(e.torrents)) {
			if t := e.torrents[h]; match(t) {
				f(t)
			}
		}
	}
}

// Payload has a single-file torrent with hash finish at once with payload:
// a download of it writes the payload as its file.
func (w *World) Payload(h bencode.InfoHash, payload []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.payloads[h] = payload
}

// Open opens an engine in a World of its own.
func Open(cfg engine.Config) (engine.Engine, error) { return NewWorld().Open(cfg) }

// Payloads opens engines, each in a World of its own holding payloads.
func Payloads(payloads map[bencode.InfoHash][]byte) func(engine.Config) (engine.Engine, error) {
	return func(cfg engine.Config) (engine.Engine, error) {
		w := NewWorld()
		maps.Copy(w.payloads, payloads)
		return w.Open(cfg)
	}
}

type Engine struct {
	w *World
	// listen is the zero AddrPort while the engine does not listen.
	listen netip.AddrPort
	floor  time.Duration
	events chan engine.Event
	limits engine.Limits
	// sent and recv are the payload bytes moved since the engine opened.
	sent, recv int64

	closed  bool
	dropped bool
	// resync is set once events were dropped: the next PostUpdates carries
	// every torrent.
	resync   bool
	next     engine.Handle
	torrents map[engine.Handle]*torrent
	// parting holds the torrents removed while a tracker awaits their
	// started announce's answer, by info hash.
	parting map[bencode.InfoHash]*torrent
	// posted is each torrent's status as PostUpdates last sent it, its
	// clocks zeroed: libtorrent counts a torrent changed by what moved, not
	// by time passing.
	posted map[engine.Handle]engine.Status
}

type torrent struct {
	e        *Engine
	handle   engine.Handle
	meta     bencode.Metainfo
	raw      []byte
	savePath string
	files    []engine.File
	state    engine.State
	paused   bool
	// pausing is a pause waiting for a tracker's answer to started.
	pausing bool
	// removing is a removal waiting likewise, deleting files when
	// deleteFiles.
	removing, deleteFiles bool
	done, wanted          int64
	// totals hold the clocks up to since, which now extends.
	totals   engine.Totals
	since    time.Time
	needSave bool
	trackers *announcing
	peerID   string
	// lastPayload is when the torrent last downloaded payload.
	lastPayload time.Time
	// transfer fires when a paced download finishes.
	transfer *time.Timer
}

func (e *Engine) Add(p engine.AddParams) (engine.Handle, error) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if e.closed {
		return 0, engine.ErrClosed
	}
	t, err := e.torrent(p)
	if err != nil {
		return 0, err
	}
	if was, ok := e.parting[t.meta.InfoHash]; ok {
		// libtorrent hands back a torrent whose removal waits, and it stays.
		delete(e.parting, was.meta.InfoHash)
		was.removing = false
		e.torrents[was.handle] = was
		was.pause()
		return was.handle, nil
	}
	e.next++
	t.handle = e.next
	e.torrents[t.handle] = t
	e.emit(engine.Added{Torrent: t.handle})
	e.emit(engine.StateChanged{Torrent: t.handle, State: t.state})
	return t.handle, nil
}

// torrent is the paused torrent p adds.
func (e *Engine) torrent(p engine.AddParams) (*torrent, error) {
	r := resume{Metainfo: p.Metainfo}
	if p.Resume != nil {
		var err error
		if r, err = decodeResume(p.Resume); err != nil {
			return nil, err
		}
	}
	m, err := bencode.DecodeMetainfo(r.Metainfo)
	if err != nil {
		return nil, &engine.Error{Category: engine.CategoryBdecode, Text: err.Error()}
	}
	t := &torrent{
		e: e, meta: m, raw: r.Metainfo, savePath: p.SavePath, files: files(m.Info), paused: true, needSave: true,
		totals: engine.Totals{
			Uploaded:   max(r.Totals.Uploaded, p.Baseline.Uploaded),
			Downloaded: max(r.Totals.Downloaded, p.Baseline.Downloaded),
			Active:     max(r.Totals.Active, p.Baseline.Active),
			Seeding:    max(r.Totals.Seeding, p.Baseline.Seeding),
		},
		since: time.Now(),
	}
	for _, f := range t.files {
		t.wanted += f.Size
	}
	t.trackers = &announcing{t: t}
	e.w.adds++
	t.peerID = fmt.Sprintf("%s%012d", version.PeerIDPrefix, e.w.adds)
	switch {
	case p.SeedMode:
		t.done, t.state = t.wanted, engine.StateSeeding
	case p.Resume != nil:
		t.done, t.state = r.Done, engine.StateDownloading
		if t.done == t.wanted {
			t.state = engine.StateSeeding
		}
	case t.anyFile():
		t.state = engine.StateCheckingFiles
	default:
		t.state = engine.StateDownloading
	}
	// libtorrent takes a torrent added or checked complete to have sent
	// completed when it finished.
	t.trackers.completeSent = t.state == engine.StateSeeding
	return t, nil
}

func (t *torrent) anyFile() bool {
	return slices.ContainsFunc(t.files, func(f engine.File) bool {
		_, err := os.Stat(t.path(f))
		return err == nil
	})
}

func (t *torrent) path(f engine.File) string {
	return filepath.Join(t.savePath, filepath.FromSlash(f.Path))
}

func (e *Engine) Pause(h engine.Handle) error {
	return e.with(h, func(t *torrent) error {
		t.pause()
		return nil
	})
}

// pause pauses t now, or once no tracker awaits an answer.
func (t *torrent) pause() {
	t.pausing = true
	t.settle()
}

// settle carries out the pause or removal of t that waited for its
// trackers to answer, once none awaits an answer.
func (t *torrent) settle() {
	if t.trackers.awaitsAnswer() {
		return
	}
	if t.pausing {
		t.pausing = false
		if !t.paused {
			t.setPaused(true)
			t.reschedule()
			t.trackers.stop()
			t.e.emit(engine.Paused{Torrent: t.handle})
		}
	}
	if t.removing {
		t.removing = false
		delete(t.e.parting, t.meta.InfoHash)
		t.remove()
	}
}

// holds is whether t is the engine's to report on: libtorrent's shim drops
// the alerts of a torrent it removed.
func (e *Engine) holds(t *torrent) bool {
	return e.torrents[t.handle] == t || e.parting[t.meta.InfoHash] == t
}

func (e *Engine) Resume(h engine.Handle) error {
	return e.with(h, func(t *torrent) error {
		t.pausing = false
		if !t.paused {
			return nil
		}
		t.setPaused(false)
		t.e.emit(engine.Resumed{Torrent: t.handle})
		if t.state == engine.StateCheckingFiles {
			t.check()
		}
		t.trackers.start()
		t.reschedule()
		return nil
	})
}

// Recheck checks t's files at once while it stays paused: libtorrent's
// check runs the torrent, so it sends Resumed first, and pauses it again.
func (e *Engine) Recheck(h engine.Handle) error {
	return e.with(h, func(t *torrent) error {
		e.emit(engine.Resumed{Torrent: h})
		t.setState(engine.StateCheckingFiles)
		t.check()
		e.emit(engine.Checked{Torrent: h})
		e.emit(engine.Paused{Torrent: h})
		return nil
	})
}

func (e *Engine) Configure(l engine.Limits) error {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if e.closed {
		return engine.ErrClosed
	}
	for _, t := range e.torrents {
		t.fold()
	}
	e.limits = l
	for _, h := range slices.Sorted(maps.Keys(e.torrents)) {
		e.torrents[h].reschedule()
	}
	return nil
}

func (e *Engine) PostStats() error {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if e.closed {
		return engine.ErrClosed
	}
	for _, t := range e.torrents {
		t.fold()
	}
	e.emit(engine.Stats{SentPayload: e.sent, RecvPayload: e.recv, At: time.Now()})
	return nil
}

func (e *Engine) Version() engine.Version { return version }

// rates are the payload rates t moves at now.
func (t *torrent) rates() rates {
	if t.paused {
		return rates{}
	}
	tr := t.e.w.transfers[t.meta.InfoHash]
	r := rates{up: held(tr.up, t.e.limits.UploadLimit)}
	if _, ok := t.e.w.payloads[t.meta.InfoHash]; ok && t.state == engine.StateDownloading {
		r.down = held(tr.down, t.e.limits.DownloadLimit)
	}
	return r
}

// held is rate held to limit, 0 being no limit.
func held(rate, limit int64) int64 {
	if limit == 0 {
		return rate
	}
	return min(rate, limit)
}

// reschedule sets when t's download of the World's payload finishes: at
// once unless the World paces it, and never while the disk is full, which
// pauses it.
func (t *torrent) reschedule() {
	t.stopTransfer()
	payload, ok := t.e.w.payloads[t.meta.InfoHash]
	if !ok || t.paused || t.state != engine.StateDownloading {
		return
	}
	if t.e.w.full {
		t.e.emit(engine.FileError{Torrent: t.handle, Err: engine.ErrDiskFull})
		t.pause()
		return
	}
	if _, paced := t.e.w.transfers[t.meta.InfoHash]; !paced {
		t.finish(payload)
		return
	}
	down := t.rates().down
	if down == 0 {
		return
	}
	left := time.Duration(math.Ceil(float64(t.wanted-t.done) / float64(down) * float64(time.Second)))
	t.transfer = time.AfterFunc(left, func() {
		t.e.w.mu.Lock()
		defer t.e.w.mu.Unlock()
		if !t.e.closed && t.e.holds(t) {
			t.catchUp()
		}
	})
}

// catchUp folds t up to now and finishes a paced download that got all its
// payload, whether its timer or a reader came first.
func (t *torrent) catchUp() {
	t.fold()
	if t.state == engine.StateDownloading && t.done == t.wanted && !t.paused {
		t.finish(t.e.w.payloads[t.meta.InfoHash])
	}
}

func (t *torrent) stopTransfer() {
	if t.transfer != nil {
		t.transfer.Stop()
		t.transfer = nil
	}
}

// setPaused folds the clocks up to now before t stops or starts them.
func (t *torrent) setPaused(paused bool) {
	t.fold()
	t.paused, t.needSave = paused, true
}

func (t *torrent) setState(s engine.State) {
	t.fold()
	t.state, t.needSave = s, true
	t.e.emit(engine.StateChanged{Torrent: t.handle, State: s})
}

// fold moves t's payload and clocks up to now.
func (t *torrent) fold() {
	now := time.Now()
	if d := now.Sub(t.since).Seconds(); d > 0 {
		r := t.rates()
		down := min(t.wanted-t.done, int64(float64(r.down)*d))
		up := int64(float64(r.up) * d)
		t.done += down
		t.totals.Downloaded += down
		t.totals.Uploaded += up
		t.e.recv += down
		t.e.sent += up
		if down > 0 {
			t.lastPayload = now
		}
		if down > 0 || up > 0 {
			t.needSave = true
		}
	}
	if !t.paused {
		t.totals.Active += now.Sub(t.since)
		if t.state == engine.StateSeeding {
			t.totals.Seeding += now.Sub(t.since)
		}
	}
	t.since = now
}

// check hashes the files under t's save path, as libtorrent checks a
// torrent added without resume data once it runs.
func (t *torrent) check() {
	t.done = t.verified()
	if t.done < t.wanted {
		t.setState(engine.StateDownloading)
		return
	}
	t.setState(engine.StateSeeding)
	t.trackers.completeSent = true
	t.e.emit(engine.Finished{Torrent: t.handle})
}

// verified is the bytes of the pieces whose data under the save path
// matches their hash. A missing or short file reads as zeros.
func (t *torrent) verified() int64 {
	readers := make([]io.Reader, 0, len(t.files))
	var opened []*os.File
	defer func() {
		for _, fh := range opened {
			_ = fh.Close()
		}
	}()
	for _, f := range t.files {
		r := io.Reader(zeros{})
		if fh, err := os.Open(t.path(f)); err == nil {
			opened = append(opened, fh)
			r = io.MultiReader(fh, zeros{})
		}
		readers = append(readers, io.LimitReader(r, f.Size))
	}
	all := io.MultiReader(readers...)
	piece := make([]byte, t.meta.Info.PieceLength)
	var done int64
	for i := 0; ; i++ {
		n, err := io.ReadFull(all, piece)
		if n == 0 {
			return done
		}
		if sha1.Sum(piece[:n]) == [sha1.Size]byte(t.meta.Info.Pieces[i*sha1.Size:]) { //nolint:gosec // BitTorrent v1 piece hashes are SHA-1 by spec (BEP 3)
			done += int64(n)
		}
		if err != nil {
			return done
		}
	}
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

// finish writes payload as t's files, in order, and seeds.
func (t *torrent) finish(payload []byte) {
	for _, f := range t.files {
		data := payload[:min(f.Size, int64(len(payload)))]
		payload = payload[len(data):]
		if err := write(t.path(f), data); err != nil {
			t.e.emit(engine.FileError{Torrent: t.handle, Err: &engine.Error{Category: engine.CategorySystem, Text: err.Error()}})
			return
		}
	}
	t.totals.Downloaded += t.wanted - t.done
	t.e.recv += t.wanted - t.done
	t.done, t.lastPayload = t.wanted, time.Now()
	t.setState(engine.StateSeeding)
	t.e.emit(engine.Finished{Torrent: t.handle})
	t.trackers.completed()
}

func write(file string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil { //nolint:gosec // a download is as readable as libtorrent leaves it
		return fmt.Errorf("make the folder of %s: %w", file, err)
	}
	if err := os.WriteFile(file, data, 0o644); err != nil { //nolint:gosec // a download is as readable as libtorrent leaves it
		return fmt.Errorf("write %s: %w", file, err)
	}
	return nil
}

func (e *Engine) Reannounce(h engine.Handle, tracker int, delay time.Duration, ignoreMinInterval bool) error {
	return e.with(h, func(t *torrent) error {
		t.trackers.reannounce(tracker, delay, ignoreMinInterval)
		return nil
	})
}

func (e *Engine) PostTrackers(h engine.Handle) error {
	return e.with(h, func(t *torrent) error {
		e.emit(engine.TrackerList{Torrent: h, Trackers: t.trackers.list()})
		return nil
	})
}

func (e *Engine) SaveResume(h engine.Handle) error {
	return e.with(h, func(t *torrent) error {
		t.fold()
		data, err := resume{Metainfo: t.raw, Done: t.done, Totals: t.totals}.encode()
		if err != nil {
			return err
		}
		t.needSave = false
		e.emit(engine.ResumeSaved{Torrent: h, Data: data})
		return nil
	})
}

func (e *Engine) PostUpdates() error {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if e.closed {
		return engine.ErrClosed
	}
	var changed []engine.Status
	for _, h := range slices.Sorted(maps.Keys(e.torrents)) {
		s := e.torrents[h].status()
		moved := s
		moved.Totals.Active, moved.Totals.Seeding = 0, 0
		if last, ok := e.posted[h]; e.resync || !ok || last != moved {
			changed = append(changed, s)
			e.posted[h] = moved
		}
	}
	e.resync = false
	e.emit(engine.Updates{Statuses: changed})
	return nil
}

func (t *torrent) status() engine.Status {
	t.catchUp()
	r := t.rates()
	return engine.Status{
		Torrent: t.handle, State: t.state, Paused: t.paused, Done: t.done, Wanted: t.wanted,
		DownBps: r.down, UpBps: r.up, Totals: t.totals, LastPayload: t.lastPayload, NeedSave: t.needSave,
	}
}

func (e *Engine) ConnectPeer(h engine.Handle, _ netip.AddrPort) error {
	return e.with(h, func(*torrent) error { return nil })
}

// Status is h's status now, for a test to look into the fake: the port
// reads status only through PostUpdates.
func (e *Engine) Status(h engine.Handle) (engine.Status, error) {
	var s engine.Status
	err := e.with(h, func(t *torrent) error {
		s = t.status()
		return nil
	})
	return s, err
}

func (e *Engine) Files(h engine.Handle) ([]engine.File, error) {
	var out []engine.File
	err := e.with(h, func(t *torrent) error {
		out = slices.Clone(t.files)
		return nil
	})
	return out, err
}

// FileProgress spreads the bytes done over the files in order.
func (e *Engine) FileProgress(h engine.Handle) ([]int64, error) {
	var out []int64
	err := e.with(h, func(t *torrent) error {
		left := t.done
		for _, f := range t.files {
			done := min(left, f.Size)
			out = append(out, done)
			left -= done
		}
		return nil
	})
	return out, err
}

// files lists the metainfo's files as given: unlike libtorrent, the
// in-memory engine sanitizes no name.
func files(info bencode.Info) []engine.File {
	out := make([]engine.File, len(info.Files))
	for i, f := range info.Files {
		out[i] = engine.File{Index: i, Path: path.Join(append([]string{info.Name}, f.Path...)...), Size: f.Length}
	}
	return out
}

func (e *Engine) Remove(h engine.Handle, deleteFiles bool) error {
	return e.with(h, func(t *torrent) error {
		t.fold()
		t.stopTransfer()
		delete(e.torrents, h)
		delete(e.posted, h)
		t.pausing, t.removing, t.deleteFiles = false, true, deleteFiles
		e.parting[t.meta.InfoHash] = t
		t.settle()
		return nil
	})
}

func (t *torrent) remove() {
	t.trackers.stop()
	t.e.emit(engine.Removed{Torrent: t.handle})
	if t.deleteFiles {
		deleted := engine.FilesDeleted{Torrent: t.handle}
		if err := t.deleteData(); err != nil {
			deleted.Err = &engine.Error{Category: engine.CategorySystem, Text: err.Error()}
		}
		t.e.emit(deleted)
	}
}

// deleteData deletes t's files and the folders its file paths name, as
// libtorrent's delete_files does, then its save path if that is left
// empty.
func (t *torrent) deleteData() error {
	var errs []error
	dirs := map[string]bool{}
	for _, f := range t.files {
		if err := os.Remove(t.path(f)); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
		for d := path.Dir(f.Path); d != "."; d = path.Dir(d) {
			dirs[d] = true
		}
	}
	for _, d := range slices.SortedFunc(maps.Keys(dirs), func(a, b string) int { return cmp.Compare(len(b), len(a)) }) {
		if err := os.Remove(filepath.Join(t.savePath, filepath.FromSlash(d))); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if left, err := os.ReadDir(t.savePath); err == nil && len(left) == 0 {
		if err := os.Remove(t.savePath); err != nil {
			return fmt.Errorf("remove the emptied save path: %w", err)
		}
	}
	return nil
}

func (e *Engine) ListenAddr() (netip.AddrPort, error) {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if !e.listen.IsValid() {
		return netip.AddrPort{}, engine.ErrNotListening
	}
	return e.listen, nil
}

func (e *Engine) Events() <-chan engine.Event { return e.events }

// Close pauses every torrent as Pause does, so stopped goes out where
// started was answered once no announce awaits its answer.
func (e *Engine) Close() error {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if e.closed {
		return nil
	}
	e.w.engines = slices.DeleteFunc(e.w.engines, func(o *Engine) bool { return o == e })
	e.closed = true
	close(e.events)
	for _, t := range e.torrents {
		t.stopTransfer()
		t.pause()
	}
	return nil
}

func (e *Engine) with(h engine.Handle, f func(*torrent) error) error {
	e.w.mu.Lock()
	defer e.w.mu.Unlock()
	if e.closed {
		return engine.ErrClosed
	}
	t, ok := e.torrents[h]
	if !ok {
		return engine.ErrUnknownTorrent
	}
	return f(t)
}

// emit queues ev without blocking. Once a reader falls behind it drops
// events, and the next one it can queue is Dropped.
func (e *Engine) emit(ev engine.Event) {
	if e.closed {
		return
	}
	if e.dropped {
		select {
		case e.events <- engine.Dropped{}:
			e.dropped = false
		default:
			e.resync = true
			return
		}
	}
	select {
	case e.events <- ev:
	default:
		e.dropped, e.resync = true, true
	}
}

// resume is the in-memory engine's resume data.
type resume struct {
	Metainfo []byte
	Done     int64
	Totals   engine.Totals
}

func (r resume) encode() ([]byte, error) {
	return bencode.Encode(map[string]any{
		"metainfo": r.Metainfo, "done": r.Done,
		"uploaded": r.Totals.Uploaded, "downloaded": r.Totals.Downloaded,
		"active": int64(r.Totals.Active), "seeding": int64(r.Totals.Seeding),
	})
}

func decodeResume(data []byte) (resume, error) {
	v, err := bencode.Decode(data)
	d, ok := v.(map[string]any)
	if err != nil || !ok {
		return resume{}, &engine.Error{Category: engine.CategoryBdecode, Text: fmt.Sprintf("resume data: %v", cmp.Or(err, errors.New("not a dictionary")))}
	}
	meta, _ := d["metainfo"].(string)
	num := func(k string) int64 { n, _ := d[k].(int64); return n }
	return resume{
		Metainfo: []byte(meta),
		Done:     num("done"),
		Totals: engine.Totals{
			Uploaded: num("uploaded"), Downloaded: num("downloaded"),
			Active: time.Duration(num("active")), Seeding: time.Duration(num("seeding")),
		},
	}, nil
}
