package release

// Read is what one name says: the Content Type's part and the core's.
type Read struct {
	// Release is the name as the group, the stream facts, the id tags and
	// a Release title condition read it: Radarr's SimpleReleaseTitle, with
	// the movie's title masked, or Sonarr's releaseTitle.
	Release string
	Name    Name
	Facts   Facts
}

// Name is the Content Type's reading of a name: every title it gives, as
// in "A AKA B", its year, its numbering and what follows them.
type Name struct {
	Titles []string
	Year   Fact[int]
	Marks  []Mark
	Span   Span
	// Tail is the name as the type's upstream reads languages, the cut
	// and the edition text from, with its title out of the way: Sonarr's
	// ReleaseTokens, Radarr's Masked. Empty reads the whole
	// name.
	Tail string
	// Masked is the name with its title replaced, as Radarr reads the
	// group from it. Empty reads the whole name, as Sonarr does.
	Masked string
}

// Mark is one piece of a type's numbering. Words is how the Admin reads
// it, such as S01E02 or Season 2. Key is the type's own encoding, which
// only that type's Covers reads.
type Mark struct {
	Words string
	Key   string
}

// Span is Sonarr's release type, for any type: one Item, several Items of
// one level, a whole level, several levels.
type Span uint8

const (
	SpanUnknown Span = iota
	One
	Several
	Level
	Levels
)

var spanKeys = []string{SpanUnknown: "", One: "one", Several: "several", Level: "level", Levels: "levels"}

func (s Span) String() string               { return spanKeys[s] }
func (s Span) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Span) UnmarshalText(b []byte) error {
	if len(b) == 0 {
		*s = SpanUnknown
		return nil
	}
	return parseKey(s, "a span", spanKeys, b)
}

// Names is the Content Type's part of reading a name. contenttype.Library
// embeds it. A type with no numbering answers the title and year only.
type Names interface {
	// ReadName reads p's titles, year and numbering. A name it can't read
	// answers the zero Name, and every core fact is still read.
	ReadName(p Prepared) Name
	// CleanTitle is the type's upstream title rule, which title matching
	// compares under.
	CleanTitle(title string) string
}

// TitleOnly is the Names of a type with no numbering and no title rule of
// its own: the whole name is its one title.
type TitleOnly struct{}

func (TitleOnly) ReadName(p Prepared) Name { return Name{Titles: []string{p.Title.String()}} }

func (TitleOnly) CleanTitle(title string) string { return title }
