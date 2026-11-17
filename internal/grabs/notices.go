package grabs

import (
	"context"
	"strconv"

	"github.com/jatassi/slipstream/internal/files"
	"github.com/jatassi/slipstream/internal/history/entry"
	"github.com/jatassi/slipstream/internal/release"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/targets"
	"github.com/jatassi/slipstream/internal/works"
)

// GrabbedFacts is what a Grabbed message tells: the Work, the Release's
// facts, its Indexer, what found it and its score for each Target it fills.
type GrabbedFacts struct {
	Work    entry.Work    `json:"work"`
	Release release.Facts `json:"release"`
	Indexer string        `json:"indexer"`
	Source  Source        `json:"source"`
	FoundBy FoundBy       `json:"foundBy"`
	By      entry.Actor   `json:"by"`
	Scores  []ScoreAt     `json:"scores"`
}

// ScoreAt is a Release's Custom Format score at one Target, by its name.
type ScoreAt struct {
	Target string `json:"target"`
	Score  int    `json:"score"`
}

// GrabFailedFacts is what a Grab failed message tells.
type GrabFailedFacts struct {
	Work    entry.Work  `json:"work"`
	Code    Code        `json:"code"`
	Stage   Stage       `json:"stage"`
	Release string      `json:"release"`
	Source  Source      `json:"source"`
	By      entry.Actor `json:"by"`
}

func (f GrabbedFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

func (f GrabFailedFacts) About() entry.About {
	return entry.About{Type: f.Work.Type, Works: []works.WorkID{f.Work.ID}}
}

var (
	grabbedRow = entry.Row[GrabbedFacts]{
		Spec:   entry.Spec{Kind: "grabbed", Label: "Grabbed", Group: entry.Grabs, BellByDefault: true, Tally: works.Noun{One: "Grab", Many: "Grabs"}},
		Render: renderGrabbed,
	}
	grabFailedRow = entry.Row[GrabFailedFacts]{
		Spec:   entry.Spec{Kind: "grab_failed", Label: "Grab failed", Group: entry.Grabs, OnByDefault: true, BellByDefault: true},
		Render: renderGrabFailed,
	}
)

// Entries is the kinds grabs records.
func Entries() []entry.Entry { return []entry.Entry{grabbedRow, grabFailedRow} }

func renderGrabbed(f GrabbedFacts, _ entry.Context) entry.Message {
	lines := []string{release.Describe(release.Read{Facts: f.Release}), FoundByWords(f.FoundBy, f.Indexer, f.By)}
	if len(f.Scores) > 0 {
		lines = append(lines, scoresWords(f.Scores))
	}
	return entry.Message{
		Title: "Grabbed " + f.Work.Label, Lines: lines, Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster,
		Tone: entry.Info, Body: workBody(f.Work),
	}
}

func renderGrabFailed(f GrabFailedFacts, _ entry.Context) entry.Message {
	w := WordsOf(f.Code)
	lines := []string{w.Cause}
	if w.Recovery != "" {
		lines = append(lines, w.Recovery)
	}
	return entry.Message{
		Title: "The Grab for " + f.Work.Label + " failed", Lines: lines, Link: entry.WorkLink(f.Work.ID), Poster: f.Work.Poster,
		Tone: entry.Fail, Body: workBody(f.Work),
	}
}

func workBody(w entry.Work) entry.Body {
	return entry.Body{Work: &entry.BodyWork{ID: w.ID.String(), Title: w.Label, ContentType: string(w.Type)}}
}

// FoundByWords is the sentence for what found a Grab's Release on
// indexer, worded as web/src/grabs/search-words.ts words FoundBy.
func FoundByWords(f FoundBy, indexer string, by entry.Actor) string {
	switch f {
	case FoundBySearch:
		return "Automatic search found it on " + indexer + "."
	case FoundByBacklog:
		return "Backlog search found it on " + indexer + "."
	case FoundByRSS:
		return "RSS sync found it on " + indexer + "."
	case FoundByFailedGrab:
		return "A search after a failed Grab found it on " + indexer + "."
	case FoundByManual:
		return by.Subject() + " picked it in Manual search from " + indexer + "."
	}
	panic("grabs: no words for found_by " + string(f))
}

// scoresWords is "It scores 1250 for the 1080p Target and 900 for the
// 720p Target."
func scoresWords(scores []ScoreAt) string {
	parts := make([]string, len(scores))
	for i, s := range scores {
		parts[i] = strconv.Itoa(s.Score) + " for the " + s.Target + " Target"
	}
	return "It scores " + joinWords(parts) + "."
}

// scoresOf is a's score at each distinct Target of cells, in cells' order.
func scoresOf(a *assessed, cells []files.Cell) []ScoreAt {
	var out []ScoreAt
	seen := map[targets.TargetID]bool{}
	for _, c := range cells {
		if seen[c.Target.ID] {
			continue
		}
		seen[c.Target.ID] = true
		out = append(out, ScoreAt{Target: c.Target.Name, Score: c.Target.Score(&a.scored)})
	}
	return out
}

// noticeActor is a as the actor a message names.
func noticeActor(a *Actor) entry.Actor {
	if a == nil {
		return entry.Actor{}
	}
	return entry.Actor{Name: a.Name, APIKey: a.Account == nil}
}

// announceGrabbed records Grabbed for g, a Grab insertGrab just made.
func (gr *Grabs) announceGrabbed(ctx context.Context, tx *storage.Tx[file], g Grab, n newGrab, indexer string) error {
	w, err := entry.WorkIn(ctx, tx, gr.cfg.Works, g.WorkID)
	if err != nil {
		return err
	}
	return grabbedRow.Record(ctx, tx, gr.cfg.History, GrabbedFacts{
		Work: w, Release: n.facts, Indexer: indexer, Source: g.Source, FoundBy: g.FoundBy, By: noticeActor(g.By), Scores: n.scores,
	})
}

// announceFailed records Grab failed when Activity would list g, failed
// with code, as needing the Admin: NeedsYou, not every cell searched
// again, and not declined. g's row is already failed in tx and
// recordFailures has written its stops, so grabsCellsReadFrom reads what
// Activity will.
func (gr *Grabs) announceFailed(ctx context.Context, tx *storage.Tx[file], g Grab, code Code) error {
	fg := g
	fg.Failed = &Failure{Stage: g.Stage, Code: code}
	if !fg.NeedsYou() || code == Declined {
		return nil
	}
	_, again, err := grabsCellsReadFrom(ctx, tx, []Grab{fg})
	if err != nil || again[fg.ID] {
		return err
	}
	w, err := entry.WorkIn(ctx, tx, gr.cfg.Works, g.WorkID)
	if err != nil {
		return err
	}
	return grabFailedRow.Record(ctx, tx, gr.cfg.History, GrabFailedFacts{
		Work: w, Code: code, Stage: g.Stage, Release: g.Release.Title, Source: g.Source, By: noticeActor(g.By),
	})
}
