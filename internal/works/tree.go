package works

import (
	"fmt"
	"strings"

	"github.com/jatassi/slipstream/internal/storage"
)

type node struct {
	Key         Key
	Parent      Key
	Depth       int
	Label       string
	Position    int
	WithdrawnAt storage.Time
}

type nodes struct {
	Levels []node
	Items  []node
}

type parsed struct {
	Label     string
	SortLabel string
	nodes
	Detail []byte
}

func parseFetched(d Description, key Key, f Fetched) (parsed, error) {
	work := strings.ToLower(d.Work.One) + " " + string(key)
	if f.Label == "" {
		return parsed{}, invalid("%s: its label is empty", work)
	}
	p := parser{d: d, levels: map[Key]bool{}, items: map[Key]bool{}}
	if err := p.entries(work, "", 1, f.Children); err != nil {
		return parsed{}, err
	}
	sortLabel := f.SortLabel
	if sortLabel == "" {
		sortLabel = f.Label
	}
	return parsed{Label: f.Label, SortLabel: sortLabel, nodes: p.out, Detail: f.Detail}, nil
}

type parser struct {
	d      Description
	levels map[Key]bool
	items  map[Key]bool
	out    nodes
}

func (p *parser) entries(parentName string, parent Key, depth int, es []Entry) error {
	isItem := depth == len(p.d.Levels)+1
	noun, seen, tier := p.d.Item, p.items, "Items"
	if !isItem {
		noun, seen, tier = p.d.Levels[depth-1], p.levels, "levels"
	}
	for i, e := range es {
		if len(e.Key) == 0 || len(e.Key) > maxKey {
			return invalid("%s: %s %d has a key of %d bytes; a key is 1 to %d bytes", parentName, strings.ToLower(noun.One), i+1, len(e.Key), maxKey)
		}
		name := strings.ToLower(noun.One) + " " + string(e.Key)
		switch {
		case seen[e.Key]:
			return invalid("%s: two %s have key %s", parentName, tier, e.Key)
		case e.Label == "":
			return invalid("%s: its label is empty", name)
		case isItem && len(e.Children) > 0:
			return invalid("%s: an Item has no children, and it lists %d", name, len(e.Children))
		}
		seen[e.Key] = true
		n := node{Key: e.Key, Parent: parent, Label: e.Label, Position: i + 1}
		if isItem {
			p.out.Items = append(p.out.Items, n)
			continue
		}
		n.Depth = depth
		p.out.Levels = append(p.out.Levels, n)
		if err := p.entries(name, e.Key, depth+1, e.Children); err != nil {
			return err
		}
	}
	return nil
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

type change struct {
	node
	Insert bool
}

type plan struct {
	Label       string
	SortLabel   string
	RefreshedAt storage.Time
	Levels      []change
	Items       []change
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

func mergeTier(stored, fetched []node, now storage.Time) []change {
	byKey := make(map[Key]node, len(stored))
	for _, n := range stored {
		byKey[n.Key] = n
	}
	listed := make(map[Key]bool, len(fetched))
	var out []change
	for _, n := range fetched {
		listed[n.Key] = true
		old, ok := byKey[n.Key]
		if !ok || old != n {
			out = append(out, change{node: n, Insert: !ok})
		}
	}
	for _, n := range stored {
		if !listed[n.Key] && n.WithdrawnAt.IsZero() {
			n.WithdrawnAt = now
			out = append(out, change{node: n})
		}
	}
	return out
}
