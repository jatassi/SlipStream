package formats

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/jatassi/slipstream/internal/regex"
	"github.com/jatassi/slipstream/internal/release"
)

type kind struct {
	KindInfo
	// cost orders a format's groups for Match: value kinds, then patterns
	// by the length of the text they read.
	cost    int
	compile func(fields) (test, error)
}

var kinds = []kind{
	{KindInfo{ReleaseTitle, []string{"value"}, true}, 3, patternOn(onTitle)},
	{KindInfo{Edition, []string{"value"}, true}, 2, patternOn(onEdition)},
	{KindInfo{Language, []string{"value", "exceptLanguage"}, false}, 0, compileLanguage},
	{KindInfo{IndexerFlag, []string{"value"}, false}, 0, compileFlag},
	{KindInfo{Source, []string{"value"}, false}, 0, compileSource},
	{KindInfo{Resolution, []string{"value"}, false}, 0, compileResolution},
	{KindInfo{QualityModifier, []string{"value"}, false}, 0, compileModifier},
	{KindInfo{Size, []string{"min", "max"}, false}, 0, compileSize},
	{KindInfo{ReleaseGroup, []string{"value"}, true}, 1, patternOn(onGroup)},
	{KindInfo{ReleaseType, []string{"value"}, false}, 0, compileSpan},
	{KindInfo{Year, []string{"min", "max"}, false}, 0, compileYear},
}

// Kinds lists every kind with its fields, in upstream's order. The editor
// and /custom-formats/schema read it.
func Kinds() []KindInfo {
	out := make([]KindInfo, len(kinds))
	for i, k := range kinds {
		out[i] = k.KindInfo
	}
	return out
}

func kindOf(k Kind) (*kind, bool) {
	i := slices.IndexFunc(kinds, func(row kind) bool { return row.Kind == k })
	if i < 0 {
		return nil, false
	}
	return &kinds[i], true
}

// test is one condition before Negate. Every test is comparable, so a
// condition two formats share compiles to one test and runs once per name.
type test interface {
	run(*input) (matched, timedOut bool)
}

// input is one Read as the tests read it, each text split into runes
// once for every format.
type input struct {
	read      *release.Read
	texts     [3]regex.Text
	known     [3]bool
	languages []release.Language
}

type textOf uint8

const (
	onTitle textOf = iota
	onGroup
	onEdition
)

func newInput(r *release.Read) *input {
	in := &input{read: r, languages: languagesOf(r.Facts)}
	in.texts[onTitle], in.known[onTitle] = regex.TextOf(r.Given), true
	if g, ok := r.Facts.Group.Get(); ok {
		in.texts[onGroup], in.known[onGroup] = regex.TextOf(g), true
	}
	if e, ok := r.Facts.Edition.Get(); ok {
		in.texts[onEdition], in.known[onEdition] = regex.TextOf(e), true
	}
	return in
}

// languagesOf is upstream's AggregateLanguages: a name with no language
// tag reads as the Work's original language, and a name tagged as the
// original reads as it.
func languagesOf(f release.Facts) []release.Language {
	langs, _ := f.Languages.Get()
	original, ok := f.Coverage.Original.Get()
	if !ok {
		original = unknownLanguage
	}
	if len(langs) == 0 {
		return []release.Language{original}
	}
	i := slices.Index(langs, release.Original)
	if i < 0 {
		return langs
	}
	langs = slices.Delete(slices.Clone(langs), i, i+1)
	if slices.Contains(langs, original) {
		return append(langs, unknownLanguage)
	}
	return append(langs, original)
}

type fields struct {
	at      int
	c       *Condition
	by      map[string]json.RawMessage
	dialect *dialect
	timeout time.Duration
}

// pattern is a regex kind: RegexSpecificationBase.MatchString on one text,
// which never matches an unknown text.
type pattern struct {
	on textOf
	re *regex.Regex
}

func (p pattern) run(in *input) (matched, timedOut bool) {
	if !in.known[p.on] {
		return false, false
	}
	ok, err := p.re.Match(in.texts[p.on])
	return ok, err != nil
}

func patternOn(on textOf) func(fields) (test, error) {
	return func(f fields) (test, error) {
		var s string
		if err := f.decode("value", &s, "text"); err != nil {
			return nil, err
		}
		if s == "" {
			return nil, &PatternError{Condition: f.at, Name: f.c.Name, Offset: -1, Reason: "it's empty"}
		}
		re, err := regex.Cached(s, regex.Options{IgnoreCase: true, Timeout: f.timeout})
		if err != nil {
			return nil, f.patternError(err)
		}
		return pattern{on: on, re: re}, nil
	}
}

// languageIs is LanguageSpecification. Original reads as the Work's
// original language when it is known. Negate needs no case of its own
// here, because the languages are never null.
type languageIs struct {
	lang   release.Language
	except bool
}

func (l languageIs) run(in *input) (matched, timedOut bool) {
	want := l.lang
	if o, ok := in.read.Facts.Coverage.Original.Get(); ok && want == release.Original {
		want = o
	}
	if l.except {
		return slices.ContainsFunc(in.languages, func(have release.Language) bool { return have != want }), false
	}
	return slices.Contains(in.languages, want), false
}

