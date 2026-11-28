package conversions

import (
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/works"
)

// FailedFacts is a Conversion that failed: its Work, the Target it made a
// file for, the Nodes its counted attempts failed on, its fault, and the
// fault's sentence.
type FailedFacts struct {
	Work   entry.Work `json:"work"`
	Target string     `json:"target"`
	Nodes  []string   `json:"nodes"`
	Code   Code       `json:"code"`
	Class  Class      `json:"class"`
	Words  string     `json:"words"`
}

func (f FailedFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

// NoNodeFacts is a cause that opened: how many Works of a Content Type
// wait on it, and its sentence.
type NoNodeFacts struct {
	Type  works.TypeID `json:"type"`
	Noun  works.Noun   `json:"noun"`
	Works int          `json:"works"`
	Words string       `json:"words"`
}

func (f NoNodeFacts) About() entry.About { return entry.About{Type: f.Type} }

var (
	ConversionFailed = entry.Row[FailedFacts]{
		Spec:   entry.Spec{Kind: "conversion_failed", Label: "Conversion failed", Group: entry.Conversions, OnByDefault: true, BellByDefault: true},
		Render: renderFailed,
	}
	NoNodeCanConvert = entry.Row[NoNodeFacts]{
		Spec:   entry.Spec{Kind: "no_node_can_convert", Label: "No Node can convert this", Group: entry.Conversions, OnByDefault: true, BellByDefault: true},
		Render: renderNoNode,
	}
)

// Entries is the kinds conversions records.
func Entries() []entry.Entry { return []entry.Entry{ConversionFailed, NoNodeCanConvert} }

func renderFailed(f FailedFacts, _ entry.Context) entry.Message {
	return entry.Message{
		Title: "Couldn't convert " + f.Work.Label + " to " + f.Target, Lines: []string{f.Words},
		Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: entry.Fail,
	}
}

func renderNoNode(f NoNodeFacts, _ entry.Context) entry.Message {
	return entry.Message{
		Title: "No Node can convert " + entry.Count(f.Works, f.Noun), Lines: []string{f.Words},
		Link: entry.Link{Path: "/activity/conversions"}, Tone: entry.Warn,
	}
}
