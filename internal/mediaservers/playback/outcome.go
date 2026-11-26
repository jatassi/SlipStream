package playback

// outcomeOf is what would avoid p's transcode, tried in the spec's order.
func outcomeOf(in Input, p Play, dims []Blocked, pr Proofs) Outcome {
	if !allChoosable(dims) {
		return Outcome{Kind: OutcomeNoTarget}
	}
	f := p.Snapshot.Facts
	env := envelopeOf(p, dims, pr)
	if miss, ok := env.firstMissing(f, pr); ok {
		return Outcome{Kind: OutcomeUnproven, Missing: &miss}
	}
	s := env.suggestion(f, dims, pr)
	return Outcome{Kind: OutcomeSuggest, Suggestion: &s, Evidence: pr.evidenceOf(in, env.facts())}
}

func allChoosable(dims []Blocked) bool {
	for _, d := range dims {
		if !d.Choosable {
			return false
		}
	}
	return true
}
