package conversions

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/converting"
	"github.com/jatassi/slipstream/internal/converting/pipeline"
	"github.com/jatassi/slipstream/internal/encode"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/storage"
)

// The caps on what a Worker reports, applied before any Write.
const (
	tailCap      = 2 << 10
	maxBytes     = int64(1) << 50
	maxSpeed     = 1000.0
	minTestSpeed = 0.01
	maxDevices   = 8
	nameCap      = 128
	maxTries     = 8
	maxVolume    = int64(1) << 60
)

var errRefused = errors.New("the report is outside its caps")

// eventOf is a holder's report as an event, refused when it names no
// Conversion, carries no result or an output past its cap. An unknown
// fault code reads as crash.
func eventOf(r *cv1.Result) (Event, error) {
	id, err := storage.UUIDFromBytes(r.GetConversionId())
	if err != nil {
		return Event{}, fmt.Errorf("%w: conversion id: %w", errRefused, err)
	}
	ev := Event{Conversion: ID{UUID: id}}
	switch res := r.GetResult().(type) {
	case *cv1.Result_Started:
		ev.On = Started
	case *cv1.Result_Paused:
		ev.On = Paused
	case *cv1.Result_Resumed:
		ev.On = Resumed
	case *cv1.Result_Lost:
		ev.On = Lost
	case *cv1.Result_DiskFull:
		ev.On = DiskFull
	case *cv1.Result_Exited:
		b := res.Exited.GetBytes()
		if b < 0 || b > maxBytes {
			return Event{}, fmt.Errorf("%w: an output of %d bytes", errRefused, b)
		}
		ev.On, ev.Bytes = Exited, b
	case *cv1.Result_Fault:
		code := Code(res.Fault.GetCode())
		if _, ok := faults[code]; !ok {
			code = CodeCrash
		}
		ev.On, ev.Code, ev.Tail = Fault, code, capTail(res.Fault.GetTail())
		if faults[code].Class == PutBackClass {
			ev.On, ev.Code = DiskFull, ""
		}
	default:
		return Event{}, fmt.Errorf("%w: no result", errRefused)
	}
	return ev, nil
}

// capTail is the last 2 KiB of s, as valid UTF-8.
func capTail(s string) string {
	if len(s) > tailCap {
		s = s[len(s)-tailCap:]
		for s != "" && !utf8.RuneStart(s[0]) {
			s = s[1:]
		}
	}
	return strings.ToValidUTF8(s, "")
}

func capName(s string) string {
	if len(s) <= nameCap {
		return strings.ToValidUTF8(s, "")
	}
	s = s[:nameCap]
	for s != "" && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// signatureOf is sig when it is 64 hex characters.
func signatureOf(sig string) (converting.Signature, bool) {
	if len(sig) != 64 || strings.Trim(sig, "0123456789abcdef") != "" {
		return "", false
	}
	return converting.Signature(sig), true
}

// proofOf is a test run's answer from a Node on os, refused when its
// signature isn't one, a passing run names a pipeline os has no row for,
// or a failed run names no try.
func proofOf(res *cv1.TestResponse, os string) (converting.Proved, error) {
	sig, ok := signatureOf(res.GetSignature())
	if !ok {
		return converting.Proved{}, fmt.Errorf("%w: a signature of %d characters", errRefused, len(res.GetSignature()))
	}
	p := encode.Pipeline{Path: encode.Path(res.GetPipeline().GetPath()), API: encode.API(res.GetPipeline().GetApi())}
	if res.GetPassed() && !slices.ContainsFunc(pipeline.Rows, func(r pipeline.Row) bool { return r.Pipeline == p && slices.Contains(r.OS, os) }) {
		return converting.Proved{}, fmt.Errorf("%w: no %s pipeline %s %s", errRefused, os, p.Path, p.API)
	}
	if !res.GetPassed() && len(res.GetTries()) == 0 {
		return converting.Proved{}, fmt.Errorf("%w: a failed test run with no tries", errRefused)
	}
	out := converting.Proved{
		Passed: res.GetPassed(), Pipeline: p, Device: capName(res.GetDevice()), Pauses: res.GetPauses(), Signature: sig,
	}
	if out.Passed {
		out.Speed = min(max(res.GetSpeed(), minTestSpeed), maxSpeed)
	}
	for _, t := range res.GetTries()[:min(len(res.GetTries()), maxTries)] {
		out.Tries = append(out.Tries, converting.Try{
			Pipeline: encode.Pipeline{Path: encode.Path(t.GetPipeline().GetPath()), API: encode.API(t.GetPipeline().GetApi())},
			Err:      capTail(t.GetError()),
		})
	}
	return out, nil
}

// ScratchOf is f's scratch volume with its names cut to their caps, nil
// while it is unknown or its figures are past them.
func ScratchOf(f *cv1.Facts) *cv1.Volume {
	v := f.GetScratch()
	if v == nil || v.GetId() == "" || v.GetFree() < 0 || v.GetFree() > v.GetTotal() || v.GetTotal() > maxVolume {
		return nil
	}
	return &cv1.Volume{
		Free: v.GetFree(), Total: v.GetTotal(), Id: capName(v.GetId()), Name: capName(v.GetName()), Mount: capName(v.GetMount()),
		SampledAtUnixMs: v.GetSampledAtUnixMs(),
	}
}

// Facts are a Converting Node's facts within their caps.
type Facts struct {
	// Signature is empty when the Node's isn't one, so it qualifies for
	// nothing.
	Signature converting.Signature
	Devices   []converting.Device
	// Scratch is nil while the volume is unknown.
	Scratch  *Volume
	Progress map[ID]RunProgress
	Override bool
}

// RunProgress is one run's progress, for the job at Attempt.
type RunProgress struct {
	Attempt int
	Out     time.Duration
	Speed   float64
	Paused  bool
}

// factsOf caps f: at most atOnce+1 runs, each of a job in lengths at
// its attempt, with its output time within 1.5 times the job's length.
func factsOf(f *cv1.Facts, atOnce int, lengths map[ID]time.Duration) Facts {
	out := Facts{Override: f.GetOverride(), Progress: map[ID]RunProgress{}}
	out.Signature, _ = signatureOf(f.GetSignature())
	for _, d := range f.GetDevices()[:min(len(f.GetDevices()), maxDevices)] {
		out.Devices = append(out.Devices, converting.Device{Name: capName(d.GetName()), Vendor: capName(d.GetVendor()), Driver: capName(d.GetDriver())})
	}
	if v := ScratchOf(f); v != nil {
		out.Scratch = &Volume{Free: v.GetFree(), Total: v.GetTotal()}
	}
	for _, p := range f.GetProgress() {
		if len(out.Progress) > atOnce {
			break
		}
		u, err := storage.UUIDFromBytes(p.GetConversionId())
		length, ok := lengths[ID{UUID: u}]
		if err != nil || !ok {
			continue
		}
		at := time.Duration(p.GetOutTimeUs()) * time.Microsecond
		out.Progress[ID{UUID: u}] = RunProgress{
			Attempt: int(p.GetAttempt()), Out: min(max(at, 0), length*3/2),
			Speed: min(max(float64(p.GetSpeed()), 0), maxSpeed), Paused: p.GetPaused(),
		}
	}
	return out
}
