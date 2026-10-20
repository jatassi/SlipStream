package metadata

import (
	"slices"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

func TestDue(t *testing.T) {
	zone := time.FixedZone("UTC+10", 10*60*60)
	now := time.Date(2026, time.March, 10, 4, 10, 0, 0, zone)
	at := func(day, hour int) time.Time { return time.Date(2026, time.March, day, hour, 0, 0, 0, zone) }
	ids := map[string]works.WorkID{}
	work := func(label string) works.WorkID {
		if _, ok := ids[label]; !ok {
			ids[label] = works.WorkID{UUID: storage.NewUUID()}
		}
		return ids[label]
	}
	tiers := tiers{unreleased: day, growing: 2 * day, others: 30 * day}
	cases := []struct {
		name string
		in   []standing
		want []string
	}{
		{
			name: "an unreleased Work refreshed yesterday is due, whatever the hour",
			in:   []standing{{label: "late", answered: at(9, 4).Add(25 * time.Minute), unreleased: true}},
			want: []string{"late"},
		},
		{
			name: "an unreleased Work refreshed today is not",
			in:   []standing{{label: "today", answered: at(10, 1), unreleased: true}},
		},
		{
			name: "the day counts in now's zone",
			in:   []standing{{label: "utc", answered: time.Date(2026, time.March, 9, 23, 0, 0, 0, time.UTC), unreleased: true}},
		},
		{
			name: "a growing Work waits its own tier",
			in: []standing{
				{label: "one day", answered: at(9, 4), growing: true},
				{label: "two days", answered: at(8, 4), growing: true},
			},
			want: []string{"two days"},
		},
		{
			name: "unreleased wins over growing",
			in:   []standing{{label: "both", answered: at(9, 4), unreleased: true, growing: true}},
			want: []string{"both"},
		},
		{
			name: "every other Work waits the others tier",
			in: []standing{
				{label: "29 days", answered: at(10, 4).AddDate(0, 0, -29)},
				{label: "30 days", answered: at(10, 4).AddDate(0, 0, -30)},
			},
			want: []string{"30 days"},
		},
		{
			name: "a gone Work rides the others tier even while unreleased",
			in: []standing{
				{label: "gone yesterday", answered: at(9, 4), gone: true, unreleased: true},
				{label: "gone a month ago", answered: at(10, 4).AddDate(0, 0, -30), gone: true, unreleased: true},
			},
			want: []string{"gone a month ago"},
		},
		{
			name: "a mark after the last answer makes a Work due, one before it doesn't",
			in: []standing{
				{label: "marked", answered: at(10, 1), marked: at(10, 4)},
				{label: "satisfied", answered: at(10, 2), marked: at(10, 1)},
			},
			want: []string{"marked"},
		},
		{
			name: "a refresh-all request after the last answer makes a Work due",
			in: []standing{
				{label: "asked", answered: at(10, 1), all: at(10, 3)},
				{label: "answered since", answered: at(10, 4), all: at(10, 3)},
			},
			want: []string{"asked"},
		},
		{
			name: "longest due first, then by label",
			in: []standing{
				{label: "b mark", answered: at(10, 1), marked: at(10, 3)},
				{label: "tier", answered: at(5, 4), unreleased: true},
				{label: "a mark", answered: at(10, 1), marked: at(10, 3)},
				{label: "old request", answered: at(1, 1), all: at(2, 0), unreleased: true},
			},
			want: []string{"old request", "tier", "a mark", "b mark"},
		},
		{
			name: "a Work due by mark and tier is due from the earlier",
			in: []standing{
				{label: "tier first", answered: at(8, 4), unreleased: true, marked: at(10, 3)},
				{label: "between", answered: at(9, 5), marked: at(9, 6)},
				{label: "mark first", answered: at(8, 1), unreleased: true, marked: at(8, 2)},
			},
			want: []string{"mark first", "tier first", "between"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for i := range c.in {
				c.in[i].work = work(c.in[i].label)
			}
			var got []string
			for _, s := range due(now, c.in, tiers) {
				got = append(got, s.label)
			}
			if !slices.Equal(got, c.want) {
				t.Fatalf("due = %q, want %q", got, c.want)
			}
		})
	}
}
