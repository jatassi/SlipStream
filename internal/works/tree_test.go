package works

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

var (
	earlier = storage.FromTime(time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	now     = storage.FromTime(time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC))
)

func album(key Key, pos int) level {
	return level{node: node{Key: key, Label: "Album " + string(key), Position: pos}, Depth: 1}
}

func track(key, on Key, pos int) node {
	return node{Key: key, Parent: on, Label: "Track " + string(key), Position: pos}
}

func fetched(levels []level, items []node) parsed {
	return parsed{Label: "Artist", SortLabel: "Artist, The", nodes: nodes{Levels: levels, Items: items}}
}

func withdrawn[N any, P tierNode[N]](n N, at storage.Time) N {
	P(&n).entry().WithdrawnAt = at
	return n
}

func TestPlanMerge(t *testing.T) {
	renamed := func(n node, label string) node { n.Label = label; return n }
	moved := func(n node, parent Key, pos int) node { n.Parent, n.Position = parent, pos; return n }
	stored := nodes{
		Levels: []level{album("b1", 1), album("b2", 2)},
		Items:  []node{track("c1", "b1", 1), track("c2", "b1", 2), track("c3", "b2", 1)},
	}
	cases := []struct {
		name    string
		stored  nodes
		fetched parsed
		want    plan
	}{
		{
			name:    "fresh tree",
			fetched: fetched([]level{album("b1", 1)}, []node{track("c1", "b1", 1), track("c2", "b1", 2)}),
			want: plan{Label: "Artist", SortLabel: "Artist, The", RefreshedAt: now,
				Levels: []change[level]{{Node: album("b1", 1), Insert: true}},
				Items:  []change[node]{{Node: track("c1", "b1", 1), Insert: true}, {Node: track("c2", "b1", 2), Insert: true}},
			},
		},
		{
			name:    "unchanged tree",
			stored:  stored,
			fetched: fetched(stored.Levels, stored.Items),
			want:    plan{Label: "Artist", SortLabel: "Artist, The", RefreshedAt: now},
		},
		{
			name:    "rename",
			stored:  stored,
			fetched: fetched(stored.Levels, []node{renamed(track("c1", "b1", 1), "Intro"), track("c2", "b1", 2), track("c3", "b2", 1)}),
			want: plan{Label: "Artist", SortLabel: "Artist, The", RefreshedAt: now,
				Items: []change[node]{{Node: renamed(track("c1", "b1", 1), "Intro")}},
			},
		},
		{
			name:    "reorder",
			stored:  stored,
			fetched: fetched([]level{album("b2", 1), album("b1", 2)}, []node{track("c2", "b1", 1), track("c1", "b1", 2), track("c3", "b2", 1)}),
			want: plan{Label: "Artist", SortLabel: "Artist, The", RefreshedAt: now,
				Levels: []change[level]{{Node: album("b2", 1)}, {Node: album("b1", 2)}},
				Items:  []change[node]{{Node: track("c2", "b1", 1)}, {Node: track("c1", "b1", 2)}},
			},
		},
		{
			name:    "withdrawal",
			stored:  stored,
			fetched: fetched([]level{album("b1", 1)}, []node{track("c1", "b1", 1)}),
			want: plan{Label: "Artist", SortLabel: "Artist, The", RefreshedAt: now,
				Levels: []change[level]{{Node: withdrawn(album("b2", 2), now)}},
				Items:  []change[node]{{Node: withdrawn(track("c2", "b1", 2), now)}, {Node: withdrawn(track("c3", "b2", 1), now)}},
			},
		},
		{
			name: "return",
			stored: nodes{
				Levels: []level{album("b1", 1), withdrawn(album("b2", 2), earlier)},
				Items:  []node{track("c1", "b1", 1), withdrawn(track("c2", "b1", 2), earlier), withdrawn(track("c3", "b2", 1), earlier)},
			},
			fetched: fetched([]level{album("b1", 1), album("b2", 2)}, []node{track("c1", "b1", 1), track("c3", "b2", 1)}),
			want: plan{Label: "Artist", SortLabel: "Artist, The", RefreshedAt: now,
				Levels: []change[level]{{Node: album("b2", 2)}},
				Items:  []change[node]{{Node: track("c3", "b2", 1)}},
			},
		},
		{
			name:    "parent change",
			stored:  stored,
			fetched: fetched(stored.Levels, []node{track("c1", "b1", 1), track("c3", "b2", 1), track("c2", "b2", 2)}),
			want: plan{Label: "Artist", SortLabel: "Artist, The", RefreshedAt: now,
				Items: []change[node]{{Node: moved(track("c2", "b1", 2), "b2", 2)}},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := planMerge(c.stored, c.fetched, now); !reflect.DeepEqual(got, c.want) {
				t.Errorf("plan =\n%+v\nwant\n%+v", got, c.want)
			}
		})
	}
}

var records = Description{
	Name:   "Records",
	Work:   Noun{One: "Artist", Many: "Artists"},
	Levels: []Noun{{One: "Album", Many: "Albums"}},
	Item:   Noun{One: "Track", Many: "Tracks"},
}

