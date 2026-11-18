package timing

import (
	"math"
	"math/bits"

	"gonum.org/v1/gonum/dsp/fourier"
)

// signal is a subtitle's speech at one ratio: the frames [lo, hi) of its
// cue extent, with on set for the frames some voiced cue covers.
type signal struct {
	lo, hi int
	on     []uint64
}

func (s *signal) at(frame int) bool {
	i := frame - s.lo
	return s.on[i/64]&(1<<(i%64)) != 0
}

// raster draws the voiced cues of spans at ratio rho into s, reusing its
// words.
func (s *signal) raster(spans []Span, rho float64) bool {
	s.lo, s.hi = math.MaxInt, math.MinInt
	for _, sp := range spans {
		if from, to, ok := frames(sp, rho); ok {
			s.lo, s.hi = min(s.lo, from), max(s.hi, to)
		}
	}
	if s.lo >= s.hi {
		return false
	}
	words := (s.hi - s.lo + 63) / 64
	s.on = append(s.on[:0], make([]uint64, words)...)
	for _, sp := range spans {
		from, to, ok := frames(sp, rho)
		if !ok {
			continue
		}
		for f := from - s.lo; f < to-s.lo; f++ {
			s.on[f/64] |= 1 << (f % 64)
		}
	}
	return true
}

// frames is a voiced span's frames at ratio rho.
func frames(sp Span, rho float64) (from, to int, ok bool) {
	if sp.Skip {
		return 0, 0, false
	}
	from = int(math.Round(rho * float64(sp.Start) / float64(Frame)))
	to = int(math.Round(rho * float64(sp.End) / float64(Frame)))
	return from, to, to > from
}

// step is the frames one correlation sample sums. At 50 ms the
// correlation ranks ratios and finds its peaks as well as at 10 ms, for a
// fifth of the FFT work; the split search then places each cue to the
// Frame.
const step = 5

// correlator scores a signal against the reference at every lag of
// [-reach, reach] steps: score[reach+lag] = Σ x[f]·r[f+lag·step] over
// frames f. x is +1 on the signal's cue frames and −1 on the rest of its
// extent, as in ffsubsync's FFTAligner. r is the reference centered on its
// density, 1−c on speech and −c elsewhere, so a lag that matches nothing
// scores about zero whatever the two densities are, and a peak's share of
// the best is a fair Prominence.
//
// It correlates the extent in blocks by overlap-save, so its buffers are a
// few FFT lengths whatever the file's length: one FFT over a two-hour film
// would need tens of MB.
type correlator struct {
	ref     Reference
	density float64
	reach   int
	block   int
	fft     *fourier.FFT
	xs, rs  []float64
	xc, rc  []complex128
	out     []float64
	scores  []float64
}

// newCorrelator looks reach frames either way.
func newCorrelator(ref Reference, reach int) *correlator {
	r := (reach + step - 1) / step
	n := 1 << bits.Len(uint(4*r+1))
	return &correlator{
		ref: ref, density: ref.Density(), reach: r, block: n - 2*r,
		fft: fourier.NewFFT(n),
		xs:  make([]float64, n), rs: make([]float64, n),
		xc: make([]complex128, n/2+1), rc: make([]complex128, n/2+1),
		out: make([]float64, n), scores: make([]float64, 2*r+1),
	}
}

// correlate fills and returns c.scores, which the next call reuses.
func (c *correlator) correlate(x signal) []float64 {
	clear(c.scores)
	n := len(c.xs)
	samples := (x.hi - x.lo + step - 1) / step
	for start := 0; start < samples; start += c.block {
		for j := range n {
			c.xs[j] = 0
			if i := start + j; j < c.block && i < samples {
				c.xs[j] = x.sum(x.lo + i*step)
			}
			c.rs[j] = c.refSum(x.lo + (start-c.reach+j)*step)
		}
		c.fft.Coefficients(c.xc, c.xs)
		c.fft.Coefficients(c.rc, c.rs)
		for i, v := range c.xc {
			c.rc[i] *= complex(real(v), -imag(v))
		}
		c.fft.Sequence(c.out, c.rc)
		for k := range c.scores {
			c.scores[k] += c.out[k] / float64(n)
		}
	}
	return c.scores
}

// sum is x over the step frames from frame, within the extent.
func (s *signal) sum(frame int) float64 {
	v := 0.0
	for f := frame; f < min(frame+step, s.hi); f++ {
		v--
		if s.at(f) {
			v += 2
		}
	}
	return v
}

// refSum is the centered reference over the step frames from frame,
// within the map.
func (c *correlator) refSum(frame int) float64 {
	valid := min(frame+step, c.ref.frames) - max(frame, 0)
	if valid <= 0 {
		return 0
	}
	return float64(c.ref.Overlap(frame, frame+step)) - c.density*float64(valid)
}

// peak is the best lag of scores, in frames, and its score.
func (c *correlator) peak(scores []float64) (lag int, score float64) {
	best := 0
	for k, v := range scores {
		if v > scores[best] {
			best = k
		}
	}
	return (best - c.reach) * step, scores[best]
}
