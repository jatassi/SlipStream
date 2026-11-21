package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jatassi/slipstream/internal/link"
	"github.com/jatassi/slipstream/internal/nodes/internal/db"
	linkv1 "github.com/jatassi/slipstream/internal/pb/slipstream/link/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Fencer checks a proof that a restore replaced this Controller: one the
// restored Controller, holding key replacement, minted for key old from
// the fence secret this Controller holds.
type Fencer interface {
	Proven(old, replacement, proof []byte) bool
}

// FenceState is how far this Controller is fenced: a pinned Worker said a
// restore replaced it without a proof, or with one, or an Admin stopped or
// retired it.
type FenceState string

const (
	Unproven FenceState = "unproven"
	Proven   FenceState = "proven"
	Stopped  FenceState = "stopped"
	Retired  FenceState = "retired"
)

// Fence is the nodes_fence row: the state, the key that replaced this
// Controller and its addresses as the Worker gave them, the Worker that
// carried the claim, and when.
type Fence struct {
	State     FenceState
	By        link.Key
	ByNode    ID
	Addresses link.Addresses
	At        time.Time
}

// after is the fence a pinned Worker's claim leaves on cur: the first
// claim is recorded as it is, and a proven claim fences a Controller that
// holds only an unproven one. Nothing overwrites a proven fence, a Stop or
// a Retire.
func after(cur Fence, fenced bool, claim Fence) (Fence, bool) {
	switch {
	case !fenced:
		return claim, true
	case cur.State == Unproven && claim.State == Proven:
		return claim, true
	}
	return cur, false
}

type toldClaim struct {
	from link.Key
	s    link.Superseded
}

// Told is link.Config.Told on a Controller: a pinned Worker said a restore
// replaced this Controller. Run checks the proof and records the fence.
// A claim that arrives while Run is behind is dropped, since a Worker says
// it again at its next dial. It never blocks.
func (s *Service) Told(from link.Key, sup link.Superseded) {
	select {
	case s.tells <- toldClaim{from, sup}:
	default:
	}
}

// fence records c's claim as the fence it leaves, checking its proof
// against the Fencer, and tells Admins when the row changed. A claim from
// a key the roster doesn't hold changes nothing.
func (s *Service) fence(ctx context.Context, c toldClaim) error {
	node, ok := s.NodeOf(c.from)
	if !ok {
		return nil
	}
	claim := Fence{State: Unproven, By: c.s.By, ByNode: node, Addresses: c.s.Addresses, At: time.Now()}
	if s.cfg.Fence.Proven(s.cfg.Key[:], c.s.By[:], c.s.Proof) {
		claim.State = Proven
	}
	changed := false
	err := s.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[file]) error {
		cur, fenced, err := FenceIn(ctx, tx)
		if err != nil {
			return err
		}
		next, ok := after(cur, fenced, claim)
		if !ok {
			return nil
		}
		changed = true
		return putFence(ctx, db.New(tx), next)
	})
	if err != nil {
		return fmt.Errorf("record that a restore replaced this Controller: %w", err)
	}
	if !changed {
		return nil
	}
	name := s.Name(node)
	if claim.State == Proven {
		s.cfg.Log.Warn(name+" sent proof that a Backup of this library was restored on another Controller", "by", c.s.By.Fingerprint())
	} else {
		s.cfg.Log.Warn(name+" is paired with another Controller and sent no proof of a restore", "by", c.s.By.Fingerprint())
	}
	s.changed(node)
	return nil
}

func putFence(ctx context.Context, q *db.Queries, f Fence) error {
	addrs, err := json.Marshal(f.Addresses.All())
	if err != nil {
		return fmt.Errorf("encode the fence's addresses: %w", err)
	}
	var by []byte
	if f.By != (link.Key{}) {
		by = f.By[:]
	}
	return q.PutFence(ctx, db.PutFenceParams{State: string(f.State), ByKey: by, ByNode: f.ByNode, Addresses: string(addrs), FencedAt: storage.FromTime(f.At)})
}

