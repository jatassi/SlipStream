package naming

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/jatassi/slipstream/internal/setting"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// Panel is t's naming settings: Fields lists every key with its default,
// and Apply runs the save check on the whole row as the patch leaves it
// and stores all or none. Apply refuses with *setting.UnknownKeyError and
// *RefusedError, and publishes Changed once it stores.
func (n *Naming) Panel(t works.TypeID) setting.Panel { return panel{n: n, t: t} }

type panel struct {
	n *Naming
	t works.TypeID
}

func (p panel) Fields(ctx context.Context) ([]setting.Field, error) {
	s, err := storedIn(ctx, p.n.cfg.Library, p.t)
	if err != nil {
		return nil, err
	}
	ks := p.n.keysOf(p.t)
	out := make([]setting.Field, len(ks))
	for i, k := range ks {
		out[i] = k.wire(s)
	}
	return out, nil
}

func (p panel) Apply(ctx context.Context, patch setting.Patch) error {
	err := p.n.cfg.Library.Write(ctx, func(ctx context.Context, tx *storage.Tx[storage.Library]) error {
		_, s, err := p.n.draft(ctx, tx, p.t, patch)
		if err != nil {
			return err
		}
		return store(ctx, tx, p.t, s)
	})
	if err != nil {
		return err
	}
	p.n.cfg.Changed.Publish(ctx, Changed{ContentType: string(p.t)})
	return nil
}

// key is one naming setting of a type: its wire field read from the
// stored Settings, and how a patch value changes them.
type key struct {
	// name is the setting key, "<type>.naming." and field.
	name, field string
	wire        func(Settings) setting.Field
	set         func(*Settings, json.RawMessage) error
}

type keys []key

// keysOf is t's settings in the order the card lists them: the rename
// switch, the folder templates outermost first, the file templates, the
// colon mode and its text, and the multi-episode style when t's file
// templates have an episode block.
func (n *Naming) keysOf(t works.TypeID) keys {
	d := n.cfg.Facets[t].Naming()
	nouns, _ := n.cfg.Describe(t)
	prefix := prefixOf(t)
	sw := setting.Switch{Key: prefix + "renameFiles", Label: "Rename files", Default: defaultRename}
	ks := keys{keyOf(sw.Key, "renameFiles", sw.ParseJSON, sw.Field,
		func(s Settings) *bool { return s.Rename }, func(s *Settings, v *bool) { s.Rename = v })}
	tpl := func(field, label, text string) {
		ks = append(ks, templateKey(setting.Template{Key: prefix + field, Label: label, Default: text, Tokens: field, Check: parses}))
	}
	tpl(folderKey, nouns.Work.One+" folder", d.Folder)
	for i, text := range d.Levels {
		tpl(levelKey(i), nouns.Levels[i].One+" folder", text)
	}
	if d.Aside != "" {
		tpl(asideKey, nouns.Aside.Many+" folder", d.Aside)
	}
	for _, v := range d.Files {
		tpl(fileKey(v), v.Label, v.File)
	}
	colon := choiceOf(prefix+"colon", "Colon replacement", defaultColon, colonWords)
	ks = append(ks,
		keyOf(colon.Key, "colon", colon.ParseJSON, colon.Field,
			func(s Settings) *string { return toWord(colonWords, s.Colon) },
			func(s *Settings, w *string) { s.Colon = fromWord(colonWords, w) }),
		templateKey(setting.Template{Key: prefix + colonCustomKey, Label: "Custom colon replacement", Tokens: colonCustomKey, Check: parses, Plain: true}),
	)
	if slices.ContainsFunc(d.Tokens, func(tk Token) bool { return tk.Seq != 0 }) {
		multi := choiceOf(prefix+"multiEpisode", fmt.Sprintf("Multi-%s style", strings.ToLower(nouns.Item.One)), defaultMulti, multiWords)
		ks = append(ks, keyOf(multi.Key, "multiEpisode", multi.ParseJSON, multi.Field,
			func(s Settings) *string { return toWord(multiWords, s.MultiEpisode) },
			func(s *Settings, w *string) { s.MultiEpisode = fromWord(multiWords, w) }))
	}
	return ks
}

