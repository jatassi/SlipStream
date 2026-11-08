// Package formats is the Custom Format model in Sonarr's and Radarr's
// shape: its condition kinds, the combining rule, the dialect tables, the
// compile check and the paste shapes. It reads release.Read and stores
// nothing.
package formats

import (
	"encoding/json"
	"fmt"
	"time"
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

// String is id in the UUID text form.
func (id ID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16])
}

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

// KindInfo is one kind as the editor offers it: its name in words, its
// fields, and whether its value is a pattern.
type KindInfo struct {
	Kind   Kind
	Words  string
	Fields []string
	Regex  bool
}

// PatternError is a condition whose pattern doesn't compile.
type PatternError struct {
	Condition int
	Name      string
	// Offset is the rune the engine stopped at, -1 when it can't say.
	Offset int
	Reason string
}

func (e *PatternError) Error() string {
	return fmt.Sprintf("the pattern of condition \"%s\" doesn't compile: %s", e.Name, e.Reason)
}

// KindError is a pasted condition of a kind no Custom Format has.
type KindError struct {
	Name string
	Kind string
}

func (e *KindError) Error() string {
	return fmt.Sprintf("condition \"%s\" has the kind %s, which isn't a Custom Format condition kind", e.Name, e.Kind)
}

// ValueError is a condition with a field its kind can't read in the
// format's dialect.
type ValueError struct {
	Name  string
	Kind  Kind
	Field string
	// Value is the field's JSON, verbatim.
	Value string
	// Want is what the field takes, in words: "a number", "a radarr
	// SourceSpecification value".
	Want string
}

func (e *ValueError) Error() string {
	return fmt.Sprintf("condition \"%s\" has the %s %s, which isn't %s", e.Name, e.Field, e.Value, e.Want)
}

// FieldError is a condition with a field its kind lacks.
type FieldError struct {
	Name  string
	Kind  Kind
	Field string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("condition \"%s\" has the field %s, which %s conditions don't have", e.Name, e.Field, e.Kind)
}

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
