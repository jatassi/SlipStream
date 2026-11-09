package naming

import (
	"fmt"

	"github.com/jatassi/slipstream/internal/media"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/works"
)

// Declaration is everything a Content Type says about naming its files:
// its tokens, its default templates, its file variants and its samples.
// The core derives the title variants, crosses the samples with its own
// file samples and runs the check. It is the same on every call.
type Declaration struct {
	Tokens []Token
	// Folder is the default Work folder template.
	Folder string
	// Levels are the default level folder templates, outermost first.
	// Empty puts files in the Work folder.
	Levels []string
	// Aside replaces the outermost level's template for an aside Item.
	Aside string
	// Files is one template per variant. The first is the fallback: a
	// Unit whose Item lacks a value its variant's anchor names renders the
	// first variant.
	Files   []Variant
	Samples []Sample
}

// Token is one name a template may use. A name and its aliases are
// matched by Key, so no two of a catalog's names may share one.
type Token struct {
	Name    string
	Aliases []string
	Kind    Kind
	Scope   Scope
	// Seq marks a number token's part in the episode block. Only KindNumber
	// tokens carry one.
	Seq     Seq
	Example string
	// About is one sentence for the token builder, written to WRITING.md.
	About string
}

// Kind is what a token's value is, which decides the modifiers it takes
// and the Value it holds.
type Kind uint8

const (
	// KindTitle is text that takes cuts, case and separators. A token
	// named "<X> Title" derives Sonarr's title variants.
	KindTitle Kind = iota + 1
	// KindYear is a whole year. It pairs with a KindTitle in its scope
	// for the TitleYear variants.
	KindYear
	// KindNumber pads.
	KindNumber
	// KindDate prints as "2026 11 09", so a separator replaces the spaces.
	KindDate
	// KindText takes cuts, case and separators and derives nothing.
	KindText
	// KindID prints as it is. Tags wrap it.
	KindID
	// KindList is a language list or a format list, which filters.
	KindList
)

// Scope is which tier of a Work a token's value comes from, and so which
// templates may name it.
type Scope uint8

const (
	// WorkScope tokens may appear in every template.
	WorkScope Scope = iota + 1
	// LevelScope tokens may appear in level and file templates.
	LevelScope
	// ItemScope tokens may appear in file templates.
	ItemScope
	// FileScope tokens may appear in file templates. Only the core
	// declares them.
	FileScope
)

// Seq is a number token's part in the episode block. The block runs from
// the first Outer or Inner ref to the last Inner ref of a file template,
// literals inside it included, and each Alone ref is a block of its own.
type Seq uint8

const (
	// SeqOuter numbers the level an Item sits in: a season.
	SeqOuter Seq = iota + 1
	// SeqInner numbers an Item within its Outer: an episode.
	SeqInner
	// SeqAlone numbers an Item with no Outer: an absolute episode.
	SeqAlone
)

// Variant is one file template the type picks per Item.
type Variant struct {
	// Key is stable: the setting key is "<type>.naming.file.<Key>".
	Key    string
	Label  string
	File   string
	Anchor Anchor
}

// Anchor is what every file name of a Work starts with up to the literal
// " - ", so a Media Server shows the Work's files as one item. Only
// FolderAnchor, TokenAnchor and BlockAnchor make one.
type Anchor struct {
	kind  anchorKind
	token string
}

type anchorKind uint8

const (
	folderAnchor anchorKind = iota + 1
	tokenAnchor
	blockAnchor
)

// FolderAnchor anchors a file name on the folder template's parts, and at
// render time on the Work folder the file lands in.
func FolderAnchor() Anchor { return Anchor{kind: folderAnchor} }

// TokenAnchor anchors a file name on the run of parts that ends with the
// named token.
func TokenAnchor(name string) Anchor { return Anchor{kind: tokenAnchor, token: name} }

// BlockAnchor anchors a file name on the run of parts that ends with the
// episode block.
func BlockAnchor() Anchor { return Anchor{kind: blockAnchor} }

// Sample is a subject the preview and the check render. The core pairs
// each with its own file samples: a Standard and a non-Standard cut, a
// Kept file, every release fact present and every one missing.
type Sample struct {
	Label string
	Work  Set
	// Units are the Items one file covers. Two or more make a
	// multi-episode sample.
	Units []Unit
	// Missing marks the sample with every optional value absent. The core
	// pairs it with the file sample that has no release facts.
	Missing bool
}

