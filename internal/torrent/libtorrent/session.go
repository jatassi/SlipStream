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
	"net/netip"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/jatassi/slipstream/internal/torrent/engine"
)

const pollMillis = 100

var _ engine.Engine = (*Session)(nil)

// Session owns one libtorrent session and the single goroutine that pumps its
// alerts. mu guards closed: every call into the shim holds it for reading, so
// Close, which holds it for writing, never frees the session under a call.
type Session struct {
	ses    *C.lts_session
	listen netip.Addr
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

// Open returns a session that listens, or the reason it could not. For port 0
// it tries random dynamic ports until one takes both libtorrent's TCP
// listener and its uTP socket, which libtorrent binds on the same port and
// retries only on address-in-use. Windows refuses either bind with WSAEACCES
// on a port in one of its excluded ranges, which span runs of adjacent
// ports, so candidates from an allocator that walks ports in sequence fail
// together.
func Open(cfg engine.Config) (*Session, error) {
	ports := []uint16{cfg.ListenInterface.Port()}
	if ports[0] == 0 {
		ports = dynamicPorts(cfg.Rand, anyPortTries)
	}
	s, err := firstListening(ports, func(port uint16) (*Session, error) {
		return openListening(netip.AddrPortFrom(cfg.ListenInterface.Addr(), port), cfg)
	})
	if err != nil {
		return nil, err
	}
	go s.pump()
	return s, nil
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

// openListening runs before the pump starts, so it reads the session's
// alerts itself to say why a session that does not listen was refused.
func openListening(ap netip.AddrPort, cfg engine.Config) (*Session, error) {
	addr := C.CString(ap.Addr().String())
	defer C.free(unsafe.Pointer(addr))
	var cerr C.lts_error
	ses := C.lts_open(addr, C.uint16_t(ap.Port()), discoveryBits(cfg.Discovery), cBool(cfg.OutgoingUTP), &cerr)
	if ses == nil {
		return nil, goError(&cerr)
	}
	var port C.uint16_t
	if C.lts_listen_port(ses, &port, &cerr) != 0 {
		err := goError(&cerr)
		C.lts_close(ses, &cerr)
		return nil, err
	}
	if port == 0 {
		err := fmt.Errorf("%w on %s", engine.ErrNotListening, ap)
		reason := listenFailure(ses)
		if reason != nil {
			err = fmt.Errorf("%w: %w", err, reason)
		}
		C.lts_close(ses, &cerr)
		return nil, err
	}
	return &Session{
		ses:    ses,
		listen: ap.Addr(),
		events: make(chan engine.Event, 64),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}, nil
}

// listenFailure finds the listen_failed alert libtorrent queued while it
// opened the session, before lts_listen_port's call ran on its network thread.
func listenFailure(ses *C.lts_session) error {
	var buf [8]C.lts_alert
	var cerr C.lts_error
	n := int(C.lts_wait_alerts(ses, 0, &buf[0], C.int(len(buf)), &cerr))
	if n < 0 {
		return goError(&cerr)
	}
	for i := range n {
		if buf[i].kind == C.LTS_ALERT_LISTEN_FAILED {
			return goError(&buf[i].error)
		}
	}
	return nil
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
	return netip.AddrPortFrom(s.listen, uint16(port)), nil
}

func (s *Session) Add(p engine.AddParams) (engine.Handle, error) {
	savePath := C.CString(p.SavePath)
	defer C.free(unsafe.Pointer(savePath))
	meta := (*C.char)(unsafe.Pointer(unsafe.SliceData(p.Metainfo)))
	var id C.int64_t
	err := s.call(func(cerr *C.lts_error) C.int {
		return C.lts_add(s.ses, meta, C.size_t(len(p.Metainfo)), savePath, cBool(p.SeedMode), &id, cerr)
	})
	return engine.Handle(id), err
}

func (s *Session) ConnectPeer(h engine.Handle, peer netip.AddrPort) error {
	addr := C.CString(peer.Addr().String())
	defer C.free(unsafe.Pointer(addr))
	return s.call(func(cerr *C.lts_error) C.int {
		return C.lts_connect_peer(s.ses, C.int64_t(h), addr, C.uint16_t(peer.Port()), cerr)
	})
}

func (s *Session) Status(h engine.Handle) (engine.Status, error) {
	var st C.lts_status
	err := s.call(func(cerr *C.lts_error) C.int { return C.lts_status_get(s.ses, C.int64_t(h), &st, cerr) })
	if err != nil {
		return engine.Status{}, err
	}
	return engine.Status{
		State:  states[st.state],
		Peers:  int(st.peers),
		Done:   int64(st.done),
		Wanted: int64(st.wanted),
	}, nil
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

func (s *Session) Remove(h engine.Handle) error {
	return s.call(func(cerr *C.lts_error) C.int { return C.lts_remove(s.ses, C.int64_t(h), cerr) })
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
// then fills its own alert queue and reports the overflow as Dropped.
func (s *Session) pump() {
	defer close(s.done)
	var buf [64]C.lts_alert
	for {
		select {
		case <-s.stop:
			return
		default:
		}
		var cerr C.lts_error
		n := int(C.lts_wait_alerts(s.ses, pollMillis, &buf[0], C.int(len(buf)), &cerr))
		if n < 0 {
			// The shim lost the batch it was converting, which is a drop.
			if !s.send(engine.Dropped{}) {
				return
			}
			continue
		}
		for i := range n {
			if !s.send(goEvent(&buf[i])) {
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
	case C.LTS_ALERT_TRACKER_REPLIED:
		return engine.TrackerReplied{Torrent: h}
	case C.LTS_ALERT_TRACKER_FAILED:
		return engine.TrackerFailed{Torrent: h, Err: goError(&a.error)}
	case C.LTS_ALERT_LISTEN_FAILED:
		return engine.ListenFailed{Err: goError(&a.error)}
	case C.LTS_ALERT_DROPPED:
		return engine.Dropped{}
	}
	panic("libtorrent: shim sent unknown alert kind")
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

func discoveryBits(d engine.Discovery) C.uint32_t {
	var bits C.uint32_t
	for _, b := range []struct {
		on  bool
		bit C.uint32_t
	}{
		{d.DHT, C.LTS_DISCOVERY_DHT},
		{d.LSD, C.LTS_DISCOVERY_LSD},
		{d.UPnP, C.LTS_DISCOVERY_UPNP},
		{d.NATPMP, C.LTS_DISCOVERY_NATPMP},
	} {
		if b.on {
			bits |= b.bit
		}
	}
	return bits
}

func cBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}
