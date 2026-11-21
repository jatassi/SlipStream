package timing

import (
	"math"
	"time"
)

// The split search is alass's (GPL-3.0-or-later) as ffsubsync's split_aligner.py
// (MIT) ports it: every cue may take its own offset, each change of offset
// between neighboring cues costs the split penalty, and a forward pass
// over cues by offsets with a backtrack finds the best total. See
// internal/notices/alass and internal/notices/ffsubsync.

// grid is the offsets Align may give a cue at one ratio: whole frames in
// [-w, w], index j for offset j − w.
type grid struct {
	ref   Reference
	w     int
	guard int
	lp    float64
	rho   float64
}

// cue is a span in frames at the grid's ratio.
type cue struct {
	from, to int
	voiced   bool
}

func (g grid) cues(spans []Span) []cue {
	out := make([]cue, len(spans))
	for i, sp := range spans {
		from, to, ok := frames(sp, g.rho)
		out[i] = cue{from, to, ok}
	}
	return out
}

func (g grid) offset(j int) time.Duration { return time.Duration(j-g.w) * Frame }

// rate is a cue's speech overlap at offset index j, less LengthPenalty
// times the speech in a guard band of the cue's own length, at most
// GuardMax, just outside each edge.
func (g grid) rate(c cue, j int) float64 {
	if !c.voiced {
		return 0
	}
	s, e := c.from+j-g.w, c.to+j-g.w
	band := min(c.to-c.from, g.guard)
	ps, pe := g.ref.at(s), g.ref.at(e)
	guard := ps - g.ref.at(s-band) + g.ref.at(e+band) - pe
	return float64(pe-ps) - g.lp*float64(guard)
}

// split is the offset index of each cue. Memory is one bit per cue and
// offset, about 1.1 MB for 1,500 cues at ±30 s, where ffsubsync keeps an
// int64 per cell.
func (g grid) split(cues []cue, penalty float64) []int {
	n := 2*g.w + 1
	words := (n + 63) / 64
	jumped := make([]uint64, len(cues)*words)
	prev := make([]int, len(cues))
	v := make([]float64, n)
	for j := range v {
		v[j] = g.rate(cues[0], j)
	}
	for i := 1; i < len(cues); i++ {
		best := argmax(v)
		jump := v[best] - penalty
		prev[i] = best
		row := jumped[i*words : (i+1)*words]
		for j, stay := range v {
			if jump > stay {
				stay = jump
				row[j/64] |= 1 << (j % 64)
			}
			v[j] = stay + g.rate(cues[i], j)
		}
	}
	at := make([]int, len(cues))
	j := argmax(v)
	for i := len(cues) - 1; i >= 0; i-- {
		at[i] = j
		if i > 0 && jumped[i*words+j/64]&(1<<(j%64)) != 0 {
			j = prev[i]
		}
	}
	return at
}

func argmax(v []float64) int {
	best := 0
	for j, x := range v {
		if x > v[best] {
			best = j
		}
	}
	return best
}

// run is the cues from first to the next run's first at offset index j.
type run struct{ first, j int }

func runs(at []int) []run {
	var rs []run
	for i, j := range at {
		if len(rs) == 0 || rs[len(rs)-1].j != j {
			rs = append(rs, run{i, j})
		}
	}
	return rs
}

// settle merges runs until a timing of several has none shorter than
// MinSegment and at most MaxSegments. Each step moves the run whose move
// to a neighbor's offset loses the least rating: a short run first, then
// any run.
func (g grid) settle(spans []Span, cues []cue, rs []run, k Constants) []run {
	p := plan{g, spans, cues, rs}
	short := func(i int) bool { return p.extent(i) < k.MinSegment }
	for len(p.rs) > 1 {
		from, to := p.cheapest(short)
		if from < 0 {
			if len(p.rs) <= k.MaxSegments {
				break
			}
			from, to = p.cheapest(func(int) bool { return true })
		}
		p.rs[from].j = p.rs[to].j
		p.rs = coalesce(p.rs)
	}
	return p.rs
}

// plan is runs being settled.
type plan struct {
	grid
	spans []Span
	cues  []cue
	rs    []run
}

func (p plan) end(i int) int {
	if i+1 < len(p.rs) {
		return p.rs[i+1].first
	}
	return len(p.cues)
}

func (p plan) extent(i int) time.Duration {
	return extent(p.spans[p.rs[i].first:p.end(i)], p.rho, p.offset(p.rs[i].j))
}

// rating is run i's total rating at offset index j.
func (p plan) rating(i, j int) float64 {
	sum := 0.0
	for c := p.rs[i].first; c < p.end(i); c++ {
		sum += p.rate(p.cues[c], j)
	}
	return sum
}

// cheapest is the run of those in that loses least by taking a neighbor's
// offset, and that neighbor: −1 and −1 when none is in.
func (p plan) cheapest(in func(i int) bool) (from, to int) {
	from, to = -1, -1
	least := math.Inf(1)
	for i := range p.rs {
		if !in(i) {
			continue
		}
		for _, nb := range [2]int{i - 1, i + 1} {
			if nb < 0 || nb >= len(p.rs) {
				continue
			}
			if loss := p.rating(i, p.rs[i].j) - p.rating(i, p.rs[nb].j); loss < least {
				from, to, least = i, nb, loss
			}
		}
	}
	return from, to
}

func coalesce(rs []run) []run {
	out := rs[:1]
	for _, r := range rs[1:] {
		if r.j != out[len(out)-1].j {
			out = append(out, r)
		}
	}
	return out
}

// extent is the original time a run spans over points Judge is sure to
// fit at any stamp unit up to a Frame: the starts that map to a Frame or
// later, and the ends of those cues when they map at least two Frames
// long. Judge's own extent is therefore never shorter, so a segment Align
// keeps always clears Judge's MinSegment.
func extent(spans []Span, rho float64, o time.Duration) time.Duration {
	lo, hi := time.Duration(math.MaxInt64), time.Duration(math.MinInt64)
	for _, sp := range spans {
		if rho*float64(sp.Start)+float64(o) < float64(Frame) {
			continue
		}
		lo, hi = min(lo, sp.Start), max(hi, sp.Start)
		if rho*float64(sp.End-sp.Start) >= float64(2*Frame) {
			hi = max(hi, sp.End)
		}
	}
	if hi < lo {
		return 0
	}
	return hi - lo
}
