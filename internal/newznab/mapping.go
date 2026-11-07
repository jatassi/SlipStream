package newznab

import (
	"slices"
	"strings"
)

// Mapping pairs one of an Indexer's categories with a Newznab category.
type Mapping struct {
	Tracker string
	Desc    string
	Newznab ID
}

// Node is one entry of an Indexer's category tree: a parent category and the
// subcategories the Indexer maps to it.
type Node struct {
	Category
	Subs []Category
}

// Categories is an Indexer's category mapping and the Newznab tree it builds,
// Prowlarr's IndexerCapabilitiesCategories.
type Categories struct {
	mappings []Mapping
	tree     []Node
}

// Add maps the tracker's category to cat. A non-empty desc also adds the
// Indexer's own category, numbered by Custom, as Prowlarr does when a
// mapping has a desc.
func (c *Categories) Add(tracker string, cat Category, desc string) {
	c.mappings = append(c.mappings, Mapping{Tracker: tracker, Desc: desc, Newznab: cat.ID})
	c.addToTree(cat)
	if desc == "" {
		return
	}
	custom := Category{ID: Custom(tracker), Name: desc}
	c.mappings = append(c.mappings, Mapping{Tracker: tracker, Desc: desc, Newznab: custom.ID})
	c.addToTree(custom)
}

// Mappings are the pairs Add recorded, in order.
func (c *Categories) Mappings() []Mapping { return c.mappings }

// Tree is the Newznab tree the mappings build, in the order Prowlarr builds
// it.
func (c *Categories) Tree() []Node { return c.tree }

func (c *Categories) addToTree(cat Category) {
	if isParent(cat.ID) {
		if c.node(cat.ID) == nil {
			c.tree = append(c.tree, Node{Category: cat})
		}
		return
	}
	parent, ok := parentOf(cat)
	if !ok {
		c.tree = append(c.tree, Node{Category: cat})
		return
	}
	n := c.node(parent)
	if n == nil {
		p, _ := Lookup(parent)
		c.tree = append(c.tree, Node{Category: p})
		n = &c.tree[len(c.tree)-1]
	}
	if n.ID != cat.ID && !slices.ContainsFunc(n.Subs, func(s Category) bool { return s.ID == cat.ID }) {
		n.Subs = append(n.Subs, cat)
	}
}

func (c *Categories) node(id ID) *Node {
	for i := range c.tree {
		if c.tree[i].ID == id {
			return &c.tree[i]
		}
	}
	return nil
}

// TrackerIDs are the tracker categories mapped to any of the query's
// Newznab categories, a parent standing for its subcategories too, in
// mapping order without repeats: Prowlarr's MapTorznabCapsToTrackers.
func (c *Categories) TrackerIDs(query []ID) []string {
	var expanded []ID
	for _, q := range query {
		expanded = append(expanded, q)
		if q >= CustomBase {
			continue
		}
		if n := c.node(q); n != nil {
			for _, s := range n.Subs {
				expanded = append(expanded, s.ID)
			}
		}
	}
	var out []string
	for _, m := range c.mappings {
		if slices.Contains(expanded, m.Newznab) && !slices.Contains(out, m.Tracker) {
			out = append(out, m.Tracker)
		}
	}
	return out
}

// FromTracker are the Newznab categories mapped from the tracker's category,
// compared case-insensitively: Prowlarr's MapTrackerCatToNewznab.
func (c *Categories) FromTracker(tracker string) []Category {
	return c.lookup(tracker, func(m Mapping) string { return m.Tracker })
}

// FromDesc are the Newznab categories mapped from the tracker's category
// description, compared case-insensitively: Prowlarr's
// MapTrackerCatDescToNewznab.
func (c *Categories) FromDesc(desc string) []Category {
	return c.lookup(desc, func(m Mapping) string { return m.Desc })
}

func (c *Categories) lookup(key string, field func(Mapping) string) []Category {
	if strings.TrimSpace(key) == "" {
		return nil
	}
	var out []Category
	for _, m := range c.mappings {
		v := field(m)
		if strings.TrimSpace(v) == "" || !strings.EqualFold(v, key) {
			continue
		}
		cat, ok := Lookup(m.Newznab)
		if !ok {
			cat = Category{ID: m.Newznab}
		}
		out = append(out, cat)
	}
	return out
}

// parentOf is the parent Prowlarr's AddTorznabCategoryTree files cat
// under: a standard category's own, or for an unknown id between 1000 and
// 9999 the thousand it falls in.
func parentOf(cat Category) (ID, bool) {
	if std, ok := Lookup(cat.ID); ok {
		return std.Parent, true
	}
	if cat.ID > 1000 && cat.ID < 10000 {
		return cat.ID / 1000 * 1000, true
	}
	return 0, false
}
