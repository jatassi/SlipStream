package health

import (
	"context"
	"strings"
	"testing"
)

func TestValidRefusesABrokenAnswer(t *testing.T) {
	noCheck := Category{About: About{ID: NodesID, Label: "Nodes", Phrase: "Nodes"}}
	item := func(k Key) Item { return Item{Key: k, Level: Warn, Says: says("Kitchen")} }
	one := int64(1)
	cases := []struct {
		name   string
		answer Answer
		want   string
	}{
		{name: "an empty key", answer: Answer{Items: []Item{item("")}}, want: "no key"},
		{name: "a duplicate key", answer: Answer{Items: []Item{item("a"), item("a")}}, want: "twice"},
		{name: "a key both unsure and answered", answer: Answer{Items: []Item{item("a")}, Unsure: []Unsure{{Key: "a", Says: says("Kitchen")}}}, want: "twice"},
		{name: "no subject", answer: Answer{Items: []Item{{Key: "a", Level: Warn, Says: Says{Text: "Fix it."}}}}, want: "subject"},
		{name: "no level", answer: Answer{Items: []Item{{Key: "a", Says: says("Kitchen")}}}, want: "level"},
		{name: "a fact with two values", answer: Answer{Items: []Item{{Key: "a", Level: Warn, Says: Says{Subject: "K", Text: "T", Detail: []Fact{{Label: "Free", Text: "1", Bytes: &one}}}}}}, want: "exactly one"},
		{name: "Check now without a Check", answer: Answer{Items: []Item{{Key: "a", Level: Fail, Says: Says{Subject: "K", Text: "T", Action: CheckAction()}}}}, want: "no Check"},
		{name: "Run now without a task", answer: Answer{Items: []Item{{Key: "a", Level: Fail, Says: Says{Subject: "K", Text: "T", Action: RunAction("")}}}}, want: "incomplete"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := valid(noCheck, c.answer); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("valid = %v, want an error naming %q", err, c.want)
			}
		})
	}
	withCheck := noCheck
	withCheck.Check = func(context.Context, Key) ([]Unreached, error) { return nil, nil }
	ok := Answer{
		Items:  []Item{{Key: "a", Level: Fail, Says: Says{Subject: "K", Text: "T", Action: CheckAction(), Detail: []Fact{{Label: "Free", Bytes: &one}}}}},
		Unsure: []Unsure{{Key: "b", Says: says("Kitchen")}},
	}
	if err := valid(withCheck, ok); err != nil {
		t.Fatalf("valid = %v on a sound answer", err)
	}
}

func TestRegisteredPanicsUnlessTheCategoriesAreIDs(t *testing.T) {
	answer := func(context.Context) (Answer, error) { return Answer{}, nil }
	cat := func(id ID) Category {
		return Category{About: About{ID: id, Label: string(id), Phrase: string(id)}, Source: Source{Answer: answer}}
	}
	cases := map[string][]Category{
		"one missing":    {cat(StorageID), cat(NodesID), cat(ScheduledTasksID)},
		"one unknown":    {cat(StorageID), cat(NodesID), cat(DownloadClientsID), cat(ScheduledTasksID), cat("backups")},
		"out of order":   {cat(StorageID), cat(DownloadClientsID), cat(NodesID), cat(ScheduledTasksID)},
		"listed twice":   {cat(StorageID), cat(NodesID), cat(NodesID), cat(DownloadClientsID), cat(ScheduledTasksID)},
		"with no answer": {cat(StorageID), cat(NodesID), cat(DownloadClientsID), {About: About{ID: ScheduledTasksID, Label: "S", Phrase: "s"}}},
	}
	for name, all := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("registered did not panic")
				}
			}()
			registered(all)
		})
	}
	registered([]Category{cat(StorageID), cat(NodesID), cat(DownloadClientsID), cat(ScheduledTasksID)})
}
