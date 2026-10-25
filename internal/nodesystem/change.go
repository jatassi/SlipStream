// Package nodesystem is a Node's system as an Admin manages it: its facts,
// the settings it keeps in config.yaml and why a change to them is
// refused, its restart, its log and the zip of its log folder. Every Node
// serves its Host over NodeSystemService, and the Controller reaches every
// Node, itself included, through Nodes.
package nodesystem

import "fmt"

// Change is one setting's new value, as any surface sends it.
type Change struct {
	Key   string
	Value Value
}

type form uint8

const (
	reset form = iota
	number
	word
	toggle
	text
)

// Value is a setting's new value. The zero Value resets the setting to its
// default, removing the key from config.yaml. The API sends a Number, Word
// or Switch, and the host CLI sends Text, the key's config.yaml spelling.
type Value struct {
	form form
	n    int64
	s    string
	b    bool
}

func Reset() Value            { return Value{} }
func Number(n int64) Value    { return Value{form: number, n: n} }
func Word(s string) Value     { return Value{form: word, s: s} }
func Switch(b bool) Value     { return Value{form: toggle, b: b} }
func Text(s string) Value     { return Value{form: text, s: s} }
func (v Value) IsReset() bool { return v.form == reset }

func (v Value) Number() (n int64, ok bool) { return v.n, v.form == number }
func (v Value) Word() (s string, ok bool)  { return v.s, v.form == word }
func (v Value) Switch() (b, ok bool)       { return v.b, v.form == toggle }
func (v Value) Text() (s string, ok bool)  { return v.s, v.form == text }

// Reason is why a change was refused.
type Reason uint8

const (
	SetByEnvironment Reason = iota + 1
	OutOfRange
	PortUsedByNode
	UnknownSetting
	NotWritable
	// UnusableFolder is a folder the Node can't use for the setting's
	// purpose.
	UnusableFolder
	// Invalid is a config.yaml the change would leave unloadable for a
	// reason no other names.
	Invalid
)

var reasons = [...]string{
	SetByEnvironment: "set_by_environment",
	OutOfRange:       "out_of_range",
	PortUsedByNode:   "port_used_by_node",
	UnknownSetting:   "unknown_setting",
	NotWritable:      "not_writable",
	UnusableFolder:   "unusable_folder",
	Invalid:          "invalid",
}

func (r Reason) String() string { return reasons[r] }

func (r Reason) MarshalText() ([]byte, error) { return []byte(r.String()), nil }

func (r *Reason) UnmarshalText(b []byte) error {
	for i, name := range reasons {
		if name != "" && name == string(b) {
			*r = Reason(i)
			return nil
		}
	}
	return fmt.Errorf("%q is not a refusal reason", b)
}

// Refusal is why a set of changes changed nothing. Sentence is what the
// setting's control shows, and Variable names the variable that locks the
// key.
type Refusal struct {
	Key      string `json:"key,omitempty"`
	Reason   Reason `json:"reason"`
	Sentence string `json:"sentence"`
	Variable string `json:"variable,omitempty"`
}

func (r *Refusal) Error() string {
	if r.Key == "" {
		return r.Sentence
	}
	return r.Key + ": " + r.Sentence
}
