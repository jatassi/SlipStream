package imports

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// treeWording words reasons for one Work: its tree, its Content Type's
// nouns, and the names of the Targets and cuts it may name.
type treeWording struct {
	tree    works.Tree
	nouns   works.Description
	names   map[targets.TargetID]string
	cuts    []release.Cut
	numbers map[works.ItemID]int
}

func newWording(tree works.Tree, nouns works.Description, names map[targets.TargetID]string, cuts []release.Cut) treeWording {
	w := treeWording{tree: tree, nouns: nouns, names: names, cuts: cuts, numbers: map[works.ItemID]int{}}
	place := map[works.LevelID]int{}
	for _, it := range tree.Items {
		place[it.LevelID]++
		w.numbers[it.ID] = place[it.LevelID]
	}
	return w
}

func (w treeWording) targets(ids []targets.TargetID) string {
	names := make([]string, len(ids))
	for i, id := range ids {
		names[i] = w.names[id]
	}
	return joinWords(slices.Compact(names))
}

// items is ids in words, a run of three or more numbers in a row written
// "episodes 1 to 4", with each level's words when they span more than one.
func (w treeWording) items(ids []works.ItemID) string {
	if w.nouns.Items == works.ItemsOne {
		return w.tree.Label
	}
	var order []works.LevelID
	byLevel := map[works.LevelID][]int{}
	for _, it := range w.tree.Items {
		if !slices.Contains(ids, it.ID) {
			continue
		}
		if _, ok := byLevel[it.LevelID]; !ok {
			order = append(order, it.LevelID)
		}
		byLevel[it.LevelID] = append(byLevel[it.LevelID], w.numbers[it.ID])
	}
	parts := make([]string, len(order))
	for i, l := range order {
		parts[i] = numbersWords(w.nouns.Item, byLevel[l])
		if len(order) > 1 {
			parts[i] += " of " + w.level(l)
		}
	}
	return joinWords(parts)
}

func numbersWords(noun works.Noun, ns []int) string {
	var runs []string
	for i := 0; i < len(ns); {
		k := i
		for k+1 < len(ns) && ns[k+1] == ns[k]+1 {
			k++
		}
		if k-i >= 2 {
			runs = append(runs, fmt.Sprintf("%d to %d", ns[i], ns[k]))
		} else {
			for _, n := range ns[i : k+1] {
				runs = append(runs, strconv.Itoa(n))
			}
		}
		i = k + 1
	}
	if len(ns) == 1 {
		return noun.One + " " + runs[0]
	}
	return noun.Many + " " + joinWords(runs)
}

func (w treeWording) level(id works.LevelID) string {
	i := slices.IndexFunc(w.tree.Levels, func(l works.Level) bool { return l.ID == id })
	if i < 0 {
		return ""
	}
	label := w.tree.Levels[i].Label
	r, n := utf8.DecodeRuneInString(label)
	return string(unicode.ToLower(r)) + label[n:]
}

func (w treeWording) edition(cut release.CutKey) string {
	if i := slices.IndexFunc(w.cuts, func(c release.Cut) bool { return c.Key == cut }); i >= 0 {
		return w.cuts[i].Name
	}
	return string(cut)
}

func (w treeWording) item() string  { return w.nouns.Item.One }
func (w treeWording) work() string  { return strings.ToLower(w.nouns.Work.One) }
func (w treeWording) label() string { return w.tree.Label }

// Sentences words each of imp's files whose reason has a sentence, by
// position, read for its Work: with none, only the sentences that name
// nothing of one.
func (im *Imports) Sentences(ctx context.Context, by identity.Admin, imp Import) (map[int]string, error) {
	ts, err := im.cfg.Targets.ListTargets(ctx, by)
	if err != nil {
		return nil, err
	}
	names := make(map[targets.TargetID]string, len(ts))
	for _, t := range ts {
		names[t.ID] = t.Name
	}
	var w treeWording
	err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		cuts, err := im.cfg.Targets.CutsIn(ctx, r)
		if err != nil {
			return err
		}
		var tree works.Tree
		if !imp.Work.IsZero() {
			if tree, err = im.cfg.Works.TreeIn(ctx, r, imp.Work); err != nil {
				return err
			}
		}
		nouns, _ := im.cfg.Works.Describe(imp.ContentType)
		w = newWording(tree, nouns, names, cuts.For(imp.ContentType))
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := map[int]string{}
	for _, f := range imp.Files {
		if f.Reason == nil {
			continue
		}
		if s := f.Reason.sentence(w); s != "" {
			out[f.Position] = s
		}
	}
	return out, nil
}
