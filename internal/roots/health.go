package roots

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jatassi/slipstream/internal/health"
	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/nodes"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// HealthEvents are the events after which HealthSource may answer
// differently.
var HealthEvents = []live.Event{ChangedEvent, nodes.ChangedEvent}

// HealthVolumes is the volumes that hold a root last seen ready at its
// place, as Volumes lists them, so an unmounted root's parent disk is never
// judged. An offline Node's volumes keep their last figures.
func (r *Roots) HealthVolumes() health.VolumeSource {
	return health.VolumeSource{
		Volumes: r.healthVolumes,
		Sample: func(ctx context.Context, node role.NodeID) error {
			return unreachable(r.Sample(ctx, node))
		},
		Events: []live.Event{VolumesChangedEvent, ChangedEvent, nodes.ChangedEvent},
	}
}

func (r *Roots) healthVolumes(ctx context.Context) ([]health.Volume, error) {
	names, err := r.cfg.Names(ctx)
	if err != nil {
		return nil, err
	}
	listed, err := r.volumes(ctx)
	if err != nil {
		return nil, err
	}
	var out []health.Volume
	for _, n := range listed {
		for _, v := range n.Volumes {
			out = append(out, health.Volume{
				Node: n.Node, NodeName: names[n.Node].Name, Mount: v.Mount, Name: v.Name,
				Total: v.Total, Free: v.Free, SampledAt: v.SampledAt, Live: n.Live,
			})
		}
	}
	return out, nil
}

// unreachable wraps health.ErrUnreachable around a Node that is away or no
// longer holds Storage.
func unreachable(err error) error {
	if errors.Is(err, ErrNodeAway) || errors.Is(err, ErrNotStorage) {
		return fmt.Errorf("%w: %w", health.ErrUnreachable, err)
	}
	return err
}

// trouble is what a root item says of each State that is wrong: what the
// root's row says, then what to do.
var trouble = map[State]struct{ sentence, fix string }{
	Missing:      {"The folder isn't there.", "Restore it, or edit the root to where it is now."},
	Unmounted:    {"The folder is there but looks unmounted.", "Plug in or mount the disk."},
	Foreign:      {"The folder holds another library root's marker.", "Edit the root to a folder of its own."},
	Unwritable:   {"SlipStream can't write to the folder.", "Let SlipStream write to it."},
	Unresponsive: {"The folder isn't answering.", "Check the disk or share it is on."},
	WrittenOff:   {"Its Node was written off, so its files count as missing.", "Re-home the root to a Node that holds Storage."},
}

// HealthSource answers a Fail item per root whose State is in trouble: a
// written-off one links to Re-home, the rest offer Check now, which samples
// the root's Node. A root on an offline Node, or not yet checked, is
// unsure, so an offline Node raises only its Nodes item and a restart
// keeps since.
func (r *Roots) HealthSource() health.Source {
	return health.Source{Answer: r.healthAnswer, Check: r.healthCheck}
}

type rootSaid struct {
	root  Root
	key   health.Key
	node  string
	words string
	judged
}

// checked is whether Check now on the root's item samples its Node.
func (s rootSaid) checked() bool {
	_, wrong := trouble[s.state]
	return wrong && s.state != WrittenOff
}

func (s rootSaid) item() health.Item {
	t := trouble[s.state]
	detail := []health.Fact{{Label: "Path", Text: s.root.Path}, {Label: "State", Text: t.sentence}}
	if s.detail != "" {
		detail = append(detail, health.Fact{Label: "Error", Text: s.detail})
	}
	action := health.LinkAction("/settings/library-roots", "Re-home")
	if s.checked() {
		action = health.CheckAction()
	}
	return health.Item{Key: s.key, Level: health.Fail, Says: health.Says{Subject: s.words, Text: t.sentence + " " + t.fix, Detail: detail, Action: action}}
}