// FenceIn is this Controller's fence in r's snapshot, false while it has
// none.
func FenceIn(ctx context.Context, r storage.Reader[file]) (Fence, bool, error) {
	row, err := db.New(r).GetFence(ctx)
	if errors.Is(err, storage.ErrNoRows) {
		return Fence{}, false, nil
	}
	if err != nil {
		return Fence{}, false, fmt.Errorf("read the fence: %w", err)
	}
	f := Fence{State: FenceState(row.State), ByNode: row.ByNode, At: row.FencedAt.Time()}
	if row.ByKey != nil {
		if f.By, err = link.KeyFromBytes(row.ByKey); err != nil {
			return Fence{}, false, fmt.Errorf("read the fence: %w", err)
		}
	}
	var aps []netip.AddrPort
	if err := json.Unmarshal([]byte(row.Addresses), &aps); err != nil {
		return Fence{}, false, fmt.Errorf("read the fence's addresses: %w", err)
	}
	if len(aps) > 0 {
		if f.Addresses, err = link.ParseAddresses(aps); err != nil {
			return Fence{}, false, fmt.Errorf("read the fence's addresses: %w", err)
		}
	}
	return f, true, nil
}

// ClearFence deletes this Controller's fence.
func ClearFence(ctx context.Context, tx *storage.Tx[file]) error {
	return db.New(tx).DeleteFence(ctx)
}

// Rekey gives the Controller row, which holds key old, this Node's key
// self, and records old as superseded with proof, so every Worker learns
// it. A live row that already holds self, a Node paired after the Backup
// on this machine, is removed first, which writes off its work; its id is
// returned, zero when there was none.
func Rekey(ctx context.Context, tx *storage.Tx[file], self, old link.Key, proof []byte) (ID, error) {
	q := db.New(tx)
	controller, err := q.GetNodeByKey(ctx, old[:])
	if err != nil {
		return ID{}, fmt.Errorf("find the Controller row by its old key: %w", err)
	}
	var removed ID
	switch row, err := q.GetNodeByKey(ctx, self[:]); {
	case err == nil:
		if err := entomb(ctx, q, row.ID); err != nil {
			return ID{}, err
		}
		removed = row.ID
	case !errors.Is(err, storage.ErrNoRows):
		return ID{}, err
	}
	if err := q.SetNodeKey(ctx, db.SetNodeKeyParams{PublicKey: self[:], ID: controller.ID}); err != nil {
		return ID{}, err
	}
	return removed, q.InsertSuperseded(ctx, db.InsertSupersededParams{PublicKey: old[:], NodeID: controller.ID, Proof: proof, SupersededAt: storage.Now()})
}

// RemoveIn removes node in tx, as an Admin's removal does: its row becomes
// a tombstone, so its work counts as written off.
func RemoveIn(ctx context.Context, tx *storage.Tx[file], node ID) error {
	return entomb(ctx, db.New(tx), node)
}

// RemoveKeyIn removes the live Node holding key in tx, if any.
func RemoveKeyIn(ctx context.Context, tx *storage.Tx[file], key link.Key) error {
	q := db.New(tx)
	row, err := q.GetNodeByKey(ctx, key[:])
	if errors.Is(err, storage.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return entomb(ctx, q, row.ID)
}

// DeleteSelfAddresses deletes the addresses recorded for the Node holding
// key self, which the old Controller's machine gave.
func DeleteSelfAddresses(ctx context.Context, tx *storage.Tx[file], self link.Key) error {
	q := db.New(tx)
	row, err := q.GetNodeByKey(ctx, self[:])
	if err != nil {
		return fmt.Errorf("find this Node's row: %w", err)
	}
	if err := q.DeleteAddresses(ctx, row.ID); err != nil {
		return err
	}
	return q.DeleteDials(ctx, row.ID)
}

// AssignIn gives node r in tx with no probe or reach test: the caller has
// the Node's own report that it already runs r's work.
func AssignIn(ctx context.Context, tx *storage.Tx[file], node ID, r role.Name) error {
	return db.New(tx).InsertRole(ctx, db.InsertRoleParams{NodeID: node, Role: r, AssignedAt: storage.Now()})
}

// unpaired is each old Controller key this Controller replaced that no
// Node holds or held, with its proof.
func unpaired(ctx context.Context, q *db.Queries) ([]link.SupersededProof, error) {
	rows, err := q.ListUnpairedSuperseded(ctx)
	if err != nil {
		return nil, fmt.Errorf("list the superseded Controller keys: %w", err)
	}
	out := make([]link.SupersededProof, len(rows))
	for i, row := range rows {
		key, err := link.KeyFromBytes(row.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("superseded key: %w", err)
		}
		out[i] = link.SupersededProof{Key: key, Proof: row.Proof}
	}
	return out, nil
}

func supersededProto(ps []link.SupersededProof) []*linkv1.SupersededProof {
	out := make([]*linkv1.SupersededProof, len(ps))
	for i, p := range ps {
		out[i] = &linkv1.SupersededProof{OldKey: p.Key[:], Proof: p.Proof}
	}
	return out
}
