package firstrun

import (
	"reflect"
	"testing"
)

func TestViewOfDerivesEachShownStepsState(t *testing.T) {
	cases := []struct {
		name     string
		read     read
		stored   Outcome
		finished bool
		want     State
		hidden   bool
	}{
		{name: "nothing stored or derived", read: read{shows: true}, want: StateTodo},
		{name: "derived done", read: read{shows: true, done: true}, want: StateDone},
		{name: "stored done", read: read{shows: true}, stored: OutcomeDone, want: StateDone},
		{name: "stored skip", read: read{shows: true}, stored: OutcomeSkipped, want: StateSkipped},
		{name: "a derived done wins over a stored skip", read: read{shows: true, done: true}, stored: OutcomeSkipped, want: StateDone},
		{name: "todo once finished", read: read{shows: true}, finished: true, want: StateSkipped},
		{name: "derived done once finished", read: read{shows: true, done: true}, finished: true, want: StateDone},
		{name: "stored done once finished", read: read{shows: true}, stored: OutcomeDone, finished: true, want: StateDone},
		{name: "hidden", read: read{done: true}, stored: OutcomeDone, hidden: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.read.id = RootsID
			stored := map[StepID]Outcome{}
			if c.stored != "" {
				stored[RootsID] = c.stored
			}
			want := View{Open: !c.finished, Steps: []StepView{}}
			if !c.hidden {
				want.Steps = append(want.Steps, StepView{ID: RootsID, State: c.want})
			}
			if got := viewOf([]read{c.read}, stored, c.finished); !reflect.DeepEqual(got, want) {
				t.Errorf("viewOf = %+v, want %+v", got, want)
			}
		})
	}
}

func TestViewOfKeepsTheStepsOrder(t *testing.T) {
	reads := []read{{id: AddressID, shows: true}, {id: PasskeyID, shows: true, done: true}, {id: MediaID}, {id: LanguageID, shows: true}}
	got := viewOf(reads, map[StepID]Outcome{LanguageID: OutcomeSkipped, MediaID: OutcomeDone}, false)
	want := View{Open: true, Steps: []StepView{{AddressID, StateTodo}, {PasskeyID, StateDone}, {LanguageID, StateSkipped}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("viewOf = %+v, want %+v", got, want)
	}
}
