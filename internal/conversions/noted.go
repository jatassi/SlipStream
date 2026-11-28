package conversions

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/converting"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// Noted is told each facts change: a holder's Converting facts reach the
// reconciler, and a media-server-link change Touches every Converting
// holder and wakes it.
func (c *Conversions) Noted(node role.NodeID, scope string) {
	if role.Name(scope) == role.MediaServerLink {
		c.TouchHolders()
		c.cfg.Poke.Wake()
	}
	if role.Name(scope) != role.Converting {
		return
	}
	c.rec.mu.Lock()
	c.rec.moved[node] = true
	c.rec.mu.Unlock()
	c.rec.note()
}

// TouchHolders Touches every Converting holder's intent, for
// mediaservers.Config.Touched.
func (c *Conversions) TouchHolders() {
	c.cfg.Control().Touch(c.cfg.Board().Who(role.Converting).Serving()...)
}

// followFacts reads the facts of each Node that reported since it last
// ran: a new signature replaces its proofs, and its runs' progress goes
// live.
func (c *Conversions) followFacts(ctx context.Context) {
	c.rec.mu.Lock()
	moved := c.rec.moved
	c.rec.moved = map[role.NodeID]bool{}
	c.rec.mu.Unlock()
	for node := range moved {
		f, ok := c.cfg.Control().Facts(node)
		if !ok {
			continue
		}
		replaced, err := c.observe(ctx, node, f)
		if err != nil {
			c.cfg.Log.Warn("replace a Node's proofs", "node", node, "err", err)
		}
		if replaced {
			c.cfg.Poke.Wake()
		}
	}
	c.sample(ctx, moved)
}

// observe keeps node's signature, and replaces its proofs when it
// differs from the last one, true when it did.
func (c *Conversions) observe(ctx context.Context, node role.NodeID, f *cv1.Facts) (bool, error) {
	sig, ok := signatureOf(f.GetSignature())
	c.rec.mu.Lock()
	same := c.rec.signed[node] == sig
	c.rec.signed[node] = sig
	c.rec.mu.Unlock()
	if !ok || same {
		return false, nil
	}
	return true, c.replaceProofs(ctx, node, sig)
}

// replaceProofs deletes every proof of node under another signature, in
// one Write, so the keys its Conversions need read Untested.
func (c *Conversions) replaceProofs(ctx context.Context, node role.NodeID, sig converting.Signature) error {
	err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).DeleteProofsBesides(ctx, db.DeleteProofsBesidesParams{NodeID: node, Signature: string(sig)})
	})
	if err != nil {
		return fmt.Errorf("delete the Node's older proofs: %w", err)
	}
	return nil
}

// signature is node's last reported signature, empty before any.
func (c *Conversions) signature(node role.NodeID) converting.Signature {
	c.rec.mu.Lock()
	defer c.rec.mu.Unlock()
	return c.rec.signed[node]
}