func TestParseFetchedPlacesEveryNode(t *testing.T) {
	got, err := parseFetched(records, "a1", Fetched{
		Label: "Artist",
		Children: []Entry{
			{Key: "b1", Label: "First", Children: []Entry{{Key: "c1", Label: "One"}, {Key: "c2", Label: "Two"}}},
			{Key: "b2", Label: "Second", Children: []Entry{{Key: "c3", Label: "Three"}}},
		},
		Detail: []byte("raw"),
	})
	want := parsed{
		Label: "Artist", SortLabel: "Artist",
		nodes: nodes{
			Levels: []level{{node: node{Key: "b1", Label: "First", Position: 1}, Depth: 1}, {node: node{Key: "b2", Label: "Second", Position: 2}, Depth: 1}},
			Items: []node{
				{Key: "c1", Parent: "b1", Label: "One", Position: 1},
				{Key: "c2", Parent: "b1", Label: "Two", Position: 2},
				{Key: "c3", Parent: "b2", Label: "Three", Position: 1},
			},
		},
		Detail: []byte("raw"),
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed = %+v, %v; want %+v", got, err, want)
	}
}

func TestParseFetchedRefusesATreeThatBreaksARule(t *testing.T) {
	two := Description{
		Name: "Series", Work: Noun{"Series", "Series"},
		Levels: []Noun{{"Season", "Seasons"}, {"Part", "Parts"}}, Item: Noun{"Episode", "Episodes"},
	}
	cases := []struct {
		name string
		d    Description
		f    Fetched
		want string
	}{
		{"work label", records, Fetched{}, "artist a1: its label is empty"},
		{"level label", records, Fetched{Label: "A", Children: []Entry{{Key: "b1"}}}, "album b1: its label is empty"},
		{"item label", records, Fetched{Label: "A", Children: []Entry{{Key: "b1", Label: "B", Children: []Entry{{Key: "c1"}}}}}, "track c1: its label is empty"},
		{"empty key", records, Fetched{Label: "A", Children: []Entry{{Label: "B"}}}, "artist a1: album 1 has a key of 0 bytes; a key is 1 to 200 bytes"},
		{"long key", records, Fetched{Label: "A", Children: []Entry{{Key: "b1", Label: "B", Children: []Entry{{Key: "c1", Label: "C"}, {Key: Key(strings.Repeat("k", 201)), Label: "C"}}}}}, "album b1: track 2 has a key of 201 bytes; a key is 1 to 200 bytes"},
		{"two items", records, Fetched{Label: "A", Children: []Entry{
			{Key: "b1", Label: "B", Children: []Entry{{Key: "c1", Label: "C"}}},
			{Key: "b2", Label: "B", Children: []Entry{{Key: "c1", Label: "C"}}},
		}}, "album b2: two Items have key c1"},
		{"two levels", two, Fetched{Label: "A", Children: []Entry{
			{Key: "s1", Label: "S", Children: []Entry{{Key: "p1", Label: "P"}}},
			{Key: "s2", Label: "S", Children: []Entry{{Key: "p1", Label: "P"}}},
		}}, "season s2: two levels have key p1"},
		{"item children", records, Fetched{Label: "A", Children: []Entry{{Key: "b1", Label: "B", Children: []Entry{{Key: "c1", Label: "C", Children: []Entry{{Key: "d1", Label: "D"}}}}}}}, "track c1: an Item has no children, and it lists 1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseFetched(c.d, "a1", c.f)
			if want := "the metadata source sent a record SlipStream can't use: " + c.want; err == nil || err.Error() != want || !errors.Is(err, ErrInvalid) {
				t.Errorf("err = %v, want %q wrapping ErrInvalid", err, want)
			}
		})
	}
}

func TestParseFetchedHangsItemsOffAWorkWithNoLevels(t *testing.T) {
	flat := Description{Name: "Film", Work: Noun{"Film", "Films"}, Item: Noun{"Cut", "Cuts"}}
	got, err := parseFetched(flat, "f1", Fetched{Label: "Film", SortLabel: "film", Children: []Entry{{Key: "x", Label: "Theatrical"}}})
	want := parsed{Label: "Film", SortLabel: "film", nodes: nodes{Items: []node{{Key: "x", Label: "Theatrical", Position: 1}}}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("parsed = %+v, %v; want %+v", got, err, want)
	}
}

func TestParseDate(t *testing.T) {
	for in, want := range map[string]string{"1999": "1999", "1999-03": "1999-03", "2024-02-29": "2024-02-29"} {
		if d, err := ParseDate(in); err != nil || d.String() != want {
			t.Errorf("ParseDate(%q) = %v, %v; want %s", in, d, err, want)
		}
	}
	for in, want := range map[string]string{
		"1999-3":        `date "1999-3" is not YYYY, YYYY-MM or YYYY-MM-DD`,
		"1999-13":       `date "1999-13" is not a calendar date`,
		"2023-02-29":    `date "2023-02-29" is not a calendar date`,
		"1999-01-01-01": `date "1999-01-01-01" is not YYYY, YYYY-MM or YYYY-MM-DD`,
	} {
		if _, err := ParseDate(in); err == nil || err.Error() != want {
			t.Errorf("ParseDate(%q) = %v, want %q", in, err, want)
		}
	}
}
