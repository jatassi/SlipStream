package reconciling

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Refusal is the detail of the reconciling problem: what Reconciling
// waits on as the refusal is written.
func (s *Service) Refusal(ctx context.Context) (string, error) {
	_, w, names, err := s.waits(ctx)
	if err != nil {
		return "", err
	}
	return refusalWords(w, names), nil
}

// Banner is what the shell's line reads while Reconciling: whether the
// old Controller's folders still need finding, each Node it waits on by
// name, and how many roots.
type Banner struct {
	Rehome bool
	Nodes  []string
	Roots  int
}

// Banner is the shell's line, nil outside a restore's Reconciling.
func (s *Service) Banner(ctx context.Context) (*Banner, error) {
	on, w, names, err := s.waits(ctx)
	if err != nil || !on {
		return nil, err
	}
	b := &Banner{Rehome: w.Rehome, Roots: len(w.Roots), Nodes: make([]string, len(w.Nodes))}
	for i, n := range w.Nodes {
		b.Nodes[i] = names[n.Node]
	}
	return b, nil
}

func (s *Service) waits(ctx context.Context) (on bool, w Waits, names map[nodes.ID]string, err error) {
	names = map[nodes.ID]string{}
	err = s.cfg.Library.Read(ctx, func(ctx context.Context, r storage.Reader[file]) error {
		phase, err := State(ctx, r)
		if on = phase == Reconciling; err != nil || !on {
			return err
		}
		if w, err = Waiting(ctx, r); err != nil {
			return err
		}
		named, err := nodes.Names(ctx, r)
		for _, n := range named {
			names[n.ID] = n.Name
		}
		return err
	})
	return on, w, names, err
}

func refusalWords(w Waits, names map[nodes.ID]string) string {
	const lead = "SlipStream is matching the restored library with what your Nodes hold. "
	var on []string
	if w.Rehome {
		on = append(on, "the old Controller's folders")
	}
	for _, n := range w.Nodes {
		on = append(on, names[n.Node])
	}
	switch len(w.Roots) {
	case 0:
	case 1:
		on = append(on, "1 library root")
	default:
		on = append(on, setting.Grouped(len(w.Roots))+" library roots")
	}
	if len(on) == 0 {
		return lead + "Try again once it finishes."
	}
	return fmt.Sprintf("%sIt's waiting for %s. Try again once it finishes.", lead, works.JoinWords(on))
}

// GrantWords is the Reconciling page's line for a Node adoption gave the
// Downloading Role.
func GrantWords(name string, held int) string {
	if held == 1 {
		return name + " now holds Downloading for the 1 torrent it reported."
	}
	return fmt.Sprintf("%s now holds Downloading for the %s torrents it reported.", name, setting.Grouped(held))
}
