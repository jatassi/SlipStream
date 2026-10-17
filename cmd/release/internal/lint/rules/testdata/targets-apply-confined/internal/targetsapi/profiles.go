package targetsapi

import (
	rules "example.test/slip/internal/targets"
	"example.test/slip/internal/files"
)

type handlers struct{ files files.Shelf }

func (h handlers) save(c rules.Change) error {
	h.files.Apply(c)
	return rules.Apply(c)
}
