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

// allCats is Prowlarr's NewznabStandardCategory.AllCats in its order, so a
// lookup by name finds the entry Prowlarr's does: Other is 8000, not 0.
var allCats = []Category{
	{1000, "Console", 0}, {1010, "Console/NDS", 1000}, {1020, "Console/PSP", 1000},
	{1030, "Console/Wii", 1000}, {1040, "Console/XBox", 1000}, {1050, "Console/XBox 360", 1000},
	{1060, "Console/Wiiware", 1000}, {1070, "Console/XBox 360 DLC", 1000}, {1080, "Console/PS3", 1000},
	{1090, "Console/Other", 1000}, {1110, "Console/3DS", 1000}, {1120, "Console/PS Vita", 1000},
	{1130, "Console/WiiU", 1000}, {1140, "Console/XBox One", 1000}, {1180, "Console/PS4", 1000},
	{2000, "Movies", 0}, {2010, "Movies/Foreign", 2000}, {2020, "Movies/Other", 2000},
	{2030, "Movies/SD", 2000}, {2040, "Movies/HD", 2000}, {2045, "Movies/UHD", 2000},
	{2050, "Movies/BluRay", 2000}, {2060, "Movies/3D", 2000}, {2070, "Movies/DVD", 2000},
	{2080, "Movies/WEB-DL", 2000}, {2090, "Movies/x265", 2000},
	{3000, "Audio", 0}, {3010, "Audio/MP3", 3000}, {3020, "Audio/Video", 3000},
	{3030, "Audio/Audiobook", 3000}, {3040, "Audio/Lossless", 3000}, {3050, "Audio/Other", 3000},
	{3060, "Audio/Foreign", 3000},
	{4000, "PC", 0}, {4010, "PC/0day", 4000}, {4020, "PC/ISO", 4000}, {4030, "PC/Mac", 4000},
	{4040, "PC/Mobile-Other", 4000}, {4050, "PC/Games", 4000}, {4060, "PC/Mobile-iOS", 4000},
	{4070, "PC/Mobile-Android", 4000},
	{5000, "TV", 0}, {5010, "TV/WEB-DL", 5000}, {5020, "TV/Foreign", 5000}, {5030, "TV/SD", 5000},
	{5040, "TV/HD", 5000}, {5045, "TV/UHD", 5000}, {5050, "TV/Other", 5000}, {5060, "TV/Sport", 5000},
	{5070, "TV/Anime", 5000}, {5080, "TV/Documentary", 5000}, {5090, "TV/x265", 5000},
	{6000, "XXX", 0}, {6010, "XXX/DVD", 6000}, {6020, "XXX/WMV", 6000}, {6030, "XXX/XviD", 6000},
	{6040, "XXX/x264", 6000}, {6045, "XXX/UHD", 6000}, {6050, "XXX/Pack", 6000},
	{6060, "XXX/ImageSet", 6000}, {6070, "XXX/Other", 6000}, {6080, "XXX/SD", 6000},
	{6090, "XXX/WEB-DL", 6000},
	{7000, "Books", 0}, {7010, "Books/Mags", 7000}, {7020, "Books/EBook", 7000},
	{7030, "Books/Comics", 7000}, {7040, "Books/Technical", 7000}, {7050, "Books/Other", 7000},
	{7060, "Books/Foreign", 7000},
	{8000, "Other", 0}, {8010, "Other/Misc", 8000}, {8020, "Other/Hashed", 8000},
	{0, "Other", 0}, {20, "Other/Hashed", 0}, {10, "Other/Misc", 0},
}

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
