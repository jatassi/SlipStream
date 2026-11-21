//go:build cgo

// Package fvad is libfvad, WebRTC's voice activity detector, vendored in
// libfvad/ and built through cgo, and the Mapper that builds Speech Maps
// with it. It is the only package that links it.
package fvad

// #include "libfvad/include/fvad.h"
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// VAD is one libfvad instance. It adapts to the audio it has heard, so one
// instance reads one stream from its start, in order.
type VAD struct{ c *C.Fvad }

func New(mode, rate int) (*VAD, error) {
	c := C.fvad_new()
	if c == nil {
		return nil, errors.New("allocate a voice detector")
	}
	v := &VAD{c}
	if C.fvad_set_mode(c, C.int(mode)) != 0 {
		v.Close()
		return nil, fmt.Errorf("%w: %d", ErrMode, mode)
	}
	if C.fvad_set_sample_rate(c, C.int(rate)) != 0 {
		v.Close()
		return nil, fmt.Errorf("%w: %d", ErrRate, rate)
	}
	return v, nil
}

// Speech reads the next frame: 10, 20 or 30 ms of samples at the rate.
func (v *VAD) Speech(frame []int16) (bool, error) {
	if len(frame) == 0 {
		return false, ErrFrame
	}
	switch C.fvad_process(v.c, (*C.int16_t)(unsafe.Pointer(&frame[0])), C.size_t(len(frame))) {
	case 1:
		return true, nil
	case 0:
		return false, nil
	}
	return false, fmt.Errorf("%w: %d samples", ErrFrame, len(frame))
}

func (v *VAD) Close() { C.fvad_free(v.c) }
