package health

import (
	"reflect"
	"testing"
	"time"
)

func TestDiff(t *testing.T) {
	then := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	now := then.Add(time.Hour)
	warnRow := row{Level: Warn, Since: then, ChangedAt: then}
	failRow := row{Level: Fail, Since: then, ChangedAt: then}
	item := func(k Key, l Level, text string) Item {
		return Item{Key: k, Level: l, Says: Says{Subject: "D: on Media server", Text: text}}
	}
	unsure := func(k Key) Unsure {
		return Unsure{Key: k, Says: Says{Subject: "D: on Media server", Text: "As of 14:05."}}
	}
	cases := []struct {
		name   string
		rows   map[Key]row
		answer Answer
		failed bool
		want   []change
	}{
		{
			name:   "a new item is raised with since and changed_at now",
			answer: Answer{Items: []Item{item("a", Warn, "Free up space.")}},
			want:   []change{{Key: "a", Kind: Raised, To: Warn, Row: row{Level: Warn, Since: now, ChangedAt: now}}},
		},
		{
			name:   "warn to fail is worse and keeps since",
			rows:   map[Key]row{"a": warnRow},
			answer: Answer{Items: []Item{item("a", Fail, "Free up space.")}},
			want:   []change{{Key: "a", Kind: Worse, From: Warn, To: Fail, Row: row{Level: Fail, Since: then, ChangedAt: now}}},
		},
		{
			name:   "fail to warn is better and keeps since",
			rows:   map[Key]row{"a": failRow},
			answer: Answer{Items: []Item{item("a", Warn, "Free up space.")}},
			want:   []change{{Key: "a", Kind: Better, From: Fail, To: Warn, Row: row{Level: Warn, Since: then, ChangedAt: now}}},
		},
		{
			name: "an item the answer omits is cleared",
			rows: map[Key]row{"a": failRow},
			want: []change{{Key: "a", Kind: Cleared, From: Fail}},
		},
		{
			name:   "a text change at one level is no change",
			rows:   map[Key]row{"a": warnRow},
			answer: Answer{Items: []Item{item("a", Warn, "It has 12 GB left.")}},
		},
		{
			name:   "a failing source clears nothing",
			rows:   map[Key]row{"a": warnRow, "b": failRow},
			failed: true,
		},
		{
			name:   "an unsure key keeps its row",
			rows:   map[Key]row{"a": failRow},
			answer: Answer{Unsure: []Unsure{unsure("a")}},
		},
		{
			name:   "an unsure key with no row raises nothing",
			answer: Answer{Unsure: []Unsure{unsure("a")}},
		},
		{
			name:   "unsure protects only its own key",
			rows:   map[Key]row{"a": failRow, "b": warnRow},
			answer: Answer{Unsure: []Unsure{unsure("a")}},
			want:   []change{{Key: "b", Kind: Cleared, From: Warn}},
		},
		{
			name:   "changes come in key order",
			rows:   map[Key]row{"c": warnRow, "a": warnRow},
			answer: Answer{Items: []Item{item("b", Fail, "Free up space.")}},
			want: []change{
				{Key: "a", Kind: Cleared, From: Warn},
				{Key: "b", Kind: Raised, To: Fail, Row: row{Level: Fail, Since: now, ChangedAt: now}},
				{Key: "c", Kind: Cleared, From: Warn},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := diff(c.rows, c.answer, c.failed, now); !reflect.DeepEqual(got, c.want) {
				t.Fatalf("diff = %+v, want %+v", got, c.want)
			}
		})
	}
}