// Value is a token's value for one Work, level or Item. The zero Value is
// absent: the token prints nothing, its prefix, suffix and tag included.
// KindTitle, KindText and KindID tokens take Text, KindYear and KindNumber
// tokens take Number, KindDate tokens take Day and KindList tokens take
// List.
type Value struct {
	kind valueKind
	text string
	n    int64
	date works.Date
	list []string
}

type valueKind uint8

const (
	textValue valueKind = iota + 1
	numberValue
	dateValue
	listValue
)

// Text is s, and absent when s is empty.
func Text(s string) Value {
	if s == "" {
		return Value{}
	}
	return Value{kind: textValue, text: s}
}

// TextOr is *s, and absent when s is nil or empty.
func TextOr(s *string) Value {
	if s == nil {
		return Value{}
	}
	return Text(*s)
}

func Number(n int64) Value { return Value{kind: numberValue, n: n} }

// NumberOr is *n, and absent when n is nil.
func NumberOr(n *int64) Value {
	if n == nil {
		return Value{}
	}
	return Number(*n)
}

func Day(d works.Date) Value { return Value{kind: dateValue, date: d} }

// DayOr is *d, and absent when d is nil.
func DayOr(d *works.Date) Value {
	if d == nil {
		return Value{}
	}
	return Day(*d)
}

// List is items in order, and absent when there are none.
func List(items ...string) Value {
	if len(items) == 0 {
		return Value{}
	}
	return Value{kind: listValue, list: items}
}

// Set is values by declared token name.
type Set map[string]Value

// Values is one Work's: its own, each level's and each Item's.
type Values struct {
	Work   Set
	Levels map[works.LevelID]Set
	Items  map[works.ItemID]ItemValues
}

// ItemValues is an Item's values and the file variant it picks.
type ItemValues struct {
	Set
	// Variant is a Declaration.Files key. "" is the first variant.
	Variant string
}

// Subject is what one file's name reads of its Work: the Work's values
// and each Item the file covers, in order.
type Subject struct {
	Work  Set
	Units []Unit
}

// Unit is one Item of a file: its level's values, its own, and whether it
// sits in an aside level.
type Unit struct {
	Level Set
	Item  ItemValues
	Aside bool
}

// Of is the Subject of a file covering items: for each, in order, the
// values of its level in tree, its own values, and its Aside from tree.
func (v Values) Of(tree works.Tree, items []works.ItemID) Subject {
	byID := make(map[works.ItemID]works.Item, len(tree.Items))
	for _, it := range tree.Items {
		byID[it.ID] = it
	}
	s := Subject{Work: v.Work, Units: make([]Unit, len(items))}
	for i, id := range items {
		it := byID[id]
		s.Units[i] = Unit{Level: v.Levels[it.LevelID], Item: v.Items[id], Aside: it.Aside}
	}
	return s
}

// Template is a template parsed once. Rendering, the check, the anchor and
// the episode block read its parts. Nothing reparses text. The zero
// Template is the empty template.
type Template struct {
	text  string
	parts []part
}

// String is the template as typed.
func (t Template) String() string { return t.text }

// part is one of literal, *ref or *boundRef.
type part interface{ part() }

// literal is template text outside any token. Parse refuses a brace, "/"
// and "\" in one, so it never holds them.
type literal string

func (literal) part() {}

// ref is a token, or a tag around one, as typed: "{" prefix name ":" mod
// suffix "}". It knows no catalog. bind resolves it.
type ref struct {
	// text is the ref as typed, braces and tag included.
	text string
	// at is the byte offset of its first brace in the template.
	at int
	// name is as typed. Its case decides casing and the run between its
	// words decides sep.
	name string
	key  Key
	// tag wraps the value as "{" + word + "-" + value + "}", or nothing
	// when the value is absent.
	tag tag
	// prefix is from " -._[({" and suffix from " -._)]}". Both print only
	// with a value.
	prefix, suffix string
	// sep is '.', '-' or '_', which replaces spaces in the value, or 0.
	sep    byte
	casing casing
	// mod is nil, or the modifier text as syntax reads it.
	mod mod
}

func (*ref) part() {}

// Key is a token name as matched: lowercase, with spaces, dots, dashes and
// underscores dropped, as Sonarr's token comparer does.
type Key string

type tag uint8

const (
	noTag tag = iota
	editionTag
	imdbTag
	imdbidTag
	tmdbTag
	tmdbidTag
	tvdbTag
	tvdbidTag
)

type casing uint8

const (
	asIs casing = iota
	lower
	upper
)

