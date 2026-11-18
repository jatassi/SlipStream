package subtitles

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// The kinds Subtitles records in History's Subtitles group, each in the
// Write that makes it true.
const (
	SubtitlesAdded       entry.Kind = "subtitles_added"
	SubtitleReplaced     entry.Kind = "subtitle_replaced"
	SubtitlesRemoved     entry.Kind = "subtitles_removed"
	SubtitleMarkedBad    entry.Kind = "subtitle_marked_bad"
	SubtitlesMissing     entry.Kind = "subtitles_missing"
	SubtitleTimingFailed entry.Kind = "subtitle_timing_failed"
	SubtitleProviderOut  entry.Kind = "subtitle_provider_out"
)

// SidecarFacts is what a Subtitles added, replaced or removed entry tells:
// the Work, the file beside which the subtitles sit, and each subtitle.
type SidecarFacts struct {
	Work      entry.Work     `json:"work"`
	File      string         `json:"file"`
	Subtitles []SubtitleFact `json:"subtitles"`
	// Why is a removal's reason: "retime_failed".
	Why string `json:"why,omitempty"`
}

// SubtitleFact is one subtitle as History names it.
type SubtitleFact struct {
	Language string `json:"language"`
	Forced   bool   `json:"forced,omitempty"`
	HI       bool   `json:"hi,omitempty"`
	Source   Source `json:"source"`
	// Provider is the Subtitle Provider's label, "" unless Source is
	// provider.
	Provider string `json:"provider,omitempty"`
}

// MarkedBadFacts is what a Subtitle marked bad entry tells.
type MarkedBadFacts struct {
	Work     entry.Work   `json:"work"`
	File     string       `json:"file"`
	Subtitle SubtitleFact `json:"subtitle"`
	By       entry.Actor  `json:"by"`
}

// SearchFacts is what a Subtitles missing or a No subtitle matched the
// timing entry tells: the file's first search after its import.
type SearchFacts struct {
	Work     entry.Work `json:"work"`
	File     string     `json:"file"`
	Language string     `json:"language"`
	Forced   bool       `json:"forced,omitempty"`
	// Found is how many subtitles the search found.
	Found int `json:"found"`
}

// ProviderOutFacts is what a Subtitle Provider out of downloads entry
// tells.
type ProviderOutFacts struct {
	Provider string    `json:"provider"`
	Until    time.Time `json:"until"`
}

