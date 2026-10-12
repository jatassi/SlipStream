// Package op declares control requests: a typed pair that the owning package
// declares once and both the Node and the host command share, so request and
// reply types cannot drift.
package op

type Op[Req, Resp any] struct {
	Name string
}
