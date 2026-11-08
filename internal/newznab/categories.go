// Package newznab holds the Newznab category table and the mapping between
// an Indexer's own categories and Newznab's, as Prowlarr builds it.
package newznab

import (
	"crypto/sha1" //nolint:gosec // Prowlarr numbers custom categories by SHA-1, and the ids must match its
	"encoding/binary"
	"slices"
	"strconv"
	"strings"
)

// ID is a Newznab category id: 2000 is Movies, 2040 Movies/HD, 100000 and
// up a site's own.
type ID int32

// Category is a Newznab category.
type Category struct {
	ID     ID
	Name   string
	Parent ID // 0 for a top-level category, and for 10 and 20 under 0
}

// CustomBase is the first id of a site's own categories.
const CustomBase ID = 100000

// Standard is Newznab's categories in id order.
func Standard() []Category {
	out := slices.Clone(allCats)
	slices.SortFunc(out, func(a, b Category) int { return int(a.ID - b.ID) })
	return out
}

// ByName is the standard category with exactly this name, case-sensitive,
// as Prowlarr's caps parser finds it.
func ByName(name string) (Category, bool) {
	i := slices.IndexFunc(allCats, func(c Category) bool { return c.Name == name })
	if i < 0 {
		return Category{}, false
	}
	return allCats[i], true
}

// Lookup is the standard category with this id.
func Lookup(id ID) (Category, bool) {
	i := slices.IndexFunc(allCats, func(c Category) bool { return c.ID == id })
	if i < 0 {
		return Category{}, false
	}
	return allCats[i], true
}

// isParent reports whether id is one of Prowlarr's ParentCats. Other is
// there twice, as 8000 and as 0.
func isParent(id ID) bool { return id >= 0 && id < 10000 && id%1000 == 0 }

// Family is id and every standard category under it: Family(2000) is 2000
// and 2010 to 2090. A subcategory's or a site's own category's family is
// itself.
func Family(id ID) []ID {
	out := []ID{id}
	if !isParent(id) {
		return out
	}
	for _, c := range Standard() {
		if c.Parent == id && !isParent(c.ID) {
			out = append(out, c.ID)
		}
	}
	return out
}

// Meets reports whether any of have is in want or under one of want's
// families.
func Meets(have, want []ID) bool {
	for _, w := range want {
		for _, f := range Family(w) {
			if slices.Contains(have, f) {
				return true
			}
		}
	}
	return false
}

// Custom is a site category's id the way Prowlarr's AddCategoryMapping
// numbers it: 100000 plus the id when it is an integer, else 100000 plus
// the first two bytes of SHA-1 of the id read little-endian.
func Custom(siteID string) ID {
	if n, err := strconv.ParseInt(strings.TrimSpace(siteID), 10, 32); err == nil {
		return ID(n) + CustomBase
	}
	sum := sha1.Sum([]byte(siteID)) //nolint:gosec // matches Prowlarr's numbering, not a security use
	return ID(binary.LittleEndian.Uint16(sum[:2])) + CustomBase
}
