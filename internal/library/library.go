// Package library composes the Library's reads and actions over the
// owners of Works, Targets, roots and files, each in one snapshot or one
// Write across them.
package library

import (
	"errors"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// ErrNotBuilt is a method whose work hasn't shipped yet.
var ErrNotBuilt = errors.New("library: not built yet")

type Config struct {
	Library *storage.DB[storage.Library]
	Works   *works.Works
	Targets *targets.Targets
	Roots   *roots.Roots
	Files   *files.Files
}

type Library struct {
	cfg Config
}

func New(cfg Config) *Library {
	return &Library{cfg: cfg}
}
