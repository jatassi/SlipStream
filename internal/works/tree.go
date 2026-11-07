package works

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
)

type node struct {
	Key         Key
	Parent      Key
	Label       string
	Position    int
	WithdrawnAt storage.Time
	// Aside is the level's own mark, or an Item's outermost level's.
	Aside bool
}

func (n *node) entry() *node { return n }

type level struct {
	node
	Depth int
}

type nodes struct {
	Levels []level
	Items  []node
}

type parsed struct {
	Label     string
	SortLabel string
	nodes
	Detail      []byte
	Ratings     []Rating
	RatingsRead bool
	Artwork     []Art
}

func parseFetched(d Description, key Key, f Fetched) (parsed, error) {
	work := strings.ToLower(d.Work.One) + " " + string(key)
	if f.Label == "" {
		return parsed{}, invalid("%s: its label is empty", work)
	}
	p := parser{d: d, levels: map[Key]int{}, items: map[Key]bool{}}
	if err := p.entries(work, "", 1, false, f.Children); err != nil {
		return parsed{}, err
	}
	if err := p.ratings(work, f.Ratings); err != nil {
		return parsed{}, err
	}
	if err := p.artwork(work, f.Artwork); err != nil {
		return parsed{}, err
	}
	sortLabel := f.SortLabel
	if sortLabel == "" {
		sortLabel = f.Label
	}
	return parsed{
		Label: f.Label, SortLabel: sortLabel, nodes: p.out, Detail: f.Detail,
		Ratings: f.Ratings, RatingsRead: f.RatingsRead, Artwork: f.Artwork,
	}, nil
}

type parser struct {
	d Description
	// levels is each level's depth by key.
	levels map[Key]int
	items  map[Key]bool
	out    nodes
}

// entries parses es under parent, at depth, inside an aside level when
// aside is set.
func (p *parser) entries(parentName string, parent Key, depth int, aside bool, es []Entry) error {
	isItem := depth == len(p.d.Levels)+1
	noun, tier := p.d.Item, "Items"
	if !isItem {
		noun, tier = p.d.Levels[depth-1], "levels"
	}
	for i, e := range es {
		if len(e.Key) == 0 || len(e.Key) > maxKey {
			return invalid("%s: %s %d has a key of %d bytes. A key is 1 to %d bytes", parentName, strings.ToLower(noun.One), i+1, len(e.Key), maxKey)
		}
		name := strings.ToLower(noun.One) + " " + string(e.Key)
		_, seenLevel := p.levels[e.Key]
		switch {
		case isItem && p.items[e.Key], !isItem && seenLevel:
			return invalid("%s: two %s have key %s", parentName, tier, e.Key)
		case e.Label == "":
			return invalid("%s: its label is empty", name)
		case isItem && len(e.Children) > 0:
			return invalid("%s: an Item has no children, and it lists %d", name, len(e.Children))
		case e.Aside && (isItem || depth > 1):
			return invalid("%s: only an outermost level can be an aside", name)
		case e.Aside && p.d.Aside.One == "":
			return invalid("%s: it is marked an aside, and the type names no aside", name)
		}
		if depth == 1 {
			aside = e.Aside
		}
		n := node{Key: e.Key, Parent: parent, Label: e.Label, Position: i + 1, Aside: aside}
		if isItem {
			p.items[e.Key] = true
			p.out.Items = append(p.out.Items, n)
			continue
		}
		p.levels[e.Key] = depth
		p.out.Levels = append(p.out.Levels, level{node: n, Depth: depth})
		if err := p.entries(name, e.Key, depth+1, aside, e.Children); err != nil {
			return err
		}
	}
	return nil
}

// owner names the node level or item keys, the Work when both are empty,
// and the tier it sits at.
func (p *parser) owner(work string, level, item Key) (string, Tier, error) {
	switch {
	case level != "" && item != "":
		return "", 0, invalid("%s: it names both level %s and %s %s for one value", work, level, strings.ToLower(p.d.Item.One), item)
	case item != "":
		if !p.items[item] {
			return "", 0, invalid("%s: it names %s %s, which it doesn't list", work, strings.ToLower(p.d.Item.One), item)
		}
		return strings.ToLower(p.d.Item.One) + " " + string(item), TierItem, nil
	case level != "":
		depth, ok := p.levels[level]
		if !ok {
			return "", 0, invalid("%s: it names level %s, which it doesn't list", work, level)
		}
		return strings.ToLower(p.d.Levels[depth-1].One) + " " + string(level), TierLevel(depth), nil
	}
	return work, TierWork, nil
}

