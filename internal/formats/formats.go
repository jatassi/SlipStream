// Package formats is the Custom Format model in Sonarr's and Radarr's
// shape: its condition kinds, the combining rule, the dialect tables, the
// compile check and the paste shapes. It reads release.Read and stores
// nothing.
package formats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jatassi/slipstream/internal/release"
)

type Timings struct {
	// Match is how long one condition's pattern may run against one name
	// before it counts as not matched.
	Match time.Duration `yaml:"match"`
}

func DefaultTimings() Timings { return Timings{Match: 100 * time.Millisecond} }

// ID is a format's id. targets mints it; formats only compares and prints
// it.
type ID [16]byte

// Dialect is the app a format was written for, which numbers its source,
// resolution, modifier, language and flag values.
type Dialect string

const (
	Radarr     Dialect = "radarr"
	Sonarr     Dialect = "sonarr"
	SlipStream Dialect = "slipstream"
)

type Format struct {
	ID                  ID
	Name                string
	IncludeWhenRenaming bool
	Dialect             Dialect
	Conditions          []Condition
	// Source is the pasted document, verbatim: trash_id, trash_scores and
	// any other key ride along. Nil for a format made here. Never edited.
	Source []byte
}

type Condition struct {
	Name     string
	Kind     Kind
	Negate   bool
	Required bool
	// Fields are upstream's object form, {"value": …} or {"min": …, "max": …}.
	Fields json.RawMessage
}

// Kind is upstream's implementation name.
type Kind string

const (
	ReleaseTitle    Kind = "ReleaseTitleSpecification"
	ReleaseGroup    Kind = "ReleaseGroupSpecification"
	Edition         Kind = "EditionSpecification"
	Language        Kind = "LanguageSpecification"
	IndexerFlag     Kind = "IndexerFlagSpecification"
	Source          Kind = "SourceSpecification"
	Resolution      Kind = "ResolutionSpecification"
	QualityModifier Kind = "QualityModifierSpecification"
	Size            Kind = "SizeSpecification"
	ReleaseType     Kind = "ReleaseTypeSpecification"
	Year            Kind = "YearSpecification"
)

// KindInfo is one kind as the editor offers it: its fields, and whether
// its value is a pattern.
type KindInfo struct {
	Kind   Kind
	Fields []string
	Regex  bool
}

// Kinds lists every kind with its fields, in upstream's order. The editor
// and /custom-formats/schema read it.
func Kinds() []KindInfo { panic("not implemented") }

// Check compiles a format with the Match limit, and names the first
// condition that doesn't compile: a *PatternError, or a *FieldError for a
// field its kind lacks.
func Check(Format, Timings) error { panic("not implemented") }

// PatternError is a condition whose pattern doesn't compile.
type PatternError struct {
	Condition int
	Name      string
	// Offset is the rune the engine stopped at, -1 when it can't say.
	Offset int
	Reason string
}

func (e *PatternError) Error() string {
	return fmt.Sprintf("the pattern of condition %q doesn't compile: %s", e.Name, e.Reason)
}

// KindError is a pasted condition of a kind no Custom Format has.
type KindError struct {
	Name string
	Kind string
}

func (e *KindError) Error() string {
	return fmt.Sprintf("condition %q has the kind %s, which isn't a Custom Format condition kind", e.Name, e.Kind)
}

// FieldError is a condition with a field its kind lacks.
type FieldError struct {
	Name  string
	Kind  Kind
	Field string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("condition %q has the field %s, which %s conditions don't have", e.Name, e.Field, e.Kind)
}

// Catalog is a compiled set of formats, immutable and safe to share.
type Catalog struct{}

// Compile compiles formats with the Match limit. A format that no longer
// compiles matches nothing and is listed by Broken.
func Compile([]Format, Timings) *Catalog { panic("not implemented") }

// Only is the catalog's formats the Set names.
func (*Catalog) Only(Set) *Catalog { panic("not implemented") }

// With is the catalog with a format added, replacing the one of its ID:
// the tester's draft.
func (*Catalog) With(Format) *Catalog { panic("not implemented") }

func (*Catalog) Broken() []ID { panic("not implemented") }

func (*Catalog) Len() int { panic("not implemented") }

// Set is matched formats, sorted by ID.
type Set []ID

type Result struct {
	Matched  Set
	Timeouts []Timeout
}

// Timeout is a condition that ran past its limit.
type Timeout struct {
	Format    ID
	Condition int
}

// Match is every format a Read matches, by upstream's rule: each condition
// matches, then Negate flips it; conditions group by kind; a group matches
// when no Required condition in it failed and one matched; a format
// matches when every group does. An unknown language reads as the Read's
// Facts.Coverage.Original. A timed-out condition counts as not matched.
// Match stops a format at its first failed group, cheap kinds first.
func (*Catalog) Match(release.Read) Result { panic("not implemented") }

// Explain is Match with every condition evaluated, for the tester.
func (*Catalog) Explain(release.Read) []Explained { panic("not implemented") }

type Explained struct {
	Format     ID
	Name       string
	Matched    bool
	TimedOut   bool
	Conditions []Evaluated
}

type Evaluated struct {
	Name     string
	Kind     Kind
	Negate   bool
	Required bool
	// Matched is after Negate.
	Matched  bool
	TimedOut bool
}

// Paste reads one format or a list, with fields as an object (TRaSH and
// export) or as Sonarr's API array. Unknown kinds and field names are a
// *KindError or *FieldError naming them. Each Format's Source is its
// document, verbatim.
func Paste([]byte) ([]Format, error) { panic("not implemented") }

// AppOf is the app formats were written for when their kinds settle it:
// Edition, QualityModifier or Year say Radarr, ReleaseType says Sonarr.
func AppOf([]Format) (Dialect, bool) { panic("not implemented") }
