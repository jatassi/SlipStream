package backups

import "crypto/rand"

// Proven is nodes' Fencer: whether proof, carried by a Worker this
// Controller pins, shows that a restore of one of its Backups on the
// Controller holding replacement replaced old, under the fence secret's
// current generation.
func (b *Backups) Proven(old, replacement, proof []byte) bool {
	v, err := b.cfg.Engine.Vault(b.cfg.Keys)
	if err != nil {
		b.cfg.Log.Error("read backup.key to check a proof of a restore", "err", err)
		return false
	}
	return v.Proven(old, replacement, proof)
}

// RegenerateFence gives the fence secret a new generation, so no proof
// minted before it fences this Controller.
func (b *Backups) RegenerateFence() error {
	v, err := b.cfg.Engine.Vault(b.cfg.Keys)
	if err != nil {
		return err
	}
	return v.RegenerateFence(rand.Reader)
}
