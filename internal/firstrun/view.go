package firstrun

// read is one step's facts as one snapshot read them.
type read struct {
	id          StepID
	shows, done bool
}

// viewOf derives each shown step's state: done when its Done holds or a
// done row exists; skipped when a skipped row exists, or once finished when
// it would be todo; todo otherwise. A derived done wins over a stored skip.
func viewOf(reads []read, stored map[StepID]Outcome, finished bool) View {
	v := View{Open: !finished, Steps: []StepView{}}
	for _, r := range reads {
		if !r.shows {
			continue
		}
		s := StepView{ID: r.id, State: StateTodo}
		switch {
		case r.done || stored[r.id] == OutcomeDone:
			s.State = StateDone
		case stored[r.id] == OutcomeSkipped || finished:
			s.State = StateSkipped
		}
		v.Steps = append(v.Steps, s)
	}
	return v
}
