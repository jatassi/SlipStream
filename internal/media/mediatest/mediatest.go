// Package mediatest is a Prober that runs no ffprobe, for in-process tests
// and the simulation tier (ADR 0007's fake runner).
package mediatest

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

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
	keys := slices.Concat(slices.Collect(maps.Keys(p.Of)), slices.Collect(maps.Keys(p.Fail)))
	slices.SortFunc(keys, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	for _, key := range keys {
		if !strings.HasSuffix(path, "/"+key) {
			continue
		}
		if line, failed := p.Fail[key]; failed {
			return media.Record{}, &media.UnreadableError{Line: line}
		}
		return p.Of[key], nil
	}
	return media.Record{}, fmt.Errorf("mediatest: no probe for %s", path[1:])
}
