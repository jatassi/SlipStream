package works

import (
	"fmt"
	"strings"

	"github.com/jatassi/slipstream/internal/storage"
)

type node struct {
	Key         Key
	Parent      Key
	Label       string
	Position    int
	WithdrawnAt storage.Time
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
		p.out.Levels = append(p.out.Levels, level{node: n, Depth: depth})
		if err := p.entries(name, e.Key, depth+1, e.Children); err != nil {
			return err
		}
	}
	return nil
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