// mod is a ref's modifier: padding, cut or words from Parse; langs,
// formatFilter or oneFormat once bind reads words by the token's kind.
type mod interface{ mod() }

// padding pads a number to its width: {season:00}.
type padding int

// cut keeps at most bytes of a value, "..." included, at a whole
// character: {Episode Title:30}. fromEnd keeps the end: {Episode Title:-30}.
type cut struct {
	bytes   int
	fromEnd bool
}

// words is modifier text that isn't a number: a language list, a format
// list or one format's name. Which it is depends on the token's kind, so
// bind reads it with langs, formats or one. drop is a leading "-".
type words struct {
	text string
	drop bool
}

// langs keeps the codes, or drops them with drop. mark prints "--" when
// the filter left a language out: {MediaInfo AudioLanguages:EN+DE+}.
type langs struct {
	codes []string
	drop  bool
	mark  bool
}

// formatFilter keeps the named formats, or drops them with drop:
// {Custom Formats:A,B}, {Custom Formats:-A,B}.
type formatFilter struct {
	names []string
	drop  bool
}

// oneFormat prints the one named format: {Custom Format:IMAX}.
type oneFormat string

func (padding) mod()      {}
func (cut) mod()          {}
func (words) mod()        {}
func (langs) mod()        {}
func (formatFilter) mod() {}
func (oneFormat) mod()    {}

// ParseError is a template Parse refuses: the byte offset of the problem,
// the token as typed (empty for a character outside any token) and a
// sentence that says what is wrong and what passes.
type ParseError struct {
	Pos      int
	Token    string
	Sentence string
	cause    string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("template at byte %d: %s", e.Pos, e.cause)
}

// field is which template of a type a template fills, which decides the
// scopes its tokens may have.
//
//nolint:unused // the engine lanes fill the code that reads it
type field uint8

//nolint:unused // the engine lanes fill the code that reads it
const (
	folderField field = iota + 1
	levelField
	fileField
)

// entry is one name a template may use: a declared token, a title variant
// derived from one, or a core token.
//
//nolint:unused // the engine lanes fill the code that reads it
type entry struct {
	Token
	key Key
	// tag is set for an alias that stands for a tagged ref: {Edition Tag}
	// is {edition-{Edition}}.
	tag tag
	// read is the entry's value for one Unit of a file: a declared token
	// looks its name up in the Subject, a derived variant computes from
	// its base token and year, a core token reads FileFacts.
	read func(s Subject, unit int, f FileFacts) Value
}

// catalog is one type's entries by key: the core's, the type's and the
// derived title variants.
//
//nolint:unused // the engine lanes fill the code that reads it
type catalog struct {
	byKey   map[Key]*entry
	ordered []*entry
}

// boundRef is a ref resolved against a catalog. Its mod is never words:
// bind reads words as langs, formatFilter or oneFormat.
type boundRef struct {
	ref
	entry *entry
}

func (*boundRef) part() {}

// bound is a Template bound to a catalog for one field: the only form
// render takes. Its parts are literals and *boundRefs.
//
//nolint:unused // the engine lanes fill the code that reads it
type bound struct {
	field field
	parts []part
	// block is the episode block's part indexes. Empty when the template
	// has no Outer or Inner ref.
	block span
	// alone are the part indexes of SeqAlone refs.
	alone []int
	// anchorEnd is where the anchor ends: parts [0, anchorEnd) are the
	// anchor and the literal " - " starts at anchorEnd.
	anchorEnd int
	// shrinkable are the part indexes of Item-scope KindTitle refs, the only
	// refs fit cuts.
	shrinkable []int
}

// span is a half-open range of part indexes.
type span struct{ from, to int }

// piece is one rendered part. Only literal pieces take part in the
// collapse, so a value's dots and a cut's "..." survive.
type piece struct {
	text    string
	literal bool
	// empty is a ref with no value, where the literals on each side meet.
	empty bool
}

// Colon is how a colon in a value prints.
type Colon uint8

const (
	ColonDelete Colon = iota + 1
	ColonDash
	ColonSpaceDash
	ColonSpaceDashSpace
	// ColonSmart is Sonarr's rule: ": " prints " - " and any other ":"
	// prints "-".
	ColonSmart
	ColonCustom
)

// MultiEpisode is how the episode block of a file with several Items
// prints, for episodes 1, 2 and 3 of season 1.
type MultiEpisode uint8

const (
	Extend        MultiEpisode = iota + 1 // S01E01-02-03
	Duplicate                             // S01E01 - S01E02 - S01E03
	Repeat                                // S01E01E02E03
	Scene                                 // S01E01-E02-E03
	Range                                 // S01E01-03
	PrefixedRange                         // S01E01-E03, the default
)

