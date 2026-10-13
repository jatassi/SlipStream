package node

import (
	"example.com/app/internal/converting"
	"example.com/app/internal/downloading"
	"example.com/app/internal/fetching"
	"example.com/app/internal/role"
	"example.com/app/internal/role/roletest"
	"example.com/app/internal/searching"
)

func servedRoles() []role.Served {
	return []role.Served{searching.Def.Serve(nil, nil), converting.Def.Serve(nil, nil), fetching.Def.Serve(nil, nil), fetching.Named.Serve(nil, nil)}
}

func heldRoles() []role.Held {
	return []role.Held{searching.Def.Hold(nil), downloading.Def.Hold(nil), fetching.Def.Hold(nil), fetching.Named.Hold(nil), roletest.Def.Hold(nil)}
}
