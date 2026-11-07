package node

import (
	"example.com/app/internal/converting"
	"example.com/app/internal/downloading"
	"example.com/app/internal/fetching"
	"example.com/app/internal/moving"
	"example.com/app/internal/relaying"
	"example.com/app/internal/role"
	"example.com/app/internal/role/roletest"
	"example.com/app/internal/searching"
	"example.com/app/internal/sharing"
	"example.com/app/internal/transfers"
)

type Node struct{}

func servedRoles() []role.Served {
	return []role.Served{searching.Def.Serve(nil, nil), converting.Def.Serve(nil, nil), fetching.Def.Serve(nil, nil), fetching.Named.Serve(nil, nil), moving.Def.Serve(nil, nil)}
}

func (n *Node) startController() {
	served := func() []role.Served {
		return []role.Served{transfers.Def.Serve(nil, nil), relaying.Def.Serve(nil, nil), sharing.Def.Serve(nil, nil)}
	}
	_ = served
	_ = relaying.Def.Hold(nil)
}

func (n *Node) heldRoles() []role.Held {
	return []role.Held{searching.Def.Hold(nil), downloading.Def.Hold(nil), fetching.Def.Hold(nil), fetching.Named.Hold(nil), roletest.Def.Hold(nil), transfers.Def.Hold(nil), moving.Def.Hold(nil), sharing.Def.Hold(nil)}
}
