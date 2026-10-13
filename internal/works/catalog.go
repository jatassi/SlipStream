package works

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/storage"
)

// Catalog is the facet through which works reaches a Content Type: the
// shape of its tree, its metadata source, and its own rows.
//
// Search and Fetch are the only calls that do network IO, outside any
// Write and bounded by timings.works.fetch. Their errors wrap ErrNotFound
// or ErrUnavailable; any other error from any method is a bug that answers
// 500 and logs the type's id.
type Catalog interface {
	// Describe is called once at start. Its level count is fixed once a
	// type is released, like its id.
	Describe() Description
	// Search returns at most 25 candidates, best first, each with its
	// Work-tier fields.
	Search(ctx context.Context, query string) ([]Candidate, error)
	// Fetch returns the whole Work as the source has it now.
	Fetch(ctx context.Context, key Key) (Fetched, error)
	// Store writes the type's own rows inside the core's Write, after the
	// core rows exist. It upserts, never deletes, and does no IO. detail is
	// Fetched.Detail, unread by the core.
	Store(ctx context.Context, tx *storage.Tx[storage.Library], at Placed, detail []byte) error
	// Fields reads the type's own rows for one Work: values keyed by node,
	// each of the kind its FieldSpec declares.
	Fields(ctx context.Context, r storage.Reader[storage.Library], w WorkID) (FieldSet, error)
}

type Description struct {
	Name string
	Work Noun
	// Levels are outermost first, empty when Items hang off the Work.
	Levels []Noun
	Item   Noun
	Fields []FieldSpec
}

type Noun struct {
	One  string
	Many string
}

// FieldSpec declares one display field. Key is unique within its tier.
type FieldSpec struct {
	Key   string
	Label string
	Tier  Tier
	Kind  FieldKind
}

type Tier int

const (
	TierWork Tier = 0
	TierItem Tier = -1
)

// TierLevel is the levels at depth, 1 for the outermost.
func TierLevel(depth int) Tier { return Tier(depth) }

func (t Tier) String() string {
	switch t {
	case TierWork:
		return "the Work"
	case TierItem:
		return "the Items"
	}
	return "level " + strconv.Itoa(int(t))
}

type FieldKind uint8

const (
	KindText FieldKind = iota + 1
	KindNumber
	KindDate
	KindDuration
)

var kindNames = map[FieldKind]string{KindText: "text", KindNumber: "number", KindDate: "date", KindDuration: "duration"}

func (k FieldKind) String() string { return kindNames[k] }

// FieldValue is one field of one node. Build it with TextField,
// NumberField, DateField or DurationField; only the value of its Kind is
// read.
type FieldValue struct {
	Key      string
	Kind     FieldKind
	Text     string
	Number   float64
	Date     Date
	Duration time.Duration
}

func TextField(key, v string) FieldValue { return FieldValue{Key: key, Kind: KindText, Text: v} }

func NumberField(key string, v float64) FieldValue {
	return FieldValue{Key: key, Kind: KindNumber, Number: v}
}

func DateField(key string, v Date) FieldValue { return FieldValue{Key: key, Kind: KindDate, Date: v} }

func DurationField(key string, v time.Duration) FieldValue {
	return FieldValue{Key: key, Kind: KindDuration, Duration: v}
}

// Date is a calendar date known to the year, the month or the day: Month
// and Day are 0 when unknown.
type Date struct {
	Year, Month, Day int
}

func ParseDate(s string) (Date, error) {
	parts := strings.Split(s, "-")
	widths := []int{4, 2, 2}
	if len(parts) > len(widths) {
		return Date{}, fmt.Errorf("date %q is not YYYY, YYYY-MM or YYYY-MM-DD", s)
	}
	n := make([]int, len(widths))
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if len(p) != widths[i] || err != nil || v < 0 {
			return Date{}, fmt.Errorf("date %q is not YYYY, YYYY-MM or YYYY-MM-DD", s)
		}
		n[i] = v
	}
	d := Date{Year: n[0], Month: n[1], Day: n[2]}
	if !d.valid(len(parts)) {
		return Date{}, fmt.Errorf("date %q is not a calendar date", s)
	}
	return d, nil
}

func (d Date) valid(parts int) bool {
	if parts == 1 {
		return true
	}
	day := d.Day
	if parts == 2 {
		day = 1
	}
	t := time.Date(d.Year, time.Month(d.Month), day, 0, 0, 0, 0, time.UTC)
	return t.Year() == d.Year && int(t.Month()) == d.Month && t.Day() == day
}

func (d Date) String() string {
	switch {
	case d.Month == 0:
		return fmt.Sprintf("%04d", d.Year)
	case d.Day == 0:
		return fmt.Sprintf("%04d-%02d", d.Year, d.Month)
	}
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day)
}

type Fetched struct {
	Label string
	// SortLabel is empty when it is Label.
	SortLabel string
	// Children are the outermost levels, or the Items when the type declares
	// no levels, in source order.
	Children []Entry
	// Detail is the type's own encoding of the record, handed to Store
	// unread, so fetching and storing can run on different Nodes.
	Detail []byte
}

// Entry is a level or an Item.
type Entry struct {
	Key      Key
	Label    string
	Children []Entry
}

// Candidate is one search result. WorkID is the core's: set when the Work
// is already in the library.
type Candidate struct {
	Key    Key
	Label  string
	Fields []FieldValue
	WorkID WorkID
}

type Placed struct {
	Work   WorkID
	Levels map[Key]LevelID
	Items  map[Key]ItemID
}

type FieldSet struct {
	Work   []FieldValue
	Levels map[LevelID][]FieldValue
	Items  map[ItemID][]FieldValue
}
