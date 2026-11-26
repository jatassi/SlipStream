package playback

// outcomeOf is what would avoid p's transcode, tried in the spec's order.
func outcomeOf(in Input, p Play, dims []Blocked, pr Proofs) Outcome {
	if v, ev, ok := versionFor(in, p, dims, pr); ok {
		return Outcome{Kind: OutcomeVersion, Version: &v, Evidence: ev}
	}
	if !allChoosable(dims) {
		return Outcome{Kind: OutcomeNoTarget}
	}
	f := p.Snapshot.Facts
	env := envelopeOf(p, dims, pr)
	if miss, ok := env.firstMissing(f, pr); ok {
		return Outcome{Kind: OutcomeUnproven, Missing: &miss}
	}
	ev := pr.evidenceOf(in, env.facts())
	if t, ok := existingIn(in, p, env); ok {
		return Outcome{Kind: OutcomeExisting, Target: &TargetRef{ID: t.ID.String(), Name: t.Name}, Works: []WorkLine{workLine(in, p, t, pr)}, Evidence: ev}
	}
	s := env.suggestion(f, dims, pr)
	return Outcome{Kind: OutcomeSuggest, Suggestion: &s, Evidence: ev}
}

func allChoosable(dims []Blocked) bool {
	for _, d := range dims {
		if !d.Choosable {
			return false
		}
	}
	return true
}
