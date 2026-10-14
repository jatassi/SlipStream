package events_test

import (
	"testing"

	"github.com/jatassi/slipstream/fakes/events"
)

func TestQueryMatches(t *testing.T) {
	announce := events.Record{Seq: 7, Kind: "tracker.announce", Node: "seedbox", Data: []byte(`{"event":"completed","left":0,"peers":[1, 2]}`)}
	for _, c := range []struct {
		name string
		q    events.Query
		want bool
	}{
		{"the zero query", events.Query{}, true},
		{"a glob on the kind", events.Query{Kind: "tracker.*"}, true},
		{"another kind", events.Query{Kind: "mail.*"}, false},
		{"its node", events.Query{Node: "seedbox"}, true},
		{"another node", events.Query{Node: "home"}, false},
		{"a string field by value", events.Query{Match: map[string]string{"event": "completed"}}, true},
		{"a number field by its JSON", events.Query{Match: map[string]string{"left": "0"}}, true},
		{"an array field by its compact JSON", events.Query{Match: map[string]string{"peers": "[1,2]"}}, true},
		{"a field that differs", events.Query{Match: map[string]string{"event": "started"}}, false},
		{"a field it lacks", events.Query{Match: map[string]string{"uploaded": "0"}}, false},
		{"after an earlier seq", events.Query{After: 6}, true},
		{"after its own seq", events.Query{After: 7}, false},
	} {
		if got := c.q.Matches(announce); got != c.want {
			t.Errorf("%s: %+v matches = %t, want %t", c.name, c.q, got, c.want)
		}
	}
	if (events.Query{Match: map[string]string{"event": "completed"}}).Matches(events.Record{Seq: 1, Kind: "run.ready"}) {
		t.Error("a Match query matched a record with no data")
	}
}
