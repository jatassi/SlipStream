package timing

// Reference is a Speech Map as Timing sync reads it: speech per Frame with
// prefix sums, so the speech in any run of frames is O(1).
type Reference struct {
	frames int
	// prefix[i] is the speech frames in [0, i).
	prefix []int32
	// first and last bound the speech: [first, last) holds every speech
	// frame.
	first, last int
}

// NewReference reads frames bits of bits, least significant first.
func NewReference(bits []uint64, frames int) Reference {
	r := Reference{frames: frames, prefix: make([]int32, frames+1), first: -1}
	for f := range frames {
		r.prefix[f+1] = r.prefix[f]
		if bits[f/64]&(1<<(f%64)) != 0 {
			r.prefix[f+1]++
			if r.first < 0 {
				r.first = f
			}
			r.last = f + 1
		}
	}
	if r.first < 0 {
		r.first = 0
	}
	return r
}

func (r Reference) Frames() int { return r.frames }

// Density is the share of frames that are speech, which is what a random
// timing scores.
func (r Reference) Density() float64 {
	if r.frames == 0 {
		return 0
	}
	return float64(r.prefix[r.frames]) / float64(r.frames)
}

// Overlap is the speech frames in [from, to), clamped to the map.
func (r Reference) Overlap(from, to int) int {
	return int(r.at(to) - r.at(from))
}

func (r Reference) at(frame int) int32 {
	return r.prefix[min(max(frame, 0), r.frames)]
}