func keyOf[T any, F setting.Field](name, field string, parse func(json.RawMessage) (setting.Stored[T], error), wire func(*T) F, get func(Settings) *T, put func(*Settings, *T)) key {
	return key{
		name:  name,
		field: field,
		wire:  func(s Settings) setting.Field { return wire(get(s)) },
		set: func(s *Settings, raw json.RawMessage) error {
			v, err := parse(raw)
			if err != nil {
				return err
			}
			put(s, v.Override())
			return nil
		},
	}
}

func templateKey(t setting.Template) key {
	return keyOf(t.Key, t.Tokens, t.ParseJSON, t.Field,
		func(s Settings) *string { return s.Templates[t.Tokens] },
		func(s *Settings, v *string) { s.Templates[t.Tokens] = v })
}

// parses is the Template kind's check: syntax only. The save check binds
// the template to its field.
func parses(text string) error {
	_, err := Parse(text)
	return err
}

// known refuses the first key of d, in key order, that ks has no setting
// for.
func (ks keys) known(d setting.Patch) error {
	for _, name := range slices.Sorted(maps.Keys(d)) {
		if !slices.ContainsFunc(ks, func(k key) bool { return k.name == name }) {
			return &setting.UnknownKeyError{Key: name}
		}
	}
	return nil
}

// patch sets each value of d on s, every key of d known. A value its kind
// refuses is a Problem, in key order.
func (ks keys) patch(s *Settings, d setting.Patch) ([]Problem, error) {
	var problems []Problem
	for _, k := range ks {
		raw, ok := d[k.name]
		if !ok {
			continue
		}
		err := k.set(s, raw)
		if refused, ok := errors.AsType[*setting.RangeError](err); ok {
			problems = append(problems, Problem{Key: k.field, Sentence: refused.Sentence})
		} else if err != nil {
			return nil, err
		}
	}
	return problems, nil
}

// The switches' defaults, which newScheme and the settings share.
const (
	defaultRename = true
	defaultColon  = ColonSmart
	defaultMulti  = PrefixedRange
)

// word is how a switch's value is stored and sent, and its label.
type word[T comparable] struct {
	v           T
	word, label string
}

var colonWords = []word[Colon]{
	{ColonDelete, "delete", "Delete"},
	{ColonDash, "dash", "Dash (-)"},
	{ColonSpaceDash, "space_dash", "Space dash ( -)"},
	{ColonSpaceDashSpace, "space_dash_space", "Space dash space ( - )"},
	{ColonSmart, "smart", "Smart"},
	{ColonCustom, "custom", "Custom"},
}

var multiWords = []word[MultiEpisode]{
	{Extend, "extend", "Extend"},
	{Duplicate, "duplicate", "Duplicate"},
	{Repeat, "repeat", "Repeat"},
	{Scene, "scene", "Scene"},
	{Range, "range", "Range"},
	{PrefixedRange, "prefixed_range", "Prefixed range"},
}

func choiceOf[T comparable](name, label string, def T, ws []word[T]) setting.Choice[string] {
	c := setting.Choice[string]{Key: name, Label: label, Default: *toWord(ws, &def)}
	for _, w := range ws {
		c.Options = append(c.Options, setting.Option[string]{Value: w.word, Label: w.label})
	}
	return c
}

func toWord[T comparable](ws []word[T], v *T) *string {
	if v == nil {
		return nil
	}
	w := ws[slices.IndexFunc(ws, func(w word[T]) bool { return w.v == *v })].word
	return &w
}

func fromWord[T comparable](ws []word[T], w *string) *T {
	if w == nil {
		return nil
	}
	v := ws[slices.IndexFunc(ws, func(x word[T]) bool { return x.word == *w })].v
	return &v
}
