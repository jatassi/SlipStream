package engine

import (
	"time"

	"google.golang.org/protobuf/proto"

	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
)

// note changes the run's facts under its lock.
func (r *run) note(change func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	change()
}

// setWait sets the receiver's wait, reporting a change at once.
func (r *run) setWait(kind tv1.Wait_Kind, detail string) {
	w := &tv1.Wait{Kind: kind, Detail: detail}
	r.mu.Lock()
	changed := !proto.Equal(r.rx.wait, w)
	r.rx.wait = w
	r.mu.Unlock()
	if changed {
		r.e.facts.urgent()
	}
}

// fact is the run's Receiving or Sending now.
func (r *run) fact(now time.Time) proto.Message {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rx == nil {
		f := &tv1.Sending{TransferId: r.spec.id.Bytes(), GaveUp: r.gaveUpRoutes(), Binding: r.binding, Route: r.r}
		r.binding = false
		return f
	}
	rx := r.rx
	rx.sampleRate(now, r.e.cfg.Timings.Estimate)
	return &tv1.Receiving{
		TransferId:    r.spec.id.Bytes(),
		Active:        true,
		VerifiedBytes: rx.verifiedBytes,
		Rate:          int64(rx.rate),
		Moving:        rx.moving != tv1.Route_ROUTE_UNSPECIFIED,
		Lanes:         uint32(rx.lanes),              //nolint:gosec // one per route
		RttMs:         uint32(rx.rtt.Milliseconds()), //nolint:gosec // a round trip, far under 49 days
		Retried:       uint32(rx.retried),            //nolint:gosec // a count of chunks asked again
		GaveUp:        r.gaveUpRoutes(),
		Wait:          rx.wait,
		Route:         r.r,
	}
}

// sampleRate folds the bytes taken since the last sample into the rate,
// weighting the new sample by its share of window.
func (rx *receiving) sampleRate(now time.Time, window time.Duration) {
	if rx.rateAt.IsZero() {
		rx.rateAt, rx.rateBytes = now, rx.moved
		return
	}
	elapsed := now.Sub(rx.rateAt)
	if elapsed <= 0 {
		return
	}
	inst := float64(rx.moved-rx.rateBytes) / elapsed.Seconds()
	weight := min(1, elapsed.Seconds()/window.Seconds())
	rx.rate += weight * (inst - rx.rate)
	rx.rateAt, rx.rateBytes = now, rx.moved
}

// keptFact is what a stopped receiver leaves in facts while its Transfer
// stays kept: the last wait it saw, if any.
func (r *run) keptFact() *tv1.Receiving {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.rx == nil || r.rx.failed || r.rx.wait.GetKind() == tv1.Wait_KIND_UNSPECIFIED {
		return nil
	}
	return &tv1.Receiving{TransferId: r.spec.id.Bytes(), Wait: r.rx.wait, Route: r.r}
}
