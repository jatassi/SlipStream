package history

import "github.com/jatassi/slipstream/internal/history/entry"

// GroupKinds is one History group and its kinds, for the filter's chips.
type GroupKinds struct {
	Group entry.Group
	Label string
	Kinds []entry.Spec
}

// Kinds is the catalog by group, in filter order, each group's kinds in
// registration order. A group with no kinds yet is left out.
func (s *Service) Kinds() []GroupKinds {
	specs := s.cfg.Catalog.Specs()
	out := make([]GroupKinds, 0, len(entry.Groups()))
	for _, g := range entry.Groups() {
		gk := GroupKinds{Group: g, Label: g.Label()}
		for _, spec := range specs {
			if spec.Group == g {
				gk.Kinds = append(gk.Kinds, spec)
			}
		}
		if len(gk.Kinds) > 0 {
			out = append(out, gk)
		}
	}
	return out
}
