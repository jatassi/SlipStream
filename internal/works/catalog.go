package works

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/metadata/artwork"
	"github.com/jatassi/slipstream/internal/storage"
)

// Catalog is the facet through which works reaches a Content Type's
// library half: the shape of its tree and its own rows. It does no network
// IO; Source does.
//
// Any error from any method is a bug that answers 500 and logs the type's
// id.
type Catalog interface {
	// Describe is called once at start. Its level count is fixed once a
	// type is released, like its id.
	Describe() Description
	// Store writes the type's own rows inside the core's Write, after the
	// core rows exist, and does no IO. detail is Fetched.Detail, unread by
	// the core. changed is false when every row already held what detail
	// says.
	Store(ctx context.Context, tx *storage.Tx[storage.Library], at Placed, detail []byte) (changed bool, err error)
	// Fields reads the type's own rows for one Work: values keyed by node,
	// each of the kind its FieldSpec declares.
	Fields(ctx context.Context, r storage.Reader[storage.Library], w WorkID) (FieldSet, error)
}

// Source is the network: a Content Type's metadata source, reached through
// a Node holding the Metadata Role. Its calls run outside any Write,
// bounded by timings.works.fetch. Errors wrap ErrNotFound, ErrRefused,
// ErrUnavailable or ErrInvalid, inside a *SourceError when the source said
// more.
type Source interface {
	// Search returns at most 25 candidates, best first, each with its
	// Work-tier fields.
	Search(ctx context.Context, t TypeID, query string) ([]Candidate, error)
	// Fetch returns the whole Work as the source has it now.
	Fetch(ctx context.Context, t TypeID, key Key, f Fetching) (Fetched, error)
}

// Purpose is why a Work is fetched, which decides the cache and the
// ratings providers.
type Purpose uint8

const (
	// ForPreview reads no ratings provider and may answer from the cache.
	ForPreview Purpose = iota + 1
	// ForAdd may answer from the cache, so it reuses a preview's answer.
	ForAdd
	// ForRefresh bypasses the cache and writes its answer back.
	ForRefresh
)

type Fetching struct {
	Purpose Purpose
}

type Description struct {
	Name string
	Icon Icon
	Work Noun
	// Levels are outermost first, empty when Items hang off the Work.
	Levels []Noun
	Item   Noun
	Fields []FieldSpec
	// Artwork is the kinds of image the type's Works and Items carry.
	Artwork []artwork.Kind
}

// Icon names the glyph that stands for a Content Type in navigation.
type Icon uint8

const (
	IconFilm Icon = iota + 1
	IconTV
	IconNote
	IconOpenBook
	IconHeadphones
	IconPanels
	IconImage
	IconShapes
)

var iconNames = map[Icon]string{
	IconFilm: "film", IconTV: "tv", IconNote: "note", IconOpenBook: "open-book",
	IconHeadphones: "headphones", IconPanels: "panels", IconImage: "image", IconShapes: "shapes",
}

func (i Icon) String() string { return iconNames[i] }

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
	// KindTexts is an ordered list of strings, such as genres.
	KindTexts
	// KindPeople is an ordered list of credits.
	KindPeople
	// KindLink is a label and a URL.
	KindLink
)

var kindNames = map[FieldKind]string{
	KindText: "text", KindNumber: "number", KindDate: "date", KindDuration: "duration",
	KindTexts: "texts", KindPeople: "people", KindLink: "link",
}

func (k FieldKind) String() string { return kindNames[k] }

// FieldValue is one field of one node. Build it with the constructor of
// its kind, such as TextField; only the value of its Kind is read.
type FieldValue struct {
	Key      string
	Kind     FieldKind
	Text     string
	Number   float64
	Date     Date
	Duration time.Duration
	Texts    []string
	People   []Person
	Link     Link
}

// Person is one credit. Its photo is a reference, never a URL, so no type
// holds the key that signs artwork URLs.
type Person struct {
	Name       string
	Role       string
	Department string
	Photo      artwork.Ref
}

type Link struct {
	Label string
	URL   string
}

func TextField(key, v string) FieldValue { return FieldValue{Key: key, Kind: KindText, Text: v} }

func NumberField(key string, v float64) FieldValue {
	return FieldValue{Key: key, Kind: KindNumber, Number: v}
}

func DateField(key string, v Date) FieldValue { return FieldValue{Key: key, Kind: KindDate, Date: v} }

func DurationField(key string, v time.Duration) FieldValue {
	return FieldValue{Key: key, Kind: KindDuration, Duration: v}
}

func TextsField(key string, v []string) FieldValue {
	return FieldValue{Key: key, Kind: KindTexts, Texts: v}
}

func PeopleField(key string, v []Person) FieldValue {
	return FieldValue{Key: key, Kind: KindPeople, People: v}
}

func LinkField(key string, v Link) FieldValue { return FieldValue{Key: key, Kind: KindLink, Link: v} }

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
	// Ratings are the source's ratings now. A source absent here keeps its
	// stored row, so an outage never wipes a rating.
	Ratings []Rating
	// RatingsRead is true when the ratings providers answered, with values
	// or with none.
	RatingsRead bool
	// Artwork is one reference per kind the source has now. A kind absent
	// here keeps its stored row.
	Artwork []Art
}

// Rating is one source's rating of the Work, or of one of its Items.
type Rating struct {
	// Item is empty for the Work's own rating.
	Item   Key
	Source RatingSource
	Value  float64
	// Votes is nil when the source counts none.
	Votes *int64
}

// RatingSource is a ratings source the core knows. Its scale and label
// come from RatingSources, so no Rating carries a scale that disagrees.
type RatingSource string

type RatingScale struct {
	Label string
	Scale float64
}

var RatingSources = map[RatingSource]RatingScale{
	"tmdb":           {Label: "TMDB", Scale: 10},
	"imdb":           {Label: "IMDb", Scale: 10},
	"rottentomatoes": {Label: "Rotten Tomatoes critics", Scale: 100},
	"metacritic":     {Label: "Metacritic", Scale: 100},
}

// Art is the source's image of one kind for the Work, or one of its Items.
type Art struct {
	// Item is empty for the Work's own image.
	Item Key
	// Kind is the key of an artwork.Kind the Description declares at the
	// Item's or the Work's tier.
	Kind string
	Ref  artwork.Ref
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
	Key   Key
	Label string
	// Original is the label in its original language, empty when it is
	// Label.
	Original string
	// Overview is one line.
	Overview string
	Poster   artwork.Ref
	Fields   []FieldValue
	WorkID   WorkID `json:"-"`
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
