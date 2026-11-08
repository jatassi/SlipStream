package scanning

// Reason is why a found file waits in Pending imports. It is stored as
// data and worded when read, so copy changes need no migration and a
// renamed root or Node reads its new name.
type Reason struct {
	Code  ReasonCode `json:"code"`
	Title string     `json:"title,omitempty"` // the hint's title, or a Work's label
	Year  int        `json:"year,omitempty"`
	Count int        `json:"count,omitempty"` // candidates, for Several
	// Names is the Item the name names, such as "S03E14", for NoSuchItem.
	Names string `json:"names,omitempty"`
	// Root is the other root's id, for Elsewhere.
	Root string `json:"root,omitempty"`
	Line string `json:"line,omitempty"` // ffprobe's first error line, for Unreadable
}

type ReasonCode string

const (
	Several      ReasonCode = "several"       // "Several movies match Halloween (no year in the name)."
	NoMatch      ReasonCode = "no_match"      // "No movie matches Brazil (1985)."
	NoYear       ReasonCode = "no_year"       // "One movie matches Halloween, but the name has no year."
	Elsewhere    ReasonCode = "elsewhere"     // "Heat (1995) is in the Movies library root at /mnt/4k on Media server."
	Unlisted     ReasonCode = "unlisted"      // "The metadata source no longer lists Heat (1995)."
	NoNumber     ReasonCode = "no_number"     // "The file name has no episode number."
	NoSuchItem   ReasonCode = "no_such_item"  // "Severance (2022) has no episode S03E14."
	SeveralItems ReasonCode = "several_items" // "The file holds several episodes."
	Unreadable   ReasonCode = "unreadable"    // "ffprobe can't read this file. Replace it, then scan again."
	Left         ReasonCode = "left"          // "Its Work was deleted and its files were left on disk."
	NoRule       ReasonCode = "no_rule"       // "SlipStream can't tell which Item this file holds."
)
