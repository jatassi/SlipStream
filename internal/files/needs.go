package files

import (
	"cmp"
	"iter"
	"slices"

	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// Need is one entry of a Work's Needs you.
type Need struct {
	Kind NeedKind
	// Items is a run of the Work's Items, consecutive in tree order, whose
	// entries are the same; none for NeedNoRoot.
	Items   []works.ItemID
	Edition targets.Edition
	// Target is the first failed Target of a NeedFailed, in Profile order,
	// and Waiting the Targets of its Items and Edition still missing.
	Target  targets.TargetID
	Waiting []targets.TargetID
	// Upgrades is every upgradable Target of a NeedUpgrade's Items and
	// Edition.
	Upgrades []targets.TargetID
	// Grab is the Grab that failed, or the one that waits; Code is what it
	// waits on.
	Grab storage.UUID
	Code string
}

// NeedKind is a Need's kind, in the order Needs you lists them.
type NeedKind uint8

const (
	NeedFailed NeedKind = iota + 1
	NeedUpgrade
	NeedWaiting
	NeedNoRoot
)

// needs is a Work's Needs you from its cells, in tree order, and the Grab
// of each. Each Item and Edition gives one entry from its wanted cells: a
// failure, which its still missing Targets wait on, else its upgrades.
// Each Grab that waits on an Admin gives one, and a rootless Work with a
// wanted cell one more. Entries sort by kind, then tree order, and an
// entry the same as one for the Item before it joins that one's run.
func needs(items []works.Item, cells []Cell, grabs map[CellKey]GrabCell, rooted bool) []Need {
	var out []Need
	waits := map[storage.UUID]bool{}
	wanted := false
	for group := range groupsOf(cells) {
		var failed, missing, upgrades []targets.TargetID
		for _, c := range group {
			g := grabs[c.key()]
			if g.Waiting != "" && !waits[g.Grab] {
				waits[g.Grab] = true
				out = append(out, Need{Kind: NeedWaiting, Items: []works.ItemID{c.Item.ID}, Edition: c.Edition, Grab: g.Grab, Code: g.Waiting})
			}
			if !c.Wanted() {
				continue
			}
			wanted = true
			switch c.State {
			case Failed:
				failed = append(failed, c.Target.ID)
			case Missing:
				missing = append(missing, c.Target.ID)
			case Upgradable:
				upgrades = append(upgrades, c.Target.ID)
			case Unreleased, Downloading, Met:
			}
		}
		head := group[0]
		at := []works.ItemID{head.Item.ID}
		switch {
		case len(failed) > 0:
			grab := grabs[CellKey{Item: head.Item.ID, Edition: head.Edition, Target: failed[0]}].Grab
			out = append(out, Need{Kind: NeedFailed, Items: at, Edition: head.Edition, Target: failed[0], Waiting: missing, Grab: grab})
		case len(upgrades) > 0:
			out = append(out, Need{Kind: NeedUpgrade, Items: at, Edition: head.Edition, Upgrades: upgrades})
		}
	}
	if !rooted && wanted {
		out = append(out, Need{Kind: NeedNoRoot})
	}
	slices.SortStableFunc(out, func(a, b Need) int { return cmp.Compare(a.Kind, b.Kind) })
	return runs(items, out)
}

// groupsOf yields each run of cells of one Item and Edition.
func groupsOf(cells []Cell) iter.Seq[[]Cell] {
	return func(yield func([]Cell) bool) {
		for len(cells) > 0 {
			n := 1 + slices.IndexFunc(cells[1:], func(c Cell) bool { return c.Item.ID != cells[0].Item.ID || c.Edition != cells[0].Edition })
			if n == 0 {
				n = len(cells)
			}
			if !yield(cells[:n]) {
				return
			}
			cells = cells[n:]
		}
	}
}

// runs joins each one-Item entry of ns to an entry the same as it whose
// run ends at the Item before.
func runs(items []works.Item, ns []Need) []Need {
	index := make(map[works.ItemID]int, len(items))
	for i, it := range items {
		index[it.ID] = i
	}
	var out []Need
	for _, n := range ns {
		i := slices.IndexFunc(out, func(o Need) bool {
			return len(n.Items) == 1 && len(o.Items) > 0 && index[o.Items[len(o.Items)-1]]+1 == index[n.Items[0]] && same(o, n)
		})
		if i < 0 {
			out = append(out, n)
			continue
		}
		out[i].Items = append(out[i].Items, n.Items[0])
	}
	return out
}

func same(a, b Need) bool {
	return a.Kind == b.Kind && a.Edition == b.Edition && a.Target == b.Target && a.Grab == b.Grab && a.Code == b.Code &&
		slices.Equal(a.Waiting, b.Waiting) && slices.Equal(a.Upgrades, b.Upgrades)
}

func (c Cell) key() CellKey { return CellKey{Item: c.Item.ID, Edition: c.Edition, Target: c.Target.ID} }
