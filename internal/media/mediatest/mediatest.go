// Package mediatest is a Prober that runs no ffprobe, for in-process tests
// and the simulation tier (ADR 0007's fake runner).
package mediatest

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/jatassi/slipstream/internal/media"
)

// Prober answers the probe Of holds for a file's name, and fails for any
// other file.
type Prober struct {
	Of map[string]media.Probe
}

func (p Prober) Probe(_ context.Context, path string) (media.Probe, error) {
	if probe, ok := p.Of[filepath.Base(path)]; ok {
		return probe, nil
	}
	return media.Probe{}, fmt.Errorf("mediatest: no probe for %s", filepath.Base(path))
}