// Settings is one Content Type's stored overrides. nil is the type's
// default. Templates are by field key: "folder", "levels.<n>", "aside"
// and "file.<variant>".
type Settings struct {
	Rename       *bool
	Templates    map[string]*string
	Colon        *Colon
	ColonCustom  *string
	MultiEpisode *MultiEpisode
}

// Scheme is a type's Declaration under its Settings, bound once: what
// names every file of the type. It is immutable and safe for concurrent
// use.
//
//nolint:unused // the engine lanes fill the code that reads it
type Scheme struct {
	decl        Declaration
	cat         *catalog
	rename      bool
	folder      bound
	levels      []bound
	aside       bound
	files       []bound // by Declaration.Files index
	colon       Colon
	colonCustom string
	multi       MultiEpisode
}

// Placing is one file to name.
type Placing struct {
	Subject      Subject
	LevelFolders bool
	Landing      Landing
	File         FileFacts
	// Ext is the file's extension with its dot, case kept.
	Ext string
	// Budget is the bytes the root-relative path may take.
	Budget int
}

// Landing is where a file goes: folders rendered from the templates, a
// Work folder that exists with its level folders rendered, or a folder
// that exists whole. A FolderAnchor is the Work folder the file lands in.
//
//nolint:unused // the engine lanes fill the code that reads it
type Landing struct {
	// work is the Work folder. "" renders it.
	work string
	// dir, when set, is the folder the file goes straight into.
	dir string
}

// Rendered lands a file where the templates say.
func Rendered() Landing { return Landing{} }

// InWork lands a file in folder, the Work folder that exists, under
// rendered level folders.
func InWork(folder string) Landing { return Landing{work: folder} }

// InDir lands a file straight in a folder that exists, root-relative. Its
// Work folder is the folder's first segment.
func InDir(string) Landing {
	panic("not implemented")
}

// FileFacts is what the core tokens read of one file.
type FileFacts struct {
	// Target is the label with " (extra N)" on.
	Target string
	// Cut is the cut's name, "" for Standard.
	Cut   string
	Facts release.Facts
	// Media is read for the facts no merged reading has: bit depth and
	// subtitle languages.
	Media media.Record
	// Formats are the matched Custom Formats marked to include, in
	// catalog order.
	Formats []string
	Release string
	// Source is the downloaded or found file name with its extension, ""
	// when unknown.
	Source string
	// Keep is the name without extension that renaming off keeps.
	Keep string
}

// Named is a file's root-relative path and its segments, Work folder
// first.
type Named struct {
	Rel      string
	Segments []string
}

// Name names one file: the Work folder, the level folders and the file
// name, with a FolderAnchor swapped for the Work folder it lands in,
// Item-scope titles cut to fit Budget. A path that still doesn't fit is a
// *TooLongError. With renaming off the file segment is Keep under the
// cross-OS map.
func (s *Scheme) Name(Placing) (Named, error) {
	panic("not implemented")
}

// Folder renders the Work folder of a Work with no files from the Work's
// values.
func (s *Scheme) Folder(Set) (string, error) {
	panic("not implemented")
}

func (s *Scheme) Renames() bool {
	panic("not implemented")
}

// Reads reports the costly facts the scheme's templates read, so files
// matches formats only when a template prints them.
func (s *Scheme) Reads() Reads {
	panic("not implemented")
}

type Reads struct{ Formats bool }

// TooLongError is a name that doesn't fit after every shrinkable title is
// cut.
type TooLongError struct {
	// Segment is the segment that doesn't fit, or "" for the whole path.
	Segment string
	Bytes   int
	Limit   int
}

func (e *TooLongError) Error() string {
	if e.Segment == "" {
		return fmt.Sprintf("the path takes %d bytes and the limit is %d", e.Bytes, e.Limit)
	}
	return fmt.Sprintf("%q takes %d bytes and the limit is %d", e.Segment, e.Bytes, e.Limit)
}

// Problem is one setting the check refuses.
type Problem struct {
	// Key is the setting key, such as "<type>.naming.file.<variant>".
	Key string
	// Sentence says what is wrong and then what passes, to WRITING.md.
	Sentence string
}

// RefusedError is naming settings the check refuses.
type RefusedError struct{ Problems []Problem }

func (e *RefusedError) Error() string {
	return fmt.Sprintf("the naming settings fail %d checks", len(e.Problems))
}
