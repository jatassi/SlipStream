package playback

import "slices"

// suggestion is the Target the new-Target page starts from. Its Only is
// exact for the one video codec; HDR, audio codecs and channels list every
// value outside the envelope under Not.
func (e envelope) suggestion(f Facts, dims []Blocked, pr Proofs) Suggestion {
	highest := e[Resolution][0].Text
	blocked := blockedValues(f, dims, pr)
	return Suggestion{
		Name: highest + " " + e[HDR][0].Text, Highest: highest, VideoCodecs: texts(e[VideoCodec]),
		NotHDR: e.outside(HDR), NotAudioCodecs: e.outside(AudioCodec), NotChannels: e.outside(Channels),
		Blocked: blocked, Unproven: unprovenValues(pr, blocked),
	}
}

// outside is the vocabulary values on d the envelope doesn't hold.
func (e envelope) outside(d Dimension) []string {
	out := []string{}
	for _, v := range vocabulary[d] {
		if !slices.Contains(e[d], v) {
			out = append(out, v.Text)
		}
	}
	return out
}

// blockedValues is the played file's values on the blocking dimensions,
// which the suggestion leaves out because they blocked direct play.
func blockedValues(f Facts, dims []Blocked, pr Proofs) []Blocked {
	out := []Blocked{}
	for _, d := range dims {
		for _, v := range fileValues(d.Dimension, f, pr) {
			if v.Text != "" {
				out = append(out, block(d.Dimension, v.Text))
			}
		}
	}
	return out
}

// unprovenValues is the vocabulary values the player hasn't received
// unchanged, but those that blocked.
func unprovenValues(pr Proofs, blocked []Blocked) []Blocked {
	out := []Blocked{}
	for _, row := range dimensions {
		for _, v := range vocabulary[row.Dimension] {
			b := block(row.Dimension, v.Text)
			if _, ok := pr.proven(v); !ok && !slices.Contains(blocked, b) {
				out = append(out, b)
			}
		}
	}
	return out
}

func texts(fs []Fact) []string {
	out := make([]string, len(fs))
	for i, f := range fs {
		out[i] = f.Text
	}
	return out
}
