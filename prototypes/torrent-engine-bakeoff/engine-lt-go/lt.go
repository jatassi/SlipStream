package main

/*
#cgo CFLAGS: -I${SRCDIR}/../lt-shim
#cgo LDFLAGS: ${SRCDIR}/../lt-shim/liblts.a -ltorrent-rasterbar -lssl -lcrypto -lstdc++
#include <stdlib.h>
#include "lts.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

const (
	alertAdded    = C.LTS_ALERT_ADDED
	alertFinished = C.LTS_ALERT_FINISHED
	alertPaused   = C.LTS_ALERT_PAUSED
	alertResumed  = C.LTS_ALERT_RESUMED
	alertRemoved  = C.LTS_ALERT_REMOVED
	alertTrkReply = C.LTS_ALERT_TRACKER_REPLY
	alertTrkError = C.LTS_ALERT_TRACKER_ERROR
	alertError    = C.LTS_ALERT_ERROR
)

type session struct{ p *C.lts_session }

type torrent int64

type alert struct {
	Kind    int
	LtType  int
	Torrent torrent
	What    string
	Message string
}

func newSession(kv [][2]string) (*session, error) {
	n := len(kv)
	ptrSize := C.size_t(unsafe.Sizeof(uintptr(0)))
	ks := (**C.char)(C.malloc(C.size_t(n+1) * ptrSize))
	vs := (**C.char)(C.malloc(C.size_t(n+1) * ptrSize))
	defer C.free(unsafe.Pointer(ks))
	defer C.free(unsafe.Pointer(vs))
	kslice := unsafe.Slice(ks, n+1)
	vslice := unsafe.Slice(vs, n+1)
	for i, p := range kv {
		kslice[i] = C.CString(p[0])
		vslice[i] = C.CString(p[1])
		defer C.free(unsafe.Pointer(kslice[i]))
		defer C.free(unsafe.Pointer(vslice[i]))
	}
	var errBuf [256]C.char
	s := C.lts_session_new(C.LTS_PRESET_SEEDBOX, ks, vs, C.int(n), &errBuf[0], C.int(len(errBuf)))
	if s == nil {
		return nil, errors.New(C.GoString(&errBuf[0]))
	}
	return &session{s}, nil
}

func (s *session) add(torrentFile, savePath string, seedMode bool) (torrent, error) {
	tf := C.CString(torrentFile)
	sp := C.CString(savePath)
	defer C.free(unsafe.Pointer(tf))
	defer C.free(unsafe.Pointer(sp))
	sm := C.int(0)
	if seedMode {
		sm = 1
	}
	var errBuf [256]C.char
	t := C.lts_add(s.p, tf, sp, sm, &errBuf[0], C.int(len(errBuf)))
	if t < 0 {
		return 0, errors.New(C.GoString(&errBuf[0]))
	}
	return torrent(t), nil
}

func check(rc C.int, op string) error {
	if rc != 0 {
		return errors.New(op + " failed")
	}
	return nil
}

func (s *session) pause(t torrent) error { return check(C.lts_pause(s.p, C.lts_torrent(t)), "pause") }
func (s *session) resume(t torrent) error {
	return check(C.lts_resume(s.p, C.lts_torrent(t)), "resume")
}
func (s *session) remove(t torrent) error {
	return check(C.lts_remove(s.p, C.lts_torrent(t)), "remove")
}

func (s *session) status(t torrent) (C.lts_status, error) {
	var st C.lts_status
	return st, check(C.lts_status_get(s.p, C.lts_torrent(t), &st), "status")
}

func (s *session) poll(buf []C.lts_alert, timeoutMs int) []alert {
	n := int(C.lts_poll_alerts(s.p, &buf[0], C.int(len(buf)), C.int(timeoutMs)))
	out := make([]alert, 0, max(n, 0))
	for i := 0; i < n; i++ {
		a := &buf[i]
		out = append(out, alert{int(a._type), int(a.lt_type), torrent(a.torrent), C.GoString(&a.what[0]), C.GoString(&a.message[0])})
	}
	return out
}

func (s *session) close() { C.lts_close(s.p) }

type ltAlertBuf = C.lts_alert
