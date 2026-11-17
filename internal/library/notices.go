package library

import (
	"context"
	"errors"
	"strings"

	"github.com/jatassi/slipstream/internal/identity"
	"github.com/jatassi/slipstream/internal/notifications/notice"
	"github.com/jatassi/slipstream/internal/roots"
	"github.com/jatassi/slipstream/internal/storage"
	"github.com/jatassi/slipstream/internal/works"
)

// AddedFacts is what a Works added message tells: the Work, the root it
// went into, empty for a type with no roots, and who added it.
type AddedFacts struct {
	Work notice.Work  `json:"work"`
	Root string       `json:"root"`
	By   notice.Actor `json:"by"`
}

// DeletedFacts is what a Works deleted message tells: who deleted them,
// and each Content Type's share.
type DeletedFacts struct {
	By    notice.Actor  `json:"by"`
	Types []DeletedType `json:"types"`
}

// DeletedType is one Content Type's share of a delete: its count, its
// first three labels, and the Work itself when it is the one Work deleted.
type DeletedType struct {
	Nouns notice.Nouns `json:"nouns"`
	Count int          `json:"count"`
	First []string     `json:"first"`
	Work  *notice.Work `json:"work,omitempty"`
}

var (
	addedRow = notice.Row[AddedFacts]{
		Spec:   notice.Spec{Kind: "works_added", Label: "Works added", Group: notice.Library, BellByDefault: true},
		Render: renderAdded,
	}
	deletedRow = notice.Row[DeletedFacts]{
		Spec:   notice.Spec{Kind: "works_deleted", Label: "Works deleted", Group: notice.Library, BellByDefault: true},
		Render: renderDeleted,
	}
)

// Notices is the events library records.
func Notices() []notice.Entry { return []notice.Entry{addedRow, deletedRow} }

func actorOf(by identity.Admin) notice.Actor {
	a, ok := by.Account()
	if !ok {
		return notice.Actor{APIKey: true}
	}
	return notice.Actor{Name: a.Profile().DisplayName}
}

// announceAdded records Works added for w, which the add flow just made,
// in the add's Write.
func (l *Library) announceAdded(ctx context.Context, tx *storage.Tx[storage.Library], by identity.Admin, w works.WorkID) error {
	work, err := notice.WorkIn(ctx, tx, l.cfg.Works, w)
	if err != nil {
		return err
	}
	f := AddedFacts{Work: work, By: actorOf(by)}
	root, err := roots.OfWork(ctx, tx, w)
	switch {
	case errors.Is(err, roots.ErrNoRoot):
	case err != nil:
		return err
	default:
		words, err := l.cfg.Roots.WordsIn(ctx, tx)
		if err != nil {
			return err
		}
		f.Root = words[root.ID]
	}
	return addedRow.Record(ctx, tx, l.cfg.Notices, f)
}

// deletedFacts reads, in the delete's Write and before any works row goes,
// the count and first three labels per Content Type of removed, and the
// Work itself when exactly one goes.
func (l *Library) deletedFacts(ctx context.Context, tx *storage.Tx[storage.Library], by identity.Admin, removed []works.WorkID) (DeletedFacts, error) {
	tallies, err := works.TallyOf(ctx, tx, removed, 3)
	if err != nil {
		return DeletedFacts{}, err
	}
	f := DeletedFacts{By: actorOf(by)}
	for _, t := range tallies {
		f.Types = append(f.Types, DeletedType{Nouns: notice.NounsOf(l.cfg.Works, t.Type), Count: t.Count, First: t.First})
	}
	if len(removed) == 1 {
		w, err := notice.WorkIn(ctx, tx, l.cfg.Works, removed[0])
		if err != nil {
			return DeletedFacts{}, err
		}
		f.Types[0].Work = &w
	}
	return f, nil
}

func renderAdded(f AddedFacts, _ notice.Context) notice.Message {
	title := "Added " + f.Work.Label
	if f.Root != "" {
		title += " to the " + f.Root
	}
	return notice.Message{
		Title: title, Lines: []string{f.By.Subject() + " added it."}, Link: notice.WorkLink(f.Work.ID), Poster: f.Work.Poster, Tone: notice.Info,
		Body: notice.Body{Work: &notice.BodyWork{ID: f.Work.ID.String(), Title: f.Work.Label, ContentType: string(f.Work.Type)}},
	}
}

func renderDeleted(f DeletedFacts, _ notice.Context) notice.Message {
	m := notice.Message{Link: notice.Link{Path: "/library"}, Tone: notice.Info}
	if len(f.Types) == 1 {
		t := f.Types[0]
		if t.Work != nil {
			m.Title, m.Lines, m.Poster = "Deleted "+t.Work.Label, []string{f.By.Subject() + " deleted it."}, t.Work.Poster
			return m
		}
		m.Title, m.Lines = "Deleted "+notice.Count(t.Count, lower(t.Nouns.Work)), notice.Titles(t.First, t.Count, "deleted")
		return m
	}
	counts := make([]string, len(f.Types))
	for i, t := range f.Types {
		noun := lower(t.Nouns.Work)
		counts[i] = notice.Count(t.Count, noun)
		line := joinWords(t.First)
		switch rest := t.Count - len(t.First); {
		case rest == 1:
			line = strings.Join(t.First, ", ") + " and 1 more " + noun.One
		case rest > 1:
			line = strings.Join(t.First, ", ") + " and " + notice.Grouped(rest) + " more " + noun.Many
		}
		m.Lines = append(m.Lines, line+".")
	}
	m.Title = "Deleted " + joinWords(counts)
	m.Lines = append(m.Lines[:min(len(m.Lines), 3)], f.By.Subject()+" deleted them.")
	return m
}

func lower(n works.Noun) works.Noun {
	return works.Noun{One: strings.ToLower(n.One), Many: strings.ToLower(n.Many)}
}

// joinWords is ws as a list in a sentence: "A", "A and B", "A, B and C".
func joinWords(ws []string) string {
	if len(ws) == 1 {
		return ws[0]
	}
	return strings.Join(ws[:len(ws)-1], ", ") + " and " + ws[len(ws)-1]
}