func (r *Roots) healthAnswer(ctx context.Context) (health.Answer, error) {
	said, err := r.said(ctx)
	if err != nil {
		return health.Answer{}, err
	}
	var a health.Answer
	for _, s := range said {
		switch s.state {
		case Ready:
		case Offline:
			a.Unsure = append(a.Unsure, health.Unsure{Key: s.key, Says: health.Says{Subject: s.words, Text: s.node + " is offline."}})
		case Checking:
			a.Unsure = append(a.Unsure, health.Unsure{Key: s.key, Says: health.Says{Subject: s.words, Text: "Waiting for " + s.node + " to check the folder."}})
		case Missing, Unmounted, Foreign, Unwritable, Unresponsive, WrittenOff:
			a.Items = append(a.Items, s.item())
		}
	}
	return a, nil
}

// said is every root with its State and the words naming it: "Movies
// library root on Media server", with its path when its Node holds
// another root of its type.
func (r *Roots) said(ctx context.Context) ([]rootSaid, error) {
	names, err := r.cfg.Names(ctx)
	if err != nil {
		return nil, err
	}
	var (
		all  []Root
		lost map[ID]bool
	)
	err = r.cfg.Library.Read(ctx, func(ctx context.Context, rd storage.Reader[file]) error {
		var err error
		if all, err = All(ctx, rd); err != nil {
			return err
		}
		lost, err = LostIn(ctx, rd)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("read the library roots: %w", err)
	}
	states, err := r.judge(all, lost)
	if err != nil {
		return nil, err
	}
	type typeOn struct {
		t    works.TypeID
		node role.NodeID
	}
	twins := map[typeOn]int{}
	for _, root := range all {
		twins[typeOn{root.ContentType, root.Node}]++
	}
	out := make([]rootSaid, len(all))
	for i, root := range all {
		node := names[root.Node].Name
		d, _ := r.cfg.Describe(root.ContentType)
		words := d.Work.Many + " library root on " + node
		if twins[typeOn{root.ContentType, root.Node}] > 1 {
			words = d.Work.Many + " library root at " + root.Path + " on " + node
		}
		out[i] = rootSaid{root: root, key: health.KeyOf("r", root.ID.String()), node: node, words: words, judged: states[i]}
	}
	return out, nil
}

// healthCheck samples the Node of the root item names. For the whole
// category it samples only the Nodes of roots in trouble that hold no
// judged volume, since Storage's own check samples those that do.
func (r *Roots) healthCheck(ctx context.Context, item health.Key) ([]health.Unreached, error) {
	said, err := r.said(ctx)
	if err != nil {
		return nil, err
	}
	var on []rootSaid
	if item != "" {
		i := slices.IndexFunc(said, func(s rootSaid) bool { return s.key == item })
		if i < 0 {
			return nil, health.ErrNoItem
		}
		on = said[i : i+1]
	} else {
		vols, err := r.healthVolumes(ctx)
		if err != nil {
			return nil, err
		}
		sampled := func(node role.NodeID) bool {
			return slices.ContainsFunc(vols, func(v health.Volume) bool { return v.Node == node }) ||
				slices.ContainsFunc(on, func(s rootSaid) bool { return s.root.Node == node })
		}
		for _, s := range said {
			if s.checked() && !sampled(s.root.Node) {
				on = append(on, s)
			}
		}
	}
	var (
		mu        sync.Mutex
		unreached []health.Unreached
		errs      []error
		wg        sync.WaitGroup
	)
	for _, s := range on {
		wg.Go(func() {
			err := unreachable(r.Sample(ctx, s.root.Node))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case errors.Is(err, health.ErrUnreachable):
				unreached = append(unreached, health.Unreached{Key: item, Words: s.node + " is offline."})
			case err != nil:
				errs = append(errs, err)
			}
		})
	}
	wg.Wait()
	slices.SortFunc(unreached, func(a, b health.Unreached) int { return cmp.Compare(a.Words, b.Words) })
	return unreached, errors.Join(errs...)
}
