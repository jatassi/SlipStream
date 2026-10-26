// Package libtorrent implements engine.Engine over libtorrent-rasterbar
// through the C shim in shim.h. Build it with the native bundle's cgo
// environment: eval "$(scripts/native-deps env)".
package libtorrent

/*
#cgo CXXFLAGS: -std=c++17
#include <stdlib.h>
#include "shim.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/jatassi/slipstream/internal/bencode"
	"github.com/jatassi/slipstream/internal/torrent/engine"
)

const pollMillis = 100

var _ engine.Engine = (*Session)(nil)

// Session owns one libtorrent session and the single goroutine that pumps its
// alerts. mu guards closed: every call into the shim holds it for reading, so
// Close, which holds it for writing, never frees the session under a call.
type Session struct {
	ses *C.lts_session
	// version is the identity the session announces with, read from it
	// once: nothing sets it.
	version engine.Version
	// listen is the address the listener is on, invalid while there is no
	// route to listen on. Only the pump moves it.
	listen atomic.Pointer[netip.Addr]
	port   uint16
	// route is the address to listen on, checked each routeCheck, for a
	// session on an unspecified address; nil for one on a named address.
	route      func() (netip.Addr, error)
	routeCheck time.Duration
	// rand draws the ports follow tries for a session that opened without
	// a route on port 0.
	rand   *rand.Rand
	events chan engine.Event
	stop   chan struct{}
	done   chan struct{}

	mu     sync.RWMutex
	closed bool
}

// Version is the linked libtorrent's version as major.minor.tiny, the form
// its releases are named by. libtorrent itself appends a fourth number.
func Version() string {
	parts := strings.SplitN(C.GoString(C.lts_version()), ".", 4)
	return strings.Join(parts[:3], ".")
}

// anyPortTries bounds the ports Open tries for port 0. With a fraction f of
// ports refused, all of them fail with probability f^anyPortTries.
const anyPortTries = 32

// Open returns a session on cfg's listen interface, or for an unspecified
// address on the address the default route leaves from. A session given a
// port binds that port or none, and then sends ListenFailed. For port 0
// Open tries random dynamic ports until one takes both libtorrent's TCP
// listener and its uTP socket, or returns the reason the last one did not.
// Windows refuses either bind with WSAEACCES on a port in one of its
// excluded ranges, which span runs of adjacent ports, so candidates from an
// allocator that walks ports in sequence fail together.
func Open(cfg engine.Config) (*Session, error) {
	var route func() (netip.Addr, error)
	if family := cfg.ListenInterface.Addr(); family.IsUnspecified() {
		route = func() (netip.Addr, error) { return defaultRoute(family) }
	}
	return openRouted(cfg, route)
}

func openRouted(cfg engine.Config, route func() (netip.Addr, error)) (*Session, error) {
	ap := cfg.ListenInterface
	var unrouted error
	if route != nil {
		addr, err := route()
		ap, unrouted = netip.AddrPortFrom(addr, ap.Port()), err
	}
	var s *Session
	var err error
	if ap.Port() != 0 || unrouted != nil {
		s, err = open(ap, cfg)
	} else {
		s, err = firstListening(dynamicPorts(cfg.Rand, anyPortTries), func(port uint16) (*Session, error) {
			return openListening(netip.AddrPortFrom(ap.Addr(), port), cfg)
		})
	}
	if err != nil {
		return nil, err
	}
	s.route, s.routeCheck, s.rand = route, cfg.RouteCheck, cfg.Rand
	if unrouted != nil {
		s.events <- engine.ListenFailed{Err: noRoute(unrouted)}
	}
	go s.pump()
	return s, nil
}

// defaultRoute is the address the host reaches the internet from in
// family's family, the source of its default route. It connects a UDP
// socket to a.root-servers.net, whose addresses IANA keeps, which sends
// nothing: the host only picks the route and the source address. So the
// root server is no endpoint, and Fake worlds need no stand-in for it.
func defaultRoute(family netip.Addr) (netip.Addr, error) {
	internet := netip.MustParseAddr("198.41.0.4")
	if family.Is6() {
		internet = netip.MustParseAddr("2001:503:ba3e::2:30")
	}
	conn, err := net.DialUDP("udp", nil, net.UDPAddrFromAddrPort(netip.AddrPortFrom(internet, 53)))
	if err != nil {
		return netip.Addr{}, err //nolint:wrapcheck // noRoute names it
	}
	defer func() { _ = conn.Close() }()
	local, err := netip.ParseAddrPort(conn.LocalAddr().String())
	return local.Addr().Unmap(), err
}

func noRoute(err error) *engine.Error {
	return &engine.Error{Category: engine.ErrNoRoute.Category, Code: engine.ErrNoRoute.Code, Text: fmt.Sprintf("%s: %v", engine.ErrNoRoute.Text, err)}
}

// firstListening opens on each port in turn and returns the first session
// that listens, or the last error. Only a refused listen moves on.
func firstListening[S any](ports []uint16, open func(port uint16) (S, error)) (S, error) {
	var err error
	for _, port := range ports {
		var s S
		if s, err = open(port); !errors.Is(err, engine.ErrNotListening) {
			return s, err
		}
	}
	var none S
	return none, err
}

func dynamicPorts(r *rand.Rand, n int) []uint16 {
	const first, count = 49152, 65536 - 49152
	ports := make([]uint16, n)
	for i := range ports {
		ports[i] = uint16(first + r.IntN(count))
	}
	return ports
}

// open opens a session on ap, or one that listens nowhere for an invalid
// address.
func open(ap netip.AddrPort, cfg engine.Config) (*Session, error) {
	addr := C.CString(addressText(ap.Addr()))
	defer C.free(unsafe.Pointer(addr))
	routers := make([]string, len(cfg.DHTRouters))
	for i, r := range cfg.DHTRouters {
		routers[i] = r.String()
	}
	dhtRouters := C.CString(strings.Join(routers, ","))
	defer C.free(unsafe.Pointer(dhtRouters))
	var cerr C.lts_error
	ses := C.lts_open(addr, C.uint16_t(ap.Port()), dhtRouters, C.int32_t(cfg.AnnounceFloor/time.Second), cBool(cfg.PeersShareAddress), C.int32_t(bencode.MaxTorrent), &cerr)
	if ses == nil {
		return nil, goError(&cerr)
	}
	var fingerprint [32]C.char
	var userAgent [128]C.char
	if C.lts_identity(ses, &fingerprint[0], C.size_t(len(fingerprint)), &userAgent[0], C.size_t(len(userAgent)), &cerr) != 0 {
		err := goError(&cerr)
		C.lts_close(ses, &cerr)
		return nil, err
	}
	s := &Session{
		ses:     ses,
		version: engine.Version{Libtorrent: Version(), PeerIDPrefix: C.GoString(&fingerprint[0]), UserAgent: C.GoString(&userAgent[0])},
		port:    ap.Port(),
		events:  make(chan engine.Event, 64),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	listen := ap.Addr()
	s.listen.Store(&listen)
	return s, nil
}

func addressText(a netip.Addr) string {
	if !a.IsValid() {
		return ""
	}
	return a.String()
}

// openListening opens a session that listens on ap, or closes it and says
// why it does not. It runs before the pump starts, so it reads the
// session's alerts itself.
func openListening(ap netip.AddrPort, cfg engine.Config) (*Session, error) {
	s, err := open(ap, cfg)
	if err != nil {
		return nil, err
	}
	_, err = s.ListenAddr()
	if err == nil {
		return s, nil
	}
	if errors.Is(err, engine.ErrNotListening) {
		err = fmt.Errorf("%w on %s", engine.ErrNotListening, ap)
		reason := listenFailure(s.ses)
		if reason != nil {
			err = fmt.Errorf("%w: %w", err, reason)
		}
	}
	var cerr C.lts_error
	C.lts_close(s.ses, &cerr)
	return nil, err
}

// listenFailure finds the listen_failed alert libtorrent queued while it
// opened the session, before lts_listen_port's call ran on its network thread.
func listenFailure(ses *C.lts_session) error {
	var buf [8]C.lts_alert
	var cerr C.lts_error
	n := int(C.lts_wait_alerts(ses, 0, &buf[0], C.int(len(buf)), &cerr)) //nolint:gocritic // cgo's pointer check repeats &buf[0]
	if n < 0 {
		return goError(&cerr)
	}
	var failed error
	for i := range n {
		if buf[i].kind == C.LTS_ALERT_LISTEN_FAILED && failed == nil {
			failed = goError(&buf[i].error)
		}
		C.lts_free_alert(&buf[i]) //nolint:gocritic // cgo's pointer check repeats &buf[i]
	}
	return failed
}

func (s *Session) Events() <-chan engine.Event { return s.events }

func (s *Session) ListenAddr() (netip.AddrPort, error) {
	var port C.uint16_t
	err := s.call(func(cerr *C.lts_error) C.int { return C.lts_listen_port(s.ses, &port, cerr) })
	if err != nil {
		return netip.AddrPort{}, err
	}
	if port == 0 {
		return netip.AddrPort{}, engine.ErrNotListening
	}
	return netip.AddrPortFrom(*s.listen.Load(), uint16(port)), nil
}

// follow moves the listener to the address the default route leaves from
// once the route moved, on the port the session opened on. A session that
// opened without a route has no port yet and tries ports as Open does. A
// host that lost its route keeps the listener where it is, and a move whose
// listen failed is tried again at the next check.
func (s *Session) follow() {
	to, err := s.route()
	if err != nil || to == *s.listen.Load() {
		return
	}
	ports := []uint16{s.port}
	if s.port == 0 {
		ports = dynamicPorts(s.rand, anyPortTries)
	}
	port, err := firstListening(ports, func(port uint16) (uint16, error) { return port, s.listenOn(to, port) })
	if err != nil {
		var failed *engine.Error
		if !errors.Is(err, engine.ErrNotListening) && errors.As(err, &failed) {
			s.send(engine.ListenFailed{Err: failed})
		}
		return
	}
	s.port = port
	s.listen.Store(&to)
}

// listenOn moves the listener to at and port, and is ErrNotListening when
// libtorrent could not bind there. Its listen_failed alert says why.
func (s *Session) listenOn(at netip.Addr, port uint16) error {
	addr := C.CString(at.String())
	defer C.free(unsafe.Pointer(addr))
	var cerr C.lts_error
	if C.lts_listen_on(s.ses, addr, C.uint16_t(port), &cerr) != 0 {
		return goError(&cerr)
	}
	_, err := s.ListenAddr()
	return err
}

func (s *Session) Add(p engine.AddParams) (engine.Handle, error) {
	savePath := C.CString(p.SavePath)
	defer C.free(unsafe.Pointer(savePath))
	var pins runtime.Pinner
	defer pins.Unpin()
	params := C.lts_add_params{
		metainfo:        pinned(&pins, p.Metainfo),
		metainfo_len:    C.size_t(len(p.Metainfo)),
		resume:          pinned(&pins, p.Resume),
		resume_len:      C.size_t(len(p.Resume)),
		save_path:       savePath,
		seed_mode:       cBool(p.SeedMode),
		discover:        cBool(p.Discover),
		base_uploaded:   C.int64_t(p.Baseline.Uploaded),
		base_downloaded: C.int64_t(p.Baseline.Downloaded),
		base_active_s:   C.int64_t(p.Baseline.Active / time.Second),
		base_seeding_s:  C.int64_t(p.Baseline.Seeding / time.Second),
	}
	var id C.int64_t
	err := s.call(func(cerr *C.lts_error) C.int { return C.lts_add(s.ses, &params, &id, cerr) }) //nolint:gocritic // cgo's pointer check repeats &params
	return engine.Handle(id), err
}

func (s *Session) FetchMetadata(p engine.MetadataParams) (engine.Handle, error) {
	uri := C.CString(p.Link.String())
	defer C.free(unsafe.Pointer(uri))
	savePath := C.CString(p.SavePath)
	defer C.free(unsafe.Pointer(savePath))
	var id C.int64_t
	err := s.call(func(cerr *C.lts_error) C.int {
		return C.lts_fetch_metadata(s.ses, uri, savePath, cBool(p.Discover), &id, cerr)
	})
	return engine.Handle(id), err
}

// pinned is b for C to read through a Go struct, which cgo allows only for
// pinned memory, and NULL for none.
func pinned(pins *runtime.Pinner, b []byte) *C.char {
	if len(b) == 0 {
		return nil
	}
	pins.Pin(unsafe.SliceData(b))
	return (*C.char)(unsafe.Pointer(unsafe.SliceData(b)))
}

func (s *Session) Pause(h engine.Handle) error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_pause(s.ses, C.int64_t(h), cerr) })
}

func (s *Session) Resume(h engine.Handle) error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_resume(s.ses, C.int64_t(h), cerr) })
}

func (s *Session) SaveResume(h engine.Handle) error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_save_resume(s.ses, C.int64_t(h), cerr) })
}

func (s *Session) Recheck(h engine.Handle) error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_recheck(s.ses, C.int64_t(h), cerr) })
}

func (s *Session) Configure(l engine.Limits) error {
	limits := C.lts_limits{
		download_limit: C.int64_t(l.DownloadLimit),
		upload_limit:   C.int64_t(l.UploadLimit),
		connections:    C.int32_t(l.Connections),
		utp:            cBool(l.UTP),
		watch_incoming: cBool(l.WatchIncoming),
		dht:            cBool(l.DHT),
	}
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_configure(s.ses, &limits, cerr) })
}

func (s *Session) Version() engine.Version { return s.version }

func (s *Session) Reannounce(h engine.Handle, tracker int, delay time.Duration, ignoreMinInterval bool) error {
	return s.call(func(cerr *C.lts_error) C.int {
		return C.lts_reannounce(s.ses, C.int64_t(h), C.int32_t(tracker), C.int32_t(delay/time.Second), cBool(ignoreMinInterval), cerr)
	})
}

func (s *Session) PostTrackers(h engine.Handle) error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_post_trackers(s.ses, C.int64_t(h), cerr) })
}

func (s *Session) PostUpdates() error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_post_updates(s.ses, cerr) })
}

func (s *Session) PostStats() error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_post_stats(s.ses, cerr) })
}

func (s *Session) ConnectPeer(h engine.Handle, peer netip.AddrPort) error {
	addr := C.CString(peer.Addr().String())
	defer C.free(unsafe.Pointer(addr))
	return s.call(func(cerr *C.lts_error) C.int {
		return C.lts_connect_peer(s.ses, C.int64_t(h), addr, C.uint16_t(peer.Port()), cerr)
	})
}

func goStatus(st *C.lts_status) engine.Status {
	out := engine.Status{
		Torrent: engine.Handle(st.torrent),
		State:   states[st.state],
		Paused:  st.paused != 0,
		Done:    int64(st.done),
		Wanted:  int64(st.wanted),
		DownBps: int64(st.down_bps),
		UpBps:   int64(st.up_bps),
		Peers:   int(st.peers),
		Seeds:   int(st.seeds),
		Totals: engine.Totals{
			Uploaded:   int64(st.uploaded),
			Downloaded: int64(st.downloaded),
			Active:     time.Duration(st.active_s) * time.Second,
			Seeding:    time.Duration(st.seeding_s) * time.Second,
		},
		NeedSave: st.need_save != 0,
	}
	out.LastPayload = unixMilli(st.last_download_ms)
	if st.error.category != 0 {
		out.Err = goError(&st.error)
	}
	return out
}

func (s *Session) Files(h engine.Handle) ([]engine.File, error) {
	var files *C.lts_file
	var count C.int32_t
	err := s.call(func(cerr *C.lts_error) C.int {
		return C.lts_torrent_files(s.ses, C.int64_t(h), &files, &count, cerr) //nolint:gocritic // cgo's pointer check repeats &files
	})
	if err != nil {
		return nil, err
	}
	defer C.lts_free_files(files, count)
	out := make([]engine.File, int(count))
	for i, f := range unsafe.Slice(files, int(count)) {
		out[i] = engine.File{Index: int(f.index), Path: filepath.ToSlash(C.GoString(f.path)), Size: int64(f.size)}
	}
	return out, nil
}

func (s *Session) FileProgress(h engine.Handle) ([]int64, error) {
	var done *C.int64_t
	var count C.int32_t
	err := s.call(func(cerr *C.lts_error) C.int {
		return C.lts_file_progress(s.ses, C.int64_t(h), &done, &count, cerr) //nolint:gocritic // cgo's pointer check repeats &done
	})
	if err != nil {
		return nil, err
	}
	defer C.lts_free(unsafe.Pointer(done))
	out := make([]int64, int(count))
	for i, d := range unsafe.Slice(done, int(count)) {
		out[i] = int64(d)
	}
	return out, nil
}

func (s *Session) Remove(h engine.Handle, deleteFiles bool) error {
	return s.call(func(cerr *C.lts_error) C.int {
		return C.lts_remove(s.ses, C.int64_t(h), cBool(deleteFiles), cerr)
	})
}

// Close stops and joins the alert pump before it closes the session, so no
// poll can race the free.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return engine.ErrClosed
	}
	s.closed = true
	close(s.stop)
	<-s.done
	close(s.events)
	var cerr C.lts_error
	if C.lts_close(s.ses, &cerr) != 0 {
		return goError(&cerr)
	}
	return nil
}

func (s *Session) call(f func(*C.lts_error) C.int) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed {
		return engine.ErrClosed
	}
	var cerr C.lts_error
	if f(&cerr) != 0 {
		return goError(&cerr)
	}
	return nil
}

// pump blocks on a slow reader instead of buffering without bound. libtorrent
// then fills its own alert queue and reports the overflow as Dropped. Each
// batch is converted, and its shim buffers freed, before any of it is sent,
// so a stop mid-batch leaks nothing.
func (s *Session) pump() {
	defer close(s.done)
	var buf [64]C.lts_alert
	events := make([]engine.Event, 0, len(buf))
	checkRoute := time.Now().Add(s.routeCheck)
	for {
		select {
		case <-s.stop:
			return
		default:
		}
		if s.route != nil && s.routeCheck > 0 && !time.Now().Before(checkRoute) {
			s.follow()
			checkRoute = time.Now().Add(s.routeCheck)
		}
		var cerr C.lts_error
		n := int(C.lts_wait_alerts(s.ses, pollMillis, &buf[0], C.int(len(buf)), &cerr)) //nolint:gocritic // cgo's pointer check repeats &buf[0]
		if n < 0 {
			// The shim lost the batch it was converting, which is a drop.
			if !s.send(engine.Dropped{}) {
				return
			}
			continue
		}
		events = events[:0]
		for i := range n {
			events = append(events, goEvent(&buf[i]))
			C.lts_free_alert(&buf[i]) //nolint:gocritic // cgo's pointer check repeats &buf[i]
		}
		for _, ev := range events {
			if !s.send(ev) {
				return
			}
		}
	}
}

func (s *Session) send(ev engine.Event) bool {
	select {
	case s.events <- ev:
		return true
	case <-s.stop:
		return false
	}
}

// alertTime is the alert posted at Unix milliseconds ms, read on the
// monotonic clock: the announce policy measures a tracker's intervals from
// it, and the wall clock may be slewed or stepped meanwhile.
func alertTime(ms C.int64_t) time.Time {
	now := time.Now()
	return now.Add(time.UnixMilli(int64(ms)).Sub(now))
}

func goEvent(a *C.lts_alert) engine.Event {
	h := engine.Handle(a.torrent)
	switch a.kind {
	case C.LTS_ALERT_ADDED:
		return engine.Added{Torrent: h}
	case C.LTS_ALERT_STATE_CHANGED:
		return engine.StateChanged{Torrent: h, State: states[a.state]}
	case C.LTS_ALERT_FINISHED:
		return engine.Finished{Torrent: h}
	case C.LTS_ALERT_REMOVED:
		return engine.Removed{Torrent: h}
	case C.LTS_ALERT_TORRENT_FAILED:
		return engine.TorrentFailed{Torrent: h, Err: goError(&a.error)}
	case C.LTS_ALERT_TRACKER_ANNOUNCED:
		return engine.TrackerAnnounced{Torrent: h, Tracker: int(a.tracker), Event: announceEvents[a.event], At: alertTime(a.at_ms)}
	case C.LTS_ALERT_TRACKER_REPLIED:
		return engine.TrackerReplied{Torrent: h, Tracker: int(a.tracker), At: alertTime(a.at_ms)}
	case C.LTS_ALERT_TRACKER_WARNED:
		return engine.TrackerWarned{Torrent: h, Tracker: int(a.tracker), Message: payloadText(a)}
	case C.LTS_ALERT_TRACKER_FAILED:
		return engine.TrackerFailed{
			Torrent: h, Tracker: int(a.tracker), Times: int(a.times), Reason: payloadText(a), Err: goError(&a.error),
			At: alertTime(a.at_ms),
		}
	case C.LTS_ALERT_TRACKER_LIST:
		trs := unsafe.Slice((*C.lts_tracker)(a.payload), int(a.payload_len))
		out := make([]engine.Tracker, len(trs))
		for i := range trs {
			out[i] = goTracker(i, &trs[i])
		}
		return engine.TrackerList{Torrent: h, Trackers: out}
	case C.LTS_ALERT_LISTEN_FAILED:
		return engine.ListenFailed{Err: goError(&a.error)}
	case C.LTS_ALERT_DROPPED:
		return engine.Dropped{}
	case C.LTS_ALERT_PAUSED:
		return engine.Paused{Torrent: h}
	case C.LTS_ALERT_RESUMED:
		return engine.Resumed{Torrent: h}
	case C.LTS_ALERT_RESUME_SAVED:
		return engine.ResumeSaved{Torrent: h, Data: C.GoBytes(a.payload, C.int(a.payload_len))}
	case C.LTS_ALERT_RESUME_FAILED:
		return engine.ResumeFailed{Torrent: h, Err: goError(&a.error)}
	case C.LTS_ALERT_FILES_DELETED:
		deleted := engine.FilesDeleted{Torrent: h}
		if a.error.category != 0 {
			deleted.Err = goError(&a.error)
		}
		return deleted
	case C.LTS_ALERT_UPDATES:
		sts := unsafe.Slice((*C.lts_status)(a.payload), int(a.payload_len))
		out := make([]engine.Status, len(sts))
		for i := range sts {
			out[i] = goStatus(&sts[i])
		}
		return engine.Updates{Statuses: out}
	case C.LTS_ALERT_CHECKED:
		return engine.Checked{Torrent: h}
	case C.LTS_ALERT_INCOMING_PEER:
		return engine.IncomingPeer{Addr: netip.MustParseAddr(payloadText(a)).Unmap()}
	case C.LTS_ALERT_FILE_ERROR:
		return engine.FileError{Torrent: h, Err: goError(&a.error)}
	case C.LTS_ALERT_HASH_FAILED:
		return engine.HashFailed{Torrent: h}
	case C.LTS_ALERT_METADATA_RECEIVED:
		return engine.MetadataReceived{Torrent: h, Info: C.GoBytes(a.payload, C.int(a.payload_len))}
	case C.LTS_ALERT_METADATA_TOO_LARGE:
		return engine.MetadataTooLarge{Torrent: h, Size: int64(a.payload_len)}
	case C.LTS_ALERT_STATS:
		counters := unsafe.Slice((*C.int64_t)(a.payload), 2)
		return engine.Stats{SentPayload: int64(counters[0]), RecvPayload: int64(counters[1]), At: alertTime(a.at_ms)}
	}
	panic("libtorrent: shim sent unknown alert kind")
}

func payloadText(a *C.lts_alert) string {
	return string(C.GoBytes(a.payload, C.int(a.payload_len)))
}

func goTracker(index int, t *C.lts_tracker) engine.Tracker {
	out := engine.Tracker{
		Index:        index,
		URL:          C.GoString(t.url),
		Tier:         int(t.tier),
		Endpoints:    int(t.endpoints),
		NextAnnounce: unixMilli(t.next_announce_ms),
		MinAnnounce:  unixMilli(t.min_announce_ms),
		Message:      C.GoString(t.message),
		Fails:        int(t.fails),
		Updating:     t.updating != 0,
		StartSent:    t.start_sent != 0,
	}
	if t.last_error.category != 0 {
		out.LastError = goError(&t.last_error)
	}
	return out
}

// unixMilli is the time ms Unix milliseconds name, zero for 0.
func unixMilli(ms C.int64_t) time.Time {
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(int64(ms))
}

var announceEvents = map[C.int32_t]engine.AnnounceEvent{
	C.LTS_EVENT_NONE:      engine.AnnounceNone,
	C.LTS_EVENT_COMPLETED: engine.AnnounceCompleted,
	C.LTS_EVENT_STARTED:   engine.AnnounceStarted,
	C.LTS_EVENT_STOPPED:   engine.AnnounceStopped,
}

var categories = map[C.int32_t]engine.Category{
	C.LTS_CATEGORY_SHIM:       engine.CategoryShim,
	C.LTS_CATEGORY_SYSTEM:     engine.CategorySystem,
	C.LTS_CATEGORY_LIBTORRENT: engine.CategoryLibtorrent,
	C.LTS_CATEGORY_BDECODE:    engine.CategoryBdecode,
	C.LTS_CATEGORY_OTHER:      engine.CategoryOther,
	C.LTS_CATEGORY_EXCEPTION:  engine.CategoryException,
}

var states = map[C.int32_t]engine.State{
	C.LTS_STATE_CHECKING_FILES:       engine.StateCheckingFiles,
	C.LTS_STATE_DOWNLOADING_METADATA: engine.StateDownloadingMetadata,
	C.LTS_STATE_DOWNLOADING:          engine.StateDownloading,
	C.LTS_STATE_FINISHED:             engine.StateFinished,
	C.LTS_STATE_SEEDING:              engine.StateSeeding,
	C.LTS_STATE_CHECKING_RESUME_DATA: engine.StateCheckingResumeData,
}

func goError(e *C.lts_error) *engine.Error {
	return &engine.Error{
		Category: categories[e.category],
		Code:     int(e.code),
		Text:     C.GoString(&e.text[0]),
	}
}

func cBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}
