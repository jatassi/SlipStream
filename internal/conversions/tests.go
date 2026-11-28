package conversions

import (
	"context"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/conversions/internal/db"
	"github.com/jatassi/slipstream/internal/conversions/internal/plan"
	"github.com/jatassi/slipstream/internal/converting/pipeline"
	"github.com/jatassi/slipstream/internal/encode"
	cv1 "github.com/jatassi/slipstream/internal/pb/slipstream/converting/v1"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
)

// test is one Node's test run of one key.
type test struct {
	node role.NodeID
	key  encode.Key
}

// inFlight is the test runs asked of each Node and not yet answered.
func (r *reconciler) inFlight() map[role.NodeID]int {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[role.NodeID]int, len(r.testing))
	for n, ks := range r.testing {
		out[n] = len(ks)
	}
	return out
}

// take marks node's test run of k in flight, false when it already is or
// its last ask failed less than a pass before now.
func (r *reconciler) take(node role.NodeID, k encode.Key, now time.Time, pass time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.testing[node][k] || now.Before(r.unasked[test{node, k}].Add(pass)) {
		return false
	}
	delete(r.unasked, test{node, k})
	if r.testing[node] == nil {
		r.testing[node] = map[encode.Key]bool{}
	}
	r.testing[node][k] = true
	return true
}

func (r *reconciler) done(node role.NodeID, k encode.Key) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.testing[node], k)
	if len(r.testing[node]) == 0 {
		delete(r.testing, node)
	}
}

// ask asks each test run outside any Write. Its answer is stored under
// the Node's signature when it comes, and wakes the reconciler.
func (c *Conversions) ask(ctx context.Context, tests []plan.Test) {
	for _, t := range tests {
		if !c.rec.take(t.Node, t.Key, c.cfg.Clock(), c.cfg.Timings.Pass) {
			continue
		}
		c.rec.tests.Go(func() {
			defer c.rec.done(t.Node, t.Key)
			if err := c.test(ctx, t.Node, t.Key); err != nil {
				c.cfg.Log.Warn("run a test run", "node", t.Node, "shape", t.Key.Shape.String(), "class", t.Key.Class, "err", err)
				c.rec.mu.Lock()
				c.rec.unasked[test{t.Node, t.Key}] = c.cfg.Clock()
				c.rec.mu.Unlock()
				return
			}
			c.cfg.Poke.Wake()
		})
	}
}

func (c *Conversions) test(ctx context.Context, node role.NodeID, k encode.Key) error {
	res, err := c.cfg.Converting(node).Test(ctx, &cv1.TestRequest{
		//nolint:gosec // a depth is 8 or 10
		Shape: &cv1.Shape{Codec: string(k.Shape.Codec), Depth: uint32(k.Shape.Depth), Range: string(k.Shape.Range), KeepDv: k.Shape.KeepDV, Curve: string(k.Shape.Curve)},
		Class: string(k.Class),
	})
	if err != nil {
		return err
	}
	p, err := proofOf(res, osOf(res))
	if err != nil {
		return err
	}
	f, _ := c.roster().facts(node)
	current, _ := signatureOf(f.GetSignature())
	if p.Signature != current {
		return fmt.Errorf("the test run ran under signature %s, and the Node reports %s", p.Signature, current)
	}
	return c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return c.prove(ctx, tx, node, current, k, p)
	})
}

// osOf is an OS whose pipeline rows list the answer's pipeline, so
// proofOf refuses a pipeline no OS has.
func osOf(res *cv1.TestResponse) string {
	p := encode.Pipeline{Path: encode.Path(res.GetPipeline().GetPath()), API: encode.API(res.GetPipeline().GetApi())}
	for _, r := range pipeline.Rows {
		if r.Pipeline == p && len(r.OS) > 0 {
			return r.OS[0]
		}
	}
	return ""
}

// Rerun deletes n's proofs, so its test runs run again.
func (c *Conversions) Rerun(ctx context.Context, n role.NodeID) error {
	if err := c.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		return db.New(tx).DeleteProofsOf(ctx, n)
	}); err != nil {
		return fmt.Errorf("delete the Node's proofs: %w", err)
	}
	c.cfg.Control().Touch(n)
	c.cfg.Poke.Wake()
	return nil
}
