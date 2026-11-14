package simtest

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jatassi/slipstream/fakes/world/sim"
	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/node"
)

// ProbeFault is how one probe of a file goes wrong.
type ProbeFault uint8

const (
	// ProbeTimeout holds the probe until its deadline, as an ffprobe stuck
	// on a file does, and fails as FFprobe does then.
	ProbeTimeout ProbeFault = iota + 1
	// ProbeUnreadable fails as ffprobe does on bytes that are no media
	// file.
	ProbeUnreadable
)

// Probes plays faults on the probes of files, in order, one per probe:
// once a file's faults are played, its probes answer as the walk's do.
// A file is keyed as the walk keys it, <torrent name>/<path>, and matches
// any path that ends with "/"+key. One Probes serves every Node and keeps
// its state across their restarts.
type Probes struct {
	mu     sync.Mutex
	faults map[string][]ProbeFault
	probed map[string]int
}

func NewProbes() *Probes {
	return &Probes{faults: map[string][]ProbeFault{}, probed: map[string]int{}}
}

// Play queues faults on key's next probes.
func (p *Probes) Play(key string, faults ...ProbeFault) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.faults[key] = append(p.faults[key], faults...)
}

// Probed is how many probes the Nodes started of key, a key Play named.
func (p *Probes) Probed(key string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.probed[key]
}

// Option puts p in front of each Node's prober.
func (p *Probes) Option() sim.Option {
	return sim.Edit(func(_ string, e *node.Env) { e.Media = faulted{p, e.Media} })
}

// next counts a probe of path and answers its fault, 0 for none.
func (p *Probes) next(path string) ProbeFault {
	path = "/" + filepath.ToSlash(path)
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, faults := range p.faults {
		if !strings.HasSuffix(path, "/"+key) {
			continue
		}
		p.probed[key]++
		if len(faults) == 0 {
			return 0
		}
		p.faults[key] = faults[1:]
		return faults[0]
	}
	return 0
}

type faulted struct {
	p    *Probes
	base media.Prober
}

func (f faulted) Probe(ctx context.Context, path string) (media.Record, error) {
	switch f.p.next(path) {
	case ProbeTimeout:
		<-ctx.Done()
		return media.Record{}, fmt.Errorf("run ffprobe: %w", ctx.Err())
	case ProbeUnreadable:
		return media.Record{}, &media.UnreadableError{Line: "Invalid data found when processing input"}
	}
	return f.base.Probe(ctx, path)
}
