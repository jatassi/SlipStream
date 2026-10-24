package health

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jatassi/slipstream/internal/live"
	"github.com/jatassi/slipstream/internal/role"
	"github.com/jatassi/slipstream/internal/setting"
)

// StorageParts is what the built-in Storage category judges: the volumes
// every holder samples, and the library roots' own items.
type StorageParts struct {
	// Volumes is one line per holder that samples volumes: library roots
	// now, the downloads volume and Backup destinations later.
	Volumes []VolumeSource
	// Roots answers a root missing, unmounted, foreign, unwritable,
	// unresponsive or written off. Its keys are KeyOf("r", ...).
	Roots Source
	// RootEvents are the events after which Roots may answer differently,
	// besides the changes Config.Roster follows.
	RootEvents []live.Event
}

// VolumeSource is one holder's volumes on every Node.
type VolumeSource struct {
	Volumes func(ctx context.Context) ([]Volume, error)
	// Sample asks node to sample now and returns once the fresh figures
	// are recorded; an error wrapping ErrUnreachable is a Node away.
	Sample func(ctx context.Context, node role.NodeID) error
	Events []live.Event
	// Remedy is what else frees a full volume of this holder's, finishing
	// "Free up space on it, or ...".
	Remedy string
}

// Volume is one volume as a holder last recorded it.
type Volume struct {
	Node     role.NodeID
	NodeName string
	// Mount is as the Node names it: "D:\", "/mnt/media", "\\nas\media".
	Mount string
	// Name is "D: (Media)".
	Name  string
	Total int64
	// Free is what SlipStream may write.
	Free      int64
	SampledAt time.Time
	// Live: the Node is in step; else the figures read as of SampledAt.
	Live bool
}

var ErrUnreachable = errors.New("the check couldn't reach its Node")

func volumeKey(node role.NodeID, mount string) Key {
	return KeyOf("v", node.String(), Opaque(mount))
}

func (h *Health) storage() Category {
	events := slices.Clone(h.cfg.Storage.RootEvents)
	for _, s := range h.cfg.Storage.Volumes {
		events = append(events, s.Events...)
	}
	return Category{
		About:  About{ID: StorageID, Label: "Storage", Phrase: "storage", Events: events},
		Source: Source{Answer: h.storageAnswer, Check: h.storageCheck},
	}
}

// sampled is every holder's volumes merged on (Node, mount), the newest
// sample winning, in Node then mount order.
type sampled struct {
	list  []Volume
	byKey map[Key]Volume
	// from is the holders reporting a volume on each Node.
	from map[role.NodeID][]VolumeSource
	// remedies is each volume's holders' remedies, in holder order.
	remedies map[Key][]string
}

func (h *Health) volumes(ctx context.Context) (sampled, error) {
	s := sampled{byKey: map[Key]Volume{}, from: map[role.NodeID][]VolumeSource{}, remedies: map[Key][]string{}}
	for _, src := range h.cfg.Storage.Volumes {
		vols, err := src.Volumes(ctx)
		if err != nil {
			return sampled{}, fmt.Errorf("read the volumes: %w", err)
		}
		on := map[role.NodeID]bool{}
		for _, v := range vols {
			k := volumeKey(v.Node, v.Mount)
			if was, ok := s.byKey[k]; !ok || v.SampledAt.After(was.SampledAt) {
				s.byKey[k] = v
			}
			if src.Remedy != "" && !slices.Contains(s.remedies[k], src.Remedy) {
				s.remedies[k] = append(s.remedies[k], src.Remedy)
			}
			if !on[v.Node] {
				on[v.Node] = true
				s.from[v.Node] = append(s.from[v.Node], src)
			}
		}
	}
	for _, v := range s.byKey {
		s.list = append(s.list, v)
	}
	slices.SortFunc(s.list, func(a, b Volume) int {
		return cmp.Or(a.Node.Compare(b.Node.UUID), cmp.Compare(a.Mount, b.Mount))
	})
	return s, nil
}

