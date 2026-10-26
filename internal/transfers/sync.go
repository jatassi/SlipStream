package transfers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jatassi/slipstream/internal/nodes"
	tv1 "github.com/jatassi/slipstream/internal/pb/slipstream/transfer/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/transfers/internal/ledger"
)

// Derive is node's intent: judge over a snapshot of rd, projected to
// node.
func (t *Transfers) Derive(ctx context.Context, rd storage.Reader[storage.Library], node role.NodeID) (*tv1.Intent, error) {
	j, err := t.judgeIn(ctx, rd, t.cfg.Board())
	if err != nil {
		return nil, err
	}
	return j.intents.of(node), nil
}

// Apply applies one of node's Results in tx. A FileVerified counts only
// from the destination's Node at its current place, for a file of an
// unfailed Transfer not yet verified there, and fires Verified for the
// group it completes. A Failed counts only from the destination's Node for
// an unfailed Transfer, and fires Failed. Anything else applies as
// nothing, so one stale Result never holds back those behind it.
func (t *Transfers) Apply(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, r *tv1.Result) error {
	switch k := r.GetResult().(type) {
	case *tv1.Result_Verified:
		return t.verified(ctx, tx, node, k.Verified)
	case *tv1.Result_Failed:
		return t.failed(ctx, tx, node, k.Failed)
	}
	return nil
}

// current is the unfailed Transfer named by id when node receives it,
// false for anything else.
func current(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, id []byte) (Transfer, roots.Root, bool, error) {
	at, ok := idOf(id)
	if !ok {
		return Transfer{}, roots.Root{}, false, nil
	}
	tr, err := load(ctx, tx, at)
	if errors.Is(err, storage.ErrNoRows) {
		return Transfer{}, roots.Root{}, false, nil
	}
	if err != nil {
		return Transfer{}, roots.Root{}, false, err
	}
	root, err := roots.Get(ctx, tx, rootOf(tr.Destination))
	if err != nil {
		return Transfer{}, roots.Root{}, false, err
	}
	return tr, root, tr.Failure == nil && root.Node == node, nil
}

func (t *Transfers) verified(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, v *tv1.FileVerified) error {
	tr, root, ok, err := current(ctx, tx, node, v.GetTransferId())
	if err != nil || !ok || v.GetPlace() != root.Place {
		return err
	}
	at := slices.IndexFunc(tr.Files, func(f File) bool { return f.Position == int(v.GetPosition()) })
	if at < 0 || !missing(tr.Files[at]) {
		return nil
	}
	if err := ledger.New(tx).SetVerified(ctx, ledger.SetVerifiedParams{VerifiedPlace: &root.Place, TransferID: tr.ID, Position: int64(tr.Files[at].Position)}); err != nil {
		return fmt.Errorf("record file %d of Transfer %s verified: %w", tr.Files[at].Position, tr.ID, err)
	}
	tr.Files[at].Verified = true
	group := tr.Files[at].Group
	if slices.ContainsFunc(tr.Files, func(f File) bool { return f.Group == group && missing(f) }) {
		return nil
	}
	if err := t.purposes[tr.Purpose].Verified(ctx, tx, tr, group); err != nil {
		return err
	}
	t.note(node, tr)
	return nil
}

var codes = map[tv1.Failed_Code]Code{
	tv1.Failed_CODE_SOURCE_GONE:   SourceGone,
	tv1.Failed_CODE_SOURCE_FAILED: SourceFailed,
	tv1.Failed_CODE_DAMAGED:       Damaged,
}

func (t *Transfers) failed(ctx context.Context, tx *storage.Tx[storage.Library], node role.NodeID, f *tv1.Failed) error {
	tr, root, ok, err := current(ctx, tx, node, f.GetTransferId())
	code, known := codes[f.GetCode()]
	if err != nil || !ok || !known {
		return err
	}
	position := -1
	if p := int(f.GetPosition()); p < len(tr.Files) {
		position = p
	}
	if err := t.fail(ctx, tx, tr, root.Node, code, position, f.GetDetail()); err != nil {
		return err
	}
	t.note(node, tr)
	return nil
}

// fail stores t's failure and fires its purpose's Failed hook, in tx.
func (t *Transfers) fail(ctx context.Context, tx *storage.Tx[storage.Library], tr Transfer, dest role.NodeID, code Code, position int, detail string) error {
	p := ledger.SetFailureParams{FailureCode: new(string(code)), FailureDetail: detail, ID: tr.ID}
	if position >= 0 {
		p.FailurePosition = new(int64(position))
	}
	if err := ledger.New(tx).SetFailure(ctx, p); err != nil {
		return fmt.Errorf("record Transfer %s failed: %w", tr.ID, err)
	}
	peers, err := nodes.PeersIn(ctx, tx)
	if err != nil {
		return err
	}
	names := make(map[role.NodeID]string, len(peers))
	for id, p := range peers {
		names[id] = p.Name
	}
	f := Failure{Code: code, Position: position, Detail: detail, Sentence: sentence(tr, dest, code, position, detail, names)}
	tr.Failure = &f
	return t.purposes[tr.Purpose].Failed(ctx, tx, tr, f)
}

// note keeps tr's job for its purpose's After, once node's batch commits.
func (t *Transfers) note(node role.NodeID, tr Transfer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.fired[node] == nil {
		t.fired[node] = map[PurposeKind][]string{}
	}
	t.fired[node][tr.Purpose] = append(t.fired[node][tr.Purpose], tr.Job)
}

// Applied is the Applied hook: once a batch of node's Results commits, it
// runs After for each purpose whose hooks fired and wakes the loop.
func (t *Transfers) Applied(node role.NodeID, scope string) {
	if scope != string(role.Transfers) {
		return
	}
	t.mu.Lock()
	fired := t.fired[node]
	delete(t.fired, node)
	t.mu.Unlock()
	t.after(fired)
	t.Touch()
}

func (t *Transfers) after(fired map[PurposeKind][]string) {
	for kind, jobs := range fired {
		t.purposes[kind].After(jobs)
	}
}

// Noted is the Facts hook: it stamps when node first reported each disk
// full and wakes the loop. It never blocks.
func (t *Transfers) Noted(node role.NodeID, scope string) {
	if scope != string(role.Transfers) {
		return
	}
	facts, _ := t.cfg.Facts(node)
	now := time.Now()
	t.mu.Lock()
	for _, r := range facts.GetReceiving() {
		id, ok := idOf(r.GetTransferId())
		if !ok {
			continue
		}
		_, stamped := t.fullAt[id]
		switch full := r.GetWait().GetKind() == tv1.Wait_KIND_DISK_FULL; {
		case full && !stamped:
			t.fullAt[id] = now
		case !full && stamped:
			delete(t.fullAt, id)
		}
	}
	t.mu.Unlock()
	t.Touch()
}

// idOf is the Transfer id b holds, false when it holds none.
func idOf(b []byte) (ID, bool) {
	u, err := storage.UUIDFromBytes(b)
	return ID{UUID: u}, err == nil
}
