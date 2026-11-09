package naming

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/works"
)

// Preview is a type's naming settings under a draft, as the card shows
// each one while the Admin types.
type Preview struct{ Fields []FieldPreview }

// FieldPreview is one setting under the draft.
type FieldPreview struct {
	// Key is the setting key.
	Key string
	// Samples are the full paths a folder or file template names for the
	// type's samples, with renaming on. A template that doesn't parse
	// has none.
	Samples []SamplePath
	// Problems are what the save check refuses in the setting, each a
	// sentence to WRITING.md.
	Problems []string
}

// SamplePath is the root-relative path one sample names.
type SamplePath struct{ Label, Path string }

// Preview is t's settings with d applied over what is stored, judged as
// a save would judge them and named for the type's samples. It stores
// nothing. A template that doesn't parse shows no paths, and the others
// name with its stored value in its place. A key t has no setting for is
// *setting.UnknownKeyError.
func (n *Naming) Preview(ctx context.Context, t works.TypeID, d setting.Patch) (Preview, error) {
	ks := n.keysOf(t)
	if err := ks.known(d); err != nil {
		return Preview{}, err
	}
	sc, problems, err := n.parsed(ctx, t, ks, d)
	if err != nil {
		return Preview{}, err
	}
	budget, err := n.cfg.Budget(ctx, n.cfg.Library, t)
	if err != nil {
		return Preview{}, err
	}
	unread := map[string]bool{}
	for _, p := range problems {
		unread[p.Key] = true
	}
	samples := sc.renaming().samplePaths(budget)
	for field := range samples {
		if unread[field] {
			delete(samples, field)
		}
	}
	for _, p := range check(sc, budget) {
		if !unread[p.Key] {
			problems = append(problems, p)
		}
	}
	out := Preview{Fields: make([]FieldPreview, len(ks))}
	for i, k := range ks {
		f := FieldPreview{Key: k.name, Samples: samples[k.field]}
		for _, p := range problems {
			if p.Key == k.field {
				f.Problems = append(f.Problems, p.Sentence)
			}
		}
		out.Fields[i] = f
	}
	return out, nil
}

// parsed is t's scheme with d applied over what is stored, and the
// Problems of each value its kind refuses or that doesn't bind. Each of
// those names with its stored value in its place.
func (n *Naming) parsed(ctx context.Context, t works.TypeID, ks keys, d setting.Patch) (*Scheme, []Problem, error) {
	stored, err := storedIn(ctx, n.cfg.Library, t)
	if err != nil {
		return nil, nil, err
	}
	s := stored
	s.Templates = maps.Clone(stored.Templates)
	problems, err := ks.patch(&s, d)
	if err != nil {
		return nil, nil, err
	}
	decl := n.cfg.Facets[t].Naming()
	sc, err := newScheme(decl, s)
	if refused, ok := errors.AsType[*RefusedError](err); ok {
		problems = append(problems, refused.Problems...)
		for _, p := range refused.Problems {
			s.revert(stored, p.Key)
		}
		sc, err = newScheme(decl, s)
	}
	if err != nil {
		return nil, nil, keyed(t, err)
	}
	return sc, problems, nil
}

// Problems is what the save check refuses in t's stored settings once
// renaming is on, nil when every file of a Work names as one item.
func (n *Naming) Problems(ctx context.Context, t works.TypeID) ([]Problem, error) {
	s, err := storedIn(ctx, n.cfg.Library, t)
	if err != nil {
		return nil, err
	}
	on := true
	s.Rename = &on
	sc, err := newScheme(n.cfg.Facets[t].Naming(), s)
	if err != nil {
		return nil, keyed(t, err)
	}
	budget, err := n.cfg.Budget(ctx, n.cfg.Library, t)
	if err != nil {
		return nil, err
	}
	return prefixed(t, check(sc, budget)), nil
}

// Types are the Content Types Naming names files for, in id order.
func (n *Naming) Types() []works.TypeID { return slices.Sorted(maps.Keys(n.cfg.Facets)) }

// revert puts back the stored value of the setting a bind problem keyed
// field names. The colon text goes back with its mode, since only the
// Custom mode reads it.
func (s *Settings) revert(stored Settings, field string) {
	if field == colonCustomKey {
		s.Colon, s.ColonCustom = stored.Colon, stored.ColonCustom
		return
	}
	s.Templates[field] = stored.Templates[field]
}

// renaming is s with renaming on, which names the templates as they
// read.
func (s *Scheme) renaming() *Scheme {
	on := *s
	on.rename = true
	return &on
}

// samplePaths are the full paths s names for the type's samples, by
// field. A folder template shows each sample once, with the core's
// Standard file or, for a Missing sample, the file with no release
// facts. A file template shows each of its crossings. A sample that
// doesn't name within budget is left out, and the check says why.
func (s *Scheme) samplePaths(budget int) map[string][]SamplePath {
	out := map[string][]SamplePath{}
	name := func(sub Subject, fs fileSample) (string, bool) {
		named, err := s.Name(Placing{Subject: sub, LevelFolders: true, Landing: Rendered(), File: fs.facts, Ext: sampleExt, Budget: budget})
		return named.Rel, err == nil
	}
	for _, sample := range s.decl.Samples {
		fs := fileSamples[slices.IndexFunc(fileSamples, func(f fileSample) bool { return f.bare == sample.Missing })]
		p, ok := name(Subject{Work: sample.Work, Units: sample.Units}, fs)
		if !ok {
			continue
		}
		for _, field := range s.folderFields(sample) {
			out[field] = append(out[field], SamplePath{Label: sample.Label, Path: p})
		}
	}
	for v, variant := range s.decl.Files {
		key := fileKey(variant)
		for x := range s.crossings(v) {
			if p, ok := name(x.sub, x.file); ok {
				out[key] = append(out[key], SamplePath{Label: fmt.Sprintf("%s with %s", x.sample.Label, x.file.label), Path: p})
			}
		}
	}
	return out
}

// folderFields are the folder templates sample's path runs through.
func (s *Scheme) folderFields(sample Sample) []string {
	out := []string{folderKey}
	if len(sample.Units) == 0 {
		return out
	}
	for i := range s.levels {
		if i == 0 && sample.Units[0].Aside && s.decl.Aside != "" {
			out = append(out, asideKey)
			continue
		}
		out = append(out, levelKey(i))
	}
	return out
}
