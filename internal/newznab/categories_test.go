package newznab

import (
	"slices"
	"testing"
)

func TestStandard(t *testing.T) {
	std := Standard()
	if len(std) != 76 {
		t.Fatalf("%d standard categories, want 76", len(std))
	}
	if !slices.IsSortedFunc(std, func(a, b Category) int { return int(a.ID - b.ID) }) {
		t.Error("Standard isn't in id order")
	}
	if c, ok := Lookup(2040); !ok || c.Name != "Movies/HD" || c.Parent != 2000 {
		t.Errorf("Lookup(2040) = %+v, %v", c, ok)
	}
	if c, ok := Lookup(20); !ok || c.Name != "Other/Hashed" || c.Parent != 0 {
		t.Errorf("Lookup(20) = %+v, %v", c, ok)
	}
}

func TestByName(t *testing.T) {
	tests := []struct {
		name string
		want ID
		ok   bool
	}{
		{"Movies/HD", 2040, true},
		{"Other", 8000, true},
		{"Other/Misc", 8010, true},
		{"movies/hd", 0, false},
		{"Movies/8K", 0, false},
	}
	for _, tt := range tests {
		c, ok := ByName(tt.name)
		if ok != tt.ok || c.ID != tt.want {
			t.Errorf("ByName(%q) = %d, %v; want %d, %v", tt.name, c.ID, ok, tt.want, tt.ok)
		}
	}
}

func TestFamily(t *testing.T) {
	tests := []struct {
		id   ID
		want []ID
	}{
		{2000, []ID{2000, 2010, 2020, 2030, 2040, 2045, 2050, 2060, 2070, 2080, 2090}},
		{8000, []ID{8000, 8010, 8020}},
		{0, []ID{0, 10, 20}},
		{2040, []ID{2040}},
		{100001, []ID{100001}},
	}
	for _, tt := range tests {
		if got := Family(tt.id); !slices.Equal(got, tt.want) {
			t.Errorf("Family(%d) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestMeets(t *testing.T) {
	tests := []struct {
		have, want []ID
		meets      bool
	}{
		{[]ID{2040}, []ID{2000}, true},
		{[]ID{2040}, []ID{2040}, true},
		{[]ID{2000}, []ID{2040}, false},
		{[]ID{5040}, []ID{2000}, false},
		{[]ID{100001}, []ID{100001}, true},
		{nil, []ID{2000}, false},
	}
	for _, tt := range tests {
		if got := Meets(tt.have, tt.want); got != tt.meets {
			t.Errorf("Meets(%v, %v) = %v", tt.have, tt.want, got)
		}
	}
}

func TestCustom(t *testing.T) {
	tests := []struct {
		siteID string
		want   ID
	}{
		{"5", 100005},
		{" 12 ", 100012},
		{"-3", 99997},
		{"movies", 100000 + 0x3198},
	}
	for _, tt := range tests {
		if got := Custom(tt.siteID); got != tt.want {
			t.Errorf("Custom(%q) = %d, want %d", tt.siteID, got, tt.want)
		}
	}
}

func TestCategories(t *testing.T) {
	var c Categories
	movies, _ := ByName("Movies")
	hd, _ := ByName("Movies/HD")
	tvhd, _ := ByName("TV/HD")
	c.Add("1", movies, "")
	c.Add("2", hd, "HD films")
	c.Add("x", tvhd, "")
	c.Add("y", Category{ID: 2999, Name: "Movies/Odd"}, "")

	tree := c.Tree()
	var roots []ID
	for _, n := range tree {
		roots = append(roots, n.ID)
	}
	if want := []ID{2000, 100002, 5000}; !slices.Equal(roots, want) {
		t.Errorf("tree roots = %v, want %v", roots, want)
	}
	var subs []ID
	for _, s := range tree[0].Subs {
		subs = append(subs, s.ID)
	}
	if want := []ID{2040, 2999}; !slices.Equal(subs, want) {
		t.Errorf("Movies subcategories = %v, want %v", subs, want)
	}

	tests := []struct {
		query []ID
		want  []string
	}{
		{[]ID{2000}, []string{"1", "2", "y"}},
		{[]ID{2040}, []string{"2"}},
		{[]ID{100002}, []string{"2"}},
		{[]ID{5000}, []string{"x"}},
		{[]ID{3000}, nil},
	}
	for _, tt := range tests {
		if got := c.TrackerIDs(tt.query); !slices.Equal(got, tt.want) {
			t.Errorf("TrackerIDs(%v) = %v, want %v", tt.query, got, tt.want)
		}
	}

	ids := func(cats []Category) []ID {
		var out []ID
		for _, c := range cats {
			out = append(out, c.ID)
		}
		return out
	}
	if got := ids(c.FromTracker("2")); !slices.Equal(got, []ID{2040, 100002}) {
		t.Errorf("FromTracker(2) = %v", got)
	}
	if got := ids(c.FromTracker("X")); !slices.Equal(got, []ID{5040}) {
		t.Errorf("FromTracker(X) = %v", got)
	}
	if got := ids(c.FromDesc("hd FILMS")); !slices.Equal(got, []ID{2040, 100002}) {
		t.Errorf("FromDesc = %v", got)
	}
	if got := c.FromTracker(" "); got != nil {
		t.Errorf("FromTracker(blank) = %v", got)
	}
}
