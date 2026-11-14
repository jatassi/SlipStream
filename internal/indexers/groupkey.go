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
