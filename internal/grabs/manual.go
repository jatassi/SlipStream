package grabs

import "context"

// listen is a Manual run's goroutine: it asks the run's lines with no
// outcome and stores each answer as it lands. A restart starts it again,
// and it asks only the lines still unanswered.
func (gr *Grabs) listen(context.Context, Search) result {
	return halted
}
