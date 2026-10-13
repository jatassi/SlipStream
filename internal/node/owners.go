package node

import (
	"context"
	"errors"

	"github.com/jatassi/slipstream/internal/link"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// controllerSide opens the nodes owner's side of the Controller half. Until
// the nodes owner exists it is a roster of this Node alone, holding no Role
// and admitting no peer.
func (n *Node) controllerSide(context.Context, *storage.DB[storage.Library]) (controllerSide, error) {
	return selfOnly{id: role.NodeID{UUID: storage.NewUUID()}, key: n.key.Public()}, nil
}

// workerSide reads the pairing owner's side of the Worker half from
// pairing.json.
func (n *Node) workerSide() (workerSide, error) {
	return nil, errors.New("this build has no pairing owner to read pairing.json")
}

// selfOnly is a Controller's roster before the nodes owner: its own row.
type selfOnly struct {
	id  role.NodeID
	key link.Key
}

func (s selfOnly) Self() role.NodeID { return s.id }

func (s selfOnly) NodeOf(key link.Key) (role.NodeID, bool) { return s.id, key == s.key }

func (s selfOnly) Key(node role.NodeID) (link.Key, bool) { return s.key, node == s.id }

func (selfOnly) Holders(role.Name) []role.NodeID { return nil }

func (selfOnly) Assigned(context.Context, storage.Reader[storage.Library], role.NodeID) ([]role.Name, error) {
	return nil, nil
}

func (selfOnly) Changed(role.NodeID) {}

func (selfOnly) Admit(link.Key) link.Admission { return link.Refused{} }

func (selfOnly) Pair(context.Context, link.Key, *linkv1.PairRequest) (*linkv1.PairResponse, error) {
	return notAController(), nil
}
