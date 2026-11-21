//go:build !cgo

package fvad

// VAD is absent from a build without cgo: New refuses, so the Node reports
// a node fault on every map build.
type VAD struct{}

func New(int, int) (*VAD, error) { return nil, ErrNoVAD }

func (*VAD) Speech([]int16) (bool, error) { return false, ErrNoVAD }

func (*VAD) Close() {}
