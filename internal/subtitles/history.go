package subtitles

import (
	"context"
	"strings"

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
)

// Entries is the rows Subtitles records. Its Sidecar writes and its
// searches add them.
func Entries() []entry.Entry { return []entry.Entry{addedRow, replacedRow, removedRow, markedBadRow} }

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