func (h *Health) storageAnswer(ctx context.Context) (Answer, error) {
	vs, err := h.volumes(ctx)
	if err != nil {
		return Answer{}, err
	}
	l := h.Levels()
	var a Answer
	for _, v := range vs.list {
		if v.Total == 0 {
			continue
		}
		lv := l.For(v.Node, v.Mount, v.Total)
		if level := lv.Judge(v.Free); level != None {
			a.Items = append(a.Items, Item{Key: volumeKey(v.Node, v.Mount), Level: level, Says: h.volumeSays(v, lv, vs.remedies[volumeKey(v.Node, v.Mount)])})
		}
	}
	if h.cfg.Storage.Roots.Answer != nil {
		roots, err := h.cfg.Storage.Roots.Answer(ctx)
		if err != nil {
			return Answer{}, fmt.Errorf("read the library roots: %w", err)
		}
		a.Items = append(a.Items, roots.Items...)
		a.Unsure = append(a.Unsure, roots.Unsure...)
		a.Next = roots.Next
	}
	return a, nil
}

func (h *Health) volumeSays(v Volume, lv VolumeLevels, remedies []string) Says {
	left := setting.Amount{N: v.Free, Unit: setting.BytesUnit}.Words()
	tail := "It has " + left + " left."
	if !v.Live {
		tail = fmt.Sprintf("It had %s left as of %s.", left, h.clock(v.SampledAt))
	}
	detail := []Fact{
		{Label: "Free", Bytes: &v.Free},
		{Label: "Total", Bytes: &v.Total},
		{Label: "Warning level", Text: levelWords(lv.Warn)},
		{Label: "Error level", Text: levelWords(lv.Error)},
	}
	if !v.SampledAt.IsZero() {
		detail = append(detail, Fact{Label: "Sampled", At: v.SampledAt})
	}
	return Says{
		Subject: v.Name + " on " + v.NodeName,
		Text:    strings.Join(append([]string{"Free up space on " + v.Name + " on " + v.NodeName}, remedies...), ", or ") + ". " + tail,
		Detail:  detail,
		Action:  CheckAction(),
	}
}

func levelWords(r Resolved) string {
	if r.Own {
		return r.Amount.Words() + " (this volume's own)"
	}
	return r.Amount.Words() + " (global)"
}

func (h *Health) clock(t time.Time) string { return t.In(h.cfg.Zone).Format("15:04") }

// storageCheck samples the Nodes behind item, or behind every volume and
// root when item is "". A root's item goes to the roots' own Check.
func (h *Health) storageCheck(ctx context.Context, item Key) ([]Unreached, error) {
	vs, err := h.volumes(ctx)
	if err != nil {
		return nil, err
	}
	var nodes []role.NodeID
	if item != "" {
		v, ok := vs.byKey[item]
		if !ok {
			return h.rootsCheck(ctx, item)
		}
		nodes = []role.NodeID{v.Node}
	} else {
		for _, v := range vs.list {
			if !slices.Contains(nodes, v.Node) {
				nodes = append(nodes, v.Node)
			}
		}
	}
	var (
		mu        sync.Mutex
		unreached []Unreached
		errs      []error
		wg        sync.WaitGroup
	)
	gather := func(got []Unreached, err error) {
		mu.Lock()
		defer mu.Unlock()
		unreached = append(unreached, got...)
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, node := range nodes {
		wg.Go(func() {
			err := sampleNode(ctx, vs.from[node], node)
			if errors.Is(err, ErrUnreachable) {
				gather([]Unreached{{Key: item, Words: h.offlineWords(vs, node)}}, nil)
				return
			}
			gather(nil, err)
		})
	}
	if item == "" && h.cfg.Storage.Roots.Check != nil {
		wg.Go(func() { gather(h.cfg.Storage.Roots.Check(ctx, "")) })
	}
	wg.Wait()
	slices.SortFunc(unreached, func(a, b Unreached) int { return cmp.Compare(a.Words, b.Words) })
	return unreached, errors.Join(errs...)
}

func (h *Health) rootsCheck(ctx context.Context, item Key) ([]Unreached, error) {
	if h.cfg.Storage.Roots.Check == nil {
		return nil, ErrNoCheck
	}
	return h.cfg.Storage.Roots.Check(ctx, item)
}

func sampleNode(ctx context.Context, from []VolumeSource, node role.NodeID) error {
	var errs []error
	for _, src := range from {
		if err := src.Sample(ctx, node); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// offlineWords is "Media server is offline, so its figures are as of
// 14:05.", from the Node's newest sample.
func (h *Health) offlineWords(vs sampled, node role.NodeID) string {
	var name string
	var at time.Time
	for _, v := range vs.list {
		if v.Node == node {
			name = v.NodeName
			if v.SampledAt.After(at) {
				at = v.SampledAt
			}
		}
	}
	return fmt.Sprintf("%s is offline, so its figures are as of %s.", name, h.clock(at))
}