func (f SearchFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

func (ProviderOutFacts) About() entry.About { return entry.About{} }

func (f SidecarFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

func (f MarkedBadFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

var (
	addedRow = entry.Row[SidecarFacts]{
		Spec: entry.Spec{Kind: SubtitlesAdded, Label: "Subtitles added", Group: entry.Subtitles},
		Render: func(f SidecarFacts, _ entry.Context) entry.Message {
			return sidecarMessage("Subtitles added for ", f, entry.Good)
		},
	}
	replacedRow = entry.Row[SidecarFacts]{
		Spec: entry.Spec{Kind: SubtitleReplaced, Label: "Subtitle replaced", Group: entry.Subtitles},
		Render: func(f SidecarFacts, _ entry.Context) entry.Message {
			return sidecarMessage("Subtitle replaced for ", f, entry.Good)
		},
	}
	removedRow = entry.Row[SidecarFacts]{
		Spec: entry.Spec{Kind: SubtitlesRemoved, Label: "Subtitles removed", Group: entry.Subtitles},
		Render: func(f SidecarFacts, _ entry.Context) entry.Message {
			return sidecarMessage("Subtitles removed for ", f, entry.Info)
		},
	}
	markedBadRow = entry.Row[MarkedBadFacts]{
		Spec: entry.Spec{Kind: SubtitleMarkedBad, Label: "Subtitle marked bad", Group: entry.Subtitles},
		Render: func(f MarkedBadFacts, _ entry.Context) entry.Message {
			return entry.Message{
				Title: "Subtitle marked bad for " + f.Work.Label, Lines: []string{f.Subtitle.words(), f.By.Subject() + " marked it bad."},
				Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: entry.Warn, Body: workBody(f.Work),
			}
		},
	}
	missingRow = entry.Row[SearchFacts]{
		Spec: entry.Spec{Kind: SubtitlesMissing, Label: "Subtitles missing", Group: entry.Subtitles},
		Render: func(f SearchFacts, _ entry.Context) entry.Message {
			return entry.Message{
				Title: "Subtitles missing for " + f.Work.Label, Lines: []string{"No " + f.subtitle() + " was found for " + f.File + ". SlipStream searches again later."},
				Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: entry.Warn, Body: workBody(f.Work),
			}
		},
	}
	timingFailedRow = entry.Row[SearchFacts]{
		Spec: entry.Spec{Kind: SubtitleTimingFailed, Label: "No subtitle matched the timing", Group: entry.Subtitles, BellByDefault: true},
		Render: func(f SearchFacts, _ entry.Context) entry.Message {
			line := fmt.Sprintf("%d %ss were found for %s, and none matched its timing.", f.Found, f.subtitle(), f.File)
			if f.Found == 1 {
				line = "1 " + f.subtitle() + " was found for " + f.File + ", and it didn't match its timing."
			}
			return entry.Message{
				Title: "No subtitle matched the timing for " + f.Work.Label,
				Lines: []string{line + " Upload one, or SlipStream searches again later."},
				Link:  entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: entry.Warn, Body: workBody(f.Work),
			}
		},
	}
	providerOutRow = entry.Row[ProviderOutFacts]{
		Spec: entry.Spec{Kind: SubtitleProviderOut, Label: "Subtitle Provider out of downloads", Group: entry.Subtitles},
		Render: func(f ProviderOutFacts, c entry.Context) entry.Message {
			return entry.Message{
				Title: f.Provider + " is out of downloads",
				Lines: []string{"SlipStream searches " + f.Provider + " again on " + c.Day(f.Until) + " at " + c.Clock(f.Until) + "."},
				Tone:  entry.Warn,
			}
		},
	}
)

// Entries is the rows Subtitles records. Its Sidecar writes and its
// searches add them.
func Entries() []entry.Entry {
	return []entry.Entry{addedRow, replacedRow, removedRow, markedBadRow, missingRow, timingFailedRow, providerOutRow}
}

// subtitle is the searched subtitle in words: "English subtitle", "forced
// English subtitle".
func (f SearchFacts) subtitle() string {
	if f.Forced {
		return "forced " + f.Language + " subtitle"
	}
	return f.Language + " subtitle"
}

var whyWords = map[string]string{
	"retime_failed": "Its timing didn't match the file's audio.",
}

func sidecarMessage(title string, f SidecarFacts, tone entry.Tone) entry.Message {
	lines := make([]string, 0, len(f.Subtitles)+1)
	for _, s := range f.Subtitles {
		lines = append(lines, s.words())
	}
	if w, ok := whyWords[f.Why]; ok {
		lines = append(lines, w)
	}
	return entry.Message{Title: title + f.Work.Label, Lines: lines, Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: tone, Body: workBody(f.Work)}
}

// words is the subtitle as a sentence: "English for the hearing impaired
// from SubDL."
func (s SubtitleFact) words() string {
	var b strings.Builder
	b.WriteString(s.Language)
	if s.Forced {
		b.WriteString(" forced")
	}
	if s.HI {
		b.WriteString(" for the hearing impaired")
	}
	switch s.Source {
	case FromProvider:
		// A Sidecar a restore took back from the ledger has no Offer.
		if s.Provider == "" {
			b.WriteString(", downloaded")
		} else {
			b.WriteString(" from " + s.Provider)
		}
	case FromUpload:
		b.WriteString(", uploaded")
	case Found:
		b.WriteString(", found beside the file")
	case FromSibling:
		b.WriteString(" from another file of the same title")
	}
	b.WriteString(".")
	return b.String()
}

func workBody(w entry.Work) entry.Body {
	return entry.Body{Work: &entry.BodyWork{ID: w.ID.String(), Title: w.Label, ContentType: string(w.Type)}}
}

// factOf is sc as History names it.
func (s *Subtitles) factOf(sc Sidecar) SubtitleFact {
	f := SubtitleFact{Language: sc.Tail.Language.Name(), Forced: sc.Tail.Forced, HI: sc.Tail.HI, Source: sc.Source}
	if sc.Offer != nil && sc.Source == FromProvider {
		f.Provider = s.spec(sc.Offer.Provider).Label
	}
	return f
}

// recordSidecars records one row of kind for sub's Work in tx: its
// subtitles scs, and why for a removal.
func (s *Subtitles) recordSidecars(ctx context.Context, tx *storage.Tx[storage.Library], row entry.Row[SidecarFacts], sub files.Subject, scs []Sidecar, why string) error {
	w, err := entry.WorkIn(ctx, tx, s.cfg.Works, sub.Work)
	if err != nil {
		return err
	}
	f := SidecarFacts{Work: w, File: sub.Path.String(), Why: why}
	for _, sc := range scs {
		f.Subtitles = append(f.Subtitles, s.factOf(sc))
	}
	return row.Record(ctx, tx, s.cfg.History, f)
}
