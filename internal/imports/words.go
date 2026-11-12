package imports

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jatassi/slipstream/internal/files"
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

// FileWords is what Pending imports reads about one file of an Import.
// Sentence is its reason worded, empty for none. A file that waits on its
// verdict is judged alone against its Work's files now: Verdicts is each
// Target's verdict on it, in Target order, and Forces the Targets its row
// offers Force into.
type FileWords struct {
	Sentence string
	Verdicts []TargetVerdict
	Forces   []ForceOption
}

// TargetVerdict is a Target's verdict on a file in one sentence.
type TargetVerdict struct {
	Target   targets.TargetID
	Sentence string
}

// ForceOption is a cell Force puts a file into, and the root-relative
// paths of the library files that deletes because they fill nothing else.
type ForceOption struct {
	Cell    files.CellKey
	Deletes []string
}

// judgedWaits are the waits whose Details read each Target's verdict.
var judgedWaits = []ReasonCode{NotAnUpgrade, MeetsNothing, Confirm}

// Words words each of imp's files by position, read for its Work: with
// none, only the sentences that name nothing of one.
func (im *Imports) Words(ctx context.Context, by identity.Admin, imp Import) (map[int]FileWords, error) {
	ts, err := im.cfg.Targets.ListTargets(ctx, by)
	if err != nil {
		return nil, err
	}
	names := make(map[targets.TargetID]string, len(ts))
	for _, t := range ts {
		names[t.ID] = t.Name
	}
	out := map[int]FileWords{}
	err = im.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		w, err := im.wordingOf(ctx, r, imp, names)
		if err != nil {
			return err
		}
		for _, f := range imp.Files {
			if f.Reason == nil {
				continue
			}
			fw := FileWords{Sentence: f.Reason.sentence(w)}
			if f.Outcome == Waiting && slices.Contains(judgedWaits, f.Reason.Code) {
				if fw.Verdicts, fw.Forces, err = im.judgeAlone(ctx, r, &imp, f, ts, w); err != nil {
					return err
				}
			}
			out[f.Position] = fw
		}
		return nil
	})
	return out, err
}

// wordingOf is the wording of imp's Work in r, with names for its Targets.
func (im *Imports) wordingOf(ctx context.Context, r storage.Reader[file], imp Import, names map[targets.TargetID]string) (treeWording, error) {
	cuts, err := im.cfg.Targets.CutsIn(ctx, r)
	if err != nil {
		return treeWording{}, err
	}
	var tree works.Tree
	if !imp.Work.IsZero() {
		if tree, err = im.cfg.Works.TreeIn(ctx, r, imp.Work); err != nil {
			return treeWording{}, err
		}
	}
	nouns, _ := im.cfg.Works.Describe(imp.ContentType)
	return newWording(tree, nouns, names, cuts.For(imp.ContentType)), nil
}

// judgeAlone judges f alone against its Work's files in r: each of ts's
// verdicts on it, and, when its row offers Force, each Target it can be
// forced into with the files that deletes.
func (im *Imports) judgeAlone(ctx context.Context, r storage.Reader[file], imp *Import, f File, ts []targets.Target, w treeWording) ([]TargetVerdict, []ForceOption, error) {
	root, err := imp.rootOf(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	cands, err := im.candidates(ctx, r, imp, root.ID, []int{indexOf(imp, f.Position)})
	if err != nil {
		return nil, nil, err
	}
	judge := func(force []files.CellKey) (files.Verdict, error) {
		c := cands[0]
		c.Force = force
		vs, err := im.cfg.Files.Judge(ctx, r, files.Batch{Work: imp.Work, Root: root.ID, Import: imp.ID.UUID, Candidates: []files.Candidate{c}})
		if err != nil {
			return files.Verdict{}, err
		}
		return vs[0], nil
	}
	v, err := judge(cands[0].Force)
	if err != nil {
		return nil, nil, err
	}
	var verdicts []TargetVerdict
	for _, t := range ts {
		if _, ok := v.Reasons[t.ID]; ok {
			verdicts = append(verdicts, TargetVerdict{Target: t.ID, Sentence: verdictSentence(v, t.ID, w)})
		}
	}
	if _, also := f.Actions(*imp); !slices.Contains(also, ActForce) {
		return verdicts, nil, nil
	}
	var forces []ForceOption
	for _, cell := range v.Meets {
		if _, lost := v.Lost[cell]; !lost || slices.ContainsFunc(forces, func(o ForceOption) bool { return o.Cell.Target == cell.Target }) {
			continue
		}
		forced, err := judge([]files.CellKey{cell})
		if err != nil {
			return nil, nil, err
		}
		o := ForceOption{Cell: cell}
		for _, d := range forced.Replaces {
			o.Deletes = append(o.Deletes, d.Path)
		}
		forces = append(forces, o)
	}
	return verdicts, forces, nil
}

// verdictSentence is target's verdict on v's file: it fills a cell of
// target, a file the Work has holds the cells it meets, or it doesn't
// meet target.
func verdictSentence(v files.Verdict, target targets.TargetID, w treeWording) string {
	of := func(cells []files.CellKey) bool {
		return slices.ContainsFunc(cells, func(c files.CellKey) bool { return c.Target == target })
	}
	name := w.names[target]
	switch {
	case of(v.Fills):
		return "Fills " + name + " for " + w.label() + "."
	case of(v.Meets):
		return "This isn't better than the file you have for " + name + "."
	}
	return "This doesn't meet " + name + "."
}
