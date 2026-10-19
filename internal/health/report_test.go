package health

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

var day0 = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

func at(hours int) time.Time { return day0.Add(time.Duration(hours) * time.Hour) }

func says(subject string) Says { return Says{Subject: subject, Text: "Fix " + subject + "."} }

func keys(items []Shown) []Key {
	out := make([]Key, len(items))
	for i, s := range items {
		out[i] = s.Key
	}
	return out
}

func TestShownOrdersErrorsFirstThenOldestThenKey(t *testing.T) {
	c := Category{About: About{ID: "storage", Label: "Storage", Phrase: "storage"}}
	rows := map[Key]row{
		"warn-old": {Level: Warn, Since: at(0)},
		"fail-new": {Level: Fail, Since: at(5)},
		"fail-old": {Level: Fail, Since: at(1)},
		"b":        {Level: Warn, Since: at(3)},
		"a":        {Level: Warn, Since: at(3)},
	}
	var a Answer
	for k := range rows {
		a.Items = append(a.Items, Item{Key: k, Level: rows[k].Level, Says: says(string(k))})
	}
	got := shownOf(c, rows, a, nil, nil)
	if want := []Key{"fail-old", "fail-new", "warn-old", "a", "b"}; !reflect.DeepEqual(keys(got.Items), want) {
		t.Fatalf("items = %v, want %v", keys(got.Items), want)
	}
	if got.Level != Fail || got.Unread != "" {
		t.Fatalf("level %v, unread %q; want fail and read", got.Level, got.Unread)
	}
}

func TestShownKeepsWordsForUnsureAndFailedAnswers(t *testing.T) {
	c := Category{About: About{ID: "nodes", Label: "Nodes", Phrase: "Nodes"}}
	rows := map[Key]row{"n1": {Level: Fail, Since: at(0)}, "n2": {Level: Warn, Since: at(1)}}
	unsure := shownOf(c, rows, Answer{Items: []Item{{Key: "n2", Level: Warn, Says: says("Kitchen")}}, Unsure: []Unsure{{Key: "n1", Says: says("Media server")}}}, nil, nil)
	if s := unsure.Items[0]; s.Key != "n1" || !s.Unsure || s.Level != Fail || s.Subject != "Media server" || !s.Since.Equal(at(0)) {
		t.Fatalf("unsure item = %+v, want n1 unsure at fail since %v as Media server", s, at(0))
	}
	last := map[Key]Says{"n1": says("Media server")}
	failed := shownOf(c, rows, Answer{}, errors.New("the board is closed"), last)
	if failed.Unread != "the board is closed" || len(failed.Items) != 2 {
		t.Fatalf("failed answer shows %+v, want both rows and the reason", failed)
	}
	if failed.Items[0].Subject != "Media server" || failed.Items[1].Subject != "Nodes" {
		t.Fatalf("subjects %q, %q; want the last words, else the category's label", failed.Items[0].Subject, failed.Items[1].Subject)
	}
}

func TestReportFirstAndWorst(t *testing.T) {
	shown := func(k Key, l Level, since time.Time) Shown {
		return Shown{Item: Item{Key: k, Level: l, Says: says(string(k))}, Since: since}
	}
	cats := []CategoryReport{
		{About: About{ID: StorageID}, Level: Warn, Items: []Shown{shown("disk", Warn, at(0))}},
		{About: About{ID: NodesID}, Level: Fail, Items: []Shown{shown("kitchen", Fail, at(4))}},
		{About: About{ID: ScheduledTasksID}, Level: Fail, Items: []Shown{shown("tidy", Fail, at(2))}},
	}
	r := reportOf(cats)
	if r.Level != Fail || r.First == nil || r.First.Category != ScheduledTasksID || r.First.Key != "tidy" || r.First.Text != "Fix tidy." {
		t.Fatalf("report level %v first %+v; want fail led by the oldest error, tidy", r.Level, r.First)
	}
	tied := reportOf([]CategoryReport{cats[1], {About: About{ID: ScheduledTasksID}, Level: Fail, Items: []Shown{shown("tidy", Fail, at(4))}}})
	if tied.First.Category != NodesID {
		t.Fatalf("a tie leads with %s, want the earlier category, nodes", tied.First.Category)
	}
	if got := r.Worst(StorageID); got != Warn {
		t.Errorf("Worst(storage) = %v, want warn", got)
	}
	if got := r.Worst(StorageID, NodesID); got != Fail {
		t.Errorf("Worst(storage, nodes) = %v, want fail", got)
	}
	if got := r.Worst(); got != Fail {
		t.Errorf("Worst() = %v, want fail", got)
	}
	if empty := reportOf([]CategoryReport{{About: About{ID: StorageID}}}); empty.First != nil || empty.Level != None || empty.Worst(StorageID) != None {
		t.Errorf("a report with nothing wrong = %+v, want no first and level none", empty)
	}
}