func (p *parser) ratings(work string, rs []Rating) error {
	type slot struct {
		item   Key
		source RatingSource
	}
	seen := map[slot]bool{}
	for _, r := range rs {
		name, _, err := p.owner(work, "", r.Item)
		if err != nil {
			return err
		}
		scale, ok := RatingSources[r.Source]
		switch {
		case !ok:
			return invalid("%s: it has a rating from %q, which is no ratings source", name, r.Source)
		case math.IsNaN(r.Value) || r.Value < 0 || r.Value > scale.Scale:
			return invalid("%s: its %s rating %v is outside 0 to %v", name, scale.Label, r.Value, scale.Scale)
		case r.Votes != nil && *r.Votes < 0:
			return invalid("%s: its %s rating has %d votes", name, scale.Label, *r.Votes)
		case seen[slot{r.Item, r.Source}]:
			return invalid("%s: it has two %s ratings", name, scale.Label)
		}
		seen[slot{r.Item, r.Source}] = true
	}
	return nil
}

func (p *parser) artwork(work string, as []Art) error {
	type slot struct {
		level, item Key
		kind        string
	}
	seen := map[slot]bool{}
	for _, a := range as {
		name, tier, err := p.owner(work, a.Level, a.Item)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(p.d.Artwork, func(k artwork.Kind) bool { return k.Key == a.Kind && k.Tier == tier })
		switch {
		case i < 0:
			return invalid("%s: it has %q artwork, a kind it can't have", name, a.Kind)
		case a.Ref.Source == "" || a.Ref.Path == "":
			return invalid("%s: its %s artwork names no source or no path", name, a.Kind)
		case a.Ref.Shape != p.d.Artwork[i].Shape:
			return invalid("%s: its %s artwork is a %s, not a %s", name, a.Kind, a.Ref.Shape, p.d.Artwork[i].Shape)
		case seen[slot{a.Level, a.Item, a.Kind}]:
			return invalid("%s: it has two %s artworks", name, a.Kind)
		}
		seen[slot{a.Level, a.Item, a.Kind}] = true
	}
	return nil
}

// tierOf is the tier a's owner sits at, in a parsed record.
func (p parsed) tierOf(a Art) Tier {
	switch {
	case a.Item != "":
		return TierItem
	case a.Level != "":
		i := slices.IndexFunc(p.Levels, func(l level) bool { return l.Key == a.Level })
		return TierLevel(p.Levels[i].Depth)
	}
	return TierWork
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

type change[N any] struct {
	Node   N
	Insert bool
}

type plan struct {
	Label       string
	SortLabel   string
	RefreshedAt storage.Time
	Levels      []change[level]
	Items       []change[node]
}

func planMerge(stored nodes, fetched parsed, now storage.Time) plan {
	return plan{
		Label:       fetched.Label,
		SortLabel:   fetched.SortLabel,
		RefreshedAt: now,
		Levels:      mergeTier(stored.Levels, fetched.Levels, now),
		Items:       mergeTier(stored.Items, fetched.Items, now),
	}
}

type tierNode[N any] interface {
	*N
	entry() *node
}

func mergeTier[N comparable, P tierNode[N]](stored, fetched []N, now storage.Time) []change[N] {
	byKey := make(map[Key]N, len(stored))
	for _, n := range stored {
		byKey[P(&n).entry().Key] = n
	}
	listed := make(map[Key]bool, len(fetched))
	var out []change[N]
	for _, n := range fetched {
		key := P(&n).entry().Key
		listed[key] = true
		old, ok := byKey[key]
		if !ok || old != n {
			out = append(out, change[N]{Node: n, Insert: !ok})
		}
	}
	for _, n := range stored {
		if e := P(&n).entry(); !listed[e.Key] && e.WithdrawnAt.IsZero() {
			e.WithdrawnAt = now
			out = append(out, change[N]{Node: n})
		}
	}
	return out
}
