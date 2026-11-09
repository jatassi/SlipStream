// Package mediatest is a Prober that runs no ffprobe, for in-process tests
// and the simulation tier (ADR 0007's fake runner).
package mediatest

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/media"
)

// Prober answers Of[key] for a path that ends with "/"+key, the longest
// key first, so a test keys by a root-relative path ("Heat (1995)/Heat.mkv")
// or by a base name. Fail[key] answers an *media.UnreadableError with that
// line. Any other path fails.
type Prober struct {
	Of   map[string]media.Record
	Fail map[string]string
}

func (p Prober) Probe(_ context.Context, path string) (media.Record, error) {
	path = "/" + filepath.ToSlash(path)
	for i := range len(path) {
		if path[i] != '/' {
			continue
		}
		key := path[i+1:]
		if line, failed := p.Fail[key]; failed {
			return media.Record{}, &media.UnreadableError{Line: line}
		}
		if r, ok := p.Of[key]; ok {
			return r, nil
		}
	}
	return media.Record{}, fmt.Errorf("mediatest: no probe for %s", path[1:])
}
