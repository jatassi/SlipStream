package indexers

import (
	"encoding/hex"

	"github.com/jatassi/slipstream/internal/works"
)

// GroupKey is a search group's key: the Items of a Work one query
// searches. It is opaque to core, built only by the helpers below, and
// stable while its Content Type's numbering stands. Migrations build the
// same text in SQL, as 'l:' || lower(hex(level_id)).
type GroupKey string

// LevelGroup is the group of a level's Items.
func LevelGroup(id works.LevelID) GroupKey { return GroupKey("l:" + hex.EncodeToString(id.Bytes())) }

// ItemGroup is the group of one Item alone.
func ItemGroup(id works.ItemID) GroupKey { return GroupKey("i:" + hex.EncodeToString(id.Bytes())) }

// NamedGroup is a group its Content Type names, such as a level under
// another numbering than the stored one.
func NamedGroup(name string) GroupKey { return GroupKey("n:" + name) }

// Group is the Items of a Work one query searches.
type Group struct {
	Key GroupKey
	// Label is a noun phrase for mid-sentence use, such as "season 3",
	// "S00E04" or a title.
	Label string
	Items []works.ItemID
}

// Singles is each of items in a group of its own, labeled label, for a
// type that searches every Item alone.
func Singles(items []works.ItemID, label string) []Group {
	groups := make([]Group, len(items))
	for i, id := range items {
		groups[i] = Group{Key: ItemGroup(id), Label: label, Items: []works.ItemID{id}}
	}
	return groups
}
