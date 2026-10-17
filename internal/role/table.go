package role

import (
	"fmt"
	"math/bits"
	"slices"
)

// Work is what a holder keeps that must finish before it lets the Role go.
type Work uint8

const (
	None Work = iota
	Portable
	Tied
	Streams
)

// Lapse is what a Role's work does once all its holders lapse.
type Lapse uint8

const (
	Wait Lapse = iota
	ToPeer
	ToController
)

// Probe is a check a Node runs on itself before it is given a Role.
type Probe string

const (
	EngineStarts Probe = "engine-starts"
	DataWritable Probe = "data-writable"
	ListenerUp   Probe = "listener-up"
)

// Fit is a probe's verdict on the Node that ran it.
type Fit uint8

const (
	Fits Fit = iota + 1
	Weak
	Fails
)

// Rule is a Role's guardrails and what its work does offline.
type Rule struct {
	// Always: the Role always has a holder, and unticking the last one hands
	// it to the Controller.
	Always bool
	// Max is the most assigned holders, 0 for any number.
	Max int
	// Controller: the Controller holds the Role whatever the Admin ticks.
	Controller bool
	Work       Work
	Lapse      Lapse
	Probe      Probe
	// Reach: the Role is tested from the Node against its owner's target
	// before it is saved.
	Reach bool
}

type row struct {
	name    Name
	label   string
	group   Group
	purpose string
	rule    Rule
}

var table = [...]row{
	{Searching, "Searching", Acquire, "searches", Rule{Always: true, Work: Portable, Lapse: ToController}},
	{Downloading, "Downloading", Acquire, "downloads", Rule{Always: true, Work: Tied, Lapse: Wait, Probe: EngineStarts}},
	{Storage, "Storage", Store, "stores the library", Rule{Always: true, Work: Tied, Lapse: Wait, Probe: DataWritable}},
	{Converting, "Converting", Process, "converts", Rule{Work: Portable, Lapse: ToPeer}},
	{Subtitles, "Subtitles", Process, "fetches subtitles", Rule{Work: Portable, Lapse: ToPeer}},
	{Metadata, "Metadata", Connect, "reads metadata", Rule{Always: true, Work: None, Lapse: ToController}},
	{MediaServerLink, "Media Server link", Connect, "talks to Media Servers", Rule{Work: None, Lapse: ToController, Reach: true}},
	{Notifying, "Notifying", Connect, "sends mail", Rule{Always: true, Work: Portable, Lapse: ToController, Reach: true}},
	{Relaying, "Relaying", Reach, "relays", Rule{Max: 1, Work: Streams, Lapse: Wait}},
	{Serving, "Serving", Reach, "serves SlipStream", Rule{Controller: true, Work: None, Lapse: Wait, Probe: ListenerUp}},
}

func lookup(n Name) (int, bool) {
	for i := range table {
		if table[i].name == n {
			return i, true
		}
	}
	return 0, false
}

func rowOf(n Name) row {
	i, ok := lookup(n)
	if !ok {
		panic(fmt.Sprintf("role: %q is not a Role", string(n)))
	}
	return table[i]
}

// Names is every Role, in the glossary's order.
func Names() []Name {
	out := make([]Name, len(table))
	for i := range table {
		out[i] = table[i].name
	}
	return out
}

func ParseName(s string) (Name, error) {
	if _, ok := lookup(Name(s)); ok {
		return Name(s), nil
	}
	return "", fmt.Errorf("%q is not a Role", s)
}

// Label is the Role's name as the UI writes it, "" for a name outside the table.
func (n Name) Label() string {
	if i, ok := lookup(n); ok {
		return table[i].label
	}
	return ""
}

func (n Name) Group() Group { return rowOf(n).group }

func (n Name) Rule() Rule { return rowOf(n).rule }

// Purpose is what a holder of the Role does, as a verb phrase: "downloads".
func (n Name) Purpose() string { return rowOf(n).purpose }

// Probes is every Probe the table names, once each, in table order.
func Probes() []Probe {
	var out []Probe
	for i := range table {
		p := table[i].rule.Probe
		if p != "" && !slices.Contains(out, p) {
			out = append(out, p)
		}
	}
	return out
}

// Set is a set of Roles from the table. A Role outside it, such as the
// test Role "probe", can never be a member.
type Set uint16

// Of is the Set of names. It panics on a name outside the table.
func Of(names ...Name) Set {
	var s Set
	for _, n := range names {
		s = s.With(n)
	}
	return s
}

func bit(n Name) Set {
	i, ok := lookup(n)
	if !ok {
		panic(fmt.Sprintf("role: %q is not a Role", string(n)))
	}
	return 1 << i
}

func (s Set) Has(n Name) bool {
	i, ok := lookup(n)
	return ok && s&(1<<i) != 0
}

func (s Set) With(n Name) Set    { return s | bit(n) }
func (s Set) Without(n Name) Set { return s &^ bit(n) }

// Names is the Set's Roles in table order.
func (s Set) Names() []Name {
	var out []Name
	for i := range table {
		if s&(1<<i) != 0 {
			out = append(out, table[i].name)
		}
	}
	return out
}

func (s Set) Len() int { return bits.OnesCount16(uint16(s)) }