func compileLanguage(f fields) (test, error) {
	k, err := f.key("value")
	if err != nil {
		return nil, err
	}
	l, ok := f.dialect.language(k)
	if !ok {
		return nil, f.valueError("value")
	}
	var except bool
	if err := f.decode("exceptLanguage", &except, "true or false"); err != nil {
		return nil, err
	}
	return languageIs{lang: l, except: except}, nil
}

type flagIs release.Flag

func (f flagIs) run(in *input) (matched, timedOut bool) {
	return slices.Contains(in.read.Facts.Flags, release.Flag(f)), false
}

func compileFlag(f fields) (test, error) {
	flag, err := lookup(f, f.dialect.flags)
	return flagIs(flag), err
}

func (s sourceIs) run(in *input) (matched, timedOut bool) {
	o, _ := in.read.Facts.Source.Get()
	if o != s.origin {
		return false, false
	}
	if s.modifier == 0 {
		return true, false
	}
	m, _ := in.read.Facts.Modifier.Get()
	return (m == s.modifier) == s.raw, false
}

func compileSource(f fields) (test, error) {
	return lookup(f, f.dialect.sources)
}

type resolutionIs release.Resolution

func (r resolutionIs) run(in *input) (matched, timedOut bool) {
	have, _ := in.read.Facts.Resolution.Get()
	return have == release.Resolution(r), false
}

func compileResolution(f fields) (test, error) {
	r, err := lookup(f, f.dialect.resolutions)
	return resolutionIs(r), err
}

type modifierIs release.Modifier

func (m modifierIs) run(in *input) (matched, timedOut bool) {
	have, _ := in.read.Facts.Modifier.Get()
	return have == release.Modifier(m), false
}

func compileModifier(f fields) (test, error) {
	m, err := lookup(f, f.dialect.modifiers)
	return modifierIs(m), err
}

type spanIs release.Span

func (s spanIs) run(in *input) (matched, timedOut bool) {
	return in.read.Name.Span == release.Span(s), false
}

func compileSpan(f fields) (test, error) {
	s, err := lookup(f, f.dialect.spans)
	return spanIs(s), err
}

// sizeIn is SizeSpecification: above Min and at most Max, in GiB as
// upstream's Gigabytes() converts them. An unknown size reads as 0, as
// upstream's does.
type sizeIn struct{ min, max int64 }

func (s sizeIn) run(in *input) (matched, timedOut bool) {
	size, _ := in.read.Facts.Size.Get()
	return size > s.min && size <= s.max, false
}

func compileSize(f fields) (test, error) {
	var lo, hi float64
	if err := f.decode("min", &lo, "a number"); err != nil {
		return nil, err
	}
	if err := f.decode("max", &hi, "a number"); err != nil {
		return nil, err
	}
	return sizeIn{min: gigabytes(lo), max: gigabytes(hi)}, nil
}

func gigabytes(gb float64) int64 { return int64(math.RoundToEven(gb * (1 << 30))) }

// yearIn is YearSpecification, inclusive at both ends. An unknown year
// never matches.
type yearIn struct{ min, max int }

func (y yearIn) run(in *input) (matched, timedOut bool) {
	year, ok := in.read.Name.Year.Get()
	return ok && year >= y.min && year <= y.max, false
}

func compileYear(f fields) (test, error) {
	var y yearIn
	if err := f.decode("min", &y.min, "a whole number"); err != nil {
		return nil, err
	}
	if err := f.decode("max", &y.max, "a whole number"); err != nil {
		return nil, err
	}
	return y, nil
}

func lookup[T any](f fields, table map[string]T) (T, error) {
	var zero T
	k, err := f.key("value")
	if err != nil {
		return zero, err
	}
	v, ok := table[k]
	if !ok {
		return zero, f.valueError("value")
	}
	return v, nil
}

// key is a value field as its dialect keys it: a number in Radarr's and
// Sonarr's, a name in SlipStream's.
func (f fields) key(name string) (string, error) {
	if f.dialect.name == SlipStream {
		var s string
		err := f.decode(name, &s, "text")
		return s, err
	}
	var n int
	err := f.decode(name, &n, "a whole number")
	return strconv.Itoa(n), err
}

// decode reads one field into v, leaving v as it is when the field is
// absent, as upstream's defaults do.
func (f fields) decode(name string, v any, want string) error {
	raw, ok := f.by[name]
	if !ok {
		return nil
	}
	if err := json.Unmarshal(raw, v); err != nil {
		return &ValueError{Name: f.c.Name, Kind: f.c.Kind, Field: name, Value: string(bytes.TrimSpace(raw)), Want: want}
	}
	return nil
}

// patternError reads Cached's error, which is always a *CompileError.
func (f fields) patternError(err error) *PatternError {
	var ce *regex.CompileError
	errors.As(err, &ce)
	e := &PatternError{Condition: f.at, Name: f.c.Name, Offset: ce.Offset, Reason: ce.Reason}
	if ce.Offset == 0 {
		e.Offset = -1
	}
	return e
}

func (f fields) valueError(name string) *ValueError {
	return &ValueError{
		Name: f.c.Name, Kind: f.c.Kind, Field: name, Value: string(bytes.TrimSpace(f.by[name])),
		Want: "a " + string(f.dialect.name) + " " + string(f.c.Kind) + " value",
	}
}
