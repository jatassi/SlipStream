package roots

import (
	"context"
	"fmt"

	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Words is each root's words, the one way copy names a root: "Movies
// library root on Media server", or "Movies library root at /mnt/4k on
// Media server" when its Node holds another root of its type.
func (r *Roots) Words(ctx context.Context) (map[ID]string, error) {
	names, err := r.cfg.Names(ctx)
	if err != nil {
		return nil, err
	}
	var all []Root
	err = r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		all, err = All(ctx, rd)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read the library roots: %w", err)
	}
	return r.wordsOf(all, names), nil
}

func (r *Roots) wordsOf(all []Root, names map[role.NodeID]nodes.Named) map[ID]string {
	type typeOn struct {
		t    works.TypeID
		node role.NodeID
	}
	twins := map[typeOn]int{}
	for _, root := range all {
		twins[typeOn{root.ContentType, root.Node}]++
	}
	out := make(map[ID]string, len(all))
	for _, root := range all {
		node := names[root.Node].Name
		d, _ := r.cfg.Describe(root.ContentType)
		out[root.ID] = d.Work.Many + " library root on " + node
		if twins[typeOn{root.ContentType, root.Node}] > 1 {
			out[root.ID] = d.Work.Many + " library root at " + root.Path + " on " + node
		}
	}
	return out
}

// trouble is what a root says of each State that is wrong: what the
// root's row says, then what to do.
var trouble = map[State]struct{ sentence, fix string }{
	Missing:      {"The folder isn't there.", "Restore it, or edit the root to where it is now."},
	Unmounted:    {"The folder is there but looks unmounted.", "Plug in or mount the disk."},
	Foreign:      {"The folder holds another library root's marker.", "Edit the root to a folder of its own."},
	Unwritable:   {"SlipStream can't write to the folder.", "Let SlipStream write to it."},
	Unresponsive: {"The folder isn't responding.", "Check the disk or share it is on."},
	WrittenOff:   {"Its Node was written off, so its files count as missing.", "Re-home the root to a Node that holds Storage."},
}

// Trouble is what a root in s says and what to do, the sentences its
// Storage item uses. ok is false for a State that is not wrong: Ready,
// and Checking or Offline, which wait on the Node.
func Trouble(s State) (sentence, fix string, ok bool) {
	t, ok := trouble[s]
	return t.sentence, t.fix, ok
}

// Ready is every root that reads Ready at its place now.
func (r *Roots) Ready(ctx context.Context) ([]Root, error) {
	all, states, err := r.states(ctx)
	if err != nil {
		return nil, err
	}
	var out []Root
	for i, root := range all {
		if states[i].state == Ready {
			out = append(out, root)
		}
	}
	return out, nil
}

// states is every root with its State, from one snapshot.
func (r *Roots) states(ctx context.Context) ([]Root, []judged, error) {
	var (
		all  []Root
		lost map[ID]bool
	)
	err := r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		if all, err = All(ctx, rd); err != nil {
			return err
		}
		lost, err = LostIn(ctx, rd)
		return err
	})
	if err != nil {
		return nil, nil, fmt.Errorf("read the library roots: %w", err)
	}
	states, err := r.judge(all, lost)
	if err != nil {
		return nil, nil, err
	}
	return all, states, nil
}
